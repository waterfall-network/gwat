// Copyright 2014 The go-ethereum Authors
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

// Package core implements the Ethereum consensus protocol.
package core

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	lru "github.com/hashicorp/golang-lru"
	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/common/mclock"
	"gitlab.waterfall.network/waterfall/protocol/gwat/common/prque"
	"gitlab.waterfall.network/waterfall/protocol/gwat/consensus"
	"gitlab.waterfall.network/waterfall/protocol/gwat/consensus/misc"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/rawdb"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/state"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/state/snapshot"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/types"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/vm"
	"gitlab.waterfall.network/waterfall/protocol/gwat/ethdb"
	"gitlab.waterfall.network/waterfall/protocol/gwat/event"
	"gitlab.waterfall.network/waterfall/protocol/gwat/internal/syncx"
	"gitlab.waterfall.network/waterfall/protocol/gwat/log"
	"gitlab.waterfall.network/waterfall/protocol/gwat/metrics"
	"gitlab.waterfall.network/waterfall/protocol/gwat/params"
	"gitlab.waterfall.network/waterfall/protocol/gwat/token"
	"gitlab.waterfall.network/waterfall/protocol/gwat/trie"
	"gitlab.waterfall.network/waterfall/protocol/gwat/validator"
	"gitlab.waterfall.network/waterfall/protocol/gwat/validator/era"
	validatorOp "gitlab.waterfall.network/waterfall/protocol/gwat/validator/operation"
	valStore "gitlab.waterfall.network/waterfall/protocol/gwat/validator/storage"
)

var (
	headBlockGauge     = metrics.NewRegisteredGauge("chain/head/block", nil)
	headHeaderGauge    = metrics.NewRegisteredGauge("chain/head/header", nil)
	headFastBlockGauge = metrics.NewRegisteredGauge("chain/head/receipt", nil)

	accountReadTimer   = metrics.NewRegisteredTimer("chain/account/reads", nil)
	accountHashTimer   = metrics.NewRegisteredTimer("chain/account/hashes", nil)
	accountUpdateTimer = metrics.NewRegisteredTimer("chain/account/updates", nil)
	accountCommitTimer = metrics.NewRegisteredTimer("chain/account/commits", nil)

	storageReadTimer   = metrics.NewRegisteredTimer("chain/storage/reads", nil)
	storageHashTimer   = metrics.NewRegisteredTimer("chain/storage/hashes", nil)
	storageUpdateTimer = metrics.NewRegisteredTimer("chain/storage/updates", nil)
	storageCommitTimer = metrics.NewRegisteredTimer("chain/storage/commits", nil)

	snapshotAccountReadTimer = metrics.NewRegisteredTimer("chain/snapshot/account/reads", nil)
	snapshotStorageReadTimer = metrics.NewRegisteredTimer("chain/snapshot/storage/reads", nil)
	snapshotCommitTimer      = metrics.NewRegisteredTimer("chain/snapshot/commits", nil)

	blockInsertTimer     = metrics.NewRegisteredTimer("chain/inserts", nil)
	blockValidationTimer = metrics.NewRegisteredTimer("chain/validation", nil)
	blockExecutionTimer  = metrics.NewRegisteredTimer("chain/execution", nil)
	blockWriteTimer      = metrics.NewRegisteredTimer("chain/write", nil)

	//blockReorgMeter = metrics.NewRegisteredMeter("chain/reorg/executes", nil)
	//blockReorgAddMeter = metrics.NewRegisteredMeter("chain/reorg/add", nil)
	//blockReorgDropMeter     = metrics.NewRegisteredMeter("chain/reorg/drop", nil)
	blockReorgInvalidatedTx = metrics.NewRegisteredMeter("chain/reorg/invalidTx", nil)

	blockPrefetchExecuteTimer   = metrics.NewRegisteredTimer("chain/prefetch/executes", nil)
	blockPrefetchInterruptMeter = metrics.NewRegisteredMeter("chain/prefetch/interrupts", nil)

	errInsertionInterrupted = errors.New("insertion is interrupted")
	errChainStopped         = errors.New("blockchain is stopped")
	errInvalidBlock         = errors.New("invalid block")
	errBlockNotFound        = errors.New("block not found")
)

const (
	valSyncCacheLimit          = 128
	bodyCacheLimit             = 256
	blockCacheLimit            = 256
	receiptsCacheLimit         = 32
	txLookupCacheLimit         = 1024
	TriesInMemory              = 128
	invBlocksCacheLimit        = 512
	optimisticSpinesCacheLimit = 128
	checkpointCacheLimit       = 16
	// BlockChainVersion ensures that an incompatible database forces a resync from scratch.
	//
	// Changelog:
	//
	// - Version 4
	//   The following incompatible database changes were added:
	//   * the `BlockNumber`, `TxHash`, `TxIndex`, `BlockHash` and `Index` fields of log are deleted
	//   * the `Bloom` field of receipt is deleted
	//   * the `BlockIndex` and `TxIndex` fields of txlookup are deleted
	// - Version 5
	//  The following incompatible database changes were added:
	//    * the `TxHash`, `GasCost`, and `ContractAddress` fields are no longer stored for a receipt
	//    * the `TxHash`, `GasCost`, and `ContractAddress` fields are computed by looking up the
	//      receipts' corresponding block
	// - Version 6
	//  The following incompatible database changes were added:
	//    * Transaction lookup information stores the corresponding block number instead of block hash
	// - Version 7
	//  The following incompatible database changes were added:
	//    * Use freezer as the ancient database to maintain all ancient data
	// - Version 8
	//  The following incompatible database changes were added:
	//    * New scheme for contract code in order to separate the codes and trie nodes
	BlockChainVersion uint64 = 8

	opCreate    = "create"
	opPropagate = "propagate"
	opDelay     = "delay"
	opSync      = "sync"
)

// CacheConfig contains the configuration values for the trie caching/pruning
// that's resident in a blockchain.
type CacheConfig struct {
	TrieCleanLimit      int           // Memory allowance (MB) to use for caching trie nodes in memory
	TrieCleanJournal    string        // Disk journal for saving clean cache entries.
	TrieCleanRejournal  time.Duration // Time interval to dump clean cache to disk periodically
	TrieCleanNoPrefetch bool          // Whether to disable heuristic state prefetching for followup blocks
	TrieDirtyLimit      int           // Memory limit (MB) at which to start flushing dirty trie nodes to disk
	TrieDirtyDisabled   bool          // Whether to disable trie write caching and GC altogether (archive node)
	TrieTimeLimit       time.Duration // Time limit after which to flush the current in-memory trie to disk
	SnapshotLimit       int           // Memory allowance (MB) to use for caching snapshot entries in memory
	Preimages           bool          // Whether to store preimage of trie key to the disk

	SnapshotWait bool // Wait for snapshot construction on startup. TODO(karalabe): This is a dirty hack for testing, nuke it
}

// defaultCacheConfig are the default caching values if none are specified by the
// user (also used during testing).
var defaultCacheConfig = &CacheConfig{
	TrieCleanLimit: 256,
	TrieDirtyLimit: 256,
	TrieTimeLimit:  5 * time.Minute,
	SnapshotLimit:  256,
	SnapshotWait:   true,
}

// BlockChain represents the canonical chain given a database with a genesis
// block. The Blockchain manages chain imports, reverts, chain reorganisations.
//
// Importing blocks in to the block chain happens according to the set of rules
// defined by the two stage Validator. Processing of blocks is done using the
// Processor which processes the included transaction. The validation of the state
// is done in the second part of the Validator. Failing results in aborting of
// the import.
//
// The BlockChain also helps in returning blocks from **any** chain included
// in the database as well as blocks that represents the canonical chain. It's
// important to note that GetBlock can return any block and does not need to be
// included in the canonical one where as GetBlockByNumber always represents the
// canonical chain.
type BlockChain struct {
	chainConfig *params.ChainConfig // Chain & network configuration
	cacheConfig *CacheConfig        // Cache configuration for pruning
	slotInfo    *types.SlotInfo     // coordinator slot settings
	db          ethdb.Database      // Low level persistent database to store final content in
	snaps       *snapshot.Tree      // Snapshot tree for fast trie leaf access
	triegc      *prque.Prque        // Priority queue mapping block numbers to tries to gc
	gcproc      time.Duration       // Accumulates canonical block processing for trie dumping

	// txLookupLimit is the maximum number of blocks from head whose tx indices
	// are reserved:
	//  * 0:   means no limit and regenerate any missing indexes
	//  * N:   means N block limit [HEAD-N+1, HEAD] and delete extra indexes
	//  * nil: disable tx reindexer/deleter, but still index new blocks
	txLookupLimit uint64

	hc             *HeaderChain
	rmLogsFeed     event.Feed
	chainFeed      event.Feed
	chainSideFeed  event.Feed
	chainHeadFeed  event.Feed
	logsFeed       event.Feed
	blockProcFeed  event.Feed
	processingFeed event.Feed
	rmTxFeed       event.Feed
	scope          event.SubscriptionScope
	genesisBlock   *types.Block

	dagMu sync.RWMutex // finalizing lock

	// This mutex synchronizes chain write operations.
	// Readers don't need to take it, they can just read the database.
	chainmu *syncx.ClosableMutex

	lastFinalizedBlock     atomic.Value // Current last finalized block of the blockchain
	lastFinalizedFastBlock atomic.Value // Current last finalized block of the fast-sync chain (may be above the blockchain!)
	eraInfo                era.EraInfo  // Current Era
	lastCoordinatedCp      atomic.Value // Current last coordinated checkpoint
	notProcValSyncOps      map[common.Hash]*types.ValidatorSync
	valSyncCache           *lru.Cache

	stateCache            state.Database // State database to reuse between imports (contains state cache)
	bodyCache             *lru.Cache     // Cache for the most recent block bodies
	bodyRLPCache          *lru.Cache     // Cache for the most recent block bodies in RLP encoded format
	receiptsCache         *lru.Cache     // Cache for the most recent receipts per block
	blockCache            *lru.Cache     // Cache for the most recent entire blocks
	txLookupCache         *lru.Cache     // Cache for the most recent transaction lookup data.
	invalidBlocksCache    *lru.Cache     // Cache for the blocks with unknown parents
	optimisticSpinesCache *lru.Cache
	checkpointCache       *lru.Cache
	checkpointSyncCache   *types.Checkpoint

	validatorStorage valStore.Storage

	insBlockCache []*types.Block // Cache for blocks to insert late

	wg            sync.WaitGroup //
	quit          chan struct{}  // shutdown signal, closed in Stop.
	running       int32          // 0 if chain is running, 1 when stopped
	procInterrupt int32          // interrupt signaler for block processing

	validator    Validator // Block and state validator interface
	prefetcher   Prefetcher
	processor    Processor // Block transaction processor interface
	vmConfig     vm.Config
	syncProvider types.SyncProvider
	isSynced     bool
	isSyncedM    sync.Mutex
}

// NewBlockChain returns a fully initialised block chain using information
// available in the database. It initialises the default Ethereum Validator and
// Processor.
func NewBlockChain(db ethdb.Database, cacheConfig *CacheConfig, chainConfig *params.ChainConfig, vmConfig vm.Config, txLookupLimit *uint64) (*BlockChain, error) {
	if cacheConfig == nil {
		cacheConfig = defaultCacheConfig
	}
	valSyncCache, _ := lru.New(valSyncCacheLimit)
	bodyCache, _ := lru.New(bodyCacheLimit)
	bodyRLPCache, _ := lru.New(bodyCacheLimit)
	receiptsCache, _ := lru.New(receiptsCacheLimit)
	blockCache, _ := lru.New(blockCacheLimit)
	txLookupCache, _ := lru.New(txLookupCacheLimit)
	invBlocksCache, _ := lru.New(invBlocksCacheLimit)
	optimisticSpinesCache, _ := lru.New(optimisticSpinesCacheLimit)
	checkpointCache, _ := lru.New(checkpointCacheLimit)

	bc := &BlockChain{
		chainConfig: chainConfig,
		cacheConfig: cacheConfig,
		db:          db,
		triegc:      prque.New(nil),
		stateCache: state.NewDatabaseWithConfig(db, &trie.Config{
			Cache:     cacheConfig.TrieCleanLimit,
			Journal:   cacheConfig.TrieCleanJournal,
			Preimages: cacheConfig.Preimages,
		}),
		quit:                  make(chan struct{}),
		chainmu:               syncx.NewClosableMutex(),
		valSyncCache:          valSyncCache,
		bodyCache:             bodyCache,
		bodyRLPCache:          bodyRLPCache,
		receiptsCache:         receiptsCache,
		blockCache:            blockCache,
		txLookupCache:         txLookupCache,
		invalidBlocksCache:    invBlocksCache,
		optimisticSpinesCache: optimisticSpinesCache,
		checkpointCache:       checkpointCache,
		vmConfig:              vmConfig,
		syncProvider:          nil,
		validatorStorage:      valStore.NewStorage(chainConfig),
	}
	bc.validator = NewBlockValidator(chainConfig, bc)
	bc.prefetcher = newStatePrefetcher(chainConfig, bc)
	bc.processor = NewStateProcessor(chainConfig, bc)

	var err error
	bc.hc, err = NewHeaderChain(db, chainConfig, bc.insertStopped)
	if err != nil {
		return nil, err
	}
	bc.genesisBlock = bc.GetBlockByNumber(0)
	if bc.genesisBlock == nil {
		return nil, ErrNoGenesis
	}

	if bc.GetBlockFinalizedNumber(bc.genesisBlock.Hash()) == nil {
		rawdb.WriteLastFinalizedHash(db, bc.genesisBlock.Hash())
		rawdb.WriteFinalizedHashNumber(db, bc.genesisBlock.Hash(), uint64(0))
		log.Info("Save genesis hash", "hash", bc.genesisBlock.Hash(), "fn", "NewBlockChain")
	}

	var nilBlock = bc.genesisBlock
	bc.lastFinalizedBlock.Store(nilBlock)
	bc.lastFinalizedFastBlock.Store(nilBlock)

	// Initialize the chain with ancient data if it isn't empty.
	var txIndexBlock uint64

	//Start Freezer
	isEmpty := bc.empty()
	if isEmpty {
		//Start Freezer
		rawdb.InitDatabaseFromFreezer(bc.db)
		// If ancient database is not empty, reconstruct all missing
		// indices in the background.
		frozen, _ := bc.db.Ancients()
		if frozen > 0 {
			txIndexBlock = frozen
		}
	}

	lastCP := bc.GetLastCoordinatedCheckpoint()
	if lastCP == nil {
		lfb := bc.GetLastFinalizedBlock()
		lastCP = bc.GetCoordinatedCheckpoint(lfb.CpHash())
	}
	lfHash := rawdb.ReadLastFinalizedHash(bc.db)
	lfNr := rawdb.ReadFinalizedNumberByHash(bc.db, lfHash)
	if lfNr != nil {
		err = bc.RollbackFinalization(lastCP.Spine, *lfNr)
		if err == nil {
			err = bc.loadLastState()
		}
	}
	if lfNr == nil || err != nil {
		log.Error("Node initializing: rollback finalization failed: try hard reset", "lfNr", lfNr, "err", err)
		// hard rollback
		err = bc.SetHead(lastCP.Spine)
		if err != nil {
			// search valid checkpoint and try again
			lastCP = bc.searchValidCheckpoint(lastCP.Epoch)
			if lastCP == nil {
				log.Crit("Node initializing failed", "err", err)
			}
			err = bc.SetHead(lastCP.Spine)
			if err != nil {
				log.Crit("Node initializing failed (retry)", "err", err)
			}
		}
	}

	// Ensure that a previous crash in SetHead doesn't leave extra ancients
	if frozen, err := bc.db.Ancients(); err == nil && frozen > 0 {
		var (
			needRewind bool
			low        uint64
			headHash   common.Hash
		)
		// The head full block may be rolled back to a very low height due to
		// blockchain repair. If the head full block is even lower than the ancient
		// chain, truncate the ancient store.
		fullBlock := bc.GetLastFinalizedBlock()
		if fullBlock != nil && fullBlock.Hash() != bc.genesisBlock.Hash() && fullBlock.Nr() < frozen-1 {
			needRewind = true
			low = fullBlock.Nr()
			headHash = fullBlock.Hash()
		}
		// In fast sync, it may happen that ancient data has been written to the
		// ancient store, but the LastFastBlock has not been updated, truncate the
		// extra data here.
		fastBlock := bc.GetLastFinalizedFastBlock()
		if fastBlock != nil && fastBlock.Nr() < frozen-1 {
			needRewind = true
			if fastBlock.Nr() < low || low == 0 {
				low = fastBlock.Nr()
				headHash = fullBlock.Hash()
			}
		}
		if needRewind {
			log.Error("Truncating ancient chain", "from", bc.GetLastFinalizedHeader().Nr(), "to", low, "hash", headHash.Hex())
			if err := bc.SetHead(headHash); err != nil {
				return nil, err
			}
		}
	}

	// Load any existing snapshot, regenerating it if loading failed
	if bc.cacheConfig.SnapshotLimit > 0 {
		// If the chain was rewound past the snapshot persistent layer (causing
		// a recovery block number to be persisted to disk), check if we're still
		// in recovery mode and in that case, don't invalidate the snapshot on a
		// head mismatch.
		//var recover bool

		head := bc.GetLastFinalizedBlock()
		//layer := rawdb.ReadSnapshotRecoveryNumber(bc.db)
		//if layer != nil && *layer > head.Nr() {
		//	log.Warn("Enabling snapshot recovery", "chainhead", head.Nr(), "diskbase", *layer)
		//	recover = true
		//}
		bc.snaps, _ = snapshot.New(bc.db, bc.stateCache.TrieDB(), bc.cacheConfig.SnapshotLimit, head.Root(), !bc.cacheConfig.SnapshotWait, true, true)
	}

	// Start tx indexer/unindexer.
	if txLookupLimit != nil {
		bc.txLookupLimit = *txLookupLimit

		bc.wg.Add(1)
		go bc.maintainTxIndex(txIndexBlock)
	}
	// If periodic cache journal is required, spin it up.
	if bc.cacheConfig.TrieCleanRejournal > 0 {
		if bc.cacheConfig.TrieCleanRejournal < time.Minute {
			log.Warn("Sanitizing invalid trie cache journal time", "provided", bc.cacheConfig.TrieCleanRejournal, "updated", time.Minute)
			bc.cacheConfig.TrieCleanRejournal = time.Minute
		}
		triedb := bc.stateCache.TrieDB()
		bc.wg.Add(1)
		go func() {
			defer bc.wg.Done()
			triedb.SaveCachePeriodically(bc.cacheConfig.TrieCleanJournal, bc.cacheConfig.TrieCleanRejournal, bc.quit)
		}()
	}

	bc.notProcValSyncOps = bc.GetNotProcessedValidatorSyncData()

	return bc, nil
}

// empty returns an indicator whether the blockchain is empty.
// Note, it's a special case that we connect a non-empty ancient
// database with an empty node, so that we can plugin the ancient
// into node seamlessly.
func (bc *BlockChain) empty() bool {
	genesis := bc.genesisBlock.Hash()
	for _, hash := range []common.Hash{rawdb.ReadLastCanonicalHash(bc.db), rawdb.ReadLastFinalizedHash(bc.db), rawdb.ReadHeadFastBlockHash(bc.db)} {
		if hash != genesis {
			return false
		}
	}
	return true
}

// loadLastState loads the last known chain state from the database. This method
// assumes that the chain manager mutex is held.
func (bc *BlockChain) loadLastState() error {
	// Restore the last known head blocks hashes
	lastFinHash := rawdb.ReadLastFinalizedHash(bc.db)
	lastFinNr := rawdb.ReadLastFinalizedNumber(bc.db)

	if lastFinHash == (common.Hash{}) {
		log.Crit("Database corrupted, use backup or init new one: no last finalized data")
	}
	// Make sure the entire head block is available
	lfBlock := bc.GetBlockByHash(lastFinHash)
	if lfBlock == nil {
		log.Crit("Database corrupted, use backup or init new one: last finalized block not found",
			"block", lastFinHash.Hex(),
		)
	}

	// Everything seems to be fine, set as the lastFinHash block
	bc.lastFinalizedBlock.Store(lfBlock)
	headBlockGauge.Update(int64(lastFinNr))

	// Restore the last known head header
	bc.hc.SetLastFinalisedHeader(lfBlock.Header(), lastFinNr)

	// Restore the last known lastFinHash fast block
	bc.lastFinalizedFastBlock.Store(lfBlock)
	headFastBlockGauge.Update(int64(lastFinNr))

	if head := rawdb.ReadHeadFastBlockHash(bc.db); head != (common.Hash{}) {
		if block := bc.GetBlockByHash(head); block != nil {
			bc.lastFinalizedFastBlock.Store(block)
			headFastBlockGauge.Update(int64(lastFinNr))
		}
	}

	//load dag part of chain
	if err := bc.hc.loadTips(); err != nil {
		log.Warn("State loading", "err", err)
		bc.ResetTips()
	}
	tips := bc.GetTips()
	if len(tips) == 0 {
		bc.ResetTips()
		tips = bc.GetTips()
	}

	// Issue a status log for the user
	lastFinalizedFastBlocks := bc.GetLastFinalizedFastBlock()

	//load state
	tmpSi := &types.SlotInfo{
		GenesisTime:    bc.Genesis().Time(),
		SecondsPerSlot: bc.Config().SecondsPerSlot,
		SlotsPerEpoch:  bc.Config().SlotsPerEpoch,
	}
	headEpoch := tmpSi.SlotToEpoch(lfBlock.Slot())
	if _, err := state.New(lfBlock.Root(), bc.stateCache, bc.snaps); err != nil {
		// search valid checkpoint
		cp := bc.searchValidCheckpoint(headEpoch)
		if cp == nil {
			log.Crit("Set head: valid checkpoint not found (init state)", "headEpoch", headEpoch, "head", lfBlock.Hash().Hex())
			return errBlockNotFound
		}
		//recursive call
		return bc.SetHead(cp.Spine)
	}

	lcp := rawdb.ReadLastCoordinatedCheckpoint(bc.db)
	bc.SetLastCoordinatedCheckpoint(lcp)

	log.Info("Loaded last finalized block", "number", lastFinNr, "slot", lfBlock.Slot(), "era", lfBlock.Era(), "hash", lfBlock.Hash().Hex())
	log.Info("Loaded last finalized checkpoint", "finEpoch", lcp.FinEpoch, "epoch", lcp.Epoch, "spine", lcp.Spine.Hex(), "root", lcp.Root.Hex())
	log.Info("Loaded last finalized fast block", "hash", lastFinalizedFastBlocks.Hash(), "finNr", lastFinNr)
	log.Info("Loaded tips", "hashes", tips.GetHashes())
	if pivot := rawdb.ReadLastPivotNumber(bc.db); pivot != nil {
		log.Info("Loaded last fast-sync pivot marker", "number", *pivot)
	}
	return nil
}

// SetSlotInfo set new slot info.
func (bc *BlockChain) SetSlotInfo(si *types.SlotInfo) error {
	if si == nil {
		return ErrBadSlotInfo
	}
	bc.slotInfo = si.Copy()
	return nil
}

// GetSlotInfo get current slot info.
func (bc *BlockChain) GetSlotInfo() *types.SlotInfo {
	return bc.slotInfo.Copy()
}

// SetLastCoordinatedCheckpoint set last coordinated checkpoint.
func (bc *BlockChain) SetLastCoordinatedCheckpoint(cp *types.Checkpoint) {
	// save new checkpoint and cache it.
	var batch ethdb.Batch
	if epCp := bc.GetCoordinatedCheckpoint(cp.Spine); epCp == nil {
		batch = bc.db.NewBatch()
		rawdb.WriteCoordinatedCheckpoint(batch, cp)
		bc.checkpointCache.Add(cp.Spine, cp)
	}
	//update current cp and apoch data.
	currCp := bc.GetLastCoordinatedCheckpoint()
	if currCp == nil || cp.Root != currCp.Root || cp.FinEpoch != currCp.FinEpoch {
		bc.lastCoordinatedCp.Store(cp.Copy())
		if batch == nil {
			batch = bc.db.NewBatch()
		}
		rawdb.WriteLastCoordinatedCheckpoint(batch, cp)
		rawdb.WriteEpoch(batch, cp.FinEpoch, cp.Spine)
		log.Info("Update coordinated checkpoint ", "cp", cp)
	}
	if batch != nil {
		if err := batch.Write(); err != nil {
			log.Crit("Set last coordinated checkpoint failed", "err", err)
		}
	}
	// rm stale blockDags
	go func() {
		if currCp != nil && cp.Root != currCp.Root {
			cpHeader := bc.GetHeader(currCp.Spine)
			if cpHeader != nil {
				uptoNr := cpHeader.CpNumber
				bc.ClearStaleBlockDags(uptoNr)
			}
		}
	}()
}

func (bc *BlockChain) ClearStaleBlockDags(uptoNr uint64) {
	for nr := uptoNr; uptoNr > 0; nr-- {
		h := bc.ReadFinalizedHashByNumber(nr)
		if h == (common.Hash{}) {
			return
		}
		bdag := bc.GetBlockDag(h)
		if bdag == nil {
			return
		}
		//log.Info("Rm blockDag", "nr", nr, "slot", bdag.Slot, "height", bdag.Height, "cpHeight", bdag.CpHeight, "hash", h.Hex())
		bc.DeleteBlockDag(h)
	}
}

// GetLastCoordinatedCheckpoint retrieves the latest coordinated checkpoint.
func (bc *BlockChain) GetLastCoordinatedCheckpoint() *types.Checkpoint {
	if bc.lastCoordinatedCp.Load() == nil {
		cp := rawdb.ReadLastCoordinatedCheckpoint(bc.db)
		if cp == nil {
			log.Warn("Checkpoint not found in db")
			return nil
		}
		bc.lastCoordinatedCp.Store(cp.Copy())
	}
	return bc.lastCoordinatedCp.Load().(*types.Checkpoint)
}

// SetSyncCheckpointCache set cp to cache for sync purposes.
func (bc *BlockChain) SetSyncCheckpointCache(cp *types.Checkpoint) {
	bc.checkpointSyncCache = cp.Copy()
}

// ResetSyncCheckpointCache reset cp to cache for sync purposes.
func (bc *BlockChain) ResetSyncCheckpointCache() {
	bc.checkpointSyncCache = nil
}
func (bc *BlockChain) getSyncCheckpointCache(cpSpine common.Hash) *types.Checkpoint {
	if bc.checkpointSyncCache != nil && bc.checkpointSyncCache.Spine == cpSpine {
		log.Info("Synchronization cp cache: used", "cpSpine", cpSpine.Hex())
		return bc.checkpointSyncCache
	}
	return nil
}

// GetCoordinatedCheckpoint retrieves a checkpoint dag from the database by hash, caching it if found.
func (bc *BlockChain) GetCoordinatedCheckpoint(cpSpine common.Hash) *types.Checkpoint {
	//check in cache
	if v, ok := bc.checkpointCache.Get(cpSpine); ok {
		val := v.(*types.Checkpoint)
		if val != nil {
			return val
		}
		bc.checkpointCache.Remove(cpSpine)
	}
	cp := rawdb.ReadCoordinatedCheckpoint(bc.db, cpSpine)
	if cp == nil {
		// final check in sync cache
		return bc.getSyncCheckpointCache(cpSpine)
	}
	// Cache the found cp for next time and return
	bc.checkpointCache.Add(cpSpine, cp)
	return cp
}

// GetEpoch retrieves the checkpoint spine by epoch.
func (bc *BlockChain) GetEpoch(epoch uint64) common.Hash {
	return rawdb.ReadEpoch(bc.db, epoch)
}

// GetValidatorSyncData retrieves a validator sync data from the database by
// creator addr and op, caching it if found.
func (bc *BlockChain) GetValidatorSyncData(initTxHash common.Hash) *types.ValidatorSync {
	// Short circuit if the body's already in the cache, retrieve otherwise
	if cached, ok := bc.valSyncCache.Get(initTxHash); ok {
		vs := cached.(*types.ValidatorSync)
		return vs
	}
	vs := rawdb.ReadValidatorSync(bc.db, initTxHash)
	if vs == nil {
		return nil
	}
	// Cache the found data for next time and return
	bc.valSyncCache.Add(initTxHash, vs)
	return vs
}

func (bc *BlockChain) SetValidatorSyncData(validatorSync *types.ValidatorSync) {
	key := validatorSync.Key()
	if _, ok := bc.valSyncCache.Get(key); ok {
		bc.valSyncCache.Remove(key)
	}
	bc.valSyncCache.Add(key, validatorSync)
	rawdb.WriteValidatorSync(bc.db, validatorSync)
	if validatorSync.TxHash != nil && bc.notProcValSyncOps[key] != nil {
		notProcValSyncOps := map[common.Hash]*types.ValidatorSync{}
		for k, vs := range bc.notProcValSyncOps {
			if k != key {
				notProcValSyncOps[k] = vs
			}
		}
		bc.notProcValSyncOps = notProcValSyncOps
		vsArr := make([]*types.ValidatorSync, 0, len(bc.notProcValSyncOps))
		for _, vs := range bc.notProcValSyncOps {
			vsArr = append(vsArr, vs)
		}
		rawdb.WriteNotProcessedValidatorSyncOps(bc.db, vsArr)
	}
}

// AppendNotProcessedValidatorSyncData append to not processed validators sync data.
// skips currently existed items
func (bc *BlockChain) AppendNotProcessedValidatorSyncData(valSyncData []*types.ValidatorSync) {
	currOps := bc.GetNotProcessedValidatorSyncData()
	isUpdated := false
	for _, vs := range valSyncData {
		if currOps[vs.Key()] == nil {
			bc.notProcValSyncOps[vs.Key()] = vs
			isUpdated = true
		}
	}
	// rm handled operations
	for k, vs := range bc.notProcValSyncOps {
		if vs.TxHash != nil {
			delete(bc.notProcValSyncOps, k)
			isUpdated = true
		}
	}

	if isUpdated {
		vsArr := make([]*types.ValidatorSync, 0, len(bc.notProcValSyncOps))
		for _, vs := range bc.notProcValSyncOps {
			vsArr = append(vsArr, vs)
		}
		rawdb.WriteNotProcessedValidatorSyncOps(bc.db, vsArr)
	}
}

// GetNotProcessedValidatorSyncData get current not processed validator sync data.
func (bc *BlockChain) GetNotProcessedValidatorSyncData() map[common.Hash]*types.ValidatorSync {
	if bc.notProcValSyncOps == nil {
		ops := rawdb.ReadNotProcessedValidatorSyncOps(bc.db)
		bc.notProcValSyncOps = make(map[common.Hash]*types.ValidatorSync, len(ops))
		for _, op := range ops {
			if op == nil {
				log.Warn("GetNotProcessedValidatorSyncData: nil data detected")
				continue
			}
			bc.notProcValSyncOps[op.Key()] = op
		}
	}
	return bc.notProcValSyncOps
}

// SetHead rewinds the local chain to a new head.
// The method searches finalised block which provide chain consistency,
// starting from passed hash;
// removes all data that follows after head;
// provide required node state to start.
func (bc *BlockChain) SetHead(head common.Hash) error {
	if !bc.chainmu.TryLock() {
		return errChainStopped
	}
	defer bc.chainmu.Unlock()

	bc.SetIsSynced(false)

	// purge bc caches
	bc.bodyCache.Purge()
	bc.bodyRLPCache.Purge()
	bc.receiptsCache.Purge()
	bc.blockCache.Purge()
	bc.txLookupCache.Purge()
	bc.invalidBlocksCache.Purge()
	bc.optimisticSpinesCache.Purge()
	bc.checkpointCache.Purge()
	bc.insBlockCache = make([]*types.Block, 0, len(bc.insBlockCache))
	// purge hc caches
	bc.hc.headerCache.Purge()
	bc.hc.numberCache.Purge()
	bc.hc.ancestorCache.Purge()
	bc.hc.blockDagCache.Purge()

	//// validator Sync ?
	//bc.notProcValSyncOps
	//bc.valSyncCache.Purge()

	err := bc.setHeadRecursive(head)
	if err != nil {
		return err
	}
	return bc.loadLastState()
}

func (bc *BlockChain) setHeadRecursive(head common.Hash) error {
	// find valid head
	headBlock := rawdb.ReadBlock(bc.db, head)
	if headBlock == nil {
		return errBlockNotFound
	}

	tmpSi := &types.SlotInfo{
		GenesisTime:    bc.Genesis().Time(),
		SecondsPerSlot: bc.Config().SecondsPerSlot,
		SlotsPerEpoch:  bc.Config().SlotsPerEpoch,
	}
	headEpoch := tmpSi.SlotToEpoch(headBlock.Slot())

	// if block is not finalized
	// - set as head valid checkpoint starting from the block's epoch
	if headBlock.Nr() == 0 && headBlock.Height() > 0 {
		cp := bc.searchValidCheckpoint(headEpoch)
		if cp == nil {
			log.Error("Set head: valid checkpoint not found (not finalized head)", "headEpoch", headEpoch, "head", head.Hex())
			return errBlockNotFound
		}
		//recursive call
		return bc.setHeadRecursive(cp.Spine)
	}

	// find checkpoint of head
	headCp := bc.searchBlockFinalizationCp(headBlock.Header())
	if headCp == nil {
		// search valid checkpoint
		headCp = bc.searchValidCheckpoint(headEpoch)
		if headCp == nil {
			log.Error("Set head: valid checkpoint not found (get head cp)", "headEpoch", headEpoch, "head", head.Hex())
			return errBlockNotFound
		}
		//recursive call
		return bc.setHeadRecursive(headCp.Spine)
	}

	// check head era
	cpEra := bc.EpochToEra(headCp.FinEpoch)
	if cpEra == nil {
		// search valid checkpoint
		cp := bc.searchValidCheckpoint(headEpoch)
		if cp == nil {
			log.Error("Set head: valid checkpoint not found (get head era)",
				"cpFinEpoch", headCp.FinEpoch,
				"headEpoch", headEpoch,
				"head", head.Hex(),
			)
			return errBlockNotFound
		}
		//recursive call
		return bc.setHeadRecursive(cp.Spine)
	}
	cpEraInfo := era.NewEraInfo(*cpEra)

	// clean chain
	// clean eras
	var lastEra *era.Era
	isHeadTransition := cpEraInfo.IsTransitionPeriodEpoch(bc, headCp.FinEpoch)
	lastEraNr := rawdb.ReadCurrentEra(bc.db)
	for eraNr := lastEraNr + 1; eraNr > cpEraInfo.Number(); eraNr-- {
		if e := rawdb.ReadEra(bc.db, eraNr); e != nil {
			if lastEra == nil {
				lastEra = e
			}
			// if era transition period - keep next era
			if isHeadTransition && eraNr == cpEraInfo.Number()+1 {
				break
			}
			rawdb.DeleteEra(bc.db, eraNr)
		}
	}
	// clean checkpoints && epochs
	var maxEpoch uint64
	if lastEra != nil {
		maxEpoch = lastEra.To
	} else if lcp := rawdb.ReadLastCoordinatedCheckpoint(bc.db); lcp != nil {
		maxEpoch = lcp.FinEpoch
	}
	var cpCache *types.Checkpoint
	for e := maxEpoch; e > headCp.FinEpoch; e-- {
		eSpine := rawdb.ReadEpoch(bc.db, e)
		rawdb.DeleteEpoch(bc.db, e)
		if cpCache == nil || cpCache.Spine != eSpine {
			cpCache = rawdb.ReadCoordinatedCheckpoint(bc.db, eSpine)
		}
		if cpCache != nil && cpCache.Epoch == e {
			rawdb.DeleteCoordinatedCheckpoint(bc.db, eSpine)
		}
	}
	//clean blocks data by slot
	maxSlot, err := tmpSi.SlotOfEpochEnd(maxEpoch)
	if err != nil {
		return err
	}
	for sl := maxSlot; sl >= headBlock.Slot() && sl != 0; sl-- {
		rmHashes := rawdb.ReadSlotBlocksHashes(bc.db, sl)
		for _, rmh := range rmHashes {
			// skip new head
			if rmh == head {
				continue
			}
			bc.rmBlockData(rmh, &sl)
		}
	}

	//clean blocks data by number
	lfNr := rawdb.ReadLastFinalizedNumber(bc.db)
	for nr := lfNr; nr > headBlock.Nr(); nr-- {
		if rmh := rawdb.ReadFinalizedHashByNumber(bc.db, nr); rmh != (common.Hash{}) {
			bc.rmBlockData(rmh, nil)
		}
	}

	//set new cain head
	rawdb.WriteLastCanonicalHash(bc.db, headBlock.Hash())
	rawdb.WriteLastFinalizedHash(bc.db, headBlock.Hash())
	bc.lastFinalizedBlock.Store(headBlock)
	headBlockGauge.Update(int64(headBlock.Nr()))

	//set correct current cp
	rawdb.WriteLastCoordinatedCheckpoint(bc.db, headCp)
	//set correct current era
	rawdb.WriteCurrentEra(bc.db, cpEraInfo.Number())

	frozen, _ := bc.db.Ancients()
	if headBlock.Nr() < frozen {
		// Truncate all relative data from ancient store.
		if err = bc.db.TruncateAncients(headBlock.Nr() + 1); err != nil {
			log.Crit("Set head: truncate ancient failed", "frozen", frozen, "truncateNr", headBlock.Nr()+1, "err", err)
		}
		frozenNr, _ := bc.db.Ancients()
		log.Info("Set head: truncate ancient success", "frozenNr", frozenNr, "truncateNr", headBlock.Nr()+1, "err", err)
	}

	// reset tips
	if err = bc.hc.ResetTips(); err != nil {
		// if any error - try set cp as head
		if headCp.Spine == head {
			//if head is checkpoint - try previous valid checkpoint
			headCp = bc.searchValidCheckpoint(headEpoch)
			if headCp == nil {
				log.Error("Set head: valid checkpoint not found (reset tips)",
					"cpEpoch", headCp.Epoch,
					"headEpoch", headEpoch,
					"head", head.Hex(),
				)
				return errBlockNotFound
			}
		}
		return bc.setHeadRecursive(headCp.Spine)
	}

	return nil
}

// rmBlockData completely removes all data related with block.
// slot - optional.
// Note: removes children recursively.
func (bc *BlockChain) rmBlockData(hash common.Hash, slot *uint64) {
	if hash == (common.Hash{}) {
		return
	}
	//collect related data
	if slot == nil {
		hdr := rawdb.ReadHeader(bc.db, hash)
		if hdr != nil {
			slot = &hdr.Slot
		}
	}

	children := rawdb.ReadChildren(bc.db, hash)
	nr := rawdb.ReadFinalizedNumberByHash(bc.db, hash)

	// rm related data:
	for _, ch := range children {
		if ch == (common.Hash{}) {
			continue
		}
		//recursive cal
		bc.rmBlockData(ch, nil)
	}
	if nr != nil {
		rawdb.DeleteFinalizedHashNumber(bc.db, hash, *nr)
	}
	if slot != nil {
		rawdb.DeleteSlotBlockHash(bc.db, *slot, hash)
	}
	rawdb.DeleteBlockWithoutNumber(bc.db, hash)

	// clear bc caches
	bc.bodyCache.Remove(hash)
	bc.bodyRLPCache.Remove(hash)
	bc.receiptsCache.Remove(hash)
	bc.blockCache.Remove(hash)
	bc.invalidBlocksCache.Remove(hash)
	if slot != nil {
		bc.optimisticSpinesCache.Remove(*slot)
	}
	bc.checkpointCache.Remove(hash)

	insBlockCache := make([]*types.Block, 0, len(bc.insBlockCache))
	for _, b := range bc.insBlockCache {
		if b != nil && b.Hash() != hash {
			insBlockCache = append(insBlockCache, b)
		}
	}
	bc.insBlockCache = insBlockCache

	// clear hc caches
	bc.hc.headerCache.Remove(hash)
	bc.hc.numberCache.Remove(hash)
	bc.hc.ancestorCache.Remove(hash)
	bc.hc.blockDagCache.Remove(hash)
}

// RollbackFinalization
// 1. set the passed block as the last finalized,
// 2. removes finalisation data from last finalized up to lfNr
// 3. move all unfinalized blocks to dag and provide consistented tips
func (bc *BlockChain) RollbackFinalization(spineHash common.Hash, lfNr uint64) error {
	newLfBlock := bc.GetBlock(spineHash)
	if newLfBlock == nil {
		log.Error("Rollback finalization: block not found", "spineHash", fmt.Sprintf("%#x", spineHash), "lfNr", lfNr)
		return ErrBlockNotFound
	}

	lastFinBlock := bc.GetLastFinalizedHeader()
	if newLfBlock.Hash() == lastFinBlock.Hash() && newLfBlock.Nr() >= lfNr {
		return nil
	}

	bc.SetRollbackActive()
	defer bc.ResetRollbackActive()

	//reorg finalized and dag chains in accordance with spineHash
	for i := lfNr; i > newLfBlock.Nr(); i-- {
		blockHeader := bc.GetHeaderByNumber(i)
		if blockHeader == nil {
			log.Warn("Rollback finalization: rollback block not found", "finNr", i)
			continue
		}
		//check blockDag record exists
		if bc.GetBlockDag(blockHeader.Hash()) == nil {
			_, ancestors, _, _ := bc.CollectAncestorsAftCpByParents(blockHeader.ParentHashes, blockHeader.CpHash)
			cpHeader := bc.GetHeader(blockHeader.CpHash)
			bc.SaveBlockDag(&types.BlockDAG{
				Hash:                   blockHeader.Hash(),
				Height:                 blockHeader.Height,
				Slot:                   blockHeader.Slot,
				CpHash:                 blockHeader.CpHash,
				CpHeight:               cpHeader.Height,
				OrderedAncestorsHashes: ancestors.Hashes(),
			})
		}
		err := bc.rollbackBlockFinalization(i)
		if err != nil {
			log.Error("Rollback finalization: rollback block finalization error", "finNr", i, "hash", blockHeader.Hash().Hex(), "err", err)
		}
	}
	// update head of finalized chain
	if err := bc.WriteFinalizedBlock(newLfBlock.Nr(), newLfBlock, true); err != nil {
		return err
	}
	// update cp
	cp := bc.searchBlockFinalizationCp(newLfBlock.Header())
	if cp == nil {
		log.Error("Rollback finalization: cp not found", "lfSlot", newLfBlock.Slot(), "lfHash", newLfBlock.Hash().Hex())
		return fmt.Errorf("cp not found")
	}
	if lastCp := bc.GetLastCoordinatedCheckpoint(); lastCp == nil || lastCp.FinEpoch != cp.FinEpoch {
		bc.SetLastCoordinatedCheckpoint(cp)
	}

	return nil
}

// FastSyncCommitHead sets the current head block to the one defined by the hash
// irrelevant what the chain contents were prior.
func (bc *BlockChain) FastSyncCommitHead(hash common.Hash) error {
	// Make sure that both the block as well at its state trie exists
	block := bc.GetBlockByHash(hash)
	if block == nil {
		return fmt.Errorf("non existent block [%x..]", hash[:4])
	}
	if _, err := trie.NewSecure(block.Root(), bc.stateCache.TrieDB()); err != nil {
		return err
	}
	// If all checks out, manually set the head block.
	if !bc.chainmu.TryLock() {
		return errChainStopped
	}
	lastFinBlock := bc.GetBlockFinalizedNumber(block.Hash())
	block.SetNumber(lastFinBlock)
	bc.lastFinalizedBlock.Store(block)
	headBlockGauge.Update(int64(*lastFinBlock))
	bc.chainmu.Unlock()

	// Destroy any existing state snapshot and regenerate it in the background,
	// also resuming the normal maintenance of any previously paused snapshot.
	if bc.snaps != nil {
		bc.snaps.Rebuild(block.Root())
	}
	log.Info("Committed new head block", "number", block.Nr(), "hash", hash.Hex())
	return nil
}

// Export writes the active chain to the given writer.
func (bc *BlockChain) Export(w io.Writer) error {
	return bc.ExportN(w, uint64(0), bc.GetLastFinalizedBlock().Nr())
}

// ExportN writes a subset of the active chain to the given writer.
func (bc *BlockChain) ExportN(w io.Writer, first uint64, last uint64) error {
	if !bc.chainmu.TryLock() {
		return errChainStopped
	}
	defer bc.chainmu.Unlock()

	if first > last {
		return fmt.Errorf("export failed: first (%d) is greater than last (%d)", first, last)
	}
	log.Info("Exporting batch of blocks", "count", last-first+1)

	start, reported := time.Now(), time.Now()
	for nr := first; nr <= last; nr++ {
		block := bc.GetBlockByNumber(nr)
		if block == nil {
			return fmt.Errorf("export failed on #%d: not found", nr)
		}
		if err := block.EncodeRLP(w); err != nil {
			return err
		}
		if time.Since(reported) >= statsReportLimit {
			log.Info("Exporting blocks", "exported", block.Hash().Hex(), "elapsed", common.PrettyDuration(time.Since(start)))
			reported = time.Now()
		}
	}
	return nil
}

// writeFinalizedBlock injects a new finalized block into the current block chain.
// Note, this function assumes that the `mu` mutex is held!
func (bc *BlockChain) writeFinalizedBlock(finNr uint64, block *types.Block, isHead bool) error {
	if finNr == 0 && block.Hash() != bc.genesisBlock.Hash() {
		log.Error("Save genesis hash", "hash", block.Hash(), "fn", "writeFinalizedBlock")
		return fmt.Errorf("received zero finalizing number")
	}

	block.SetNumber(&finNr)
	batch := bc.db.NewBatch()
	rawdb.WriteFinalizedHashNumber(batch, block.Hash(), finNr)
	if val, ok := bc.hc.numberCache.Get(block.Hash()); ok {
		log.Warn("????? Cached Nr for Dag Block", "val", val.(uint64), "hash", block.Hash().Hex())
	}

	// update finalized number cache
	bc.hc.numberCache.Remove(block.Hash())
	bc.hc.numberCache.Add(block.Hash(), finNr)

	bc.hc.headerCache.Remove(block.Hash())
	bc.hc.headerCache.Add(block.Hash(), block.Header())

	bc.blockCache.Remove(block.Hash())
	bc.blockCache.Add(block.Hash(), block)

	// If the block is better than our head or is on a different chain, force update heads
	if isHead {
		rawdb.WriteLastFinalizedHash(batch, block.Hash())
		rawdb.WriteHeadFastBlockHash(batch, block.Hash())
	}
	// Flush the whole batch into the disk, exit the node if failed
	if err := batch.Write(); err != nil {
		log.Crit("Failed to update chain indexes and markers", "err", err)
	}
	// Update all in-memory chain markers in the last step
	if isHead {
		bc.hc.SetLastFinalisedHeader(block.Header(), finNr)
		bc.lastFinalizedFastBlock.Store(block)
		bc.lastFinalizedBlock.Store(block)
		headFastBlockGauge.Update(int64(finNr))
		bc.lastFinalizedFastBlock.Store(block)
		headFastBlockGauge.Update(int64(block.Nr()))
		headBlockGauge.Update(int64(finNr))

		bc.chainHeadFeed.Send(ChainHeadEvent{Block: block, Type: ET_NETWORK})
	}

	bc.RemoveTxsFromPool(block.Transactions())

	return nil
}

// Stop stops the blockchain service. If any imports are currently in progress
// it will abort them using the procInterrupt.
func (bc *BlockChain) Stop() {
	if !atomic.CompareAndSwapInt32(&bc.running, 0, 1) {
		return
	}

	// Unsubscribe all subscriptions registered from blockchain.
	bc.scope.Close()

	// Signal shutdown to all goroutines.
	close(bc.quit)
	bc.StopInsert()

	// Now wait for all chain modifications to end and persistent goroutines to exit.
	//
	// Note: Close waits for the mutex to become available, i.e. any running chain
	// modification will have exited when Close returns. Since we also called StopInsert,
	// the mutex should become available quickly. It cannot be taken again after Close has
	// returned.
	bc.chainmu.Close()
	bc.wg.Wait()

	// Ensure that the entirety of the state snapshot is journalled to disk.
	var snapBase common.Hash
	if bc.snaps != nil {
		var err error
		//curBlock := bc.GetLastFinalizedBlock()
		cp := bc.GetLastCoordinatedCheckpoint()
		curBlock := bc.GetHeader(cp.Spine)
		if snapBase, err = bc.snaps.Journal(curBlock.Root); err != nil {
			log.Error("Failed to journal state snapshot", "err", err)
		}
		rawdb.WriteSnapshotRecoveryNumber(bc.db, curBlock.Nr())
	}

	// Ensure the state of a recent block is also stored to disk before exiting.
	// We're writing three different states to catch different restart scenarios:
	//  - HEAD:     So we don't need to reprocess any blocks in the general case
	//  - HEAD-1:   So we don't do large reorgs if our HEAD becomes an uncle
	//  - HEAD-127: So we have a hard limit on the number of blocks reexecuted
	if !bc.cacheConfig.TrieDirtyDisabled {
		triedb := bc.stateCache.TrieDB()

		for _, offset := range []uint64{0, 1, TriesInMemory - 1} {
			if number := *bc.GetBlockFinalizedNumber(bc.GetLastFinalizedBlock().Hash()); number > offset {
				recentNr := number - offset
				recent := bc.GetBlockByNumber(number - offset)

				log.Info("Writing cached state to disk", "block", recentNr, "hash", recent.Hash(), "root", recent.Root())
				if err := triedb.Commit(recent.Root(), true, nil); err != nil {
					log.Error("Failed to commit recent state trie", "err", err)
				}
			}
		}
		if snapBase != (common.Hash{}) {
			log.Info("Writing snapshot state to disk", "root", snapBase)
			if err := triedb.Commit(snapBase, true, nil); err != nil {
				log.Error("Failed to commit recent state trie", "err", err)
			}
		}
		for !bc.triegc.Empty() {
			triedb.Dereference(bc.triegc.PopItem().(common.Hash))
		}
		if size, _ := triedb.Size(); size != 0 {
			log.Error("Dangling trie nodes after full cleanup")
		}
	}
	// Ensure all live cached entries be saved into disk, so that we can skip
	// cache warmup when node restarts.
	if bc.cacheConfig.TrieCleanJournal != "" {
		triedb := bc.stateCache.TrieDB()
		triedb.SaveCache(bc.cacheConfig.TrieCleanJournal)
	}
	log.Info("Blockchain stopped")
}

// StopInsert interrupts all insertion methods, causing them to return
// errInsertionInterrupted as soon as possible. Insertion is permanently disabled after
// calling this method.
func (bc *BlockChain) StopInsert() {
	atomic.StoreInt32(&bc.procInterrupt, 1)
}

// insertStopped returns true after StopInsert has been called.
func (bc *BlockChain) insertStopped() bool {
	return atomic.LoadInt32(&bc.procInterrupt) == 1
}

// WriteStatus status of write
type WriteStatus byte

const (
	NonStatTy WriteStatus = iota
	CanonStatTy
	SideStatTy
)

// numberHash is just a container for a number and a hash, to represent a block
type numberHash struct {
	number uint64
	hash   common.Hash
}

// InsertReceiptChain attempts to complete an already existing header chain with
// transaction and receipt data.
func (bc *BlockChain) InsertReceiptChain(blockChain types.Blocks, receiptChain []types.Receipts, ancientLimit uint64) (int, error) {
	// We don't require the chainMu here since we want to maximize the
	// concurrency of header insertion and receipt insertion.
	bc.wg.Add(1)
	defer bc.wg.Done()

	var (
		ancientBlocks, liveBlocks     types.Blocks
		ancientReceipts, liveReceipts []types.Receipts
	)
	// Do a sanity check that the provided chain is actually ordered and linked
	for i := 0; i < len(blockChain); i++ {
		if blockChain[i].Number() != nil && blockChain[i].Nr() <= ancientLimit {
			ancientBlocks, ancientReceipts = append(ancientBlocks, blockChain[i]), append(ancientReceipts, receiptChain[i])
		} else {
			liveBlocks, liveReceipts = append(liveBlocks, blockChain[i]), append(liveReceipts, receiptChain[i])
		}
	}

	var (
		stats = struct{ processed, ignored int32 }{}
		start = time.Now()
		size  = int64(0)
	)

	// updateHead updates the head fast sync block if the inserted blocks are better
	// and returns an indicator whether the inserted blocks are canonical.
	updateHead := func(head *types.Block) bool {
		if !bc.chainmu.TryLock() {
			return false
		}
		defer bc.chainmu.Unlock()

		// Rewind may have occurred, skip in that case.
		if bc.GetLastFinalizedHeader().Nr() >= head.Nr() {
			rawdb.WriteHeadFastBlockHash(bc.db, head.Hash())
			bc.lastFinalizedFastBlock.Store(head)
			headFastBlockGauge.Update(int64(head.Nr()))
			return true
		}
		return false
	}

	// writeAncient writes blockchain and corresponding receipt chain into ancient store.
	//
	// this function only accepts canonical chain data. All side chain will be reverted
	// eventually.
	writeAncient := func(blockChain types.Blocks, receiptChain []types.Receipts) (int, error) {
		first := blockChain[0]
		last := blockChain[len(blockChain)-1]

		// Ensure genesis is in ancients.
		if first.Nr() == 1 {
			if frozen, _ := bc.db.Ancients(); frozen == 0 {
				b := bc.genesisBlock
				writeSize, err := rawdb.WriteAncientBlocks(bc.db, []*types.Block{b}, []types.Receipts{nil})
				size += writeSize
				if err != nil {
					log.Error("Error writing genesis to ancients", "err", err)
					return 0, err
				}
				log.Info("Wrote genesis to ancients")
			}
		}
		// Before writing the blocks to the ancients, we need to ensure that
		// they correspond to the what the headerchain 'expects'.
		// We only check the last block/header, since it's a contiguous chain.
		if !bc.HasHeader(last.Hash()) {
			return 0, fmt.Errorf("containing header #%d [%x..] unknown", last.Nr(), last.Hash().Bytes()[:4])
		}

		// Write all chain data to ancients.
		writeSize, err := rawdb.WriteAncientBlocks(bc.db, blockChain, receiptChain)
		size += writeSize
		if err != nil {
			log.Error("Error importing chain data to ancients", "err", err)
			return 0, err
		}

		// Write tx indices if any condition is satisfied:
		// * If user requires to reserve all tx indices(txlookuplimit=0)
		// * If all ancient tx indices are required to be reserved(txlookuplimit is even higher than ancientlimit)
		// * If block number is large enough to be regarded as a recent block
		// It means blocks below the ancientLimit-txlookupLimit won't be indexed.
		//
		// But if the `TxIndexTail` is not nil, e.g. Geth is initialized with
		// an external ancient database, during the setup, blockchain will start
		// a background routine to re-indexed all indices in [ancients - txlookupLimit, ancients)
		// range. In this case, all tx indices of newly imported blocks should be
		// generated.
		var batch = bc.db.NewBatch()
		for bi, block := range blockChain {
			if bc.txLookupLimit == 0 || ancientLimit <= bc.txLookupLimit || block.Nr() >= ancientLimit-bc.txLookupLimit {
				for i, tx := range block.Transactions() {
					bc.WriteTxLookupEntry(i, tx.Hash(), block.Hash(), receiptChain[bi][i].Status)
				}
			} else if rawdb.ReadTxIndexTail(bc.db) != nil {
				for i, tx := range block.Transactions() {
					bc.WriteTxLookupEntry(i, tx.Hash(), block.Hash(), receiptChain[bi][i].Status)
				}
			}
			stats.processed++
		}

		// Flush all tx-lookup index data.
		size += int64(batch.ValueSize())
		if err := batch.Write(); err != nil {
			// The tx index data could not be written.
			// Roll back the ancient store update.
			fastBlock := bc.GetLastFinalizedFastBlock().Nr()
			if err := bc.db.TruncateAncients(fastBlock + 1); err != nil {
				log.Error("Can't truncate ancient store after failed insert", "err", err)
			}
			return 0, err
		}

		// Sync the ancient store explicitly to ensure all data has been flushed to disk.
		if err := bc.db.Sync(); err != nil {
			return 0, err
		}

		// Update the current fast block because all block data is now present in DB.
		previousFastBlock := bc.GetLastFinalizedFastBlock().Nr()
		if !updateHead(blockChain[len(blockChain)-1]) {
			// We end up here if the header chain has reorg'ed, and the blocks/receipts
			// don't match the canonical chain.
			if err := bc.db.TruncateAncients(previousFastBlock + 1); err != nil {
				log.Error("Can't truncate ancient store after failed insert", "err", err)
			}
			return 0, errSideChainReceipts
		}

		// Delete block data from the main database.
		batch.Reset()
		canonHashes := make(map[common.Hash]struct{})
		for _, block := range blockChain {
			canonHashes[block.Hash()] = struct{}{}
			if block.Nr() == 0 {
				continue
			}
			rawdb.DeleteBlockWithoutNumber(batch, block.Hash())
		}
		if err := batch.Write(); err != nil {
			return 0, err
		}

		return 0, nil
	}

	// writeLive writes blockchain and corresponding receipt chain into active store.
	writeLive := func(blockChain types.Blocks, receiptChain []types.Receipts) (int, error) {
		skipPresenceCheck := false
		batch := bc.db.NewBatch()
		for i, block := range blockChain {
			// Short circuit insertion if shutting down or processing failed
			if bc.insertStopped() {
				return 0, errInsertionInterrupted
			}
			// Short circuit if the owner header is unknown
			if !bc.HasHeader(block.Hash()) {
				return i, fmt.Errorf("containing header #%d [%x..] unknown", block.Nr(), block.Hash().Bytes()[:4])
			}
			if !skipPresenceCheck {
				// Ignore if the entire data is already known
				if bc.HasBlock(block.Hash()) {
					stats.ignored++
					continue
				} else {
					// If block N is not present, neither are the later blocks.
					// This should be true, but if we are mistaken, the shortcut
					// here will only cause overwriting of some existing data
					skipPresenceCheck = true
				}
			}
			// Write all the data out into the database
			rawdb.WriteBody(batch, block.Hash(), block.Body())
			rawdb.WriteReceipts(batch, block.Hash(), receiptChain[i])
			bc.handleBlockValidatorSyncReceipts(block, receiptChain[i])

			// Always write tx indices for live blocks, we assume they are needed
			for j, tx := range block.Transactions() {
				bc.WriteTxLookupEntry(j, tx.Hash(), block.Hash(), receiptChain[i][j].Status)
			}

			// Write everything belongs to the blocks into the database. So that
			// we can ensure all components of body is completed(body, receipts,
			// tx indexes)
			if batch.ValueSize() >= ethdb.IdealBatchSize {
				if err := batch.Write(); err != nil {
					return 0, err
				}
				size += int64(batch.ValueSize())
				batch.Reset()
			}
			stats.processed++
		}
		// Write everything belongs to the blocks into the database. So that
		// we can ensure all components of body is completed(body, receipts,
		// tx indexes)
		if batch.ValueSize() > 0 {
			size += int64(batch.ValueSize())
			if err := batch.Write(); err != nil {
				return 0, err
			}
		}
		updateHead(blockChain[len(blockChain)-1])
		return 0, nil
	}

	// Write downloaded chain data and corresponding receipt chain data
	if len(ancientBlocks) > 0 {
		if n, err := writeAncient(ancientBlocks, ancientReceipts); err != nil {
			if err == errInsertionInterrupted {
				return 0, nil
			}
			return n, err
		}
	}
	// Write the tx index tail (block number from where we index) before write any live blocks
	if len(liveBlocks) > 0 && liveBlocks[0].Nr() == ancientLimit+1 {
		// The tx index tail can only be one of the following two options:
		// * 0: all ancient blocks have been indexed
		// * ancient-limit: the indices of blocks before ancient-limit are ignored
		if tail := rawdb.ReadTxIndexTail(bc.db); tail == nil {
			if bc.txLookupLimit == 0 || ancientLimit <= bc.txLookupLimit {
				rawdb.WriteTxIndexTail(bc.db, 0)
			} else {
				rawdb.WriteTxIndexTail(bc.db, ancientLimit-bc.txLookupLimit)
			}
		}
	}
	if len(liveBlocks) > 0 {
		if n, err := writeLive(liveBlocks, liveReceipts); err != nil {
			if err == errInsertionInterrupted {
				return 0, nil
			}
			return n, err
		}
	}

	head := blockChain[len(blockChain)-1]
	context := []interface{}{
		"count", stats.processed, "elapsed", common.PrettyDuration(time.Since(start)),
		"number", head.Nr(), "hash", head.Hash(), "age", common.PrettyAge(time.Unix(int64(head.Time()), 0)),
		"size", common.StorageSize(size),
	}
	if stats.ignored > 0 {
		context = append(context, []interface{}{"ignored", stats.ignored}...)
	}
	log.Info("Imported new block receipts", context...)

	return 0, nil
}

// writeBlockWithoutState writes only the block and its metadata to the database,
// but does not write any state. This is used to construct competing side forks
// up to the point where they exceed the canonical total difficulty.
func (bc *BlockChain) writeBlockWithoutState(block *types.Block) (err error) {
	if bc.insertStopped() {
		return errInsertionInterrupted
	}

	batch := bc.db.NewBatch()
	rawdb.WriteBlock(batch, block)
	if err := batch.Write(); err != nil {
		log.Crit("Failed to write block into disk", "err", err)
	}
	rawdb.AddSlotBlockHash(bc.Database(), block.Slot(), block.Hash())

	bc.AppendToChildren(block.Hash(), block.ParentHashes())
	return nil
}

// WriteFinalizedBlock writes the block finalization info to the database.
func (bc *BlockChain) WriteFinalizedBlock(finNr uint64, block *types.Block, isHead bool) error {
	if !bc.chainmu.TryLock() {
		return errInsertionInterrupted
	}
	defer bc.chainmu.Unlock()

	return bc.writeFinalizedBlock(finNr, block, isHead)
}

// SetRollbackActive set flag of rollback proc is running.
func (bc *BlockChain) SetRollbackActive() {
	bc.hc.SetRollbackActive()
}

// ResetRollbackActive reset flag of rollback proc running.
func (bc *BlockChain) ResetRollbackActive() {
	bc.hc.ResetRollbackActive()
}

// IsRollbackActive returns true if rollback proc is running.
func (bc *BlockChain) IsRollbackActive() bool {
	return bc.hc.IsRollbackActive()
}

// rollbackBlockFinalization reset block's finalization data only.
func (bc *BlockChain) rollbackBlockFinalization(finNr uint64) error {
	if !bc.chainmu.TryLock() {
		return errInsertionInterrupted
	}
	defer bc.chainmu.Unlock()

	block := bc.GetBlockByNumber(finNr)
	if block == nil {
		return errBlockNotFound
	}
	hash := block.Hash()
	block.SetNumber(nil)

	batch := bc.db.NewBatch()
	rawdb.DeleteFinalizedHashNumber(batch, hash, finNr)

	// update finalized number cache
	bc.hc.numberCache.Remove(hash)
	bc.receiptsCache.Remove(hash)

	bc.hc.headerCache.Remove(hash)
	bc.hc.headerCache.Add(hash, block.Header())

	bc.blockCache.Remove(hash)
	bc.blockCache.Add(hash, block)

	// Flush the whole batch into the disk, exit the node if failed
	if err := batch.Write(); err != nil {
		log.Error("Failed to rollback block finalization", "finNr", finNr, "hash", hash.Hex(), "err", err)
		return err
	}
	return nil
}

// WriteSyncBlocks writes the blocks and all associated state to the database while synchronization process.
func (bc *BlockChain) WriteSyncBlocks(blocks types.Blocks, validate bool) (failed *types.Block, err error) {
	bc.blockProcFeed.Send(true)
	defer bc.blockProcFeed.Send(false)

	// Pre-checks passed, start the full block imports
	if !bc.chainmu.TryLock() {
		return nil, errInsertionInterrupted
	}

	// include delayed blocks
	if len(bc.insBlockCache) > 0 {
		blocks = append(blocks, bc.insBlockCache...)
		bc.insBlockCache = []*types.Block{}
	}
	blocks = blocks.Deduplicate(true)

	// rm existed blocks
	notExisted := make(types.Blocks, 0, len(blocks))
	for _, bl := range blocks {
		if hdr := bc.GetHeader(bl.Hash()); hdr != nil {
			log.Info("Insert delayed blocks: skip inserted", "slot", bl.Slot(), "hash", bl.Hash().Hex())
			continue
		}
		notExisted = append(notExisted, bl)
	}

	// ordering by slot sequence to insert
	blocksBySlot, err := notExisted.GroupBySlot()
	if err != nil {
		bc.insBlockCache = notExisted
		return nil, err
	}
	//sort by slots
	slots := common.SorterAscU64{}
	for sl, _ := range blocksBySlot {
		slots = append(slots, sl)
	}
	sort.Sort(slots)

	orderedBlocks := make([]*types.Block, 0, len(notExisted))
	for _, slot := range slots {
		slotBlocks := blocksBySlot[slot]
		if len(slotBlocks) == 0 {
			continue
		}
		orderedBlocks = append(orderedBlocks, slotBlocks...)
	}

	// insert process
	n, err := bc.insertBlocks(orderedBlocks, validate, opSync)
	bc.chainmu.Unlock()
	if err == ErrInsertUncompletedDag {
		processing := make(map[common.Hash]bool, len(bc.insBlockCache))
		for _, b := range bc.insBlockCache {
			processing[b.Hash()] = true
		}
		for i, bl := range orderedBlocks {
			log.Info("Delay syncing block", "height", bl.Height(), "hash", bl.Hash().Hex())
			if i >= n && !processing[bl.Hash()] {
				bc.insBlockCache = append(bc.insBlockCache, bl)
				processing[bl.Hash()] = true
			}
		}
	} else if err != nil {
		return orderedBlocks[n], err
	}
	return nil, nil
}

// WriteCreatedDagBlock writes the dag block created locally.
func (bc *BlockChain) WriteCreatedDagBlock(block *types.Block) (status int, err error) {
	bc.blockProcFeed.Send(true)
	defer bc.blockProcFeed.Send(false)

	// Pre-checks passed, start the full block imports
	if !bc.chainmu.TryLock() {
		return 0, errInsertionInterrupted
	}
	defer bc.chainmu.Unlock()
	n, err := bc.insertBlocks(types.Blocks{block}, true, opCreate)

	return n, err
}

// WriteMinedBlock writes the block and all associated state to the database.
// deprecated
func (bc *BlockChain) WriteMinedBlock(block *types.Block) (status WriteStatus, err error) {
	if !bc.chainmu.TryLock() {
		return NonStatTy, errInsertionInterrupted
	}
	defer bc.chainmu.Unlock()

	// WriteBlockWithoutState
	status = SideStatTy
	err = bc.writeBlockWithoutState(block)
	if err != nil {
		return NonStatTy, err
	}

	return
}

// writeBlockWithState writes the block and all associated state to the database,
// but is expects the chain mutex to be held.
func (bc *BlockChain) writeBlockWithState(block *types.Block, receipts []*types.Receipt, logs []*types.Log, state *state.StateDB, emitHeadEvent NewBlockEvtType, kind string) (status WriteStatus, err error) {
	if bc.insertStopped() {
		return NonStatTy, errInsertionInterrupted
	}

	// Irrelevant of the canonical status, write the block itself to the database.
	//
	// Note all the components of block(td, hash->number map, header, body, receipts)
	// should be written atomically. BlockBatch is used for containing all components.
	blockBatch := bc.db.NewBatch()
	rawdb.WriteBlock(blockBatch, block)
	rawdb.WriteReceipts(blockBatch, block.Hash(), receipts)
	bc.handleBlockValidatorSyncReceipts(block, receipts)
	rawdb.WritePreimages(blockBatch, state.Preimages())

	// create transaction lookup for applied txs.
	for i, tx := range block.Transactions() {
		bc.WriteTxLookupEntry(i, tx.Hash(), block.Hash(), receipts[i].Status)
	}

	if err := blockBatch.Write(); err != nil {
		log.Crit("Failed to write block into disk", "err", err)
	}
	rawdb.AddSlotBlockHash(bc.Database(), block.Slot(), block.Hash())
	bc.removeOptimisticSpinesFromCache(block.Slot())
	bc.AppendToChildren(block.Hash(), block.ParentHashes())
	// Commit all cached state changes into underlying memory database.
	root, err := state.Commit(true)
	log.Info("Block parent hashes", "hash", block.Hash().Hex(), "ParentHashes", block.ParentHashes())
	log.Info("Block received root", "root", block.Root().Hex(), "hash", block.Hash().Hex())
	log.Info("Block committed root", "root", root.Hex(), "height", block.Height(), "Nr", block.Nr(), "kind", kind)

	if err != nil {
		log.Error("Block committed root error", "height", block.Height(), "Nr", block.Nr(), "kind", kind, "err", err)
		return NonStatTy, err
	}
	triedb := bc.stateCache.TrieDB()

	// If we're running an archive node, always flush
	if bc.cacheConfig.TrieDirtyDisabled {
		if err := triedb.Commit(root, false, nil); err != nil {
			return NonStatTy, err
		}
	} else {
		// Full but not archive node, do proper garbage collection
		triedb.Reference(root, common.Hash{}) // metadata reference to keep trie alive
		if err := triedb.Commit(root, true, nil); err != nil {
			return NonStatTy, err
		}
	}
	status = SideStatTy

	if status == CanonStatTy || kind == "syncInsertChain" {
		bc.chainFeed.Send(ChainEvent{Block: block, Hash: block.Hash(), Logs: logs})
		if len(logs) > 0 {
			bc.logsFeed.Send(logs)
		}
		// In theory we should fire a ChainHeadEvent when we inject
		// a canonical block, but sometimes we can insert a batch of
		// canonicial blocks. Avoid firing too much ChainHeadEvents,
		// we will fire an accumulated ChainHeadEvent and disable fire
		// event here.
		if emitHeadEvent != ET_SKIP {
			bc.chainHeadFeed.Send(ChainHeadEvent{Block: block, Type: emitHeadEvent})
		}
	}

	return status, nil
}

// deprecated, used for tests only
// SyncInsertChain attempts to insert the given batch of blocks in chain
// received while synchronization process
func (bc *BlockChain) SyncInsertChain(chain types.Blocks) (int, error) {
	// Sanity check that we have something meaningful to import
	if len(chain) == 0 {
		return 0, nil
	}

	bc.blockProcFeed.Send(true)
	defer bc.blockProcFeed.Send(false)

	var (
		block, prev *types.Block
	)
	// Do a sanity check that the provided chain is actually ordered and linked
	for i := 1; i < len(chain); i++ {
		block = chain[i]
		prev = chain[i-1]
		curNr := uint64(0)
		prevNr := uint64(0)
		if block.Number() != nil {
			curNr = *block.Number()
		}
		if prev.Number() != nil {
			prevNr = *prev.Number()
		}
		if curNr != prevNr+1 {
			// Chain broke ancestry, log a message (programming error) and skip insertion
			log.Error("Non contiguous block insert", "number", block.Nr(), "hash", block.Hash().Hex(), "prevnumber", prev.Nr(), "prevhash", prev.Hash())
			return 0, fmt.Errorf("non contiguous insert: item %d is #%d [%x..], item %d is #%d [%x..]", i-1, prev.Nr(),
				prev.Hash().Bytes()[:4], i, block.Nr(), block.Hash().Bytes()[:4])
		}
	}

	// Pre-checks passed, start the full block imports
	if !bc.chainmu.TryLock() {
		return 0, errChainStopped
	}
	defer bc.chainmu.Unlock()
	return bc.syncInsertChain(chain)
}

// InsertPropagatedBlocks inserts propagated block
func (bc *BlockChain) InsertPropagatedBlocks(chain types.Blocks) (int, error) {
	// Sanity check that we have something meaningful to import
	if len(chain) == 0 {
		return 0, nil
	}

	bc.DagMuLock()
	defer bc.DagMuUnlock()

	bc.blockProcFeed.Send(true)
	defer bc.blockProcFeed.Send(false)

	// Pre-checks passed, start the full block imports
	if !bc.chainmu.TryLock() {
		return 0, errChainStopped
	}
	n, err := bc.insertBlocks(chain, true, opPropagate)
	bc.chainmu.Unlock()

	if err == ErrInsertUncompletedDag {
		processing := make(map[common.Hash]bool, len(bc.insBlockCache))
		for _, b := range bc.insBlockCache {
			processing[b.Hash()] = true
		}
		for i, bl := range chain {
			log.Info("Delay propagated block", "height", bl.Height(), "hash", bl.Hash().Hex())
			if i >= n && !processing[bl.Hash()] {
				bc.insBlockCache = append(bc.insBlockCache, bl)
				processing[bl.Hash()] = true
			}
		}
	}
	return n, err
}

// IsAddressAssigned  checks if miner is allowed to add transaction from that address
func IsAddressAssigned(address common.Address, creators []common.Address, creatorNr int64) bool {
	var (
		creatorCount = len(creators)
		countVal     = big.NewInt(int64(creatorCount))
		val          = address.Hash().Big()
	)
	if creatorCount == 0 {
		return false
	}

	pos := new(big.Int).Mod(val, countVal).Int64()
	return pos == creatorNr
}

// Deprecated, used for tests only
// syncInsertChain is the internal implementation of SyncInsertChain, which assumes that
// 1) chains are contiguous, and 2) The chain mutex is held.
//
// This method is split out so that import batches that require re-injecting
// historical blocks can do so without releasing the lock, which could lead to
// racey behaviour. If a sidechain import is in progress, and the historic state
// is imported, but then new canon-head is added before the actual sidechain
// completes, then the historic state could be pruned again
func (bc *BlockChain) syncInsertChain(chain types.Blocks) (int, error) {
	// If the chain is terminating, don't even bother starting up
	if atomic.LoadInt32(&bc.procInterrupt) == 1 {
		return 0, nil
	}
	// Start a parallel signature recovery (signer will fluke on fork transition, minimal perf loss)
	senderCacher.recoverFromBlocks(types.MakeSigner(bc.chainConfig), chain)
	var (
		stats    = insertStats{startTime: mclock.Now()}
		maxFinNr = bc.GetLastFinalizedNumber()
	)

	// Start the parallel header verifier
	headers := make([]*types.Header, len(chain))
	headerMap := make(types.HeaderMap, len(chain))

	for i, block := range chain {
		headers[i] = block.Header()
		headerMap[block.Hash()] = block.Header()
		if block.Number() != nil {
			if block.Nr() > maxFinNr {
				maxFinNr = block.Nr()
			}
		} else {
			bc.MoveTxsToProcessing(block)
		}
	}

	// Peek the error for the first block to decide the directing import logic
	it := newInsertIterator(chain, bc.validator)

	block, err := it.next()

	switch {
	// First block is pruned, insert as sidechain and reorg
	case errors.Is(err, consensus.ErrPrunedAncestor):
		log.Warn("Pruned ancestor, inserting as sidechain", "hash", block.Hash().Hex())
		return bc.insertSideChain(block, it)

	// Some other error occurred, abort
	case err != nil:
		stats.ignored += len(it.chain)
		return it.index, err
	}
	// No validation errors for the first block (or chain prefix skipped)
	var activeState *state.StateDB
	defer func() {
		// The chain importer is starting and stopping trie prefetchers. If a bad
		// block or other error is hit however, an early return may not properly
		// terminate the background threads. This defer ensures that we clean up
		// and dangling prefetcher, without defering each and holding on live refs.
		if activeState != nil {
			activeState.StopPrefetcher()
		}
	}()

	for ; block != nil && err == nil; block, err = it.next() {
		// If the chain is terminating, stop processing blocks
		if bc.insertStopped() {
			log.Debug("Abort during block processing")
			break
		}

		rawdb.WriteBlock(bc.db, block)
		rawdb.AddSlotBlockHash(bc.Database(), block.Slot(), block.Hash())
		bc.AppendToChildren(block.Hash(), block.ParentHashes())
		bc.MoveTxsToProcessing(block)

		isHead := maxFinNr == block.Nr()
		bc.writeFinalizedBlock(block.Nr(), block, isHead)
		if err != nil {
			return it.index, err
		}

		//insertion of blue blocks
		start := time.Now()
		//retrieve state data
		statedb, stateBlock, stateErr := bc.CollectStateDataByBlock(block)

		if stateErr != nil {
			return it.index, stateErr
		}
		// Enable prefetching to pull in trie node paths while processing transactions
		statedb.StartPrefetcher("chain")
		activeState = statedb

		// If we have a followup block, run that against the current state to pre-cache
		// transactions and probabilistically some of the account/storage trie nodes.
		var followupInterrupt uint32
		if !bc.cacheConfig.TrieCleanNoPrefetch {
			if followup, err := it.peek(); followup != nil && err == nil {
				throwaway, _ := state.New(stateBlock.Root(), bc.stateCache, bc.snaps)

				go func(start time.Time, followup *types.Block, throwaway *state.StateDB, interrupt *uint32) {
					bc.prefetcher.Prefetch(followup, throwaway, bc.vmConfig, &followupInterrupt)

					blockPrefetchExecuteTimer.Update(time.Since(start))
					if atomic.LoadUint32(interrupt) == 1 {
						blockPrefetchInterruptMeter.Mark(1)
					}
				}(time.Now(), followup, throwaway, &followupInterrupt)
			}
		}
		// Process block using the parent state as reference point
		substart := time.Now()
		receipts, logs, usedGas, err := bc.processor.Process(block, statedb, bc.vmConfig)
		if err != nil {
			atomic.StoreUint32(&followupInterrupt, 1)
			log.Error("Error of block insertion to chain while sync (processing)", "height", block.Height(), "hash", block.Hash().Hex(), "err", err)
			return it.index, err
		}
		// Update the metrics touched during block processing
		accountReadTimer.Update(statedb.AccountReads)                 // Account reads are complete, we can mark them
		storageReadTimer.Update(statedb.StorageReads)                 // Storage reads are complete, we can mark them
		accountUpdateTimer.Update(statedb.AccountUpdates)             // Account updates are complete, we can mark them
		storageUpdateTimer.Update(statedb.StorageUpdates)             // Storage updates are complete, we can mark them
		snapshotAccountReadTimer.Update(statedb.SnapshotAccountReads) // Account reads are complete, we can mark them
		snapshotStorageReadTimer.Update(statedb.SnapshotStorageReads) // Storage reads are complete, we can mark them
		triehash := statedb.AccountHashes + statedb.StorageHashes     // Save to not double count in validation
		trieproc := statedb.SnapshotAccountReads + statedb.AccountReads + statedb.AccountUpdates
		trieproc += statedb.SnapshotStorageReads + statedb.StorageReads + statedb.StorageUpdates

		blockExecutionTimer.Update(time.Since(substart) - trieproc - triehash)

		proctime := time.Since(start)

		// Update the metrics touched during block validation
		accountHashTimer.Update(statedb.AccountHashes) // Account hashes are complete, we can mark them
		storageHashTimer.Update(statedb.StorageHashes) // Storage hashes are complete, we can mark them

		blockValidationTimer.Update(time.Since(substart) - (statedb.AccountHashes + statedb.StorageHashes - triehash))

		// Write the block to the chain and get the status.
		substart = time.Now()
		log.Info("SyncInsert chain", "height", block.Height(), "hash", block.Hash().Hex(), "err", err)
		status, err := bc.writeBlockWithState(block, receipts, logs, statedb, ET_SKIP, "syncInsertChain")
		atomic.StoreUint32(&followupInterrupt, 1)
		if err != nil {
			log.Error("Error of block insertion to chain while sync (block writing)", "height", block.Height(), "hash", block.Hash().Hex(), "err", err)
			return it.index, err
		}
		// Update the metrics touched during block commit
		accountCommitTimer.Update(statedb.AccountCommits)   // Account commits are complete, we can mark them
		storageCommitTimer.Update(statedb.StorageCommits)   // Storage commits are complete, we can mark them
		snapshotCommitTimer.Update(statedb.SnapshotCommits) // Snapshot commits are complete, we can mark them

		blockWriteTimer.Update(time.Since(substart) - statedb.AccountCommits - statedb.StorageCommits - statedb.SnapshotCommits)
		blockInsertTimer.UpdateSince(start)

		switch status {
		case CanonStatTy:
			log.Error("Inserted new block", "hash", block.Hash().Hex(),
				"txs", len(block.Transactions()), "gas", block.GasUsed(),
				"elapsed", common.PrettyDuration(time.Since(start)),
				"root", block.Root())
			// Only count canonical blocks for GC processing time
			bc.gcproc += proctime

		case SideStatTy:
			log.Debug("Inserted forked block", "hash", block.Hash().Hex(),
				"elapsed", common.PrettyDuration(time.Since(start)),
				"txs", len(block.Transactions()), "gas", block.GasUsed(),
				"root", block.Root())

		default:
			// This in theory is impossible, but lets be nice to our future selves and leave
			// a log, instead of trying to track down blocks imports that don't emit logs.
			log.Warn("Inserted block with unknown status", "hash", block.Hash().Hex(),
				"elapsed", common.PrettyDuration(time.Since(start)),
				"txs", len(block.Transactions()), "gas", block.GasUsed(),
				"root", block.Root())
		}
		stats.processed++
		stats.usedGas += usedGas

		dirty, _ := bc.stateCache.TrieDB().Size()
		stats.report(chain, it.index, dirty)

		bc.AppendToChildren(block.Hash(), block.ParentHashes())

		// update tips
		tmpTips := types.Tips{}
		for _, h := range block.ParentHashes() {
			bdag := bc.GetBlockDag(h)
			// should never happen
			if bdag == nil {
				pHeader := bc.GetHeader(h)
				_, anc, _, err := bc.CollectAncestorsAftCpByParents(pHeader.ParentHashes, pHeader.CpHash)
				if err != nil {
					return it.index, err
				}
				bdag = &types.BlockDAG{
					Hash:                   pHeader.Hash(),
					Height:                 pHeader.Height,
					Slot:                   pHeader.Slot,
					CpHash:                 pHeader.CpHash,
					CpHeight:               bc.GetHeader(pHeader.CpHash).Height,
					OrderedAncestorsHashes: anc.Hashes(),
				}
			}
			tmpTips.Add(bdag)
		}
		ancestorsHashes, err := bc.CollectAncestorsHashesByTips(tmpTips, block.CpHash())
		bc.AddTips(&types.BlockDAG{
			Hash:                   block.Hash(),
			Height:                 block.Height(),
			Slot:                   block.Slot(),
			CpHash:                 block.CpHash(),
			CpHeight:               bc.GetHeader(block.CpHash()).Height,
			OrderedAncestorsHashes: ancestorsHashes,
		})
		bc.RemoveTips(ancestorsHashes)
	}

	stats.ignored += it.remaining()

	return it.index, err
}

// verifyBlockCoinbase return false if creator is unassigned
func (bc *BlockChain) verifyBlockCoinbase(block *types.Block, slotCreators []common.Address) bool {
	coinbase := block.Header().Coinbase
	contains, _ := common.Contains(slotCreators, coinbase)
	if !contains {
		log.Warn("Block verification: creator assignment failed",
			"slot", block.Slot(),
			"hash", block.Hash().Hex(),
			"blockCreator", block.Header().Coinbase.Hex(),
			"slotCreators", slotCreators,
		)

		return false
	}

	signer, err := types.BlockHeaderSigner(block.Header())
	if err != nil {
		return false
	}

	if signer != coinbase {
		log.Warn("Block verification: creator is not a signer",
			"slot", block.Slot(),
			"hash", block.Hash().Hex(),
			"blockCreator", block.Header().Coinbase.Hex(),
			"blockSigner", signer.Hex(),
			"slotCreators", slotCreators,
		)

		return false
	}

	return true
}

// CacheInvalidBlock cache invalid block
func (bc *BlockChain) CacheInvalidBlock(block *types.Block) {
	bc.invalidBlocksCache.Add(block.Hash(), struct{}{})
}

// VerifyBlock validate block
func (bc *BlockChain) VerifyBlock(block *types.Block) (bool, error) {
	defer func(ts time.Time) {
		log.Info("^^^^^^^^^^^^ TIME",
			"elapsed", common.PrettyDuration(time.Since(ts)),
			"func:", "VerifyBlock:Total",
		)
	}(time.Now())

	// Verify block slot
	if !bc.verifyBlockSlot(block) {
		return false, nil
	}

	// Verify block era
	if !bc.verifyBlockEra(block) {
		return false, nil
	}

	slotCreators, err := bc.ValidatorStorage().GetCreatorsBySlot(bc, block.Slot())
	if err != nil {
		log.Error("VerifyBlock: can`t get shuffled validators", "error", err)
		return false, err
	}

	// Verify block coinbase
	if !bc.verifyBlockCoinbase(block, slotCreators) {
		return false, nil
	}

	err = bc.verifyEmptyBlock(block, slotCreators)
	if err != nil {
		return false, err
	}

	// Verify baseFee
	if !bc.verifyBlockBaseFee(block) {
		return false, nil
	}

	// Verify body hash and transactions hash
	if !bc.verifyBlockHashes(block) {
		return false, nil
	}

	// Verify block used gas
	if !bc.verifyBlockUsedGas(block) {
		return false, nil
	}

	isCpAncestor, ancestors, unloaded, _ := bc.CollectAncestorsAftCpByTips(block.ParentHashes(), block.CpHash())

	//check is block's chain synced and does not content rejected blocks
	if len(unloaded) > 0 {
		for _, unh := range unloaded {
			if _, ok := bc.invalidBlocksCache.Get(unh); ok {
				log.Warn("Block verification: invalid parent", "hash", block.Hash().Hex(), "parent", unh.Hex())
				return false, nil
			}
			log.Warn("Block verification: unknown parent", "hash", block.Hash().Hex(), "parent", unh.Hex())
			continue
		}
		return false, ErrInsertUncompletedDag
	}
	// cp must be an ancestor of the block
	if !isCpAncestor {
		log.Warn("Block verification: checkpoint is not ancestor", "hash", block.Hash().Hex(), "cpHash", block.CpHash().Hex())
		return false, nil
	}

	// Verify block checkpoint
	if !bc.verifyCheckpoint(block) {
		return false, nil
	}

	// Verify block height
	if !bc.verifyBlockHeight(block, len(ancestors)) {
		return false, nil
	}

	return bc.verifyBlockParents(block)
}

func (bc *BlockChain) verifyBlockUsedGas(block *types.Block) bool {
	intrGasSum := uint64(0)
	for _, tx := range block.Transactions() {
		intrGasSum += tx.Gas()
	}

	if intrGasSum > block.GasLimit() {
		log.Warn("Block verification: intrinsic gas sum > gasLimit",
			"hash", block.Hash().Hex(),
			"gasLimit", block.GasLimit(),
			"IntrinsicGas", intrGasSum,
		)
		return false
	}

	return true
}

func (bc *BlockChain) verifyBlockHeight(block *types.Block, ancestorsCount int) bool {
	cpHeader := bc.GetHeader(block.CpHash())
	calcHeight := bc.calcBlockHeight(cpHeader.Height, ancestorsCount)
	if block.Height() != calcHeight {
		log.Warn("Block verification: block invalid height",
			"calcHeight", calcHeight,
			"height", block.Height(),
			"hash", block.Hash().Hex(),
			"cpHeight", cpHeader.Height,
		)
		return false
	}
	return true
}

func (bc *BlockChain) verifyBlockHashes(block *types.Block) bool {
	// Verify body hash
	blockBody := block.Body()
	if blockBody.CalculateHash() != block.BodyHash() {
		log.Warn("Block verification: invalid body hash",
			"hash", block.Hash().Hex(),
			"bl.bodyHash", block.BodyHash().Hex(),
			"calc.bodyHash", blockBody.CalculateHash().Hex(),
		)
		return false
	}
	// Verify transactions hash
	calcTxHash := types.DeriveSha(block.Transactions(), trie.NewStackTrie(nil))
	if calcTxHash != block.TxHash() {
		log.Warn("Block verification: invalid transactions hash",
			"hash", block.Hash().Hex(),
			"txHash", block.TxHash().Hex(),
			"calc.txHash", calcTxHash.Hex(),
		)
		return false
	}
	return true
}

func (bc *BlockChain) verifyBlockSlot(block *types.Block) bool {
	if block.Slot() > bc.GetSlotInfo().CurrentSlot()+1 {
		log.Warn("Block verification: future slot",
			"currentSlot", bc.GetSlotInfo().CurrentSlot(),
			"blockSlot", block.Slot(),
			"blockHash", block.Hash().Hex(),
			"blockTime", block.Time(),
			"timeNow", time.Now().Unix(),
		)
		return false
	}
	return true
}

func (bc *BlockChain) verifyBlockParents(block *types.Block) (bool, error) {
	if len(block.ParentHashes()) == 0 {
		log.Warn("Block verification: no parents", "hash", block.Hash().Hex())
		return false, nil
	}

	var pSlot uint64
	parentsSlotsEquals := true
	parentsHeaders := bc.GetHeadersByHashes(block.ParentHashes())
	for i, parentHash := range block.ParentHashes() {
		parent := parentsHeaders[parentHash]
		if parent == nil {
			log.Warn("Block verification: parent not found",
				"hash", block.Hash().Hex(),
				"parent", parentHash.Hex(),
			)
			return false, ErrInsertUncompletedDag
		}
		//check parent Height
		if parent.Height >= block.Height() {
			log.Warn("Block verification: bad parent height",
				"height", block.Height(),
				"parent.height", parent.Height,
				"hash", block.Hash().Hex(),
				"parent", parentHash.Hex(),
			)
			return false, nil
		}
		//check parent slot
		if parent.Slot >= block.Slot() {
			log.Warn("Block verification: bad parent slot",
				"slot", block.Slot(),
				"parent.slot", parent.Slot,
				"hash", block.Hash().Hex(),
				"parent", parentHash.Hex(),
			)
			return false, nil
		}
		//check parents' slots are equals
		if i == 0 {
			pSlot = parent.Slot
		} else if pSlot != parent.Slot {
			parentsSlotsEquals = false
		}
	}

	if !parentsSlotsEquals {
		//check there are not parent-ancestor relations
		for ph, parentHeader := range parentsHeaders {
			if parentHeader.Nr() > 0 || parentHeader.Height == 0 {
				continue
			}
			for pph, pparent := range parentsHeaders {
				if ph == pph {
					continue
				}
				isAncestor, err := bc.IsAncestorByTips(parentHeader, pparent.Hash())
				if err != nil {
					return false, err
				}
				if isAncestor {
					log.Warn("Block verification: parent-ancestor detected",
						"block", block.Hash().Hex(),
						"parent", parentHeader.Hash().Hex(),
						"parent-ancestor", pparent.Hash().Hex(),
					)
					return false, nil
				}
			}
		}
	}
	return true, nil
}

func (bc *BlockChain) verifyEmptyBlock(block *types.Block, creators []common.Address) error {
	if len(block.Transactions()) > 0 {
		return nil
	}

	if block.Coinbase() != creators[0] {
		log.Warn("Empty block verification failed: invalid coinbase",
			"blockHash", block.Hash().Hex(),
			"block slot", block.Slot(),
			"coinbase", block.Coinbase().Hex(),
		)
	}

	blockEpoch := bc.GetSlotInfo().SlotToEpoch(block.Slot())
	blockEpochStartSlot, err := bc.GetSlotInfo().SlotOfEpochStart(blockEpoch)
	if err != nil {
		log.Warn("Empty block verification failed: can`t calculate block`s epoch start slot",
			"blockHash", block.Hash().Hex(),
			"block slot", block.Slot(),
			"blockEpoch", blockEpoch,
			"coinbase", block.Coinbase().Hex(),
		)

		return err
	}

	if block.Slot() != blockEpochStartSlot {
		log.Warn("Empty block verification failed: do not expect an empty block in this slot",
			"blockHash", block.Hash().Hex(),
			"block slot", block.Slot(),
			"blockEpoch", blockEpoch,
			"coinbase", block.Coinbase().Hex(),
		)

		return err
	}

	haveBlocks, err := bc.HaveEpochBlocks(blockEpoch - 1)
	if haveBlocks {
		log.Warn("Empty block verification failed: previous epoch have blocks",
			"blockHash", block.Hash().Hex(),
			"block slot", block.Slot(),
			"blockEpoch", blockEpoch,
			"coinbase", block.Coinbase().Hex(),
		)

		return errors.New("block verification: unexpected empty block")
	}

	return nil
}

func (bc *BlockChain) verifyBlockEra(block *types.Block) bool {
	// Get the epoch of the block
	blockEpoch := bc.GetSlotInfo().SlotToEpoch(block.Slot())

	calcEra := bc.EpochToEra(blockEpoch)
	if calcEra.Number != block.Era() {
		log.Warn("Block verification: invalid era",
			"hash", block.Hash().Hex(),
			"era", block.Era(),
		)
		return false
	}
	return true
}

func (bc *BlockChain) verifyCheckpoint(block *types.Block) bool {
	// cp must be coordinated (received from coordinator)
	coordCp := bc.GetCoordinatedCheckpoint(block.CpHash())
	if coordCp == nil {
		log.Warn("Block verification: cp not found",
			"cp.Hash", block.CpHash().Hex(),
			"bl.Hash", block.Hash().Hex(),
		)
		return false
	}
	if bc.IsCheckpointOutdated(coordCp) {
		log.Warn("Block verification: cp is outdated",
			"cp.Hash", block.CpHash().Hex(),
			"bl.Hash", block.Hash().Hex(),
		)
		return false
	}
	// check cp block exists
	cpHeader := bc.GetHeader(block.CpHash())
	if cpHeader == nil {
		log.Warn("Block verification: cp block not found",
			"cp.Hash", block.CpHash().Hex(),
			"bl.Hash", block.Hash().Hex(),
		)
		return false
	}
	// cp must be finalized
	if cpHeader.Height > 0 && cpHeader.Nr() == 0 {
		log.Warn("Block verification: cp is not finalized",
			"bl.CpNumber", block.CpNumber(),
			"cp.Number", cpHeader.Nr(),
			"cp.Height", cpHeader.Height,
			"cp.Hash", block.CpHash().Hex(),
			"bl.Hash", block.Hash().Hex(),
		)
		return false
	}
	if block.CpNumber() != cpHeader.Nr() {
		log.Warn("Block verification: mismatch cp fin numbers",
			"cp.Height", cpHeader.Height,
			"cp.Hash", block.CpHash().Hex(),
			"bl.Hash", block.Hash().Hex(),
			"cp.Number", cpHeader.Nr(),
			"bl.CpNumber", block.CpNumber(),
		)
		return false
	}
	if block.CpHash() != cpHeader.Hash() {
		log.Warn("Block verification: mismatch cp fin hashes",
			"bl.CpNumber", block.CpNumber(),
			"cp.Number", cpHeader.Nr(),
			"cp.Height", cpHeader.Height,
			"bl.Hash", block.Hash().Hex(),
			"cp.Hash", block.CpHash().Hex(),
			"bl.CpHash", block.CpHash().Hex(),
		)
		return false
	}
	if block.CpRoot() != cpHeader.Root {
		log.Warn("Block verification: mismatch cp roots",
			"bl.CpNumber", block.CpNumber(),
			"cp.Number", cpHeader.Nr(),
			"cp.Height", cpHeader.Height,
			"cp.Hash", block.CpHash().Hex(),
			"bl.Hash", block.Hash().Hex(),
			"cp.Root", cpHeader.Root.Hex(),
			"bl.CpRoot", block.CpRoot().Hex(),
		)
		return false
	}
	if block.CpReceiptHash() != cpHeader.ReceiptHash {
		log.Warn("Block verification: mismatch cp receipt hashes",
			"bl.CpNumber", block.CpNumber(),
			"cp.Number", cpHeader.Nr(),
			"cp.Height", cpHeader.Height,
			"cp.Hash", block.CpHash().Hex(),
			"bl.Hash", block.Hash().Hex(),
			"cp.ReceiptHash", cpHeader.ReceiptHash.Hex(),
			"bl.CpReceiptHash", block.CpReceiptHash().Hex(),
		)
		return false
	}
	if block.CpGasUsed() != cpHeader.GasUsed {
		log.Warn("Block verification: mismatch cp used gas",
			"bl.CpNumber", block.CpNumber(),
			"cp.Number", cpHeader.Nr(),
			"cp.Height", cpHeader.Height,
			"cp.Hash", block.CpHash().Hex(),
			"bl.Hash", block.Hash().Hex(),
			"cp.GasUsed", cpHeader.GasUsed,
			"bl.CpGasUsed", block.CpGasUsed(),
		)
		return false
	}
	if block.CpBloom() != cpHeader.Bloom {
		log.Warn("Block verification: mismatch cp bloom",
			"bl.CpNumber", block.CpNumber(),
			"cp.Number", cpHeader.Nr(),
			"cp.Height", cpHeader.Height,
			"cp.Hash", block.CpHash().Hex(),
			"bl.Hash", block.Hash().Hex(),
			"cp.Bloom", cpHeader.Bloom,
			"bl.CpBloom", block.CpBloom(),
		)
		return false
	}

	if cpHeader.BaseFee.Cmp(block.CpBaseFee()) != 0 {
		log.Warn("Block verification: mismatch cp base fee",
			"bl.CpNumber", block.CpNumber(),
			"cp.Number", cpHeader.Nr(),
			"cp.Height", cpHeader.Height,
			"cp.Hash", block.CpHash().Hex(),
			"bl.Hash", block.Hash().Hex(),
			"cp.BaseFee", cpHeader.BaseFee.String(),
			"bl.CpBaseFee", block.CpBaseFee().String(),
		)
		return false
	}
	// check accordance to parent checkpoints
	for _, ph := range block.ParentHashes() {
		parBdag := bc.GetBlockDag(ph)
		// block cp must be same or greater than parents
		if block.CpHash() == parBdag.CpHash {
			continue
		}
		if cpHeader.Height <= parBdag.CpHeight {
			log.Warn("Block verification: cp height less of parent cp",
				"parent.CpHeight", parBdag.CpHeight,
				"cp.Height", cpHeader.Height,
				"parent.Hash", ph.Hex(),
				"cp.Hash", block.CpHash().Hex(),
				"bl.Hash", block.Hash().Hex(),
			)
			return false
		}
		// otherwise block cp must be in past of parent and grater parent cp
		if !parBdag.OrderedAncestorsHashes.Has(block.CpHash()) {
			log.Warn("Block verification: cp not found in range from parent cp",
				"parent.Hash", ph.Hex(),
				"range", parBdag.OrderedAncestorsHashes,
				"cp.Hash", block.CpHash().Hex(),
				"bl.Hash", block.Hash().Hex(),
			)
			return false
		}
	}
	return true
}

func (bc *BlockChain) IsCheckpointOutdated(cp *types.Checkpoint) bool {
	lastCp := bc.GetLastCoordinatedCheckpoint()
	//if is current cp
	if cp.Epoch >= lastCp.Epoch {
		return false
	}
	// compare with prev cp
	lcpHeader := bc.GetHeader(lastCp.Spine)
	if lcpHeader == nil {
		return false
	}
	prevCpHash := lcpHeader.CpHash
	if lcpHeader.Hash() == bc.genesisBlock.Hash() {
		prevCpHash = bc.genesisBlock.Hash()
	}
	prevCp := bc.GetCoordinatedCheckpoint(prevCpHash)
	if cp.Epoch >= prevCp.Epoch {
		return false
	}
	log.Warn("Outdated cp detected",
		"cp.Epoch", cp.Epoch,
		"cp.FinEpoch", cp.FinEpoch,
		"cp.Spine", cp.Spine.Hex(),
		"lastCp.Epoch", lastCp.Epoch,
		"lastCp.FinEpoch", lastCp.FinEpoch,
		"lastCp.Spine", lastCp.Spine.Hex(),
		"prevCp.Epoch", prevCp.Epoch,
		"prevCp.FinEpoch", prevCp.FinEpoch,
		"prevCp.Spine", prevCp.Spine.Hex(),
	)
	return true
}

// insertBlocks inserts blocks to chain
func (bc *BlockChain) insertBlocks(chain types.Blocks, validate bool, op string) (int, error) {

	// If the chain is terminating, don't even bother starting up
	if atomic.LoadInt32(&bc.procInterrupt) == 1 {
		return 0, nil
	}

	// Start a parallel signature recovery (signer will fluke on fork transition, minimal perf loss)
	senderCacher.recoverFromBlocks(types.MakeSigner(bc.chainConfig), chain)

	var (
		stats     = insertStats{startTime: mclock.Now()}
		lastCanon *types.Block
	)
	// Fire a single chain head event if we've progressed the chain
	defer func() {
		lfb := bc.GetLastFinalizedBlock()
		if lastCanon != nil && lfb.Hash() == lastCanon.Hash() {
			bc.chainHeadFeed.Send(ChainHeadEvent{lastCanon, ET_SYNC_FIN})
		}
	}()
	// Start the parallel header verifier
	headers := make([]*types.Header, len(chain))
	headerMap := make(types.HeaderMap, len(chain))

	for i, block := range chain {
		headers[i] = block.Header()
		headerMap[block.Hash()] = block.Header()
	}

	// Peek the error for the first block to decide the directing import logic
	it := newInsertIterator(chain, bc.validator)

	block, err := it.next()

	switch {
	// First block is pruned, insert as sidechain and reorg
	case errors.Is(err, consensus.ErrPrunedAncestor):
		log.Warn("Insert blocks: pruned ancestor, inserting as sidechain", "op", op, "hash", block.Hash().Hex())
		return bc.insertSideChain(block, it)

	// Some other error occurred, abort
	case err != nil:
		log.Error("Insert blocks: err", "hash", block.Hash().Hex(), "err", err)
		stats.ignored += len(it.chain)
		return it.index, err
	}

	for ; block != nil && err == nil; block, err = it.next() {
		// If the chain is terminating, stop processing blocks
		if bc.insertStopped() {
			log.Debug("Insert blocks: abort during block processing", "op", op)
			break
		}

		if validate {
			// cp must be coordinated (received from coordinator)
			if coordCp := bc.GetCoordinatedCheckpoint(block.CpHash()); coordCp == nil {
				log.Warn("Insert blocks: CP verification: CP not found as coordinated cp",
					"op", op,
					"cp.Nr", block.CpNumber(),
					"cp.Hash", block.CpHash().Hex(),
					"bl.Slot", block.Slot(),
					"bl.Hash", block.Hash().Hex(),
				)
				// if checkpoint of propagated block is not finalized - set IsSynced=false
				cpHeader := bc.GetHeaderByHash(block.CpHash())
				if cpHeader != nil {
					si := bc.GetSlotInfo()
					lCp := bc.GetLastCoordinatedCheckpoint()
					cpEpoch := si.SlotToEpoch(cpHeader.Slot)
					currEpoch := si.SlotToEpoch(si.CurrentSlot())
					if cpEpoch <= lCp.Epoch && cpEpoch >= currEpoch {
						log.Error("Insert blocks: CP verification: block rejected (bad cp epoch)",
							"op", op,
							"cpEpoch", cpEpoch,
							"lCp.Epoch", lCp.Epoch,
							"currEpoch", currEpoch,
							"cp.Nr", block.CpNumber(),
							"cp.Hash", block.CpHash().Hex(),
							"bl.Slot", block.Slot(),
							"bl.Hash", block.Hash().Hex(),
						)
						bc.CacheInvalidBlock(block)
						continue
					}
					// if cp is not finalized
					if cpHeader.Height > 0 && cpHeader.Nr() == 0 {
						lfb := bc.GetLastFinalizedHeader()
						if block.CpNumber() <= lfb.Nr() {
							log.Error("Insert blocks: CP verification: block rejected (bad cp nr)",
								"op", op,
								"bl.CpNr", block.CpNumber(),
								"lfNr", lfb.Nr(),
								"cp.Hash", block.CpHash().Hex(),
								"bl.Slot", block.Slot(),
								"bl.Hash", block.Hash().Hex(),
							)
							bc.CacheInvalidBlock(block)
							continue
						}
						// todo fix the gap here
						log.Warn("Insert blocks: CP verification: cp is not finalized",
							"op", op,
							"cpHash", block.CpHash(),
							"cpSlot", cpHeader.Slot,
						)
						bc.SetIsSynced(false)
					} else {
						if block.CpNumber() != cpHeader.Nr() {
							log.Error("Insert blocks: CP verification: block rejected (bad cp mismatch nr)",
								"op", op,
								"bl.CpNr", block.CpNumber(),
								"cp.Nr", cpHeader.Nr(),
								"cp.Hash", block.CpHash().Hex(),
								"bl.Slot", block.Slot(),
								"bl.Hash", block.Hash().Hex(),
							)
							bc.CacheInvalidBlock(block)
							continue
						}
					}
				} else {
					log.Error("Insert blocks: CP verification: block rejected (cp not found)", "op", op, "cpHash", block.CpHash())
					bc.CacheInvalidBlock(block)
					continue
				}
			}

			if ok, err := bc.VerifyBlock(block); !ok {
				if err != nil {
					return it.index, err
				}
				bc.CacheInvalidBlock(block)
				if op == opCreate || op == opPropagate {
					log.Warn("Error while insert block", "err", errInvalidBlock, "op", op)
					return it.index, errInvalidBlock
				}
				continue
			}
		}

		log.Info("Insert blocks:", "op", op, "Slot", block.Slot(), "Height", block.Height(), "Hash", block.Hash().Hex(), "txs", len(block.Transactions()), "parents", block.ParentHashes())

		rawdb.WriteBlock(bc.db, block)
		rawdb.AddSlotBlockHash(bc.Database(), block.Slot(), block.Hash())
		bc.AppendToChildren(block.Hash(), block.ParentHashes())

		log.Debug("Insert blocks: remove optimistic spines from cache", "op", op, "slot", block.Slot())
		bc.removeOptimisticSpinesFromCache(block.Slot())

		tmpTips := types.Tips{}
		for _, h := range block.ParentHashes() {
			bdag := bc.GetBlockDag(h)
			if bdag == nil {
				// create parent blockDag
				parentBlock := bc.GetHeader(h)
				if parentBlock == nil {
					log.Error("Insert blocks: create parent blockDag: parent not found",
						"slot", block.Slot(),
						"height", block.Height(),
						"hash", block.Hash().Hex(),
						"parent", h.Hex(),
						"err", ErrInsertUncompletedDag,
					)
					return it.index, ErrInsertUncompletedDag
				}
				cpHeader := bc.GetHeader(parentBlock.CpHash)
				if cpHeader == nil {
					log.Error("Insert blocks: create parent blockDag: parent cp not found",
						"slot", block.Slot(),
						"height", block.Height(),
						"hash", block.Hash().Hex(),
						"parent", h.Hex(),
						"parentCP", parentBlock.CpHash.Hex(),
						"err", ErrInsertUncompletedDag,
					)
					return it.index, ErrInsertUncompletedDag
				}

				log.Warn("Insert blocks: create parent blockDag",
					"parent.slot", parentBlock.Slot,
					"parent.height", parentBlock.Height,
					"parent", h.Hex(),
					"slot", block.Slot(),
					"height", block.Height(),
					"hash", block.Hash().Hex(),
				)
				//isCpAncestor, ancestors, unl, err := bc.CollectAncestorsAftCpByParents(parentBlock.ParentHashes, parentBlock.CpHash)
				_, ancestors, unl, err := bc.CollectAncestorsAftCpByParents(parentBlock.ParentHashes, parentBlock.CpHash)
				if err != nil {
					return it.index, err
				}
				if len(unl) > 0 {
					log.Error("Insert blocks: create parent blockDag: incomplete dag",
						"err", ErrInsertUncompletedDag,
						"parent", h.Hex(),
						"parent.slot", parentBlock.Slot,
						"parent.height", parentBlock.Height,
						"slot", block.Slot(),
						"height", block.Height(),
						"hash", block.Hash().Hex(),
					)
					return it.index, ErrInsertUncompletedDag
				}
				//if !isCpAncestor {
				//	log.Error("Insert blocks: create parent blockDag: cp is not ancestor",
				//		"err", ErrCpIsnotAncestor,
				//		"parent", h.Hex(),
				//		"parent.slot", parentBlock.Slot,
				//		"parent.height", parentBlock.Height,
				//		"slot", block.Slot(),
				//		"height", block.Height(),
				//		"hash", block.Hash().Hex(),
				//	)
				//	return it.index, ErrCpIsnotAncestor
				//}
				delete(ancestors, cpHeader.Hash())
				bdag = &types.BlockDAG{
					Hash:                   h,
					Height:                 parentBlock.Height,
					Slot:                   parentBlock.Slot,
					CpHash:                 parentBlock.CpHash,
					CpHeight:               cpHeader.Height,
					OrderedAncestorsHashes: ancestors.Hashes(),
				}
			}
			bdag.OrderedAncestorsHashes = bdag.OrderedAncestorsHashes.Difference(common.HashArray{bc.Genesis().Hash()})
			tmpTips.Add(bdag)
		}
		dagChainHashes, err := bc.CollectAncestorsHashesByTips(tmpTips, block.CpHash())
		if err != nil {
			return it.index, err
		}
		cpHeader := bc.GetHeader(block.CpHash())
		if cpHeader == nil {
			return it.index, ErrInsertUncompletedDag
		}
		dagBlock := &types.BlockDAG{
			Hash:                   block.Hash(),
			Height:                 block.Height(),
			Slot:                   block.Slot(),
			CpHash:                 block.CpHash(),
			CpHeight:               cpHeader.Height,
			OrderedAncestorsHashes: dagChainHashes,
		}
		bc.AddTips(dagBlock)
		bc.RemoveTips(dagBlock.OrderedAncestorsHashes)
		bc.WriteCurrentTips()
		bc.MoveTxsToProcessing(block)

		log.Info("Insert blocks: success", "op", op, "slot", block.Slot(), "height", block.Height(), "hash", block.Hash().Hex())
	}

	return it.index, err
}

func (bc *BlockChain) UpdateFinalizingState(block *types.Block, stateBlock *types.Block) error {
	var (
		lastCanon   *types.Block
		activeState *state.StateDB
	)

	defer func() {
		lfb := bc.GetLastFinalizedBlock()
		if lastCanon != nil && lfb.Hash() == lastCanon.Hash() {
			bc.chainHeadFeed.Send(ChainHeadEvent{lastCanon, ET_SYNC_FIN})
		}
	}()

	defer func() {
		// The chain importer is starting and stopping trie prefetchers. If a bad
		// block or other error is hit however, an early return may not properly
		// terminate the background threads. This defer ensures that we clean up
		// and dangling prefetcher, without defering each and holding on live refs.
		if activeState != nil {
			activeState.StopPrefetcher()
		}
	}()

	if stateBlock == nil {
		log.Error("PreFinalizingUpdateState: CpBlock = nil", "CpNumber", block.CpNumber())
		return fmt.Errorf("PreFinalizingUpdateState: unknown CpBlock, number=%v", block.CpNumber())
	}
	statedb, stateErr := bc.StateAt(stateBlock.Root())
	if stateErr != nil && stateBlock == nil {
		log.Error("Propagated block import state err", "Height", block.Height(), "hash", block.Hash().Hex(), "stateBlock", stateBlock, "err", stateErr)
		return stateErr
	}

	start := time.Now()
	// Enable prefetching to pull in trie node paths while processing transactions
	statedb.StartPrefetcher("chain")
	activeState = statedb

	header := block.Header()

	// Set baseFee and GasLimit
	creatorsPerSlotCount := bc.Config().ValidatorsPerSlot
	if creatorsPerSlot, err := bc.ValidatorStorage().GetCreatorsBySlot(bc, header.Slot); err == nil {
		creatorsPerSlotCount = uint64(len(creatorsPerSlot))
	}
	validators, _ := bc.ValidatorStorage().GetValidators(bc, header.Slot, true, false, "UpdateFinalizingState")
	header.BaseFee = misc.CalcSlotBaseFee(bc.Config(), creatorsPerSlotCount, uint64(len(validators)), bc.Genesis().GasLimit())

	block.SetHeader(header)

	// Process block using the parent state as reference point
	subStart := time.Now()
	statedb, receipts, logs, usedGas := bc.CommitBlockTransactions(block, statedb)

	header.GasUsed = usedGas
	block.SetReceipt(receipts, trie.NewStackTrie(nil))

	header.Root = statedb.IntermediateRoot(true)

	// Update the metrics touched during block processing
	accountReadTimer.Update(statedb.AccountReads)                 // Account reads are complete, we can mark them
	storageReadTimer.Update(statedb.StorageReads)                 // Storage reads are complete, we can mark them
	accountUpdateTimer.Update(statedb.AccountUpdates)             // Account updates are complete, we can mark them
	storageUpdateTimer.Update(statedb.StorageUpdates)             // Storage updates are complete, we can mark them
	snapshotAccountReadTimer.Update(statedb.SnapshotAccountReads) // Account reads are complete, we can mark them
	snapshotStorageReadTimer.Update(statedb.SnapshotStorageReads) // Storage reads are complete, we can mark them
	trieHash := statedb.AccountHashes + statedb.StorageHashes     // Save to not double count in validation
	trieProc := statedb.SnapshotAccountReads + statedb.AccountReads + statedb.AccountUpdates
	trieProc += statedb.SnapshotStorageReads + statedb.StorageReads + statedb.StorageUpdates

	blockExecutionTimer.Update(time.Since(subStart) - trieProc - trieHash)

	// Validate the state using the default validator
	subStart = time.Now()
	if err := bc.validator.ValidateState(block, statedb, receipts, usedGas); err != nil {
		log.Warn("Red block insertion to chain while propagate", "nr", block.Nr(), "height", block.Height(), "slot", block.Slot(), "hash", block.Hash().Hex(), "err", err)
		return err
	}
	procTime := time.Since(start)

	// Update the metrics touched during block validation
	accountHashTimer.Update(statedb.AccountHashes) // Account hashes are complete, we can mark them
	storageHashTimer.Update(statedb.StorageHashes) // Storage hashes are complete, we can mark them

	blockValidationTimer.Update(time.Since(subStart) - (statedb.AccountHashes + statedb.StorageHashes - trieHash))

	// Write the block to the chain and get the status.
	subStart = time.Now()
	log.Info("Finalization: update state", "height", block.Height(), "hash", block.Hash().Hex())
	status, err := bc.writeBlockWithState(block, receipts, logs, statedb, ET_SKIP, "updateFinalizingState")
	if err != nil {
		return err
	}
	// Update the metrics touched during block commit
	accountCommitTimer.Update(statedb.AccountCommits)   // Account commits are complete, we can mark them
	storageCommitTimer.Update(statedb.StorageCommits)   // Storage commits are complete, we can mark them
	snapshotCommitTimer.Update(statedb.SnapshotCommits) // Snapshot commits are complete, we can mark them

	blockWriteTimer.Update(time.Since(subStart) - statedb.AccountCommits - statedb.StorageCommits - statedb.SnapshotCommits)
	blockInsertTimer.UpdateSince(start)

	switch status {
	case CanonStatTy:
		log.Debug("Inserted new block", "hash", block.Hash(),
			"txs", len(block.Transactions()), "gas", block.GasUsed(),
			"elapsed", common.PrettyDuration(time.Since(start)),
			"root", block.Root())

		lastCanon = block

		// Only count canonical blocks for GC processing time
		bc.gcproc += procTime

	case SideStatTy:
		log.Debug("Inserted forked block", "hash", block.Hash(),
			"elapsed", common.PrettyDuration(time.Since(start)),
			"txs", len(block.Transactions()), "gas", block.GasUsed(),
			"root", block.Root())

	default:
		// This in theory is impossible, but lets be nice to our future selves and leave
		// a log, instead of trying to track down blocks imports that don't emit logs.
		log.Warn("Inserted block with unknown status", "hash", block.Hash(),
			"elapsed", common.PrettyDuration(time.Since(start)),
			"txs", len(block.Transactions()), "gas", block.GasUsed(),
			"root", block.Root())
	}

	return nil
}

// InsertChain attempts to insert the given batch of blocks in to the canonical
// chain or, otherwise, create a fork. If an error is returned it will return
// the index number of the failing block as well an error describing what went
// wrong.
//
// After insertion is done, all accumulated events will be fired.
func (bc *BlockChain) InsertChain(chain types.Blocks) (int, error) {
	// Sanity check that we have something meaningful to import
	if len(chain) == 0 {
		return 0, nil
	}

	bc.blockProcFeed.Send(true)
	defer bc.blockProcFeed.Send(false)

	// Remove already known canon-blocks
	var (
		block, prev *types.Block
	)
	// Do a sanity check that the provided chain is actually ordered and linked
	for i := 1; i < len(chain); i++ {
		block = chain[i]
		prev = chain[i-1]
		if block.Number() != nil && prev.Number() != nil && *block.Number() != *prev.Number()+1 {
			// Chain broke ancestry, log a message (programming error) and skip insertion
			log.Error("Non contiguous block insert", "number", block.Nr(), "hash", block.Hash().Hex(), "prevnumber", prev.Nr(), "prevhash", prev.Hash())
			return 0, fmt.Errorf("non contiguous insert: item %d is #%d [%x..], item %d is #%d [%x..]", i-1, prev.Nr(),
				prev.Hash().Bytes()[:4], i, block.Nr(), block.Hash().Bytes()[:4])
		}
	}

	// Pre-check passed, start the full block imports.
	if !bc.chainmu.TryLock() {
		return 0, errChainStopped
	}
	defer bc.chainmu.Unlock()
	return bc.insertChain(chain)
}

func (bc *BlockChain) CollectAncestorsHashesByTips(tips types.Tips, cpHash common.Hash) (common.HashArray, error) {
	cpHeader := bc.GetHeader(cpHash)
	ancestorsHashes := make(common.HashArray, 0)
	for _, tip := range tips {
		if tip.Hash == cpHash {
			continue
		}
		if tip.CpHash == cpHash {
			ancestorsHashes = append(ancestorsHashes, tip.OrderedAncestorsHashes...)
			ancestorsHashes = append(ancestorsHashes, tip.Hash)
			ancestorsHashes.Deduplicate()
			continue
		}
		// current cp must be in past of parent
		if !tip.OrderedAncestorsHashes.Has(cpHash) {
			// must be rejected by validation
			log.Error("Collect OrderedAncestorsHashes: bad tips",
				"tips.Hash", tip.Hash.Hex(),
				"CpHash", cpHash,
				"tip.ancestors", tip.OrderedAncestorsHashes,
				"tips", tips.Print(),
			)
			return nil, fmt.Errorf("bad tips: not found cp=%#x for tip=%#x", cpHash, tip.Hash)
		}
		// exclude cp hash
		ancHashes := tip.OrderedAncestorsHashes.Difference(common.HashArray{cpHash})
		ancestors := bc.GetHeadersByHashes(ancHashes)
		for h, anc := range ancestors {
			// skipping if finalised before current checkpoint
			if anc.Height > 0 && anc.Nr() > 0 && anc.Nr() < cpHeader.Nr() {
				continue
			}
			ancestorsHashes = append(ancestorsHashes, h)
		}
		ancestorsHashes = append(ancestorsHashes, tip.Hash)
		ancestorsHashes.Deduplicate()
	}
	return ancestorsHashes, nil
}

func (bc *BlockChain) CalcBlockHeightByTips(tips types.Tips, cpHash common.Hash) (uint64, error) {
	ancestors, err := bc.CollectAncestorsHashesByTips(tips, cpHash)
	if err != nil {
		return 0, err
	}
	cpHeader := bc.GetHeader(cpHash)
	return bc.calcBlockHeight(cpHeader.Height, len(ancestors)), nil
}

func (bc *BlockChain) CalcBlockHeightByParents(parents common.HashArray, cpHash common.Hash) (uint64, error) {
	var (
		unl       common.HashArray
		err       error
		ancestors = types.HeaderMap{}
	)
	cpHead := bc.GetHeader(cpHash)
	if cpHead == nil || cpHead.Height > 0 && cpHead.Nr() == 0 {
		return 0, ErrCpNotFinalized
	}
	_, ancestors, unl, err = bc.CollectAncestorsAftCpByParents(parents, cpHash)
	if err != nil {
		return 0, err
	}
	if len(unl) > 0 {
		return 0, ErrInsertUncompletedDag
	}
	if len(ancestors) != len(ancestors.RmEmpty()) {
		return 0, ErrInsertUncompletedDag
	}
	return bc.calcBlockHeight(cpHead.Height, len(ancestors)), nil
}

func (bc *BlockChain) calcBlockHeight(baseHeight uint64, ancestorsCount int) uint64 {
	return baseHeight + uint64(ancestorsCount) + 1
}

// CollectStateDataByParents collects state data of current dag chain to insert block.
func (bc *BlockChain) CollectStateDataByParents(parents common.HashArray) (statedb *state.StateDB, stateBlock *types.Block, recommitBlocks []*types.Block, calcHeight uint64, err error) {
	lastFinBlock := bc.GetLastFinalizedBlock()
	parentBlocks := bc.GetBlocksByHashes(parents)
	//check is parents exists
	unl := common.HashArray{}
	for ph, b := range parentBlocks {
		if b == nil {
			unl = append(unl, ph)
		}
	}
	if len(unl) > 0 {
		log.Error("Error while collect state data by block (unknown blocks detected)", "parents", parents, "unknown", unl)
		return statedb, stateBlock, recommitBlocks, calcHeight, ErrInsertUncompletedDag
	}

	sortedBlocks := types.SpineSortBlocks(parentBlocks.ToArray())

	stateBlock = sortedBlocks[0]
	statedb, err = bc.StateAt(stateBlock.Root())
	if err != nil {
		log.Error("Error while get state by parents", "slot", stateBlock.Slot(), "nr", stateBlock.Nr(), "height", stateBlock.Height(), "hash", stateBlock.Hash().Hex(), "err", err)
	}
	if statedb != nil {
		baseRecommitBlocks := sortedBlocks[1:]

		//check that all parents are in state
		stateParents := stateBlock.ParentHashes()
		for _, rb := range baseRecommitBlocks {
			phs := rb.ParentHashes()
			difParents := phs.Difference(stateParents)
			if len(difParents) > 0 {
				_, _, parentRecommits, _, err := bc.CollectStateDataByParents(phs)
				if err != nil {
					log.Error("Error while get state by parents (forked parents)", "slot", stateBlock.Slot(), "nr", stateBlock.Nr(), "height", stateBlock.Height(), "hash", stateBlock.Hash().Hex(), "err", err)
					return statedb, stateBlock, recommitBlocks, calcHeight, err
				}
				for _, parentRb := range parentRecommits {
					if difParents.Has(parentRb.Hash()) {
						recommitBlocks = append(recommitBlocks, parentRb)
					}
				}
			}
			recommitBlocks = append(recommitBlocks, rb)
		}

		calcHeight = bc.calcBlockHeight(stateBlock.Height(), len(recommitBlocks))
		return statedb, stateBlock, recommitBlocks, calcHeight, nil
	}
	//if state is last finalized block
	if sortedBlocks[0].Nr() == lastFinBlock.Nr() ||
		//if state is dag block
		sortedBlocks[0].Slot() > lastFinBlock.Slot() && sortedBlocks[0].Nr() == 0 && sortedBlocks[0].Height() > 0 {
		log.Error("Error while collect state data by block (bad spine state)", "parents", parents)
		return statedb, stateBlock, recommitBlocks, calcHeight, ErrSpineStateNF
	}

	//if state is finalized block - search first spine in ancestors
	lfAncestor := bc.GetBlockByHash(sortedBlocks[0].Hash())
	var recomFinBlocks []*types.Block
	//todo check
	//statedb, stateBlock, recomFinBlocks, err = bc.CollectStateDataByFinalizedBlockRecursive(lfAncestor, nil)
	statedb, stateBlock, recomFinBlocks, err = bc.CollectStateDataByFinalizedBlock(lfAncestor)
	if err != nil {
		return statedb, stateBlock, recommitBlocks, calcHeight, err
	}
	recommitBlocks = append(recomFinBlocks, sortedBlocks[1:]...)
	calcHeight = bc.calcBlockHeight(stateBlock.Height(), len(recommitBlocks))
	return statedb, stateBlock, recommitBlocks, calcHeight, nil
}

// CollectStateDataByFinalizedBlock collects state data of current dag chain to insert new block.
func (bc *BlockChain) CollectStateDataByFinalizedBlock(block *types.Block) (statedb *state.StateDB, stateBlock *types.Block, recommitBlocks []*types.Block, err error) {
	finNr := block.Nr()
	if finNr == 0 {
		if block.Hash() != bc.genesisBlock.Hash() {
			log.Error("Collect State Data By Finalized Block: bad block number", "nr", finNr, "height", block.Height(), "hash", block.Hash().Hex())
			return statedb, stateBlock, recommitBlocks, fmt.Errorf("Collect State Data By Finalized Block: bad block number: nr=%d (height=%d  hash=%v)", finNr, block.Height(), block.Hash().Hex())
		}
		stdb, err := bc.StateAt(block.Root())
		if err == nil || stdb != nil {
			return stdb, block, recommitBlocks, nil
		}
	}

	parentBlocks := bc.GetBlocksByHashes(block.ParentHashes())
	sortedBlocks := types.SpineSortBlocks(parentBlocks.ToArray())
	stateBlock = sortedBlocks[0]
	statedb, err = bc.StateAt(stateBlock.Root())
	if err != nil || statedb == nil {
		return statedb, stateBlock, recommitBlocks, fmt.Errorf("Collect State Data By Finalized Block: state not found number: nr=%d (height=%d  hash=%v) err=%s", finNr, block.Height(), block.Hash().Hex(), err)
	}
	baseRecommitBlocks := sortedBlocks[1:]

	//check that all parents are in state
	stateParents := stateBlock.ParentHashes()
	for _, rb := range baseRecommitBlocks {
		phs := rb.ParentHashes()
		difParents := phs.Difference(stateParents)
		if len(difParents) > 0 {
			_, _, parentRecommits, err := bc.CollectStateDataByFinalizedBlock(rb)
			if err != nil {
				log.Error("Error while get state by parents (forked parents)", "slot", stateBlock.Slot(), "nr", stateBlock.Nr(), "height", stateBlock.Height(), "hash", stateBlock.Hash().Hex(), "err", err)
				return statedb, stateBlock, recommitBlocks, err
			}
			for _, parentRb := range parentRecommits {
				if difParents.Has(parentRb.Hash()) {
					recommitBlocks = append(recommitBlocks, parentRb)
				}
			}
		}
		recommitBlocks = append(recommitBlocks, rb)
	}

	return statedb, stateBlock, recommitBlocks, nil
}

// CollectStateDataByBlock collects state data of current dag chain to insert new block.
func (bc *BlockChain) CollectStateDataByBlock(block *types.Block) (statedb *state.StateDB, stateBlock *types.Block, err error) {
	finNr := block.Nr()
	if finNr == 0 {
		if block.Hash() != bc.genesisBlock.Hash() {
			log.Error("Collect State Data By Finalized Block: bad block number", "nr", finNr, "height", block.Height(), "hash", block.Hash().Hex())
			return statedb, stateBlock, fmt.Errorf("Collect State Data By Finalized Block: bad block number: nr=%d (height=%d  hash=%v)", finNr, block.Height(), block.Hash().Hex())
		}
		stdb, err := bc.StateAt(block.Root())
		if err == nil || stdb != nil {
			return stdb, block, nil
		}
	}
	stateBlock = bc.GetBlockByNumber(block.Nr() - 1)

	statedb, err = bc.StateAt(stateBlock.Root())
	if err != nil || statedb == nil {
		return statedb, stateBlock, fmt.Errorf("Collect State Data By Finalized Block: state not found number: nr=%d (height=%d  hash=%v) err=%s", finNr, block.Height(), block.Hash().Hex(), err)
	}

	return statedb, stateBlock, nil
}

// CommitBlockTransactions commits transactions of red blocks.
func (bc *BlockChain) CommitBlockTransactions(block *types.Block, statedb *state.StateDB) (*state.StateDB, []*types.Receipt, []*types.Log, uint64) {

	log.Info("Commit block transactions", "Nr", block.Nr(), "height", block.Height(), "slot", block.Slot(), "hash", block.Hash().Hex())

	gasPool := new(GasPool).AddGas(block.GasLimit())
	signer := types.MakeSigner(bc.chainConfig)

	var coalescedLogs []*types.Log
	var receipts []*types.Receipt
	var rlogs []*types.Log

	gasUsed := new(uint64)
	for i, tx := range block.Transactions() {
		from, _ := types.Sender(signer, tx)
		// Start executing the transaction
		statedb.Prepare(tx.Hash(), i)

		receipt, err := ApplyTransaction(bc.chainConfig, bc, &block.Header().Coinbase, gasPool, statedb, block.Header(), tx, gasUsed, *bc.GetVMConfig(), bc)
		receipts = append(receipts, receipt)
		rlogs = append(rlogs, receipt.Logs...)
		switch {
		case errors.Is(err, ErrGasLimitReached):
			// Pop the current out-of-gas transaction without shifting in the next from the account
			log.Error("Gas limit exceeded for current block while recommit", "sender", from, "hash", tx.Hash().Hex())

		case errors.Is(err, ErrNonceTooLow):
			// New head notification data race between the transaction pool and miner, shift
			log.Error("Skipping transaction with low nonce while commit", "bl.height", block.Height(), "bl.hash", block.Hash().Hex(), "sender", from, "nonce", tx.Nonce(), "hash", tx.Hash().Hex())

		case errors.Is(err, ErrNonceTooHigh):
			// Reorg notification data race between the transaction pool and miner, skip account =
			log.Error("Skipping account with hight nonce while commit", "bl.height", block.Height(), "bl.hash", block.Hash().Hex(), "sender", from, "nonce", tx.Nonce(), "hash", tx.Hash().Hex())

		case errors.Is(err, nil):
			// Everything ok, collect the logs and shift in the next transaction from the same account
			coalescedLogs = append(coalescedLogs, receipt.Logs...)
			// create transaction lookup
			bc.WriteTxLookupEntry(i, tx.Hash(), block.Hash(), receipt.Status)

		case errors.Is(err, ErrTxTypeNotSupported):
			// Pop the unsupported transaction without shifting in the next from the account
			log.Error("Skipping unsupported transaction type while commit", "sender", from, "type", tx.Type(), "hash", tx.Hash().Hex())

		default:
			// Strange error, discard the transaction and get the next in line (note, the
			// nonce-too-high clause will prevent us from executing in vain).
			log.Error("Transaction failed, account skipped while commit", "hash", tx.Hash().Hex(), "err", err)
		}
	}

	rawdb.WriteReceipts(bc.db, block.Hash(), receipts)
	bc.handleBlockValidatorSyncReceipts(block, receipts)

	bc.chainFeed.Send(ChainEvent{Block: block, Hash: block.Hash(), Logs: rlogs})
	if len(rlogs) > 0 {
		bc.logsFeed.Send(rlogs)
	}

	return statedb, receipts, rlogs, *gasUsed
}

func (bc *BlockChain) EstimateGas(msg types.Message, header *types.Header) (uint64, error) {
	if len(msg.Data()) == 0 {
		return params.TxGas, nil
	}

	if header == nil {
		header = bc.GetLastFinalizedHeader()
	}

	blockContext := NewEVMBlockContext(header, bc, &header.Coinbase)
	stateDb, err := bc.StateAt(header.Root)
	if err != nil {
		return 0, err
	}

	tokenProcessor := token.NewProcessor(blockContext, stateDb)
	validatorProcessor := validator.NewProcessor(blockContext, stateDb, bc)
	txType := GetTxType(msg, validatorProcessor, tokenProcessor)

	switch txType {
	case ValidatorMethodTxType, ValidatorSyncTxType:
		return IntrinsicGas(msg.Data(), msg.AccessList(), false, true)
	case ContractMethodTxType, ContractCreationTxType:
		return bc.EstimateGasByEvm(msg, header, stateDb, tokenProcessor, validatorProcessor)
	case TokenCreationTxType, TokenMethodTxType:
		return IntrinsicGas(msg.Data(), msg.AccessList(), false, false)
	default:
		return 0, ErrTxTypeNotSupported
	}
}

//func (bc *BlockChain) EstimateGasByEvm(msg types.Message,
//	header *types.Header,
//	blkCtx vm.BlockContext,
//	statedb *state.StateDB,
//	vp *validator.Processor,
//	tp *token.Processor,
//) (uint64, error) {
//	defer func(start time.Time) { log.Info("Executing EVM call finished", "runtime", time.Since(start)) }(time.Now())
//	from := msg.From()
//	statedb.SetNonce(from, msg.Nonce())
//	maxGas := (new(big.Int)).SetUint64(header.GasLimit)
//	gasBalance := new(big.Int).Mul(maxGas, msg.GasPrice())
//	reqBalance := new(big.Int).Add(gasBalance, msg.Value())
//	statedb.SetBalance(from, reqBalance)
//
//	gasPool := new(GasPool).AddGas(math.MaxUint64)
//
//	evm := vm.NewEVM(blkCtx, vm.TxContext{}, statedb, bc.Config(), *bc.GetVMConfig())
//
//	receipt, err := ApplyMessage(evm, tp, vp, msg, gasPool)
//	if err != nil {
//		log.Error("Tx estimate gas by evm: error", "lfNumber", header.Nr(), "tx", msg.TxHash().Hex(), "err", err)
//		return 0, err
//	}
//	log.Info("Tx estimate gas by evm: success", "lfNumber", header.Nr(), "tx", msg.TxHash().Hex(), "txGas", msg.Gas(), "calcGas", receipt.UsedGas)
//	return receipt.UsedGas, nil
//}

func (bc *BlockChain) EstimateGasByEvm(msg Message,
	header *types.Header,
	stateDb *state.StateDB,
	tp *token.Processor,
	vp *validator.Processor,
) (uint64, error) {
	// Binary search the gas requirement, as it may be higher than the amount used
	var (
		lo  = params.TxGas - 1
		hi  uint64
		cap uint64
	)

	// Determine the highest gas limit can be used during the estimation.
	if msg.Gas() >= params.TxGas {
		hi = msg.Gas()
	} else {
		hi = header.GasLimit
	}
	// Normalize the max fee per gas the call is willing to spend.
	var feeCap *big.Int
	if msg.GasPrice() != nil {
		feeCap = msg.GasPrice()
	} else if msg.GasFeeCap() != nil {
		feeCap = msg.GasPrice()
	} else {
		feeCap = common.Big0
	}
	// Recap the highest gas limit with account's available balance.
	if feeCap.BitLen() != 0 {
		balance := stateDb.GetBalance(msg.From()) // from can't be nil
		available := new(big.Int).Set(balance)
		if msg.Value() != nil {
			if msg.Value().Cmp(available) >= 0 {
				return 0, errors.New("insufficient funds for transfer")
			}
			available.Sub(available, msg.Value())
		}
		allowance := new(big.Int).Div(available, feeCap)

		// If the allowance is larger than maximum uint64, skip checking
		if allowance.IsUint64() && hi > allowance.Uint64() {
			transfer := msg.Value()
			if transfer == nil {
				transfer = new(big.Int)
			}
			log.Warn("Gas estimation capped by limited funds", "original", hi, "balance", balance,
				"sent", transfer, "maxFeePerGas", feeCap, "fundable", allowance)
			hi = allowance.Uint64()
		}
	}

	if hi > header.GasLimit {
		log.Warn("Caller gas above allowance, capping", "requested", hi, "cap", header.GasLimit)
		hi = header.GasLimit
	}

	// Create a helper to check if a gas allowance results in an executable transaction
	executable := func(gas uint64) (bool, *ExecutionResult, error) {
		msg = msg.SetGas(gas)

		result, err := bc.doCall(msg, header, tp, vp)
		if err != nil {
			if errors.Is(err, ErrIntrinsicGas) {
				return true, nil, nil // Special case, raise gas limit
			}
			return true, nil, err // Bail out
		}
		return result.Failed(), result, nil
	}
	for lo+1 < hi {
		mid := (hi + lo) / 2
		failed, _, err := executable(mid)
		if err != nil {
			return 0, err
		}
		if failed {
			lo = mid
		} else {
			hi = mid
		}
	}
	// Reject the transaction as invalid if it still fails at the highest allowance
	if hi == cap {
		failed, result, err := executable(hi)
		if err != nil {
			return 0, err
		}
		if failed {
			return 0, result.Err
		}
		// Otherwise, the specified gas cap is too low
		return 0, fmt.Errorf("gas required exceeds allowance (%d)", cap)
	}

	return hi, nil
}

// insertChain is the internal implementation of InsertChain, which assumes that
// 1) chains are contiguous, and 2) The chain mutex is held.
//
// This method is split out so that import batches that require re-injecting
// historical blocks can do so without releasing the lock, which could lead to
// racey behaviour. If a sidechain import is in progress, and the historic state
// is imported, but then new canon-head is added before the actual sidechain
// completes, then the historic state could be pruned again
func (bc *BlockChain) insertChain(chain types.Blocks) (int, error) {
	// If the chain is terminating, don't even bother starting up.
	if bc.insertStopped() {
		return 0, nil
	}
	// Start a parallel signature recovery (signer will fluke on fork transition, minimal perf loss)
	senderCacher.recoverFromBlocks(types.MakeSigner(bc.chainConfig), chain)

	var (
		stats     = insertStats{startTime: mclock.Now()}
		lastCanon *types.Block
	)
	// Fire a single chain head event if we've progressed the chain
	defer func() {
		lfb := bc.GetLastFinalizedBlock()
		if lastCanon != nil && lfb.Hash() == lastCanon.Hash() {
			bc.chainHeadFeed.Send(ChainHeadEvent{lastCanon, ET_OTHER})
		}
	}()
	// Start the parallel header verifier
	headers := make([]*types.Header, len(chain))
	headerMap := make(types.HeaderMap, len(chain))

	for i, block := range chain {
		headers[i] = block.Header()
		headerMap[block.Hash()] = block.Header()
	}

	// Peek the error for the first block to decide the directing import logic
	it := newInsertIterator(chain, bc.validator)

	block, err := it.next()

	switch {
	// First block is pruned, insert as sidechain and reorg
	case errors.Is(err, consensus.ErrPrunedAncestor):
		log.Warn("Pruned ancestor, inserting as sidechain", "number", block.Nr(), "hash", block.Hash().Hex())
		return bc.insertSideChain(block, it)

	// Some other error occurred, abort
	case err != nil:
		log.Error("insert chain err", "hash", block.Hash().Hex(), "err", err)
		stats.ignored += len(it.chain)
		return it.index, err
	}
	// No validation errors for the first block (or chain prefix skipped)
	var activeState *state.StateDB
	defer func() {
		// The chain importer is starting and stopping trie prefetchers. If a bad
		// block or other error is hit however, an early return may not properly
		// terminate the background threads. This defer ensures that we clean up
		// and dangling prefetcher, without defering each and holding on live refs.
		if activeState != nil {
			activeState.StopPrefetcher()
		}
	}()

	for ; block != nil && err == nil; block, err = it.next() {
		// If the chain is terminating, stop processing blocks
		if bc.insertStopped() {
			log.Debug("Abort during block processing")
			break
		}

		// Retrieve the parent block, and it's state to execute on top
		start := time.Now()

		rawdb.WriteBlock(bc.db, block)
		rawdb.AddSlotBlockHash(bc.Database(), block.Slot(), block.Hash())
		bc.AppendToChildren(block.Hash(), block.ParentHashes())

		//retrieve state data
		statedb, stateBlock, recommitBlocks, _, stateErr := bc.CollectStateDataByParents(block.ParentHashes())
		if stateErr != nil {
			return it.index, stateErr
		}
		// Enable prefetching to pull in trie node paths while processing transactions
		statedb.StartPrefetcher("chain")
		activeState = statedb

		// recommit red blocks transactions
		for _, bl := range recommitBlocks {
			statedb, _, _, _ = bc.CommitBlockTransactions(bl, statedb)
		}

		// If we have a followup block, run that against the current state to pre-cache
		// transactions and probabilistically some of the account/storage trie nodes.
		var followupInterrupt uint32
		if !bc.cacheConfig.TrieCleanNoPrefetch {
			if followup, err := it.peek(); followup != nil && err == nil {
				throwaway, _ := state.New(stateBlock.Root(), bc.stateCache, bc.snaps)

				go func(start time.Time, followup *types.Block, throwaway *state.StateDB, interrupt *uint32) {
					bc.prefetcher.Prefetch(followup, throwaway, bc.vmConfig, &followupInterrupt)

					blockPrefetchExecuteTimer.Update(time.Since(start))
					if atomic.LoadUint32(interrupt) == 1 {
						blockPrefetchInterruptMeter.Mark(1)
					}
				}(time.Now(), followup, throwaway, &followupInterrupt)
			}
		}
		// Process block using the parent state as reference point
		substart := time.Now()
		receipts, logs, usedGas, err := bc.processor.Process(block, statedb, bc.vmConfig)
		if err != nil {
			atomic.StoreUint32(&followupInterrupt, 1)
			return it.index, err
		}
		// Update the metrics touched during block processing
		accountReadTimer.Update(statedb.AccountReads)                 // Account reads are complete, we can mark them
		storageReadTimer.Update(statedb.StorageReads)                 // Storage reads are complete, we can mark them
		accountUpdateTimer.Update(statedb.AccountUpdates)             // Account updates are complete, we can mark them
		storageUpdateTimer.Update(statedb.StorageUpdates)             // Storage updates are complete, we can mark them
		snapshotAccountReadTimer.Update(statedb.SnapshotAccountReads) // Account reads are complete, we can mark them
		snapshotStorageReadTimer.Update(statedb.SnapshotStorageReads) // Storage reads are complete, we can mark them
		triehash := statedb.AccountHashes + statedb.StorageHashes     // Save to not double count in validation
		trieproc := statedb.SnapshotAccountReads + statedb.AccountReads + statedb.AccountUpdates
		trieproc += statedb.SnapshotStorageReads + statedb.StorageReads + statedb.StorageUpdates

		blockExecutionTimer.Update(time.Since(substart) - trieproc - triehash)

		// Validate the state using the default validator
		substart = time.Now()
		if err := bc.validator.ValidateState(block, statedb, receipts, usedGas); err != nil {
			log.Warn("Red block insertion", "nr", block.Nr(), "height", block.Height(), "slot", block.Slot(), "hash", block.Hash().Hex(), "err", err)
			continue
		}
		proctime := time.Since(start)

		// Update the metrics touched during block validation
		accountHashTimer.Update(statedb.AccountHashes) // Account hashes are complete, we can mark them
		storageHashTimer.Update(statedb.StorageHashes) // Storage hashes are complete, we can mark them

		blockValidationTimer.Update(time.Since(substart) - (statedb.AccountHashes + statedb.StorageHashes - triehash))

		// Write the block to the chain and get the status.
		substart = time.Now()
		status, err := bc.writeBlockWithState(block, receipts, logs, statedb, ET_SKIP, "insertChain")
		atomic.StoreUint32(&followupInterrupt, 1)
		if err != nil {
			return it.index, err
		}
		// Update the metrics touched during block commit
		accountCommitTimer.Update(statedb.AccountCommits)   // Account commits are complete, we can mark them
		storageCommitTimer.Update(statedb.StorageCommits)   // Storage commits are complete, we can mark them
		snapshotCommitTimer.Update(statedb.SnapshotCommits) // Snapshot commits are complete, we can mark them

		blockWriteTimer.Update(time.Since(substart) - statedb.AccountCommits - statedb.StorageCommits - statedb.SnapshotCommits)
		blockInsertTimer.UpdateSince(start)

		switch status {
		case CanonStatTy:
			log.Debug("Inserted new block", "number", block.Nr(), "hash", block.Hash().Hex(),
				"txs", len(block.Transactions()), "gas", block.GasUsed(),
				"elapsed", common.PrettyDuration(time.Since(start)),
				"root", block.Root())

			lastCanon = block

			// Only count canonical blocks for GC processing time
			bc.gcproc += proctime

		case SideStatTy:
			log.Debug("Inserted forked block", "number", block.Nr(), "hash", block.Hash().Hex(),
				"elapsed", common.PrettyDuration(time.Since(start)),
				"txs", len(block.Transactions()), "gas", block.GasUsed(),
				"root", block.Root())

		default:
			// This in theory is impossible, but lets be nice to our future selves and leave
			// a log, instead of trying to track down blocks imports that don't emit logs.
			log.Warn("Inserted block with unknown status", "number", block.Nr(), "hash", block.Hash().Hex(),
				"elapsed", common.PrettyDuration(time.Since(start)),
				"txs", len(block.Transactions()), "gas", block.GasUsed(),
				"root", block.Root())
		}
		stats.processed++
		stats.usedGas += usedGas

		dirty, _ := bc.stateCache.TrieDB().Size()
		stats.report(chain, it.index, dirty)
	}
	stats.ignored += it.remaining()

	return it.index, err
}

// insertSideChain is called when an import batch hits upon a pruned ancestor
// error, which happens when a sidechain with a sufficiently old fork-block is
// found.
//
// The method writes all (header-and-body-valid) blocks to disk, then tries to
// switch over to the new chain
func (bc *BlockChain) insertSideChain(block *types.Block, it *insertIterator) (int, error) {
	// The first sidechain block error is already verified to be ErrPrunedAncestor.
	// Since we don't import them here, we expect ErrUnknownAncestor for the remaining
	// ones. Any other errors means that the block is invalid, and should not be written
	// to disk.
	err := consensus.ErrPrunedAncestor
	for ; block != nil && errors.Is(err, consensus.ErrPrunedAncestor); block, err = it.next() {
		if !bc.HasBlock(block.Hash()) {
			start := time.Now()
			if err := bc.writeBlockWithoutState(block); err != nil {
				return it.index, err
			}
			log.Warn("====== Injected sidechain block (TODO CHECK) ======", "number", block.Nr(), "hash", block.Hash().Hex(),
				"elapsed", common.PrettyDuration(time.Since(start)),
				"txs", len(block.Transactions()), "gas", block.GasUsed(),
				"root", block.Root())
		}
	}
	// At this point, we've written all sidechain blocks to database. Loop ended
	// either on some other error or all were processed. If there was some other
	// error, we can ignore the rest of those blocks.
	//

	// Gather all the sidechain hashes (full blocks may be memory heavy)
	var (
		hashes []common.Hash
	)
	parent := it.previous()
	for parent != nil && !bc.HasState(parent.Root) {
		hashes = append(hashes, parent.Hash())

		parent = bc.GetHeader(parent.ParentHashes[0])
	}
	if parent == nil {
		return it.index, errors.New("missing parent")
	}
	// Import all the pruned blocks to make the state available
	var (
		blocks []*types.Block
		memory common.StorageSize
	)
	for i := len(hashes) - 1; i >= 0; i-- {
		// Append the next block to our batch
		block := bc.GetBlock(hashes[i])

		blocks = append(blocks, block)
		memory += block.Size()

		// If memory use grew too large, import and continue. Sadly we need to discard
		// all raised events and logs from notifications since we're too heavy on the
		// memory here.
		if len(blocks) >= 2048 || memory > 64*1024*1024 {
			log.Info("Importing heavy sidechain segment", "blocks", len(blocks), "start", blocks[0].Hash().Hex(), "end", block.Hash().Hex())
			if _, err := bc.insertChain(blocks); err != nil {
				return 0, err
			}
			blocks, memory = blocks[:0], 0

			// If the chain is terminating, stop processing blocks
			if bc.insertStopped() {
				log.Info("Abort during blocks processing")
				return 0, nil
			}
		}
	}
	if len(blocks) > 0 {
		log.Info("Importing sidechain segment", "start", blocks[0].Nr(), "end", blocks[len(blocks)-1].Nr(), "start", blocks[0].Hash().Hex(), "end", blocks[len(blocks)-1].Hash().Hex())
		return bc.insertChain(blocks)
	}
	return 0, nil
}

// maintainTxIndex is responsible for the construction and deletion of the
// transaction index.
//
// User can use flag `txlookuplimit` to specify a "recentness" block, below
// which ancient tx indices get deleted. If `txlookuplimit` is 0, it means
// all tx indices will be reserved.
//
// The user can adjust the txlookuplimit value for each launch after fast
// sync, Geth will automatically construct the missing indices and delete
// the extra indices.
func (bc *BlockChain) maintainTxIndex(ancients uint64) {
	defer bc.wg.Done()

	// Before starting the actual maintenance, we need to handle a special case,
	// where user might init Geth with an external ancient database. If so, we
	// need to reindex all necessary transactions before starting to process any
	// pruning requests.
	if ancients > 0 {
		var from = uint64(0)
		if bc.txLookupLimit != 0 && ancients > bc.txLookupLimit {
			from = ancients - bc.txLookupLimit
		}
		rawdb.IndexTransactions(bc.db, from, ancients, bc.quit)
	}

	// indexBlocks reindexes or unindexes transactions depending on user configuration
	indexBlocks := func(tail *uint64, head uint64, done chan struct{}) {
		defer func() { done <- struct{}{} }()

		// If the user just upgraded Geth to a new version which supports transaction
		// index pruning, write the new tail and remove anything older.
		if tail == nil {
			if bc.txLookupLimit == 0 || head < bc.txLookupLimit {
				// Nothing to delete, write the tail and return
				rawdb.WriteTxIndexTail(bc.db, 0)
			} else {
				// Prune all stale tx indices and record the tx index tail
				rawdb.UnindexTransactions(bc.db, 0, head-bc.txLookupLimit+1, bc.quit)
			}
			return
		}
		// If a previous indexing existed, make sure that we fill in any missing entries
		if bc.txLookupLimit == 0 || head < bc.txLookupLimit {
			if *tail > 0 {
				rawdb.IndexTransactions(bc.db, 0, *tail, bc.quit)
			}
			return
		}
		// Update the transaction index to the new chain state
		if head-bc.txLookupLimit+1 < *tail {
			// Reindex a part of missing indices and rewind index tail to HEAD-limit
			rawdb.IndexTransactions(bc.db, head-bc.txLookupLimit+1, *tail, bc.quit)
		} else {
			// Unindex a part of stale indices and forward index tail to HEAD-limit
			rawdb.UnindexTransactions(bc.db, *tail, head-bc.txLookupLimit+1, bc.quit)
		}
	}

	// Any reindexing done, start listening to chain events and moving the index window
	var (
		done   chan struct{}                  // Non-nil if background unindexing or reindexing routine is active.
		headCh = make(chan ChainHeadEvent, 1) // Buffered to avoid locking up the event feed
	)
	sub := bc.SubscribeChainHeadEvent(headCh)
	if sub == nil {
		return
	}
	defer sub.Unsubscribe()

	for {
		select {
		case head := <-headCh:
			if done == nil {
				done = make(chan struct{})
				go indexBlocks(rawdb.ReadTxIndexTail(bc.db), head.Block.Nr(), done)
			}
		case <-done:
			done = nil
		case <-bc.quit:
			if done != nil {
				log.Info("Waiting background transaction indexer to exit")
				<-done
			}
			return
		}
	}
}

// InsertHeaderChain attempts to insert the given header chain in to the local
// chain, possibly creating a reorg. If an error is returned, it will return the
// index number of the failing header as well an error describing what went wrong.
//
// The verify parameter can be used to fine tune whether nonce verification
// should be done or not. The reason behind the optional check is because some
// of the header retrieval mechanisms already need to verify nonces, as well as
// because nonces can be verified sparsely, not needing to check each.
func (bc *BlockChain) InsertHeaderChain(chain []*types.Header) (int, error) {
	start := time.Now()
	if i, err := bc.hc.ValidateHeaderChain(chain); err != nil {
		return i, err
	}

	if !bc.chainmu.TryLock() {
		return 0, errChainStopped
	}
	defer bc.chainmu.Unlock()
	_, err := bc.hc.InsertHeaderChain(chain, start)
	return 0, err
}

/********** BlockDAG **********/

// GetDagHashes retrieves all non finalized block's hashes
func (bc *BlockChain) GetDagHashes() *common.HashArray {
	tips := *bc.hc.GetTips()
	aproxLen := len(tips) * int(bc.Config().SlotsPerEpoch) * int(bc.Config().ValidatorsPerSlot)
	ancHashes := make(common.HashArray, 0, aproxLen)
	tipsHashes := make(common.HashArray, 0, len(tips))

	for hash, tip := range tips {
		tipsHashes = append(tipsHashes, hash)
		// if tips block finalized
		tHeader := bc.GetHeader(hash)
		if hash == tip.CpHash || tHeader.Nr() > 0 || hash == bc.Genesis().Hash() {
			continue
		}
		ancHashes = append(ancHashes, tip.OrderedAncestorsHashes...)
	}
	ancHashes.Deduplicate()
	// rm finalized
	dag := make(common.HashArray, 0, len(ancHashes)+len(tipsHashes))
	for _, h := range ancHashes {
		if hdr := bc.GetHeader(h); hdr.Nr() > 0 && hdr.Height > 0 {
			dag = append(dag, h)
		}
	}
	dag = append(dag, tipsHashes...)
	dag.Deduplicate()
	dag = dag.Sort()
	return &dag
}

// GetUnsynchronizedTipsHashes retrieves tips with incomplete chain to finalized state
func (bc *BlockChain) GetUnsynchronizedTipsHashes() common.HashArray {
	tipsHashes := common.HashArray{}
	tips := bc.hc.GetTips()
	for hash, dag := range *tips {
		if dag == nil || dag.CpHash == (common.Hash{}) && dag.Hash != bc.genesisBlock.Hash() {
			tipsHashes = append(tipsHashes, hash)
		}
	}
	return tipsHashes
}

func (bc *BlockChain) WriteCurrentTips() {
	bc.hc.writeCurrentTips(false)
}

type ExploreResult struct {
	unloaded  common.HashArray
	loaded    common.HashArray
	finalized common.HashArray
	graph     *types.GraphDag
	cache     ExploreResultMap
	err       error
}

type ExploreResultMap map[common.Hash]*ExploreResult

// ExploreChainRecursive recursively collect chain info about
// locally unknown, existed and latest finalized parent blocks,
// creates GraphDag structure until latest finalized ancestors.
func (bc *BlockChain) ExploreChainRecursive(headHash common.Hash, memo ...ExploreResultMap) (unloaded, loaded, finalized common.HashArray, graph *types.GraphDag, cache ExploreResultMap, err error) {
	if len(memo) == 0 {
		memo = append(memo, make(ExploreResultMap))
	}
	lfNr := bc.GetLastFinalizedBlock().Nr()

	block := bc.GetBlockByHash(headHash)
	if block == nil {
		return common.HashArray{headHash}, loaded, finalized, graph, memo[0], err
	}
	if block.Nr() > lfNr {
		block.SetNumber(nil)
	}
	graph = &types.GraphDag{
		Hash:   headHash,
		Height: 0,
		Graph:  []*types.GraphDag{},
		State:  types.BSS_NOT_LOADED,
	}
	if block == nil {
		// if block not loaded
		return common.HashArray{headHash}, loaded, finalized, graph, memo[0], nil
	}
	graph.State = types.BSS_LOADED
	graph.Height = block.Height()
	if nr := block.Number(); nr != nil {
		// if finalized
		graph.State = types.BSS_FINALIZED
		graph.Number = *nr
		return unloaded, loaded, common.HashArray{headHash}, graph, memo[0], nil
	}
	loaded = common.HashArray{headHash}
	if block.ParentHashes() == nil || len(block.ParentHashes()) < 1 {
		if block.Hash() == bc.genesisBlock.Hash() {
			return unloaded, loaded, common.HashArray{headHash}, graph, memo[0], nil
		}
		log.Warn("Detect block without parents", "hash", block.Hash().Hex(), "height", block.Height(), "slot", block.Slot())
		err = fmt.Errorf("Detect block without parents hash=%s, height=%d", block.Hash().Hex(), block.Height())
		return unloaded, loaded, finalized, graph, memo[0], err
	}

	//parentHashes := types.GetOrderedParentHashes(bc, block)
	//for _, ph := range parentHashes {
	for _, ph := range block.ParentHashes() {
		var (
			_unloaded  common.HashArray
			_loaded    common.HashArray
			_finalized common.HashArray
			_graph     *types.GraphDag
			_cache     ExploreResultMap
			_err       error
		)

		if memo[0][ph] != nil {
			_unloaded = memo[0][ph].unloaded
			_loaded = memo[0][ph].loaded
			_finalized = memo[0][ph].finalized
			_graph = memo[0][ph].graph
			_cache = memo[0]
			_err = memo[0][ph].err
		} else {
			_unloaded, _loaded, _finalized, _graph, _cache, _err = bc.ExploreChainRecursive(ph, memo[0])
			if memo[0] == nil {
				memo[0] = make(ExploreResultMap, 1)
			}
			memo[0][ph] = &ExploreResult{
				unloaded:  _unloaded,
				loaded:    _loaded,
				finalized: _finalized,
				graph:     _graph,
				cache:     _cache,
				err:       _err,
			}
		}
		unloaded = unloaded.Concat(_unloaded)
		unloaded.Deduplicate()
		loaded = loaded.Concat(_loaded)
		loaded.Deduplicate()
		finalized = finalized.Concat(_finalized)
		finalized.Deduplicate()
		graph.Graph = append(graph.Graph, _graph)
		err = _err
	}
	return unloaded, loaded, finalized, graph, cache, err
}

// CollectAncestorsAftCpByParents recursively collect ancestors by block parents
// which have to be finalized after checkpoint up to block.
func (bc *BlockChain) CollectAncestorsAftCpByParents(parents common.HashArray, cpHash common.Hash) (isCpAncestor bool, ancestors types.HeaderMap, unloaded common.HashArray, err error) {
	cpHeader := bc.GetHeader(cpHash)
	if cpHeader == nil || cpHeader.Height > 0 && cpHeader.Nr() == 0 {
		return false, nil, nil, ErrCpNotFinalized
	}
	return bc.hc.CollectAncestorsAftCpByParents(parents, cpHeader)
}

// CollectAncestorsAftCpByTips recursively collect ancestors by block parents
// which have to be finalized after checkpoint up to block.
func (bc *BlockChain) CollectAncestorsAftCpByTips(parents common.HashArray, cpHash common.Hash) (
	isCpAncestor bool,
	ancestors types.HeaderMap,
	unloaded common.HashArray,
	tips types.Tips,
) {
	return bc.hc.CollectAncestorsAftCpByTips(parents, cpHash)
}

// IsAncestorRecursive checks the passed ancestorHash is an ancestor of the given block.
func (bc *BlockChain) IsAncestorByTips(header *types.Header, ancestorHash common.Hash) (bool, error) {
	if header.Hash() == ancestorHash {
		return false, nil
	}
	//if ancestorHash is genesis
	if bc.genesisBlock.Hash() == ancestorHash {
		return true, nil
	}
	// if ancestorHash in parents
	if header.ParentHashes.Has(ancestorHash) {
		return true, nil
	}
	ancestorHead := bc.GetHeader(ancestorHash)
	if ancestorHead == nil {
		return false, ErrInsertUncompletedDag
	}
	if ancestorHead.Nr() > 0 && ancestorHead.Nr() < header.CpNumber {
		return true, nil
	}
	_, ancestors, unl, _ := bc.CollectAncestorsAftCpByTips(header.ParentHashes, header.CpHash)
	if len(unl) > 0 {
		return false, ErrInsertUncompletedDag
	}
	return ancestors[ancestorHash] != nil, nil
}

// IsAncestorRecursive checks the passed ancestorHash is an ancestor of the given block.
func (bc *BlockChain) IsAncestorRecursive(header *types.Header, ancestorHash common.Hash) (bool, error) {
	if header.Hash() == ancestorHash {
		return false, nil
	}
	//if ancestorHash is genesis
	if bc.genesisBlock.Hash() == ancestorHash {
		return true, nil
	}
	// if ancestorHash in parents
	if header.ParentHashes.Has(ancestorHash) {
		return true, nil
	}
	ancestorHead := bc.GetHeader(ancestorHash)
	if ancestorHead == nil {
		return false, ErrInsertUncompletedDag
	}
	if ancestorHead.Nr() > 0 && ancestorHead.Nr() < header.CpNumber {
		return true, nil
	}
	_, ancestors, unl, err := bc.CollectAncestorsAftCpByParents(header.ParentHashes, header.CpHash)
	if err != nil {
		return false, err
	}
	if len(unl) > 0 {
		return false, ErrInsertUncompletedDag
	}
	return ancestors[ancestorHash] != nil, nil
}

// GetTips retrieves active tips headers:
// - no descendants
// - chained to finalized state (skips unchained)
func (bc *BlockChain) GetTips() types.Tips {
	return *bc.hc.GetTips()
}

// ResetTips set last finalized block to tips for stable work
func (bc *BlockChain) ResetTips() error {
	return bc.hc.ResetTips()
}

// AddTips add BlockDag to tips
func (bc *BlockChain) AddTips(blockDag *types.BlockDAG) {
	bc.hc.AddTips(blockDag)
}

// RemoveTips remove BlockDag from tips by hash from tips
func (bc *BlockChain) RemoveTips(hashes common.HashArray) {
	bc.hc.RemoveTips(hashes)
}

// FinalizeTips update tips in accordance with finalization result
func (bc *BlockChain) FinalizeTips(finHashes common.HashArray, lastFinHash common.Hash, lastFinNr uint64) {
	bc.hc.FinalizeTips(finHashes, lastFinHash, lastFinNr)
}

// AppendToChildren append block hash as child of block
func (bc *BlockChain) AppendToChildren(child common.Hash, parents common.HashArray) {
	batch := bc.db.NewBatch()
	for _, parent := range parents {
		children := rawdb.ReadChildren(bc.db, parent)
		children = append(children, child)
		rawdb.WriteChildren(batch, parent, children)
	}
	if err := batch.Write(); err != nil {
		log.Crit("Failed to write block children", "err", err)
	}
}

// ReadChildren retrieves hashes of the children blocks of given block hash.
func (bc *BlockChain) ReadChildren(hash common.Hash) common.HashArray {
	return rawdb.ReadChildren(bc.db, hash)
}

// GetBlockDag retrieves BlockDag by hash.
func (bc *BlockChain) GetBlockDag(hash common.Hash) *types.BlockDAG {
	return bc.hc.GetBlockDag(hash)
}

func (bc *BlockChain) SaveBlockDag(blockDag *types.BlockDAG) {
	bc.hc.SaveBlockDag(blockDag)
}

// DeleteBlockDag removes BlockDag by hash.
func (bc *BlockChain) DeleteBlockDag(hash common.Hash) {
	bc.hc.DeleteBlockDag(hash)
}

// WriteTxLookupEntry write TxLookupEntry and cache it.
func (bc *BlockChain) WriteTxLookupEntry(txIndex int, txHash, blockHash common.Hash, receiptStatus uint64) bool {
	if receiptStatus == types.ReceiptStatusSuccessful {
		// create transaction lookup
		rawdb.WriteTxLookupEntry(bc.db, txHash, blockHash)
		//cash tx lookup
		lookup := &rawdb.LegacyTxLookupEntry{BlockHash: blockHash, Index: uint64(txIndex)}
		bc.txLookupCache.Add(txHash, lookup)
		return true
	}
	if existed := rawdb.ReadTxLookupEntry(bc.db, txHash); existed == (common.Hash{}) {
		// create transaction lookup
		rawdb.WriteTxLookupEntry(bc.db, txHash, blockHash)
		//cash tx lookup
		lookup := &rawdb.LegacyTxLookupEntry{BlockHash: blockHash, Index: uint64(txIndex)}
		bc.txLookupCache.Add(txHash, lookup)
		return true
	}
	return false
}

func (bc *BlockChain) moveTxsToProcessing(txs *types.BlockTransactions) {
	bc.processingFeed.Send(txs)
}

func (bc *BlockChain) MoveTxsToProcessing(block *types.Block) {
	if block == nil {
		return
	}

	txs := types.NewBlockTransactions(block.Hash())
	bc.handleBlockValidatorSyncTxs(block)
	txs.Transactions = append(txs.Transactions, block.Transactions()...)

	sort.Slice(txs.Transactions, func(i, j int) bool {
		return txs.Transactions[i].Nonce() < txs.Transactions[j].Nonce()
	})

	bc.moveTxsToProcessing(txs)
}

func (bc *BlockChain) RemoveTxsFromPool(tx types.Transactions) {
	bc.rmTxFeed.Send(tx)
}

/* synchronization functionality */

// todo RM/check
// SetSyncProvider set provider of access to synchronization functionality
func (bc *BlockChain) SetSyncProvider(provider types.SyncProvider) {
	bc.syncProvider = provider
}

// Synchronising returns whether the downloader is currently synchronising.
func (bc *BlockChain) Synchronising() bool {
	return !bc.isSynced
}

// FinSynchronising returns whether the downloader is currently retrieving finalized blocks.
func (bc *BlockChain) FinSynchronising() bool {
	if bc.syncProvider == nil {
		return false
	}
	return bc.syncProvider.FinSynchronising()
}

// DagSynchronising returns whether the downloader is currently retrieving dag chain blocks.
func (bc *BlockChain) DagSynchronising() bool {
	if bc.syncProvider == nil {
		return false
	}
	return bc.syncProvider.DagSynchronising()
}

func (bc *BlockChain) ValidatorStorage() valStore.Storage {
	return bc.validatorStorage
}

func (bc *BlockChain) GetEraInfo() *era.EraInfo {
	return &bc.eraInfo
}

// SetNewEraInfo sets new era info.
func (bc *BlockChain) SetNewEraInfo(newEra era.Era) {
	log.Info("New era",
		"num", newEra.Number,
		"begin", newEra.From,
		"end", newEra.To,
		"root", newEra.Root,
	)

	bc.eraInfo = era.NewEraInfo(newEra)
}

func (bc *BlockChain) Database() ethdb.Database {
	return bc.db
}

func (bc *BlockChain) DagMuLock() {
	bc.dagMu.Lock()
}

func (bc *BlockChain) DagMuUnlock() {
	bc.dagMu.Unlock()
}

func (bc *BlockChain) EnterNextEra(nextEraEpochFrom uint64, root common.Hash) *era.Era {
	nextEra := rawdb.ReadEra(bc.db, bc.eraInfo.Number()+1)
	if nextEra != nil {
		rawdb.WriteCurrentEra(bc.db, nextEra.Number)
		log.Info("######### if nextEra != nil EnterNextEra",
			"num", nextEra.Number,
			"begin", nextEra.From,
			"end", nextEra.To,
			"root", nextEra.Root,
			"currSlot", bc.GetSlotInfo().CurrentSlot(),
			"currEpoch", bc.GetSlotInfo().SlotToEpoch(bc.GetSlotInfo().CurrentSlot()),
		)
		bc.SetNewEraInfo(*nextEra)
		return nextEra
	}

	transitionSlot, err := bc.GetSlotInfo().SlotOfEpochStart(nextEraEpochFrom - bc.Config().TransitionPeriod)
	if err != nil {
		log.Error("Next era: calculate transition slot failed", "err", err)
	}

	validators, _ := bc.ValidatorStorage().GetValidators(bc, transitionSlot, true, false, "EnterNextEra")
	nextEra = era.NextEra(bc, root, uint64(len(validators)))
	rawdb.WriteEra(bc.db, nextEra.Number, *nextEra)
	rawdb.WriteCurrentEra(bc.db, nextEra.Number)
	log.Info("######### if nextEra == nil EnterNextEra",
		"num", nextEra.Number,
		"begin", nextEra.From,
		"end", nextEra.To,
		"root", nextEra.Root,
		"currSlot", bc.GetSlotInfo().CurrentSlot(),
		"currEpoch", bc.GetSlotInfo().SlotToEpoch(bc.GetSlotInfo().CurrentSlot()),
		"validators", validators,
	)
	bc.SetNewEraInfo(*nextEra)
	return nextEra
}

func (bc *BlockChain) StartTransitionPeriod(cp *types.Checkpoint, spineRoot common.Hash) {
	nextEra := rawdb.ReadEra(bc.db, bc.eraInfo.Number()+1)
	if nextEra == nil {
		log.Info("GetValidators StartTransitionPeriod", "slot", bc.GetSlotInfo().CurrentSlot(),
			"curEpoch", bc.GetSlotInfo().SlotToEpoch(bc.GetSlotInfo().CurrentSlot()),
			"curEra", bc.GetEraInfo().GetEra().Number,
			"fromEp", bc.GetEraInfo().GetEra().From,
			"toEp", bc.GetEraInfo().GetEra().To,
			"nextEraFirstEpoch", bc.GetEraInfo().NextEraFirstEpoch(),
			"nextEraFirstSlot", bc.GetEraInfo().NextEraFirstSlot(bc),
		)

		cpEpochSlot, err := bc.GetSlotInfo().SlotOfEpochStart(cp.FinEpoch - bc.Config().TransitionPeriod)
		if err != nil {
			panic("StartTransitionPeriod slot of epoch start error")
		}

		validators, _ := bc.ValidatorStorage().GetValidators(bc, cpEpochSlot, true, false, "StartTransitionPeriod")
		nextEra := era.NextEra(bc, spineRoot, uint64(len(validators)))

		rawdb.WriteEra(bc.db, nextEra.Number, *nextEra)

		log.Info("Era transition period", "from", bc.GetEraInfo().Number(), "num", nextEra.Number, "begin", nextEra.From, "end", nextEra.To, "length", nextEra.Length())
	} else {
		log.Info("######## HandleEra transitionPeriod skipped already done", "cpEpoch", cp.Epoch,
			"cpFinEpoch", cp.FinEpoch,
			"curEpoch", bc.GetSlotInfo().SlotInEpoch(bc.GetSlotInfo().CurrentSlot()),
			"curSlot", bc.GetSlotInfo().CurrentSlot(),
			"bc.GetEraInfo().ToEpoch", bc.GetEraInfo().ToEpoch(),
			"bc.GetEraInfo().FromEpoch", bc.GetEraInfo().FromEpoch(),
			"bc.GetEraInfo().Number", bc.GetEraInfo().Number(),
		)
	}
}

func (bc *BlockChain) IsTxValidatorSync(tx *types.Transaction) bool {
	if tx == nil {
		return false
	}
	return tx.To() != nil && bc.Config().ValidatorsStateAddress != nil && bytes.Equal(tx.To().Bytes(), bc.Config().ValidatorsStateAddress.Bytes())
}

func (bc *BlockChain) verifyBlockValidatorSyncTx(block *types.Block, tx *types.Transaction) error {
	if !bc.IsTxValidatorSync(tx) {
		return nil
	}
	op, err := validatorOp.DecodeBytes(tx.Data())
	if err != nil {
		log.Error("can`t unmarshal validator sync operation from tx data", "err", err)
		return err
	}

	switch v := op.(type) {
	case validatorOp.ValidatorSync:
		validator.ValidateValidatorSyncOp(bc, v, block.Slot(), tx.Hash())
	}
	return nil
}

func (bc *BlockChain) handleBlockValidatorSyncTxs(block *types.Block) {
	for _, tx := range block.Transactions() {
		if !bc.IsTxValidatorSync(tx) {
			continue
		}
		if err := bc.verifyBlockValidatorSyncTx(block, tx); err != nil {
			log.Error("Validator sync tx handling",
				"err", err,
				"tx.Hash", tx.Hash().Hex(),
				"tx.To", tx.To().Hex(),
				"ValidatorsStateAddress", bc.Config().ValidatorsStateAddress.Bytes(),
				"condIsValSync", bytes.Equal(tx.To().Bytes(), bc.Config().ValidatorsStateAddress.Bytes()),
			)
			continue
		}

		op, err := validatorOp.DecodeBytes(tx.Data())
		if err != nil {
			log.Error("can`t unmarshal validator sync operation from tx data", "err", err)
			continue
		}
		switch v := op.(type) {
		case validatorOp.ValidatorSync:
			txHash := tx.Hash()
			txValSyncOp := &types.ValidatorSync{
				InitTxHash: v.InitTxHash(),
				OpType:     v.OpType(),
				ProcEpoch:  v.ProcEpoch(),
				Index:      v.Index(),
				Creator:    v.Creator(),
				Amount:     v.Amount(),
				TxHash:     &txHash,
			}
			log.Info("Validator sync tx",
				"OpType", txValSyncOp.OpType,
				"ProcEpoch", txValSyncOp.ProcEpoch,
				"Index", txValSyncOp.Index,
				"Creator", fmt.Sprintf("%#x", txValSyncOp.Creator),
				"amount", txValSyncOp.Amount,
				"TxHash", fmt.Sprintf("%#x", txValSyncOp.TxHash),
				"InitTxHash", fmt.Sprintf("%#x", txValSyncOp.InitTxHash),
			)
			bc.SetValidatorSyncData(txValSyncOp)
		}
	}
}

func (bc *BlockChain) handleBlockValidatorSyncReceipts(block *types.Block, receipts types.Receipts) {
	for _, receipt := range receipts {
		if receipt == nil {
			continue
		}
		if receipt.Status != types.ReceiptStatusSuccessful {
			continue
		}
		tx := block.Transactions()[receipt.TransactionIndex]
		if !bc.IsTxValidatorSync(tx) {
			continue
		}

		log.Info("Validator sync tx receipts (start)",
			"tx.Hash", tx.Hash().Hex(),
			"rc.Hash", receipt.TxHash.Hex(),
			"conditionHash", tx.Hash() == receipt.TxHash,
			"tx.To", tx.To().Hex(),
			"ValidatorsStateAddress", bc.Config().ValidatorsStateAddress.Hex(),
			"condIsValSync", bytes.Equal(tx.To().Bytes(), bc.Config().ValidatorsStateAddress.Bytes()),
			"rc.Status", receipt.Status,
			"conditionStatus", receipt.Status != types.ReceiptStatusSuccessful,
		)

		op, err := validatorOp.DecodeBytes(tx.Data())
		if err != nil {
			log.Error("can`t unmarshal validator sync operation from tx data", "err", err)
			continue
		}

		switch v := op.(type) {
		case validatorOp.ValidatorSync:
			txHash := tx.Hash()
			txValSyncOp := &types.ValidatorSync{
				InitTxHash: v.InitTxHash(),
				OpType:     v.OpType(),
				ProcEpoch:  v.ProcEpoch(),
				Index:      v.Index(),
				Creator:    v.Creator(),
				Amount:     v.Amount(),
				TxHash:     &txHash,
			}
			log.Info("Validator sync tx receipts",
				"OpType", txValSyncOp.OpType,
				"ProcEpoch", txValSyncOp.ProcEpoch,
				"Index", txValSyncOp.Index,
				"Creator", fmt.Sprintf("%#x", txValSyncOp.Creator),
				"amount", txValSyncOp.Amount,
				"TxHash", fmt.Sprintf("%#x", txValSyncOp.TxHash),
				"InitTxHash", txValSyncOp.InitTxHash.Hex(),
			)
			bc.SetValidatorSyncData(txValSyncOp)
		}
	}
}

func (bc *BlockChain) SetOptimisticSpinesToCache(slot uint64, spines common.HashArray) {
	bc.optimisticSpinesCache.Add(slot, spines)
}

func (bc *BlockChain) GetOptimisticSpinesFromCache(slot uint64) common.HashArray {
	blocks, ok := bc.optimisticSpinesCache.Get(slot)
	if !ok {
		return nil
	}

	blk := blocks.(common.HashArray)
	if blk != nil {
		return blk
	}

	return nil
}

func (bc *BlockChain) removeOptimisticSpinesFromCache(slot uint64) {
	bc.optimisticSpinesCache.Remove(slot)
}

// SetIsSynced sets the value of isSynced with mu
func (bc *BlockChain) SetIsSynced(synced bool) {
	bc.isSyncedM.Lock()
	defer bc.isSyncedM.Unlock()

	log.Info("Switched sync status to", "isSynced", synced)
	bc.isSynced = synced
}

// IsSynced gets the value of isSynced with mu
func (bc *BlockChain) IsSynced() bool {
	bc.isSyncedM.Lock()
	defer bc.isSyncedM.Unlock()
	return bc.isSynced
}

func (bc *BlockChain) GetOptimisticSpines(gtSlot uint64) ([]common.HashArray, error) {
	currentSlot := bc.GetTips().GetMaxSlot()
	if currentSlot <= gtSlot {
		return []common.HashArray{}, nil
	}

	var err error
	optimisticSpines := make([]common.HashArray, 0)

	for slot := gtSlot + 1; slot <= currentSlot; slot++ {
		slotSpines := bc.GetOptimisticSpinesFromCache(slot)
		if slotSpines == nil {
			slotBlocks := make(types.Headers, 0)
			slotBlocksHashes := rawdb.ReadSlotBlocksHashes(bc.Database(), slot)
			for _, hash := range slotBlocksHashes {
				block := bc.GetHeader(hash)
				slotBlocks = append(slotBlocks, block)
			}
			slotSpines, err = types.CalculateOptimisticSpines(slotBlocks)
			if err != nil {
				return []common.HashArray{}, err
			}
			bc.SetOptimisticSpinesToCache(slot, slotSpines)
		}

		// Only append slotSpines if it's not empty
		if len(slotSpines) > 0 {
			optimisticSpines = append(optimisticSpines, slotSpines)
		}
	}

	return optimisticSpines, nil
}

func (bc *BlockChain) EpochToEra(epoch uint64) *era.Era {
	curEra := bc.eraInfo.GetEra()
	if curEra == nil {
		currentEraNumber := rawdb.ReadCurrentEra(bc.db)
		if eraInfo := rawdb.ReadEra(bc.db, currentEraNumber); eraInfo != nil {
			bc.SetNewEraInfo(*eraInfo)
			curEra = eraInfo
		} else {
			return nil
		}
	}
	if curEra.IsContainsEpoch(epoch) {
		return curEra
	}

	eraNumber := curEra.Number
	findingEra := curEra

	if epoch < curEra.From {
		low := uint64(0)
		high := eraNumber
		for !findingEra.IsContainsEpoch(epoch) {
			eraNumber = (low + high) / 2
			findingEra = rawdb.ReadEra(bc.db, eraNumber)
			if findingEra.To > epoch {
				high = eraNumber - 1
			} else {
				low = eraNumber + 1
			}
		}
	} else if epoch > curEra.To {
		for !findingEra.IsContainsEpoch(epoch) {
			eraNumber++
			findingEra = rawdb.ReadEra(bc.db, eraNumber)
		}
	}

	return findingEra
}

func (bc *BlockChain) HaveEpochBlocks(epoch uint64) (bool, error) {
	startSlot, err := bc.GetSlotInfo().SlotOfEpochStart(epoch)
	if err != nil {
		return false, err
	}

	endSlot, err := bc.GetSlotInfo().SlotOfEpochEnd(epoch)
	if err != nil {
		return false, err
	}

	for startSlot <= endSlot {
		blocks := bc.GetBlockHashesBySlot(startSlot)
		if len(blocks) > 0 {
			return true, nil
		}
		startSlot++
	}

	return false, nil
}

func (bc *BlockChain) verifyBlockBaseFee(block *types.Block) bool {
	creatorsPerSlotCount := bc.Config().ValidatorsPerSlot
	if creatorsPerSlot, err := bc.ValidatorStorage().GetCreatorsBySlot(bc, block.Slot()); err == nil {
		creatorsPerSlotCount = uint64(len(creatorsPerSlot))
	}

	validators, _ := bc.ValidatorStorage().GetValidators(bc, block.Slot(), true, false, "verifyBlockBaseFee")
	expectedBaseFee := misc.CalcSlotBaseFee(bc.Config(), creatorsPerSlotCount, uint64(len(validators)), bc.Genesis().GasLimit())

	if expectedBaseFee.Cmp(block.BaseFee()) != 0 {
		log.Warn("Block verification: invalid base fee",
			"calcBaseFee", expectedBaseFee.String(),
			"blockBaseFee", block.BaseFee().String(),
			"blockHash", block.Hash().Hex(),
		)
		return false
	}

	return true
}

func (bc *BlockChain) GetEVM(msg Message, state *state.StateDB, header *types.Header, vmConfig *vm.Config) (*vm.EVM, func() error, error) {
	vmError := func() error { return nil }
	if vmConfig == nil {
		vmConfig = bc.GetVMConfig()
	}
	txContext := NewEVMTxContext(msg)
	context := NewEVMBlockContext(header, bc, nil)
	return vm.NewEVM(context, txContext, state, bc.Config(), *vmConfig), vmError, nil
}

// GetTP retrieves the token processor.
func (bc *BlockChain) GetTP(state *state.StateDB, header *types.Header) (*token.Processor, func() error, error) {
	tpError := func() error { return nil }
	context := NewEVMBlockContext(header, bc, nil)
	return token.NewProcessor(context, state), tpError, nil
}

// GetVP retrieves the validator processor.
func (bc *BlockChain) GetVP(state *state.StateDB, header *types.Header) (*validator.Processor, func() error, error) {
	tpError := func() error { return nil }
	context := NewEVMBlockContext(header, bc, nil)
	return validator.NewProcessor(context, state, bc), tpError, nil
}

func (bc *BlockChain) doCall(msg Message, header *types.Header, tp *token.Processor, vp *validator.Processor) (*ExecutionResult, error) {
	defer func(start time.Time) { log.Info("Executing EVM call finished", "runtime", time.Since(start)) }(time.Now())

	state, err := bc.StateAt(header.Root)
	if state == nil || err != nil {
		return nil, err
	}

	evm, _, err := bc.GetEVM(msg, state, header, &vm.Config{NoBaseFee: true})
	if err != nil {
		return nil, err
	}
	// Wait for the context to be done and cancel the evm. Even if the
	// EVM has finished, cancelling may be done (repeatedly)
	defer evm.Cancel()

	// Execute the message.
	gp := new(GasPool).AddGas(math.MaxUint64)
	result, err := ApplyMessage(evm, tp, vp, msg, gp)

	if evm.Cancelled() {
		return nil, fmt.Errorf("execution aborted")
	}
	if err != nil {
		return result, fmt.Errorf("err: %w (supplied gas %d)", err, msg.Gas())
	}
	return result, nil
}

// searchValidCheckpoint descending search of the first valid checkpoint
func (bc *BlockChain) searchValidCheckpoint(fromEpoch uint64) *types.Checkpoint {
	for ; fromEpoch >= 0; fromEpoch-- {
		cpSpine := rawdb.ReadEpoch(bc.db, fromEpoch)
		if cpSpine == (common.Hash{}) {
			if fromEpoch == 0 {
				return nil
			}
			continue
		}
		cp := rawdb.ReadCoordinatedCheckpoint(bc.db, cpSpine)
		if cp == nil {
			if fromEpoch == 0 {
				return nil
			}
			continue
		}
		return cp
	}
	return nil
}

// searchBlockFinalizationCp searching for checkpoint of block finalization by number.
func (bc *BlockChain) searchBlockFinalizationCp(hdr *types.Header) *types.Checkpoint {
	if hdr == nil || hdr.Number == nil {
		return nil
	}
	nr := hdr.Nr()
	if nr > 0 {
		nr--
	}
	for ; nr >= hdr.CpNumber; nr-- {
		cpSpine := rawdb.ReadFinalizedHashByNumber(bc.db, nr)
		if cpSpine == (common.Hash{}) {
			return nil
		}
		cp := rawdb.ReadCoordinatedCheckpoint(bc.db, cpSpine)
		if cp == nil {
			if nr == 0 {
				return nil
			}
			continue
		}
		return cp
	}
	return nil
}
