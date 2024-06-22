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
	MaxBlockFetch     = 128 // Amount of blocks to be fetched per retrieval request
	MaxHeaderFetch    = 192 // Amount of block headers to be fetched per retrieval request
	MaxSkeletonSize   = 128 // Number of header fetches to need for a skeleton assembly
	MaxReceiptFetch   = 256 // Amount of transaction receipts to allow fetching per request
	MaxStateFetch     = 384 // Amount of node state values to allow fetching per request
	MaxPeerCon        = 3
	maxResultsProcess = 2048 // Number of content download results to import at once into the chain
)

var (
	errBusy                  = errors.New("busy")
	errUnknownPeer           = errors.New("peer is unknown or unhealthy")
	errBadPeer               = errors.New("action from bad peer ignored")
	errStallingPeer          = errors.New("peer is stalling")
	errNoPeers               = errors.New("no peers to keep download active")
	errTimeout               = errors.New("timeout")
	errEmptyHeaderSet        = errors.New("empty header set by peer")
	errPeersUnavailable      = errors.New("no peers available or all tried for download")
	errInvalidAncestor       = errors.New("retrieved ancestor is invalid")
	errInvalidChain          = errors.New("retrieved hash chain is invalid")
	errInvalidBody           = errors.New("retrieved block body is invalid")
	errInvalidReceipt        = errors.New("retrieved receipt is invalid")
	errInvalidDag            = errors.New("retrieved dag chain is invalid")
	ErrInvalidBaseSpine      = errors.New("invalid base spine")
	errCancelStateFetch      = errors.New("state data download canceled (requested)")
	errCanceled              = errors.New("syncing canceled (requested)")
	errNoSyncActive          = errors.New("no sync active")
	errTooOld                = errors.New("peer's protocol version too old")
	errDataSizeLimitExceeded = errors.New("data size limit exceeded")
	errSyncPeerNotFound      = errors.New("sync peer not found")
)

type Downloader struct {
	mode uint32         // Synchronisation mode defining the strategy used (per sync cycle), use d.getMode() to get the SyncMode
	mux  *event.TypeMux // Event multiplexer to announce sync operation events

	syncByHashesLock sync.RWMutex // Lock protecting the sync stats fields

	checkpoint uint64   // Checkpoint block number to enforce head against (e.g. fast sync)
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
	finSyncing int32
	dagSyncing int32

	// Channels
	dagCh         chan dataPack        // Channel receiving inbound dag hashes
	headerCh      chan dataPack        // Channel receiving inbound block headers
	bodyCh        chan dataPack        // Channel receiving inbound block bodies
	receiptCh     chan dataPack        // Channel receiving inbound receipts
	bodyWakeCh    chan bool            // Channel to signal the block body fetcher of new tasks
	receiptWakeCh chan bool            // Channel to signal the receipt fetcher of new tasks
	headerProcCh  chan []*types.Header // Channel to feed the header processor new tasks

	snapSync       bool         // Whether to run state sync over the snap protocol
	SnapSyncer     *snap.Syncer // TODO(karalabe): make private! hack for now
	stateSyncStart chan *stateSync
	trackStateReq  chan *stateReq
	stateCh        chan dataPack // Channel receiving inbound node state data

	// multi sync
	syncPeersMutex      *sync.Mutex
	syncPeerHashesMutex *sync.Mutex

	syncPeers      map[string]*syncPeerChans
	syncPeerHashes map[string]common.HashArray

	// Cancellation and termination
	cancelPeer string         // Identifier of the peer currently being used as the master (cancel on drop)
	cancelCh   chan struct{}  // Channel to cancel mid-flight syncs
	cancelLock sync.RWMutex   // Lock to protect the cancel channel and peer in delivers
	cancelWg   sync.WaitGroup // Make sure all fetcher goroutines have exited.

	quitCh   chan struct{} // Quit channel to signal termination
	quitLock sync.Mutex    // Lock to prevent double closes
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

		syncPeers:           make(map[string]*syncPeerChans),
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

// SyncUnloadedParents download blocks ffrom remote peer by hashes.
func (d *Downloader) SyncUnloadedParents(id string, hashes common.HashArray) error {
	var err error

	defer func(start time.Time) {
		atomic.StoreInt32(&d.dagSyncing, 0)
		log.Info("Sync unknown parents: end", "err", err, "elapsed", time.Since(start), "peer", id, "parents", hashes)
	}(time.Now())

	d.syncByHashesLock.Lock()
	defer d.syncByHashesLock.Unlock()

	unloaded := make(common.HashArray, 0, len(hashes))
	for _, h := range hashes {
		if d.blockchain.GetHeaderByHash(h) == nil {
			unloaded = append(unloaded, h)
		}
	}
	if len(unloaded) == 0 {
		log.Info("Sync unknown parents: already inserted", "peer", id, "parents", hashes)
		return nil
	}

	//multi peers sync support
	if !d.isPeerSync(id) {
		d.setPeerSync(id)
		defer d.resetPeerSync(id)
	} else {
		return errBusy
	}

	// If we are already full syncing, but have a fast-sync bloom filter laying
	// around, make sure it doesn't use memory anymore. This is a special case
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
	err = d.syncWithPeerUnknownBlocksWithParents(p, unloaded)
	if err != nil {
		log.Error("Sync unknown parents: failed", "err", err, "peer", id, "parents", unloaded)
		return err
	}
	log.Info("Sync unknown parents: success", "peer", id, "parents", unloaded)
	return nil
}

func (d *Downloader) SynchroniseDagOnly(id string) error {
	err := d.synchroniseDagOnly(id)
	switch err {
	case nil, errBusy, errCanceled:
		return err
	}
	if errors.Is(err, errInvalidChain) || errors.Is(err, errBadPeer) || errors.Is(err, errTimeout) ||
		errors.Is(err, errStallingPeer) || errors.Is(err, errEmptyHeaderSet) ||
		errors.Is(err, errPeersUnavailable) || errors.Is(err, errTooOld) || errors.Is(err, errInvalidAncestor) ||
		errors.Is(err, errInvalidBody) {
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

	if err := d.peerSyncBySpinesByChunk(p, baseSpine, common.HashArray{common.Hash{}}, false); err != nil {
		log.Error("Sync of dag chain failed", "err", err)
		return err
	}
	return nil
}

// syncWithPeerDagOnly starts a block synchronization based on the hash chain from the
// specified peer and head hash.
//
// nolint:unused // todo: check
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

// syncWithPeerUnknownDagBlocks if remote peer has unknown dag blocks only sync such blocks only
// nolint:unused
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

	rawHdrs, err := d.fetchDagHeaders(p, dag)
	log.Info("Sync of unknown dag blocks: dag headers retrieved", "count", len(rawHdrs), "headers", rawHdrs, "err", err)
	if err != nil {
		return err
	}

	headers, _, _ := d.filterConsistentParentsChains(rawHdrs)
	if len(headers) == 0 {
		return nil
	}

	dag = *headers.GetHashes()

	txsMap, err := d.fetchDagTxs(p, dag)
	log.Info("Sync of unknown dag blocks: dag transactions retrieved", "count", len(txsMap), "err", err)
	if err != nil {
		return err
	}

	blocks := make(types.Blocks, 0, len(headers))
	for _, header := range headers {
		txs := txsMap[header.Hash()]
		block := types.NewBlockWithHeader(header).WithBody(txs)
		// quick body validation
		if block.BodyHash() != block.Body().CalculateHash() {
			return errInvalidBody
		}
		blocks = append(blocks, block)
	}
	log.Info("Sync of unknown dag blocks: NewBlockWithHeader 111", "blocks", len(blocks), "err", err)

	blocksBySlot, err := (&blocks).GroupBySlot()
	log.Info("Sync of unknown dag blocks: GroupBySlot 222", "blocks", len(blocks), "err", err)
	if err != nil {
		return err
	}
	//sort by slots
	slots := common.SorterAscU64{}
	for sl := range blocksBySlot {
		slots = append(slots, sl)
	}
	sort.Sort(slots)
	log.Info("Sync of unknown dag blocks: sort slots 333", "blocks", len(blocks), "err", err, "slots", slots)

	insBlocks := make(types.Blocks, 0, len(headers))
	for _, slot := range slots {
		slotBlocks := types.SpineSortBlocks(blocksBySlot[slot])
		insBlocks = append(insBlocks, slotBlocks...)
	}
	log.Info("Sync of unknown dag blocks: SpineSortBlocks 444", "blocks", len(blocks), "err", err)

	//try to insert all blocks
	for {
		var bl *types.Block
		if bl, err = d.blockchain.WriteSyncBlocks(insBlocks, true); err != nil {
			log.Error("Sync of unknown dag blocks: Failed writing blocks to chain  (sync unl)", "err", err, "bl.Slot", bl.Slot(), "hash", bl.Hash().Hex())
			//if insertion failed - trying to insert after failed block
			if bl != nil {
				failedIx := 0
				for i, b := range insBlocks {
					failedIx = i
					if bl.Hash() == b.Hash() {
						break
					}
				}
				if failedIx+1 < len(insBlocks) {
					insBlocks = insBlocks[failedIx+1:]
					continue
				}
				return nil
			}
		}
		return nil
	}
}

// syncWithPeerUnknownBlocksWithParents fetching unloaded blocks by hashes from remote peer.
func (d *Downloader) syncWithPeerUnknownBlocksWithParents(p *peerConnection, hashes common.HashArray) (err error) {
	// Make sure only one goroutine is ever allowed past this point at once
	if !atomic.CompareAndSwapInt32(&d.dagSyncing, 0, 1) {
		return errBusy
	}

	iterCount := 0
	defer func(start time.Time) {
		atomic.StoreInt32(&d.dagSyncing, 0)
		log.Info("Sync unknown blocks: end",
			"elapsed", time.Since(start),
			"iterCount", iterCount,
		)
	}(time.Now())

	hashes.Deduplicate()
	// filter existed blocks
	delayed := d.blockchain.GetInsertDelayedHashes()
	diffHashes := hashes.Difference(delayed)
	dagBlocks := d.blockchain.GetBlocksByHashes(diffHashes)
	reqHashes := make(common.HashArray, 0, len(diffHashes))
	for _, h := range diffHashes {
		if dagBlocks[h] == nil && h != (common.Hash{}) {
			reqHashes = append(reqHashes, h)
		}
	}

	log.Info("Sync unknown blocks", "reqHashes", len(reqHashes), "reqHashes", reqHashes)

	if len(reqHashes) == 0 {
		return nil
	}

	loadedHashesMap := make(map[common.Hash]bool, len(delayed))
	for _, h := range delayed {
		loadedHashesMap[h] = true
	}
	blocks := make(types.Blocks, 0, len(reqHashes))
	for {
		rawHdrs, err := d.fetchDagHeaders(p, reqHashes)
		if err != nil {
			log.Error("Sync unknown blocks: fetch headers failed",
				"i", iterCount,
				"err", err,
				"reqHashes", reqHashes,
			)
			return err
		}
		hdrHashes := make(common.HashArray, 0, len(rawHdrs))
		headers := make(types.Headers, 0, len(rawHdrs))
		for _, hdr := range rawHdrs {
			if hdr != nil {
				headers = append(headers, hdr)
				hdrHashes = append(hdrHashes, hdr.Hash())
			}
		}
		if len(headers) == 0 {
			log.Warn("Sync unknown blocks: no headers fetched",
				"i", iterCount,
				"headers", len(headers),
				"raw", len(rawHdrs),
				"reqHashes", reqHashes,
			)
			break
		}

		log.Info("Sync unknown blocks: fetch headers",
			"i", iterCount,
			"headers", len(headers),
			"raw", len(rawHdrs),
			"reqHashes", reqHashes,
		)

		// request bodies for retrieved headers only
		txsMap, err := d.fetchDagTxs(p, hdrHashes)
		if err != nil {
			log.Error("Sync unknown blocks: fetch bodies failed",
				"i", iterCount,
				"err", err,
			)
			return err
		}
		log.Info("Sync unknown blocks: fetch bodies", "count", len(txsMap), "reqHashes", reqHashes)

		parents := make(map[common.Hash]bool)
		for _, header := range headers {
			hash := header.Hash()
			txs := txsMap[hash]
			block := types.NewBlockWithHeader(header).WithBody(txs)
			// quick body validation
			if block.BodyHash() != block.Body().CalculateHash() {
				log.Error("Sync unknown blocks: bad block bodyHash",
					"i", iterCount,
					"bodyHash", block.BodyHash().Hex(),
					"calcBodyHash", block.Body().CalculateHash().Hex(),
					"err", errInvalidBody,
				)
				return errInvalidBody
			}
			blocks = append(blocks, block)
			loadedHashesMap[hash] = true
			for _, ph := range block.ParentHashes() {
				parents[ph] = true
			}
		}
		log.Info("Sync unknown blocks: blocks assembled", "i", iterCount, "blocks", len(blocks), "reqHashes", reqHashes)

		//collect unloaded parents
		reqHashes = make(common.HashArray, 0, len(parents))
		var wg sync.WaitGroup
		var loadedMu sync.RWMutex
		for ph := range parents {
			if loadedHashesMap[ph] {
				continue
			}
			wg.Add(1)
			go func(pHash common.Hash) {
				defer wg.Done()
				parent := d.blockchain.GetHeaderByHash(pHash)
				if parent == nil {
					reqHashes = append(reqHashes, pHash)
				} else {
					loadedMu.Lock()
					loadedHashesMap[pHash] = true
					loadedMu.Unlock()
				}
			}(ph)
		}
		wg.Wait()

		log.Info("Sync unknown blocks: unloaded parents", "i", iterCount, "unloaded", reqHashes)

		if len(reqHashes) == 0 {
			break
		}
		iterCount++
	}
	log.Info("Sync unknown blocks: all blocks fetched", "blocks", len(blocks))

	blocksBySlot, err := (&blocks).GroupBySlot()
	if err != nil {
		log.Error("Sync unknown blocks: group by slot failed",
			"i", iterCount,
			"err", err,
		)
		return err
	}
	//sort by slots
	slots := common.SorterAscU64{}
	for sl := range blocksBySlot {
		slots = append(slots, sl)
	}
	sort.Sort(slots)
	log.Info("Sync unknown blocks: sort slots", "blocks", len(blocks), "slots", slots)

	insBlocks := make(types.Blocks, 0, len(blocks))
	for _, slot := range slots {
		slotBlocks := types.SpineSortBlocks(blocksBySlot[slot])
		insBlocks = append(insBlocks, slotBlocks...)
	}
	log.Info("Sync unknown blocks: sort blocks", "blocks", len(blocks), "slots", slots)

	//try to insert all blocks
	for {
		var bl *types.Block
		if bl, err = d.blockchain.WriteSyncBlocks(insBlocks, true); err != nil {
			log.Error("Sync unknown blocks: Failed writing blocks to chain  (sync unl)", "err", err, "bl.Slot", bl.Slot(), "hash", bl.Hash().Hex())
			//if insertion failed - trying to insert after failed block
			if bl != nil {
				failedIx := 0
				for i, b := range insBlocks {
					failedIx = i
					if bl.Hash() == b.Hash() {
						break
					}
				}
				if failedIx+1 < len(insBlocks) {
					insBlocks = insBlocks[failedIx+1:]
					continue
				}
				return nil
			}
		}
		return nil
	}
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
//
// nolint:unused // todo: check
func (d *Downloader) fetchDagHashesBySlots(p *peerConnection, from, to uint64) (dag common.HashArray, err error) {
	//slots limit 1024
	slotsLimit := eth.LimitDagHashes / d.lightchain.Config().ValidatorsPerSlot
	if to-from > slotsLimit {
		return nil, errDataSizeLimitExceeded
	}
	p.log.Info("Retrieving remote dag hashes by slot: start", "from", from, "to", to)

	//multi peers sync support
	d.setPeerSync(p.id)
	defer d.resetPeerSync(p.id)

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
	d.setPeerSync(p.id)
	defer d.resetPeerSync(p.id)

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
	d.setPeerSync(p.id)
	defer d.resetPeerSync(p.id)

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
	d.setPeerSync(p.id)
	defer d.resetPeerSync(p.id)

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

func (d *Downloader) OptimisticSpineSync(spines common.HashArray) error {
	log.Info("Sync opt spines", "spines", len(spines), "peers", d.peers.Len(), "spines", spines)
	if len(spines) == 0 {
		return nil
	}
	if d.peers.Len() == 0 {
		log.Error("Sync opt spines", "spines", len(spines), "peers.Len", d.peers.Len(), "err", errNoPeers, "spines", spines)
		return errNoPeers
	}

	//select peer
	for _, con := range d.peers.AllPeers() {
		err := d.SyncUnloadedParents(con.id, spines)
		switch err {
		case nil:
			//check all spines loaded
			unloadedSpines := make(common.HashArray, 0, len(spines))
			for _, spine := range spines {
				if d.blockchain.GetHeaderByHash(spine) == nil {
					unloadedSpines = append(unloadedSpines, spine)
				}
			}
			if len(unloadedSpines) > 0 {
				spines = unloadedSpines
				continue
			}
			return nil
		case errBusy, errCanceled:
			log.Warn("Sync opt spines failed, trying next peer",
				"err", err,
				"spines", spines,
			)
			continue
		}
		if errors.Is(err, errInvalidChain) || errors.Is(err, errBadPeer) || errors.Is(err, errTimeout) ||
			errors.Is(err, errStallingPeer) || errors.Is(err, errEmptyHeaderSet) ||
			errors.Is(err, errPeersUnavailable) || errors.Is(err, errTooOld) || errors.Is(err, errInvalidAncestor) ||
			errors.Is(err, errInvalidBody) {
			log.Warn("Sync opt spines failed, dropping peer", "peer", con.id, "err", err)
			if d.dropPeer == nil {
				// The dropPeer method is nil when `--copydb` is used for a local copy.
				// Timeouts can occur if e.g. compaction hits at the wrong time, and can be ignored
				log.Warn("Sync opt spines wants to drop peer, but peerdrop-function is not set", "peer", con.id)
			} else {
				d.dropPeer(con.id)
			}
		}
		log.Warn("Sync opt spines failed, trying next peer", "err", err)
	}
	return errNoPeers
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

	//select peer
	for _, con := range d.peers.AllPeers() {
		err := d.peerSyncBySpines(con, baseSpine, spines)
		switch err {
		case nil:
			return nil
		case ErrInvalidBaseSpine:
			return ErrInvalidBaseSpine
		case errBusy, errCanceled:
			log.Warn("Sync failed, trying next peer",
				"err", err,
				"baseSpine", baseSpine.Hex(),
			)
			continue
		}
		if errors.Is(err, errInvalidChain) || errors.Is(err, errBadPeer) || errors.Is(err, errTimeout) ||
			errors.Is(err, errStallingPeer) || errors.Is(err, errEmptyHeaderSet) ||
			errors.Is(err, errPeersUnavailable) || errors.Is(err, errTooOld) || errors.Is(err, errInvalidAncestor) ||
			errors.Is(err, errInvalidBody) {
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
	if err = d.peerSyncBySpinesByChunk(p, baseSpine, spines, true); err != nil {
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
		return false, nil, ErrInvalidBaseSpine
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
	d.setPeerSync(p.id)
	defer d.resetPeerSync(p.id)

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
	d.setPeerSync(p.id)
	defer d.resetPeerSync(p.id)

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
func (d *Downloader) peerSyncBySpinesByChunk(p *peerConnection, baseSpine common.Hash, spines common.HashArray, forceDownload bool) error {
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
	const iterLimit = 10 //xeth.LimitDagHashes = 10240 hashes
	for i := 0; ; i++ {
		lastHash, err := d.syncBySpines(p, fromHash, terminalSpine, forceDownload)
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
		if lastHash == (common.Hash{}) || lastHash == fromHash || i > iterLimit {
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

func (d *Downloader) syncBySpines(p *peerConnection, baseSpine, terminalSpine common.Hash, forceDownload bool) (common.Hash, error) {
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

	var dag common.HashArray
	if forceDownload {
		dag = remoteHashes.Copy()
	} else {
		// filter existed blocks
		dag = make(common.HashArray, 0, len(remoteHashes))
		dagBlocks := d.blockchain.GetBlocksByHashes(remoteHashes)
		for h, b := range dagBlocks {
			if b == nil && h != (common.Hash{}) {
				dag = append(dag, h)
			}
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

	blocks := make(types.Blocks, len(headers))
	for i, header := range headers {
		txs := txsMap[header.Hash()]
		block := types.NewBlockWithHeader(header).WithBody(txs)
		// quick body validation
		if block.BodyHash() != block.Body().CalculateHash() {
			return lastHash, errInvalidBody
		}
		blocks[i] = block
	}

	if bl, err := d.blockchain.WriteSyncBlocks(blocks, false); err != nil {
		if errors.Is(err, core.ErrInsertUncompletedDag) {
			log.Warn("Sync by spines: writing blocks failed", "err", err, "bl.Slot", bl.Slot(), "hash", bl.Hash().Hex())
			return lastHash, nil
		}
		log.Error("Sync by spines: writing blocks failed", "err", err, "bl.Slot", bl.Slot(), "hash", bl.Hash().Hex())
		return lastHash, err
	}
	return lastHash, err
}

// nolint:unused // todo: check
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
		// quick body validation
		if block.BodyHash() != block.Body().CalculateHash() {
			return errInvalidBody
		}
		blocks[i] = block
	}

	if bl, err := d.blockchain.WriteSyncBlocks(blocks, true); err != nil {
		if errors.Is(err, core.ErrInsertUncompletedDag) {
			log.Warn("Sync by slots: writing blocks failed", "err", err, "bl.Slot", bl.Slot(), "hash", bl.Hash().Hex())
			return nil
		}
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
			go func(i int, con *peerConnection) {
				defer d.resetPeerSync(con.id)
				defer wg.Done()
				if err = d.multiPeerGetHashes(con, baseSpine, spines); err != nil {
					if errors.Is(err, errTimeout) {
						log.Warn("Sync head failed: timed out", "i", i, "peer", con.id, "err", err)
					}
					if errors.Is(err, errBusy) {
						log.Warn("Sync head failed: peer busy", "i", i, "peer", con.id, "err", err)
					}
					if errors.Is(err, errInvalidChain) || errors.Is(err, errBadPeer) || //errors.Is(err, errTimeout) ||
						errors.Is(err, errStallingPeer) || errors.Is(err, errEmptyHeaderSet) ||
						errors.Is(err, errPeersUnavailable) || errors.Is(err, errTooOld) || errors.Is(err, errInvalidAncestor) ||
						errors.Is(err, errInvalidBody) {
						log.Warn("Sync head failed, dropping peer", "i", i, "peer", con.id, "err", err)
						if d.dropPeer == nil {
							// The dropPeer method is nil when `--copydb` is used for a local copy.
							// Timeouts can occur if e.g. compaction hits at the wrong time, and can be ignored
							log.Warn("Sync head: downloader wants to drop peer, but peerdrop-function is not set", "peer", con.id)
						} else {
							log.Warn("Sync head failed, dropping peer", "i", i, "peer", con.id)
							d.dropPeer(con.id)
						}
					}
					log.Warn("Sync head: failed", "i", i, "peer", con.id, "err", err)
					sem <- struct{}{}
					return
				} else {
					count++
					if count == MaxPeerCon {
						cancel()
					}
				}
			}(i, con)
		}
	}
	wg.Wait()
	log.Info("Sync head: hashes retrieved", "dag", d.syncPeerHashes)

	// single peer sync

	for pid, dag := range d.syncPeerHashes {
		con := d.peers.Peer(pid)
		err = d.syncWithPeerUnknownDagBlocks(con, dag)
		//err = d.syncWithPeerUnknownBlocksWithParents(con, dag)
		if err != nil {
			log.Error("Sync head: block fetching failed", "err", err, "peer", pid, "blocks", dag)
		}
	}
	log.Info("Sync head:", "dag", d.syncPeerHashes)
	cancel()
	return nil
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
	d.setPeerSync(p.id)
	defer d.resetPeerSync(p.id)

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
	d.syncPeers[peerId].IncrProcCount()
}

func (d *Downloader) resetPeerSync(peerId string) {
	d.syncPeersMutex.Lock()
	defer d.syncPeersMutex.Unlock()

	if ch, ok := d.syncPeers[peerId]; ok {
		ch.DecrProcCount()
		if ch.ProcCount() <= 0 {
			ch.Close()
			delete(d.syncPeers, peerId)
		}
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
		return ch
	}
	return nil
}

func (d *Downloader) filterConsistentParentsChains(headers types.Headers) (filtered, failed types.Headers, unknownParents common.HashArray) {
	knownHeaderMap := make(map[common.Hash]bool)
	parentsHdrMap := make(map[common.Hash]common.HashArray)
	for _, hdr := range headers {
		if hdr == nil {
			continue
		}
		knownHeaderMap[hdr.Hash()] = true
		hdrHash := hdr.Hash()
		for _, ph := range hdr.ParentHashes {
			if _, ok := parentsHdrMap[ph]; !ok {
				parentsHdrMap[ph] = make(common.HashArray, 0, len(headers))
			}
			parentsHdrMap[ph] = append(parentsHdrMap[ph], hdrHash)
		}
	}
	unknownParents = make(common.HashArray, 0, len(parentsHdrMap))

	//collect unloaded parents
	var wg sync.WaitGroup
	var loadedMu sync.RWMutex
	for ph := range parentsHdrMap {
		if knownHeaderMap[ph] {
			continue
		}
		wg.Add(1)
		go func(pHash common.Hash) {
			defer wg.Done()
			parent := d.blockchain.GetHeaderByHash(pHash)
			if parent == nil {
				unknownParents = append(unknownParents, pHash)
			} else {
				loadedMu.Lock()
				knownHeaderMap[pHash] = true
				loadedMu.Unlock()
			}
		}(ph)
	}
	wg.Wait()

	//collect failed headers
	failedHeaderMap := make(map[common.Hash]bool)
	for _, unph := range unknownParents {
		for _, fh := range parentsHdrMap[unph] {
			failedHeaderMap[fh] = true
		}
	}

	// filter target headers
	filtered = make(types.Headers, 0, len(headers)-len(failedHeaderMap))
	failed = make(types.Headers, 0, len(failedHeaderMap))
	for _, hdr := range headers {
		hash := hdr.Hash()
		if failedHeaderMap[hash] {
			failed = append(failed, hdr)
		} else {
			filtered = append(filtered, hdr)
		}
	}
	return filtered, failed, unknownParents
}
