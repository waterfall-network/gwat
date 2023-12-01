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

	lru "github.com/hashicorp/golang-lru"
	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/rawdb"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/types"
	"gitlab.waterfall.network/waterfall/protocol/gwat/ethdb"
	"gitlab.waterfall.network/waterfall/protocol/gwat/log"
	"gitlab.waterfall.network/waterfall/protocol/gwat/params"
)

const (
	ancestorCacheLimit = 32 * 8 * 4 //slotsPerEpoch * blocksPerSlot * epochs
	headerCacheLimit   = 32 * 8 * 8 //slotsPerEpoch * blocksPerSlot * epochs
	numberCacheLimit   = 2048
	blockDagCacheLimit = 32
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

	headerCache   *lru.Cache // Cache for the most recent block headers
	numberCache   *lru.Cache // Cache for the most recent block numbers
	ancestorCache *lru.Cache // Cache for ancestors
	blockDagCache *lru.Cache // Blockdag cache

	procInterrupt func() bool

	rand *mrand.Rand

	procRollback int32
}

// NewHeaderChain creates a new HeaderChain structure. ProcInterrupt points
// to the parent's interrupt semaphore.
func NewHeaderChain(chainDb ethdb.Database, config *params.ChainConfig, procInterrupt func() bool) (*HeaderChain, error) {
	ancestorCache, _ := lru.New(ancestorCacheLimit)
	headerCache, _ := lru.New(headerCacheLimit)
	numberCache, _ := lru.New(numberCacheLimit)
	blockDagCache, _ := lru.New(blockDagCacheLimit)

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
		ancestorCache: ancestorCache,
		blockDagCache: blockDagCache,
	}

	hc.genesisHeader = hc.GetHeaderByNumber(0)
	if hc.genesisHeader == nil {
		return nil, ErrNoGenesis
	}

	lfNr := uint64(0)
	hc.lastFinalisedHeader.Store(hc.genesisHeader)
	if head := rawdb.ReadLastFinalizedHash(chainDb); head != (common.Hash{}) {
		if chead := hc.GetHeaderByHash(head); chead != nil {
			if pNr := rawdb.ReadFinalizedNumberByHash(chainDb, head); pNr != nil {
				lfNr = *pNr
				hc.lastFinalisedHeader.Store(chead)
			}
		}
	}
	hc.lastFinalisedHash = hc.GetLastFinalizedHeader().Hash()
	hc.tips.Store(&types.Tips{})
	headHeaderGauge.Update(int64(lfNr))
	return hc, nil
}

// SetRollbackActive set flag of rollback proc is running.
func (hc *HeaderChain) SetRollbackActive() {
	atomic.StoreInt32(&hc.procRollback, 1)
}

// ResetRollbackActive reset flag of rollback proc running.
func (hc *HeaderChain) ResetRollbackActive() {
	atomic.StoreInt32(&hc.procRollback, 0)
}

// IsRollbackActive returns true if rollback proc is running.
func (hc *HeaderChain) IsRollbackActive() bool {
	return atomic.LoadInt32(&hc.procRollback) == 1
}

// GetBlockFinalizedNumber retrieves the block number belonging to the given hash
// from the cache or database
func (hc *HeaderChain) GetBlockFinalizedNumber(hash common.Hash) *uint64 {
	if cached, ok := hc.numberCache.Get(hash); ok {
		number := cached.(uint64)
		if number > 0 {
			return &number
		}
	}
	number := rawdb.ReadFinalizedNumberByHash(hc.chainDb, hash)
	if number != nil {
		hc.numberCache.Remove(hash)
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
			hc.headerCache.Remove(hash)
			hc.headerCache.Add(hash, header)
			hc.numberCache.Remove(hash)
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

func (hc *HeaderChain) ValidateHeaderChain(chain []*types.Header) (int, error) {
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
	if header, ok := hc.headerCache.Get(hash); ok && header != nil {
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
	hc.numberCache.Remove(head.Hash())
	hc.numberCache.Add(head.Hash(), lastFinNr)
	headHeaderGauge.Update(int64(lastFinNr))
}

// ResetTips set last finalized and not merged forks as tips and
// clean all deprecated tips data.
func (hc *HeaderChain) ResetTips() error {
	hc.tipsMu.Lock()
	defer hc.tipsMu.Unlock()

	// Clear out any stale content from the caches
	hc.headerCache.Purge()
	hc.numberCache.Purge()

	lfHash := rawdb.ReadLastFinalizedHash(hc.chainDb)
	if lfHash == (common.Hash{}) {
		log.Error("last finalized hash not found")
		return errors.New("last finalized hash not found")
	}
	head := hc.GetHeaderByHash(lfHash)
	if head == nil {
		log.Error("nil last finalized header")
		return errors.New("nil last finalized header")
	}
	lfnr := *rawdb.ReadFinalizedNumberByHash(hc.chainDb, lfHash)
	if lfnr == 0 && head.Height > 0 {
		log.Error("last finalized header number not found")
		return errors.New("last finalized header number not found")
	}
	hc.SetLastFinalisedHeader(head, lfnr)

	// remove all blockDags
	hc.ClearBlockDag()

	var (
		cpHeader     *types.Header
		isCpAncestor bool
		ancestors    types.HeaderMap
		unloaded     common.HashArray
		err          error
	)

	lastFinHeader := hc.GetLastFinalizedHeader()
	if lastFinHeader.Hash() == hc.genesisHeader.Hash() {
		isCpAncestor = true
		ancestors = types.HeaderMap{}
		cpHeader = lastFinHeader
	} else {
		cpHeader = hc.GetHeader(lastFinHeader.CpHash)
		isCpAncestor, ancestors, unloaded, err = hc.CollectAncestorsAftCpByParents(lastFinHeader.ParentHashes, cpHeader)
	}
	if err != nil {
		return err
	}
	if len(unloaded) > 0 {
		return ErrInsertUncompletedDag
	}
	if !isCpAncestor {
		return ErrCpIsnotAncestor
	}
	delete(ancestors, lastFinHeader.CpHash)

	//set head blockDag
	dag := &types.BlockDAG{
		Hash:                   lastFinHeader.Hash(),
		Height:                 lastFinHeader.Height,
		Slot:                   lastFinHeader.Slot,
		CpHash:                 cpHeader.Hash(),
		CpHeight:               cpHeader.Height,
		OrderedAncestorsHashes: ancestors.Hashes(),
	}
	hc.SaveBlockDag(dag)
	tipsHashes := common.HashArray{dag.Hash}

	//search forks and add to tips
	for nr := cpHeader.Nr() + 1; nr < lastFinHeader.Nr(); nr++ {
		hdr := hc.GetHeaderByNumber(nr)
		if hdr == nil {
			return errBlockNotFound
		}
		isFork := !dag.OrderedAncestorsHashes.Has(hdr.Hash())
		if !isFork {
			continue
		}
		hdrCpHeader := hc.GetHeader(hdr.CpHash)
		isCpAncestor, ancestors, unloaded, err = hc.CollectAncestorsAftCpByParents(hdr.ParentHashes, hdrCpHeader)
		if err != nil {
			return err
		}
		if len(unloaded) > 0 {
			return ErrInsertUncompletedDag
		}
		if !isCpAncestor {
			return ErrCpIsnotAncestor
		}
		delete(ancestors, hdr.CpHash)
		//set head blockDag
		hdrDag := &types.BlockDAG{
			Hash:                   hdr.Hash(),
			Height:                 hdr.Height,
			Slot:                   hdr.Slot,
			CpHash:                 hdr.CpHash,
			CpHeight:               hdrCpHeader.Height,
			OrderedAncestorsHashes: ancestors.Hashes(),
		}
		hc.SaveBlockDag(hdrDag)
		tipsHashes = append(tipsHashes, hdr.Hash())
	}
	rawdb.WriteTipsHashes(hc.chainDb, tipsHashes)

	return hc.loadTips(true)
}

// ClearBlockDag removes all BlockDag records
func (hc *HeaderChain) ClearBlockDag() {
	dagHashes := rawdb.ReadAllBlockDagHashes(hc.chainDb)
	for _, hash := range dagHashes {
		rawdb.DeleteBlockDag(hc.chainDb, hash)
	}
}

// loadTips retrieves tips from db and set caches.
func (hc *HeaderChain) loadTips(skipLock ...bool) error {
	if len(skipLock) == 0 || !skipLock[0] {
		hc.tipsMu.Lock()
		defer hc.tipsMu.Unlock()
	}
	tipsHashes := rawdb.ReadTipsHashes(hc.chainDb)
	if len(tipsHashes) == 0 {
		// Corrupt or empty database
		return fmt.Errorf("tips missing")
	}
	curTips := &types.Tips{}
	for _, th := range tipsHashes {
		if th == (common.Hash{}) {
			log.Error("Bad tips hash", "hash", th.Hex())
			continue
		}
		// rm finalized blocks from tips
		tipHeader := hc.GetHeader(th)
		if tipHeader == nil {
			return fmt.Errorf("tips header not found")
		}
		bdag := hc.GetBlockDag(th)
		if bdag == nil {
			return fmt.Errorf("block dag not found")
		}
		curTips.Add(bdag)
	}
	// check top hashes in chains
	ancestors := curTips.GetAncestorsHashes()
	for hash := range *curTips {
		if ancestors.Has(hash) {
			curTips.Remove(hash)
		}
	}
	hc.tips.Store(curTips)
	hc.writeCurrentTips(true)
	return nil
}

// GetTips retrieves active tips
func (hc *HeaderChain) GetTips(skipLock ...bool) *types.Tips {
	if len(skipLock) == 0 || !skipLock[0] {
		hc.tipsMu.Lock()
		defer hc.tipsMu.Unlock()
	}
	cpy := hc.tips.Load().(*types.Tips).Copy()
	finTips := make(types.Tips, len(cpy))
	for h, t := range cpy {
		nr := hc.GetBlockFinalizedNumber(h)
		if nr != nil {
			finTips = finTips.Add(t)
			cpy = cpy.Remove(h)
			continue
		}
	}
	if len(cpy) == 0 {
		return &finTips
	}
	hc.tips.Store(&cpy)
	return &cpy
}

// AddTips add BlockDag to tips
func (hc *HeaderChain) AddTips(blockDag *types.BlockDAG, skipLock ...bool) {
	if len(skipLock) == 0 || !skipLock[0] {
		hc.tipsMu.Lock()
		defer hc.tipsMu.Unlock()
	}

	tips := hc.GetTips(true)
	tips.Add(blockDag)
	hc.tips.Store(tips)
}

// RemoveTips remove BlockDag from tips by hashes from tips
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

// FinalizeTips update tips in accordance with finalization result
func (hc *HeaderChain) FinalizeTips(finHashes common.HashArray, lastFinHash common.Hash, lastFinNr uint64) {
	hc.tipsMu.Lock()
	defer hc.tipsMu.Unlock()
	tips := hc.GetTips(true)

	for _, t := range *tips {
		tHeader := hc.GetHeaderByHash(t.Hash)
		// if tip isn't finalized - update it
		if tHeader.Nr() == 0 && tHeader.Height > 0 {
			continue
		}
		// if tip isn't finalized - rm it
		if tHeader.Nr() <= lastFinNr {
			tips.Remove(t.Hash)
		}
	}

	//if tips is empty - set last fin block
	if len(*tips) == 0 {
		bdag := hc.GetBlockDag(lastFinHash)
		if bdag == nil {
			tHeader := hc.GetHeaderByHash(lastFinHash)
			cpHeader := hc.GetHeaderByHash(tHeader.CpHash)
			_, ancestors, _, _ := hc.CollectAncestorsAftCpByParents(tHeader.ParentHashes, cpHeader)
			delete(ancestors, cpHeader.Hash())
			bdag = &types.BlockDAG{
				Hash:                   tHeader.Hash(),
				Height:                 tHeader.Height,
				Slot:                   tHeader.Slot,
				CpHash:                 tHeader.CpHash,
				CpHeight:               cpHeader.Height,
				OrderedAncestorsHashes: ancestors.Hashes(),
			}
		}
		tips.Add(bdag)
	}

	hc.tips.Store(tips)
	hc.writeCurrentTips(true)
}

// WriteCurrentTips save current tips blockDags and hashes
func (hc *HeaderChain) writeCurrentTips(skipLock ...bool) {
	if len(skipLock) == 0 || !skipLock[0] {
		hc.tipsMu.Lock()
		defer hc.tipsMu.Unlock()
	}
	batch := hc.chainDb.NewBatch()
	tips := hc.GetTips(true)
	//1. save blockDags of tips
	for _, dag := range *tips {
		hc.SaveBlockDag(dag)
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
// Deprecated
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

// GetBlock implements consensus.ChainReader, and returns nil for every input as
// a header chain does not have blocks available for retrieval.
func (hc *HeaderChain) GetBlock(hash common.Hash) *types.Block {
	return nil
}

type CollectAncestorsResult struct {
	cpHash       common.Hash
	isCpAncestor bool
	ancestors    types.HeaderMap
	unloaded     common.HashArray
	cache        CollectAncestorsResultMap
	err          error
}
type CollectAncestorsResultMap map[common.Hash]*CollectAncestorsResult

// CollectAncestorsAftCpByTips collect ancestors by block parent tips
// which have to be finalized after checkpoint up to block.
// the method is not recursive, and strong depends on correct state of tips.
func (hc *HeaderChain) CollectAncestorsAftCpByTips(parents common.HashArray, cpHash common.Hash) (
	isCpAncestor bool,
	ancestors types.HeaderMap,
	unloaded common.HashArray,
	tips types.Tips,
) {
	defer func(start time.Time) {
		log.Debug("TIME",
			"elapsed", common.PrettyDuration(time.Since(start)),
			"func:", "CollectAncestorsAftCpByTips",
			"cache.len", hc.ancestorCache.Len(),
		)
	}(time.Now())

	ancHashes := common.HashArray{}
	tips = types.Tips{}
	ancestors = types.HeaderMap{}
	for _, parentHash := range parents {
		bdag := hc.GetBlockDag(parentHash)
		if bdag == nil {
			log.Warn("Collect ancestors by tips: block dag not found", "parent", parentHash.Hex())
			unloaded = append(unloaded, parentHash)
		} else {
			tips.Add(bdag)
		}
	}
	// check isCpAncestor
	for _, tip := range tips {
		if tip.CpHash == cpHash || tip.OrderedAncestorsHashes.Has(cpHash) {
			isCpAncestor = true
		}
		ancHashes = append(ancHashes, tip.OrderedAncestorsHashes...)
		ancHashes = append(ancHashes, tip.Hash)
		ancHashes.Deduplicate()
	}
	//collect ancestors
	cpHead := hc.GetHeader(cpHash)
	cpBlDag := hc.GetBlockDag(cpHash)
	if cpBlDag == nil {
		prevCpHead := hc.GetHeader(cpHead.CpHash)
		if prevCpHead == nil {
			log.Error("CollectAncestorsAftCpByTips failed: cp of cp not found")
			return isCpAncestor, ancestors, unloaded, tips
		}
		_, anc, _, err := hc.CollectAncestorsAftCpByParents(cpHead.ParentHashes, prevCpHead)
		if err != nil {
			log.Error("CollectAncestorsAftCpByTips get ancestors failed", "err", err)
			return isCpAncestor, ancestors, unloaded, tips
		}
		cpBlDag = &types.BlockDAG{
			Hash:                   cpHead.Hash(),
			Height:                 cpHead.Height,
			Slot:                   cpHead.Slot,
			CpHash:                 cpHead.CpHash,
			CpHeight:               hc.GetHeader(cpHead.CpHash).Height,
			OrderedAncestorsHashes: anc.Hashes(),
		}
		hc.SaveBlockDag(cpBlDag)
	}

	ancestors = hc.GetHeadersByHashes(ancHashes)
	for h, anc := range ancestors {
		if anc == nil {
			delete(ancestors, h)
			continue
		}
		// exclude finalized before cp
		if h == cpHash {
			delete(ancestors, h)
			continue
		}
		if anc.Height > 0 && anc.Nr() > 0 && cpBlDag.OrderedAncestorsHashes.Has(anc.Hash()) {
			delete(ancestors, h)
			continue
		}
		//if anc.Height > 0 && anc.Nr() > 0 && anc.Nr() <= cpHead.Nr() {
		//	delete(ancestors, h)
		//	continue
		//}
	}
	return isCpAncestor, ancestors, unloaded, tips
}

// CollectAncestorsAftCpByParents recursively collect ancestors by block parents
// which have to be finalized after checkpoint up to block.
func (hc *HeaderChain) CollectAncestorsAftCpByParents(parents common.HashArray, cpHeader *types.Header) (
	isCpAncestor bool,
	ancestors types.HeaderMap,
	unloaded common.HashArray,
	err error,
) {
	start := time.Now()

	ancestors = types.HeaderMap{}
	for _, h := range parents {
		var (
			isCpAnc bool
			anc     types.HeaderMap
			unl     common.HashArray
		)
		isCpAnc, anc, unl, err = hc.collectAncestorsAftCpByParents(h, cpHeader)
		if err != nil {
			log.Error("Collect ancestors by parents err", "err", err, "hash", h)
			return isCpAncestor, ancestors, unloaded, err
		}
		if isCpAnc {
			isCpAncestor = true
		}
		for ah, hdr := range anc {
			ancestors[ah] = hdr
		}
		if len(unl) > 0 {
			unloaded = append(unloaded, unl...).Uniq()
		}
	}

	log.Debug("TIME",
		"elapsed", common.PrettyDuration(time.Since(start)),
		"func:", "CollectAncestorsAftCpByParents",
		"cache.len", hc.ancestorCache.Len(),
	)
	return isCpAncestor, ancestors, unloaded, err
}

func (hc *HeaderChain) collectAncestorsAftCpByParents(headHash common.Hash, cpHeader *types.Header) (
	isCpAncestor bool,
	ancestors types.HeaderMap,
	unloaded common.HashArray,
	err error,
) {
	if ancestors == nil {
		ancestors = types.HeaderMap{}
	}
	if cpHeader.Height > 0 && cpHeader.Nr() == 0 {
		return false, ancestors, common.HashArray{}, ErrCpNotFinalized
	}
	headHeader := hc.GetHeader(headHash)
	// if headHeader is not found
	if headHeader == nil {
		return false, ancestors, common.HashArray{headHash}, nil
	}
	// if headHeader is checkpoint
	if headHeader.Hash() == cpHeader.Hash() {
		return true, ancestors, common.HashArray{}, nil
	}
	// if headHeader is finalized before checkpoint
	if nr := headHeader.Nr(); !(headHeader.Height > 0 && nr == 0) && nr < cpHeader.Nr() {
		return false, ancestors, common.HashArray{}, nil
	}

	if headHeader.ParentHashes == nil || len(headHeader.ParentHashes) == 0 {
		if headHeader.Hash() == hc.genesisHeader.Hash() {
			return false, ancestors, common.HashArray{}, nil
		}
		log.Warn("Detect headHeader without parents", "hash", headHeader.Hash().Hex(), "height", headHeader.Height, "slot", headHeader.Slot)
		err = fmt.Errorf("Detect headHeader without parents hash=%s, height=%d", headHeader.Hash().Hex(), headHeader.Height)
		return false, ancestors, common.HashArray{}, err
	}
	ancestors[headHash] = headHeader
	for _, ph := range headHeader.ParentHashes {
		var (
			_isCpAncestor bool
			_ancestors    types.HeaderMap
			_unloaded     common.HashArray
			_err          error
			ancCache      *CollectAncestorsResult
		)

		if cache, ok := hc.ancestorCache.Get(ph); ok {
			ancCache = cache.(*CollectAncestorsResult)
		}

		if ancCache != nil && ancCache.cpHash == cpHeader.Hash() {
			log.Debug("collectAncestorsAftCpByParents: find cached data", "hash", headHeader.Hash().Hex(), "cpHash", cpHeader.Hash())
			_isCpAncestor = ancCache.isCpAncestor
			_ancestors = ancCache.ancestors
			_unloaded = ancCache.unloaded
			_err = ancCache.err
		} else {
			log.Debug("collectAncestorsAftCpByParents: recursive call", "hash", headHeader.Hash().Hex(), "cpHash", cpHeader.Hash())
			_isCpAncestor, _ancestors, _unloaded, _err = hc.collectAncestorsAftCpByParents(ph, cpHeader)
			hc.ancestorCache.Add(ph, &CollectAncestorsResult{
				cpHash:       cpHeader.Hash(),
				isCpAncestor: _isCpAncestor,
				ancestors:    _ancestors,
				unloaded:     _unloaded,
				err:          _err,
			})
		}
		unloaded = unloaded.Concat(_unloaded).Uniq()
		for h, hd := range _ancestors {
			ancestors[h] = hd
		}
		if _isCpAncestor {
			isCpAncestor = true
		}
		err = _err
	}
	return isCpAncestor, ancestors, unloaded, err
}

// GetBlockDag retrieves a block dag from the database by hash, caching it if found.
func (hc *HeaderChain) GetBlockDag(hash common.Hash) *types.BlockDAG {
	//check in cache
	if v, ok := hc.blockDagCache.Get(hash); ok {
		return v.(*types.BlockDAG)
	}
	bdag := rawdb.ReadBlockDag(hc.chainDb, hash)
	if bdag == nil {
		return nil
	}
	// Cache the found bdag for next time and return
	hc.blockDagCache.Add(hash, bdag)
	return bdag
}

// SaveBlockDag save a block dag to the database by hash, and caching it.
func (hc *HeaderChain) SaveBlockDag(bdag *types.BlockDAG) {
	if bdag == nil {
		hc.DeleteBlockDag(bdag.Hash)
		return
	}
	rawdb.WriteBlockDag(hc.chainDb, bdag)
	// Cache the bdag for next time and return
	hc.blockDagCache.Add(bdag.Hash, bdag)
}

// DeleteBlockDag remove a block dag from the database and cache.
func (hc *HeaderChain) DeleteBlockDag(hash common.Hash) {
	rawdb.DeleteBlockDag(hc.chainDb, hash)
	hc.blockDagCache.Remove(hash)
}
