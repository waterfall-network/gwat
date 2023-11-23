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
	"context"
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
	MaxPeerCon      = 3

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
	errSyncPeerNotFound        = errors.New("sync peer not found")
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

	// multi sync
	syncPeersMutex      *sync.Mutex
	syncPeerHashesMutex *sync.Mutex

	syncPeers      map[string]syncPeerChans
	syncPeerHashes map[string]common.HashArray

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
	InsertHeaderChain([]*types.Header) (int, error)

	// SetHead rewinds the local chain to a new head.
	SetHead(array common.Hash) error

	// WriteSyncBlocks writes the dag blocks and all associated state to the database for dag synchronization process
	WriteSyncBlocks(blocks types.Blocks, validate bool) (failed *types.Block, err error)

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

		syncPeers:           make(map[string]syncPeerChans),
		syncPeerHashes:      make(map[string]common.HashArray),
		syncPeerHashesMutex: new(sync.Mutex),
		syncPeersMutex:      new(sync.Mutex),
	}
	chain.SetSyncProvider(dl)
	go dl.stateFetcher()
	return dl
}

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
		log.Warn("Sync failed, dropping peer", "peer", id, "err", err)
		if d.dropPeer == nil {
			// The dropPeer method is nil when `--copydb` is used for a local copy.
			// Timeouts can occur if e.g. compaction hits at the wrong time, and can be ignored
			log.Warn("Downloader wants to drop peer, but peerdrop-function is not set", "peer", id)
		} else {
			d.dropPeer(id)
		}
		return err
	}
	log.Warn("Sync failed, retrying", "err", err)
	return err
}

func (d *Downloader) synchroniseDagOnly(id string) error {
	if d.Synchronising() {
		log.Warn("Sync of dag chain canceled (synchronise process busy)")
		return errBusy
	}

	// Make sure only one goroutine is ever allowed past this point at once
	if !atomic.CompareAndSwapInt32(&d.dagSyncing, 0, 1) {
		return errBusy
	}
	defer func(start time.Time) {
		atomic.StoreInt32(&d.dagSyncing, 0)
		log.Info("Sync of dag chain terminated", "elapsed", common.PrettyDuration(time.Since(start)))
	}(time.Now())

	// If we are already full syncing, but have a fast-sync bloom filter laying
	// around, make sure it doesn't use memory any more. This is a special case
	// when the user attempts to fast sync a new empty network.
	if d.stateBloom != nil {
		d.stateBloom.Close()
	}

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

	// Retrieve the origin peer and initiate the downloading process
	p := d.peers.Peer(id)
	if p == nil {
		return errUnknownPeer
	}
	//Synchronization of dag chain
	baseSpine := d.blockchain.GetLastCoordinatedCheckpoint().Spine

	if err := d.peerSyncBySpinesByChunk(p, baseSpine, common.HashArray{common.Hash{}}); err != nil {
		log.Error("Sync of dag chain failed", "err", err)
		return err
	}
	return nil
	//return d.syncWithPeerDagOnly(p)
}

// syncWithPeerDagOnly starts a block synchronization based on the hash chain from the
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

	remoteDag, err := d.fetchHashesBySpines(p, baseSpine, common.Hash{})
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
	return nil
}

func (d *Downloader) getMode() SyncMode {
	return SyncMode(atomic.LoadUint32(&d.mode))
}

// syncWithPeer starts a block synchronization based on the hash chain from the
// specified peer and head hash.
// Deprecated
func (d *Downloader) syncWithPeer(p *peerConnection, dag common.HashArray) (err error) {
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
		log.Debug("Sync terminated", "elapsed", common.PrettyDuration(time.Since(start)))
	}(time.Now())

	log.Info("Synchronising with the network", "peer", p.id, "eth", p.version, "mode", mode, "dag", dag)

	// if remote peer has unknown dag blocks only
	// sync such blocks only
	if err = d.syncWithPeerUnknownDagBlocks(p, dag); err != nil {
		log.Error("Sync of unknown dag blocks failed", "err", err)
		return err
	}
	return nil
}

// syncWithPeerUnknownDagBlocks if remote peer has unknown dag blocks only sync such blocks only
func (d *Downloader) syncWithPeerUnknownDagBlocks(p *peerConnection, dag common.HashArray) (err error) {
	// Make sure only one goroutine is ever allowed past this point at once
	if !atomic.CompareAndSwapInt32(&d.dagSyncing, 0, 1) {
		return errBusy
	}

	defer func(start time.Time) {
		atomic.StoreInt32(&d.dagSyncing, 0)
		log.Info("Sync of dag chain terminated", "elapsed", common.PrettyDuration(time.Since(start)))
	}(time.Now())

	// filter existed blocks
	remoteHashes := dag.Copy()
	remoteHashes.Deduplicate()
	delayed := d.blockchain.GetInsertDelayedHashes()
	remoteHashes = remoteHashes.Difference(delayed)
	dagBlocks := d.blockchain.GetBlocksByHashes(remoteHashes)
	dag = make(common.HashArray, 0, len(remoteHashes))
	for _, h := range remoteHashes {
		if dagBlocks[h] == nil && h != (common.Hash{}) {
			dag = append(dag, h)
		}
	}
	if len(dag) == 0 {
		return nil
	}

	log.Info("Sync of unknown dag blocks", "count", len(dag), "dag", dag)

	headers, err := d.fetchDagHeaders(p, dag)
	log.Info("Sync of unknown dag blocks: dag headers retrieved", "count", len(headers), "headers", headers, "err", err)
	if err != nil {
		return err
	}
	// request bodies for retrieved headers only
	dag = make(common.HashArray, 0, len(headers))
	for _, hdr := range headers {
		if hdr != nil {
			dag = append(dag, hdr.Hash())
		}
	}
	if len(dag) == 0 {
		return nil
	}
	txsMap, err := d.fetchDagTxs(p, dag)
	log.Info("Sync of unknown dag blocks: dag transactions retrieved", "count", len(txsMap), "err", err)
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
	for sl := range blocksBySlot {
		slots = append(slots, sl)
	}
	sort.Sort(slots)

	insBlocks := make(types.Blocks, 0, len(headers))
	for _, slot := range slots {
		slotBlocks := types.SpineSortBlocks(blocksBySlot[slot])
		insBlocks = append(insBlocks, slotBlocks...)
	}
	if bl, err := d.blockchain.WriteSyncBlocks(insBlocks, true); err != nil {
		log.Error("Failed writing block to chain  (sync unl)", "err", err, "bl.Slot", bl.Slot(), "hash", bl.Hash().Hex())
		return err
	}
	return nil
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

// fillHeaderSkeleton concurrently retrieves headers from all our available peers
// and maps them to the provided skeleton header chain.
//
// Any partial results from the beginning of the skeleton is (if possible) forwarded
// immediately to the header processor to keep the rest of the pipeline full even
// in the case of header stalls.
//
// The method returns the entire filled skeleton and also the number of headers
// already forwarded for processing.
// Deprecated
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

// fetchDagHashesBySlots retrieves the dag chain hashes by slot range.
func (d *Downloader) fetchDagHashesBySlots(p *peerConnection, from, to uint64) (dag common.HashArray, err error) {
	//slots limit 1024
	slotsLimit := eth.LimitDagHashes / d.lightchain.Config().ValidatorsPerSlot
	if to-from > slotsLimit {
		return nil, errDataSizeLimitExceeded
	}
	p.log.Info("Retrieving remote dag hashes by slot: start", "from", from, "to", to)

	//multi peers sync support
	if !d.isPeerSync(p.id) {
		d.setPeerSync(p.id)
		defer d.resetPeerSync(p.id)
	}
	peerCh := d.getPeerSyncChans(p.id).GetDagChan()

	err = p.peer.RequestHashesBySlots(from, to)
	if err != nil {
		return nil, err
	}

	ttl := d.peers.rates.TargetTimeout()
	timeout := time.After(ttl)

	for {
		if err != nil {
			log.Error("Sync: error while handling peer", "err", err, "peer", p.id, "fn", "fetchDagHashesBySlots")
		}
		err = nil
		select {
		case <-d.cancelCh:
			return nil, errCanceled
		case packet := <-d.dagCh:
			err = d.redirectPacketToSyncPeerChan(packet, dagCh)
		case packet := <-d.headerCh:
			err = d.redirectPacketToSyncPeerChan(packet, headerCh)
		case packet := <-d.bodyCh:
			err = d.redirectPacketToSyncPeerChan(packet, bodyCh)
		case packet := <-d.receiptCh:
			err = d.redirectPacketToSyncPeerChan(packet, receiptCh)
		case packet := <-peerCh:
			// handle redirected packet at peer chan
			log.Info("Sync: dag by slots: received packet", "packet.peer", packet.PeerId(), "peer", p.id)
			// Discard anything not from the origin peer
			if packet.PeerId() != p.id {
				log.Error("Sync: dag by slots: received from incorrect peer", "packet.peer", packet.PeerId(), "peer", p.id)
				break
			}
			dag = packet.(*dagPack).dag
			if len(dag) == 0 {
				log.Info("Sync: No block hashes found for provided slots", "from:", from, "to:", to)
			}
			return dag, err
		case <-timeout:
			p.log.Debug("Waiting for dag timed out", "elapsed", ttl)
			return nil, errTimeout
		}
	}
}

// fetchHashesBySpines retrieves the dag chain hashes beginning from finalized block (excluded from response).
func (d *Downloader) fetchHashesBySpines(p *peerConnection, baseSpine common.Hash, terminalSpine common.Hash) (dag common.HashArray, err error) {
	p.log.Info("Sync: retrieving remote dag hashes: start",
		"baseSpine", fmt.Sprintf("%#x", baseSpine),
		"terminalSpine", fmt.Sprintf("%#x", terminalSpine),
	)

	//multi peers sync support
	if !d.isPeerSync(p.id) {
		d.setPeerSync(p.id)
		defer d.resetPeerSync(p.id)
	}
	peerCh := d.getPeerSyncChans(p.id).GetDagChan()

	err = p.peer.RequestDag(baseSpine, terminalSpine)
	if err != nil {
		return nil, err
	}

	ttl := d.peers.rates.TargetTimeout()
	timeout := time.After(ttl)

	for {
		if err != nil {
			log.Error("Sync: error while handling peer", "err", err, "peer", p.id, "fn", "fetchHashesBySpines")
		}
		err = nil
		select {
		case <-d.cancelCh:
			return nil, errCanceled
		case packet := <-d.dagCh:
			err = d.redirectPacketToSyncPeerChan(packet, dagCh)
		case packet := <-d.headerCh:
			err = d.redirectPacketToSyncPeerChan(packet, headerCh)
		case packet := <-d.bodyCh:
			err = d.redirectPacketToSyncPeerChan(packet, bodyCh)
		case packet := <-d.receiptCh:
			err = d.redirectPacketToSyncPeerChan(packet, receiptCh)
		case packet := <-peerCh:
			// handle redirected packet at peer chan
			log.Info("Sync: dag by slots: received packet", "packet.peer", packet.PeerId(), "peer", p.id)
			// Discard anything not from the origin peer
			if packet.PeerId() != p.id {
				log.Error("Sync: dag by slots: received from incorrect peer", "packet.peer", packet.PeerId(), "peer", p.id)
				break
			}
			dag = packet.(*dagPack).dag
			if len(dag) == 0 {
				err = errInvalidDag
			}
			return dag, err
		case <-timeout:
			p.log.Warn("Sync: Waiting for dag timed out", "elapsed", ttl)
			return nil, errTimeout
		}
	}
}

// fetchDagHeaders retrieves the dag headers by hashes from a remote peer.
func (d *Downloader) fetchDagHeaders(p *peerConnection, hashes common.HashArray) (headers []*types.Header, err error) {
	p.log.Info("Retrieving remote dag headers: start", "hashes", len(hashes))

	//multi peers sync support
	if !d.isPeerSync(p.id) {
		d.setPeerSync(p.id)
		defer d.resetPeerSync(p.id)
	}
	peerCh := d.getPeerSyncChans(p.id).GetHeaderChan()

	err = p.peer.RequestHeadersByHashes(hashes)
	if err != nil {
		return nil, err
	}

	ttl := d.peers.rates.TargetTimeout()
	timeout := time.After(ttl)

	for {
		if err != nil {
			log.Error("Sync: error while handling peer", "err", err, "peer", p.id, "fn", "fetchDagHeaders")
		}
		err = nil
		select {
		case <-d.cancelCh:
			return nil, errCanceled
		case packet := <-d.dagCh:
			err = d.redirectPacketToSyncPeerChan(packet, dagCh)
		case packet := <-d.headerCh:
			err = d.redirectPacketToSyncPeerChan(packet, headerCh)
		case packet := <-d.bodyCh:
			err = d.redirectPacketToSyncPeerChan(packet, bodyCh)
		case packet := <-d.receiptCh:
			err = d.redirectPacketToSyncPeerChan(packet, receiptCh)
		case packet := <-peerCh:
			// handle redirected packet at peer chan
			log.Info("Sync: header by hash: received packet", "packet.peer", packet.PeerId(), "peer", p.id)
			// Discard anything not from the origin peer
			if packet.PeerId() != p.id {
				log.Error("Sync: header by hash: received from incorrect peer", "packet.peer", packet.PeerId(), "peer", p.id)
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
	txsMap := map[common.Hash][]*types.Transaction{}
	var undelivered = hashes.Copy()
	for {
		transactions, err := d.requestDagTxs(p, undelivered)
		if err != nil {
			return nil, err
		}
		for i, txs := range transactions {
			txsMap[undelivered[i]] = txs
		}
		if len(txsMap) < len(hashes) {
			undelivered = hashes[len(txsMap):]
			continue
		}
		break
	}
	return txsMap, nil
}

// requestDagTxs retrieves the dag transactions by blocks hashes from a remote peer.
func (d *Downloader) requestDagTxs(p *peerConnection, hashes common.HashArray) ([][]*types.Transaction, error) {
	p.log.Info("Request remote dag block txs: start", "hashes", len(hashes))

	//multi peers sync support
	if !d.isPeerSync(p.id) {
		d.setPeerSync(p.id)
		defer d.resetPeerSync(p.id)
	}
	peerCh := d.getPeerSyncChans(p.id).GetBodyChan()

	err := p.peer.RequestBodies(hashes)
	if err != nil {
		return nil, err
	}

	ttl := d.peers.rates.TargetTimeout()
	timeout := time.After(ttl)

	for {
		if err != nil {
			log.Error("Sync: error while handling peer", "err", err, "peer", p.id, "fn", "fetchDagTxs")
		}
		err = nil
		select {
		case <-d.cancelCh:
			return nil, errCanceled
		case packet := <-d.headerCh:
			err = d.redirectPacketToSyncPeerChan(packet, headerCh)
		case packet := <-d.dagCh:
			err = d.redirectPacketToSyncPeerChan(packet, dagCh)
		case packet := <-d.bodyCh:
			err = d.redirectPacketToSyncPeerChan(packet, bodyCh)
		case packet := <-d.receiptCh:
			err = d.redirectPacketToSyncPeerChan(packet, receiptCh)
		case packet := <-peerCh:
			// handle redirected packet at peer chan
			log.Info("Sync: body by hashes: received packet", "packet.peer", packet.PeerId(), "peer", p.id)
			// Discard anything not from the origin peer
			if packet.PeerId() != p.id {
				log.Error("Sync: body by hashes: received from incorrect peer", "packet.peer", packet.PeerId(), "peer", p.id)
				break
			}

			// Discard anything not from the origin peer
			if packet.PeerId() != p.id {
				log.Warn("Sync: body by hashes: received from incorrect peer", "packet.peer", packet.PeerId(), "peer", p.id)
				break
			}
			return packet.(*bodyPack).transactions, nil
		case <-timeout:
			p.log.Warn("Sync: body by hashes: timed out", "elapsed", ttl)
			return nil, errTimeout
		}
	}
}

func (d *Downloader) MainSync(baseSpine common.Hash, spines common.HashArray) error {
	log.Info("Sync chain by spines", "baseSpine", baseSpine.Hex(), "spines", len(spines), "len(d.peers)", len(d.peers.peers))
	if len(spines) == 0 {
		return nil
	}
	if d.peers.Len() == 0 {
		log.Error("Sync chain by spines", "baseSpine", baseSpine.Hex(), "spines", spines, "peers.Len", d.peers.Len(), "err", errNoPeers)
		return errNoPeers
	}

	// check it
	//d.blockchain.RemoveTips(d.blockchain.GetTips().GetHashes())
	//d.ClearBlockDag()

	//select peer
	for _, con := range d.peers.AllPeers() {
		err := d.peerSyncBySpines(con, baseSpine, spines)
		switch err {
		case nil:
			return nil
		case errInvalidBaseSpine:
			return errInvalidBaseSpine
		case errBusy, errCanceled:
			log.Warn("Sync failed, trying next peer",
				"err", err,
				"baseSpine", baseSpine.Hex(),
			)
			continue
		}
		if errors.Is(err, errInvalidChain) || errors.Is(err, errBadPeer) || errors.Is(err, errTimeout) ||
			errors.Is(err, errStallingPeer) || errors.Is(err, errUnsyncedPeer) || errors.Is(err, errEmptyHeaderSet) ||
			errors.Is(err, errPeersUnavailable) || errors.Is(err, errTooOld) || errors.Is(err, errInvalidAncestor) {
			log.Warn("Sync failed, dropping peer", "peer", con.id, "err", err)
			if d.dropPeer == nil {
				// The dropPeer method is nil when `--copydb` is used for a local copy.
				// Timeouts can occur if e.g. compaction hits at the wrong time, and can be ignored
				log.Warn("Downloader wants to drop peer, but peerdrop-function is not set", "peer", con.id)
			} else {
				d.dropPeer(con.id)
			}
		}
		log.Warn("Sync failed, trying next peer", "err", err)
	}
	return errNoPeers
}

// DagSync retrieves all blocks starting from baseSpine and add it to not finalized chain.
// param spines is optional and used for check remote peer and completeness data retrieved.
func (d *Downloader) DagSync(baseSpine common.Hash, spines common.HashArray) error {
	log.Info("Sync chain by spines", "baseSpine", baseSpine.Hex(), "spines", len(spines), "len(d.peers)", len(d.peers.peers))
	if d.peers.Len() == 0 {
		log.Error("Sync chain by spines", "baseSpine", baseSpine.Hex(), "spines", spines, "peers.Len", d.peers.Len(), "err", errNoPeers)
		return errNoPeers
	}

	// set param to load full dag
	log.Info("Sync of chain head detected", "baseSpine", baseSpine.Hex(), "spines", spines)
	return d.multiPeersHeadSync(baseSpine, spines)
}

func (d *Downloader) peerSyncBySpines(p *peerConnection, baseSpine common.Hash, spines common.HashArray) error {
	if d.Synchronising() {
		log.Warn("Sync canceled (synchronise process busy)")
		return errBusy
	}
	// Make sure only one goroutine is ever allowed past this point at once
	if !atomic.CompareAndSwapInt32(&d.finSyncing, 0, 1) {
		return errBusy
	}
	defer func(start time.Time) {
		atomic.StoreInt32(&d.finSyncing, 0)
		log.Info("Sync of dag chain terminated", "elapsed", common.PrettyDuration(time.Since(start)))
	}(time.Now())

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
	var err error
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
		log.Info("Sync terminated", "elapsed", common.PrettyDuration(time.Since(start)))
	}(time.Now())

	log.Info("Synchronising with the network", "peer", p.id, "eth", p.version, "mode", mode, "baseSpine", baseSpine.Hex(), "spines", spines)

	//Synchronization of dag chain
	if err = d.peerSyncBySpinesByChunk(p, baseSpine, spines); err != nil {
		log.Error("Sync of dag chain failed", "err", err)
		return err
	}
	return nil
}

func (d *Downloader) checkPeer(p *peerConnection, baseSpine common.Hash, spines common.HashArray) (isPeerAcceptable bool, terminalRemote *types.Header, err error) {
	// check remote header
	baseHeader := d.lightchain.GetHeaderByHash(baseSpine)
	if baseHeader == nil || baseHeader.Height > 0 && baseHeader.Nr() == 0 {
		if baseHeader == nil {
			log.Error("Check peer: invalid base spine: base header not found", "hash", baseSpine.Hex())
		} else {
			log.Error("Check peer: invalid base spine", "slot", baseHeader.Slot, "height", baseHeader.Height, "nr", baseHeader.Nr(), "height", baseHeader.Hash().Hex())
		}
		return false, nil, errInvalidBaseSpine
	}
	baseNr := baseHeader.Nr()
	baseRemote, err := d.fetchHeaderByNr(p, baseNr)
	if err == errBadPeer {
		return false, nil, errCanceled
	}
	if err != nil {
		return false, nil, err
	}
	if baseRemote.Hash() != baseHeader.Hash() || baseRemote.Root != baseHeader.Root {
		return false, nil, errBadPeer
	}
	if len(spines) == 0 {
		return true, terminalRemote, nil
	}
	terminalSpine := spines[len(spines)-1]
	if terminalSpine == (common.Hash{}) {
		return true, terminalRemote, nil
	}
	terminalRemote, err = d.fetchHeaderByHash(p, terminalSpine)
	if err != nil {
		return false, terminalRemote, err
	}
	if terminalRemote.Hash() != terminalSpine {
		return false, terminalRemote, errBadPeer
	}
	return true, terminalRemote, nil
}

// fetchHeaderByNr retrieves the header by finalized number from a remote peer.
func (d *Downloader) fetchHeaderByNr(p *peerConnection, nr uint64) (header *types.Header, err error) {
	p.log.Info("Sync: Retrieving remote chain header by nr")
	fetch := 1

	//multi peers sync support
	if !d.isPeerSync(p.id) {
		d.setPeerSync(p.id)
		defer d.resetPeerSync(p.id)
	}
	peerCh := d.getPeerSyncChans(p.id).GetHeaderChan()

	err = p.peer.RequestHeadersByNumber(nr, 1, 0, true)
	if err != nil {
		return nil, err
	}

	ttl := d.peers.rates.TargetTimeout()
	timeout := time.After(ttl)
	for {
		if err != nil {
			log.Error("Sync: error while handling peer", "err", err, "peer", p.id, "fn", "fetchHeaderByNr")
		}
		err = nil
		select {
		case <-d.cancelCh:
			return nil, errCanceled
		case packet := <-d.dagCh:
			err = d.redirectPacketToSyncPeerChan(packet, dagCh)
		case packet := <-d.headerCh:
			err = d.redirectPacketToSyncPeerChan(packet, headerCh)
		case packet := <-d.bodyCh:
			err = d.redirectPacketToSyncPeerChan(packet, bodyCh)
		case packet := <-d.receiptCh:
			err = d.redirectPacketToSyncPeerChan(packet, receiptCh)
		case packet := <-peerCh:
			// handle redirected packet at peer chan
			log.Info("Sync: header by nr: received packet", "packet.peer", packet.PeerId(), "peer", p.id)
			// Discard anything not from the origin peer
			if packet.PeerId() != p.id {
				log.Error("Sync: header by nr: received from incorrect peer", "packet.peer", packet.PeerId(), "peer", p.id)
				break
			}
			// Make sure the peer gave us at least one and at most the requested headers
			headers := packet.(*headerPack).headers
			if len(headers) == 0 || len(headers) > fetch {
				log.Error(fmt.Sprintf("Sync: header by nr: %s: returned headers %d != requested %d", errBadPeer, len(headers), fetch), "peer", p.id)
				return nil, errBadPeer
			}
			header = headers[0]
			return header, nil

		case <-timeout:
			p.log.Warn("Sync: header by nr: timed out", "elapsed", ttl)
			return nil, errTimeout
		}
	}
}

// fetchHeaderByHash retrieves the header by hash from a remote peer.
func (d *Downloader) fetchHeaderByHash(p *peerConnection, hash common.Hash) (header *types.Header, err error) {
	p.log.Info("Sync: Retrieving remote chain header by hash")
	fetch := 1

	//multi peers sync support
	if !d.isPeerSync(p.id) {
		d.setPeerSync(p.id)
		defer d.resetPeerSync(p.id)
	}
	peerCh := d.getPeerSyncChans(p.id).GetHeaderChan()

	err = p.peer.RequestHeadersByHashes(common.HashArray{hash})
	if err != nil {
		return nil, err
	}

	ttl := d.peers.rates.TargetTimeout()
	timeout := time.After(ttl)
	for {
		if err != nil {
			log.Error("Sync: error while handling peer", "err", err, "peer", p.id, "fn", "fetchHeaderByHash")
		}
		err = nil
		select {
		case <-d.cancelCh:
			return nil, errCanceled
		case packet := <-d.dagCh:
			err = d.redirectPacketToSyncPeerChan(packet, dagCh)
		case packet := <-d.headerCh:
			err = d.redirectPacketToSyncPeerChan(packet, headerCh)
		case packet := <-d.bodyCh:
			err = d.redirectPacketToSyncPeerChan(packet, bodyCh)
		case packet := <-d.receiptCh:
			err = d.redirectPacketToSyncPeerChan(packet, receiptCh)
		case packet := <-peerCh:
			// handle redirected packet at peer chan
			log.Info("Sync: header by hash: received packet", "packet.peer", packet.PeerId(), "peer", p.id)
			// Discard anything not from the origin peer
			if packet.PeerId() != p.id {
				log.Error("Sync: header by hash: received from incorrect peer", "packet.peer", packet.PeerId(), "peer", p.id)
				break
			}
			// Make sure the peer gave us at least one and at most the requested headers
			headers := packet.(*headerPack).headers
			if len(headers) == 0 || len(headers) > fetch {
				log.Error(fmt.Sprintf("Sync: header by hash: %s: returned headers %d != requested %d", errBadPeer, len(headers), fetch), "peer", p.id)
				return nil, errBadPeer
			}
			header = headers[0]
			return header, nil
		case <-timeout:
			p.log.Warn("Sync: header by hash: timed out", "elapsed", ttl)
			return nil, errTimeout
		}
	}
}

// peerSyncBySpinesByChunk downloads and set on current node unfinalized chain from remote peer.
func (d *Downloader) peerSyncBySpinesByChunk(p *peerConnection, baseSpine common.Hash, spines common.HashArray) error {
	defer func(ts time.Time) {
		log.Info("^^^^^^^^^^^^ TIME",
			"elapsed", common.PrettyDuration(time.Since(ts)),
			"func:", "peerSyncBySpinesByChunk",
		)
	}(time.Now())

	//check remote peer
	isPeerAcceptable, _, err := d.checkPeer(p, baseSpine, spines)
	if err != nil {
		return err
	}
	if !isPeerAcceptable {
		log.Error("Sync: peer is not acceptable", "peer", p.id)
		return errCanceled
	}

	terminalSpine := spines[len(spines)-1]
	baseHeader := d.lightchain.GetHeaderByHash(baseSpine)

	p.log.Info("Sync by spines: chunking",
		"base.Slot", baseHeader.Slot,
		"baseSpine", baseSpine.Hex(),
		"terminalSpine", terminalSpine.Hex(),
	)

	fromHash := baseSpine
	for i := 0; ; {
		lastHash, err := d.syncBySpines(p, fromHash, terminalSpine)
		if err != nil {
			p.log.Error("Sync by spines: error",
				"err", err,
				"fromHash", baseSpine.Hex(),
				"terminalSpine", terminalSpine.Hex(),
				"fromHash", fromHash.Hex(),
			)
			return err
		}
		log.Info("Sync by spines: iter",
			"i", i,
			"baseSpine", baseSpine.Hex(),
			"terminalSpine", terminalSpine.Hex(),
			"fromHash", fromHash.Hex(),
			"lastHash", lastHash.Hex(),
		)
		if terminalSpine == lastHash {
			log.Info("Sync by spines: success")
			break
		}
		if lastHash == (common.Hash{}) {
			return errCanceled
		}
		// sync next part
		fromHash = lastHash
	}
	// check all spines received
	for _, s := range spines {
		if s == (common.Hash{}) {
			continue
		}
		if !d.blockchain.HasHeader(s) {
			p.log.Error("Sync by spines: spine not found",
				"err", errCanceled,
				"lost", s.Hex(),
				"fromHash", baseSpine.Hex(),
				"terminalSpine", terminalSpine.Hex(),
			)
			return errCanceled
		}
	}
	return nil
}

func (d *Downloader) syncBySpines(p *peerConnection, baseSpine, terminalSpine common.Hash) (common.Hash, error) {
	var (
		remoteHashes common.HashArray
		err          error
		lastHash     common.Hash
	)

	defer func(ts time.Time) {
		log.Info("^^^^^^^^^^^^ TIME",
			"elapsed", common.PrettyDuration(time.Since(ts)),
			"baseSpine", baseSpine.Hex(),
			"terminalSpine", terminalSpine.Hex(),
			"len(hashes)", len(remoteHashes),
			"len(hashes)", len(remoteHashes),
			"func:", "syncBySpines",
		)
	}(time.Now())

	p.log.Info("Sync by spines: start", "baseSpine", baseSpine.Hex(), "terminalSpine", terminalSpine.Hex())
	// fetch dag hashes

	// fetch dag hashes
	remoteHashes, err = d.fetchHashesBySpines(p, baseSpine, terminalSpine)
	if err != nil {
		p.log.Error("Sync by spines: error 0", "err", err, "baseSpine", baseSpine.Hex(), "terminalSpine", terminalSpine.Hex())
		return lastHash, err
	}
	if len(remoteHashes) > 0 {
		lastHash = remoteHashes[len(remoteHashes)-1]
	}

	// filter existed blocks
	dag := make(common.HashArray, 0, len(remoteHashes))
	dagBlocks := d.blockchain.GetBlocksByHashes(remoteHashes)
	for h, b := range dagBlocks {
		if b == nil && h != (common.Hash{}) {
			dag = append(dag, h)
		}
	}

	if len(dag) == 0 {
		return lastHash, nil
	}

	log.Info("Sync by spines: dag hashes retrieved", "dag", len(dag))

	headers, err := d.fetchDagHeaders(p, dag)
	log.Info("Sync by spines: dag headers retrieved", "count", len(headers), "headers", len(headers), "err", err)
	if err != nil {
		p.log.Error("Sync by spines: error 1", "err", err, "from", "baseSpine", baseSpine.Hex(), "terminalSpine", terminalSpine.Hex())
		return lastHash, err
	}
	// request bodies for retrieved headers only
	dag = make(common.HashArray, 0, len(headers))
	for _, hdr := range headers {
		if hdr != nil {
			dag = append(dag, hdr.Hash())
		}
	}
	if len(dag) == 0 {
		return lastHash, nil
	}
	txsMap, err := d.fetchDagTxs(p, dag)
	log.Info("Sync by spines: dag transactions retrieved", "count", len(txsMap), "txs", len(txsMap), "err", err)
	if err != nil {
		p.log.Error("Sync by spines: error 2", "err", err, "baseSpine", baseSpine.Hex(), "terminalSpine", terminalSpine.Hex())
		return lastHash, err
	}

	blocks := make(types.Blocks, len(headers), len(dagBlocks))
	for i, header := range headers {
		txs := txsMap[header.Hash()]
		block := types.NewBlockWithHeader(header).WithBody(txs)
		blocks[i] = block
	}

	if bl, err := d.blockchain.WriteSyncBlocks(blocks, false); err != nil {
		log.Error("Sync by spines: writing blocks failed", "err", err, "bl.Slot", bl.Slot(), "hash", bl.Hash().Hex())
		return lastHash, err
	}
	return lastHash, err
}

// peerSyncDagChainHeadBySlots downloads and set on current node not finalized chain from remote peer.
func (d *Downloader) peerSyncDagChainHeadBySlots(p *peerConnection, baseSpine common.Hash) (fullySynced bool, err error) {
	defer func(ts time.Time) {
		log.Info("^^^^^^^^^^^^ TIME",
			"elapsed", common.PrettyDuration(time.Since(ts)),
			"func:", "peerSyncDagChainHeadBySlots",
		)
	}(time.Now())

	baseHeader := d.lightchain.GetHeaderByHash(baseSpine)
	si := d.lightchain.GetSlotInfo()

	d.blockchain.RemoveTips(d.blockchain.GetTips().GetHashes())

	p.log.Info("Sync by slots: chunking",
		"CurrSlot", si.CurrentSlot(),
		"base.Slot", baseHeader.Slot,
		"baseSpine", baseSpine.Hex(),
	)

	d.blockchain.RemoveTips(d.blockchain.GetTips().GetHashes())
	for from := baseHeader.Slot; from <= si.CurrentSlot(); {
		to := from + d.slotRangeLimit()
		if err = d.syncBySlots(p, from, to); err != nil {
			p.log.Error("Sync by slots: error", "err", err, "from", from, "to", to)
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

	p.log.Info("Sync by slots: start", "from", from, "to", to)
	// fetch dag hashes
	remoteHashes, err = d.fetchDagHashesBySlots(p, from, to)
	if err != nil {
		p.log.Error("Sync by slots: error 0", "err", err, "from", from, "to", to)
		return err
	}

	// filter existed blocks
	dag := make(common.HashArray, 0, len(remoteHashes))
	dagBlocks := d.blockchain.GetBlocksByHashes(remoteHashes)
	for h, b := range dagBlocks {
		if b == nil && h != (common.Hash{}) {
			dag = append(dag, h)
		}
	}

	if len(dag) == 0 {
		return nil
	}

	log.Info("Sync by slots: dag hashes retrieved", "dag", len(dag))

	headers, err := d.fetchDagHeaders(p, dag)
	log.Info("Sync by slots: dag headers retrieved", "count", len(headers), "headers", len(headers), "err", err)
	if err != nil {
		p.log.Error("Sync by slots: error 2", "err", err, "from", from, "to", to)
		return err
	}
	// request bodies for retrieved headers only
	dag = make(common.HashArray, 0, len(headers))
	for _, hdr := range headers {
		if hdr != nil {
			dag = append(dag, hdr.Hash())
		}
	}
	if len(dag) == 0 {
		return nil
	}
	txsMap, err := d.fetchDagTxs(p, dag)
	log.Info("Sync by slots: dag transactions retrieved", "count", len(txsMap), "txs", len(txsMap), "err", err)
	if err != nil {
		p.log.Error("Sync by slots: error 3", "err", err, "from", from, "to", to)
		return err
	}

	blocks := make(types.Blocks, len(headers), len(dagBlocks))
	for i, header := range headers {
		txs := txsMap[header.Hash()]
		block := types.NewBlockWithHeader(header).WithBody(txs)
		blocks[i] = block
	}

	if bl, err := d.blockchain.WriteSyncBlocks(blocks, true); err != nil {
		log.Error("Sync by slots: writing blocks failed", "err", err, "bl.Slot", bl.Slot(), "hash", bl.Hash().Hex())
		return err
	}
	return nil
}

func (d *Downloader) multiPeersHeadSync(baseSpine common.Hash, spines common.HashArray) error {
	var err error
	if d.Synchronising() {
		log.Warn("Sync head canceled (synchronise process busy)")
		return errBusy
	}
	// Make sure only one goroutine is ever allowed past this point at once
	if !atomic.CompareAndSwapInt32(&d.finSyncing, 0, 1) {
		return errBusy
	}
	defer func(start time.Time) {
		atomic.StoreInt32(&d.finSyncing, 0)
		log.Info("Sync head terminated", "elapsed", common.PrettyDuration(time.Since(start)))
	}(time.Now())

	// set param to load full dag
	log.Info("Sync head 000", "baseSpine", baseSpine.Hex(), "spines", len(spines), "len(d.peers)", len(d.peers.peers))
	if d.peers.Len() == 0 {
		log.Error("Sync head: failed", "err", errNoPeers, "baseSpine", baseSpine.Hex(), "spines", spines, "peers.Len", d.peers.Len())
		return errNoPeers
	}

	defer func(start time.Time) {
		log.Info("Sync head: finish",
			"elapsed", common.PrettyDuration(time.Since(start)),
		)
	}(time.Now())

	// Atomically set the requested sync mode
	atomic.StoreUint32(&d.mode, uint32(FullSync))

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
	//d.cancelPeer = p.id
	d.cancelLock.Unlock()
	defer d.Cancel() // No matter what, we can't leave the cancel channel open

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

	d.multiPeerReset()
	defer d.multiPeerReset()

	var wg sync.WaitGroup // Step 2: Create a WaitGroup instance
	doneChan := make(chan struct{}, 1)
	sem := make(chan struct{}, MaxPeerCon)
	for i := 0; i < MaxPeerCon; i++ {
		sem <- struct{}{}
	}
	ctx, cancel := context.WithCancel(context.Background())
	var count int
Loop:
	for i, con := range d.peers.AllPeers() {
		select {
		case <-ctx.Done():
			close(sem)
			break Loop
		case <-sem:

			if con.version < eth.ETH66 {
				con.log.Error("Sync head: bad peer",
					"err", errTooOld,
					"version", con.version,
					"required", eth.ETH66,
				)
				d.dropPeer(con.id)
				continue
			}

			log.Info("Sync head: run peer", "i", i, "peer", con.id)
			d.setPeerSync(con.id)

			wg.Add(1)
			go func(con *peerConnection) {
				ii := i
				defer d.resetPeerSync(con.id)
				defer wg.Done()
				if err = d.multiPeerGetHashes(con, baseSpine, spines); err != nil {
					if errors.Is(err, errTimeout) {
						log.Warn("Sync head failed: timed out", "i", ii, "peer", con.id, "err", err)
					}
					if errors.Is(err, errInvalidChain) || errors.Is(err, errBadPeer) || //errors.Is(err, errTimeout) ||
						errors.Is(err, errStallingPeer) || errors.Is(err, errUnsyncedPeer) || errors.Is(err, errEmptyHeaderSet) ||
						errors.Is(err, errPeersUnavailable) || errors.Is(err, errTooOld) || errors.Is(err, errInvalidAncestor) {
						log.Warn("Sync head failed, dropping peer", "i", ii, "peer", con.id, "err", err)
						if d.dropPeer == nil {
							// The dropPeer method is nil when `--copydb` is used for a local copy.
							// Timeouts can occur if e.g. compaction hits at the wrong time, and can be ignored
							log.Warn("Sync head: downloader wants to drop peer, but peerdrop-function is not set", "peer", con.id)
						} else {
							log.Warn("Sync head failed, dropping peer", "i", ii, "peer", con.id)
							d.dropPeer(con.id)
						}
					}
					log.Info("Sync head: finished peer", "i", ii, "peer", con.id)
					sem <- struct{}{}
					return
				} else {
					count++
					if count == MaxPeerCon {
						cancel()
					}
				}
			}(con)
		}
	}
	wg.Wait()
	doneChan <- struct{}{}
	log.Info("Sync head: hashes retrieved", "dag", d.syncPeerHashes)

	// single peer sync
	for {
		select {
		case <-doneChan:
			for pid, dag := range d.syncPeerHashes {
				con := d.peers.Peer(pid)
				err = d.syncWithPeerUnknownDagBlocks(con, dag)
				if err != nil {
					log.Error("Sync head: block fetching failed", "err", err, "peer", pid, "blocks", dag)
				}
			}
			log.Info("Sync head:", "dag", d.syncPeerHashes)
			cancel()
			return nil
		}
	}
}

// multiPeerSyncDagChainHeadBySlots downloads and set on current node unfinalized chain from remote peer.
func (d *Downloader) multiPeerGetHashes(p *peerConnection, baseSpine common.Hash, spines common.HashArray) error {
	var err error
	//check remote peer
	isPeerAcceptable, _, err := d.checkPeer(p, baseSpine, spines)
	if err != nil {
		log.Error("Sync head peer: check err", "err", err, "peer", p.id)
		return err
	}
	if !isPeerAcceptable {
		log.Error("Sync head peer: unacceptable", "peer", p.id)
		return errCanceled
	}

	baseHeader := d.lightchain.GetHeaderByHash(baseSpine)
	si := d.lightchain.GetSlotInfo()

	p.log.Info("Sync head peer: chunking",
		"CurrSlot", si.CurrentSlot(),
		"base.Slot", baseHeader.Slot,
		"baseSpine", baseSpine.Hex(),
	)

	for from := baseHeader.Slot; from <= si.CurrentSlot(); {
		to := from + d.slotRangeLimit()
		log.Info("Sync head peer: hashes by slots",
			"from", from,
			"to", to,
			"baseHeader.Slot", baseHeader.Slot,
			"si.CurrentSlot()", si.CurrentSlot(),
		)

		if err = d.multiSyncBySlots(p, from, to); err != nil {
			p.log.Error("Sync head peer: hashes by slots: error",
				"err", err,
				"from", from,
				"to", to,
				"currSlot", si.CurrentSlot(),
				"baseHeader.Slot", baseHeader.Slot,
			)
			return err
		}
		from = to
	}
	p.log.Info("Sync head peer: finish",
		"currSlot", si.CurrentSlot(),
		"baseHeader.Slot", baseHeader.Slot,
	)
	return nil
}

func (d *Downloader) multiSyncBySlots(p *peerConnection, from, to uint64) error {
	var (
		remoteHashes common.HashArray
		err          error
	)
	// fetch dag hashes
	remoteHashes, err = d.multiFetchDagHashesBySlots(p, from, to)
	if err != nil {
		p.log.Error("Sync head peer: error 0", "err", err, "fromSlot", from, "toSlot", to)
		return err
	}

	p.log.Info("Sync head peer: hashes received",
		"fromSlot", from,
		"toSlot", to,
		"len", len(remoteHashes),
		"remoteHashes", remoteHashes,
	)

	if len(remoteHashes) == 0 {
		return nil
	}

	d.syncPeerHashesMutex.Lock()
	if _, ok := d.syncPeerHashes[p.id]; !ok {
		d.syncPeerHashes[p.id] = remoteHashes
	} else {
		d.syncPeerHashes[p.id] = append(d.syncPeerHashes[p.id], remoteHashes...)
	}
	d.syncPeerHashesMutex.Unlock()

	return err
}

// fetchDagHashes retrieves the dag chain hashes beginning from finalized block (excluded from response).
func (d *Downloader) multiFetchDagHashesBySlots(p *peerConnection, from, to uint64) (dag common.HashArray, err error) {
	//slots limit 1024
	slotsLimit := eth.LimitDagHashes / d.lightchain.Config().ValidatorsPerSlot
	if to-from > slotsLimit {
		return nil, errDataSizeLimitExceeded
	}
	p.log.Info("Retrieving remote dag hashes by slot: start", "from", from, "to", to)

	//multi peers sync support
	if !d.isPeerSync(p.id) {
		d.setPeerSync(p.id)
		defer d.resetPeerSync(p.id)
	}
	peerCh := d.getPeerSyncChans(p.id).GetDagChan()

	err = p.peer.RequestHashesBySlots(from, to)
	if err != nil {
		return nil, err
	}

	ttl := d.peers.rates.TargetTimeout()
	timeout := time.After(ttl)

	for {
		if err != nil {
			log.Error("Sync: error while handling peer", "err", err, "peer", p.id, "fn", "multiFetchDagHashesBySlots")
		}
		err = nil
		select {
		case <-d.cancelCh:
			return nil, errCanceled
		case packet := <-d.dagCh:
			err = d.redirectPacketToSyncPeerChan(packet, dagCh)
		case packet := <-d.headerCh:
			err = d.redirectPacketToSyncPeerChan(packet, headerCh)
		case packet := <-d.bodyCh:
			err = d.redirectPacketToSyncPeerChan(packet, bodyCh)
		case packet := <-d.receiptCh:
			err = d.redirectPacketToSyncPeerChan(packet, receiptCh)
		case packet := <-peerCh:
			// handle redirected packet at peer chan
			log.Info("Sync: dag by slots: received packet", "packet.peer", packet.PeerId(), "peer", p.id)
			// Discard anything not from the origin peer
			if packet.PeerId() != p.id {
				log.Error("Sync: dag by slots: received from incorrect peer", "packet.peer", packet.PeerId(), "peer", p.id)
				break
			}
			dag = packet.(*dagPack).dag
			if len(dag) == 0 {
				log.Info("No block hashes found for provided slots", "from:", from, "to:", to)
			}
			return dag, err
		case <-timeout:
			p.log.Debug("Waiting for dag timed out", "elapsed", ttl)
			return nil, errTimeout
		}
	}
}

// redirectPacketToSyncPeerChan redirects data from common chans to individual peer's chans.
func (d *Downloader) redirectPacketToSyncPeerChan(packet dataPack, chType syncPeerChanType) error {
	log.Info("Sync: redirect packet 000", "packet.peer", packet.PeerId(), "chType", chType)
	if !d.isPeerSync(packet.PeerId()) {
		log.Error("Sync: dag by slots: received from incorrect peer", "peer", packet.PeerId(), "chType", chType)
		return errSyncPeerNotFound
	}
	pChs := d.getPeerSyncChans(packet.PeerId())
	if pChs == nil {
		log.Error("Sync: dag by slots: sync peer not found", "peer", packet.PeerId(), "chType", chType)
		return errSyncPeerNotFound
	}
	switch chType {
	case dagCh:
		if len(pChs.GetDagChan()) > 0 {
			<-pChs.GetDagChan()
		}
		pChs.GetDagChan() <- packet
	case headerCh:
		if len(pChs.GetHeaderChan()) > 0 {
			<-pChs.GetHeaderChan()
		}
		pChs.GetHeaderChan() <- packet
	case bodyCh:
		if len(pChs.GetBodyChan()) > 0 {
			<-pChs.GetBodyChan()
		}
		pChs.GetBodyChan() <- packet
	case receiptCh:
		if len(pChs.GetReceiptChan()) > 0 {
			<-pChs.GetReceiptChan()
		}
		pChs.GetReceiptChan() <- packet
	default:
		log.Error("Sync: dag by slots: bad chan type", "peer", packet.PeerId(), "chType", chType)
		return fmt.Errorf("bad chan type=%d", chType)
	}
	log.Info("Sync: redirect packet 111", "packet.peer", packet.PeerId(), "chType", chType)
	return nil
}

// multiPeerReset empties the syncPeerHashes map.
func (d *Downloader) multiPeerReset() {
	d.syncPeerHashesMutex.Lock()
	defer d.syncPeerHashesMutex.Unlock()
	d.syncPeerHashes = make(map[string]common.HashArray)
}

// slotStep calculates max slots numbers for sync request ramge.
func (d *Downloader) slotRangeLimit() uint64 {
	return eth.LimitDagHashes / d.lightchain.Config().ValidatorsPerSlot
}

func (d *Downloader) setPeerSync(peerId string) {
	d.syncPeersMutex.Lock()
	defer d.syncPeersMutex.Unlock()

	if _, ok := d.syncPeers[peerId]; !ok {
		d.syncPeers[peerId] = initSyncPeerChans()
	}
}

func (d *Downloader) resetPeerSync(peerId string) {
	d.syncPeersMutex.Lock()
	defer d.syncPeersMutex.Unlock()

	if ch, ok := d.syncPeers[peerId]; ok {
		ch.Close()
		delete(d.syncPeers, peerId)
	}
}

func (d *Downloader) isPeerSync(peerId string) bool {
	d.syncPeersMutex.Lock()
	defer d.syncPeersMutex.Unlock()

	_, ok := d.syncPeers[peerId]
	return ok
}

func (d *Downloader) getPeerSyncChans(peerId string) *syncPeerChans {
	d.syncPeersMutex.Lock()
	defer d.syncPeersMutex.Unlock()

	if ch, ok := d.syncPeers[peerId]; ok {
		return &ch
	}
	return nil
}
