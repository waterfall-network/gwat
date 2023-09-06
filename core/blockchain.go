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

	var nilBlock *types.Block = bc.genesisBlock
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

	err = bc.rollbackCheckpoint()
	if err != nil {
		return nil, err
	}

	if err := bc.loadLastState(); err != nil {
		return nil, err
	}

	// Make sure the state associated with the block is available
	head := bc.GetLastFinalizedBlock()
	if _, err := state.New(head.Root(), bc.stateCache, bc.snaps); err != nil {
		// Head state is missing, before the state recovery, find out the
		// disk layer point of snapshot(if it's enabled). Make sure the
		// rewound point is lower than disk layer.
		var diskRoot common.Hash
		if bc.cacheConfig.SnapshotLimit > 0 {
			//todo uncomment
			//diskRoot = rawdb.ReadSnapshotRoot(bc.db)
		}
		if diskRoot != (common.Hash{}) {
			log.Warn("Head state missing, repairing", "number", head.Nr(), "hash", head.Hash().Hex(), "snaproot", diskRoot)
			snapDisk, err := bc.SetHeadBeyondRoot(head.Hash(), diskRoot)
			if err != nil {
				return nil, err
			}
			// Chain rewound, persist old snapshot number to indicate recovery procedure
			if snapDisk != 0 {
				rawdb.WriteSnapshotRecoveryNumber(bc.db, snapDisk)
			}
		} else {
			log.Warn("Head state missing, repairing", "number", head.Nr(), "hash", head.Hash())
			prevStateHeader := bc.SearchPrevFinalizedBlueHeader(head.Nr())
			if prevStateHeader != nil {
				head = bc.GetBlock(prevStateHeader.Hash())
			}
			if err := bc.SetHead(head.Hash()); err != nil {
				return nil, err
			}
			bc.hc.ResetTips()
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

	// Check the current state of the block hashes and make sure that we do not have any of the bad blocks in our chain
	for hash := range BadHashes {
		if header := bc.GetHeaderByHash(hash); header != nil {
			// get the canonical block corresponding to the offending header's number
			headerByNumber := bc.GetHeaderByNumber(header.Nr())
			// make sure the headerByNumber (if present) is in our current canonical chain
			if headerByNumber != nil && headerByNumber.Hash() == header.Hash() {
				log.Error("Found bad hash, rewinding chain", "number", header.Nr(), "hash", header.Hash().Hex())
				panic("Bad blocks handling implementation required")
				//todo set last finalized block and block dag
				//if err := bc.SetHead(header.Number.Uint64() - 1); err != nil {
				//	return nil, err
				//}
				//log.Error("Chain rewind was successful, resuming normal operation")
			}
		}
	}

	// Load any existing snapshot, regenerating it if loading failed
	if bc.cacheConfig.SnapshotLimit > 0 {
		// If the chain was rewound past the snapshot persistent layer (causing
		// a recovery block number to be persisted to disk), check if we're still
		// in recovery mode and in that case, don't invalidate the snapshot on a
		// head mismatch.
		var recover bool

		head := bc.GetLastFinalizedBlock()
		if layer := rawdb.ReadSnapshotRecoveryNumber(bc.db); layer != nil && *layer > head.Nr() {
			log.Warn("Enabling snapshot recovery", "chainhead", head.Nr(), "diskbase", *layer)
			recover = true
		}
		bc.snaps, _ = snapshot.New(bc.db, bc.stateCache.TrieDB(), bc.cacheConfig.SnapshotLimit, head.Root(), !bc.cacheConfig.SnapshotWait, true, recover)
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

	// Initial coordinator sync node status
	bc.SetIsSynced(false)

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
		// Corrupt or empty database, init from scratch
		log.Warn("Empty database, resetting chain")
		return bc.Reset()
	}
	// Make sure the entire head block is available
	lastFinalisedBlock := bc.GetBlockByHash(lastFinHash)
	if lastFinalisedBlock == nil {
		// Corrupt or empty database, init from scratch
		log.Warn("Head block missing, resetting chain", "hash", lastFinHash.Hex())
		return bc.Reset()
	}

	//remove finalized numbers that more than last one
	rmNr := bc.GetLastFinalizedNumber() + 1
	for {
		rmHash := rawdb.ReadFinalizedHashByNumber(bc.db, rmNr)
		if rmHash == (common.Hash{}) {
			break
		}
		rawdb.DeleteFinalizedHashNumber(bc.db, rmHash, rmNr)
		rmNr++
	}

	// Everything seems to be fine, set as the lastFinHash block
	bc.lastFinalizedBlock.Store(lastFinalisedBlock)
	headBlockGauge.Update(int64(lastFinNr))

	// Restore the last known head header
	lastFinalisedHeader := lastFinalisedBlock.Header()
	if lastFinHash != (common.Hash{}) {
		if header := bc.GetHeaderByHash(lastFinHash); header != nil {
			lastFinalisedHeader = header
		}
	}
	bc.hc.SetLastFinalisedHeader(lastFinalisedHeader, lastFinNr)

	// Restore the last known lastFinHash fast block
	bc.lastFinalizedFastBlock.Store(lastFinalisedBlock)
	headFastBlockGauge.Update(int64(lastFinNr))

	if head := rawdb.ReadHeadFastBlockHash(bc.db); head != (common.Hash{}) {
		if block := bc.GetBlockByHash(head); block != nil {
			bc.lastFinalizedFastBlock.Store(block)
			headFastBlockGauge.Update(int64(lastFinNr))
		}
	}

	//load BlockDag
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

	log.Info("Loaded tips", "hashes", tips.GetHashes())
	log.Info("Loaded most recent local header", "hash", lastFinalisedHeader.Hash(), "finNr", lastFinNr)
	log.Info("Loaded most recent local full block", "hash", lastFinalisedBlock.Hash(), "finNr", lastFinNr)
	log.Info("Loaded most recent local fast block", "hash", lastFinalizedFastBlocks.Hash(), "finNr", lastFinNr)
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
	if epCp := bc.GetCoordinatedCheckpoint(cp.Spine); epCp == nil {
		rawdb.WriteCoordinatedCheckpoint(bc.db, cp)
		bc.checkpointCache.Add(cp.Spine, cp)
	}
	//update current cp and apoch data.
	currCp := bc.GetLastCoordinatedCheckpoint()
	if currCp == nil || cp.Root != currCp.Root || cp.FinEpoch != currCp.FinEpoch {
		bc.lastCoordinatedCp.Store(cp.Copy())
		rawdb.WriteLastCoordinatedCheckpoint(bc.db, cp)
		log.Info("CheckShuffle - write checkpoint to db", "epoch", cp.FinEpoch, "checkpoint", cp)
		rawdb.WriteEpoch(bc.db, cp.FinEpoch, cp.Spine)
		log.Info("Update coordinated checkpoint ", "cp", cp)
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

func (bc *BlockChain) rollbackCheckpoint() error {
	lastCp := bc.GetLastCoordinatedCheckpoint()
	if lastCp == nil {
		return errors.New("no finalized checkpoint")
	}

	prevEpoch := bc.GetEpoch(lastCp.FinEpoch - 1)
	prevCp := bc.GetCoordinatedCheckpoint(prevEpoch)
	if prevCp != nil {
		bc.SetLastCoordinatedCheckpoint(prevCp)
		lastFinNum := bc.GetLastFinalizedNumber()
		rollbackFinNum := bc.GetHeader(prevCp.Spine).Nr()
		for i := lastFinNum; i > rollbackFinNum; i-- {
			err := bc.RollbackFinalization(i)
			if err != nil {
				return err
			}
		}

		rawdb.DeleteCoordinatedCheckpoint(bc.db, lastCp.Spine)
		rawdb.WriteLastFinalizedHash(bc.db, prevCp.Spine)

		dagHashes := bc.GetDagHashes()
		if dagHashes != nil {
			for _, hash := range *dagHashes {
				bc.DeleteBlockDag(hash)
			}
		}

		bc.SetIsSynced(false)

		log.Info("Rollback checkpoint",
			"checkpoint", lastCp,
			"prevCheckpoint", prevCp,
		)
	}

	return nil
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

// SetHead rewinds the local chain to a new head. Depending on whether the node
// was fast synced or full synced and in which state, the method will try to
// delete minimal data from disk whilst retaining chain consistency.
func (bc *BlockChain) SetHead(head common.Hash) error {
	_, err := bc.SetHeadBeyondRoot(head, common.Hash{})
	return err
}

// SetHeadBeyondRoot rewinds the local chain to a new head with the extra condition
// that the rewind must pass the specified state root. This method is meant to be
// used when rewinding with snapshots enabled to ensure that we go back further than
// persistent disk layer. Depending on whether the node was fast synced or full, and
// in which state, the method will try to delete minimal data from disk whilst
// retaining chain consistency.
//
// The method returns the block number where the requested root cap was found.
func (bc *BlockChain) SetHeadBeyondRoot(head common.Hash, root common.Hash) (uint64, error) {
	if !bc.chainmu.TryLock() {
		return 0, errChainStopped
	}
	defer bc.chainmu.Unlock()

	hdr := rawdb.ReadHeader(bc.db, head)
	if hdr.Nr() != hdr.Height {
		currNr := hdr.Nr()
		if hdr.Number == nil {
			currNr = rawdb.ReadLastFinalizedNumber(bc.db)
		}
		for currNr > 0 {
			currHash := rawdb.ReadFinalizedHashByNumber(bc.db, currNr)
			if currHash != (common.Hash{}) {
				currHeader := rawdb.ReadHeader(bc.db, currHash)
				if currHeader.Nr() == currHeader.Height {
					head = currHeader.Hash()
					//root = currHeader.Root
					break
				}
			}
			currNr--
		}
	}

	// Track the block number of the requested root hash
	var rootNumber uint64 // (no root == always 0)

	// Retrieve the last pivot block to short circuit rollbacks beyond it and the
	// current freezer limit to start nuking id underflown
	pivot := rawdb.ReadLastPivotNumber(bc.db)
	frozen, _ := bc.db.Ancients()

	updateFn := func(db ethdb.KeyValueWriter, header *types.Header) (common.Hash, bool) {
		// Rewind the block chain, ensuring we don't end up with a stateless head
		// block. Note, depth equality is permitted to allow using SetHead as a
		// chain reparation mechanism without deleting any data!
		currentBlock := bc.GetLastFinalizedBlock()
		blHeigt := bc.GetBlockFinalizedNumber(currentBlock.Hash())
		headerHeight := rawdb.ReadFinalizedNumberByHash(bc.db, header.Hash())
		if currentBlock != nil && headerHeight != nil && blHeigt != nil && *headerHeight <= *blHeigt {
			newHeadBlock := bc.GetBlock(header.Hash())
			if newHeadBlock == nil {
				log.Error("Gap in the chain, rewinding to genesis", "number", headerHeight, "hash", header.Hash())
				newHeadBlock = bc.genesisBlock
			} else {
				// Block exists, keep rewinding until we find one with state,
				// keeping rewinding until we exceed the optional threshold
				// root hash
				beyondRoot := (root == common.Hash{}) // Flag whether we're beyond the requested root (no root, always true)

				rootNumber = newHeadBlock.Nr()

				for {
					// If a root threshold was requested but not yet crossed, check
					if root != (common.Hash{}) && !beyondRoot && newHeadBlock.Root() == root {
						beyondRoot = true
						rootNumber = uint64(0)
						if bh := bc.GetBlockFinalizedNumber(newHeadBlock.Hash()); bh != nil {
							rootNumber = *bh
						}
					}

					if newHeadBlock.Nr() != newHeadBlock.Height() {
						parent := bc.GetBlockByNumber(rootNumber - 1)
						if parent != nil {
							newHeadBlock = parent
							rootNumber = newHeadBlock.Nr()
							rawdb.DeleteChildren(db, parent.Hash())
							bc.DeleteBlockDag(parent.Hash())
							continue
						}
					}
					if _, err := state.New(newHeadBlock.Root(), bc.stateCache, bc.snaps); err != nil {
						log.Warn("Block state missing, rewinding further", "number", rootNumber, "hash", newHeadBlock.Hash())
						if pivot == nil || newHeadBlock.Nr() > *pivot {
							parent := bc.GetBlockByNumber(rootNumber - 1)
							if parent != nil {
								newHeadBlock = parent
								rootNumber = newHeadBlock.Nr()
								rawdb.DeleteChildren(db, parent.Hash())
								bc.DeleteBlockDag(parent.Hash())
								continue
							}
							log.Error("Missing block in the middle, aiming genesis", "number", rootNumber-1)
							newHeadBlock = bc.genesisBlock
						} else {
							log.Trace("Rewind passed pivot, aiming genesis", "number", rootNumber, "hash", newHeadBlock.Hash(), "pivot", *pivot)
							newHeadBlock = bc.genesisBlock
						}
					}
					if beyondRoot || rootNumber == 0 {
						log.Debug("Rewound to block with state", "number", rootNumber, "hash", newHeadBlock.Hash().Hex())
						break
					}
					log.Debug("Skipping block with threshold state", "number", rootNumber, "hash", newHeadBlock.Hash(), "root", newHeadBlock.Root())
					newHeadBlock = bc.GetBlockByNumber(newHeadBlock.Nr()) // Keep rewinding
					if rootNumber == newHeadBlock.Nr() {
						newHeadBlock = bc.GetBlockByNumber(newHeadBlock.Nr() - 1)
						log.Warn("set head beyond root: check next root", "rootNumber", rootNumber, "nr", newHeadBlock.Nr(), "hash", newHeadBlock.Hash().Hex())
					}
				}
			}
			rawdb.WriteLastCanonicalHash(db, newHeadBlock.Hash())
			rawdb.WriteLastFinalizedHash(db, newHeadBlock.Hash())

			// Degrade the chain markers if they are explicitly reverted.
			// In theory we should update all in-memory markers in the
			// last step, however the direction of SetHead is from high
			// to low, so it's safe the update in-memory markers directly.
			bc.lastFinalizedBlock.Store(newHeadBlock)
			headBlockGauge.Update(int64(rootNumber))

			// update tips
			//bc.ResetTips()
			newBlockDag := bc.GetBlockDag(newHeadBlock.CpHash())
			if newBlockDag == nil {
				cpHeader := bc.GetHeader(newHeadBlock.CpHash())
				_, ancestors, _, err := bc.CollectAncestorsAftCpByParents(newHeadBlock.ParentHashes(), newHeadBlock.CpHash())
				if err != nil {
					log.Error("Set head beyond root: collact ancestors failed", "number", headerHeight, "hash", header.Hash())
				}
				delete(ancestors, cpHeader.Hash())
				newBlockDag = &types.BlockDAG{
					Hash:                   newHeadBlock.Hash(),
					Height:                 newHeadBlock.Height(),
					Slot:                   newHeadBlock.Slot(),
					CpHash:                 newHeadBlock.CpHash(),
					CpHeight:               cpHeader.Height,
					OrderedAncestorsHashes: ancestors.Hashes(),
				}
			}
			bc.AddTips(newBlockDag)
			bc.WriteCurrentTips()
		}
		// Rewind the fast block in a simpleton way to the target head
		if lastFinalizedFastBlock := bc.GetLastFinalizedFastBlock(); lastFinalizedFastBlock != nil && header.Nr() < lastFinalizedFastBlock.Nr() {
			curFinHeight := rawdb.ReadFinalizedNumberByHash(bc.db, lastFinalizedFastBlock.Hash())
			if curFinHeight != nil && headerHeight != nil && *headerHeight < *curFinHeight {
				newHeadFastBlock := bc.GetBlock(header.Hash())
				// If either blocks reached nil, reset to the genesis state
				if newHeadFastBlock == nil {
					newHeadFastBlock = bc.genesisBlock
				}
				//rawdb.WriteHeadFastBlockHash(db, newHeadFastBlock.Hash())
				rawdb.WriteLastFinalizedHash(db, newHeadFastBlock.Hash())

				// Degrade the chain markers if they are explicitly reverted.
				// In theory we should update all in-memory markers in the
				// last step, however the direction of SetHead is from high
				// to low, so it's safe the update in-memory markers directly.
				bc.lastFinalizedFastBlock.Store(newHeadFastBlock)
				headFastBlockGauge.Update(int64(*headerHeight))
			}
		}
		headHash := bc.GetLastFinalizedBlock().Hash()
		headHeight := bc.GetBlockFinalizedNumber(headHash)

		// If setHead underflown the freezer threshold and the block processing
		// intent afterwards is full block importing, delete the chain segment
		// between the stateful-block and the sethead target.
		var wipe = pivot == nil
		if !wipe && headHeight != nil && *headHeight+1 < frozen {
			wipe = *headHeight >= *pivot
		}
		return headHash, wipe // Only force wipe if full synced
	}
	// Rewind the header chain, deleting all block bodies until then
	delFn := func(db ethdb.KeyValueWriter, hash common.Hash) {
		num := rawdb.ReadFinalizedNumberByHash(bc.db, hash)
		// Ignore the error here since light client won't hit this path
		frozen, _ := bc.db.Ancients()
		if num != nil && *num+1 <= frozen {
			// Truncate all relative data(header, total difficulty, body, receipt
			// and canonical hash) from ancient store.
			if err := bc.db.TruncateAncients(*num); err != nil {
				log.Crit("Failed to truncate ancient data", "number", num, "err", err)
			}
		}

		delBlock := bc.GetBlockByHash(hash)
		log.Info("Delete block from db", "slot", delBlock.Slot(), "hash", hash)
		rawdb.DeleteSlotBlockHash(bc.Database(), delBlock.Slot(), hash)

		if num != nil {
			rawdb.DeleteBlock(db, hash, num)
		} else {
			rawdb.DeleteBlockWithoutNumber(db, hash)
		}
		bc.RemoveTips(common.HashArray{hash})

		// Todo(rjl493456442) txlookup, bloombits, etc
	}
	// If SetHead was only called as a chain reparation method, try to skip
	// touching the header chain altogether, unless the freezer is broken
	block := bc.GetLastFinalizedBlock()
	if block.Hash() == head {
		if target, force := updateFn(bc.db, block.Header()); force {
			bc.hc.SetHead(target, updateFn, delFn)
		}
	} else {
		// Rewind the chain to the requested head and keep going backwards until a
		// block with a state is found or fast sync pivot is passed
		log.Warn("Rewinding blockchain", "target", head.Hex())
		bc.hc.SetHead(head, updateFn, delFn)
	}
	// Clear out any stale content from the caches
	bc.valSyncCache.Purge()
	bc.bodyCache.Purge()
	bc.bodyRLPCache.Purge()
	bc.receiptsCache.Purge()
	bc.blockCache.Purge()
	bc.txLookupCache.Purge()
	bc.optimisticSpinesCache.Purge()
	log.Info("Purge caches of blockchain", "head", head.Hex())

	return rootNumber, bc.loadLastState()
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

// Reset purges the entire blockchain, restoring it to its genesis state.
func (bc *BlockChain) Reset() error {
	return bc.ResetWithGenesisBlock(bc.genesisBlock)
}

// ResetWithGenesisBlock purges the entire blockchain, restoring it to the
// specified genesis state.
func (bc *BlockChain) ResetWithGenesisBlock(genesis *types.Block) error {
	// Dump the entire block chain and purge the caches
	if err := bc.SetHead(genesis.Hash()); err != nil {
		return err
	}
	if !bc.chainmu.TryLock() {
		return errChainStopped
	}
	defer bc.chainmu.Unlock()

	// Prepare the genesis block and reinitialise the chain
	batch := bc.db.NewBatch()
	rawdb.WriteBlock(batch, genesis)
	if err := batch.Write(); err != nil {
		log.Crit("Failed to write genesis block", "err", err)
	}
	rawdb.AddSlotBlockHash(bc.Database(), genesis.Slot(), genesis.Hash())

	bc.writeFinalizedBlock(0, genesis, true)

	// Last update all in-memory chain markers
	genesisHeight := uint64(0)
	bc.genesisBlock = genesis
	bc.lastFinalizedBlock.Store(bc.genesisBlock)
	headBlockGauge.Update(int64(genesisHeight))
	bc.hc.SetGenesis(bc.genesisBlock.Header())
	genesisHeader := bc.genesisBlock.Header()
	bc.hc.SetLastFinalisedHeader(genesisHeader, genesisHeight)
	bc.lastFinalizedFastBlock.Store(bc.genesisBlock)
	headFastBlockGauge.Update(int64(genesisHeight))
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
		curBlock := bc.GetLastFinalizedBlock()
		if snapBase, err = bc.snaps.Journal(curBlock.Root()); err != nil {
			log.Error("Failed to journal state snapshot", "err", err)
		}
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

//var lastWrite uint64

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

// WriteFinalizedBlock writes the block and all associated state to the database.
func (bc *BlockChain) WriteFinalizedBlock(finNr uint64, block *types.Block, receipts []*types.Receipt, logs []*types.Log, state *state.StateDB, isHead bool) error {
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

// RollbackFinalization writes the block and all associated state to the database.
func (bc *BlockChain) RollbackFinalization(finNr uint64) error {
	if !bc.chainmu.TryLock() {
		return errInsertionInterrupted
	}
	defer bc.chainmu.Unlock()

	block := bc.GetBlockByNumber(finNr)
	block.SetNumber(nil)

	batch := bc.db.NewBatch()
	rawdb.DeleteFinalizedHashNumber(batch, block.Hash(), finNr)

	// update finalized number cache
	bc.hc.numberCache.Remove(block.Hash())

	bc.hc.headerCache.Remove(block.Hash())
	bc.hc.headerCache.Add(block.Hash(), block.Header())

	bc.blockCache.Remove(block.Hash())
	bc.blockCache.Add(block.Hash(), block)

	// Flush the whole batch into the disk, exit the node if failed
	if err := batch.Write(); err != nil {
		log.Crit("Failed to rollback block finalization", "finNr", finNr, "hash", block.Hash().Hex(), "err", err)
	}
	return nil
}

// WriteSyncDagBlock writes the dag block and all associated state to the database
// for dag synchronization process
func (bc *BlockChain) WriteSyncDagBlock(block *types.Block, validate bool) (status int, err error) {
	bc.blockProcFeed.Send(true)
	defer bc.blockProcFeed.Send(false)

	// Pre-checks passed, start the full block imports
	if !bc.chainmu.TryLock() {
		return 0, errInsertionInterrupted
	}
	n, err := bc.insertBlocks(types.Blocks{block}, validate, opSync)
	bc.chainmu.Unlock()

	err = bc.insertDalayedBloks()
	if err != nil {
		return n, err
	}
	return n, err
}

// WriteSyncBlocks writes the blocks and all associated state to the database while synchronization process.
func (bc *BlockChain) WriteSyncBlocks(blocks types.Blocks, validate bool) (status int, err error) {
	bc.blockProcFeed.Send(true)
	defer bc.blockProcFeed.Send(false)

	// Pre-checks passed, start the full block imports
	if !bc.chainmu.TryLock() {
		return 0, errInsertionInterrupted
	}
	n, err := bc.insertBlocks(blocks, validate, opSync)
	bc.chainmu.Unlock()
	if err != nil {
		return n, err
	}
	err = bc.insertDalayedBloks()
	return n, err
}

func (bc *BlockChain) insertDalayedBloks() error {
	if len(bc.insBlockCache) == 0 {
		return nil
	}
	log.Info("Insert delayed blocks: start", "count", len(bc.insBlockCache))
	insBlockCache := []*types.Block{}
	for _, bl := range bc.insBlockCache {
		if hdr := bc.GetHeader(bl.Hash()); hdr != nil {
			log.Info("Insert delayed blocks: skip inserted", "slot", bl.Slot(), "hash", bl.Hash().Hex())
			continue
		}
		_, insErr := bc.insertBlocks(types.Blocks{bl}, true, opDelay)
		if insErr == ErrInsertUncompletedDag {
			insBlockCache = append(insBlockCache, bl)
			log.Info("Insert delayed blocks: retry", "slot", bl.Slot(), "hash", bl.Hash().Hex(), "err", insErr)
		} else if insErr != nil {
			log.Error("Insert delayed blocks: error", "slot", bl.Slot(), "hash", bl.Hash().Hex(), "err", insErr)
			return insErr
		}
	}
	bc.insBlockCache = insBlockCache
	return nil
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
		//bc.triegc.Push(root, -int64(block.NumberU64()))

		//if current := block.NumberU64(); current > TriesInMemory {
		//	// If we exceeded our memory allowance, flush matured singleton nodes to disk
		//	var (
		//		nodes, imgs = triedb.Size()
		//		limit       = common.StorageSize(bc.cacheConfig.TrieDirtyLimit) * 1024 * 1024
		//	)
		//	if nodes > limit || imgs > 4*1024*1024 {
		//		triedb.Cap(limit - ethdb.IdealBatchSize)
		//	}
		//	// Find the next state trie we need to commit
		//	chosen := current - TriesInMemory
		//
		//	// If we exceeded out time allowance, flush an entire trie to disk
		//	if bc.gcproc > bc.cacheConfig.TrieTimeLimit {
		//		// If the header is missing (canonical chain behind), we're reorging a low
		//		// diff sidechain. Suspend committing until this operation is completed.
		//		header := bc.GetHeaderByNumber(chosen)
		//		if header == nil {
		//			log.Warn("Reorg in progress, trie commit postponed", "number", chosen)
		//		} else {
		//			// If we're exceeding limits but haven't reached a large enough memory gap,
		//			// warn the user that the system is becoming unstable.
		//			if chosen < lastWrite+TriesInMemory && bc.gcproc >= 2*bc.cacheConfig.TrieTimeLimit {
		//				log.Info("State in memory for too long, committing", "time", bc.gcproc, "allowance", bc.cacheConfig.TrieTimeLimit, "optimum", float64(chosen-lastWrite)/TriesInMemory)
		//			}
		//			// Flush an entire trie and restart the counters
		//			triedb.Commit(header.Root, true, nil)
		//			lastWrite = chosen
		//			bc.gcproc = 0
		//		}
		//	}
		//	// Garbage collect anything below our required write retention
		//	for !bc.triegc.Empty() {
		//		root, number := bc.triegc.Pop()
		//		if uint64(-number) > chosen {
		//			bc.triegc.Push(root, number)
		//			break
		//		}
		//		triedb.Dereference(root.(common.Hash))
		//	}
		//}
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

// deprecated, used for tests only
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
		// If the header is a banned one, straight out abort
		if BadHashes[block.Hash()] {
			return it.index, ErrBannedHash
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
		log.Info(" >>>>>>>>>>>>> SyncInsert schain <<<<<<<<<<<<<<", "height", block.Height(), "hash", block.Hash().Hex(), "err", err)
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

func (bc *BlockChain) verifyCpData(block *types.Block) bool {
	CpHeader := bc.GetHeaderByNumber(block.CpNumber())
	if CpHeader == nil {
		logValidationIssue("CpBlock header not found", block)
		return false
	}
	if block.CpNumber() != CpHeader.Nr() {
		logValidationIssue("CpBlock number above node GetLastFinalizedNumber", block)
		return false
	}
	if block.CpHash() != CpHeader.Hash() {
		logValidationIssue("Hash mismatch", block)
		return false
	}
	if block.CpRoot() != CpHeader.Root {
		logValidationIssue("CpHash mismatch", block)
		return false
	}
	if block.CpReceiptHash() != CpHeader.ReceiptHash {
		logValidationIssue("ReceiptHash mismatch", block)
		return false
	}
	if block.CpGasUsed() != CpHeader.GasUsed {
		logValidationIssue("GasUsed mismatch", block)
		return false
	}
	if block.CpBloom() != CpHeader.Bloom {
		logValidationIssue("Bloom mismatch", block)
		return false
	}
	return true
}

func logValidationIssue(issue string, block *types.Block) {
	log.Warn("Block verification: "+issue,
		"block hash", block.Hash().Hex(),
		"CpHash", block.CpHash(),
		"CpNumber", block.CpNumber(),
		"CpReceiptHash", block.CpReceiptHash(),
		"CpRoot", block.CpRoot(),
	)
}

// verifyCreators return false if creator is unassigned
func (bc *BlockChain) verifyCreators(block *types.Block) bool {
	var (
		creators []common.Address
		err      error
	)

	creators, err = bc.ValidatorStorage().GetCreatorsBySlot(bc, block.Slot())
	if err != nil {
		log.Error("can`t get shuffled validators", "error", err)
		return false
	}

	coinbase := block.Header().Coinbase
	contains, index := common.Contains(creators, coinbase)
	if !contains {
		log.Warn("Block verification: creator assignment failed", "slot", block.Slot(), "hash", block.Hash().Hex(), "block creator", block.Header().Coinbase.Hex(), "slot creators", creators)
		return false
	} else {
		signer := types.LatestSigner(bc.chainConfig)
		for _, tx := range block.Body().Transactions {
			from, _ := types.Sender(signer, tx)
			if !IsAddressAssigned(from, creators, int64(index)) && from != block.Coinbase() {
				log.Warn("Block verification: creator txs assignment failed", "slot", block.Slot(), "hash", block.Hash().Hex(), "block creator", block.Header().Coinbase.Hex(), "slot creators", creators, "txFrom", from)
				return false
			}
		}
	}
	return true
}

// CacheInvalidBlock cache invalid block
func (bc *BlockChain) CacheInvalidBlock(block *types.Block) {
	bc.invalidBlocksCache.Add(block.Hash(), struct{}{})
}

// VerifyBlock validate block
func (bc *BlockChain) VerifyBlock(block *types.Block) (ok bool, err error) {
	defer func(ts time.Time) {
		log.Info("^^^^^^^^^^^^ TIME",
			"elapsed", common.PrettyDuration(time.Since(ts)),
			"func:", "VerifyBlock:Total",
		)
	}(time.Now())

	ts := time.Now()

	// check future slot
	curSlot := bc.GetSlotInfo().CurrentSlot()
	if block.Slot() > curSlot+1 {
		log.Warn("Block verification: future slot", "curSlot", curSlot, "bl.slot", block.Slot(), "hash", block.Hash().Hex(), "bl.time", block.Time(), "now", time.Now().Unix())
		return false, nil
	}

	log.Info("^^^^^^^^^^^^ TIME",
		"elapsed", common.PrettyDuration(time.Since(ts)),
		"func:", "VerifyBlock: check future slot",
	)
	ts = time.Now()

	if len(block.ParentHashes()) == 0 {
		log.Warn("Block verification: no parents", "hash", block.Hash().Hex())
		return false, nil
	}
	if !bc.verifyCreators(block) {
		return false, nil
	}

	log.Info("^^^^^^^^^^^^ TIME",
		"elapsed", common.PrettyDuration(time.Since(ts)),
		"func:", "VerifyBlock:verifyCreators",
	)
	ts = time.Now()

	// Verify block era
	isValidEra := bc.verifyBlockEra(block)
	if !isValidEra {
		log.Warn("Block verification: invalid era", "hash", block.Hash().Hex(), "block era", block.Era())
		return false, nil
	}

	log.Info("^^^^^^^^^^^^ TIME",
		"elapsed", common.PrettyDuration(time.Since(ts)),
		"func:", "VerifyBlock:verifyBlockEra",
	)
	ts = time.Now()

	//isCpAncestor, ancestors, unloaded, err := bc.CollectAncestorsAftCpByParents(block.ParentHashes(), block.CpHash())
	isCpAncestor, ancestors, unloaded, _, err := bc.CollectAncestorsAftCpByTips(block.ParentHashes(), block.CpHash())
	if err != nil {
		log.Error("Block verification: check ancestors err", "err", err, "block hash", block.Hash().Hex())
		return false, err
	}
	//check is block's chain synced and does not content rejected blocks
	if len(unloaded) > 0 {
		for _, unh := range unloaded {
			if _, ok := bc.invalidBlocksCache.Get(unh); ok {
				log.Warn("Block verification: invalid parent", "hash", block.Hash().Hex(), "invalid parent", unh.Hex())
				return false, nil
			}
			log.Warn("Block verification: unknown parent", "hash", block.Hash().Hex(), "unknown parent", unh.Hex())
			continue
		}
		return false, ErrInsertUncompletedDag
	}
	// cp must be an ancestor of the block
	if !isCpAncestor {
		log.Warn("Block verification: checkpoint is not ancestor", "hash", block.Hash().Hex(), "cpHash", block.CpHash().Hex())
		return false, nil
	}

	log.Info("^^^^^^^^^^^^ TIME",
		"elapsed", common.PrettyDuration(time.Since(ts)),
		"func:", "VerifyBlock:isCpAncestor",
	)
	ts = time.Now()

	// parents' heights must be less than block height
	for _, parentHash := range block.ParentHashes() {
		parent := bc.GetHeader(parentHash)
		if _, ok := bc.invalidBlocksCache.Get(parent); ok {
			log.Warn("Block verification: invalid parent", "hash", block.Hash().Hex(), "invalid parent", parentHash.Hex())
			return false, nil
		}
		if parent.Height >= block.Height() || parent.Slot >= block.Slot() {
			log.Warn("Block verification: invalid parent", "height", block.Height(), "slot", block.Slot(), "parent height", parent.Height, "parent slot", parent.Slot)
			return false, nil
		}
	}

	log.Info("^^^^^^^^^^^^ TIME",
		"elapsed", common.PrettyDuration(time.Since(ts)),
		"func:", "VerifyBlock:parents' heights",
	)
	ts = time.Now()

	//validate height
	cpHeader := bc.GetHeader(block.CpHash())
	calcHeight := bc.calcBlockHeight(cpHeader.Height, len(ancestors))
	if block.Height() != calcHeight {
		log.Warn("Block verification: block invalid height",
			"calcHeight", calcHeight,
			"height", block.Height(),
			"hash", block.Hash().Hex(),
			"cpHeight", cpHeader.Height,
		)
		return false, nil
	}

	if !bc.verifyBlockBaseFee(block) {
		return false, nil
	}

	log.Info("^^^^^^^^^^^^ TIME",
		"elapsed", common.PrettyDuration(time.Since(ts)),
		"func:", "VerifyBlock:validate height",
	)
	ts = time.Now()

	// Verify block checkpoint
	isValidCp := bc.verifyCheckpoint(block)
	if !isValidCp {
		log.Warn("Block verification: invalid checkpoint", "hash", block.Hash().Hex(), "cp.hash", block.CpHash().Hex())
		return false, nil
	}

	log.Info("^^^^^^^^^^^^ TIME",
		"elapsed", common.PrettyDuration(time.Since(ts)),
		"func:", "VerifyBlock:Verify block checkpoint",
	)
	ts = time.Now()

	intrGasSum := uint64(0)
	signer := types.MakeSigner(bc.Config())
	for _, tx := range block.Transactions() {
		msg, err := tx.AsMessage(signer, block.BaseFee())
		intrGas, err := bc.EstimateGas(msg, block.Header())
		if err != nil {
			log.Warn("Block verification: gas usage error", "err", err)
			return false, nil
		}
		intrGasSum += intrGas
	}
	if intrGasSum > block.GasLimit() {
		log.Warn("Block verification: intrinsic gas sum > gasLimit", "block hash", block.Hash().Hex(), "gasLimit", block.GasLimit(), "IntrinsicGas sum", intrGasSum)
		return false, nil
	}

	log.Info("^^^^^^^^^^^^ TIME",
		"elapsed", common.PrettyDuration(time.Since(ts)),
		"func:", "VerifyBlock:TxEstimateGas",
		"txs", len(block.Transactions()),
	)
	ts = time.Now()

	isValid, err := bc.verifyBlockParents(block)
	if err != nil {
		return false, err
	}

	log.Info("^^^^^^^^^^^^ TIME",
		"elapsed", common.PrettyDuration(time.Since(ts)),
		"func:", "VerifyBlock:verifyBlockParents",
		"parents", len(block.ParentHashes()),
	)
	ts = time.Now()

	// validate cp data
	isCpValid := bc.verifyCpData(block)

	log.Info("^^^^^^^^^^^^ TIME",
		"elapsed", common.PrettyDuration(time.Since(ts)),
		"func:", "VerifyBlock:verifyCpData",
		"parents", len(block.ParentHashes()),
	)

	return isValid && isCpValid, nil
}

func (bc *BlockChain) verifyBlockParents(block *types.Block) (bool, error) {
	parentsHeaders := bc.GetHeadersByHashes(block.ParentHashes())
	if bc.AreParentSlotsUniform(block.Slot(), parentsHeaders) {
		log.Info("##### verifyBlockParents : light")
		return true, nil
	}

	log.Info("##### verifyBlockParents : heavy")
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
				log.Warn("Block verification: parent-ancestor detected", "block", block.Hash().Hex(), "parent", parentHeader.Hash().Hex(), "parent-ancestor", pparent.Hash().Hex())
				return false, nil
			}
		}
	}
	return true, nil
}

// AreParentSlotsUniform checks if all parent headers in the provided map have the same slot value.
// It returns true if all slots are the same, and false otherwise.
func (bc *BlockChain) AreParentSlotsUniform(slot uint64, parentsHeaders types.HeaderMap) bool {
	var lastSlot *uint64 = nil
	for _, parentHeader := range parentsHeaders {
		if parentHeader.Slot >= slot {
			log.Warn("Validation error: The slot of the parent is equal to or larger than the block's slot.",
				"hash", parentHeader.Hash(),
				"parSlot", parentHeader.Slot,
				"slot", slot,
			)
			return false
		}

		if lastSlot == nil {
			lastSlot = &parentHeader.Slot
		} else if *lastSlot != parentHeader.Slot {
			return false
		}
	}
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
		// If the header is a banned one, straight out abort
		if BadHashes[block.Hash()] {
			return it.index, ErrBannedHash
		}

		//// if checkpoint of propagated block is not finalized - set IsSynced=false
		//if cpHeader := bc.GetHeaderByHash(block.CpHash()); cpHeader != nil {
		//	if cpHeader.Height > 0 && cpHeader.Nr() == 0 {
		//		log.Warn("Insert blocks: Check is synchronized: cp not finalized","op", op, "cpHash", block.CpHash(), "cpSlot", cpHeader.Slot)
		//		bc.SetIsSynced(false)
		//	}
		//} else {
		//	log.Warn("Insert blocks: Check is synchronized: cp not found","op", op, "cpHash", block.CpHash())
		//	bc.SetIsSynced(false)
		//	return it.index, ErrInsertUncompletedDag
		//}

		if validate {
			//todo check
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
				//should never happen
				panic("should never happen")
			}
			tmpTips.Add(bdag)
		}
		dagChainHashes, err := bc.CollectAncestorsHashesByTips(tmpTips, block.CpHash())
		if err != nil {
			return it.index, err
		}
		dagBlock := &types.BlockDAG{
			Hash:                   block.Hash(),
			Height:                 block.Height(),
			Slot:                   block.Slot(),
			CpHash:                 block.CpHash(),
			CpHeight:               block.CpNumber(),
			OrderedAncestorsHashes: dagChainHashes,
		}
		bc.AddTips(dagBlock)
		bc.RemoveTips(dagBlock.OrderedAncestorsHashes)
		bc.MoveTxsToProcessing(block)
		bc.WriteCurrentTips()

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

	////if state is last finalized block
	//if sortedBlocks[0].Nr() == lastFinBlock.Nr() ||
	//	//if state is dag block
	//	sortedBlocks[0].Slot() > lastFinBlock.Slot() && sortedBlocks[0].Nr() == 0 && sortedBlocks[0].Height() > 0 {
	//	stateBlock = sortedBlocks[0]
	//	statedb, err = bc.StateAt(stateBlock.Root())
	//	if err != nil {
	//		log.Error("Error while get state by parents", "slot", stateBlock.Slot(), "nr", stateBlock.Nr(), "height", stateBlock.Height(), "hash", stateBlock.Hash().Hex())
	//	}
	//	recommitBlocks = sortedBlocks[1:]
	//	calcHeight = bc.calcBlockHeight(stateBlock.Height(), len(recommitBlocks))
	//	return statedb, stateBlock, recommitBlocks, calcHeight, nil
	//} else {
	//	//if state is finalized block - search first spine in ancestors
	//	stateBlock = sortedBlocks[0]
	//	statedb, err = bc.StateAt(stateBlock.Root())
	//	if err != nil {
	//		log.Error("Error while get state by parents", "slot", stateBlock.Slot(), "nr", stateBlock.Nr(), "height", stateBlock.Height(), "hash", stateBlock.Hash().Hex(), "err", err)
	//	}
	//	if statedb != nil {
	//		recommitBlocks = sortedBlocks[1:]
	//		calcHeight = bc.calcBlockHeight(stateBlock.Height(), len(recommitBlocks))
	//		return statedb, stateBlock, recommitBlocks, calcHeight, nil
	//	}
	//}

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

// CollectStateDataByFinalizedBlockRecursive collects state data of current dag chain to insert new block.
func (bc *BlockChain) CollectStateDataByFinalizedBlockRecursive(block *types.Block, _memo types.BlockMap) (statedb *state.StateDB, stateBlock *types.Block, recommitBlocks []*types.Block, err error) {
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

	if _memo == nil {
		_memo = types.BlockMap{}
	}

	parentBlocks := bc.GetBlocksByHashes(block.ParentHashes()).ToArray()
	for _, b := range parentBlocks {
		if b != nil {
			_memo.Add(b)
		}
	}
	parentBlocks = types.SpineSortBlocks(parentBlocks)
	spineBlock := parentBlocks[0]
	_stdb, err := bc.StateAt(spineBlock.Root())
	if err != nil || _stdb == nil {
		log.Warn("Collect State Data By Finalized Block: skip block", "nr", finNr, "height", block.Height(), "slot", block.Slot(), "hash", block.Hash().Hex(), "err", err)
	}
	if stateBlock == nil || stateBlock.Nr() < spineBlock.Nr() {
		statedb = _stdb
		stateBlock = spineBlock
	}
	if statedb == nil {
		_stdb, _stBlock, _recomBls, err := bc.CollectStateDataByFinalizedBlockRecursive(spineBlock, _memo)
		if err != nil {
			log.Warn("Collect State Data By Finalized Block: skip block", "nr", finNr, "height", block.Height(), "slot", block.Slot(), "hash", block.Hash().Hex(), "err", err)
		}
		//todo check condition
		// if stateBlock == nil || stateBlock.Nr() > stateBlock.Nr() {
		if stateBlock == nil || stateBlock.Nr() > spineBlock.Nr() {
			statedb = _stdb
			stateBlock = _stBlock
			for _, b := range _recomBls {
				if b != nil {
					_memo.Add(b)
				}
			}
		}
	}
	//rm stateBlock and blocks with lt nr
	nrs := common.SorterAscU64{}
	blockMap := types.BlockMap{}
	for _, bl := range _memo {
		if bl == nil || bl.Nr() <= stateBlock.Nr() {
			continue
		}
		blockMap[bl.Hash()] = bl
		nrs = append(nrs, bl.Nr())
	}
	//sort by number
	recommitBlocks = make([]*types.Block, len(nrs))
	sort.Sort(nrs)
	for i, nr := range nrs {
		for _, bl := range blockMap {
			if nr == bl.Nr() {
				recommitBlocks[i] = bl
				break
			}
		}
	}
	//recommitBlocks = sortedRecomBls
	return statedb, stateBlock, recommitBlocks, nil
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
		return bc.EstimateGasByEvm(msg, header, blockContext, stateDb, validatorProcessor, tokenProcessor)
	case TokenCreationTxType, TokenMethodTxType:
		return IntrinsicGas(msg.Data(), msg.AccessList(), false, false)
	default:
		return 0, ErrTxTypeNotSupported
	}
}

func (bc *BlockChain) EstimateGasByEvm(msg types.Message,
	header *types.Header,
	blkCtx vm.BlockContext,
	statedb *state.StateDB,
	vp *validator.Processor,
	tp *token.Processor,
) (uint64, error) {
	defer func(start time.Time) { log.Info("Executing EVM call finished", "runtime", time.Since(start)) }(time.Now())
	from := msg.From()
	statedb.SetNonce(from, msg.Nonce())
	maxGas := (new(big.Int)).SetUint64(header.GasLimit)
	gasBalance := new(big.Int).Mul(maxGas, msg.GasPrice())
	reqBalance := new(big.Int).Add(gasBalance, msg.Value())
	statedb.SetBalance(from, reqBalance)

	gasPool := new(GasPool).AddGas(math.MaxUint64)

	evm := vm.NewEVM(blkCtx, vm.TxContext{}, statedb, bc.Config(), *bc.GetVMConfig())

	receipt, err := ApplyMessage(evm, tp, vp, msg, gasPool)
	if err != nil {
		log.Error("Tx estimate gas by evm: error", "lfNumber", header.Nr(), "tx", msg.TxHash().Hex(), "err", err)
		return 0, err
	}
	log.Info("Tx estimate gas by evm: success", "lfNumber", header.Nr(), "tx", msg.TxHash().Hex(), "txGas", msg.Gas(), "calcGas", receipt.UsedGas)
	return receipt.UsedGas, nil
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
		// If the header is a banned one, straight out abort
		if BadHashes[block.Hash()] {
			return it.index, ErrBannedHash
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
	//var (
	//current  = bc.GetLastFinalizedBlock()
	//)
	// The first sidechain block error is already verified to be ErrPrunedAncestor.
	// Since we don't import them here, we expect ErrUnknownAncestor for the remaining
	// ones. Any other errors means that the block is invalid, and should not be written
	// to disk.
	err := consensus.ErrPrunedAncestor
	for ; block != nil && errors.Is(err, consensus.ErrPrunedAncestor); block, err = it.next() {
		//todo
		// Check the canonical state root for that number
		//if number := block.NumberU64(); current.NumberU64() >= number {
		//	canonical := bc.GetBlockByNumber(number)
		//	if canonical != nil && canonical.Hash() == block.Hash() {
		//		// Not a sidechain block, this is a re-import of a canon block which has it's state pruned
		//		continue
		//	}
		//	if canonical != nil && canonical.Root() == block.Root() {
		//		// This is most likely a shadow-state attack. When a fork is imported into the
		//		// database, and it eventually reaches a block height which is not pruned, we
		//		// just found that the state already exist! This means that the sidechain block
		//		// refers to a state which already exists in our canon chain.
		//		//
		//		// If left unchecked, we would now proceed importing the blocks, without actually
		//		// having verified the state of the previous blocks.
		//		log.Warn("Sidechain ghost-state attack detected", "number", block.Nr(), "sideroot", block.Root(), "canonroot", canonical.Root())
		//
		//		// If someone legitimately side-mines blocks, they would still be imported as usual. However,
		//		// we cannot risk writing unverified blocks to disk when they obviously target the pruning
		//		// mechanism.
		//		return it.index, errors.New("sidechain ghost-state attack")
		//	}
		//}

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
	dagHashes := common.HashArray{}
	tips := *bc.hc.GetTips()

	//tipsHashes := tips.GetOrderedAncestorsHashes()
	//dagBlocks := bc.GetBlocksByHashes(tipsHashes)
	//for hash, bl := range dagBlocks {
	//	if bl != nil && bl.Nr() == 0 && bl.Height() > 0 {
	//		dagHashes = append(dagHashes, hash)
	//	}
	//}
	//if len(dagHashes) == 0 {
	//	dagHashes = common.HashArray{bc.GetLastFinalizedBlock().Hash()}
	//}

	expCache := ExploreResultMap{}
	for hash, tip := range tips {
		if hash == tip.CpHash {
			dagHashes = append(dagHashes, hash)
			continue
		}
		_, loaded, _, _, c, _ := bc.ExploreChainRecursive(hash, expCache)
		expCache = c
		dagHashes = dagHashes.Concat(loaded)
	}
	dagHashes.Deduplicate()
	dagHashes.Sort()
	return &dagHashes
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
	return bc.hc.CollectAncestorsAftCpByParents(parents, cpHeader)
}

// CollectAncestorsAftCpByTips recursively collect ancestors by block parents
// which have to be finalized after checkpoint up to block.
func (bc *BlockChain) CollectAncestorsAftCpByTips(parents common.HashArray, cpHash common.Hash) (
	isCpAncestor bool,
	ancestors types.HeaderMap,
	unloaded common.HashArray,
	tips types.Tips,
	err error,
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
	_, ancestors, unl, _, err := bc.CollectAncestorsAftCpByTips(header.ParentHashes, header.CpHash)
	if err != nil {
		return false, err
	}
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
	tips := types.Tips{}
	for hash, dag := range *bc.hc.GetTips() {
		if dag != nil && dag.CpHash != (common.Hash{}) || dag.Hash == bc.genesisBlock.Hash() {
			tips[hash] = dag
		}
	}
	return tips
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

func (bc *BlockChain) verifyBlockEra(block *types.Block) bool {
	// Get the epoch of the block
	blockEpoch := bc.GetSlotInfo().SlotToEpoch(block.Slot())

	fEpochSl, _ := bc.slotInfo.SlotOfEpochStart(bc.eraInfo.FirstEpoch())
	lEpochSl, _ := bc.slotInfo.SlotOfEpochEnd(bc.eraInfo.LastEpoch())

	// Get the era that the block belongs to
	var valEra *era.Era
	if bc.GetEraInfo().GetEra().Number == block.Era() {
		log.Info("Block verification: era",
			"eraInfo", bc.GetEraInfo().GetEra(),
			"blEra", block.Era(),
			"blEpoch", blockEpoch,
			"blSlot", block.Slot(),
			"blHash", block.Hash(),
			"blNr", block.Nr(),
			"FirsEpoch", bc.eraInfo.FirstEpoch(),
			"LastEpoch", bc.eraInfo.LastEpoch(),
			"FirstEpochEraSlot", fEpochSl,
			"LastEpochEraSlot", lEpochSl,
		)

		valEra = bc.GetEraInfo().GetEra()
	} else {
		valEra = rawdb.ReadEra(bc.db, block.Era())
		log.Info("Block verification: era", "eraInfo", valEra, "blEra", block.Era(), "blockEpoch", blockEpoch, "blSlot", block.Slot(), "hash", block.Hash(), "blNr", block.Nr())
	}

	// Check if the block epoch is within the era
	return valEra != nil && valEra.IsContainsEpoch(blockEpoch)
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
	// cp must be correct finalized
	if block.CpNumber() != cpHeader.Nr() {
		log.Warn("Block verification: cp mismatch fin numbers",
			"bl.CpNumber", block.CpNumber(),
			"cp.Number", cpHeader.Nr(),
			"cp.Height", cpHeader.Height,
			"cp.Hash", block.CpHash().Hex(),
			"bl.Hash", block.Hash().Hex(),
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

func (bc *BlockChain) EnterNextEra(cp *types.Checkpoint, root common.Hash) *era.Era {
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

	epochSlot, err := bc.GetSlotInfo().SlotOfEpochStart(cp.FinEpoch - bc.Config().TransitionPeriod)
	if err != nil {
		log.Error("EnterNextEra slot of cp finEpoch start error", "err", err)
	}

	validators, _ := bc.ValidatorStorage().GetValidators(bc, epochSlot, true, false, "EnterNextEra")
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
	if curEra.IsContainsEpoch(epoch) {
		return curEra
	}

	eraNumber := curEra.Number
	findingEra := new(era.Era)

	if epoch < curEra.From {
		for !findingEra.IsContainsEpoch(epoch) {
			eraNumber--
			findingEra = rawdb.ReadEra(bc.db, eraNumber)
		}
	} else if epoch > curEra.To {
		for !findingEra.IsContainsEpoch(epoch) {
			eraNumber++
			findingEra = rawdb.ReadEra(bc.db, eraNumber)
		}
	}

	return findingEra
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
