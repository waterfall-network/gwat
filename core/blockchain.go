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
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/rlp"
	"io"
	"math/big"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/common/prque"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/state/snapshot"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/internal/syncx"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie"
	lru "github.com/hashicorp/golang-lru"
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

	blockReorgMeter         = metrics.NewRegisteredMeter("chain/reorg/executes", nil)
	blockReorgAddMeter      = metrics.NewRegisteredMeter("chain/reorg/add", nil)
	blockReorgDropMeter     = metrics.NewRegisteredMeter("chain/reorg/drop", nil)
	blockReorgInvalidatedTx = metrics.NewRegisteredMeter("chain/reorg/invalidTx", nil)

	blockPrefetchExecuteTimer   = metrics.NewRegisteredTimer("chain/prefetch/executes", nil)
	blockPrefetchInterruptMeter = metrics.NewRegisteredMeter("chain/prefetch/interrupts", nil)

	errInsertionInterrupted = errors.New("insertion is interrupted")
	errChainStopped         = errors.New("blockchain is stopped")
)

const (
	bodyCacheLimit      = 256
	blockCacheLimit     = 256
	receiptsCacheLimit  = 32
	txLookupCacheLimit  = 1024 //262144
	maxFutureBlocks     = 256
	maxTimeFutureBlocks = 30
	TriesInMemory       = 128

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

	db     ethdb.Database // Low level persistent database to store final content in
	snaps  *snapshot.Tree // Snapshot tree for fast trie leaf access
	triegc *prque.Prque   // Priority queue mapping block numbers to tries to gc
	gcproc time.Duration  // Accumulates canonical block processing for trie dumping

	// txLookupLimit is the maximum number of blocks from head whose tx indices
	// are reserved:
	//  * 0:   means no limit and regenerate any missing indexes
	//  * N:   means N block limit [HEAD-N+1, HEAD] and delete extra indexes
	//  * nil: disable tx reindexer/deleter, but still index new blocks
	txLookupLimit uint64

	hc            *HeaderChain
	rmLogsFeed    event.Feed
	chainFeed     event.Feed
	chainSideFeed event.Feed
	chainHeadFeed event.Feed
	logsFeed      event.Feed
	blockProcFeed event.Feed
	scope         event.SubscriptionScope
	genesisBlock  *types.Block

	DagMu sync.RWMutex // finalizing lock

	// This mutex synchronizes chain write operations.
	// Readers don't need to take it, they can just read the database.
	chainmu *syncx.ClosableMutex

	lastFinalizedBlock     atomic.Value // Current last finalized block of the blockchain
	lastFinalizedFastBlock atomic.Value // Current last finalized block of the fast-sync chain (may be above the blockchain!)

	stateCache    state.Database // State database to reuse between imports (contains state cache)
	bodyCache     *lru.Cache     // Cache for the most recent block bodies
	bodyRLPCache  *lru.Cache     // Cache for the most recent block bodies in RLP encoded format
	receiptsCache *lru.Cache     // Cache for the most recent receipts per block
	blockCache    *lru.Cache     // Cache for the most recent entire blocks
	txLookupCache *lru.Cache     // Cache for the most recent transaction lookup data.
	futureBlocks  *lru.Cache     // future blocks are blocks added for later processing

	insBlockCache []*types.Block // Cache for blocks to insert late

	wg            sync.WaitGroup //
	quit          chan struct{}  // shutdown signal, closed in Stop.
	running       int32          // 0 if chain is running, 1 when stopped
	procInterrupt int32          // interrupt signaler for block processing

	engine     consensus.Engine
	validator  Validator // Block and state validator interface
	prefetcher Prefetcher
	processor  Processor // Block transaction processor interface
	vmConfig   vm.Config

	shouldPreserve func(*types.Block) bool // Function used to determine whether should preserve the given block.
}

// NewBlockChain returns a fully initialised block chain using information
// available in the database. It initialises the default Ethereum Validator and
// Processor.
func NewBlockChain(db ethdb.Database, cacheConfig *CacheConfig, chainConfig *params.ChainConfig, engine consensus.Engine, vmConfig vm.Config, shouldPreserve func(block *types.Block) bool, txLookupLimit *uint64) (*BlockChain, error) {
	if cacheConfig == nil {
		cacheConfig = defaultCacheConfig
	}
	bodyCache, _ := lru.New(bodyCacheLimit)
	bodyRLPCache, _ := lru.New(bodyCacheLimit)
	receiptsCache, _ := lru.New(receiptsCacheLimit)
	blockCache, _ := lru.New(blockCacheLimit)
	txLookupCache, _ := lru.New(txLookupCacheLimit)
	futureBlocks, _ := lru.New(maxFutureBlocks)

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
		quit:           make(chan struct{}),
		chainmu:        syncx.NewClosableMutex(),
		shouldPreserve: shouldPreserve,
		bodyCache:      bodyCache,
		bodyRLPCache:   bodyRLPCache,
		receiptsCache:  receiptsCache,
		blockCache:     blockCache,
		txLookupCache:  txLookupCache,
		futureBlocks:   futureBlocks,
		engine:         engine,
		vmConfig:       vmConfig,
	}
	bc.validator = NewBlockValidator(chainConfig, bc, engine)
	bc.prefetcher = newStatePrefetcher(chainConfig, bc, engine)
	bc.processor = NewStateProcessor(chainConfig, bc, engine)

	var err error
	bc.hc, err = NewHeaderChain(db, chainConfig, engine, bc.insertStopped)
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
	}

	var nilBlock *types.Block = bc.genesisBlock
	bc.lastFinalizedBlock.Store(nilBlock)
	bc.lastFinalizedFastBlock.Store(nilBlock)

	// Initialize the chain with ancient data if it isn't empty.
	var txIndexBlock uint64

	if bc.empty() {
		rawdb.InitDatabaseFromFreezer(bc.db)
		// If ancient database is not empty, reconstruct all missing
		// indices in the background.
		frozen, _ := bc.db.Ancients()
		if frozen > 0 {
			txIndexBlock = frozen
		}
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
			diskRoot = rawdb.ReadSnapshotRoot(bc.db)
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
		)
		// The head full block may be rolled back to a very low height due to
		// blockchain repair. If the head full block is even lower than the ancient
		// chain, truncate the ancient store.
		fullBlock := bc.CurrentBlock()
		if fullBlock != nil && fullBlock.Hash() != bc.genesisBlock.Hash() && fullBlock.NumberU64() < frozen-1 {
			needRewind = true
			low = fullBlock.NumberU64()
		}
		// In fast sync, it may happen that ancient data has been written to the
		// ancient store, but the LastFastBlock has not been updated, truncate the
		// extra data here.
		fastBlock := bc.CurrentFastBlock()
		if fastBlock != nil && fastBlock.NumberU64() < frozen-1 {
			needRewind = true
			if fastBlock.NumberU64() < low || low == 0 {
				low = fastBlock.NumberU64()
			}
		}
		if needRewind {
			log.Error("Truncating ancient chain", "from", bc.CurrentHeader().Number.Uint64(), "to", low)
			if err := bc.SetHead(low); err != nil {
				return nil, err
			}
		}
	}
	// The first thing the node will do is reconstruct the verification data for
	// the head block (ethash cache or clique voting snapshot). Might as well do
	// it in advance.
	bc.engine.VerifyHeader(bc, bc.CurrentHeader(), true)

	// Check the current state of the block hashes and make sure that we do not have any of the bad blocks in our chain
	for hash := range BadHashes {
		if header := bc.GetHeaderByHash(hash); header != nil {
			// get the canonical block corresponding to the offending header's number
			headerByNumber := bc.GetHeaderByNumber(header.Number.Uint64())
			// make sure the headerByNumber (if present) is in our current canonical chain
			if headerByNumber != nil && headerByNumber.Hash() == header.Hash() {
				log.Error("Found bad hash, rewinding chain", "number", header.Number, "hash", header.ParentHash)
				if err := bc.SetHead(header.Number.Uint64() - 1); err != nil {
					return nil, err
				}
				log.Error("Chain rewind was successful, resuming normal operation")
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

		head := bc.CurrentBlock()
		if layer := rawdb.ReadSnapshotRecoveryNumber(bc.db); layer != nil && *layer > head.NumberU64() {
			log.Warn("Enabling snapshot recovery", "chainhead", head.NumberU64(), "diskbase", *layer)
			recover = true
		}
		bc.snaps, _ = snapshot.New(bc.db, bc.stateCache.TrieDB(), bc.cacheConfig.SnapshotLimit, head.Root(), !bc.cacheConfig.SnapshotWait, true, recover)
	}

	// Start future block processor.
	bc.wg.Add(1)
	go bc.futureBlocksLoop()

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
	//// cache TransactionLookup for dag chain
	//for _, block := range bc.GetBlocksByHashes(*bc.GetDagHashes()) {
	//	bc.CacheTransactionLookup(block)
	//}
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
		//return bc.Reset()
	}
	tips, _ := bc.ReviseTips()

	if len(*tips) == 0 {
		bc.ResetTips()
		tips, _ = bc.ReviseTips()
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
							bc.DeleteBockDag(parent.Hash())
							continue
						}
					}
					if _, err := state.New(newHeadBlock.Root(), bc.stateCache, bc.snaps); err != nil {
						log.Warn("Block state missing, rewinding further", "number", rootNumber, "hash", newHeadBlock.Hash())
						// if pivot == nil || *pivot == 0 {
						if pivot == nil || newHeadBlock.Nr() > *pivot {
							parent := bc.GetBlockByNumber(rootNumber - 1)
							if parent != nil {
								newHeadBlock = parent
								rootNumber = newHeadBlock.Nr()
								rawdb.DeleteChildren(db, parent.Hash())
								bc.DeleteBockDag(parent.Hash())
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
						log.Debug("Rewound to block with state", "number", uint64(rootNumber), "hash", newHeadBlock.Hash().Hex())
						break
					}
					log.Debug("Skipping block with threshold state", "number", rootNumber, "hash", newHeadBlock.Hash(), "root", newHeadBlock.Root())
					newHeadBlock = bc.GetBlockByNumber(newHeadBlock.Nr()) // Keep rewinding
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

			newBlockDag := &types.BlockDAG{
				Hash:                newHeadBlock.Hash(),
				Height:              newHeadBlock.Height(),
				LastFinalizedHash:   newHeadBlock.Hash(),
				LastFinalizedHeight: newHeadBlock.Nr(),
				DagChainHashes:      common.HashArray{},
				FinalityPoints:      common.HashArray{},
			}
			bc.AddTips(newBlockDag)
			bc.ReviseTips()
		}
		// Rewind the fast block in a simpleton way to the target head
		//if currentFastBlock := bc.CurrentFastBlock(); currentFastBlock != nil && header.Number.Uint64() < currentFastBlock.NumberU64() {

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
			// Remove the hash <-> number mapping from the active store.
			rawdb.DeleteHeader(db, hash)
		} else {
			// Remove relative body and receipts from the active store.
			// The header, total difficulty and canonical hash will be
			// removed in the hc.SetHead function.
			rawdb.DeleteBody(db, hash)
			rawdb.DeleteReceipts(db, hash)
		}
		// Todo(rjl493456442) txlookup, bloombits, etc
	}

	// If SetHead was only called as a chain reparation method, try to skip
	// touching the header chain altogether, unless the freezer is broken
	if block := bc.GetLastFinalizedBlock(); block.Hash() == head {
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
	bc.bodyCache.Purge()
	bc.bodyRLPCache.Purge()
	bc.receiptsCache.Purge()
	bc.blockCache.Purge()
	bc.txLookupCache.Purge()
	bc.futureBlocks.Purge()

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

// GasLimit returns the gas limit of the current HEAD block.
func (bc *BlockChain) GasLimit() uint64 {
	curBlock := bc.GetLastFinalizedBlock()
	return curBlock.GasLimit()
}

// GetLastFinalizedBlock retrieves the current Last Finalized block of the canonical chain. The
// block is retrieved from the blockchain's internal cache.
func (bc *BlockChain) GetLastFinalizedBlock() *types.Block {
	lfb := bc.lastFinalizedBlock.Load().(*types.Block)
	return lfb
}

// GetDagHashes retrieves all non finalized block's hashes
func (bc *BlockChain) GetDagHashes() *common.HashArray {
	dagHashes := common.HashArray{}
	for hash, tip := range *bc.hc.GetTips() {
		if hash == tip.LastFinalizedHash {
			dagHashes = append(dagHashes, hash)
			continue
		}
		_, loaded, _, _, _ := bc.ExploreChainRecursive(hash)
		dagHashes = dagHashes.Concat(loaded)
	}
	dagHashes = dagHashes.Uniq()
	return &dagHashes
}

// GetUnsynchronizedTipsHashes retrieves tips with incomplete chain to finalized state
func (bc *BlockChain) GetUnsynchronizedTipsHashes() common.HashArray {
	tipsHashes := common.HashArray{}
	for hash, dag := range *bc.hc.GetTips() {
		if dag == nil || dag.LastFinalizedHash == (common.Hash{}) {
			tipsHashes = append(tipsHashes, hash)
		}
	}
	return tipsHashes
}

//todo check and RM
// EmitTipsSynced if tips are synchronized - emit tipsSynced event
// returns true if evt is emitted, otherwise - false
func (bc *BlockChain) EmitTipsSynced(tips *types.Tips) bool {
	if tips != nil {
		bc.tipsSynced.Send(*tips)
		return true
	}
	if unsynced := bc.GetUnsynchronizedTipsHashes(); len(unsynced) == 0 {
		tips := bc.GetTips().Copy()
		bc.tipsSynced.Send(tips)
		return true
	}
	return false
}

// ReviseTips revise tips state
// explore chains to update tips in accordance with sync process
func (bc *BlockChain) ReviseTips() (tips *types.Tips, unloadedHashes common.HashArray) {
	return bc.hc.ReviseTips(bc)
}

func (bc *BlockChain) ExploreChainRecursive(headHash common.Hash) (unloaded, loaded, finalized common.HashArray, graph *types.GraphDag, err error) {
	block := bc.GetBlockByHash(headHash)
	graph = &types.GraphDag{
		Hash:   headHash,
		Height: 0,
		Graph:  []*types.GraphDag{},
		State:  types.BSS_NOT_LOADED,
	}
	if block == nil {
		// if block not loaded
		return common.HashArray{headHash}, loaded, finalized, graph, nil
	}
	graph.State = types.BSS_LOADED
	graph.Height = block.Height()
	if nr := rawdb.ReadFinalizedNumberByHash(bc.db, headHash); nr != nil {
		// if finalized
		graph.State = types.BSS_FINALIZED
		graph.Number = *nr
		return unloaded, loaded, common.HashArray{headHash}, graph, nil
	}
	loaded = common.HashArray{headHash}
	if block.ParentHashes() == nil || len(block.ParentHashes()) < 1 {
		if block.Hash() == bc.genesisBlock.Hash() {
			return unloaded, loaded, common.HashArray{headHash}, graph, nil
		}
		log.Warn("Detect block without parents", "hash", block.Hash(), "height", block.Height(), "epoch", block.Epoch(), "slot", block.Slot())
		err = fmt.Errorf("Detect block without parents hash=%s, height=%d", block.Hash().Hex(), block.Height())
		return unloaded, loaded, finalized, graph, err
	}
	for _, ph := range block.ParentHashes() {
		_unloaded, _loaded, _finalized, _graph, _err := bc.ExploreChainRecursive(ph)
		unloaded = unloaded.Concat(_unloaded).Uniq()
		loaded = loaded.Concat(_loaded).Uniq()
		finalized = finalized.Concat(_finalized).Uniq()
		graph.Graph = append(graph.Graph, _graph)
		err = _err
	}
	return unloaded, loaded, finalized, graph, err
}

func (bc *BlockChain) IsAncestorRecursive(block *types.Block, ancestorHash common.Hash) bool {
	//if ancestorHash is genesis
	if bc.genesisBlock.Hash() == ancestorHash {
		return true
	}
	// if ancestorHash in parents
	if block.ParentHashes().Has(ancestorHash) {
		return true
	}
	// if ancestor not exists
	ancestor := bc.GetBlockByHash(ancestorHash)
	if ancestor == nil {
		return false
	}
	//if ancestor is finalised and block is not finalised
	if ancestor.Number() != nil && block.Number() == nil {
		return true
	}
	//if ancestor is not finalised and block is finalised
	if ancestor.Number() == nil && block.Number() != nil {
		return false
	}
	//if ancestor is finalised and block is finalised
	if ancestor.Number() != nil && block.Number() != nil {
		if ancestor.Nr() > block.Nr() {
			return false
		}
	}
	// otherwise, recursive check parents
	for _, pHash := range block.ParentHashes() {
		pBlock := bc.GetBlock(pHash)
		if pBlock != nil && bc.IsAncestorRecursive(pBlock, ancestorHash) {
			return true
		}
	}
	return false
}

//GetTips retrieves active tips headers:
// - no descendants
// - chained to finalized state (skips unchained)
func (bc *BlockChain) GetTips() types.Tips {
	tips := types.Tips{}
	for hash, dag := range *bc.hc.GetTips() {
		if dag != nil && dag.LastFinalizedHash != (common.Hash{}) {
			tips[hash] = dag
		}
	}
	return tips
}

// ResetTips set last finalized block to tips for stable work
func (bc *BlockChain) ResetTips() error {
	return bc.hc.ResetTips()
}

//AddTips add BlockDag to tips
func (bc *BlockChain) AddTips(blockDag *types.BlockDAG) {
	bc.hc.AddTips(blockDag)
}

//RemoveTips remove BlockDag from tips by hash from tips
func (bc *BlockChain) RemoveTips(hashes common.HashArray) {
	bc.hc.RemoveTips(hashes)
}

//FinalizeTips update tips in accordance with finalization result
func (bc *BlockChain) FinalizeTips(finHashes common.HashArray, lastFinHash common.Hash, lastFinNr uint64) {
	bc.hc.FinalizeTips(finHashes, lastFinHash, lastFinNr)
}

//AppendToChildren append block hash as child of block
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

func (bc *BlockChain) ReadChildren(hash common.Hash) common.HashArray {
	return rawdb.ReadChildren(bc.db, hash)
}

func (bc *BlockChain) DeleteBockDag(hash common.Hash) {
	rawdb.DeleteBlockDag(bc.db, hash)
}

func (bc *BlockChain) ReadBockDag(hash common.Hash) *types.BlockDAG {
	return rawdb.ReadBlockDag(bc.db, hash)
}

// Snapshots returns the blockchain snapshot tree.
func (bc *BlockChain) Snapshots() *snapshot.Tree {
	return bc.snaps
}

// GetLastFinalizedFastBlock retrieves the current fast-sync last finalized block of the canonical
// chain. The block is retrieved from the blockchain's internal cache.
func (bc *BlockChain) GetLastFinalizedFastBlock() *types.Block {
	lfb := bc.lastFinalizedBlock.Load().(*types.Block)
	return lfb
}

//func (bc *BlockChain) GetLastFinalizedFastBlock() types.Block {
//	return bc.lastFinalizedFastBlock.Load().(types.Block)
//}

// Validator returns the current validator.
func (bc *BlockChain) Validator() Validator {
	return bc.validator
}

// Processor returns the current processor.
func (bc *BlockChain) Processor() Processor {
	return bc.processor
}

// State returns a new mutable state based on the current HEAD block.
func (bc *BlockChain) State() (*state.StateDB, error) {
	curBlock := bc.GetLastFinalizedBlock()
	return bc.StateAt(curBlock.Root())
}

// StateAt returns a new mutable state based on a particular point in time.
func (bc *BlockChain) StateAt(root common.Hash) (*state.StateDB, error) {
	return state.New(root, bc.stateCache, bc.snaps)
}

// StateCache returns the caching database underpinning the blockchain instance.
func (bc *BlockChain) StateCache() state.Database {
	return bc.stateCache
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
			log.Info("Exporting blocks", "exported", block.Hash(), "elapsed", common.PrettyDuration(time.Since(start)))
			reported = time.Now()
		}
	}
	return nil
}

//// writeHeadBlock injects a new head block into the current block chain. This method
//// assumes that the block is indeed a true head. It will also reset the head
//// header and the head fast sync block to this very same block if they are older
//// or if they are on a different side chain.
////
//// Note, this function assumes that the `mu` mutex is held!
//func (bc *BlockChain) writeHeadBlock(block *types.Block) {
//	// If the block is on a side chain or an unknown one, force other heads onto it too
//	updateHeads := rawdb.ReadCanonicalHash(bc.db, block.NumberU64()) != block.Hash()
//
//	// Add the block to the canonical chain number scheme and mark as the head
//	batch := bc.db.NewBatch()
//	rawdb.WriteCanonicalHash(batch, block.Hash(), block.NumberU64())
//	rawdb.WriteTxLookupEntriesByBlock(batch, block)
//	rawdb.WriteHeadBlockHash(batch, block.Hash())
//
//	// If the block is better than our head or is on a different chain, force update heads
//	if updateHeads {
//		rawdb.WriteLastFinalizedHash(batch, block.Hash())
//		rawdb.WriteHeadFastBlockHash(batch, block.Hash())
//	}
//	// Flush the whole batch into the disk, exit the node if failed
//	if err := batch.Write(); err != nil {
//		log.Crit("Failed to update chain indexes and markers", "err", err)
//	}
//	// Update all in-memory chain markers in the last step
//	if updateHeads {
//		bc.hc.SetCurrentHeader(block.Header())
//		bc.currentFastBlock.Store(block)
//		headFastBlockGauge.Update(int64(block.NumberU64()))
//	}
//	bc.currentBlock.Store(block)
//	headBlockGauge.Update(int64(block.NumberU64()))
//}

// writeFinalizedBlock injects a new finalized block into the current block chain.
// Note, this function assumes that the `mu` mutex is held!
func (bc *BlockChain) writeFinalizedBlock(finNr uint64, block *types.Block, isHead bool) error {
	block.SetNumber(&finNr)

	// If the block is on a side chain or an unknown one, force other heads onto it too
	//updateHeads := rawdb.ReadCanonicalHash(bc.db, block.Nr()) != block.Hash()

	// ~Add the block to the canonical chain number scheme and mark as the head~
	// Add the block to the finalized chain number scheme
	batch := bc.db.NewBatch()
	rawdb.WriteCanonicalHash(batch, block.Hash(), block.Nr())
	rawdb.WriteTxLookupEntriesByBlock(batch, block)
	bc.CacheTransactionLookup(block)
	//todo ???
	//rawdb.WriteLastCanonicalHash(batch, block.Hash())
	rawdb.WriteFinalizedHashNumber(batch, block.Hash(), finNr)

	// update finalized number cache
	bc.hc.numberCache.Add(block.Hash(), finNr)
	bc.hc.headerCache.Remove(block.Hash())

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
		headFastBlockGauge.Update(int64(block.NumberU64()))
		headBlockGauge.Update(int64(finNr))

		bc.chainHeadFeed.Send(ChainHeadEvent{Block: block, Type: ET_NETWORK})
	}
	return nil
}

// GetLastFinalizedNumber checks if a block is fully present in the database or not.
func (bc *BlockChain) GetLastFinalizedNumber() uint64 {
	if cached, ok := bc.hc.numberCache.Get(bc.hc.lastFinalisedHash); ok {
		number := cached.(uint64)
		return number
	}
	return rawdb.ReadLastFinalizedNumber(bc.db)
}

// GetBody retrieves a block body (transactions and uncles) from the database by
// hash, caching it if found.
func (bc *BlockChain) GetBody(hash common.Hash) *types.Body {
	// Short circuit if the body's already in the cache, retrieve otherwise
	if cached, ok := bc.bodyCache.Get(hash); ok {
		body := cached.(*types.Body)
		return body
	}
	//number := bc.hc.GetBlockFinalizedNumber(hash)
	//if number == nil {
	//	return nil
	//}
	body := rawdb.ReadBody(bc.db, hash)
	if body == nil {
		return nil
	}
	// Cache the found body for next time and return
	bc.bodyCache.Add(hash, body)
	return body
}

// GetBodyRLP retrieves a block body in RLP encoding from the database by hash,
// caching it if found.
func (bc *BlockChain) GetBodyRLP(hash common.Hash) rlp.RawValue {
	// Short circuit if the body's already in the cache, retrieve otherwise
	if cached, ok := bc.bodyRLPCache.Get(hash); ok {
		return cached.(rlp.RawValue)
	}
	//number := bc.hc.GetBlockFinalizedNumber(hash)
	//if number == nil {
	//	return nil
	//}
	body := rawdb.ReadBodyRLP(bc.db, hash)
	if len(body) == 0 {
		return nil
	}
	// Cache the found body for next time and return
	bc.bodyRLPCache.Add(hash, body)
	return body
}

// HasBlock checks if a block is fully present in the database or not.
func (bc *BlockChain) HasBlock(hash common.Hash) bool {
	if bc.blockCache.Contains(hash) {
		return true
	}
	return rawdb.HasBody(bc.db, hash)
}

// HasFastBlock checks if a fast block is fully present in the database or not.
func (bc *BlockChain) HasFastBlock(hash common.Hash) bool {
	if !bc.HasBlock(hash) {
		return false
	}
	if bc.receiptsCache.Contains(hash) {
		return true
	}
	return rawdb.HasReceipts(bc.db, hash)
}

// HasState checks if state trie is fully present in the database or not.
func (bc *BlockChain) HasState(hash common.Hash) bool {
	_, err := bc.stateCache.OpenTrie(hash)
	return err == nil
}

// HasBlockAndState checks if a block and associated state trie is fully present
// in the database or not, caching it if present.
func (bc *BlockChain) HasBlockAndState(hash common.Hash, number uint64) bool {
	// Check first that the block itself is known
	block := bc.GetBlock(hash)
	if block == nil {
		return false
	}
	return bc.HasState(block.Root())
}

// GetBlock retrieves a block from the database by hash and number,
// caching it if found.
func (bc *BlockChain) GetBlock(hash common.Hash) *types.Block {
	finNr := bc.hc.GetBlockFinalizedNumber(hash)
	// Short circuit if the block's already in the cache, retrieve otherwise
	if block, ok := bc.blockCache.Get(hash); ok {
		blk := block.(*types.Block)
		blk.SetNumber(finNr)
		return blk
	}
	block := rawdb.ReadBlock(bc.db, hash)
	if block == nil {
		return nil
	}
	block.SetNumber(finNr)
	// Cache the found block for next time and return
	bc.blockCache.Add(block.Hash(), block)
	return block
}

// GetBlockByHash retrieves a block from the database by hash, caching it if found.
func (bc *BlockChain) GetBlockByHash(hash common.Hash) *types.Block {
	return bc.GetBlock(hash)
}

func (bc *BlockChain) GetBlocksByHashes(hashes common.HashArray) types.BlockMap {
	blocks := make(types.BlockMap, len(hashes))
	for _, hash := range hashes {
		blocks[hash] = bc.GetBlock(hash)
	}
	return blocks
}

// GetBlockByNumber retrieves a block from the database by number, caching it
// (associated with its hash) if found.
func (bc *BlockChain) GetBlockByNumber(number uint64) *types.Block {
	hash := rawdb.ReadCanonicalHash(bc.db, number)
	if hash == (common.Hash{}) {
		hash = rawdb.ReadFinalizedHashByNumber(bc.db, number)
		if hash == (common.Hash{}) {
			return nil
		}
	}
	block := bc.GetBlock(hash)
	if block != nil {
		block.SetNumber(&number)
	}
	return block
}

func (bc *BlockChain) ReadFinalizedHashByNumber(number uint64) common.Hash {
	return rawdb.ReadFinalizedHashByNumber(bc.db, number)
}
func (bc *BlockChain) ReadFinalizedNumberByHash(hash common.Hash) *uint64 {
	return rawdb.ReadFinalizedNumberByHash(bc.db, hash)
}

// GetBlockFinalizedNumber retrieves a block finalized height
func (bc *BlockChain) GetBlockFinalizedNumber(hash common.Hash) *uint64 {
	return bc.hc.GetBlockFinalizedNumber(hash)
}

func (bc *BlockChain) GetGenesisBlock() *types.Block {
	hash := rawdb.ReadCanonicalHash(bc.db, 0)
	if hash == (common.Hash{}) {
		return nil
	}
	return bc.GetBlock(hash)
}

// GetReceiptsByHash retrieves the receipts for all transactions in a given block.
func (bc *BlockChain) GetReceiptsByHash(hash common.Hash) types.Receipts {
	if receipts, ok := bc.receiptsCache.Get(hash); ok {
		return receipts.(types.Receipts)
	}
	//number := rawdb.ReadHeaderNumber(bc.db, hash)
	//if number == nil {
	//	return nil
	//}
	receipts := rawdb.ReadReceipts(bc.db, hash, bc.chainConfig)
	if receipts == nil {
		return nil
	}
	bc.receiptsCache.Add(hash, receipts)
	return receipts
}

// GetFinalizedBlocksFromHash returns the block corresponding to hash and up to n-1 ancestors.
// [deprecated by eth/62]
func (bc *BlockChain) GetFinalizedBlocksFromHash(hash common.Hash, n int) (blocks []*types.Block) {
	number := bc.hc.GetBlockFinalizedNumber(hash)
	if number == nil {
		return nil
	}
	for i := 0; i < n; i++ {
		block := bc.GetBlockByNumber(*number)
		if block == nil {
			break
		}
		blocks = append(blocks, block)
		*number--
	}
	return
}

// TrieNode retrieves a blob of data associated with a trie node
// either from ephemeral in-memory cache, or from persistent storage.
func (bc *BlockChain) TrieNode(hash common.Hash) ([]byte, error) {
	return bc.stateCache.TrieDB().Node(hash)
}

// ContractCode retrieves a blob of data associated with a contract hash
// either from ephemeral in-memory cache, or from persistent storage.
func (bc *BlockChain) ContractCode(hash common.Hash) ([]byte, error) {
	return bc.stateCache.ContractCode(common.Hash{}, hash)
}

// ContractCodeWithPrefix retrieves a blob of data associated with a contract
// hash either from ephemeral in-memory cache, or from persistent storage.
//
// If the code doesn't exist in the in-memory cache, check the storage with
// new code scheme.
func (bc *BlockChain) ContractCodeWithPrefix(hash common.Hash) ([]byte, error) {
	type codeReader interface {
		ContractCodeWithPrefix(addrHash, codeHash common.Hash) ([]byte, error)
	}
	return bc.stateCache.(codeReader).ContractCodeWithPrefix(common.Hash{}, hash)
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

func (bc *BlockChain) procFutureBlocks() {
	blocks := make([]*types.Block, 0, bc.futureBlocks.Len())
	for _, hash := range bc.futureBlocks.Keys() {
		if block, exist := bc.futureBlocks.Peek(hash); exist {
			blocks = append(blocks, block.(*types.Block))
		}
	}
	if len(blocks) > 0 {
		sort.Slice(blocks, func(i, j int) bool {
			return blocks[i].Nr() < blocks[j].Nr()
		})
		// Insert one by one as chain insertion needs contiguous ancestry between blocks
		for i := range blocks {
			bc.InsertChain(blocks[i : i+1])
		}
	}
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
		//if i != 0 {
		//	if blockChain[i].NumberU64() != blockChain[i-1].NumberU64()+1 || blockChain[i].ParentHash() != blockChain[i-1].Hash() {
		//		log.Error("Non contiguous receipt insert", "number", blockChain[i].Number(), "hash", blockChain[i].Hash(), "parent", blockChain[i].ParentHash(),
		//			"prevnumber", blockChain[i-1].Number(), "prevhash", blockChain[i-1].Hash())
		//		return 0, fmt.Errorf("non contiguous insert: item %d is #%d [%x..], item %d is #%d [%x..] (parent [%x..])", i-1, blockChain[i-1].NumberU64(),
		//			blockChain[i-1].Hash().Bytes()[:4], i, blockChain[i].NumberU64(), blockChain[i].Hash().Bytes()[:4], blockChain[i].ParentHash().Bytes()[:4])
		//	}
		//}
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
		if bc.GetLastFinalisedHeader().Nr() >= head.Nr() {
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
		for _, block := range blockChain {
			if bc.txLookupLimit == 0 || ancientLimit <= bc.txLookupLimit || block.NumberU64() >= ancientLimit-bc.txLookupLimit {
				rawdb.WriteTxLookupEntriesByBlock(batch, block)
			} else if rawdb.ReadTxIndexTail(bc.db) != nil {
				rawdb.WriteTxLookupEntriesByBlock(batch, block)
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
			rawdb.DeleteCanonicalHash(batch, block.Nr())
			rawdb.DeleteBlockWithoutNumber(batch, block.Hash())
		}
		//todo
		//// Delete side chain hash-to-number mappings.
		//for _, nh := range rawdb.ReadAllHashesInRange(bc.db, first.NumberU64(), last.NumberU64()) {
		//	if _, canon := canonHashes[nh.Hash]; !canon {
		//		rawdb.DeleteHeader(batch, nh.Hash, nh.Number)
		//	}
		//}
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

			rawdb.WriteTxLookupEntriesByBlock(batch, block) // Always write tx indices for live blocks, we assume they are needed
			bc.CacheTransactionLookup(block)                // Always write tx indices for live blocks, we assume they are needed

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
	//if len(liveBlocks) > 0 {
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

var lastWrite uint64

// writeBlockWithoutState writes only the block and its metadata to the database,
// but does not write any state. This is used to construct competing side forks
// up to the point where they exceed the canonical total difficulty.
func (bc *BlockChain) writeBlockWithoutState(block *types.Block, td *big.Int) (err error) {
	if bc.insertStopped() {
		return errInsertionInterrupted
	}

	batch := bc.db.NewBatch()
	rawdb.WriteBlock(batch, block)
	if err := batch.Write(); err != nil {
		log.Crit("Failed to write block into disk", "err", err)
	}
	bc.AppendToChildren(block.Hash(), block.ParentHashes())
	return nil
}

//// writeKnownBlock updates the head block flag with a known block
//// and introduces chain reorg if necessary.
//func (bc *BlockChain) writeKnownBlock(block *types.Block) error {
//	current := bc.GetLastFinalizedBlock()
//	if block.ParentHashes()[0] != current.Hash() {
//		if err := bc.reorg(current, block); err != nil {
//			return err
//		}
//	}
//	bc.writeFinalizedBlock(block)
//	return nil
//}

//// WriteBlockWithState writes the block and all associated state to the database.
//func (bc *BlockChain) WriteBlockWithState(block *types.Block, receipts []*types.Receipt, logs []*types.Log, state *state.StateDB, emitHeadEvent bool) (status WriteStatus, err error) {
//	if !bc.chainmu.TryLock() {
//		return NonStatTy, errInsertionInterrupted
//	}
//	defer bc.chainmu.Unlock()
//
//	return bc.writeBlockWithState(block, receipts, logs, state, emitHeadEvent)
//}

// WriteFinalizedBlock writes the block and all associated state to the database.
func (bc *BlockChain) WriteFinalizedBlock(finNr uint64, block *types.Block, receipts []*types.Receipt, logs []*types.Log, state *state.StateDB, isHead bool) error {
	if !bc.chainmu.TryLock() {
		return errInsertionInterrupted
	}
	defer bc.chainmu.Unlock()

	if isHead && block.Height() != finNr {
		log.Error("WriteFinalizedBlock: height!=finNr", "isHead", isHead, "finNr", finNr, "Height", block.Height(), "Hash", block.Hash().Hex())
		isHead = false
	}

	return bc.writeFinalizedBlock(finNr, block, isHead)
}

// WriteSyncDagBlock writes the dag block and all associated state to the database
//for dag synchronization process
func (bc *BlockChain) WriteSyncDagBlock(block *types.Block) (status int, err error) {
	bc.blockProcFeed.Send(true)
	defer bc.blockProcFeed.Send(false)

	// Pre-checks passed, start the full block imports
	//bc.wg.Add(1)
	if !bc.chainmu.TryLock() {
		return 0, errInsertionInterrupted
	}
	n, err := bc.insertPropagatedBlocks(types.Blocks{block}, true, true)
	bc.chainmu.Unlock()
	//bc.wg.Done()

	if len(bc.insBlockCache) > 0 {
		log.Info("Insert delayed propagated blocks", "count", len(bc.insBlockCache))
		insBlockCache := []*types.Block{}
		for _, bl := range bc.insBlockCache {
			_, insErr := bc.insertPropagatedBlocks(types.Blocks{bl}, true, false)
			if insErr == ErrInsertUncompletedDag {
				insBlockCache = append(insBlockCache, bl)
			} else if insErr != nil {
				log.Crit("Insert delayed propagated blocks error", "height", bl.Height(), "hash", bl.Hash().Hex(), "err", insErr)
			}
		}
		bc.insBlockCache = insBlockCache
	}

	return n, err
}

// WriteMinedBlock writes the block and all associated state to the database.
func (bc *BlockChain) WriteMinedBlock(block *types.Block, receipts []*types.Receipt, logs []*types.Log, state *state.StateDB) (status WriteStatus, err error) {
	if !bc.chainmu.TryLock() {
		return NonStatTy, errInsertionInterrupted
	}
	defer bc.chainmu.Unlock()

	return bc.writeBlockWithState(block, receipts, logs, state, ET_MINING, "WriteMinedBlock")
}

// WriteMinedBlock writes the block and all associated state to the database.
func (bc *BlockChain) WriteRecommitedBlock(block *types.Block, receipts []*types.Receipt, logs []*types.Log, state *state.StateDB) (status WriteStatus, err error) {
	bc.chainmu.Lock()
	defer bc.chainmu.Unlock()
	return bc.writeBlockWithState(block, receipts, logs, state, ET_SKIP, "WriteRecommitedBlock")
}

// writeBlockWithState writes the block and all associated state to the database,
// but is expects the chain mutex to be held.
func (bc *BlockChain) writeBlockWithState(block *types.Block, receipts []*types.Receipt, logs []*types.Log, state *state.StateDB, emitHeadEvent NewBlockEvtType, kind string) (status WriteStatus, err error) {
	if bc.insertStopped() {
		return NonStatTy, errInsertionInterrupted
	}

	//// Make sure no inconsistent state is leaked during insertion
	//currentBlock := bc.CurrentBlock()

	// Irrelevant of the canonical status, write the block itself to the database.
	//
	// Note all the components of block(td, hash->number map, header, body, receipts)
	// should be written atomically. BlockBatch is used for containing all components.
	blockBatch := bc.db.NewBatch()
	rawdb.WriteBlock(blockBatch, block)
	rawdb.WriteReceipts(blockBatch, block.Hash(), receipts)
	rawdb.WritePreimages(blockBatch, state.Preimages())

	rawdb.WriteTxLookupEntriesByBlock(blockBatch, block)
	bc.CacheTransactionLookup(block)

	//rawdb.WriteFinalizedHashNumber(blockBatch, block.Hash(), 0)
	//rawdb.WriteLastFinalizedHash(blockBatch, block.Hash())
	////todo fast
	////rawdb.WriteLastFinalizedFastHash(db, block.Hash())

	if err := blockBatch.Write(); err != nil {
		log.Crit("Failed to write block into disk", "err", err)
	}
	bc.AppendToChildren(block.Hash(), block.ParentHashes())
	// Commit all cached state changes into underlying memory database.
	//root, err := state.Commit(bc.chainConfig.IsEIP158(block.Number()))
	root, err := state.Commit(true)
	log.Info("BLOCK ::ParentHashes", "hash", block.Hash().Hex(), "ParentHashes", block.ParentHashes())
	log.Info("======================= BLOCK ::ROOT =====================", "root", block.Root().Hex(), "hash", block.Hash().Hex())
	log.Info("======================= Commit::ROOT =====================", "root", root.Hex(), "height", block.Height(), "Nr", block.Nr(), "kind", kind)

	if err != nil {
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
	//// If the total difficulty is higher than our known, add it to the canonical chain
	//// Second clause in the if statement reduces the vulnerability to selfish mining.
	//// Please refer to http://www.cs.cornell.edu/~ie53/publications/btcProcFC.pdf
	//reorg := externTd.Cmp(localTd) > 0
	//currentBlock = bc.CurrentBlock()
	//if !reorg && externTd.Cmp(localTd) == 0 {
	//	// Split same-difficulty blocks by number, then preferentially select
	//	// the block generated by the local miner as the canonical block.
	//	if block.NumberU64() < currentBlock.NumberU64() {
	//		reorg = true
	//	} else if block.NumberU64() == currentBlock.NumberU64() {
	//		var currentPreserve, blockPreserve bool
	//		if bc.shouldPreserve != nil {
	//			currentPreserve, blockPreserve = bc.shouldPreserve(currentBlock), bc.shouldPreserve(block)
	//		}
	//		reorg = !currentPreserve && (blockPreserve || mrand.Float64() < 0.5)
	//	}
	//}
	//if reorg {
	//	// Reorganise the chain if the parent is not the head block
	//	if block.ParentHash() != currentBlock.Hash() {
	//		if err := bc.reorg(currentBlock, block); err != nil {
	//			return NonStatTy, err
	//		}
	//	}
	//	status = CanonStatTy
	//} else {
	status = SideStatTy
	//}
	//// Set new head.
	//if status == CanonStatTy {
	//	bc.writeFinalizedBlock(block)
	//}
	bc.futureBlocks.Remove(block.Hash())

	if status == CanonStatTy {
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
	} else {
		bc.chainSideFeed.Send(ChainSideEvent{Block: block})
	}
	return status, nil
}

// addFutureBlock checks if the block is within the max allowed window to get
// accepted for future processing, and returns an error if the block is too far
// ahead and was not added.
func (bc *BlockChain) addFutureBlock(block *types.Block) error {
	max := uint64(time.Now().Unix() + maxTimeFutureBlocks)
	if block.Time() > max {
		return fmt.Errorf("future block timestamp %v > allowed %v", block.Time(), max)
	}
	bc.futureBlocks.Add(block.Hash(), block)
	return nil
}

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
			log.Error("Non contiguous block insert", "number", block.Nr(), "hash", block.Hash(), "prevnumber", prev.Nr(), "prevhash", prev.Hash())
			return 0, fmt.Errorf("non contiguous insert: item %d is #%d [%x..], item %d is #%d [%x..]", i-1, prev.Nr(),
				prev.Hash().Bytes()[:4], i, block.Nr(), block.Hash().Bytes()[:4])
		}
	}

	// Pre-checks passed, start the full block imports
	//bc.wg.Add(1)
	//defer bc.wg.Done()
	if !bc.chainmu.TryLock() {
		return 0, errChainStopped
	}
	defer bc.chainmu.Unlock()
	return bc.syncInsertChain(chain, true)
}

func (bc *BlockChain) InsertPropagatedBlocks(chain types.Blocks) (int, error) {
	// Sanity check that we have something meaningful to import
	if len(chain) == 0 {
		return 0, nil
	}

	bc.DagMu.Lock()
	defer bc.DagMu.Unlock()

	bc.blockProcFeed.Send(true)
	defer bc.blockProcFeed.Send(false)

	// Pre-checks passed, start the full block imports
	//bc.wg.Add(1)
	//defer bc.wg.Done()
	if !bc.chainmu.TryLock() {
		return 0, errChainStopped
	}
	defer bc.chainmu.Unlock()
	n, err := bc.insertPropagatedBlocks(chain, true, false)
	bc.chainmu.Unlock()
	//bc.wg.Done()

	if err == ErrInsertUncompletedDag {
		for i, bl := range chain {
			log.Info("Delay propagated block", "height", bl.Height(), "hash", bl.Hash().Hex())
			if i >= n {
				bc.insBlockCache = append(bc.insBlockCache, bl)
			}
		}
	}
	return n, err
}

// syncInsertChain is the internal implementation of SyncInsertChain, which assumes that
// 1) chains are contiguous, and 2) The chain mutex is held.
//
// This method is split out so that import batches that require re-injecting
// historical blocks can do so without releasing the lock, which could lead to
// racey behaviour. If a sidechain import is in progress, and the historic state
// is imported, but then new canon-head is added before the actual sidechain
// completes, then the historic state could be pruned again
func (bc *BlockChain) syncInsertChain(chain types.Blocks, verifySeals bool) (int, error) {
	// If the chain is terminating, don't even bother starting up
	if atomic.LoadInt32(&bc.procInterrupt) == 1 {
		return 0, nil
	}

	// Start a parallel signature recovery (signer will fluke on fork transition, minimal perf loss)
	senderCacher.recoverFromBlocks(types.MakeSigner(bc.chainConfig), chain)

	var (
		stats = insertStats{startTime: mclock.Now()}

		//lastCanon *types.Block

		maxFinNr = bc.GetLastFinalizedNumber()
	)

	//todo deprecated check ???
	//// Fire a single chain head event if we've progressed the chain
	//defer func() {
	//	lfb := bc.GetLastFinalizedBlock()
	//	if lastCanon != nil && lfb.Hash() == lastCanon.Hash() {
	//		bc.chainHeadFeed.Send(ChainHeadEvent{lastCanon, ET_SYNC_FIN})
	//	}
	//}()

	// Start the parallel header verifier
	headers := make([]*types.Header, len(chain))
	headerMap := make(types.HeaderMap, len(chain))
	seals := make([]bool, len(chain))

	for i, block := range chain {
		headers[i] = block.Header()
		headerMap[block.Hash()] = block.Header()
		seals[i] = verifySeals
		if block.Number() != nil && block.Nr() > maxFinNr {
			maxFinNr = block.Nr()
		}
	}
	abort, results := bc.engine.VerifyHeaders(bc, headerMap.ToArray(), seals)
	defer close(abort)

	// Peek the error for the first block to decide the directing import logic
	it := newInsertIterator(chain, results, bc.validator)

	block, err := it.next()

	// Left-trim all the known blocks
	if err == ErrKnownBlock {
		log.Warn("========= ERROR::ErrKnownBlock:1 ==========")
		//// First block (and state) is known
		////   1. We did a roll-back, and should now do a re-import
		////   2. The block is stored as a sidechain, and is lying about it's stateroot, and passes a stateroot
		//// 	    from the canonical chain, which has not been verified.
		//// Skip all known blocks that are behind us
		////var (
		////	current  = bc.GetLastFinalizedBlock()
		////	localTd  = bc.GetTd(current.Hash())
		////	externTd = bc.GetTd(block.ParentHashes()[0]) // The first block can't be nil
		////)
		//for block != nil && err == ErrKnownBlock {
		//	//externTd = new(big.Int).Add(externTd, block.Difficulty())
		//	//if localTd.Cmp(externTd) < 0 {
		//	//	break
		//	//}
		//	log.Debug("Ignoring already known block", "hash", block.Hash())
		//	stats.ignored++
		//
		//	block, err = it.next()
		//}
		//// The remaining blocks are still known blocks, the only scenario here is:
		//// During the fast sync, the pivot point is already submitted but rollback
		//// happens. Then node resets the head full block to a lower height via `rollback`
		//// and leaves a few known blocks in the database.
		////
		//// When node runs a fast sync again, it can re-import a batch of known blocks via
		//// `insertChain` while a part of them have higher total difficulty than current
		//// head full block(new pivot point).
		//for block != nil && err == ErrKnownBlock {
		//	log.Debug("Writing previously known block", "hash", block.Hash())
		//	if err := bc.writeKnownBlock(block); err != nil {
		//		return it.index, err
		//	}
		//	lastCanon = block
		//
		//	block, err = it.next()
		//}
		// Falls through to the block import
	}
	switch {
	// First block is pruned, insert as sidechain and reorg only if TD grows enough
	case errors.Is(err, consensus.ErrPrunedAncestor):
		log.Debug("Pruned ancestor, inserting as sidechain", "hash", block.Hash())
		return bc.insertSideChain(block, it)

	// First block is future, shove it (and all children) to the future queue (unknown ancestor)
	case errors.Is(err, consensus.ErrFutureBlock) || (errors.Is(err, consensus.ErrUnknownAncestor) && bc.futureBlocks.Contains(it.first().ParentHashes()[0])):
		for block != nil && (it.index == 0 || errors.Is(err, consensus.ErrUnknownAncestor)) {
			log.Debug("Future block, postponing import", "hash", block.Hash())
			if err := bc.addFutureBlock(block); err != nil {
				return it.index, err
			}
			block, err = it.next()
		}
		stats.queued += it.processed()
		stats.ignored += it.remaining()

		// If there are any still remaining, mark as ignored
		return it.index, err

	// Some other error occurred, abort
	case err != nil:
		bc.futureBlocks.Remove(block.Hash())
		stats.ignored += len(it.chain)
		bc.reportBlock(block, nil, err)
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

	for ; block != nil && err == nil || err == ErrKnownBlock; block, err = it.next() {
		// If the chain is terminating, stop processing blocks
		if bc.insertStopped() {
			log.Debug("Abort during block processing")
			break
		}
		// If the header is a banned one, straight out abort
		if BadHashes[block.Hash()] {
			bc.reportBlock(block, nil, ErrBannedHash)
			return it.index, ErrBannedHash
		}
		// If the block is known (in the middle of the chain), it's a special case for
		// Clique blocks where they can share state among each other, so importing an
		// older block might complete the state of the subsequent one. In this case,
		// just skip the block (we already validated it once fully (and crashed), since
		// its header and body was already in the database).
		if err == ErrKnownBlock {
			log.Warn("========= ERROR::ErrKnownBlock:2 ==========")
			log.Warn("Inserted known block", "hash", block.Hash(),
				"txs", len(block.Transactions()), "gas", block.GasUsed(),
				"root", block.Root())

			//// Special case. Commit the empty receipt slice if we meet the known
			//// block in the middle. It can only happen in the clique chain. Whenever
			//// we insert blocks via `insertSideChain`, we only commit `td`, `header`
			//// and `body` if it's non-existent. Since we don't have receipts without
			//// reexecution, so nothing to commit. But if the sidechain will be adpoted
			//// as the canonical chain eventually, it needs to be reexecuted for missing
			//// state, but if it's this special case here(skip reexecution) we will lose
			//// the empty receipt entry.
			//if len(block.Transactions()) == 0 {
			//	rawdb.WriteReceipts(bc.db, block.Hash(), nil)
			//} else {
			//	log.Error("Please file an issue, skip known block execution without receipt",
			//		"hash", block.Hash())
			//}
			//if err := bc.writeKnownBlock(block); err != nil {
			//	return it.index, err
			//}
			//stats.processed++
			//
			//// We can assume that logs are empty here, since the only way for consecutive
			//// Clique blocks to have the same state is if there are no transactions.
			//lastCanon = block
			continue
		}

		rawdb.WriteBlock(bc.db, block)
		bc.AppendToChildren(block.Hash(), block.ParentHashes())
		isHead := maxFinNr == block.Nr()
		bc.writeFinalizedBlock(block.Nr(), block, isHead)
		if err != nil {
			return it.index, err
		}
		// insertion of red blocks
		if block.Height() != block.Nr() {
			continue
		}

		//insertion of blue blocks
		start := time.Now()
		//retrieve state data
		statedb, stateBlock, recommitBlocks, stateErr := bc.CollectStateDataByFinalizedBlock(block)
		if stateErr != nil {
			return it.index, stateErr
		}
		// Enable prefetching to pull in trie node paths while processing transactions
		statedb.StartPrefetcher("chain")
		activeState = statedb

		// recommit red blocks transactions
		for _, bl := range recommitBlocks {
			statedb = bc.RecommitBlockTransactions(bl, statedb)
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
			bc.reportBlock(block, receipts, err)
			atomic.StoreUint32(&followupInterrupt, 1)

			log.Error("ERROR::syncInsertChain: 111", "height", block.Height(), "hash", block.Hash().Hex(), "err", err)

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
			bc.reportBlock(block, receipts, err)
			atomic.StoreUint32(&followupInterrupt, 1)

			log.Error("ERROR::syncInsertChain: 111", "height", block.Height(), "hash", block.Hash().Hex(), "err", err)

			return it.index, err
		}
		proctime := time.Since(start)

		// Update the metrics touched during block validation
		accountHashTimer.Update(statedb.AccountHashes) // Account hashes are complete, we can mark them
		storageHashTimer.Update(statedb.StorageHashes) // Storage hashes are complete, we can mark them

		blockValidationTimer.Update(time.Since(substart) - (statedb.AccountHashes + statedb.StorageHashes - triehash))

		// Write the block to the chain and get the status.
		substart = time.Now()
		status, err := bc.writeBlockWithState(block, receipts, logs, statedb, ET_SKIP, "syncInsertChain")
		atomic.StoreUint32(&followupInterrupt, 1)
		if err != nil {

			log.Error("ERROR::syncInsertChain: 222", "height", block.Height(), "hash", block.Hash().Hex(), "err", err)

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
			log.Error("Inserted new block :: require FIX lastCanon", "hash", block.Hash(),
				"uncles", len(block.Uncles()), "txs", len(block.Transactions()), "gas", block.GasUsed(),
				"elapsed", common.PrettyDuration(time.Since(start)),
				"root", block.Root())
			//lastCanon = block
			// Only count canonical blocks for GC processing time
			bc.gcproc += proctime

		case SideStatTy:
			log.Debug("Inserted forked block", "hash", block.Hash(),
				"diff", block.Difficulty(), "elapsed", common.PrettyDuration(time.Since(start)),
				"txs", len(block.Transactions()), "gas", block.GasUsed(), "uncles", len(block.Uncles()),
				"root", block.Root())

		default:
			// This in theory is impossible, but lets be nice to our future selves and leave
			// a log, instead of trying to track down blocks imports that don't emit logs.
			log.Warn("Inserted block with unknown status", "hash", block.Hash(),
				"diff", block.Difficulty(), "elapsed", common.PrettyDuration(time.Since(start)),
				"txs", len(block.Transactions()), "gas", block.GasUsed(), "uncles", len(block.Uncles()),
				"root", block.Root())
		}
		stats.processed++
		stats.usedGas += usedGas

		dirty, _ := bc.stateCache.TrieDB().Size()
		stats.report(chain, it.index, dirty)

		bc.AppendToChildren(block.Hash(), block.ParentHashes())
		//if block.Number() != nil {
		//	isHead := maxFinNr == *block.Number()
		//	bc.writeFinalizedBlock(*block.Number(), block, isHead)
		//}

		// update tips
		bc.RemoveTips(block.ParentHashes())
		bc.AddTips(&types.BlockDAG{
			Hash:                block.Hash(),
			Height:              block.Height(),
			LastFinalizedHash:   common.Hash{},
			LastFinalizedHeight: 0,
			DagChainHashes:      common.HashArray{}.Concat(block.ParentHashes()),
		})
	}

	// Any blocks remaining here? The only ones we care about are the future ones
	if block != nil && errors.Is(err, consensus.ErrFutureBlock) {
		if err := bc.addFutureBlock(block); err != nil {
			log.Error("ERROR::syncInsertChain: 333", "height", block.Height(), "hash", block.Hash().Hex(), "err", err)
			return it.index, err
		}
		block, err = it.next()

		for ; block != nil && errors.Is(err, consensus.ErrUnknownAncestor); block, err = it.next() {
			if err := bc.addFutureBlock(block); err != nil {
				log.Error("ERROR::syncInsertChain: 444", "height", block.Height(), "hash", block.Hash().Hex(), "err", err)
				return it.index, err
			}
			stats.queued++
		}
	}
	stats.ignored += it.remaining()

	return it.index, err
}

func (bc *BlockChain) insertPropagatedBlocks(chain types.Blocks, verifySeals bool, stateOnly bool) (int, error) {
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
	seals := make([]bool, len(chain))

	for i, block := range chain {
		headers[i] = block.Header()
		headerMap[block.Hash()] = block.Header()
		seals[i] = verifySeals
	}
	abort, results := bc.engine.VerifyHeaders(bc, headerMap.ToArray(), seals)
	defer close(abort)

	// Peek the error for the first block to decide the directing import logic
	it := newInsertIterator(chain, results, bc.validator)

	block, err := it.next()

	// Left-trim all the known blocks
	if err == ErrKnownBlock {
		log.Warn("========= ERROR::ErrKnownBlock:1 ==========")
		//// First block (and state) is known
		////   1. We did a roll-back, and should now do a re-import
		////   2. The block is stored as a sidechain, and is lying about it's stateroot, and passes a stateroot
		//// 	    from the canonical chain, which has not been verified.
		//// Skip all known blocks that are behind us
		////var (
		////	current  = bc.GetLastFinalizedBlock()
		////	localTd  = bc.GetTd(current.Hash())
		////	externTd = bc.GetTd(block.ParentHashes()[0]) // The first block can't be nil
		////)
		//for block != nil && err == ErrKnownBlock {
		//	//externTd = new(big.Int).Add(externTd, block.Difficulty())
		//	//if localTd.Cmp(externTd) < 0 {
		//	//	break
		//	//}
		//	log.Debug("Ignoring already known block", "hash", block.Hash())
		//	stats.ignored++
		//
		//	block, err = it.next()
		//}
		//// The remaining blocks are still known blocks, the only scenario here is:
		//// During the fast sync, the pivot point is already submitted but rollback
		//// happens. Then node resets the head full block to a lower height via `rollback`
		//// and leaves a few known blocks in the database.
		////
		//// When node runs a fast sync again, it can re-import a batch of known blocks via
		//// `insertChain` while a part of them have higher total difficulty than current
		//// head full block(new pivot point).
		//for block != nil && err == ErrKnownBlock {
		//	log.Debug("Writing previously known block", "hash", block.Hash())
		//	if err := bc.writeKnownBlock(block); err != nil {
		//		return it.index, err
		//	}
		//	lastCanon = block
		//
		//	block, err = it.next()
		//}
		// Falls through to the block import
	}
	switch {
	// First block is pruned, insert as sidechain and reorg only if TD grows enough
	case errors.Is(err, consensus.ErrPrunedAncestor):
		log.Debug("Pruned ancestor, inserting as sidechain", "hash", block.Hash())
		return bc.insertSideChain(block, it)

	// First block is future, shove it (and all children) to the future queue (unknown ancestor)
	case errors.Is(err, consensus.ErrFutureBlock) || (errors.Is(err, consensus.ErrUnknownAncestor) && bc.futureBlocks.Contains(it.first().ParentHashes()[0])):
		for block != nil && (it.index == 0 || errors.Is(err, consensus.ErrUnknownAncestor)) {
			log.Debug("Future block, postponing import", "hash", block.Hash())
			if err := bc.addFutureBlock(block); err != nil {
				return it.index, err
			}
			block, err = it.next()
		}
		stats.queued += it.processed()
		stats.ignored += it.remaining()

		// If there are any still remaining, mark as ignored
		return it.index, err

	// Some other error occurred, abort
	case err != nil:
		bc.futureBlocks.Remove(block.Hash())
		stats.ignored += len(it.chain)
		bc.reportBlock(block, nil, err)
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

	for ; block != nil && err == nil || err == ErrKnownBlock; block, err = it.next() {
		// If the chain is terminating, stop processing blocks
		if bc.insertStopped() {
			log.Debug("Abort during block processing")
			break
		}
		// If the header is a banned one, straight out abort
		if BadHashes[block.Hash()] {
			bc.reportBlock(block, nil, ErrBannedHash)
			return it.index, ErrBannedHash
		}
		// If the block is known (in the middle of the chain), it's a special case for
		// Clique blocks where they can share state among each other, so importing an
		// older block might complete the state of the subsequent one. In this case,
		// just skip the block (we already validated it once fully (and crashed), since
		// its header and body was already in the database).
		if err == ErrKnownBlock {
			log.Warn("========= ERROR::ErrKnownBlock:2 ==========")
			log.Warn("Inserted known block", "hash", block.Hash(),
				"uncles", len(block.Uncles()), "txs", len(block.Transactions()), "gas", block.GasUsed(),
				"root", block.Root())

			//// Special case. Commit the empty receipt slice if we meet the known
			//// block in the middle. It can only happen in the clique chain. Whenever
			//// we insert blocks via `insertSideChain`, we only commit `td`, `header`
			//// and `body` if it's non-existent. Since we don't have receipts without
			//// reexecution, so nothing to commit. But if the sidechain will be adpoted
			//// as the canonical chain eventually, it needs to be reexecuted for missing
			//// state, but if it's this special case here(skip reexecution) we will lose
			//// the empty receipt entry.
			//if len(block.Transactions()) == 0 {
			//	rawdb.WriteReceipts(bc.db, block.Hash(), nil)
			//} else {
			//	log.Error("Please file an issue, skip known block execution without receipt",
			//		"hash", block.Hash())
			//}
			//if err := bc.writeKnownBlock(block); err != nil {
			//	return it.index, err
			//}
			//stats.processed++
			//
			//// We can assume that logs are empty here, since the only way for consecutive
			//// Clique blocks to have the same state is if there are no transactions.
			//lastCanon = block
			continue
		}

		log.Info("Insert propagated block", "Height", block.Height(), "Hash", block.Hash().Hex(), "txs", len(block.Transactions()), "parents", block.ParentHashes())

		if checkBlock := bc.GetBlock(block.Hash()); checkBlock != nil && !stateOnly {
			if checkBlock.Nr() > 0 {
				log.Info("<<<<<<< CHECK BLOCK EXISTS :: block finalized >>>>>>>", "stateOnly", stateOnly, "Nr", checkBlock.Nr(), "Height", checkBlock.Height(), "Hash", checkBlock.Hash().Hex())
				stateOnly = true
				continue
			}
			if tips := bc.GetTips(); tips.GetHashes().Has(block.Hash()) || tips.GetAncestorsHashes().Has(block.Hash()) {
				stateOnly = true
			} else if children := bc.ReadChildren(block.Hash()); len(children) > 0 {
				stateOnly = true
			}
			log.Info("<<<<<<< CHECK BLOCK EXISTS >>>>>>>", "stateOnly", stateOnly, "Nr", checkBlock.Nr(), "Height", checkBlock.Height(), "Hash", checkBlock.Hash().Hex())
		}

		rawdb.WriteBlock(bc.db, block)
		bc.AppendToChildren(block.Hash(), block.ParentHashes())

		//retrieve state data
		statedb, stateBlock, recommitBlocks, stateErr := bc.CollectStateDataByBlock(block)
		if stateErr != nil {
			return it.index, stateErr
		}

		//dagChainHashes := common.HashArray{}
		//for _, bl :=  range recommitBlocks{
		//	dagChainHashes = append(dagChainHashes, bl.Hash())
		//}

		//if block.Height() > 600 && stateOnly{
		//	log.Crit("STOP")
		//}

		if !stateOnly {
			// update tips
			bc.RemoveTips(block.ParentHashes())
			bc.AddTips(&types.BlockDAG{
				Hash:                block.Hash(),
				Height:              block.Height(),
				LastFinalizedHash:   common.Hash{},
				LastFinalizedHeight: 0,
				DagChainHashes:      common.HashArray{}.Concat(block.ParentHashes()),
			})
			//if block is red - skip processing
			upTips, _ := bc.ReviseTips()
			finDag := upTips.GetFinalizingDag()
			if finDag.Hash != block.Hash() {
				log.Info("============================ Insert propagated block :: SKIP RED BLOCK ============================", "height", block.Height(), "hash", block.Hash().Hex(), "upTips", upTips.Print())
				continue
			}

			log.Info("============================ Insert propagated block :: INSERT BLUE BLOCK ============================", "height", block.Height(), "hash", block.Hash().Hex(), "upTips", upTips.GetHashes())
		}

		start := time.Now()
		// Enable prefetching to pull in trie node paths while processing transactions
		statedb.StartPrefetcher("chain")
		activeState = statedb

		// recommit red blocks transactions
		for _, bl := range recommitBlocks {
			statedb = bc.RecommitBlockTransactions(bl, statedb)
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
			bc.reportBlock(block, receipts, err)
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
			bc.reportBlock(block, receipts, err)
			atomic.StoreUint32(&followupInterrupt, 1)
			return it.index, err
		}
		proctime := time.Since(start)

		// Update the metrics touched during block validation
		accountHashTimer.Update(statedb.AccountHashes) // Account hashes are complete, we can mark them
		storageHashTimer.Update(statedb.StorageHashes) // Storage hashes are complete, we can mark them

		blockValidationTimer.Update(time.Since(substart) - (statedb.AccountHashes + statedb.StorageHashes - triehash))

		// Write the block to the chain and get the status.
		substart = time.Now()
		status, err := bc.writeBlockWithState(block, receipts, logs, statedb, ET_SKIP, "insertPropagatedBlocks")
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
			log.Debug("Inserted new block", "hash", block.Hash(),
				"uncles", len(block.Uncles()), "txs", len(block.Transactions()), "gas", block.GasUsed(),
				"elapsed", common.PrettyDuration(time.Since(start)),
				"root", block.Root())

			lastCanon = block

			// Only count canonical blocks for GC processing time
			bc.gcproc += proctime

		case SideStatTy:
			log.Debug("Inserted forked block", "hash", block.Hash(),
				"diff", block.Difficulty(), "elapsed", common.PrettyDuration(time.Since(start)),
				"txs", len(block.Transactions()), "gas", block.GasUsed(), "uncles", len(block.Uncles()),
				"root", block.Root())

		default:
			// This in theory is impossible, but lets be nice to our future selves and leave
			// a log, instead of trying to track down blocks imports that don't emit logs.
			log.Warn("Inserted block with unknown status", "hash", block.Hash(),
				"diff", block.Difficulty(), "elapsed", common.PrettyDuration(time.Since(start)),
				"txs", len(block.Transactions()), "gas", block.GasUsed(), "uncles", len(block.Uncles()),
				"root", block.Root())
		}
		stats.processed++
		stats.usedGas += usedGas

		dirty, _ := bc.stateCache.TrieDB().Size()
		stats.report(chain, it.index, dirty)

		//bc.AppendToChildren(block.Hash(), block.ParentHashes())
		//// update tips
		//bc.RemoveTips(block.ParentHashes())
		//bc.AddTips(&types.BlockDAG{
		//	Hash:                block.Hash(),
		//	LastFinalizedHash:   common.Hash{},
		//	LastFinalizedHeight: 0,
		//	DagChainHashes:      common.HashArray{}.Concat(block.ParentHashes()),
		//})
	}

	if !stateOnly {
		upTips, unloaded := bc.ReviseTips()
		if len(unloaded) == 0 {
			bc.EmitTipsSynced(upTips)
		}
	}

	// Any blocks remaining here? The only ones we care about are the future ones
	if block != nil && errors.Is(err, consensus.ErrFutureBlock) {
		if err := bc.addFutureBlock(block); err != nil {
			return it.index, err
		}
		block, err = it.next()

		for ; block != nil && errors.Is(err, consensus.ErrUnknownAncestor); block, err = it.next() {
			if err := bc.addFutureBlock(block); err != nil {
				return it.index, err
			}
			stats.queued++
		}
	}
	stats.ignored += it.remaining()

	return it.index, err
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
			log.Error("Non contiguous block insert", "number", block.Nr(), "hash", block.Hash(), "prevnumber", prev.Nr(), "prevhash", prev.Hash())
			return 0, fmt.Errorf("non contiguous insert: item %d is #%d [%x..], item %d is #%d [%x..]", i-1, prev.Nr(),
				prev.Hash().Bytes()[:4], i, block.Nr(), block.Hash().Bytes()[:4])
		}
	}

	// Pre-check passed, start the full block imports.
	if !bc.chainmu.TryLock() {
		return 0, errChainStopped
	}
	defer bc.chainmu.Unlock()
	return bc.insertChain(chain, true)
}

// InsertChainWithoutSealVerification works exactly the same
// except for seal verification, seal verification is omitted
func (bc *BlockChain) InsertChainWithoutSealVerification(block *types.Block) (int, error) {
	bc.blockProcFeed.Send(true)
	defer bc.blockProcFeed.Send(false)

	if !bc.chainmu.TryLock() {
		return 0, errChainStopped
	}
	defer bc.chainmu.Unlock()
	return bc.insertChain(types.Blocks([]*types.Block{block}), false)
}

// CollectStateDataByTips retrieves state data to calculate rootState for insertion new block to chain
// by parsing tips
func (bc *BlockChain) CollectStateDataByTips(tips types.Tips) (statedb *state.StateDB, stateBlock *types.Block, recommitBlocks []*types.Block, err error) {
	//retrieve the stable state block
	stateHash := tips.GetStableStateHash()
	if stateHash == (common.Hash{}) {
		stateHash = bc.GetLastFinalisedHeader().Hash()
	}
	stateBlock = bc.GetBlockByHash(stateHash)

	log.Info("CollectStateDataByTips State Block >>>>>>>>>>>>>>>", "height", stateBlock.Height(), "hash", stateBlock.Hash().Hex())

	statedb, err = bc.StateAt(stateBlock.Root())
	if err != nil {
		return statedb, stateBlock, recommitBlocks, err
	}
	// collect red blocks (not in finalization points)
	ordHashes := tips.GetOrderedDagChainHashes().Uniq()
	stateIndex := ordHashes.IndexOf(stateHash)
	recalcHashes := common.HashArray{}
	if stateIndex > -1 {
		recalcHashes = ordHashes[stateIndex+1:]
	}

	log.Info("CollectStateDataByTips ordHashes >>>>>>>>>>>>>>>", "ordHashes", ordHashes)
	log.Info("CollectStateDataByTips recalcHashes >>>>>>>>>", "recalcHashes", recalcHashes)

	recommitBlocks = []*types.Block{}
	for _, h := range recalcHashes {
		bl := bc.GetBlockByHash(h)
		if bl == nil {
			log.Warn("Insert block: red block not found", "block", h)
			continue
		}
		// skip already finalized blocks
		if bl.Number() != nil {
			continue
		}
		recommitBlocks = append(recommitBlocks, bl)
	}
	return statedb, stateBlock, recommitBlocks, err
}

func (bc *BlockChain) CollectStateDataByBlock(block *types.Block) (statedb *state.StateDB, stateBlock *types.Block, recommitBlocks []*types.Block, err error) {
	unl, _, _, graph, _err := bc.ExploreChainRecursive(block.Hash())
	if _err != nil {
		log.Error("ERROR::CollectStateDataByBlock", "err", _err)
		return statedb, stateBlock, recommitBlocks, err
	}
	if len(unl) > 0 {
		log.Error("ERROR::CollectStateDataByBlock", "Number", block.Nr(), "Height", block.Height(), "Hash", block.Hash().Hex(), "unloaded", unl)
		return statedb, stateBlock, recommitBlocks, ErrInsertUncompletedDag
	}

	finPoints := common.HashArray{}
	if fp := graph.GetFinalityPoints(); fp != nil {
		finPoints = *fp
	}

	log.Info("CollectStateDataByBlock State Block >>>>>>>>>>>>>>>new", "newFinPoints", finPoints)

	finPoints = finPoints.Uniq()

	var stateHash common.Hash

	if len(finPoints) > 0 {
		stateHash = []common.Hash(finPoints)[len(finPoints)-1]
	} else if lastFinAncestor := graph.GetLastFinalizedAncestor(); lastFinAncestor != nil {
		stateHash = lastFinAncestor.Hash
	}
	if stateHash == (common.Hash{}) {
		log.Error("ERROR::CollectStateDataByBlock", "error", ErrStateBlockNotFound)
		return statedb, stateBlock, recommitBlocks, ErrStateBlockNotFound
	}
	stateBlock = bc.GetBlockByHash(stateHash)

	log.Info("CollectStateDataByBlock State Block >>>>>>>>>>>>>>>", "height", stateBlock.Height(), "hash", stateBlock.Hash().Hex())

	statedb, err = bc.StateAt(stateBlock.Root())
	if err != nil {
		return statedb, stateBlock, recommitBlocks, err
	}

	// collect red blocks (not in finalization points)
	ordHashes := *graph.GetDagChainHashes()
	stateIndex := ordHashes.IndexOf(stateHash)
	recalcHashes := common.HashArray{}
	if stateIndex > -1 {
		recalcHashes = ordHashes[stateIndex+1:]
	} else {
		recalcHashes = ordHashes
	}

	log.Info("CollectStateDataByBlock ordHashes >>>>>>>>>>>>>>>", "ordHashes", ordHashes)
	log.Info("CollectStateDataByBlock recalcHashes >>>>>>>>>", "recalcHashes", recalcHashes)

	recommitBlocks = []*types.Block{}
	for _, h := range recalcHashes {
		bl := bc.GetBlockByHash(h)
		if bl == nil {
			log.Warn("Insert block: red block not found", "block", h)
			continue
		}
		recommitBlocks = append(recommitBlocks, bl)
	}
	return statedb, stateBlock, recommitBlocks, err
}

func (bc *BlockChain) CollectStateDataByFinalizedBlock(block *types.Block) (statedb *state.StateDB, stateBlock *types.Block, recommitBlocks []*types.Block, err error) {
	finNr := block.Nr()
	if finNr == 0 {
		if block.Hash() != bc.genesisBlock.Hash() {
			log.Error("Collect State Data By Finalized Block: bad block number", "nr", finNr, "height", block.Height(), "hash", block.Hash().Hex())
			return statedb, stateBlock, recommitBlocks, fmt.Errorf("Collect State Data By Finalized Block: bad block number: nr=%d (height=%d  hash=%v)", finNr, block.Height(), block.Hash().Hex())
		}
	}

	// search previous blue block and collect red blocks
	stateBlock = block
	redBlocks := []*types.Block{}
	for i := finNr - 1; i >= 0; i-- {
		prevBlock := bc.GetBlockByNumber(i)
		if prevBlock == nil {
			log.Error("Collect State Data By Finalized Block: bad finalized chain", "nr", finNr, "height", block.Height(), "hash", block.Hash().Hex())
			return statedb, stateBlock, redBlocks, ErrInsertUncompletedDag
		}
		if prevBlock.Nr() == prevBlock.Height() {
			stateBlock = prevBlock
			break
		}
		redBlocks = append(redBlocks, prevBlock)
	}
	// reverse redBlocks
	for i := len(redBlocks) - 1; i >= 0; i-- {
		recommitBlocks = append(recommitBlocks, redBlocks[i])
	}

	log.Info("============= SyncInsertChain =============", "isRed", block.Height() != block.Nr(), "Nr", block.Nr(), "Height", block.Height(), "Hash", block.Hash().Hex())

	statedb, err = bc.StateAt(stateBlock.Root())
	if err != nil {
		return statedb, stateBlock, recommitBlocks, err
	}
	return statedb, stateBlock, recommitBlocks, err
}

func (bc *BlockChain) RecommitBlockTransactions(block *types.Block, statedb *state.StateDB) *state.StateDB {

	log.Info("Recommit block >>>>>>>>>>>>>>>", "Nr", block.Nr(), "height", block.Height(), "hash", block.Hash().Hex())

	gasPool := new(GasPool).AddGas(block.GasLimit())
	signer := types.MakeSigner(bc.chainConfig)

	var coalescedLogs []*types.Log
	var receipts []*types.Receipt
	var rlogs []*types.Log

	for i, tx := range block.Transactions() {
		from, _ := types.Sender(signer, tx)
		// Start executing the transaction
		statedb.Prepare(tx.Hash(), i)

		receipt, logs, err := bc.recommitBlockTransaction(tx, statedb, block, gasPool)
		receipts = append(receipts, receipt)
		rlogs = append(rlogs, logs...)
		switch {
		case errors.Is(err, ErrGasLimitReached):
			// Pop the current out-of-gas transaction without shifting in the next from the account
			log.Trace("Gas limit exceeded for current block", "sender", from)
			//txs.Pop()

		case errors.Is(err, ErrNonceTooLow):
			// New head notification data race between the transaction pool and miner, shift
			log.Trace("Skipping transaction with low nonce", "sender", from, "nonce", tx.Nonce())
			//txs.Shift()

		case errors.Is(err, ErrNonceTooHigh):
			// Reorg notification data race between the transaction pool and miner, skip account =
			log.Trace("Skipping account with hight nonce", "sender", from, "nonce", tx.Nonce())
			//txs.Pop()

		case errors.Is(err, nil):
			// Everything ok, collect the logs and shift in the next transaction from the same account
			coalescedLogs = append(coalescedLogs, logs...)
			//w.current.tcount++

		case errors.Is(err, ErrTxTypeNotSupported):
			// Pop the unsupported transaction without shifting in the next from the account
			log.Trace("Skipping unsupported transaction type", "sender", from, "type", tx.Type())

		default:
			// Strange error, discard the transaction and get the next in line (note, the
			// nonce-too-high clause will prevent us from executing in vain).
			log.Debug("Transaction failed, account skipped", "hash", tx.Hash(), "err", err)
		}
	}

	rawdb.WriteReceipts(bc.db, block.Hash(), receipts)

	bc.chainFeed.Send(ChainEvent{Block: block, Hash: block.Hash(), Logs: rlogs})
	if len(rlogs) > 0 {
		bc.logsFeed.Send(rlogs)
	}

	return statedb
}

func (bc *BlockChain) recommitBlockTransaction(tx *types.Transaction, statedb *state.StateDB, block *types.Block, gasPool *GasPool) (*types.Receipt, []*types.Log, error) {
	snap := statedb.Snapshot()
	receipt, err := ApplyTransaction(bc.chainConfig, bc, &block.Header().Coinbase, gasPool, statedb, block.Header(), tx, &block.Header().GasUsed, *bc.GetVMConfig())
	if err != nil {
		log.Error("Recommit Block Transaction", "height", block.Height(), "hash", block.Hash().Hex(), "tx", tx.Hash().Hex())
		statedb.RevertToSnapshot(snap)
		return nil, nil, err
	}
	return receipt, receipt.Logs, nil
}

// insertChain is the internal implementation of InsertChain, which assumes that
// 1) chains are contiguous, and 2) The chain mutex is held.
//
// This method is split out so that import batches that require re-injecting
// historical blocks can do so without releasing the lock, which could lead to
// racey behaviour. If a sidechain import is in progress, and the historic state
// is imported, but then new canon-head is added before the actual sidechain
// completes, then the historic state could be pruned again
func (bc *BlockChain) insertChain(chain types.Blocks, verifySeals bool) (int, error) {
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
	seals := make([]bool, len(chain))

	for i, block := range chain {
		headers[i] = block.Header()
		headerMap[block.Hash()] = block.Header()
		seals[i] = verifySeals
	}

	abort, results := bc.engine.VerifyHeaders(bc, headerMap.ToArray(), seals)
	defer close(abort)

	// Peek the error for the first block to decide the directing import logic
	it := newInsertIterator(chain, results, bc.validator)

	block, err := it.next()

	// Left-trim all the known blocks
	if err == ErrKnownBlock {
		log.Warn("========= ERROR::ErrKnownBlock:3 ==========")

		//// First block (and state) is known
		////   1. We did a roll-back, and should now do a re-import
		////   2. The block is stored as a sidechain, and is lying about it's stateroot, and passes a stateroot
		//// 	    from the canonical chain, which has not been verified.
		//// Skip all known blocks that are behind us
		//var (
		//	current  = bc.GetLastFinalizedBlock()
		//	localTd  = bc.GetTd(current.Hash())
		//	externTd = bc.GetTd(block.ParentHashes()[0]) // The first block can't be nil
		//)
		//for block != nil && err == ErrKnownBlock {
		//	externTd = new(big.Int).Add(externTd, block.Difficulty())
		//	if localTd.Cmp(externTd) < 0 {
		//		break
		//	}
		//	log.Debug("Ignoring already known block", "hash", block.Hash())
		//	stats.ignored++
		//
		//// When node runs a fast sync again, it can re-import a batch of known blocks via
		//// `insertChain` while a part of them have higher total difficulty than current
		//// head full block(new pivot point).
		//for block != nil && err == ErrKnownBlock {
		//	log.Debug("Writing previously known block", "hash", block.Hash())
		//	if err := bc.writeKnownBlock(block); err != nil {
		//		return it.index, err
		//	}
		//	lastCanon = block
		//
		//	block, err = it.next()
		//}
		//// Falls through to the block import
	}
	switch {
	// First block is pruned, insert as sidechain and reorg only if TD grows enough
	case errors.Is(err, consensus.ErrPrunedAncestor):
		log.Debug("Pruned ancestor, inserting as sidechain", "number", block.Nr(), "hash", block.Hash().Hex())
		return bc.insertSideChain(block, it)

	// First block is future, shove it (and all children) to the future queue (unknown ancestor)
	case errors.Is(err, consensus.ErrFutureBlock) || (errors.Is(err, consensus.ErrUnknownAncestor) && bc.futureBlocks.Contains(it.first().ParentHashes()[0])):
		for block != nil && (it.index == 0 || errors.Is(err, consensus.ErrUnknownAncestor)) {
			log.Debug("Future block, postponing import", "number", block.Nr(), "hash", block.Hash())
			if err := bc.addFutureBlock(block); err != nil {
				return it.index, err
			}
			block, err = it.next()
		}
		stats.queued += it.processed()
		stats.ignored += it.remaining()

		// If there are any still remaining, mark as ignored
		return it.index, err

	// Some other error occurred, abort
	case err != nil:
		bc.futureBlocks.Remove(block.Hash())
		stats.ignored += len(it.chain)
		bc.reportBlock(block, nil, err)
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

	for ; block != nil && err == nil || err == ErrKnownBlock; block, err = it.next() {
		// If the chain is terminating, stop processing blocks
		if bc.insertStopped() {
			log.Debug("Abort during block processing")
			break
		}
		// If the header is a banned one, straight out abort
		if BadHashes[block.Hash()] {
			bc.reportBlock(block, nil, ErrBannedHash)
			return it.index, ErrBannedHash
		}
		// If the block is known (in the middle of the chain), it's a special case for
		// Clique blocks where they can share state among each other, so importing an
		// older block might complete the state of the subsequent one. In this case,
		// just skip the block (we already validated it once fully (and crashed), since
		// its header and body was already in the database).
		if err == ErrKnownBlock {
			log.Warn("========= ERROR::ErrKnownBlock:4 ==========")
			log.Warn("Inserted known block", "hash", block.Hash(),
				"txs", len(block.Transactions()), "gas", block.GasUsed(),
				"root", block.Root())

			//// Special case. Commit the empty receipt slice if we meet the known
			//// block in the middle. It can only happen in the clique chain. Whenever
			//// we insert blocks via `insertSideChain`, we only commit `td`, `header`
			//// and `body` if it's non-existent. Since we don't have receipts without
			//// reexecution, so nothing to commit. But if the sidechain will be adpoted
			//// as the canonical chain eventually, it needs to be reexecuted for missing
			//// state, but if it's this special case here(skip reexecution) we will lose
			//// the empty receipt entry.
			//if len(block.Transactions()) == 0 {
			//	rawdb.WriteReceipts(bc.db, block.Hash(), nil)
			//} else {
			//	log.Error("Please file an issue, skip known block execution without receipt",
			//		"hash", block.Hash())
			//}
			//if err := bc.writeKnownBlock(block); err != nil {
			//	return it.index, err
			//}
			//stats.processed++
			//
			//// We can assume that logs are empty here, since the only way for consecutive
			//// Clique blocks to have the same state is if there are no transactions.
			//lastCanon = block
			continue
		}
		// Retrieve the parent block and it's state to execute on top
		start := time.Now()

		rawdb.WriteBlock(bc.db, block)
		bc.AppendToChildren(block.Hash(), block.ParentHashes())

		//retrieve state data
		statedb, stateBlock, recommitBlocks, stateErr := bc.CollectStateDataByBlock(block)
		if stateErr != nil {
			return it.index, stateErr
		}
		// Enable prefetching to pull in trie node paths while processing transactions
		statedb.StartPrefetcher("chain")
		activeState = statedb

		// recommit red blocks transactions
		for _, bl := range recommitBlocks {
			statedb = bc.RecommitBlockTransactions(bl, statedb)
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
			bc.reportBlock(block, receipts, err)
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
			bc.reportBlock(block, receipts, err)
			atomic.StoreUint32(&followupInterrupt, 1)
			return it.index, err
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
			log.Debug("Inserted new block", "number", block.Nr(), "hash", block.Hash(),
				"txs", len(block.Transactions()), "gas", block.GasUsed(),
				"elapsed", common.PrettyDuration(time.Since(start)),
				"root", block.Root())

			lastCanon = block

			// Only count canonical blocks for GC processing time
			bc.gcproc += proctime

		case SideStatTy:
			log.Debug("Inserted forked block", "number", block.Nr(), "hash", block.Hash(),
				"elapsed", common.PrettyDuration(time.Since(start)),
				"txs", len(block.Transactions()), "gas", block.GasUsed(),
				"root", block.Root())

		default:
			// This in theory is impossible, but lets be nice to our future selves and leave
			// a log, instead of trying to track down blocks imports that don't emit logs.
			log.Warn("Inserted block with unknown status", "number", block.Nr(), "hash", block.Hash(),
				"elapsed", common.PrettyDuration(time.Since(start)),
				"txs", len(block.Transactions()), "gas", block.GasUsed(),
				"root", block.Root())
		}
		stats.processed++
		stats.usedGas += usedGas

		dirty, _ := bc.stateCache.TrieDB().Size()
		stats.report(chain, it.index, dirty)
	}
	// Any blocks remaining here? The only ones we care about are the future ones
	if block != nil && errors.Is(err, consensus.ErrFutureBlock) {
		if err := bc.addFutureBlock(block); err != nil {
			return it.index, err
		}
		block, err = it.next()

		for ; block != nil && errors.Is(err, consensus.ErrUnknownAncestor); block, err = it.next() {
			if err := bc.addFutureBlock(block); err != nil {
				return it.index, err
			}
			stats.queued++
		}
	}
	stats.ignored += it.remaining()

	return it.index, err
}

// insertSideChain is called when an import batch hits upon a pruned ancestor
// error, which happens when a sidechain with a sufficiently old fork-block is
// found.
//
// The method writes all (header-and-body-valid) blocks to disk, then tries to
// switch over to the new chain if the TD exceeded the current chain.
func (bc *BlockChain) insertSideChain(block *types.Block, it *insertIterator) (int, error) {
	//var (
	//externTd *big.Int
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
		//
		//		// Collect the TD of the block. Since we know it's a canon one,
		//		// we can get it directly, and not (like further below) use
		//		// the parent and then add the block on top
		//		externTd = bc.GetTd(block.Hash())
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
			log.Debug("Injected sidechain block", "number", block.Nr(), "hash", block.Hash(),
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
			log.Info("Importing heavy sidechain segment", "blocks", len(blocks), "start", blocks[0].Hash(), "end", block.Hash())
			if _, err := bc.insertChain(blocks, false); err != nil {
				return 0, err
			}
			blocks, memory = blocks[:0], 0

			// If the chain is terminating, stop processing blocks
			if bc.insertStopped() {
				log.Debug("Abort during blocks processing")
				return 0, nil
			}
		}
	}
	if len(blocks) > 0 {
		log.Info("Importing sidechain segment", "start", blocks[0].Nr(), "end", blocks[len(blocks)-1].Nr(), "start", blocks[0].Hash().Hex(), "end", blocks[len(blocks)-1].Hash().Hex())
		return bc.insertChain(blocks, false)
	}
	return 0, nil
}

// reorg takes two blocks, an old chain and a new chain and will reconstruct the
// blocks and inserts them to be part of the new canonical chain and accumulates
// potential missing transactions and post an event about them.
func (bc *BlockChain) reorg(oldBlock, newBlock *types.Block) error {
	var (
		newChain    types.Blocks
		oldChain    types.Blocks
		commonBlock *types.Block

		deletedTxs types.Transactions
		addedTxs   types.Transactions

		deletedLogs [][]*types.Log
		rebirthLogs [][]*types.Log

		// collectLogs collects the logs that were generated or removed during
		// the processing of the block that corresponds with the given hash.
		// These logs are later announced as deleted or reborn
		collectLogs = func(hash common.Hash, removed bool) {
			//number := bc.hc.GetBlockFinalizedNumber(hash)
			//if number == nil {
			//	return
			//}
			receipts := rawdb.ReadReceipts(bc.db, hash, bc.chainConfig)

			var logs []*types.Log
			for _, receipt := range receipts {
				for _, log := range receipt.Logs {
					l := *log
					if removed {
						l.Removed = true
					}
					logs = append(logs, &l)
				}
			}
			if len(logs) > 0 {
				if removed {
					deletedLogs = append(deletedLogs, logs)
				} else {
					rebirthLogs = append(rebirthLogs, logs)
				}
			}
		}
		// mergeLogs returns a merged log slice with specified sort order.
		mergeLogs = func(logs [][]*types.Log, reverse bool) []*types.Log {
			var ret []*types.Log
			if reverse {
				for i := len(logs) - 1; i >= 0; i-- {
					ret = append(ret, logs[i]...)
				}
			} else {
				for i := 0; i < len(logs); i++ {
					ret = append(ret, logs[i]...)
				}
			}
			return ret
		}
	)
	//todo
	// Reduce the longer chain to the same number as the shorter one
	//if oldBlock.NumberU64() > newBlock.NumberU64() {
	//	// Old chain is longer, gather all transactions and logs as deleted ones
	//	for ; oldBlock != nil && oldBlock.Hash() != newBlock.Hash(); oldBlock = bc.GetBlock(oldBlock.ParentHashes()[0]) {
	//		oldChain = append(oldChain, oldBlock)
	//		deletedTxs = append(deletedTxs, oldBlock.Transactions()...)
	//		collectLogs(oldBlock.Hash(), true)
	//	}
	//} else {
	// New chain is longer, stash all blocks away for subsequent insertion
	for ; newBlock != nil && newBlock.Hash() != oldBlock.Hash(); newBlock = bc.GetBlock(newBlock.ParentHashes()[0]) {
		newChain = append(newChain, newBlock)
	}
	//}
	if oldBlock == nil {
		return fmt.Errorf("invalid old chain")
	}
	if newBlock == nil {
		return fmt.Errorf("invalid new chain")
	}
	// Both sides of the reorg are at the same number, reduce both until the common
	// ancestor is found
	for {
		// If the common ancestor was found, bail out
		if oldBlock.Hash() == newBlock.Hash() {
			commonBlock = oldBlock
			break
		}
		// Remove an old block as well as stash away a new block
		oldChain = append(oldChain, oldBlock)
		deletedTxs = append(deletedTxs, oldBlock.Transactions()...)
		collectLogs(oldBlock.Hash(), true)

		newChain = append(newChain, newBlock)

		//todo
		// Step back with both chains
		oldBlock = bc.GetBlock(oldBlock.ParentHashes()[0])
		if oldBlock == nil {
			return fmt.Errorf("invalid old chain")
		}
		//todo
		newBlock = bc.GetBlock(newBlock.ParentHashes()[0])
		if newBlock == nil {
			return fmt.Errorf("invalid new chain")
		}
	}
	// Ensure the user sees large reorgs
	if len(oldChain) > 0 && len(newChain) > 0 {
		logFn := log.Info
		msg := "Chain reorg detected"
		if len(oldChain) > 63 {
			msg = "Large chain reorg detected"
			logFn = log.Warn
		}
		logFn(msg, "number", commonBlock.Nr(), "hash", commonBlock.Hash(),
			"drop", len(oldChain), "dropfrom", oldChain[0].Hash(), "add", len(newChain), "addfrom", newChain[0].Hash())
		blockReorgAddMeter.Mark(int64(len(newChain)))
		blockReorgDropMeter.Mark(int64(len(oldChain)))
		blockReorgMeter.Mark(1)
	} else {
		log.Error("Impossible reorg, please file an issue", "oldnum", oldBlock.Nr(), "oldhash", oldBlock.Hash().Hex(), "newnum", newBlock.Nr(), "newhash", newBlock.Hash().Hex())
	}

	//// Insert the new chain(except the head block(reverse order)),
	//// taking care of the proper incremental order.
	//for i := len(newChain) - 1; i >= 1; i-- {
	//	// Insert the block in the canonical way, re-writing history
	//	bc.writeFinalizedBlock(newChain[i])
	//
	//	// Collect reborn logs due to chain reorg
	//	collectLogs(newChain[i].Hash(), false)
	//
	//	// Collect the new added transactions.
	//	addedTxs = append(addedTxs, newChain[i].Transactions()...)
	//}

	// Delete useless indexes right now which includes the non-canonical
	// transaction indexes, canonical chain indexes which above the head.
	indexesBatch := bc.db.NewBatch()
	for _, tx := range types.TxDifference(deletedTxs, addedTxs) {
		rawdb.DeleteTxLookupEntry(indexesBatch, tx.Hash())
	}
	// Delete any canonical number assignments above the new head
	number := bc.GetLastFinalizedNumber()
	for i := number + 1; ; i++ {
		hash := rawdb.ReadCanonicalHash(bc.db, i)
		if hash == (common.Hash{}) {
			break
		}
		rawdb.DeleteCanonicalHash(indexesBatch, i)
	}
	if err := indexesBatch.Write(); err != nil {
		log.Crit("Failed to delete useless indexes", "err", err)
	}
	// If any logs need to be fired, do it now. In theory we could avoid creating
	// this goroutine if there are no events to fire, but realistcally that only
	// ever happens if we're reorging empty blocks, which will only happen on idle
	// networks where performance is not an issue either way.
	if len(deletedLogs) > 0 {
		bc.rmLogsFeed.Send(RemovedLogsEvent{mergeLogs(deletedLogs, true)})
	}
	if len(rebirthLogs) > 0 {
		bc.logsFeed.Send(mergeLogs(rebirthLogs, false))
	}
	if len(oldChain) > 0 {
		for i := len(oldChain) - 1; i >= 0; i-- {
			bc.chainSideFeed.Send(ChainSideEvent{Block: oldChain[i]})
		}
	}
	return nil
}

// futureBlocksLoop processes the 'future block' queue.
func (bc *BlockChain) futureBlocksLoop() {
	defer bc.wg.Done()

	futureTimer := time.NewTicker(5 * time.Second)
	defer futureTimer.Stop()
	for {
		select {
		case <-futureTimer.C:
			bc.procFutureBlocks()
		case <-bc.quit:
			return
		}
	}
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

// reportBlock logs a bad block error.
func (bc *BlockChain) reportBlock(block *types.Block, receipts types.Receipts, err error) {
	rawdb.WriteBadBlock(bc.db, block)

	var receiptString string
	for i, receipt := range receipts {
		receiptString += fmt.Sprintf("\t %d: cumulative: %v gas: %v contract: %v status: %v tx: %v logs: %v bloom: %x state: %x\n",
			i, receipt.CumulativeGasUsed, receipt.GasUsed, receipt.ContractAddress.Hex(),
			receipt.Status, receipt.TxHash.Hex(), receipt.Logs, receipt.Bloom, receipt.PostState)
	}
	log.Error(fmt.Sprintf(`
########## BAD BLOCK #########
Chain config: %v

Number: %v
Hash: 0x%x
%v

Error: %v
##############################
`, bc.chainConfig, block.Nr(), block.Hash(), receiptString, err))
}

// InsertHeaderChain attempts to insert the given header chain in to the local
// chain, possibly creating a reorg. If an error is returned, it will return the
// index number of the failing header as well an error describing what went wrong.
//
// The verify parameter can be used to fine tune whether nonce verification
// should be done or not. The reason behind the optional check is because some
// of the header retrieval mechanisms already need to verify nonces, as well as
// because nonces can be verified sparsely, not needing to check each.
func (bc *BlockChain) InsertHeaderChain(chain []*types.Header, checkFreq int) (int, error) {
	start := time.Now()
	if i, err := bc.hc.ValidateHeaderChain(chain, checkFreq); err != nil {
		return i, err
	}

	if !bc.chainmu.TryLock() {
		return 0, errChainStopped
	}
	defer bc.chainmu.Unlock()
	_, err := bc.hc.InsertHeaderChain(chain, start)
	return 0, err
}

// GetLastFinalisedHeader retrieves the current head header of the canonical chain. The
// header is retrieved from the HeaderChain's internal cache.
func (bc *BlockChain) GetLastFinalisedHeader() *types.Header {
	return bc.hc.GetLastFinalisedHeader()
}

// GetTdByHash retrieves a block's total difficulty in the canonical chain from the
// database by hash, caching it if found.
func (bc *BlockChain) GetTdByHash(hash common.Hash) *big.Int {
	return bc.hc.GetTdByHash(hash)
}

// GetHeader retrieves a block header from the database by hash and number,
// caching it if found.
func (bc *BlockChain) GetHeader(hash common.Hash) *types.Header {
	// Blockchain might have cached the whole block, only if not go to headerchain
	if block, ok := bc.blockCache.Get(hash); ok {
		return block.(*types.Block).Header()
	}

	return bc.hc.GetHeader(hash)
}

// GetHeaderByHash retrieves a block header from the database by hash, caching it if found.
func (bc *BlockChain) GetHeaderByHash(hash common.Hash) *types.Header {
	// Blockchain might have cached the whole block, only if not go to headerchain
	if block, ok := bc.blockCache.Get(hash); ok {
		return block.(*types.Block).Header()
	}

	return bc.hc.GetHeaderByHash(hash)
}

// GetHeadersByHashes retrieves a blocks headers from the database by hashes, caching it if found.
func (bc *BlockChain) GetHeadersByHashes(hashes common.HashArray) types.HeaderMap {
	headers := types.HeaderMap{}
	for _, hash := range hashes {
		headers[hash] = bc.GetHeader(hash)
	}
	return headers
}

// HasHeader checks if a block header is present in the database or not, caching
// it if present.
func (bc *BlockChain) HasHeader(hash common.Hash) bool {
	return bc.hc.HasHeader(hash)
}

// GetCanonicalHash returns the canonical hash for a given block number
func (bc *BlockChain) GetCanonicalHash(number uint64) common.Hash {
	return bc.hc.GetCanonicalHash(number)
}

// GetBlockHashesFromHash retrieves a number of block hashes starting at a given
// hash, fetching towards the genesis block.
func (bc *BlockChain) GetBlockHashesFromHash(hash common.Hash, max uint64) []common.Hash {
	return bc.hc.GetBlockHashesFromHash(hash, max)
}

// GetAncestor retrieves the Nth ancestor of a given block. It assumes that either the given block or
// a close ancestor of it is canonical. maxNonCanonical points to a downwards counter limiting the
// number of blocks to be individually checked before we reach the canonical chain.
//
// Note: ancestor == 0 returns the same block, 1 returns its parent and so on.
func (bc *BlockChain) GetAncestor(hash common.Hash, number, ancestor uint64, maxNonCanonical *uint64) (common.Hash, uint64) {
	return bc.hc.GetAncestor(hash, number, ancestor, maxNonCanonical)
}

// GetHeaderByNumber retrieves a block header from the database by number,
// caching it (associated with its hash) if found.
func (bc *BlockChain) GetHeaderByNumber(number uint64) *types.Header {
	return bc.hc.GetHeaderByNumber(number)
}

func (bc *BlockChain) SearchPrevFinalizedBlueHeader(finNr uint64) *types.Header {
	for i := finNr - 1; i > 0; i-- {
		header := bc.GetHeaderByNumber(i)
		if header.Height == i {
			return header
		}
	}
	return nil
}

// GetTransactionLookup retrieves the lookup associate with the given transaction
// hash from the cache or database.
func (bc *BlockChain) GetTransactionLookup(hash common.Hash) *rawdb.LegacyTxLookupEntry {
	// Short circuit if the txlookup already in the cache, retrieve otherwise
	if lookup, exist := bc.txLookupCache.Get(hash); exist {
		return lookup.(*rawdb.LegacyTxLookupEntry)
	}
	tx, blockHash, txIndex := rawdb.ReadTransaction(bc.db, hash)
	if tx == nil {
		return nil
	}
	lookup := &rawdb.LegacyTxLookupEntry{BlockHash: blockHash, Index: txIndex}
	bc.txLookupCache.Add(hash, lookup)
	return lookup
}

// GetTxBlockHash retrieves block hash of transaction
func (bc *BlockChain) GetTxBlockHash(txHash common.Hash) common.Hash {
	if lookup, exist := bc.txLookupCache.Get(txHash); exist {
		return lookup.(*rawdb.LegacyTxLookupEntry).BlockHash
	}
	return rawdb.ReadTxLookupEntry(bc.db, txHash)
}

// CacheTransactionLookup add to cache positions of block transactions
func (bc *BlockChain) CacheTransactionLookup(block *types.Block) {
	for i, tx := range block.Transactions() {
		lookup := &rawdb.LegacyTxLookupEntry{BlockHash: block.Hash(), Index: uint64(i)}
		bc.txLookupCache.Add(tx.Hash(), lookup)
	}
}

// Config retrieves the chain's fork configuration.
func (bc *BlockChain) Config() *params.ChainConfig { return bc.chainConfig }

// Engine retrieves the blockchain's consensus engine.
func (bc *BlockChain) Engine() consensus.Engine { return bc.engine }

// SubscribeRemovedLogsEvent registers a subscription of RemovedLogsEvent.
func (bc *BlockChain) SubscribeRemovedLogsEvent(ch chan<- RemovedLogsEvent) event.Subscription {
	return bc.scope.Track(bc.rmLogsFeed.Subscribe(ch))
}

// SubscribeChainEvent registers a subscription of ChainEvent.
func (bc *BlockChain) SubscribeChainEvent(ch chan<- ChainEvent) event.Subscription {
	return bc.scope.Track(bc.chainFeed.Subscribe(ch))
}

// SubscribeChainHeadEvent registers a subscription of ChainHeadEvent.
func (bc *BlockChain) SubscribeChainHeadEvent(ch chan<- ChainHeadEvent) event.Subscription {
	return bc.scope.Track(bc.chainHeadFeed.Subscribe(ch))
}

// SubscribeChainSideEvent registers a subscription of ChainSideEvent.
func (bc *BlockChain) SubscribeChainSideEvent(ch chan<- ChainSideEvent) event.Subscription {
	return bc.scope.Track(bc.chainSideFeed.Subscribe(ch))
}

// SubscribeLogsEvent registers a subscription of []*types.Log.
func (bc *BlockChain) SubscribeLogsEvent(ch chan<- []*types.Log) event.Subscription {
	return bc.scope.Track(bc.logsFeed.Subscribe(ch))
}

// SubscribeBlockProcessingEvent registers a subscription of bool where true means
// block processing has started while false means it has stopped.
func (bc *BlockChain) SubscribeBlockProcessingEvent(ch chan<- bool) event.Subscription {
	return bc.scope.Track(bc.blockProcFeed.Subscribe(ch))
}

//// SubscribeTipsSyncedEvent registers a subscription for tips synchronized event
//func (bc *BlockChain) SubscribeTipsSyncedEvent(ch chan<- types.Tips) event.Subscription {
//	return bc.tipsSynced.Subscribe(ch)
//	//return bc.scope.Track(bc.tipsSynced.Subscribe(ch))
//}

//// GetSlotInfo retrieves slot info
//func (bc *BlockChain) GetSlotInfo(timestamp int64) *types.SlotInfo {
//	//timestamp := time.Now().Unix()
//	period := int64(bc.chainConfig.Clique.Period)
//	spe := bc.chainConfig.Clique.SlotsPerEpoch
//	if period == 0 {
//		//default
//		period = 5
//	}
//	if spe == 0 {
//		log.Crit("unacceptable genesis: clique.slotsPerEpoch is required")
//	}
//	zerroSlotTime := int64(1633709000)
//	nodeTime := timestamp - zerroSlotTime
//	slot := nodeTime / period
//	current := nodeTime % period
//	remain := period - current
//	return &types.SlotInfo{
//		Epoch:     slot / spe,
//		Period:    period,
//		NodeTime:  nodeTime,
//		EpochSlot: slot % spe,
//		Slot:      slot,
//		Remain:    remain,
//		Current:   current,
//	}
//
//}
