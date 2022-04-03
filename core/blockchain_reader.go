// Copyright 2021 The go-ethereum Authors
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

package core

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/state/snapshot"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
)

// CurrentHeader retrieves the current head header of the canonical chain.
// depracated.
func (bc *BlockChain) CurrentHeader() *types.Header {
	//TODO implement me
	panic("implement me")
}

// GetLastFinalizedBlock retrieves the current Last Finalized block of the canonical chain. The
// block is retrieved from the blockchain's internal cache.
func (bc *BlockChain) GetLastFinalizedBlock() *types.Block {
	lfb := bc.lastFinalizedBlock.Load().(*types.Block)
	return lfb
}

// GetLastFinalizedFastBlock retrieves the current fast-sync last finalized block of the canonical
// chain. The block is retrieved from the blockchain's internal cache.
func (bc *BlockChain) GetLastFinalizedFastBlock() *types.Block {
	lfb := bc.lastFinalizedFastBlock.Load().(*types.Block)
	return lfb
}

// HasHeader checks if a block header is present in the database or not, caching
// it if present.
func (bc *BlockChain) HasHeader(hash common.Hash) bool {
	return bc.hc.HasHeader(hash)
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

// GetHeaderByNumber retrieves a block header from the database by number,
// caching it (associated with its hash) if found.
func (bc *BlockChain) GetHeaderByNumber(number uint64) *types.Header {
	return bc.hc.GetHeaderByNumber(number)
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

// GetBlock retrieves a block from the database by hash and number,
// caching it if found.
func (bc *BlockChain) GetBlock(hash common.Hash) *types.Block {
	finNr := bc.hc.GetBlockFinalizedNumber(hash)
	// Short circuit if the block's already in the cache, retrieve otherwise
	if block, ok := bc.blockCache.Get(hash); ok {
		blk := block.(*types.Block)
		if blk != nil {
			blk.SetNumber(finNr)
			return blk
		}
		bc.blockCache.Remove(hash)
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

// GetBlocksByHashes retrieves block by hash.
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
	hash := rawdb.ReadFinalizedHashByNumber(bc.db, number)
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

// ReadFinalizedHashByNumber retrieves block finalization hash by number.
func (bc *BlockChain) ReadFinalizedHashByNumber(number uint64) common.Hash {
	return rawdb.ReadFinalizedHashByNumber(bc.db, number)
}

// ReadFinalizedNumberByHash retrieves block finalization number by hash.
func (bc *BlockChain) ReadFinalizedNumberByHash(hash common.Hash) *uint64 {
	return rawdb.ReadFinalizedNumberByHash(bc.db, hash)
}

// GetBlockFinalizedNumber retrieves a block finalized height
func (bc *BlockChain) GetBlockFinalizedNumber(hash common.Hash) *uint64 {
	return bc.hc.GetBlockFinalizedNumber(hash)
}

// GetReceiptsByHash retrieves the receipts for all transactions in a given block.
func (bc *BlockChain) GetReceiptsByHash(hash common.Hash) types.Receipts {
	if receipts, ok := bc.receiptsCache.Get(hash); ok {
		return receipts.(types.Receipts)
	}
	receipts := rawdb.ReadReceipts(bc.db, hash, bc.chainConfig)
	if receipts == nil {
		return nil
	}
	bc.receiptsCache.Add(hash, receipts)
	return receipts
}

// GetLastFinalizedHeader retrieves the current head header of the canonical chain. The
// header is retrieved from the HeaderChain's internal cache.
func (bc *BlockChain) GetLastFinalizedHeader() *types.Header {
	return bc.hc.GetLastFinalizedHeader()
}

// GetHeadersByHashes retrieves a blocks headers from the database by hashes, caching it if found.
func (bc *BlockChain) GetHeadersByHashes(hashes common.HashArray) types.HeaderMap {
	headers := types.HeaderMap{}
	for _, hash := range hashes {
		headers[hash] = bc.GetHeader(hash)
	}
	return headers
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

// SearchPrevFinalizedBlueHeader searches previous finalized blue block
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

// HasState checks if state trie is fully present in the database or not.
func (bc *BlockChain) HasState(hash common.Hash) bool {
	_, err := bc.stateCache.OpenTrie(hash)
	return err == nil
}

// HasBlockAndState checks if a block and associated state trie is fully present
// in the database or not, caching it if present.
func (bc *BlockChain) HasBlockAndState(hash common.Hash) bool {
	// Check first that the block itself is known
	block := bc.GetBlock(hash)
	if block == nil {
		return false
	}
	if block.Number() != nil && block.Height() == block.Nr() {
		return bc.HasState(block.Root())
	}
	return true
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

// State returns a new mutable state based on the current HEAD block.
func (bc *BlockChain) State() (*state.StateDB, error) {
	curBlock := bc.GetLastFinalizedBlock()
	return bc.StateAt(curBlock.Root())
}

// StateAt returns a new mutable state based on a particular point in time.
func (bc *BlockChain) StateAt(root common.Hash) (*state.StateDB, error) {
	return state.New(root, bc.stateCache, bc.snaps)
}

// Config retrieves the chain's fork configuration.
func (bc *BlockChain) Config() *params.ChainConfig { return bc.chainConfig }

// Engine retrieves the blockchain's consensus engine.
func (bc *BlockChain) Engine() consensus.Engine { return bc.engine }

// Snapshots returns the blockchain snapshot tree.
func (bc *BlockChain) Snapshots() *snapshot.Tree {
	return bc.snaps
}

// Validator returns the current validator.
func (bc *BlockChain) Validator() Validator {
	return bc.validator
}

// Processor returns the current processor.
func (bc *BlockChain) Processor() Processor {
	return bc.processor
}

// StateCache returns the caching database underpinning the blockchain instance.
func (bc *BlockChain) StateCache() state.Database {
	return bc.stateCache
}

// GasLimit returns the gas limit of the current HEAD block.
func (bc *BlockChain) GasLimit() uint64 {
	curBlock := bc.GetLastFinalizedBlock()
	return curBlock.GasLimit()
}

// Genesis retrieves the chain's genesis block.
func (bc *BlockChain) Genesis() *types.Block {
	return bc.genesisBlock
}

// GetVMConfig returns the block chain VM config.
func (bc *BlockChain) GetVMConfig() *vm.Config {
	return &bc.vmConfig
}

// SetTxLookupLimit is responsible for updating the txlookup limit to the
// original one stored in db if the new mismatches with the old one.
func (bc *BlockChain) SetTxLookupLimit(limit uint64) {
	bc.txLookupLimit = limit
}

// TxLookupLimit retrieves the txlookup limit used by blockchain to prune
// stale transaction indices.
func (bc *BlockChain) TxLookupLimit() uint64 {
	return bc.txLookupLimit
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

// GetTxBlockHash retrieves statedb for recently recommited block sequence.
func (bc *BlockChain) GetCashedRecommit(chain common.HashArray) (statedb *state.StateDB, noCachedHashes, cachedHashes common.HashArray) {
	for i, _ := range chain {
		cachedHashes = chain[:i]
		key := cachedHashes.Key()
		if data, exist := bc.recommitCache.Get(key); exist {
			noCachedHashes = chain[i:]
			root := data.(common.Hash)
			if root != (common.Hash{}) {
				var err error
				statedb, err = bc.StateAt(root)
				if statedb != nil {
					log.Info("+++++ GetCashedRecommit +++++", "statedb", statedb != nil, "root", root.Hex(), "chain", chain, "cachedHashes", cachedHashes, "noCachedHashes", noCachedHashes, "len", len(chain), "i", i)
					return statedb, noCachedHashes, cachedHashes
				} else {
					log.Info("+++++ GetCashedRecommit::ERROR +++++", "statedb", statedb != nil, "root", root.Hex(), "err", err, "len", len(chain), "i", i)
				}
			}
		}
	}
	log.Info("----- GetCashedRecommit -----", "statedb", statedb, "chain", chain, "len", len(chain))
	return statedb, chain, common.HashArray{}
}

// SetCashedRecommit add to cache statedb for recommited block sequence.
func (bc *BlockChain) SetCashedRecommit(chain common.HashArray, statedb *state.StateDB, blueRoot *common.Hash) {
	if len(chain) == 0 {
		return
	}
	key := chain.Key()
	if _, exist := bc.recommitCache.Get(key); !exist {
		var (
			root common.Hash
			err  error
		)
		if blueRoot != nil && *blueRoot != (common.Hash{}) {
			root = *blueRoot
			log.Info(">>>>> GetCashedRecommit::BLUE ROOT <<<<<", "root", root.Hex(), "chain", chain)
		} else {
			root, err = statedb.Commit(false)
			if err != nil {
				return
			}
		}
		bc.recommitCache.Add(key, root)
	}
}

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
