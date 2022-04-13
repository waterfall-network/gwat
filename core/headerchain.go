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

package core

import (
	crand "crypto/rand"
	"errors"
	"fmt"
	"math"
	"math/big"
	mrand "math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	lru "github.com/hashicorp/golang-lru"
)

const (
	headerCacheLimit = 512
	numberCacheLimit = 2048
)

// HeaderChain implements the basic block header chain logic that is shared by
// core.BlockChain and light.LightChain. It is not usable in itself, only as
// a part of either structure.
//
// HeaderChain is responsible for maintaining the header chain including the
// header query and updating.
//
// The components maintained by headerchain includes: (1) total difficult
// (2) header (3) block hash -> number mapping (4) canonical number -> hash mapping
// and (5) head header flag.
//
// It is not thread safe either, the encapsulating chain structures should do
// the necessary mutex locking/unlocking.
type HeaderChain struct {
	config *params.ChainConfig

	chainDb       ethdb.Database
	genesisHeader *types.Header

	tips   atomic.Value
	tipsMu sync.RWMutex

	lastFinalisedHeader atomic.Value // Current head of the header chain (may be above the block chain!)
	lastFinalisedHash   common.Hash  // Hash of the current head of the header chain (prevent recomputing all the time)

	headerCache *lru.Cache // Cache for the most recent block headers
	numberCache *lru.Cache // Cache for the most recent block numbers

	procInterrupt func() bool

	rand   *mrand.Rand
	engine consensus.Engine
}

// NewHeaderChain creates a new HeaderChain structure. ProcInterrupt points
// to the parent's interrupt semaphore.
func NewHeaderChain(chainDb ethdb.Database, config *params.ChainConfig, engine consensus.Engine, procInterrupt func() bool) (*HeaderChain, error) {
	headerCache, _ := lru.New(headerCacheLimit)
	numberCache, _ := lru.New(numberCacheLimit)

	// Seed a fast but crypto originating random generator
	seed, err := crand.Int(crand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		return nil, err
	}

	hc := &HeaderChain{
		config:        config,
		chainDb:       chainDb,
		headerCache:   headerCache,
		numberCache:   numberCache,
		procInterrupt: procInterrupt,
		rand:          mrand.New(mrand.NewSource(seed.Int64())),
		engine:        engine,
	}

	heihgt := uint64(0)
	hc.genesisHeader = hc.GetHeaderByNumber(heihgt)
	if hc.genesisHeader == nil {
		return nil, ErrNoGenesis
	}

	hc.lastFinalisedHeader.Store(hc.genesisHeader)
	if head := rawdb.ReadLastFinalizedHash(chainDb); head != (common.Hash{}) {
		if chead := hc.GetHeaderByHash(head); chead != nil {
			heihgt = *rawdb.ReadFinalizedNumberByHash(chainDb, head)
			hc.lastFinalisedHeader.Store(chead)
		}
	}
	hc.lastFinalisedHash = hc.GetLastFinalizedHeader().Hash()
	hc.tips.Store(&types.Tips{})
	headHeaderGauge.Update(int64(heihgt))
	return hc, nil
}

// GetBlockFinalizedNumber retrieves the block number belonging to the given hash
// from the cache or database
func (hc *HeaderChain) GetBlockFinalizedNumber(hash common.Hash) *uint64 {
	if cached, ok := hc.numberCache.Get(hash); ok {
		number := cached.(uint64)
		return &number
	}
	number := rawdb.ReadFinalizedNumberByHash(hc.chainDb, hash)
	if number != nil {
		hc.numberCache.Add(hash, *number)
	}
	return number
}

type headerWriteResult struct {
	status     WriteStatus
	ignored    int
	imported   int
	lastHash   common.Hash
	lastHeader *types.Header
}

// todo fix
// WriteHeaders writes a chain of headers into the local chain, given that the parents
// are already known. If the total difficulty of the newly inserted chain becomes
// greater than the current known TD, the canonical chain is reorged.
//
// Note: This method is not concurrent-safe with inserting blocks simultaneously
// into the chain, as side effects caused by reorganisations cannot be emulated
// without the real blocks. Hence, writing headers directly should only be done
// in two scenarios: pure-header mode of operation (light clients), or properly
// separated header/block phases (non-archive clients).
func (hc *HeaderChain) writeHeaders(headers []*types.Header) (result *headerWriteResult, err error) {
	if len(headers) == 0 {
		return &headerWriteResult{}, nil
	}
	var (
		lastHash = headers[0].ParentHashes[0] // Last imported header hash

		lastHeader    *types.Header
		inserted      []numberHash // Ephemeral lookup of number/hash for the chain
		firstInserted = -1         // Index of the first non-ignored header
	)

	batch := hc.chainDb.NewBatch()
	parentKnown := true // Set to true to force hc.HasHeader check the first iteration
	for i, header := range headers {
		var hash common.Hash
		// The headers have already been validated at this point, so we already
		// know that it's a contiguous chain, where
		// headers[i].Hash() == headers[i+1].ParentHash
		if i < len(headers)-1 {
			hash = headers[i+1].ParentHashes[0]
		} else {
			hash = header.Hash()
		}
		number := header.Nr()

		// If the parent was not present, store it
		// If the header is already known, skip it, otherwise store
		alreadyKnown := parentKnown && hc.HasHeader(hash)
		if !alreadyKnown {
			rawdb.WriteHeader(batch, header)
			inserted = append(inserted, numberHash{number, hash})
			hc.headerCache.Add(hash, header)
			hc.numberCache.Add(hash, number)
			if firstInserted < 0 {
				firstInserted = i
			}
		}
		parentKnown = alreadyKnown
		lastHeader, lastHash = header, hash
	}

	// Skip the slow disk write of all headers if interrupted.
	if hc.procInterrupt() {
		log.Debug("Premature abort during headers import")
		return &headerWriteResult{}, errors.New("aborted")
	}
	// Commit to disk!
	if err := batch.Write(); err != nil {
		log.Crit("Failed to write headers", "error", err)
	}
	batch.Reset()

	var (
		status = SideStatTy
	)

	if len(inserted) == 0 {
		status = NonStatTy
	}
	return &headerWriteResult{
		status:     status,
		ignored:    len(headers) - len(inserted),
		imported:   len(inserted),
		lastHash:   lastHash,
		lastHeader: lastHeader,
	}, nil
}

func (hc *HeaderChain) ValidateHeaderChain(chain []*types.Header, checkFreq int) (int, error) {
	// Do a sanity check that the provided chain is actually ordered and linked
	for i := 1; i < len(chain); i++ {
		if chain[i].Number != nil && chain[i].Nr() != chain[i-1].Nr()+1 {
			hash := chain[i].Hash()
			parentHash := chain[i-1].Hash()
			// Chain broke ancestry, log a message (programming error) and skip insertion
			log.Error("Non contiguous header insert", "number", chain[i].Nr(), "hash", hash,
				"parent", chain[i].ParentHashes, "prevnumber", chain[i-1].Nr(), "prevhash", parentHash)

			return 0, fmt.Errorf("non contiguous insert: item %d is #%d [%x..], item %d is #%d [%x..] (parent [%x..])", i-1, chain[i-1].Nr(),
				parentHash.Bytes()[:4], i, chain[i].Number, hash.Bytes()[:4], chain[i].ParentHashes)
		}
		// If the header is a banned one, straight out abort
		for _, pHash := range chain[i].ParentHashes {
			if BadHashes[pHash] {
				return i - 1, ErrBannedHash
			}
		}
		// If it's the last header in the cunk, we need to check it too
		if i == len(chain)-1 && BadHashes[chain[i].Hash()] {
			return i, ErrBannedHash
		}
	}

	// Generate the list of seal verification requests, and start the parallel verifier
	seals := make([]bool, len(chain))
	if checkFreq != 0 {
		// In case of checkFreq == 0 all seals are left false.
		for i := 0; i <= len(seals)/checkFreq; i++ {
			index := i*checkFreq + hc.rand.Intn(checkFreq)
			if index >= len(seals) {
				index = len(seals) - 1
			}
			seals[index] = true
		}
		// Last should always be verified to avoid junk.
		seals[len(seals)-1] = true
	}

	abort, results := hc.engine.VerifyHeaders(hc, chain, seals)
	defer close(abort)

	// Iterate over the headers and ensure they all check out
	for i := range chain {
		// If the chain is terminating, stop processing blocks
		if hc.procInterrupt() {
			log.Debug("Premature abort during headers verification")
			return 0, errors.New("aborted")
		}
		// Otherwise wait for headers checks and ensure they pass
		if err := <-results; err != nil {
			return i, err
		}
	}

	return 0, nil
}

// InsertHeaderChain inserts the given headers.
//
// The validity of the headers is NOT CHECKED by this method, i.e. they need to be
// validated by ValidateHeaderChain before calling InsertHeaderChain.
//
// This insert is all-or-nothing. If this returns an error, no headers were written,
// otherwise they were all processed successfully.
//
// The returned 'write status' says if the inserted headers are part of the canonical chain
// or a side chain.
func (hc *HeaderChain) InsertHeaderChain(chain []*types.Header, start time.Time) (WriteStatus, error) {
	if hc.procInterrupt() {
		return 0, errors.New("aborted")
	}
	res, err := hc.writeHeaders(chain)

	// Report some public statistics so the user has a clue what's going on
	context := []interface{}{
		"count", res.imported,
		"elapsed", common.PrettyDuration(time.Since(start)),
	}
	if err != nil {
		context = append(context, "err", err)
	}
	if last := res.lastHeader; last != nil {
		context = append(context, "number", last.Nr(), "hash", res.lastHash)
		if timestamp := time.Unix(int64(last.Time), 0); time.Since(timestamp) > time.Minute {
			context = append(context, []interface{}{"age", common.PrettyAge(timestamp)}...)
		}
	}
	if res.ignored > 0 {
		context = append(context, []interface{}{"ignored", res.ignored}...)
	}
	log.Info("Imported new block headers", context...)
	return res.status, err
}

// GetBlockHashesFromHash retrieves a number of block hashes starting at a given
// hash, fetching towards the genesis block.
func (hc *HeaderChain) GetBlockHashesFromHash(hash common.Hash, max uint64) []common.Hash {
	// Get the origin header from which to fetch
	header := hc.GetHeaderByHash(hash)
	if header == nil {
		return nil
	}
	// Iterate the headers until enough is collected or the genesis reached
	chain := make([]common.Hash, 0, max)
	for i := uint64(0); i < max; i++ {
		next := header.ParentHashes[0]
		if header = hc.GetHeader(next); header == nil {
			break
		}
		chain = append(chain, next)
		if header.Nr() == 0 {
			break
		}
		if len(header.ParentHashes) == 0 {
			break
		}
	}
	return chain
}

// GetAncestor retrieves the Nth ancestor of a given block. It assumes that either the given block or
// a close ancestor of it is canonical. maxNonCanonical points to a downwards counter limiting the
// number of blocks to be individually checked before we reach the canonical chain.
//
// Note: ancestor == 0 returns the same block, 1 returns its parent and so on.
func (hc *HeaderChain) GetAncestor(hash common.Hash, number, ancestor uint64, maxNonCanonical *uint64) (common.Hash, uint64) {
	if ancestor > number {
		return common.Hash{}, 0
	}
	if ancestor == 1 {
		// in this case it is cheaper to just read the header
		if header := hc.GetHeaderByNumber(number - 1); header != nil {
			return header.Hash(), number - 1
		}
		return common.Hash{}, 0
	}
	for ancestor != 0 {
		if rawdb.ReadFinalizedHashByNumber(hc.chainDb, number) == hash {
			ancestorHash := rawdb.ReadFinalizedHashByNumber(hc.chainDb, number-ancestor)
			if rawdb.ReadFinalizedHashByNumber(hc.chainDb, number) == hash {
				number -= ancestor
				return ancestorHash, number
			}
		}
		if *maxNonCanonical == 0 {
			return common.Hash{}, 0
		}
		*maxNonCanonical--
		ancestor--
		header := hc.GetHeaderByNumber(number - 1)
		if header == nil {
			return common.Hash{}, 0
		}
		hash = header.Hash()
		number--
	}
	return hash, number
}

// GetHeader retrieves a block header from the database by hash, caching it if found.
func (hc *HeaderChain) GetHeader(hash common.Hash) *types.Header {
	finNr := hc.GetBlockFinalizedNumber(hash)
	// Short circuit if the header's already in the cache, retrieve otherwise
	if header, ok := hc.headerCache.Get(hash); ok {
		hdr := header.(*types.Header)
		hdr.Number = finNr
		return hdr
	}
	header := rawdb.ReadHeader(hc.chainDb, hash)
	if header == nil {
		return nil
	}
	// Cache the found header for next time and return
	hc.headerCache.Add(hash, header)
	header.Number = finNr
	return header
}

// GetHeaderByHash retrieves a block header from the database by hash, caching it if
// found.
func (hc *HeaderChain) GetHeaderByHash(hash common.Hash) *types.Header {
	return hc.GetHeader(hash)
}

func (hc *HeaderChain) GetHeadersByHashes(hashes common.HashArray) types.HeaderMap {
	headers := types.HeaderMap{}
	for _, hash := range hashes {
		headers[hash] = hc.GetHeader(hash)
	}
	return headers
}

// HasHeader checks if a block header is present in the database or not.
// In theory, if header is present in the database, all relative components
// like td and hash->number should be present too.
func (hc *HeaderChain) HasHeader(hash common.Hash) bool {
	if hc.numberCache.Contains(hash) || hc.headerCache.Contains(hash) {
		return true
	}
	return rawdb.HasHeader(hc.chainDb, hash)
}

// GetHeaderByNumber retrieves a block header from the database by number,
// caching it (associated with its hash) if found.
func (hc *HeaderChain) GetHeaderByNumber(number uint64) *types.Header {
	hash := rawdb.ReadFinalizedHashByNumber(hc.chainDb, number)
	if hash == (common.Hash{}) {
		hash = rawdb.ReadFinalizedHashByNumber(hc.chainDb, number)
	}
	if hash == (common.Hash{}) {
		return nil
	}
	return hc.GetHeader(hash)
}

func (hc *HeaderChain) GetCanonicalHash(number uint64) common.Hash {
	return rawdb.ReadFinalizedHashByNumber(hc.chainDb, number)
}

// GetLastFinalizedHeader retrieves the current head header of the canonical chain. The
// header is retrieved from the HeaderChain's internal cache.
func (hc *HeaderChain) GetLastFinalizedHeader() *types.Header {
	lastFinHead := hc.lastFinalisedHeader.Load().(*types.Header)
	if lastFinHead.Number == nil {
		finNr := hc.GetBlockFinalizedNumber(lastFinHead.Hash())
		lastFinHead.Number = finNr
	}
	return lastFinHead
}

// SetLastFinalisedHeader sets the in-memory head header marker of the canonical chan
// as the given header.
func (hc *HeaderChain) SetLastFinalisedHeader(head *types.Header, lastFinNr uint64) {
	head.Number = &lastFinNr
	hc.lastFinalisedHeader.Store(head)
	hc.lastFinalisedHash = head.Hash()
	hc.numberCache.Add(head.Hash(), lastFinNr)
	headHeaderGauge.Update(int64(lastFinNr))
}

// ResetTips set base tips for stable work
// in case synchronization not finished
func (hc *HeaderChain) ResetTips() error {
	hc.tipsMu.Lock()

	// Clear out any stale content from the caches
	hc.headerCache.Purge()
	hc.numberCache.Purge()
	if head := rawdb.ReadLastFinalizedHash(hc.chainDb); head != (common.Hash{}) {
		if chead := hc.GetHeaderByHash(head); chead != nil {
			lfnr := *rawdb.ReadFinalizedNumberByHash(hc.chainDb, head)
			hc.SetLastFinalisedHeader(chead, lfnr)
		}
	}

	lastFinHeader := hc.GetLastFinalizedHeader()
	//set genesis blockDag
	dag := &types.BlockDAG{
		Hash:                lastFinHeader.Hash(),
		Height:              lastFinHeader.Height,
		LastFinalizedHash:   lastFinHeader.Hash(),
		LastFinalizedHeight: lastFinHeader.Nr(),
		DagChainHashes:      common.HashArray{},
		FinalityPoints:      common.HashArray{},
	}
	rawdb.WriteBlockDag(hc.chainDb, dag)
	rawdb.WriteTipsHashes(hc.chainDb, common.HashArray{dag.Hash})
	rawdb.DeleteChildren(hc.chainDb, dag.Hash)

	hc.tipsMu.Unlock()
	return hc.loadTips()
}

// loadTips retrieves tips with incomplete chain to finalized state
func (hc *HeaderChain) loadTips() error {
	hc.tipsMu.Lock()
	defer hc.tipsMu.Unlock()
	tipsHashes := rawdb.ReadTipsHashes(hc.chainDb)
	if len(tipsHashes) == 0 {
		// Corrupt or empty database
		return fmt.Errorf("tips missing")
	}
	for _, th := range tipsHashes {
		if th == (common.Hash{}) {
			log.Error("Bad tips hash", "hash", th.Hex())
			continue
		}

		bdag := rawdb.ReadBlockDag(hc.chainDb, th)
		if bdag == nil {
			bdag = &types.BlockDAG{
				Hash:                th,
				Height:              0,
				LastFinalizedHash:   common.Hash{},
				LastFinalizedHeight: 0,
				DagChainHashes:      common.HashArray{},
			}
		}
		hc.AddTips(bdag, true)
	}
	return nil
}

//GetTips retrieves active tips
func (hc *HeaderChain) GetTips(skipLock ...bool) *types.Tips {
	if len(skipLock) == 0 || !skipLock[0] {
		hc.tipsMu.Lock()
		defer hc.tipsMu.Unlock()
	}
	cpy := hc.tips.Load().(*types.Tips).Copy()
	if len(cpy) > 1 {
		// if last finalised block stuck in tips - rm it
		lfHash := hc.GetLastFinalizedHeader().Hash()
		if tip := cpy.Get(lfHash); tip != nil {
			cpy = cpy.Remove(lfHash)
			hc.tips.Store(&cpy)
		}
	}
	return &cpy
}

//AddTips add BlockDag to tips
func (hc *HeaderChain) AddTips(blockDag *types.BlockDAG, skipLock ...bool) {
	if len(skipLock) == 0 || !skipLock[0] {
		hc.tipsMu.Lock()
		defer hc.tipsMu.Unlock()
	}

	tips := hc.GetTips(true)
	tips.Add(blockDag)
	hc.tips.Store(tips)
}

//RemoveTips remove BlockDag from tips by hashes from tips
func (hc *HeaderChain) RemoveTips(hashes common.HashArray, skipLock ...bool) {
	if len(skipLock) == 0 || !skipLock[0] {
		hc.tipsMu.Lock()
		defer hc.tipsMu.Unlock()
	}

	tips := hc.GetTips(true)
	for _, h := range hashes {
		tips.Remove(h)
	}
	hc.tips.Store(tips)
}

//FinalizeTips update tips in accordance with finalization result
func (hc *HeaderChain) FinalizeTips(finHashes common.HashArray, lastFinHash common.Hash, lastFinNr uint64) {
	hc.tipsMu.Lock()
	defer hc.tipsMu.Unlock()
	tips := hc.GetTips(true)
	for _, t := range *tips {
		t.DagChainHashes = t.DagChainHashes.Difference(finHashes)
		t.FinalityPoints = t.FinalityPoints.Difference(finHashes)
		// if tips is synced - update finalized data
		if t.LastFinalizedHash != (common.Hash{}) {
			t.LastFinalizedHash = lastFinHash
			t.LastFinalizedHeight = lastFinNr
		}
		tips.Add(t)
	}
	hc.tips.Store(tips)
	hc.writeCurrentTips(true)
}

// ReviseTips revise tips state
// explore chains to update tips in accordance with sync process
func (hc *HeaderChain) ReviseTips(bc *BlockChain) (tips *types.Tips, unloadedHashes common.HashArray) {
	hc.tipsMu.Lock()
	defer hc.tipsMu.Unlock()

	unloadedHashes = common.HashArray{}
	saveTips := false
	rmTips := common.HashArray{}

	curTips := *hc.GetTips(true)

	// check top hashes in chains
	ancestors := curTips.GetAncestorsHashes()
	for hash := range curTips {
		if ancestors.Has(hash) {
			curTips.Remove(hash)
		}
	}

	for hash, dag := range curTips {
		// if has children - rm from curTips
		if children := bc.ReadChildren(hash); len(children) > 0 {
			rmTips = append(rmTips, hash)
			saveTips = true
		}
		if dag == nil || dag.LastFinalizedHash == (common.Hash{}) {
			block := bc.GetBlockByHash(hash)
			// if block not loaded
			if block == nil {
				unloadedHashes = append(unloadedHashes, hash)
				continue
			}
			// if block is finalized - update curr curTips
			if nr := rawdb.ReadFinalizedNumberByHash(bc.db, hash); nr != nil {
				header := hc.GetHeader(hash)
				dag.Hash = hash
				dag.Height = header.Height
				dag.LastFinalizedHash = hash
				dag.LastFinalizedHeight = *nr
				hc.AddTips(dag, true)
				saveTips = true
				continue
			}
			// if block exists - check all ancestors to finalized state
			unloaded, loaded, finalized, graph, _, err := bc.ExploreChainRecursive(hash)
			if err != nil {
				hc.RemoveTips(common.HashArray{hash}, true)
				saveTips = true
				continue
			}

			if dag.Height == 0 {
				header := hc.GetHeader(hash)
				dag.Height = header.Height
			}

			dag.Hash = hash
			chain := dag.DagChainHashes
			chain = chain.Concat(unloaded)
			chain = chain.Concat(loaded)
			chain = chain.Difference(finalized)
			// bad order of hashes
			dag.DagChainHashes = chain.Difference(common.HashArray{hash})
			dag.FinalityPoints = dag.FinalityPoints.Difference(finalized)
			dag.FinalityPoints = dag.FinalityPoints.Difference(common.HashArray{hash}).Uniq()
			// if curTips synchronized
			if len(unloaded) == 0 {
				dag.LastFinalizedHash = bc.GetLastFinalizedBlock().Hash()
				dag.LastFinalizedHeight = bc.GetLastFinalizedNumber()
				dag.DagChainHashes = *graph.GetDagChainHashes()
				//dag.FinalityPoints = *graph.GetFinalityPointsByLastFinNr(dag.LastFinalizedHeight)
				dag.FinalityPoints = *graph.GetFinalityPoints()
				hc.AddTips(dag, true)
			} else {
				log.Warn("Unknown blocks detected", "hashes", unloaded)
			}
			saveTips = true
		}
	}
	hc.RemoveTips(rmTips, true)

	hc.writeCurrentTips(true)
	if saveTips {
	}
	return hc.GetTips(true), unloadedHashes
}

//WriteCurrentTips save current tips blockDags and hashes
func (hc *HeaderChain) writeCurrentTips(skipLock ...bool) {
	if len(skipLock) == 0 || !skipLock[0] {
		hc.tipsMu.Lock()
		defer hc.tipsMu.Unlock()
	}
	batch := hc.chainDb.NewBatch()
	tips := hc.GetTips(true)
	//1. save blockDags of tips
	for _, dag := range *tips {
		rawdb.WriteBlockDag(batch, dag)
	}
	//2. save tips hashes
	rawdb.WriteTipsHashes(batch, tips.GetHashes())
	if err := batch.Write(); err != nil {
		log.Crit("Failed to write tips", "error", err)
	}
}

type (
	// UpdateHeadBlocksCallback is a callback function that is called by SetHead
	// before head header is updated. The method will return the actual block it
	// updated the head to (missing state) and a flag if setHead should continue
	// rewinding till that forcefully (exceeded ancient limits)
	UpdateHeadBlocksCallback func(ethdb.KeyValueWriter, *types.Header) (common.Hash, bool)

	// DeleteBlockContentCallback is a callback function that is called by SetHead
	// before each header is deleted.
	DeleteBlockContentCallback func(ethdb.KeyValueWriter, common.Hash)
)

// SetHead rewinds the local chain to a new head. Everything above the new head
// will be deleted and the new one set.
func (hc *HeaderChain) SetHead(headHash common.Hash, updateFn UpdateHeadBlocksCallback, delFn DeleteBlockContentCallback) {
	var (
		parentHash common.Hash
		batch      = hc.chainDb.NewBatch()
		origin     = true
		lastFinNr  = hc.GetBlockFinalizedNumber(headHash)
	)

	for hdrHeight := hc.GetBlockFinalizedNumber(hc.GetLastFinalizedHeader().Hash()); hdrHeight != nil && lastFinNr != nil && *hdrHeight > *lastFinNr; hdrHeight = hc.GetBlockFinalizedNumber(hc.GetLastFinalizedHeader().Hash()) {
		hdr := hc.GetLastFinalizedHeader()
		num := *hdrHeight

		// Rewind block chain to new head.
		parent := hc.GetHeaderByNumber(num - 1)
		if parent == nil {
			parent = hc.genesisHeader
		}
		parentHash = parent.Hash()

		// Notably, since geth has the possibility for setting the head to a low
		// height which is even lower than ancient head.
		// In order to ensure that the head is always no higher than the data in
		// the database (ancient store or active store), we need to update head
		// first then remove the relative data from the database.
		//
		// Update head first(head fast block, head full block) before deleting the data.
		markerBatch := hc.chainDb.NewBatch()
		if updateFn != nil {
			newHead, force := updateFn(markerBatch, parent)
			newHeadHeight := hc.GetBlockFinalizedNumber(newHead)
			if force && newHeadHeight != nil && *newHeadHeight < *lastFinNr {
				log.Warn("Force rewinding till ancient limit", "head", newHeadHeight)
				lastFinNr = newHeadHeight
			}
		}
		// Update head header then.
		rawdb.WriteLastFinalizedHash(markerBatch, parentHash)
		if err := markerBatch.Write(); err != nil {
			log.Crit("Failed to update chain markers", "error", err)
		}
		hc.lastFinalisedHeader.Store(parent)
		hc.lastFinalisedHash = parentHash
		headHeaderGauge.Update(int64(num - 1))

		// If this is the first iteration, wipe any leftover data upwards too so
		// we don't end up with dangling daps in the database
		var nums []uint64
		if origin {
			for n, exists := num+1, true; exists; n++ {
				fnh := rawdb.ReadFinalizedHashByNumber(hc.chainDb, n)
				exists = fnh != (common.Hash{})
				if exists {
					nums = append([]uint64{n}, nums...)
				}
			}
			origin = false
		}
		nums = append(nums, num)

		// Remove the related data from the database on all sidechains
		for _, num := range nums {
			// Gather all the side fork hashes
			//hashes := rawdb.ReadAllHashes(hc.chainDb, num)
			hashes := []common.Hash{rawdb.ReadFinalizedHashByNumber(hc.chainDb, num)}
			if len(hashes) == 0 {
				// No hashes in the database whatsoever, probably frozen already
				hashes = append(hashes, hdr.Hash())
			}
			for _, hash := range hashes {
				if delFn != nil {
					delFn(batch, hash)
				}
				rawdb.DeleteHeader(batch, hash, &num)
				rawdb.DeleteFinalizedHashNumber(batch, hash, num)
			}
		}
	}
	// Flush all accumulated deletions.
	if err := batch.Write(); err != nil {
		log.Crit("Failed to rewind block", "error", err)
	}
	// Clear out any stale content from the caches
	hc.headerCache.Purge()
	hc.numberCache.Purge()
}

// SetGenesis sets a new genesis block header for the chain
func (hc *HeaderChain) SetGenesis(head *types.Header) {
	hc.genesisHeader = head
}

// Config retrieves the header chain's chain configuration.
func (hc *HeaderChain) Config() *params.ChainConfig { return hc.config }

// Engine retrieves the header chain's consensus engine.
func (hc *HeaderChain) Engine() consensus.Engine { return hc.engine }

// GetBlock implements consensus.ChainReader, and returns nil for every input as
// a header chain does not have blocks available for retrieval.
func (hc *HeaderChain) GetBlock(hash common.Hash) *types.Block {
	return nil
}
