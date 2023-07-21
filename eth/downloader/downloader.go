// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

// Package downloader contains the manual full chain synchronisation.
package downloader

import (
	"errors"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	ethereum "gitlab.waterfall.network/waterfall/protocol/gwat"
	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/rawdb"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/state/snapshot"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/types"
	"gitlab.waterfall.network/waterfall/protocol/gwat/eth/protocols/eth"
	"gitlab.waterfall.network/waterfall/protocol/gwat/eth/protocols/snap"
	"gitlab.waterfall.network/waterfall/protocol/gwat/ethdb"
	"gitlab.waterfall.network/waterfall/protocol/gwat/event"
	"gitlab.waterfall.network/waterfall/protocol/gwat/log"
	"gitlab.waterfall.network/waterfall/protocol/gwat/metrics"
	"gitlab.waterfall.network/waterfall/protocol/gwat/params"
	"gitlab.waterfall.network/waterfall/protocol/gwat/trie"
	"gitlab.waterfall.network/waterfall/protocol/gwat/validator/era"
)

var (
	MaxBlockFetch   = 128 // Amount of blocks to be fetched per retrieval request
	MaxHeaderFetch  = 192 // Amount of block headers to be fetched per retrieval request
	MaxSkeletonSize = 128 // Number of header fetches to need for a skeleton assembly
	MaxReceiptFetch = 256 // Amount of transaction receipts to allow fetching per request
	MaxStateFetch   = 384 // Amount of node state values to allow fetching per request

	maxQueuedHeaders            = 32 * 1024                         // [eth/62] Maximum number of headers to queue for import (DOS protection)
	maxHeadersProcess           = 2048                              // Number of header download results to import at once into the chain
	maxResultsProcess           = 2048                              // Number of content download results to import at once into the chain
	fullMaxForkAncestry  uint64 = params.FullImmutabilityThreshold  // Maximum chain reorganisation (locally redeclared so tests can reduce it)
	lightMaxForkAncestry uint64 = params.LightImmutabilityThreshold // Maximum chain reorganisation (locally redeclared so tests can reduce it)

	reorgProtThreshold   = 48 // Threshold number of recent blocks to disable mini reorg protection
	reorgProtHeaderDelay = 2  // Number of headers to delay delivering to cover mini reorgs

	fsHeaderCheckFrequency = 100             // Verification frequency of the downloaded headers during fast sync
	fsHeaderSafetyNet      = 2048            // Number of headers to discard in case a chain violation is detected
	fsHeaderForceVerify    = 24              // Number of headers to verify before and after the pivot to accept it
	fsHeaderContCheck      = 3 * time.Second // Time interval to check for header continuations during state download
	fsMinFullBlocks        = 64              // Number of blocks to retrieve fully even in fast sync
)

var (
	errBusy                    = errors.New("busy")
	errUnknownPeer             = errors.New("peer is unknown or unhealthy")
	errBadPeer                 = errors.New("action from bad peer ignored")
	errStallingPeer            = errors.New("peer is stalling")
	errUnsyncedPeer            = errors.New("unsynced peer")
	errNoPeers                 = errors.New("no peers to keep download active")
	errTimeout                 = errors.New("timeout")
	errEmptyHeaderSet          = errors.New("empty header set by peer")
	errPeersUnavailable        = errors.New("no peers available or all tried for download")
	errInvalidAncestor         = errors.New("retrieved ancestor is invalid")
	errInvalidChain            = errors.New("retrieved hash chain is invalid")
	errInvalidBody             = errors.New("retrieved block body is invalid")
	errInvalidReceipt          = errors.New("retrieved receipt is invalid")
	errInvalidDag              = errors.New("retrieved dag chain is invalid")
	errInvalidBaseSpine        = errors.New("invalid base spine")
	errCancelStateFetch        = errors.New("state data download canceled (requested)")
	errCancelContentProcessing = errors.New("content processing canceled (requested)")
	errCanceled                = errors.New("syncing canceled (requested)")
	errNoSyncActive            = errors.New("no sync active")
	errTooOld                  = errors.New("peer's protocol version too old")
	errNoAncestorFound         = errors.New("no common ancestor found")
	errDataSizeLimitExceeded   = errors.New("data size limit exceeded")
)

type Downloader struct {
	mode uint32         // Synchronisation mode defining the strategy used (per sync cycle), use d.getMode() to get the SyncMode
	mux  *event.TypeMux // Event multiplexer to announce sync operation events

	checkpoint uint64   // Checkpoint block number to enforce head against (e.g. fast sync)
	genesis    uint64   // Genesis block number to limit sync to (e.g. light client CHT)
	queue      *queue   // Scheduler for selecting the hashes to download
	peers      *peerSet // Set of active peers from which download can proceed

	stateDB    ethdb.Database  // Database to state sync into (and deduplicate via)
	stateBloom *trie.SyncBloom // Bloom filter for fast trie node and contract code existence checks

	// Statistics
	syncStatsChainOrigin uint64 // Origin block number where syncing started at
	syncStatsChainHeight uint64 // Highest block number known when syncing started
	syncStatsState       stateSyncStats
	syncStatsLock        sync.RWMutex // Lock protecting the sync stats fields

	lightchain LightChain
	blockchain BlockChain

	// Callbacks
	dropPeer peerDropFn // Drops a peer for misbehaving

	// Status
	synchroniseMock func(id string, hash common.HashArray) error // Replacement for synchronise during testing
	finSyncing      int32
	dagSyncing      int32
	notified        int32
	committed       int32
	ancientLimit    uint64 // The maximum block number which can be regarded as ancient data.

	// Channels
	dagCh         chan dataPack        // Channel receiving inbound dag hashes
	headerCh      chan dataPack        // Channel receiving inbound block headers
	bodyCh        chan dataPack        // Channel receiving inbound block bodies
	receiptCh     chan dataPack        // Channel receiving inbound receipts
	bodyWakeCh    chan bool            // Channel to signal the block body fetcher of new tasks
	receiptWakeCh chan bool            // Channel to signal the receipt fetcher of new tasks
	headerProcCh  chan []*types.Header // Channel to feed the header processor new tasks

	// State sync
	pivotHeader *types.Header // Pivot block header to dynamically push the syncing state root
	pivotLock   sync.RWMutex  // Lock protecting pivot header reads from updates

	snapSync       bool         // Whether to run state sync over the snap protocol
	SnapSyncer     *snap.Syncer // TODO(karalabe): make private! hack for now
	stateSyncStart chan *stateSync
	trackStateReq  chan *stateReq
	stateCh        chan dataPack // Channel receiving inbound node state data

	// Cancellation and termination
	cancelPeer string         // Identifier of the peer currently being used as the master (cancel on drop)
	cancelCh   chan struct{}  // Channel to cancel mid-flight syncs
	cancelLock sync.RWMutex   // Lock to protect the cancel channel and peer in delivers
	cancelWg   sync.WaitGroup // Make sure all fetcher goroutines have exited.

	quitCh   chan struct{} // Quit channel to signal termination
	quitLock sync.Mutex    // Lock to prevent double closes

	// Testing hooks
	syncInitHook     func(uint64, uint64)  // Method to call upon initiating a new sync run
	bodyFetchHook    func([]*types.Header) // Method to call upon starting a block body fetch
	receiptFetchHook func([]*types.Header) // Method to call upon starting a receipt fetch
	chainInsertHook  func([]*fetchResult)  // Method to call upon inserting a chain of blocks (possibly in multiple invocations)
}

// LightChain encapsulates functions required to synchronise a light chain.
type LightChain interface {
	// HasHeader verifies a header's presence in the local chain.
	HasHeader(common.Hash) bool

	// GetHeaderByHash retrieves a header from the local chain.
	GetHeaderByHash(common.Hash) *types.Header

	// GetHeaderByNumber retrieves a header from the local chain by finalized number.
	GetHeaderByNumber(number uint64) *types.Header

	// GetLastFinalizedHeader retrieves the head header from the local chain.
	GetLastFinalizedHeader() *types.Header

	// GetLastCoordinatedCheckpoint retrieves the last coordinated checkpoint.
	GetLastCoordinatedCheckpoint() *types.Checkpoint

	// InsertHeaderChain inserts a batch of headers into the local chain.
	InsertHeaderChain([]*types.Header, int) (int, error)

	// SetHead rewinds the local chain to a new head.
	SetHead(array common.Hash) error

	// WriteSyncDagBlock writes the dag block and all associated state to the database for dag synchronization process
	WriteSyncDagBlock(block *types.Block, validate bool) (status int, err error)

	WriteSyncBlocks(blocks types.Blocks, validate bool) (status int, err error)

	GetInsertDelayedHashes() common.HashArray

	GetSlotInfo() *types.SlotInfo

	Config() *params.ChainConfig
}

// BlockChain encapsulates functions required to sync a (full or fast) blockchain.
type BlockChain interface {
	LightChain
	era.Blockchain

	// HasBlock verifies a block's presence in the local chain.
	HasBlock(common.Hash) bool

	// HasFastBlock verifies a fast block's presence in the local chain.
	HasFastBlock(common.Hash) bool

	// GetBlockByHash retrieves a block from the local chain.
	GetBlockByHash(common.Hash) *types.Block

	// GetBlocksByHashes retrieves block by hash.
	GetBlocksByHashes(hashes common.HashArray) types.BlockMap

	// GetLastFinalizedNumber retrieves the last finalized block number.
	GetLastFinalizedNumber() uint64

	// GetBlockFinalizedNumber retrieves the finalized block number by hash.
	GetBlockFinalizedNumber(hash common.Hash) *uint64

	// GetLastFinalizedBlock retrieves the last finalized block from the local chain.
	GetLastFinalizedBlock() *types.Block

	// GetLastFinalizedFastBlock retrieves the head fast block from the local chain.
	GetLastFinalizedFastBlock() *types.Block

	// FastSyncCommitHead directly commits the head block to a certain entity.
	FastSyncCommitHead(common.Hash) error

	// InsertChain inserts a batch of blocks into the local chain.
	InsertChain(types.Blocks) (int, error)

	// SyncInsertChain save a batch of blocks into the local db.
	SyncInsertChain(types.Blocks) (int, error)

	// InsertReceiptChain inserts a batch of receipts into the local chain.
	InsertReceiptChain(types.Blocks, []types.Receipts, uint64) (int, error)

	// Snapshots returns the blockchain snapshot tree to paused it during sync.
	Snapshots() *snapshot.Tree

	//GetTips retrieves active tips headers (no descendants)
	GetTips() types.Tips
	//AddTips add BlockDag to tips
	AddTips(blockDag *types.BlockDAG)
	//RemoveTips remove BlockDag from tips by hash from tips
	RemoveTips(hashes common.HashArray)
	ResetTips() error
	GetUnsynchronizedTipsHashes() common.HashArray
	ExploreChainRecursive(headHash common.Hash, memo ...core.ExploreResultMap) (unloaded, loaded, finalized common.HashArray, graph *types.GraphDag, cache core.ExploreResultMap, err error)
	//SetSyncProvider set provider of access to synchronization functionality
	SetSyncProvider(provider types.SyncProvider)
	IsSynced() bool
}

// New creates a new downloader to fetch hashes and blocks from remote peers.
func New(checkpoint uint64, stateDb ethdb.Database, stateBloom *trie.SyncBloom, mux *event.TypeMux, chain BlockChain, lightchain LightChain, dropPeer peerDropFn) *Downloader {
	if lightchain == nil {
		lightchain = chain
	}
	dl := &Downloader{
		stateDB:        stateDb,
		stateBloom:     stateBloom,
		mux:            mux,
		checkpoint:     checkpoint,
		queue:          newQueue(blockCacheMaxItems, blockCacheInitialItems),
		peers:          newPeerSet(),
		blockchain:     chain,
		lightchain:     lightchain,
		dropPeer:       dropPeer,
		dagCh:          make(chan dataPack, 1),
		headerCh:       make(chan dataPack, 1),
		bodyCh:         make(chan dataPack, 1),
		receiptCh:      make(chan dataPack, 1),
		bodyWakeCh:     make(chan bool, 1),
		receiptWakeCh:  make(chan bool, 1),
		headerProcCh:   make(chan []*types.Header, 1),
		quitCh:         make(chan struct{}),
		stateCh:        make(chan dataPack),
		SnapSyncer:     snap.NewSyncer(stateDb),
		stateSyncStart: make(chan *stateSync),
		syncStatsState: stateSyncStats{
			processed: rawdb.ReadFastTrieProgress(stateDb),
		},
		trackStateReq: make(chan *stateReq),
	}
	chain.SetSyncProvider(dl)
	go dl.stateFetcher()
	return dl
}

//// ClearBlockDag removes all BlockDag records
//func (d *Downloader) ClearBlockDag() {
//	dagHashes := rawdb.ReadAllBlockDagHashes(d.stateDB)
//	for _, hash := range dagHashes {
//		rawdb.DeleteBlockDag(d.stateDB, hash)
//	}
//}

// Progress retrieves the synchronisation boundaries, specifically the origin
// block where synchronisation started at (may have failed/suspended); the block
// or header sync is currently at; and the latest known block which the sync targets.
//
// In addition, during the state download phase of fast synchronisation the number
// of processed and the total number of known states are also returned. Otherwise
// these are zero.
func (d *Downloader) Progress() ethereum.SyncProgress {
	// Lock the current stats and return the progress
	d.syncStatsLock.RLock()
	defer d.syncStatsLock.RUnlock()

	current := uint64(0)
	mode := d.getMode()
	switch {
	case d.blockchain != nil && mode == FullSync:
		current = d.blockchain.GetLastFinalizedNumber()
	case d.blockchain != nil && mode == FastSync:
		current = d.blockchain.GetLastFinalizedFastBlock().Nr()
	case d.lightchain != nil:
		current = d.lightchain.GetLastFinalizedHeader().Nr()
	default:
		log.Error("Unknown downloader chain/mode combo", "light", d.lightchain != nil, "full", d.blockchain != nil, "mode", mode)
	}

	currSlot := uint64(0)
	if si := d.blockchain.GetSlotInfo(); si != nil {
		currSlot = si.CurrentSlot()
	}

	return ethereum.SyncProgress{
		StartingBlock: d.syncStatsChainOrigin,
		CurrentBlock:  current,
		HighestBlock:  d.syncStatsChainHeight,
		PulledStates:  d.syncStatsState.processed,
		KnownStates:   d.syncStatsState.processed + d.syncStatsState.pending,
		FinalizedSlot: d.blockchain.GetLastFinalizedHeader().Slot,
		MaxDagSlot:    d.blockchain.GetTips().GetMaxSlot(),
		CurrentSlot:   currSlot,
	}
}

// Synchronising returns whether the downloader is currently synchronising.
func (d *Downloader) Synchronising() bool {
	return d.FinSynchronising() || d.DagSynchronising()
}

// FinSynchronising returns whether the downloader is currently retrieving finalized blocks.
func (d *Downloader) FinSynchronising() bool {
	return atomic.LoadInt32(&d.finSyncing) > 0
}

// DagSynchronising returns whether the downloader is currently retrieving dag chain blocks.
func (d *Downloader) DagSynchronising() bool {
	return atomic.LoadInt32(&d.dagSyncing) > 0
}

// RegisterPeer injects a new download peer into the set of block source to be
// used for fetching hashes and blocks from.
func (d *Downloader) RegisterPeer(id string, version uint, peer Peer) error {
	var logger log.Logger
	if len(id) < 16 {
		// Tests use short IDs, don't choke on them
		logger = log.New("peer", id)
	} else {
		logger = log.New("peer", id[:8])
	}
	logger.Info("Registering sync peer", "id", id)
	if err := d.peers.Register(newPeerConnection(id, version, peer, logger)); err != nil {
		logger.Error("Failed to register sync peer", "err", err)
		return err
	}
	return nil
}

// GetPeers returns all active peers.
func (d *Downloader) GetPeers() *peerSet {
	return d.peers
}

// RegisterLightPeer injects a light client peer, wrapping it so it appears as a regular peer.
func (d *Downloader) RegisterLightPeer(id string, version uint, peer LightPeer) error {
	return d.RegisterPeer(id, version, &lightPeerWrapper{peer})
}

// UnregisterPeer remove a peer from the known list, preventing any action from
// the specified peer. An effort is also made to return any pending fetches into
// the queue.
func (d *Downloader) UnregisterPeer(id string) error {
	// Unregister the peer from the active peer set and revoke any fetch tasks
	var logger log.Logger
	if len(id) < 16 {
		// Tests use short IDs, don't choke on them
		logger = log.New("peer", id)
	} else {
		logger = log.New("peer", id[:8])
	}
	logger.Trace("Unregistering sync peer")
	if err := d.peers.Unregister(id); err != nil {
		logger.Error("Failed to unregister sync peer", "err", err)
		return err
	}
	d.queue.Revoke(id)

	return nil
}

func (d *Downloader) SynchroniseDagOnly(id string) error {
	err := d.synchroniseDagOnly(id)
	switch err {
	case nil, errBusy, errCanceled:
		return err
	}
	if errors.Is(err, errInvalidChain) || errors.Is(err, errBadPeer) || errors.Is(err, errTimeout) ||
		errors.Is(err, errStallingPeer) || errors.Is(err, errUnsyncedPeer) || errors.Is(err, errEmptyHeaderSet) ||
		errors.Is(err, errPeersUnavailable) || errors.Is(err, errTooOld) || errors.Is(err, errInvalidAncestor) {
		log.Warn("Synchronisation failed, dropping peer", "peer", id, "err", err)
		if d.dropPeer == nil {
			// The dropPeer method is nil when `--copydb` is used for a local copy.
			// Timeouts can occur if e.g. compaction hits at the wrong time, and can be ignored
			log.Warn("Downloader wants to drop peer, but peerdrop-function is not set", "peer", id)
		} else {
			d.dropPeer(id)
		}
		return err
	}
	log.Warn("Synchronisation failed, retrying", "err", err)
	return err
}

func (d *Downloader) synchroniseDagOnly(id string) error {
	if d.Synchronising() {
		log.Warn("Synchronization canceled (synchronise process busy)")
		return errBusy
	}

	// Make sure only one goroutine is ever allowed past this point at once
	if !atomic.CompareAndSwapInt32(&d.dagSyncing, 0, 1) {
		return errBusy
	}
	defer func(start time.Time) {
		atomic.StoreInt32(&d.dagSyncing, 0)
		log.Info("Synchronisation of dag chain terminated", "elapsed", common.PrettyDuration(time.Since(start)))
	}(time.Now())

	//todo check
	//// Make sure only one goroutine is ever allowed past this point at once
	//if !atomic.CompareAndSwapInt32(&d.finSyncing, 0, 1) {
	//	return errBusy
	//}
	//defer atomic.StoreInt32(&d.finSyncing, 0)

	//todo check
	//// Post a user notification of the sync (only once per session)
	//if atomic.CompareAndSwapInt32(&d.notified, 0, 1) {
	//	log.Info("Block synchronisation started")
	//}

	// If we are already full syncing, but have a fast-sync bloom filter laying
	// around, make sure it doesn't use memory any more. This is a special case
	// when the user attempts to fast sync a new empty network.
	if d.stateBloom != nil {
		d.stateBloom.Close()
	}

	//todo check
	//// Reset the queue, peer set and wake channels to clean any internal leftover state
	//d.queue.Reset(blockCacheMaxItems, blockCacheInitialItems)
	//d.peers.Reset()

	for _, ch := range []chan bool{d.bodyWakeCh, d.receiptWakeCh} {
		select {
		case <-ch:
		default:
		}
	}
	for _, ch := range []chan dataPack{d.headerCh, d.bodyCh, d.receiptCh, d.dagCh} {
		for empty := false; !empty; {
			select {
			case <-ch:
			default:
				empty = true
			}
		}
	}
	for empty := false; !empty; {
		select {
		case <-d.headerProcCh:
		default:
			empty = true
		}
	}
	// Create cancel channel for aborting mid-flight and mark the master peer
	d.cancelLock.Lock()
	d.cancelCh = make(chan struct{})
	d.cancelPeer = id
	d.cancelLock.Unlock()
	defer d.Cancel() // No matter what, we can't leave the cancel channel open

	//todo check
	//// Atomically set the requested sync mode
	//atomic.StoreUint32(&d.mode, uint32(FullSync))

	// Retrieve the origin peer and initiate the downloading process
	p := d.peers.Peer(id)
	if p == nil {
		return errUnknownPeer
	}
	return d.syncWithPeerDagOnly(p)
}

// syncWithPeer starts a block synchronization based on the hash chain from the
// specified peer and head hash.
func (d *Downloader) syncWithPeerDagOnly(p *peerConnection) (err error) {
	d.mux.Post(StartEvent{})
	defer func() {
		// reset on error
		if err != nil {
			d.mux.Post(FailedEvent{err})
		} else {
			latest := d.lightchain.GetLastFinalizedHeader()
			d.mux.Post(DoneEvent{latest})
		}
	}()
	if p.version < eth.ETH66 {
		return fmt.Errorf("%w: advertized %d < required %d", errTooOld, p.version, eth.ETH66)
	}

	defer func(start time.Time) {
		log.Info("^^^^^^^^^^^^ TIME",
			"elapsed", common.PrettyDuration(time.Since(start)),
			"func:", "sync:syncWithPeer",
		)
	}(time.Now())

	// fetch dag hashes
	baseSpine := d.lightchain.GetLastCoordinatedCheckpoint().Spine

	log.Info("Synchronising unloaded dag: start", "peer", p.id, "baseSpine", baseSpine.Hex())

	remoteDag, err := d.fetchDagHashes(p, baseSpine, common.Hash{})
	if err != nil {
		return err
	}

	log.Info("Synchronising unloaded dag: remoteDag 000", "remoteDag", remoteDag)

	delayed := d.blockchain.GetInsertDelayedHashes()
	remoteDag = remoteDag.Difference(delayed)
	log.Info("Synchronising unloaded dag: delayed   111", "remoteDag", remoteDag, "delayedIns", delayed)

	// filter existed blocks
	unloaded := make(common.HashArray, 0, len(remoteDag))
	dagBlocks := d.blockchain.GetBlocksByHashes(remoteDag)
	for h, b := range dagBlocks {
		if b == nil {
			unloaded = append(unloaded, h)
		}
	}

	log.Info("Synchronising unloaded dag: unloaded  222", "peer", p.id, "baseSpine", baseSpine.Hex(), "unloaded", unloaded)

	if err = d.syncWithPeerUnknownDagBlocks(p, unloaded); err != nil {
		log.Error("Synchronising unloaded dag failed", "err", err)
		return err
	}
	d.Cancel()
	return nil
}

// Synchronise tries to sync up our local block chain with a remote peer, both
// adding various sanity checks as well as wrapping it with various log entries.
func (d *Downloader) Synchronise(id string, dag common.HashArray, lastFinNr uint64, mode SyncMode, dagOnly bool) error {
	err := d.synchronise(id, dag, lastFinNr, mode, dagOnly)

	switch err {
	case nil, errBusy, errCanceled:
		return err
	}
	if errors.Is(err, errInvalidChain) || errors.Is(err, errBadPeer) || errors.Is(err, errTimeout) ||
		errors.Is(err, errStallingPeer) || errors.Is(err, errUnsyncedPeer) || errors.Is(err, errEmptyHeaderSet) ||
		errors.Is(err, errPeersUnavailable) || errors.Is(err, errTooOld) || errors.Is(err, errInvalidAncestor) {
		log.Warn("Synchronisation failed, dropping peer", "peer", id, "err", err)
		if d.dropPeer == nil {
			// The dropPeer method is nil when `--copydb` is used for a local copy.
			// Timeouts can occur if e.g. compaction hits at the wrong time, and can be ignored
			log.Warn("Downloader wants to drop peer, but peerdrop-function is not set", "peer", id)
		} else {
			d.dropPeer(id)
		}
		return err
	}
	log.Warn("Synchronisation failed, retrying", "err", err)
	return err
}

// synchronise will select the peer and use it for synchronising. If an empty string is given
// it will use the best peer possible and synchronize. If any of the
// checks fail an error will be returned. This method is synchronous
// deprecated
func (d *Downloader) synchronise(id string, dag common.HashArray, lastFinNr uint64, mode SyncMode, dagOnly bool) error {
	//// Mock out the synchronisation if testing
	//if d.synchroniseMock != nil {
	//	return d.synchroniseMock(id, dag)
	//}

	if d.DagSynchronising() {
		log.Warn("Synchronization canceled (synchronise process busy)")
		return errBusy
	}

	// Make sure only one goroutine is ever allowed past this point at once
	if !atomic.CompareAndSwapInt32(&d.finSyncing, 0, 1) {
		return errBusy
	}
	defer atomic.StoreInt32(&d.finSyncing, 0)

	// Post a user notification of the sync (only once per session)
	if atomic.CompareAndSwapInt32(&d.notified, 0, 1) {
		log.Info("Block synchronisation started")
	}
	// If we are already full syncing, but have a fast-sync bloom filter laying
	// around, make sure it doesn't use memory any more. This is a special case
	// when the user attempts to fast sync a new empty network.
	if mode == FullSync && d.stateBloom != nil {
		d.stateBloom.Close()
	}
	// If snap sync was requested, create the snap scheduler and switch to fast
	// sync mode. Long term we could drop fast sync or merge the two together,
	// but until snap becomes prevalent, we should support both. TODO(karalabe).
	if mode == SnapSync {
		if !d.snapSync {
			// Snap sync uses the snapshot namespace to store potentially flakey data until
			// sync completely heals and finishes. Pause snapshot maintenance in the mean
			// time to prevent access.
			if snapshots := d.blockchain.Snapshots(); snapshots != nil { // Only nil in tests
				snapshots.Disable()
			}
			log.Warn("Enabling snapshot sync prototype")
			d.snapSync = true
		}
		mode = FastSync
	}
	// Reset the queue, peer set and wake channels to clean any internal leftover state
	d.queue.Reset(blockCacheMaxItems, blockCacheInitialItems)
	d.peers.Reset()

	for _, ch := range []chan bool{d.bodyWakeCh, d.receiptWakeCh} {
		select {
		case <-ch:
		default:
		}
	}
	for _, ch := range []chan dataPack{d.headerCh, d.bodyCh, d.receiptCh, d.dagCh} {
		for empty := false; !empty; {
			select {
			case <-ch:
			default:
				empty = true
			}
		}
	}
	for empty := false; !empty; {
		select {
		case <-d.headerProcCh:
		default:
			empty = true
		}
	}
	// Create cancel channel for aborting mid-flight and mark the master peer
	d.cancelLock.Lock()
	d.cancelCh = make(chan struct{})
	d.cancelPeer = id
	d.cancelLock.Unlock()

	defer d.Cancel() // No matter what, we can't leave the cancel channel open

	// Atomically set the requested sync mode
	atomic.StoreUint32(&d.mode, uint32(mode))

	// Retrieve the origin peer and initiate the downloading process
	p := d.peers.Peer(id)
	if p == nil {
		return errUnknownPeer
	}
	if dagOnly {
		return d.syncWithPeerDagOnly(p)
	}
	return d.syncWithPeer(p, dag, lastFinNr, dagOnly)
}

func (d *Downloader) getMode() SyncMode {
	return SyncMode(atomic.LoadUint32(&d.mode))
}

// syncWithPeer starts a block synchronization based on the hash chain from the
// specified peer and head hash.
// deprecated
func (d *Downloader) syncWithPeer(p *peerConnection, dag common.HashArray, lastFinNr uint64, dagOnly bool) (err error) {
	d.mux.Post(StartEvent{})
	defer func() {
		// reset on error
		if err != nil {
			d.mux.Post(FailedEvent{err})
		} else {
			latest := d.lightchain.GetLastFinalizedHeader()
			d.mux.Post(DoneEvent{latest})
		}
	}()
	if p.version < eth.ETH66 {
		return fmt.Errorf("%w: advertized %d < required %d", errTooOld, p.version, eth.ETH66)
	}
	mode := d.getMode()

	defer func(start time.Time) {
		log.Debug("Synchronisation terminated", "elapsed", common.PrettyDuration(time.Since(start)))
	}(time.Now())

	log.Info("Synchronising with the network", "peer", p.id, "eth", p.version, "mode", mode, "lastFinNr", lastFinNr, "dag", dag)

	// if remote peer has unknown dag blocks only
	// sync such blocks only
	if dagOnly {
		if err = d.syncWithPeerUnknownDagBlocks(p, dag); err != nil {
			log.Error("Synchronization of unknown dag blocks failed", "err", err)
			return err
		}
		d.Cancel()
		return nil
	}

	//// Look up the sync boundaries: the common ancestor and the target block
	//latest, pivot, err := d.fetchHead(p)
	//if err != nil {
	//	return err
	//}
	//if mode == FastSync && pivot == nil {
	//	// If no pivot block was returned, the head is below the min full block
	//	// threshold (i.e. new chain). In that case we won't really fast sync
	//	// anyway, but still need a valid pivot block to avoid some code hitting
	//	// nil panics on an access.
	//	pivot = d.blockchain.GetLastFinalizedBlock().Header()
	//}
	//height := lastFinNr
	//
	//origin, err := d.findCoordinatedAncestor(p, latest)
	//log.Info("Synchronization of finalized chain: start", "origin", origin, "latest.Number", latest.Nr(), "latest.Hash", latest.Hash().Hex())
	//if err != nil {
	//	d.dropPeer(p.id)
	//	return err
	//}
	//d.syncStatsLock.Lock()
	//if d.syncStatsChainHeight <= origin || d.syncStatsChainOrigin > origin {
	//	d.syncStatsChainOrigin = origin
	//}
	//d.syncStatsChainHeight = height
	//d.syncStatsLock.Unlock()
	//
	//// Ensure our origin point is below any fast sync pivot point
	//if mode == FastSync {
	//	if height <= uint64(fsMinFullBlocks) {
	//		origin = 0
	//	} else {
	//		pivotNumber := *pivot.Number
	//		if pivotNumber <= origin {
	//			origin = pivotNumber - 1
	//		}
	//		// Write out the pivot into the database so a rollback beyond it will
	//		// reenable fast sync
	//		rawdb.WriteLastPivotNumber(d.stateDB, pivotNumber)
	//	}
	//}
	//d.committed = 1
	//if mode == FastSync && *pivot.Number != 0 {
	//	d.committed = 0
	//}
	//if mode == FastSync {
	//	// Set the ancient data limitation.
	//	// If we are running fast sync, all block data older than ancientLimit will be
	//	// written to the ancient store. More recent data will be written to the active
	//	// database and will wait for the freezer to migrate.
	//	//
	//	// If there is a checkpoint available, then calculate the ancientLimit through
	//	// that. Otherwise calculate the ancient limit through the advertised height
	//	// of the remote peer.
	//	//
	//	// The reason for picking checkpoint first is that a malicious peer can give us
	//	// a fake (very high) height, forcing the ancient limit to also be very high.
	//	// The peer would start to feed us valid blocks until head, resulting in all of
	//	// the blocks might be written into the ancient store. A following mini-reorg
	//	// could cause issues.
	//	if d.checkpoint != 0 && d.checkpoint > fullMaxForkAncestry+1 {
	//		d.ancientLimit = d.checkpoint
	//	} else if height > fullMaxForkAncestry+1 {
	//		d.ancientLimit = height - fullMaxForkAncestry - 1
	//	} else {
	//		d.ancientLimit = 0
	//	}
	//	frozen, _ := d.stateDB.Ancients() // Ignore the error here since light client can also hit here.
	//
	//	// If a part of blockchain data has already been written into active store,
	//	// disable the ancient style insertion explicitly.
	//	if origin >= frozen && frozen != 0 {
	//		d.ancientLimit = 0
	//		log.Info("Disabling direct-ancient mode", "origin", origin, "ancient", frozen-1)
	//	} else if d.ancientLimit > 0 {
	//		log.Debug("Enabling direct-ancient mode", "ancient", d.ancientLimit)
	//	}
	//	// Rewind the ancient store and blockchain if reorg happens.
	//	if origin+1 < frozen {
	//		header := d.lightchain.GetHeaderByNumber(origin + 1)
	//		if header != nil {
	//			if err := d.lightchain.SetHead(header.Hash()); err != nil {
	//				return err
	//			}
	//		}
	//	}
	//}
	// Initiate the sync using a concurrent header and content retrieval algorithm
	//d.queue.Prepare(origin+1, mode)
	//if d.syncInitHook != nil {
	//	d.syncInitHook(origin, height)
	//}
	//fetchers := []func() error{
	//	func() error { return d.fetchHeaders(p, origin+1) }, // Headers are always retrieved
	//	func() error { return d.fetchBodies(origin + 1) },   // Bodies are retrieved during normal and fast sync
	//	func() error { return d.fetchReceipts(origin + 1) }, // Receipts are retrieved during fast sync
	//	func() error { return d.processHeaders(origin + 1) },
	//}
	//if mode == FastSync {
	//	d.pivotLock.Lock()
	//	d.pivotHeader = pivot
	//	d.pivotLock.Unlock()
	//
	//	fetchers = append(fetchers, func() error { return d.processFastSyncContent() })
	//} else if mode == FullSync {
	//	fetchers = append(fetchers, d.processFullSyncContent)
	//}
	////Synchronisation of finalized chain
	//if err = d.spawnSync(fetchers); err != nil {
	//	log.Error("Synchronization of finalized chain failed", "err", err)
	//	d.Cancel()
	//	return err
	//}
	//Synchronisation of dag chain
	//if err = d.syncWithPeerDagChain(p); err != nil {
	//	log.Error("Synchronization of dag chain failed", "err", err)
	//	d.Cancel()
	//	return err
	//}
	d.Cancel()
	return nil
}

// deprecated
// syncWithPeerDagChain downloads and set on current node dag chain from remote peer
func (d *Downloader) syncWithPeerDagChain(p *peerConnection) (err error) {
	// Make sure only one goroutine is ever allowed past this point at once
	if !atomic.CompareAndSwapInt32(&d.dagSyncing, 0, 1) {
		return errBusy
	}
	defer func(start time.Time) {
		atomic.StoreInt32(&d.dagSyncing, 0)
		log.Debug("Synchronisation of dag chain terminated", "elapsed", common.PrettyDuration(time.Since(start)))
	}(time.Now())

	lcp := d.blockchain.GetLastCoordinatedCheckpoint()
	lastFinBlock := d.blockchain.GetLastFinalizedBlock()
	spineBlock := d.blockchain.GetBlockByHash(lcp.Spine)

	// fetch dag up to next epoch
	dag, err := d.fetchDagHashes(p, spineBlock.Hash(), common.Hash{})
	if len(dag) == 1 {
		// if remote tips set to last finalized block - do same
		block := d.blockchain.GetBlockByHash(dag[0])
		if block != nil && block.Nr() == spineBlock.Nr() {
			return d.blockchain.ResetTips()
		}
	}

	log.Info("Synchronization of dag chain: dag hashes retrieved", "dag", dag, "err", err)
	if err != nil {
		return err
	}
	headers, err := d.fetchDagHeaders(p, dag)
	log.Info("Synchronization of dag chain: dag headers retrieved", "count", len(headers), "headers", headers, "err", err)
	if err != nil {
		return err
	}
	txsMap, err := d.fetchDagTxs(p, dag)
	log.Info("Synchronization of dag chain: dag transactions retrieved", "count", len(txsMap), "txs", txsMap, "err", err)
	if err != nil {
		return err
	}
	// rm deprecated dag info
	//d.ClearBlockDag()
	d.blockchain.RemoveTips(d.blockchain.GetTips().GetHashes())
	blocks := make(types.Blocks, len(headers))
	for i, header := range headers {
		txs := txsMap[header.Hash()]
		block := types.NewBlockWithHeader(header).WithBody(txs)
		blocks[i] = block
	}
	blocksBySlot, err := (&blocks).GroupBySlot()
	if err != nil {
		return err
	}
	//sort by slots
	slots := common.SorterAscU64{}
	for sl, _ := range blocksBySlot {
		slots = append(slots, sl)
	}
	sort.Sort(slots)

	for _, slot := range slots {
		slotBlocks := types.SpineSortBlocks(blocksBySlot[slot])
		if len(slotBlocks) == 0 {
			continue
		}
		spine := slotBlocks[0]
		if spine.Slot() > lastFinBlock.Slot() {
			_, err = d.blockchain.WriteSyncDagBlock(spine, true)
			if err != nil {
				log.Error("Failed writing block to chain (sync dep)", "err", err)
				return err
			}
			slotBlocks = slotBlocks[1:]
		}
		//handle by reverse order
		for _, block := range slotBlocks {
			// if block is finalized
			if block.Nr() != 0 && block.Height() > 0 && block.Nr() < lastFinBlock.Nr() || block.Height() == 0 {
				continue
			}
			// Commit block and state to database.
			_, err = d.blockchain.WriteSyncDagBlock(block, true)
			if err != nil {
				log.Error("Failed writing block to chain (sync dep)", "err", err)
				return err
			}

		}
	}
	return err
}

// syncWithPeerUnknownDagBlocks if remote peer has unknown dag blocks only sync such blocks only
func (d *Downloader) syncWithPeerUnknownDagBlocks(p *peerConnection, dag common.HashArray) (err error) {
	// Make sure only one goroutine is ever allowed past this point at once
	if !atomic.CompareAndSwapInt32(&d.dagSyncing, 0, 1) {
		return errBusy
	}

	defer func(start time.Time) {
		atomic.StoreInt32(&d.dagSyncing, 0)
		log.Debug("Synchronisation of dag chain terminated", "elapsed", common.PrettyDuration(time.Since(start)))
	}(time.Now())

	log.Warn("Synchronization of unknown dag blocks", "count", len(dag), "dag", dag)

	headers, err := d.fetchDagHeaders(p, dag)
	log.Info("Synchronization of unknown dag blocks: dag headers retrieved", "count", len(headers), "headers", headers, "err", err)
	if err != nil {
		return err
	}
	txsMap, err := d.fetchDagTxs(p, dag)
	log.Info("Synchronization of unknown dag blocks: dag transactions retrieved", "count", len(txsMap), "err", err)
	if err != nil {
		return err
	}

	blocks := make(types.Blocks, 0, len(headers))
	for _, header := range headers {
		txs := txsMap[header.Hash()]
		block := types.NewBlockWithHeader(header).WithBody(txs)
		blocks = append(blocks, block)
	}

	blocksBySlot, err := (&blocks).GroupBySlot()
	if err != nil {
		return err
	}
	//sort by slots
	slots := common.SorterAscU64{}
	for sl, _ := range blocksBySlot {
		slots = append(slots, sl)
	}
	sort.Sort(slots)

	insBlocks := make(types.Blocks, 0, len(headers))
	for _, slot := range slots {
		slotBlocks := types.SpineSortBlocks(blocksBySlot[slot])
		insBlocks = append(insBlocks, slotBlocks...)
	}
	if i, err := d.blockchain.WriteSyncBlocks(insBlocks, true); err != nil {
		bl := insBlocks[i]
		log.Error("Failed writing block to chain  (sync unl)", "err", err, "bl.Slot", bl.Slot(), "hash", bl.Hash().Hex())
		return err
	}
	return nil
}

// spawnSync runs d.process and all given fetcher functions to completion in
// separate goroutines, returning the first error that appears.
func (d *Downloader) spawnSync(fetchers []func() error) error {
	errc := make(chan error, len(fetchers))
	d.cancelWg.Add(len(fetchers))
	for _, fn := range fetchers {
		fn := fn
		go func() {
			defer d.cancelWg.Done()
			errc <- fn()
		}()
	}
	// Wait for the first error, then terminate the others.
	var err error
	for i := 0; i < len(fetchers); i++ {
		if i == len(fetchers)-1 {
			// Close the queue when all fetchers have exited.
			// This will cause the block processor to end when
			// it has processed the queue.
			d.queue.Close()
		}
		if err = <-errc; err != nil && err != errCanceled {
			break
		}
	}
	d.queue.Close()
	//d.Cancel()
	return err
}

// cancel aborts all of the operations and resets the queue. However, cancel does
// not wait for the running download goroutines to finish. This method should be
// used when cancelling the downloads from inside the downloader.
func (d *Downloader) cancel() {
	// Close the current cancel channel
	d.cancelLock.Lock()
	defer d.cancelLock.Unlock()

	if d.cancelCh != nil {
		select {
		case <-d.cancelCh:
			// Channel was already closed
		default:
			close(d.cancelCh)
		}
	}
}

// Cancel aborts all of the operations and waits for all download goroutines to
// finish before returning.
func (d *Downloader) Cancel() {
	d.cancel()
	d.cancelWg.Wait()
}

// Terminate interrupts the downloader, canceling all pending operations.
// The downloader cannot be reused after calling Terminate.
func (d *Downloader) Terminate() {
	// Close the termination channel (make sure double close is allowed)
	d.quitLock.Lock()
	select {
	case <-d.quitCh:
	default:
		close(d.quitCh)
	}
	if d.stateBloom != nil {
		d.stateBloom.Close()
	}
	d.quitLock.Unlock()

	// Cancel any pending download requests
	d.Cancel()
}

// fetchHead retrieves the head header and prior pivot block (if available) from a remote peer.
func (d *Downloader) fetchHead(p *peerConnection) (head *types.Header, pivot *types.Header, err error) {
	p.log.Debug("Retrieving remote chain head")
	mode := d.getMode()

	// Request the advertised remote head block and wait for the response
	//lastFinNr, _ := p.peer.GetDagInfo()
	lcp := d.blockchain.GetLastCoordinatedCheckpoint()

	//lastFinNr, dag := p.peer.GetDagInfo()
	fetch := 1
	if mode == FastSync {
		fetch = 2 // head + pivot headers
	}
	// from spine hash to spine hash
	go p.peer.RequestHeadersByHash(lcp.Spine, fetch, fsMinFullBlocks-1, true)
	//go p.peer.RequestHeadersByNumber(lastFinNr, fetch, fsMinFullBlocks-1, true)

	ttl := d.peers.rates.TargetTimeout()
	timeout := time.After(ttl)
	for {
		select {
		case <-d.cancelCh:
			return nil, nil, errCanceled

		case packet := <-d.headerCh:
			// Discard anything not from the origin peer
			if packet.PeerId() != p.id {
				log.Debug("Received headers from incorrect peer", "peer", packet.PeerId())
				break
			}
			// Make sure the peer gave us at least one and at most the requested headers
			headers := packet.(*headerPack).headers
			if len(headers) == 0 || len(headers) > fetch {
				return nil, nil, fmt.Errorf("%w: returned headers %d != requested %d", errBadPeer, len(headers), fetch)
			}
			// The first header needs to be the head, validate against the checkpoint
			// and request. If only 1 header was returned, make sure there's no pivot
			// or there was not one requested.
			head := headers[0]
			if (mode == FastSync || mode == LightSync) && *head.Number < d.checkpoint {
				return nil, nil, fmt.Errorf("%w: remote head %d below checkpoint %d", errUnsyncedPeer, head.Nr(), d.checkpoint)
			}
			if len(headers) == 1 {
				if mode == FastSync && *head.Number > uint64(fsMinFullBlocks) {
					return nil, nil, fmt.Errorf("%w: no pivot included along head header", errBadPeer)
				}
				p.log.Info("Remote head identified, no pivot", "number", head.Nr(), "hash", head.Hash().Hex())
				return head, nil, nil
			}
			// At this point we have 2 headers in total and the first is the
			// validated head of the chain. Check the pivot number and return,
			pivot := headers[1]
			if *pivot.Number != *head.Number-uint64(fsMinFullBlocks) {
				return nil, nil, fmt.Errorf("%w: remote pivot %d != requested %d", errInvalidChain, pivot.Nr(), head.Nr()-uint64(fsMinFullBlocks))
			}
			return head, pivot, nil

		case <-timeout:
			p.log.Debug("Waiting for head header timed out", "elapsed", ttl)
			return nil, nil, errTimeout

		case <-d.bodyCh:
		case <-d.receiptCh:
			// Out of bounds delivery, ignore
		}
	}
}

// calculateRequestSpan calculates what headers to request from a peer when trying to determine the
// common ancestor.
// It returns parameters to be used for peer.RequestHeadersByNumber:
//
//	from - starting block number
//	count - number of headers to request
//	skip - number of headers to skip
//
// and also returns 'max', the last block which is expected to be returned by the remote peers,
// given the (from,count,skip)
// deprecated
func calculateRequestSpan(remoteHeight, localHeight uint64) (int64, int, int, uint64) {
	var (
		from     int
		count    int
		MaxCount = MaxHeaderFetch / 16
	)
	// requestHead is the highest block that we will ask for. If requestHead is not offset,
	// the highest block that we will get is 16 blocks back from head, which means we
	// will fetch 14 or 15 blocks unnecessarily in the case the height difference
	// between us and the peer is 1-2 blocks, which is most common
	requestHead := int(remoteHeight) - 1
	if requestHead < 0 {
		requestHead = 0
	}
	// requestBottom is the lowest block we want included in the query
	// Ideally, we want to include the one just below our own head
	requestBottom := int(localHeight - 1)
	if requestBottom < 0 {
		requestBottom = 0
	}
	totalSpan := requestHead - requestBottom
	span := 1 + totalSpan/MaxCount
	if span < 2 {
		span = 2
	}
	if span > 16 {
		span = 16
	}

	count = 1 + totalSpan/span
	if count > MaxCount {
		count = MaxCount
	}
	if count < 2 {
		count = 2
	}
	from = requestHead - (count-1)*span
	if from < 0 {
		from = 0
	}
	max := from + (count-1)*span
	return int64(from), count, span - 1, uint64(max)
}

// findAncestor tries to locate the common ancestor link of the local chain and
// a remote peers blockchain. In the general case when our node was in sync and
// on the correct chain, checking the top N links should already get us a match.
// In the rare scenario when we ended up on a long reorganisation (i.e. none of
// the head links match), we do a binary search to find the common ancestor.
// deprecated
func (d *Downloader) findAncestor(p *peerConnection, remoteHeader *types.Header) (uint64, error) {
	// Figure out the valid ancestor range to prevent rewrite attacks
	var (
		floor        = int64(-1)
		localHeight  uint64
		remoteHeight = *remoteHeader.Number
	)
	mode := d.getMode()
	switch mode {
	case FullSync:
		localHeight = d.blockchain.GetLastFinalizedNumber()
	case FastSync:
		localHeight = d.blockchain.GetLastFinalizedFastBlock().Nr()
	default:
		lastFinHeader := d.lightchain.GetLastFinalizedHeader()
		localHeight = *lastFinHeader.Number
	}
	p.log.Info("Looking for common ancestor", "local", localHeight, "remote", remoteHeight)

	// Recap floor value for binary search
	maxForkAncestry := fullMaxForkAncestry
	if d.getMode() == LightSync {
		maxForkAncestry = lightMaxForkAncestry
	}
	if localHeight >= maxForkAncestry {
		// We're above the max reorg threshold, find the earliest fork point
		floor = int64(localHeight - maxForkAncestry)
	}
	// If we're doing a light sync, ensure the floor doesn't go below the CHT, as
	// all headers before that point will be missing.
	if mode == LightSync {
		// If we don't know the current CHT position, find it
		if d.genesis == 0 {
			header := d.lightchain.GetLastFinalizedHeader()
			for header != nil && header.Number != nil {
				d.genesis = *header.Number
				if floor >= int64(d.genesis)-1 {
					break
				}
				header = d.lightchain.GetHeaderByNumber(*header.Number - 1)
			}
		}
		// We already know the "genesis" block number, cap floor to that
		if floor < int64(d.genesis)-1 {
			floor = int64(d.genesis) - 1
		}
	}

	ancestor, err := d.findAncestorSpanSearch(p, mode, remoteHeight, localHeight, floor)
	log.Info("Looking for common ancestor Span Search result", "mode", mode, "remoteHeight", remoteHeight, "floor", floor, "ancestor", ancestor, "err", err)

	if err == nil {
		return ancestor, nil
	}
	// The returned error was not nil.
	// If the error returned does not reflect that a common ancestor was not found, return it.
	// If the error reflects that a common ancestor was not found, continue to binary search,
	// where the error value will be reassigned.
	if !errors.Is(err, errNoAncestorFound) {
		return 0, err
	}

	ancestor, err = d.findAncestorBinarySearch(p, mode, remoteHeight, floor)
	log.Info("Looking for common ancestor Binary Search result", "mode", mode, "remoteHeight", remoteHeight, "floor", floor, "ancestor", ancestor, "err", err)

	if err != nil {
		return 0, err
	}
	return ancestor, nil
}

func (d *Downloader) findAncestorSpanSearch(p *peerConnection, mode SyncMode, remoteHeight, localHeight uint64, floor int64) (commonAncestor uint64, err error) {
	from, count, skip, max := calculateRequestSpan(remoteHeight, localHeight)

	p.log.Info("Span searching for common ancestor", "count", count, "from", from, "skip", skip)
	go p.peer.RequestHeadersByNumber(uint64(from), count, skip, false)

	// Wait for the remote response to the head fetch
	// init by genesis
	genesisHash := d.blockchain.GetHeaderByNumber(0).Hash()
	hash := genesisHash
	number := uint64(0)

	ttl := d.peers.rates.TargetTimeout()
	timeout := time.After(ttl)

	for finished := false; !finished; {
		select {
		case <-d.cancelCh:
			return 0, errCanceled

		case packet := <-d.headerCh:
			log.Info("Span searching for common ancestor", "packet", packet)

			// Discard anything not from the origin peer
			if packet.PeerId() != p.id {
				log.Debug("Received headers from incorrect peer", "peer", packet.PeerId())
				break
			}
			// Make sure the peer actually gave something valid
			headers := packet.(*headerPack).headers
			if len(headers) == 0 {
				p.log.Warn("Empty head header set")
				return 0, errEmptyHeaderSet
			}
			// Make sure the peer's reply conforms to the request
			for i, header := range headers {
				expectNumber := uint64(from) + uint64(i)*uint64(skip+1)
				if number := header.Number; number != nil && *number != expectNumber {
					p.log.Warn("Head headers broke chain ordering", "index", i, "requested", expectNumber, "received", number)
					return 0, fmt.Errorf("%w: %v", errInvalidChain, errors.New("head headers broke chain ordering"))
				}
			}
			// Check if a common ancestor was found
			finished = true
			for i := len(headers) - 1; i >= 0; i-- {
				// Skip any headers that underflow/overflow our requested set
				if headers[i].Number == nil || *headers[i].Number < uint64(from) || *headers[i].Number > max {
					continue
				}
				// Otherwise check if we already know the header or not
				h := headers[i].Hash()
				n := *headers[i].Number

				var known bool
				switch mode {
				case FullSync:
					finNr := d.blockchain.GetBlockFinalizedNumber(h)
					if finNr != nil && n == *finNr {
						known = true
					}
				case FastSync:
					known = d.blockchain.HasFastBlock(h)
					finNr := d.blockchain.GetBlockFinalizedNumber(h)
					if known && finNr != nil && n == *finNr {
						known = true
					}
				default:
					known = d.lightchain.HasHeader(h)
				}
				if known {
					number, hash = n, h
					break
				}
			}

		case <-timeout:
			p.log.Debug("Waiting for head header timed out", "elapsed", ttl)
			return 0, errTimeout

		case <-d.bodyCh:
		case <-d.receiptCh:
			// Out of bounds delivery, ignore
		}
	}
	// If the head fetch already found an ancestor, return
	if hash != genesisHash {
		if int64(number) <= floor {
			p.log.Warn("Ancestor below allowance", "number", number, "hash", hash.Hex(), "allowance", floor)
			return 0, errInvalidAncestor
		}
		p.log.Info("Found common ancestor", "number", number, "hash", hash.Hex())
		return number, nil
	}
	return 0, errNoAncestorFound
}

func (d *Downloader) findAncestorBinarySearch(p *peerConnection, mode SyncMode, remoteHeight uint64, floor int64) (commonAncestor uint64, err error) {
	// init by genesis
	hash := d.blockchain.GetHeaderByNumber(0).Hash()

	// Ancestor not found, we need to binary search over our chain
	start, end := uint64(0), remoteHeight
	if floor > 0 {
		start = uint64(floor)
	}
	p.log.Debug("Binary searching for common ancestor", "start", start, "end", end)

	for start+1 < end {
		// Split our chain interval in two, and request the hash to cross check
		check := (start + end) / 2

		ttl := d.peers.rates.TargetTimeout()
		timeout := time.After(ttl)

		go p.peer.RequestHeadersByNumber(check, 1, 0, false)

		// Wait until a reply arrives to this request
		for arrived := false; !arrived; {
			select {
			case <-d.cancelCh:
				return 0, errCanceled

			case packet := <-d.headerCh:
				// Discard anything not from the origin peer
				if packet.PeerId() != p.id {
					log.Debug("Received headers from incorrect peer", "peer", packet.PeerId())
					break
				}
				// Make sure the peer actually gave something valid
				headers := packet.(*headerPack).headers
				if len(headers) != 1 {
					p.log.Warn("Multiple headers for single request", "headers", len(headers))
					return 0, fmt.Errorf("%w: multiple headers (%d) for single request", errBadPeer, len(headers))
				}
				arrived = true

				// Modify the search interval based on the response
				h := headers[0].Hash()
				n := *headers[0].Number

				var known bool
				switch mode {
				case FullSync:
					finNr := d.blockchain.GetBlockFinalizedNumber(h)
					if finNr != nil && n == *finNr {
						known = true
					}
				case FastSync:
					known = d.blockchain.HasFastBlock(h)
					finNr := d.blockchain.GetBlockFinalizedNumber(h)
					if known && finNr != nil && n == *finNr {
						known = true
					}
				default:
					known = d.lightchain.HasHeader(h)
				}
				if !known {
					end = check
					break
				}
				header := d.lightchain.GetHeaderByHash(h) // Independent of sync mode, header surely exists
				if n != check {
					p.log.Warn("Received non requested header", "number", header.Nr(), "hash", header.Hash().Hex(), "request", check)
					return 0, fmt.Errorf("%w: non-requested header (%d)", errBadPeer, header.Height)
				}
				start = check
				hash = h

			case <-timeout:
				p.log.Debug("Waiting for search header timed out", "elapsed", ttl)
				return 0, errTimeout

			case <-d.bodyCh:
			case <-d.receiptCh:
				// Out of bounds delivery, ignore
			}
		}
	}
	// Ensure valid ancestry and return
	if int64(start) <= floor {
		p.log.Warn("Ancestor below allowance", "number", start, "hash", hash.Hex(), "allowance", floor)
		return 0, errInvalidAncestor
	}
	p.log.Debug("Found common ancestor", "number", start, "hash", hash)
	return start, nil
}

// fetchHeaders keeps retrieving headers concurrently from the number
// requested, until no more are returned, potentially throttling on the way. To
// facilitate concurrency but still protect against malicious nodes sending bad
// headers, we construct a header chain skeleton using the "origin" peer we are
// syncing with, and fill in the missing headers using anyone else. Headers from
// other peers are only accepted if they map cleanly to the skeleton. If no one
// can fill in the skeleton - not even the origin peer - it's assumed invalid and
// the origin is dropped.
func (d *Downloader) fetchHeaders(p *peerConnection, from uint64) error {
	p.log.Debug("Directing header downloads", "origin", from)
	defer p.log.Debug("Header download terminated")

	// Create a timeout timer, and the associated header fetcher
	skeleton := false           // Skeleton assembly phase or finishing up
	pivoting := false           // Whether the next request is pivot verification
	request := time.Now()       // time of the last skeleton fetch request
	timeout := time.NewTimer(0) // timer to dump a non-responsive active peer
	<-timeout.C                 // timeout channel should be initially empty
	defer timeout.Stop()

	var ttl time.Duration
	getHeaders := func(from uint64) {
		request = time.Now()

		ttl = d.peers.rates.TargetTimeout()
		timeout.Reset(ttl)

		if skeleton {
			p.log.Info("Fetching skeleton headers", "count", MaxHeaderFetch, "from", from)
			go p.peer.RequestHeadersByNumber(from+uint64(MaxHeaderFetch)-1, MaxSkeletonSize, MaxHeaderFetch-1, false)
		} else {
			p.log.Info("Fetching full headers", "count", MaxHeaderFetch, "from", from)
			go p.peer.RequestHeadersByNumber(from, MaxHeaderFetch, 0, false)
		}
	}
	getNextPivot := func() {
		pivoting = true
		request = time.Now()

		ttl = d.peers.rates.TargetTimeout()
		timeout.Reset(ttl)

		d.pivotLock.RLock()
		pivot := d.pivotHeader.Nr()
		d.pivotLock.RUnlock()

		p.log.Debug("Fetching next pivot header", "number", pivot+uint64(fsMinFullBlocks))
		go p.peer.RequestHeadersByNumber(pivot+uint64(fsMinFullBlocks), 2, fsMinFullBlocks-9, false) // move +64 when it's 2x64-8 deep
	}
	// Start pulling the header chain skeleton until all is done
	ancestor := from
	getHeaders(from)

	mode := d.getMode()
	for {
		select {
		case <-d.cancelCh:
			return errCanceled

		case packet := <-d.headerCh:
			// Make sure the active peer is giving us the skeleton headers
			if packet.PeerId() != p.id {
				log.Info("Received skeleton from incorrect peer", "peer", packet.PeerId())
				break
			}
			headerReqTimer.UpdateSince(request)
			timeout.Stop()

			// If the pivot is being checked, move if it became stale and run the real retrieval
			var pivot uint64

			d.pivotLock.RLock()
			if d.pivotHeader != nil && d.pivotHeader.Number != nil {
				pivot = *d.pivotHeader.Number
			}
			d.pivotLock.RUnlock()

			if pivoting {
				if packet.Items() == 2 {
					// Retrieve the headers and do some sanity checks, just in case
					headers := packet.(*headerPack).headers

					if have, want := *headers[0].Number, pivot+uint64(fsMinFullBlocks); have != want {
						log.Warn("Peer sent invalid next pivot", "have", have, "want", want)
						return fmt.Errorf("%w: next pivot number %d != requested %d", errInvalidChain, have, want)
					}
					if have, want := *headers[1].Number, pivot+2*uint64(fsMinFullBlocks)-8; have != want {
						log.Warn("Peer sent invalid pivot confirmer", "have", have, "want", want)
						return fmt.Errorf("%w: next pivot confirmer number %d != requested %d", errInvalidChain, have, want)
					}
					log.Warn("Pivot seemingly stale, moving", "old", pivot, "new", headers[0].Height)
					pivot = *headers[0].Number

					d.pivotLock.Lock()
					d.pivotHeader = headers[0]
					d.pivotLock.Unlock()

					// Write out the pivot into the database so a rollback beyond
					// it will reenable fast sync and update the state root that
					// the state syncer will be downloading.
					rawdb.WriteLastPivotNumber(d.stateDB, pivot)
				}
				pivoting = false
				getHeaders(from)
				continue
			}
			// If the skeleton's finished, pull any remaining head headers directly from the origin
			if skeleton && packet.Items() == 0 {
				skeleton = false
				getHeaders(from)
				continue
			}
			// If no more headers are inbound, notify the content fetchers and return
			if packet.Items() == 0 {
				// Don't abort header fetches while the pivot is downloading
				if atomic.LoadInt32(&d.committed) == 0 && pivot <= from {
					p.log.Debug("No headers, waiting for pivot commit")
					select {
					case <-time.After(fsHeaderContCheck):
						getHeaders(from)
						continue
					case <-d.cancelCh:
						return errCanceled
					}
				}
				// Pivot done (or not in fast sync) and no more headers, terminate the process
				p.log.Info("No more headers available")
				select {
				case d.headerProcCh <- nil:
					return nil
				case <-d.cancelCh:
					return errCanceled
				}
			}
			headers := packet.(*headerPack).headers

			// If we received a skeleton batch, resolve internals concurrently
			if skeleton {
				filled, proced, err := d.fillHeaderSkeleton(from, headers)
				if err != nil {
					p.log.Debug("Skeleton chain invalid", "err", err)
					return fmt.Errorf("%w: %v", errInvalidChain, err)
				}
				headers = filled[proced:]
				from += uint64(proced)
			} else {
				// If we're closing in on the chain head, but haven't yet reached it, delay
				// the last few headers so mini reorgs on the head don't cause invalid hash
				// chain errors.
				if n := len(headers); n > 0 {
					// Retrieve the current head we're at
					var head uint64
					if mode == LightSync {
						head = *d.lightchain.GetLastFinalizedHeader().Number
					} else {
						full := d.blockchain.GetLastFinalizedNumber()
						headBlock := d.blockchain.GetLastFinalizedFastBlock()
						if headBlock.Number() != nil {
							head = *headBlock.Number()
						}
						if head < full {
							head = full
						}
					}
					// If the head is below the common ancestor, we're actually deduplicating
					// already existing chain segments, so use the ancestor as the fake head.
					// Otherwise we might end up delaying header deliveries pointlessly.
					if head < ancestor {
						head = ancestor
					}
					// If the head is way older than this batch, delay the last few headers
					if head+uint64(reorgProtThreshold) < *headers[n-1].Number {
						delay := reorgProtHeaderDelay
						if delay > n {
							delay = n
						}
						headers = headers[:n-delay]
					}
				}
			}
			// Insert all the new headers and fetch the next batch
			if len(headers) > 0 {
				p.log.Info("Scheduling new headers", "count", len(headers), "from", from)
				select {
				case d.headerProcCh <- headers:
				case <-d.cancelCh:
					return errCanceled
				}
				from += uint64(len(headers))

				// If we're still skeleton filling fast sync, check pivot staleness
				// before continuing to the next skeleton filling
				if skeleton && pivot > 0 {
					getNextPivot()
				} else {
					getHeaders(from)
					log.Info("Synchronising is ending", "peer", p.id, "eth", p.version, "mode", d.mode, "from", from)
				}
			} else {
				// No headers delivered, or all of them being delayed, sleep a bit and retry
				p.log.Info("All headers delayed, waiting")
				select {
				case <-time.After(fsHeaderContCheck):
					getHeaders(from)
					continue
				case <-d.cancelCh:
					return errCanceled
				}
			}

		case <-timeout.C:
			if d.dropPeer == nil {
				// The dropPeer method is nil when `--copydb` is used for a local copy.
				// Timeouts can occur if e.g. compaction hits at the wrong time, and can be ignored
				p.log.Warn("Downloader wants to drop peer, but peerdrop-function is not set", "peer", p.id)
				break
			}
			// Header retrieval timed out, consider the peer bad and drop
			p.log.Debug("Header request timed out", "elapsed", ttl)
			headerTimeoutMeter.Mark(1)
			d.dropPeer(p.id)

			// Finish the sync gracefully instead of dumping the gathered data though
			for _, ch := range []chan bool{d.bodyWakeCh, d.receiptWakeCh} {
				select {
				case ch <- false:
				case <-d.cancelCh:
				}
			}
			select {
			case d.headerProcCh <- nil:
			case <-d.cancelCh:
			}
			return fmt.Errorf("%w: header request timed out", errBadPeer)
		}
	}
}

// fillHeaderSkeleton concurrently retrieves headers from all our available peers
// and maps them to the provided skeleton header chain.
//
// Any partial results from the beginning of the skeleton is (if possible) forwarded
// immediately to the header processor to keep the rest of the pipeline full even
// in the case of header stalls.
//
// The method returns the entire filled skeleton and also the number of headers
// already forwarded for processing.
func (d *Downloader) fillHeaderSkeleton(from uint64, skeleton []*types.Header) ([]*types.Header, int, error) {
	log.Info("Filling up skeleton", "from", from)
	d.queue.ScheduleSkeleton(from, skeleton)

	var (
		deliver = func(packet dataPack) (int, error) {
			pack := packet.(*headerPack)
			return d.queue.DeliverHeaders(pack.peerID, pack.headers, d.headerProcCh)
		}
		expire  = func() map[string]int { return d.queue.ExpireHeaders(d.peers.rates.TargetTimeout()) }
		reserve = func(p *peerConnection, count int) (*fetchRequest, bool, bool) {
			return d.queue.ReserveHeaders(p, count), false, false
		}
		fetch    = func(p *peerConnection, req *fetchRequest) error { return p.FetchHeaders(req.From, MaxHeaderFetch) }
		capacity = func(p *peerConnection) int { return p.HeaderCapacity(d.peers.rates.TargetRoundTrip()) }
		setIdle  = func(p *peerConnection, accepted int, deliveryTime time.Time) {
			p.SetHeadersIdle(accepted, deliveryTime)
		}
	)
	err := d.fetchParts(d.headerCh, deliver, d.queue.headerContCh, expire,
		d.queue.PendingHeaders, d.queue.InFlightHeaders, reserve,
		nil, fetch, d.queue.CancelHeaders, capacity, d.peers.HeaderIdlePeers, setIdle, "headers")

	log.Debug("Skeleton fill terminated", "err", err)

	filled, proced := d.queue.RetrieveHeaders()
	return filled, proced, err
}

// fetchBodies iteratively downloads the scheduled block bodies, taking any
// available peers, reserving a chunk of blocks for each, waiting for delivery
// and also periodically checking for timeouts.
func (d *Downloader) fetchBodies(from uint64) error {
	log.Debug("Downloading block bodies", "origin", from)

	var (
		deliver = func(packet dataPack) (int, error) {
			pack := packet.(*bodyPack)
			return d.queue.DeliverBodies(pack.peerID, pack.transactions)
		}
		expire   = func() map[string]int { return d.queue.ExpireBodies(d.peers.rates.TargetTimeout()) }
		fetch    = func(p *peerConnection, req *fetchRequest) error { return p.FetchBodies(req) }
		capacity = func(p *peerConnection) int { return p.BlockCapacity(d.peers.rates.TargetRoundTrip()) }
		setIdle  = func(p *peerConnection, accepted int, deliveryTime time.Time) { p.SetBodiesIdle(accepted, deliveryTime) }
	)
	err := d.fetchParts(d.bodyCh, deliver, d.bodyWakeCh, expire,
		d.queue.PendingBlocks, d.queue.InFlightBlocks, d.queue.ReserveBodies,
		d.bodyFetchHook, fetch, d.queue.CancelBodies, capacity, d.peers.BodyIdlePeers, setIdle, "bodies")

	log.Debug("Block body download terminated", "err", err)
	return err
}

// fetchReceipts iteratively downloads the scheduled block receipts, taking any
// available peers, reserving a chunk of receipts for each, waiting for delivery
// and also periodically checking for timeouts.
func (d *Downloader) fetchReceipts(from uint64) error {
	log.Debug("Downloading transaction receipts", "origin", from)

	var (
		deliver = func(packet dataPack) (int, error) {
			pack := packet.(*receiptPack)
			return d.queue.DeliverReceipts(pack.peerID, pack.receipts)
		}
		expire   = func() map[string]int { return d.queue.ExpireReceipts(d.peers.rates.TargetTimeout()) }
		fetch    = func(p *peerConnection, req *fetchRequest) error { return p.FetchReceipts(req) }
		capacity = func(p *peerConnection) int { return p.ReceiptCapacity(d.peers.rates.TargetRoundTrip()) }
		setIdle  = func(p *peerConnection, accepted int, deliveryTime time.Time) {
			p.SetReceiptsIdle(accepted, deliveryTime)
		}
	)
	err := d.fetchParts(d.receiptCh, deliver, d.receiptWakeCh, expire,
		d.queue.PendingReceipts, d.queue.InFlightReceipts, d.queue.ReserveReceipts,
		d.receiptFetchHook, fetch, d.queue.CancelReceipts, capacity, d.peers.ReceiptIdlePeers, setIdle, "receipts")

	log.Debug("Transaction receipt download terminated", "err", err)
	return err
}

// fetchParts iteratively downloads scheduled block parts, taking any available
// peers, reserving a chunk of fetch requests for each, waiting for delivery and
// also periodically checking for timeouts.
//
// As the scheduling/timeout logic mostly is the same for all downloaded data
// types, this method is used by each for data gathering and is instrumented with
// various callbacks to handle the slight differences between processing them.
//
// The instrumentation parameters:
//   - errCancel:   error type to return if the fetch operation is cancelled (mostly makes logging nicer)
//   - deliveryCh:  channel from which to retrieve downloaded data packets (merged from all concurrent peers)
//   - deliver:     processing callback to deliver data packets into type specific download queues (usually within `queue`)
//   - wakeCh:      notification channel for waking the fetcher when new tasks are available (or sync completed)
//   - expire:      task callback method to abort requests that took too long and return the faulty peers (traffic shaping)
//   - pending:     task callback for the number of requests still needing download (detect completion/non-completability)
//   - inFlight:    task callback for the number of in-progress requests (wait for all active downloads to finish)
//   - throttle:    task callback to check if the processing queue is full and activate throttling (bound memory use)
//   - reserve:     task callback to reserve new download tasks to a particular peer (also signals partial completions)
//   - fetchHook:   tester callback to notify of new tasks being initiated (allows testing the scheduling logic)
//   - fetch:       network callback to actually send a particular download request to a physical remote peer
//   - cancel:      task callback to abort an in-flight download request and allow rescheduling it (in case of lost peer)
//   - capacity:    network callback to retrieve the estimated type-specific bandwidth capacity of a peer (traffic shaping)
//   - idle:        network callback to retrieve the currently (type specific) idle peers that can be assigned tasks
//   - setIdle:     network callback to set a peer back to idle and update its estimated capacity (traffic shaping)
//   - kind:        textual label of the type being downloaded to display in log messages
func (d *Downloader) fetchParts(
	deliveryCh chan dataPack,
	deliver func(dataPack) (int, error),
	wakeCh chan bool,
	expire func() map[string]int,
	pending func() int,
	inFlight func() bool,
	reserve func(*peerConnection, int) (*fetchRequest, bool, bool),
	fetchHook func([]*types.Header),
	fetch func(*peerConnection, *fetchRequest) error,
	cancel func(*fetchRequest),
	capacity func(*peerConnection) int,
	idle func() ([]*peerConnection, int),
	setIdle func(*peerConnection, int, time.Time),
	kind string,
) error {

	// Create a ticker to detect expired retrieval tasks
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	update := make(chan struct{}, 1)

	// Prepare the queue and fetch block parts until the block header fetcher's done
	finished := false
	for {
		select {
		case <-d.cancelCh:
			return errCanceled

		case packet := <-deliveryCh:
			log.Debug("Downloader fetched parts", "kind", kind, "packet", packet)
			deliveryTime := time.Now()
			// If the peer was previously banned and failed to deliver its pack
			// in a reasonable time frame, ignore its message.
			if peer := d.peers.Peer(packet.PeerId()); peer != nil {
				// Deliver the received chunk of data and check chain validity
				accepted, err := deliver(packet)
				if errors.Is(err, errInvalidChain) {
					return err
				}
				// Unless a peer delivered something completely else than requested (usually
				// caused by a timed out request which came through in the end), set it to
				// idle. If the delivery's stale, the peer should have already been idled.
				if !errors.Is(err, errStaleDelivery) {
					setIdle(peer, accepted, deliveryTime)
				}
				// Issue a log to the user to see what's going on
				switch {
				case err == nil && packet.Items() == 0:
					peer.log.Trace("Requested data not delivered", "type", kind)
				case err == nil:
					peer.log.Trace("Delivered new batch of data", "type", kind, "count", packet.Stats())
				default:
					peer.log.Debug("Failed to deliver retrieved data", "type", kind, "err", err)
				}
			}
			// Blocks assembled, try to update the progress
			select {
			case update <- struct{}{}:
			default:
			}

		case cont := <-wakeCh:
			// The header fetcher sent a continuation flag, check if it's done
			if !cont {
				finished = true
			}
			// Headers arrive, try to update the progress
			select {
			case update <- struct{}{}:
			default:
			}

		case <-ticker.C:
			// Sanity check update the progress
			select {
			case update <- struct{}{}:
			default:
			}

		case <-update:
			// Short circuit if we lost all our peers
			if d.peers.Len() == 0 {
				return errNoPeers
			}
			// Check for fetch request timeouts and demote the responsible peers
			for pid, fails := range expire() {
				if peer := d.peers.Peer(pid); peer != nil {
					// If a lot of retrieval elements expired, we might have overestimated the remote peer or perhaps
					// ourselves. Only reset to minimal throughput but don't drop just yet. If even the minimal times
					// out that sync wise we need to get rid of the peer.
					//
					// The reason the minimum threshold is 2 is because the downloader tries to estimate the bandwidth
					// and latency of a peer separately, which requires pushing the measures capacity a bit and seeing
					// how response times reacts, to it always requests one more than the minimum (i.e. min 2).
					if fails > 2 {
						peer.log.Trace("Data delivery timed out", "type", kind)
						setIdle(peer, 0, time.Now())
					} else {
						peer.log.Debug("Stalling delivery, dropping", "type", kind)

						if d.dropPeer == nil {
							// The dropPeer method is nil when `--copydb` is used for a local copy.
							// Timeouts can occur if e.g. compaction hits at the wrong time, and can be ignored
							peer.log.Warn("Downloader wants to drop peer, but peerdrop-function is not set", "peer", pid)
						} else {
							d.dropPeer(pid)

							// If this peer was the master peer, abort sync immediately
							d.cancelLock.RLock()
							master := pid == d.cancelPeer
							d.cancelLock.RUnlock()

							if master {
								d.cancel()
								return errTimeout
							}
						}
					}
				}
			}
			// If there's nothing more to fetch, wait or terminate
			if pending() == 0 {
				if !inFlight() && finished {
					log.Debug("Data fetching completed", "type", kind)
					return nil
				}
				break
			}
			// Send a download request to all idle peers, until throttled
			progressed, throttled, running := false, false, inFlight()
			idles, total := idle()
			pendCount := pending()
			for _, peer := range idles {
				// Short circuit if throttling activated
				if throttled {
					break
				}
				// Short circuit if there is no more available task.
				if pendCount = pending(); pendCount == 0 {
					break
				}
				// Reserve a chunk of fetches for a peer. A nil can mean either that
				// no more headers are available, or that the peer is known not to
				// have them.
				request, progress, throttle := reserve(peer, capacity(peer))
				if progress {
					progressed = true
				}
				if throttle {
					throttled = true
					throttleCounter.Inc(1)
				}
				if request == nil {
					continue
				}
				if request.From > 0 {
					peer.log.Trace("Requesting new batch of data", "type", kind, "from", request.From, "request", request)
				} else {
					peer.log.Trace("Requesting new batch of data", "type", kind, "count", len(request.Headers), "from.Number", request.Headers[0].Nr(), "from.Hash", request.Headers[0].Hash().Hex())
				}
				// Fetch the chunk and make sure any errors return the hashes to the queue
				if fetchHook != nil {
					fetchHook(request.Headers)
				}
				if err := fetch(peer, request); err != nil {
					// Although we could try and make an attempt to fix this, this error really
					// means that we've double allocated a fetch task to a peer. If that is the
					// case, the internal state of the downloader and the queue is very wrong so
					// better hard crash and note the error instead of silently accumulating into
					// a much bigger issue.
					panic(fmt.Sprintf("%v: %s fetch assignment failed", peer, kind))
				}
				running = true
			}
			// Make sure that we have peers available for fetching. If all peers have been tried
			// and all failed throw an error
			if !progressed && !throttled && !running && len(idles) == total && pendCount > 0 {
				return errPeersUnavailable
			}
		}
	}
}

// processHeaders takes batches of retrieved headers from an input channel and
// keeps processing and scheduling them into the header chain and downloader's
// queue until the stream ends or a failure occurs.
func (d *Downloader) processHeaders(origin uint64) error {
	// Keep a count of uncertain headers to roll back
	var (
		rollback    uint64 // Zero means no rollback (fine as you can't unroll the genesis)
		rollbackErr error
		mode        = d.getMode()
	)
	defer func() {
		if rollback > 0 {
			lastHeader, lastFastBlock, lastBlock := *d.lightchain.GetLastFinalizedHeader().Number, uint64(0), uint64(0)
			if mode != LightSync {
				lastFastBlock = *d.blockchain.GetLastFinalizedFastBlock().Number()
				lastBlock = d.blockchain.GetLastFinalizedNumber()
			}
			header := d.lightchain.GetHeaderByNumber(rollback - 1)
			if err := d.lightchain.SetHead(header.Hash()); err != nil { // -1 to target the parent of the first uncertain block
				// We're already unwinding the stack, only print the error to make it more visible
				log.Error("Failed to roll back chain segment", "head", rollback-1, "err", err)
			}
			curFastBlock, curBlock := uint64(0), uint64(0)
			if mode != LightSync {
				curFastBlock = *d.blockchain.GetLastFinalizedFastBlock().Number()
				curBlock = d.blockchain.GetLastFinalizedNumber()
			}
			log.Warn("Rolled back chain segment",
				"header", fmt.Sprintf("%d->%d", lastHeader, d.lightchain.GetLastFinalizedHeader().Nr()),
				"fast", fmt.Sprintf("%d->%d", lastFastBlock, curFastBlock),
				"block", fmt.Sprintf("%d->%d", lastBlock, curBlock), "reason", rollbackErr)
		}
	}()
	// Wait for batches of headers to process
	for {
		select {
		case <-d.cancelCh:
			rollbackErr = errCanceled
			return errCanceled

		case headers := <-d.headerProcCh:
			// Terminate header processing if we synced up
			if len(headers) == 0 {
				// Notify everyone that headers are fully processed
				for _, ch := range []chan bool{d.bodyWakeCh, d.receiptWakeCh} {
					select {
					case ch <- false:
					case <-d.cancelCh:
					}
				}
				// Disable any rollback and return
				rollback = 0
				return nil
			}
			// Otherwise split the chunk of headers into batches and process them
			for len(headers) > 0 {
				// Terminate if something failed in between processing chunks
				select {
				case <-d.cancelCh:
					rollbackErr = errCanceled
					return errCanceled
				default:
				}
				// Select the next chunk of headers to import
				limit := maxHeadersProcess
				if limit > len(headers) {
					limit = len(headers)
				}
				chunk := headers[:limit]

				// In case of header only syncing, validate the chunk immediately
				if mode == FastSync || mode == LightSync {
					// If we're importing pure headers, verify based on their recentness
					var pivot uint64

					d.pivotLock.RLock()
					if d.pivotHeader != nil {
						pivot = *d.pivotHeader.Number
					}
					d.pivotLock.RUnlock()

					frequency := fsHeaderCheckFrequency
					if *chunk[len(chunk)-1].Number+uint64(fsHeaderForceVerify) > pivot {
						frequency = 1
					}
					if n, err := d.lightchain.InsertHeaderChain(chunk, frequency); err != nil {
						rollbackErr = err

						// If some headers were inserted, track them as uncertain
						if (mode == FastSync || frequency > 1) && n > 0 && rollback == 0 {
							rollback = *chunk[0].Number
						}
						log.Warn("Invalid header encountered", "number", chunk[n].Nr(), "height", chunk[n].Height, "hash", chunk[n].Hash(), "parent", chunk[n].ParentHashes, "err", err)
						return fmt.Errorf("%w: %v", errInvalidChain, err)
					}
					// All verifications passed, track all headers within the alloted limits
					if mode == FastSync {
						head := *chunk[len(chunk)-1].Number
						if head-rollback > uint64(fsHeaderSafetyNet) {
							rollback = head - uint64(fsHeaderSafetyNet)
						} else {
							rollback = 1
						}
					}
				}
				// Unless we're doing light chains, schedule the headers for associated content retrieval
				if mode == FullSync || mode == FastSync {
					// If we've reached the allowed number of pending headers, stall a bit
					for d.queue.PendingBlocks() >= maxQueuedHeaders || d.queue.PendingReceipts() >= maxQueuedHeaders {
						select {
						case <-d.cancelCh:
							rollbackErr = errCanceled
							return errCanceled
						case <-time.After(time.Second):
						}
					}
					// Otherwise insert the headers for content retrieval
					inserts := d.queue.Schedule(chunk, origin)
					insLen := len(inserts)
					chLen := len(chunk)
					if insLen != chLen {
						rollbackErr = fmt.Errorf("stale headers: len inserts %v len(chunk) %v", len(inserts), len(chunk))
						return fmt.Errorf("%w: stale headers", errBadPeer)
					}
				}
				headers = headers[limit:]
				origin += uint64(limit)
			}
			// Update the highest block number we know if a higher one is found.
			d.syncStatsLock.Lock()
			if d.syncStatsChainHeight < origin {
				d.syncStatsChainHeight = origin - 1
			}
			d.syncStatsLock.Unlock()

			// Signal the content downloaders of the availablility of new tasks
			for _, ch := range []chan bool{d.bodyWakeCh, d.receiptWakeCh} {
				select {
				case ch <- true:
				default:
				}
			}
		}
	}
}

// processFullSyncContent takes fetch results from the queue and imports them into the chain.
func (d *Downloader) processFullSyncContent() error {
	for {
		results := d.queue.Results(true)
		if len(results) == 0 {
			return nil
		}
		if d.chainInsertHook != nil {
			d.chainInsertHook(results)
		}
		if err := d.importBlockResults(results); err != nil {
			return err
		}
	}
}

func (d *Downloader) importBlockResults(results []*fetchResult) error {
	// Check for any early termination requests
	if len(results) == 0 {
		return nil
	}
	select {
	case <-d.quitCh:
		return errCancelContentProcessing
	default:
	}
	// Retrieve the a batch of results to import
	first, last := results[0].Header, results[len(results)-1].Header
	log.Debug("Inserting downloaded chain", "items", len(results), "firsthash", first.Hash().Hex(), "lasthash", last.Hash().Hex())
	blocks := make([]*types.Block, len(results))
	for i, result := range results {
		bh := types.NewBlockWithHeader(result.Header)
		blocks[i] = bh.WithBody(result.Transactions)
	}

	//if index, err := d.blockchain.InsertChain(blocks); err != nil {
	if index, err := d.blockchain.SyncInsertChain(blocks); err != nil {
		if index < len(results) {
			log.Error("Downloaded item processing failed", "Nr", blocks[0].Nr(), "Height", blocks[0].Height(), "hash", blocks[0].Hash().Hex(), "err", err)
			log.Error("Downloaded item processing failed", "Nr", results[index].Header.Nr(), "Height", results[index].Header.Height, "hash", results[index].Header.Hash().Hex(), "err", err)
		} else {
			// The InsertChain method in blockchain.go will sometimes return an out-of-bounds index,
			// when it needs to preprocess blocks to import a sidechain.
			// The importer will put together a new list of blocks to import, which is a superset
			// of the blocks delivered from the downloader, and the indexing will be off.
			log.Error("Downloaded item processing failed on sidechain import", "index", index, "err", err)
		}
		return fmt.Errorf("%w: %v", errInvalidChain, err)
	} else {
		lastFinNr := d.blockchain.GetLastFinalizedNumber()
		log.Info("Synchronised part of finalized chain", "mode", d.mode, "lastFinNr", lastFinNr)
	}
	return nil
}

// processFastSyncContent takes fetch results from the queue and writes them to the
// database. It also controls the synchronisation of state nodes of the pivot block.
func (d *Downloader) processFastSyncContent() error {
	// Start syncing state of the reported head block. This should get us most of
	// the state of the pivot block.
	d.pivotLock.RLock()
	sync := d.syncState(d.pivotHeader.Root)
	d.pivotLock.RUnlock()

	defer func() {
		// The `sync` object is replaced every time the pivot moves. We need to
		// defer close the very last active one, hence the lazy evaluation vs.
		// calling defer sync.Cancel() !!!
		sync.Cancel()
	}()

	closeOnErr := func(s *stateSync) {
		if err := s.Wait(); err != nil && err != errCancelStateFetch && err != errCanceled && err != snap.ErrCancelled {
			d.queue.Close() // wake up Results
		}
	}
	go closeOnErr(sync)

	// To cater for moving pivot points, track the pivot block and subsequently
	// accumulated download results separately.
	var (
		oldPivot *fetchResult   // Locked in pivot block, might change eventually
		oldTail  []*fetchResult // Downloaded content after the pivot
	)
	for {
		// Wait for the next batch of downloaded data to be available, and if the pivot
		// block became stale, move the goalpost
		results := d.queue.Results(oldPivot == nil) // Block if we're not monitoring pivot staleness
		if len(results) == 0 {
			// If pivot sync is done, stop
			if oldPivot == nil {
				return sync.Cancel()
			}
			// If sync failed, stop
			select {
			case <-d.cancelCh:
				sync.Cancel()
				return errCanceled
			default:
			}
		}
		if d.chainInsertHook != nil {
			d.chainInsertHook(results)
		}
		// If we haven't downloaded the pivot block yet, check pivot staleness
		// notifications from the header downloader
		d.pivotLock.RLock()
		pivot := d.pivotHeader
		d.pivotLock.RUnlock()

		if oldPivot == nil {
			if pivot.Root != sync.root {
				sync.Cancel()
				sync = d.syncState(pivot.Root)

				go closeOnErr(sync)
			}
		} else {
			results = append(append([]*fetchResult{oldPivot}, oldTail...), results...)
		}
		// Split around the pivot block and process the two sides via fast/full sync
		if atomic.LoadInt32(&d.committed) == 0 {
			latest := results[len(results)-1].Header
			// If the height is above the pivot block by 2 sets, it means the pivot
			// become stale in the network and it was garbage collected, move to a
			// new pivot.
			//
			// Note, we have `reorgProtHeaderDelay` number of blocks withheld, Those
			// need to be taken into account, otherwise we're detecting the pivot move
			// late and will drop peers due to unavailable state!!!
			if height := *latest.Number; height >= *pivot.Number+2*uint64(fsMinFullBlocks)-uint64(reorgProtHeaderDelay) {
				log.Warn("Pivot became stale, moving", "old", pivot.Nr(), "new", height-uint64(fsMinFullBlocks)+uint64(reorgProtHeaderDelay))
				pivot = results[len(results)-1-fsMinFullBlocks+reorgProtHeaderDelay].Header // must exist as lower old pivot is uncommitted

				d.pivotLock.Lock()
				d.pivotHeader = pivot
				d.pivotLock.Unlock()

				// Write out the pivot into the database so a rollback beyond it will
				// reenable fast sync
				rawdb.WriteLastPivotNumber(d.stateDB, *pivot.Number)
			}
		}
		P, beforeP, afterP := splitAroundPivot(*pivot.Number, results)
		if err := d.commitFastSyncData(beforeP, sync); err != nil {
			return err
		}
		if P != nil {
			// If new pivot block found, cancel old state retrieval and restart
			if oldPivot != P {
				sync.Cancel()
				sync = d.syncState(P.Header.Root)

				go closeOnErr(sync)
				oldPivot = P
			}
			// Wait for completion, occasionally checking for pivot staleness
			select {
			case <-sync.done:
				if sync.err != nil {
					return sync.err
				}
				if err := d.commitPivotBlock(P); err != nil {
					return err
				}
				oldPivot = nil

			case <-time.After(time.Second):
				oldTail = afterP
				continue
			}
		}
		// Fast sync done, pivot commit done, full import
		if err := d.importBlockResults(afterP); err != nil {
			return err
		}
	}
}

func splitAroundPivot(pivot uint64, results []*fetchResult) (p *fetchResult, before, after []*fetchResult) {
	if len(results) == 0 {
		return nil, nil, nil
	}
	if lastNum := *results[len(results)-1].Header.Number; lastNum < pivot {
		// the pivot is somewhere in the future
		return nil, results, nil
	}
	// This can also be optimized, but only happens very seldom
	for _, result := range results {
		num := *result.Header.Number
		switch {
		case num < pivot:
			before = append(before, result)
		case num == pivot:
			p = result
		default:
			after = append(after, result)
		}
	}
	return p, before, after
}

func (d *Downloader) commitFastSyncData(results []*fetchResult, stateSync *stateSync) error {
	// Check for any early termination requests
	if len(results) == 0 {
		return nil
	}
	select {
	case <-d.quitCh:
		return errCancelContentProcessing
	case <-stateSync.done:
		if err := stateSync.Wait(); err != nil {
			return err
		}
	default:
	}
	// Retrieve the a batch of results to import
	first, last := results[0].Header, results[len(results)-1].Header
	log.Debug("Inserting fast-sync blocks", "items", len(results),
		"firstnum", first.Nr(), "firsthash", first.Hash(),
		"lastnumn", last.Nr(), "lasthash", last.Hash(),
	)
	blocks := make([]*types.Block, len(results))
	receipts := make([]types.Receipts, len(results))
	for i, result := range results {
		blocks[i] = types.NewBlockWithHeader(result.Header).WithBody(result.Transactions)
		receipts[i] = result.Receipts
	}
	if index, err := d.blockchain.InsertReceiptChain(blocks, receipts, d.ancientLimit); err != nil {
		log.Error("Downloaded item processing failed", "number", results[index].Header.Nr(), "hash", results[index].Header.Hash().Hex(), "err", err)
		return fmt.Errorf("%w: %v", errInvalidChain, err)
	}
	return nil
}

func (d *Downloader) commitPivotBlock(result *fetchResult) error {
	block := types.NewBlockWithHeader(result.Header).WithBody(result.Transactions)
	log.Debug("Committing fast sync pivot as new head", "number", block.Nr(), "hash", block.Hash().Hex())

	// Commit the pivot block as the new head, will require full sync from here on
	if _, err := d.blockchain.InsertReceiptChain([]*types.Block{block}, []types.Receipts{result.Receipts}, d.ancientLimit); err != nil {
		return err
	}
	if err := d.blockchain.FastSyncCommitHead(block.Hash()); err != nil {
		return err
	}
	atomic.StoreInt32(&d.committed, 1)

	// If we had a bloom filter for the state sync, deallocate it now. Note, we only
	// deallocate internally, but keep the empty wrapper. This ensures that if we do
	// a rollback after committing the pivot and restarting fast sync, we don't end
	// up using a nil bloom. Empty bloom is fine, it just returns that it does not
	// have the info we need, so reach down to the database instead.
	if d.stateBloom != nil {
		d.stateBloom.Close()
	}
	return nil
}

// DeliverDag injects a dag chain received from a remote node.
func (d *Downloader) DeliverDag(id string, dag common.HashArray) error {
	return d.deliver(d.dagCh, &dagPack{id, dag}, dagInMeter, dagDropMeter)
}

// DeliverHeaders injects a new batch of block headers received from a remote
// node into the download schedule.
func (d *Downloader) DeliverHeaders(id string, headers []*types.Header) error {
	return d.deliver(d.headerCh, &headerPack{id, headers}, headerInMeter, headerDropMeter)
}

// DeliverBodies injects a new batch of block bodies received from a remote node.
func (d *Downloader) DeliverBodies(id string, transactions [][]*types.Transaction) error {
	return d.deliver(d.bodyCh, &bodyPack{id, transactions}, bodyInMeter, bodyDropMeter)
}

// DeliverReceipts injects a new batch of receipts received from a remote node.
func (d *Downloader) DeliverReceipts(id string, receipts [][]*types.Receipt) error {
	return d.deliver(d.receiptCh, &receiptPack{id, receipts}, receiptInMeter, receiptDropMeter)
}

// DeliverNodeData injects a new batch of node state data received from a remote node.
func (d *Downloader) DeliverNodeData(id string, data [][]byte) error {
	return d.deliver(d.stateCh, &statePack{id, data}, stateInMeter, stateDropMeter)
}

// DeliverSnapPacket is invoked from a peer's message handler when it transmits a
// data packet for the local node to consume.
func (d *Downloader) DeliverSnapPacket(peer *snap.Peer, packet snap.Packet) error {
	switch packet := packet.(type) {
	case *snap.AccountRangePacket:
		hashes, accounts, err := packet.Unpack()
		if err != nil {
			return err
		}
		return d.SnapSyncer.OnAccounts(peer, packet.ID, hashes, accounts, packet.Proof)

	case *snap.StorageRangesPacket:
		hashset, slotset := packet.Unpack()
		return d.SnapSyncer.OnStorage(peer, packet.ID, hashset, slotset, packet.Proof)

	case *snap.ByteCodesPacket:
		return d.SnapSyncer.OnByteCodes(peer, packet.ID, packet.Codes)

	case *snap.TrieNodesPacket:
		return d.SnapSyncer.OnTrieNodes(peer, packet.ID, packet.Nodes)

	default:
		return fmt.Errorf("unexpected snap packet type: %T", packet)
	}
}

// deliver injects a new batch of data received from a remote node.
func (d *Downloader) deliver(destCh chan dataPack, packet dataPack, inMeter, dropMeter metrics.Meter) (err error) {
	// Update the delivery metrics for both good and failed deliveries
	inMeter.Mark(int64(packet.Items()))
	defer func() {
		if err != nil {
			dropMeter.Mark(int64(packet.Items()))
		}
	}()
	// Deliver or abort if the sync is canceled while queuing
	d.cancelLock.RLock()
	cancel := d.cancelCh
	d.cancelLock.RUnlock()
	if cancel == nil {
		log.Error("Sync deliver error", "err", errNoSyncActive)
		return errNoSyncActive
	}
	select {
	case destCh <- packet:
		return nil
	case <-cancel:
		return errNoSyncActive
	}
}

// fetchDagHashes retrieves the dag chain hashes beginning from finalized block (excluded from response).
func (d *Downloader) fetchDagHashesBySlots(p *peerConnection, from, to uint64) (dag common.HashArray, err error) {
	//slots limit 1024
	slotsLimit := eth.LimitDagHashes / d.lightchain.Config().ValidatorsPerSlot
	if to-from > slotsLimit {
		return nil, errDataSizeLimitExceeded
	}
	p.log.Info("Retrieving remote dag hashes by slot: start", "from", from, "to", to)

	go p.peer.RequestHashesBySlots(from, to)

	ttl := d.peers.rates.TargetTimeout()
	timeout := time.After(ttl)

	for {
		err = nil
		select {
		case <-d.cancelCh:
			return nil, errCanceled
		case packet := <-d.dagCh:
			// Discard anything not from the origin peer
			if packet.PeerId() != p.id {
				log.Warn("Received dag from incorrect peer", "peer", packet.PeerId(), "fn", "sync:RequestHashesBySlots")
				break
			}
			dag = packet.(*dagPack).dag
			if len(dag) == 0 {
				err = errInvalidDag
			}
			return dag, err
		case <-timeout:
			p.log.Debug("Waiting for dag timed out", "elapsed", ttl)
			return nil, errTimeout
		case header := <-d.headerCh:
			p.log.Warn("Out of bounds delivery, ignore: <-d.headerCh:", "elapsed", ttl, "header", header, "fn", "sync:RequestHashesBySlots")
		case body := <-d.bodyCh:
			p.log.Warn("Out of bounds delivery, ignore: <-d.bodyCh:", "elapsed", ttl, "body", body, "fn", "sync:RequestHashesBySlots")
		case receipt := <-d.receiptCh:
			p.log.Warn("Out of bounds delivery, ignore: <-d.receiptCh:", "elapsed", ttl, "receipt", receipt, "fn", "sync:RequestHashesBySlots")
			// Out of bounds delivery, ignore
		}
	}
}

// fetchDagHashes retrieves the dag chain hashes beginning from finalized block (excluded from response).
func (d *Downloader) fetchDagHashes(p *peerConnection, baseSpine common.Hash, terminalSpine common.Hash) (dag common.HashArray, err error) {
	p.log.Info("Retrieving remote dag hashes: start",
		"baseSpine", fmt.Sprintf("%#x", baseSpine),
		"terminalSpine", fmt.Sprintf("%#x", terminalSpine),
	)

	go p.peer.RequestDag(baseSpine, terminalSpine)

	ttl := d.peers.rates.TargetTimeout()
	timeout := time.After(ttl)

	for {
		err = nil
		select {
		case <-d.cancelCh:
			return nil, errCanceled

		case packet := <-d.dagCh:
			// Discard anything not from the origin peer
			if packet.PeerId() != p.id {
				log.Warn("Received dag from incorrect peer", "peer", packet.PeerId())
				break
			}
			dag = packet.(*dagPack).dag
			if len(dag) == 0 {
				err = errInvalidDag
			}
			return dag, err
		case <-timeout:
			p.log.Debug("Waiting for dag timed out", "elapsed", ttl)
			return nil, errTimeout
		case header := <-d.headerCh:
			p.log.Warn("Out of bounds delivery, ignore: <-d.headerCh:", "elapsed", ttl, "header", header)
		case body := <-d.bodyCh:
			p.log.Warn("Out of bounds delivery, ignore: <-d.bodyCh:", "elapsed", ttl, "body", body)
		case receipt := <-d.receiptCh:
			p.log.Warn("Out of bounds delivery, ignore: <-d.receiptCh:", "elapsed", ttl, "receipt", receipt)
			// Out of bounds delivery, ignore
		}
	}
}

// fetchDagHeaders retrieves the dag headers by hashes from a remote peer.
func (d *Downloader) fetchDagHeaders(p *peerConnection, hashes common.HashArray) (headers []*types.Header, err error) {
	p.log.Info("Retrieving remote dag headers: start", "hashes", len(hashes))
	go p.peer.RequestHeadersByHashes(hashes)

	ttl := d.peers.rates.TargetTimeout()
	timeout := time.After(ttl)

	for {
		err = nil
		select {
		case <-d.cancelCh:
			return nil, errCanceled

		case packet := <-d.headerCh:
			// Discard anything not from the origin peer
			if packet.PeerId() != p.id {
				log.Warn("Received dag header from incorrect peer", "peer", packet.PeerId())
				break
			}
			headers = packet.(*headerPack).headers
			if len(headers) == 0 {
				err = errInvalidDag
			}
			return headers, err

		case <-timeout:
			p.log.Debug("Waiting for dag timed out", "elapsed", ttl)
			return nil, errTimeout
		}
	}
}

// fetchDagTxs retrieves the dag transactions by blocks hashes from a remote peer.
func (d *Downloader) fetchDagTxs(p *peerConnection, hashes common.HashArray) (map[common.Hash][]*types.Transaction, error) {

	p.log.Info("Retrieving remote dag block txs: start", "hashes", len(hashes))

	go p.peer.RequestBodies(hashes)

	ttl := d.peers.rates.TargetTimeout()
	timeout := time.After(ttl)

	for {
		select {
		case <-d.cancelCh:
			return nil, errCanceled

		case packet := <-d.bodyCh:
			// Discard anything not from the origin peer
			if packet.PeerId() != p.id {
				log.Warn("Received dag header from incorrect peer", "peer", packet.PeerId())
				break
			}
			transactions := packet.(*bodyPack).transactions
			txsMap := make(map[common.Hash][]*types.Transaction, len(hashes))
			for i, txs := range transactions {
				txsMap[hashes[i]] = txs
			}
			return txsMap, nil
		case <-timeout:
			p.log.Debug("Waiting for dag timed out", "elapsed", ttl)
			return nil, errTimeout
		}
	}
}

func (d *Downloader) SyncChainBySpines(baseSpine common.Hash, spines common.HashArray, finEpoch uint64) (fullySynced bool, err error) {
	log.Info("Sync chain by spines", "baseSpine", baseSpine.Hex(), "spines", len(spines), "len(d.peers)", len(d.peers.peers))
	if d.peers.Len() == 0 {
		log.Error("Sync chain by spines", "baseSpine", baseSpine.Hex(), "spines", spines, "peers.Len", d.peers.Len(), "err", errNoPeers)
		return false, errNoPeers
	}

	//select peer
	for _, con := range d.peers.AllPeers() {
		fullySynced, err = d.peerSyncChainBySpines(con, baseSpine, spines, finEpoch)
		switch err {
		case nil:
			return fullySynced, nil
		case errInvalidBaseSpine:
			return false, err
		case errBusy, errCanceled:
			log.Warn("Synchronisation failed, trying next peer", "err", err)
			continue
		}
		if errors.Is(err, errInvalidChain) || errors.Is(err, errBadPeer) || errors.Is(err, errTimeout) ||
			errors.Is(err, errStallingPeer) || errors.Is(err, errUnsyncedPeer) || errors.Is(err, errEmptyHeaderSet) ||
			errors.Is(err, errPeersUnavailable) || errors.Is(err, errTooOld) || errors.Is(err, errInvalidAncestor) {
			log.Warn("Synchronisation failed, dropping peer", "peer", con.id, "err", err)
			if d.dropPeer == nil {
				// The dropPeer method is nil when `--copydb` is used for a local copy.
				// Timeouts can occur if e.g. compaction hits at the wrong time, and can be ignored
				log.Warn("Downloader wants to drop peer, but peerdrop-function is not set", "peer", con.id)
			} else {
				d.dropPeer(con.id)
			}
		}
		log.Warn("Synchronisation failed, trying next peer", "err", err)
	}
	return false, errNoPeers
}

func (d *Downloader) peerSyncChainBySpines(p *peerConnection, baseSpine common.Hash, spines common.HashArray, finEpoch uint64) (fullySynced bool, err error) {

	if d.Synchronising() {
		log.Warn("Synchronization canceled (synchronise process busy)")
		return false, errBusy
	}
	// Make sure only one goroutine is ever allowed past this point at once
	if !atomic.CompareAndSwapInt32(&d.dagSyncing, 0, 1) {
		return false, errBusy
	}
	// Make sure only one goroutine is ever allowed past this point at once
	if !atomic.CompareAndSwapInt32(&d.finSyncing, 0, 1) {
		return false, errBusy
	}
	defer func(start time.Time) {
		atomic.StoreInt32(&d.dagSyncing, 0)
		atomic.StoreInt32(&d.finSyncing, 0)
		log.Info("Synchronisation of dag chain terminated", "elapsed", common.PrettyDuration(time.Since(start)))
	}(time.Now())

	// Post a user notification of the sync (only once per session)
	if atomic.CompareAndSwapInt32(&d.notified, 0, 1) {
		log.Info("Block synchronisation started")
	}

	// If we are already full syncing, but have a fast-sync bloom filter laying
	// around, make sure it doesn't use memory any more. This is a special case
	// when the user attempts to fast sync a new empty network.
	if d.stateBloom != nil {
		d.stateBloom.Close()
	}

	// Reset the queue, peer set and wake channels to clean any internal leftover state
	d.queue.Reset(blockCacheMaxItems, blockCacheInitialItems)
	d.peers.Reset()
	for _, ch := range []chan bool{d.bodyWakeCh, d.receiptWakeCh} {
		select {
		case <-ch:
		default:
		}
	}
	for _, ch := range []chan dataPack{d.headerCh, d.bodyCh, d.receiptCh, d.dagCh} {
		for empty := false; !empty; {
			select {
			case <-ch:
			default:
				empty = true
			}
		}
	}
	for empty := false; !empty; {
		select {
		case <-d.headerProcCh:
		default:
			empty = true
		}
	}
	// Create cancel channel for aborting mid-flight and mark the master peer
	d.cancelLock.Lock()
	d.cancelCh = make(chan struct{})
	d.cancelPeer = p.id
	d.cancelLock.Unlock()
	defer d.Cancel() // No matter what, we can't leave the cancel channel open

	// Atomically set the requested sync mode
	atomic.StoreUint32(&d.mode, uint32(FullSync))

	d.mux.Post(StartEvent{})
	defer func() {
		// reset on error
		if err != nil {
			d.mux.Post(FailedEvent{err})
		} else {
			latest := d.lightchain.GetLastFinalizedHeader()
			d.mux.Post(DoneEvent{latest})
		}
	}()
	if p.version < eth.ETH66 {
		return false, fmt.Errorf("%w: advertized %d < required %d", errTooOld, p.version, eth.ETH66)
	}
	mode := d.getMode()

	defer func(start time.Time) {
		d.Cancel()
		log.Info("Synchronisation terminated", "elapsed", common.PrettyDuration(time.Since(start)))
	}(time.Now())

	log.Info("Synchronising with the network", "peer", p.id, "eth", p.version, "mode", mode, "baseSpine", baseSpine.Hex(), "spines", spines)

	//Synchronisation of dag chain
	if fullySynced, err = d.peerSyncDagChain(p, baseSpine, spines, finEpoch); err != nil {
		log.Error("Synchronization of dag chain failed", "err", err)
		d.Cancel()
		return fullySynced, err
	}
	d.Cancel()
	return fullySynced, nil
}

func (d *Downloader) checkPeer(p *peerConnection, baseSpine common.Hash, spines common.HashArray) (isPeerAcceptable bool, terminalRemote *types.Header, err error) {
	// check remote header
	baseHeader := d.blockchain.GetHeaderByHash(baseSpine)
	if baseHeader == nil || baseHeader.Height > 0 && baseHeader.Nr() == 0 {
		return false, nil, errInvalidBaseSpine
	}
	baseNr := baseHeader.Nr()
	baseRemote, err := d.fetchHeaderByNr(p, baseNr)
	if baseRemote.Hash() != baseHeader.Hash() || baseRemote.Root != baseHeader.Root {
		return false, nil, errBadPeer
	}
	tesminalSpine := spines[len(spines)-1]
	terminalRemote, err = d.fetchHeaderByHash(p, tesminalSpine)
	if err != nil {
		return false, terminalRemote, err
	}
	if terminalRemote.Hash() != tesminalSpine {
		return false, terminalRemote, errBadPeer
	}
	return true, terminalRemote, nil
}

// fetchHeaderByNr retrieves the header by finalized number from a remote peer.
func (d *Downloader) fetchHeaderByNr(p *peerConnection, nr uint64) (header *types.Header, err error) {
	p.log.Info("Retrieving remote chain header by nr")
	fetch := 1

	go p.peer.RequestHeadersByNumber(nr, 1, 0, true)

	ttl := d.peers.rates.TargetTimeout()
	timeout := time.After(ttl)
	for {
		select {
		case <-d.cancelCh:
			return nil, errCanceled

		case packet := <-d.headerCh:
			// Discard anything not from the origin peer
			if packet.PeerId() != p.id {
				log.Debug("Received headers from incorrect peer", "peer", packet.PeerId())
				break
			}
			// Make sure the peer gave us at least one and at most the requested headers
			headers := packet.(*headerPack).headers
			if len(headers) == 0 || len(headers) > fetch {
				log.Error(fmt.Sprintf("%s: returned headers %d != requested %d", errBadPeer, len(headers), fetch))
				return nil, errBadPeer
			}
			header = headers[0]
			return header, nil

		case <-timeout:
			p.log.Warn("Waiting for header header timed out", "elapsed", ttl)
			return nil, errTimeout

		case <-d.bodyCh:
		case <-d.receiptCh:
			// Out of bounds delivery, ignore
		}
	}
}

// fetchHeaderByHash retrieves the header by hash from a remote peer.
func (d *Downloader) fetchHeaderByHash(p *peerConnection, hash common.Hash) (header *types.Header, err error) {
	p.log.Info("Retrieving remote chain header by hash")
	fetch := 1

	go p.peer.RequestHeadersByHashes(common.HashArray{hash})

	ttl := d.peers.rates.TargetTimeout()
	timeout := time.After(ttl)
	for {
		select {
		case <-d.cancelCh:
			return nil, errCanceled

		case packet := <-d.headerCh:
			// Discard anything not from the origin peer
			if packet.PeerId() != p.id {
				log.Debug("Received headers from incorrect peer", "peer", packet.PeerId())
				break
			}
			// Make sure the peer gave us at least one and at most the requested headers
			headers := packet.(*headerPack).headers
			if len(headers) == 0 || len(headers) > fetch {
				log.Error(fmt.Sprintf("%s: returned headers %d != requested %d", errBadPeer, len(headers), fetch))
				return nil, errBadPeer
			}
			header = headers[0]
			return header, nil

		case <-timeout:
			p.log.Warn("Waiting for header header timed out", "elapsed", ttl)
			return nil, errTimeout

		case <-d.bodyCh:
		case <-d.receiptCh:
			// Out of bounds delivery, ignore
		}
	}
}

// peerSyncDagChain downloads and set on current node unfinalized chain from remote peer.
func (d *Downloader) peerSyncDagChain(p *peerConnection, baseSpine common.Hash, spines common.HashArray, finEpoch uint64) (fullySynced bool, err error) {
	//check remote peer
	isPeerAcceptable, _, err := d.checkPeer(p, baseSpine, spines)
	if err != nil {
		return false, err
	}
	if !isPeerAcceptable {
		return false, errCanceled
	}
	terminalSpine := spines[len(spines)-1]

	si := d.lightchain.GetSlotInfo()
	currEpoch := si.SlotToEpoch(si.CurrentSlot())
	//terminalEpoch := si.SlotToEpoch(terminalRemote.Slot)
	// is head epoch reached
	if currEpoch == finEpoch {
		// set param to load full dag
		terminalSpine = common.Hash{}
		fullySynced = true
		log.Info("peerSyncDagChain SYNC currEpoch == finEpoch", "currEpoch", currEpoch, "finEpoch", finEpoch)
		log.Info("Synchronisation of chain head detected", "baseSpine", baseSpine.Hex(), "spines", spines)

		return d.peerSyncDagChainHeadBySlots(p, baseSpine)
	}

	// fetch dag hashes
	remoteDag, err := d.fetchDagHashes(p, baseSpine, terminalSpine)
	if err != nil {
		return false, err
	}
	// check all spines received
	for _, s := range spines {
		if !remoteDag.Has(s) {
			return false, errCanceled
		}
	}
	// if remote tips set to last finalized block - do same
	if len(remoteDag) == 1 && remoteDag[0] == baseSpine {
		return fullySynced, d.blockchain.ResetTips()
	}
	// filter existed blocks
	dag := make(common.HashArray, 0, len(remoteDag))
	existedBlocks := make(types.Blocks, 0, len(remoteDag))
	dagBlocks := d.blockchain.GetBlocksByHashes(remoteDag)
	for h, b := range dagBlocks {
		if b == nil {
			dag = append(dag, h)
		} else {
			existedBlocks = append(existedBlocks, b)
		}
	}

	log.Info("Synchronization of dag chain: dag hashes retrieved", "dag", len(dag), "err", err)
	if err != nil {
		return false, err
	}
	headers, err := d.fetchDagHeaders(p, dag)
	log.Info("Synchronization of dag chain: dag headers retrieved", "count", len(headers), "headers", len(headers), "err", err)
	if err != nil {
		return false, err
	}
	txsMap, err := d.fetchDagTxs(p, dag)
	log.Info("Synchronization of dag chain: dag transactions retrieved", "count", len(txsMap), "txs", len(txsMap), "err", err)
	if err != nil {
		return false, err
	}

	// rm deprecated dag info
	//d.ClearBlockDag()
	d.blockchain.RemoveTips(d.blockchain.GetTips().GetHashes())
	blocks := make(types.Blocks, len(headers), len(dagBlocks))
	for i, header := range headers {
		txs := txsMap[header.Hash()]
		block := types.NewBlockWithHeader(header).WithBody(txs)
		blocks[i] = block
	}

	blocksBySlot, err := (&blocks).GroupBySlot()
	if err != nil {
		return false, err
	}
	//sort by slots
	slots := common.SorterAscU64{}
	for sl, _ := range blocksBySlot {
		slots = append(slots, sl)
	}
	sort.Sort(slots)

	for _, slot := range slots {
		// era.HandleEra(d.blockchain, slot)
		slotBlocks := blocksBySlot[slot]
		if len(slotBlocks) == 0 {
			continue
		}
		//handle by reverse order
		for _, block := range slotBlocks {
			// Commit block to database.
			_, err = d.blockchain.WriteSyncDagBlock(block, fullySynced)
			if err != nil {
				log.Error("Failed writing block to chain (sync)", "err", err)
				return false, err
			}
		}
	}
	return fullySynced, err
}

// peerSyncDagChain downloads and set on current node unfinalized chain from remote peer.
func (d *Downloader) peerSyncDagChainHeadBySlots(p *peerConnection, baseSpine common.Hash) (fullySynced bool, err error) {
	defer func(ts time.Time) {
		log.Info("^^^^^^^^^^^^ TIME GetHashesBySlot",
			"elapsed", common.PrettyDuration(time.Since(ts)),
			"func:", "peerSyncDagChainHeadBySlots",
		)
	}(time.Now())

	step := (eth.LimitDagHashes / d.lightchain.Config().ValidatorsPerSlot) / 2
	baseHeader := d.lightchain.GetHeaderByHash(baseSpine)
	si := d.lightchain.GetSlotInfo()

	d.blockchain.RemoveTips(d.blockchain.GetTips().GetHashes())
	for from := baseHeader.Slot; from < si.CurrentSlot(); {
		to := from + step
		if err = d.syncBySlots(p, from, to); err != nil {
			p.log.Error("Synchronization by slots: error", "err", err, "from", from, "to", to)
			return false, err
		}
		from = to
	}
	return true, nil
}

func (d *Downloader) syncBySlots(p *peerConnection, from, to uint64) error {
	var (
		remoteHashes common.HashArray
		err          error
	)

	defer func(ts time.Time) {
		log.Info("^^^^^^^^^^^^ TIME",
			"elapsed", common.PrettyDuration(time.Since(ts)),
			"slots", to-from,
			"len(hashes)", len(remoteHashes),
			"func:", "syncBySlots",
		)
	}(time.Now())

	p.log.Info("Synchronization by slots: start", "from", from, "to", to)
	// fetch dag hashes
	remoteHashes, err = d.fetchDagHashesBySlots(p, from, to)
	if err != nil {
		p.log.Error("Synchronization by slots: error 0", "err", err, "from", from, "to", to)
		return err
	}

	// filter existed blocks
	dag := make(common.HashArray, 0, len(remoteHashes))
	dagBlocks := d.blockchain.GetBlocksByHashes(remoteHashes)
	for h, b := range dagBlocks {
		if b == nil {
			dag = append(dag, h)
		}
	}

	if len(dag) == 0 {
		return nil
	}

	log.Info("Synchronization by slots: dag hashes retrieved", "dag", len(dag), "err", err)
	if err != nil {
		p.log.Error("Synchronization by slots: error 1", "err", err, "from", from, "to", to)
		return err
	}
	headers, err := d.fetchDagHeaders(p, dag)
	log.Info("Synchronization by slots: dag headers retrieved", "count", len(headers), "headers", len(headers), "err", err)
	if err != nil {
		p.log.Error("Synchronization by slots: error 2", "err", err, "from", from, "to", to)
		return err
	}
	txsMap, err := d.fetchDagTxs(p, dag)
	log.Info("Synchronization by slots: dag transactions retrieved", "count", len(txsMap), "txs", len(txsMap), "err", err)
	if err != nil {
		p.log.Error("Synchronization by slots: error 3", "err", err, "from", from, "to", to)
		return err
	}

	blocks := make(types.Blocks, len(headers), len(dagBlocks))
	for i, header := range headers {
		txs := txsMap[header.Hash()]
		block := types.NewBlockWithHeader(header).WithBody(txs)
		blocks[i] = block
	}

	blocksBySlot, err := (&blocks).GroupBySlot()
	if err != nil {
		return err
	}
	//sort by slots
	slots := common.SorterAscU64{}
	for sl, _ := range blocksBySlot {
		slots = append(slots, sl)
	}
	sort.Sort(slots)

	for _, slot := range slots {
		// era.HandleEra(d.blockchain, slot)
		slotBlocks := blocksBySlot[slot]
		if len(slotBlocks) == 0 {
			continue
		}
		//handle by reverse order
		for _, block := range slotBlocks {
			// Commit block to database.
			_, err = d.blockchain.WriteSyncDagBlock(block, true)
			if err != nil {
				p.log.Error("Synchronization by slots: error 4", "err", err, "from", from, "to", to)
				return err
			}
		}
	}
	return err
}
