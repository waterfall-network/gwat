// Package headsync implements head synchronising of DAG with coordinating network:

package headsync

import (
	"github.com/waterfall-foundation/gwat/common"
	"github.com/waterfall-foundation/gwat/dag/finalizer"
	"sort"
	"sync/atomic"

	"github.com/waterfall-foundation/gwat/core"
	"github.com/waterfall-foundation/gwat/core/types"
	"github.com/waterfall-foundation/gwat/eth/downloader"
	"github.com/waterfall-foundation/gwat/event"
	"github.com/waterfall-foundation/gwat/log"
	"github.com/waterfall-foundation/gwat/params"
)

// Backend wraps all methods required for finalizing.
type Backend interface {
	BlockChain() *core.BlockChain
	Downloader() *downloader.Downloader
}

// Headsync creates blocks and searches for proof-of-work values.
type Headsync struct {
	chainConfig  *params.ChainConfig
	mux          *event.TypeMux
	eth          Backend
	finalizer    *finalizer.Finalizer
	ready        int32                // The indicator whether the headsync is ready to synck (SetReadyState method has been called).
	busy         int32                // The indicator whether the headsync is finalizing blocks.
	lastSyncData *types.ConsensusInfo // last applied sync data
}

// New create new instance of Headsync
func New(chainConfig *params.ChainConfig, eth Backend, mux *event.TypeMux, finalizer *finalizer.Finalizer) *Headsync {
	f := &Headsync{
		chainConfig:  chainConfig,
		eth:          eth,
		finalizer:    finalizer,
		mux:          mux,
		lastSyncData: nil,
	}
	atomic.StoreInt32(&f.ready, 0)
	atomic.StoreInt32(&f.busy, 0)

	return f
}

// SetReadyState  set initial state to start head sync with coordinating network.
func (hs *Headsync) SetReadyState(checkpoint *types.ConsensusInfo) (bool, error) {
	//skip if head synchronising is not active
	if !hs.eth.Downloader().HeadSynchronising() {
		log.Warn("⌛ Prepare to head synchronising is skipped (is not active)")
		return false, nil
	}

	if atomic.LoadInt32(&hs.busy) == 1 {
		log.Warn("⌛ Prepare to head synchronising is skipped (process busy)")
		return false, ErrBusy
	}
	atomic.StoreInt32(&hs.busy, 1)
	defer atomic.StoreInt32(&hs.busy, 0)
	// reset ready state
	atomic.StoreInt32(&hs.ready, 0)

	if checkpoint == nil {
		log.Warn("☠ Prepare to head synchronising is skipped (checkpoint is nil)")
		return false, ErrBadParams
	}
	if len(checkpoint.Finalizing) == 0 {
		log.Warn("☠ Prepare to head synchronising is skipped (spines empty)")
		return false, ErrBadParams
	}

	if ok, err := hs.validateCheckpoint(checkpoint); !ok {
		log.Warn("☠ Prepare to head synchronising is skipped (bad checkpoint)", "err", err)
	}

	//reorg finalized and dag chains in accordance with checkpoint
	bc := hs.eth.BlockChain()
	cpSpineHash := checkpoint.Finalizing[len(checkpoint.Finalizing)]
	cpBlock := bc.GetBlock(cpSpineHash)

	lfNr := bc.GetLastFinalizedNumber()
	blockDagList := []types.BlockDAG{}
	for i := lfNr; i > cpBlock.Nr(); i-- {
		block := bc.GetBlockByNumber(i)
		if block == nil {
			log.Warn("Prepare to head synchronising (rollback block not found)", "finNr", i)
			continue
		}
		blockDagList = append(blockDagList, types.BlockDAG{
			Hash:                block.Hash(),
			Height:              block.Height(),
			Slot:                block.Slot(),
			LastFinalizedHash:   block.LFHash(),
			LastFinalizedHeight: block.LFNumber(),
			DagChainHashes:      block.ParentHashes(),
		})
		err := bc.RollbackFinalization(i)
		if err != nil {
			log.Error("Prepare to head synchronising error (rollback)", "finNr", i, "hash", block.Hash().Hex(), "err", err)
		}
	}
	// update head of finalized chain
	if err := bc.WriteFinalizedBlock(cpBlock.Nr(), cpBlock, nil, nil, nil, true); err != nil {
		return false, err
	}
	// update BlockDags
	expCache := core.ExploreResultMap{}
	for _, bdag := range blockDagList {
		_, _, _, graph, exc, _ := bc.ExploreChainRecursive(bdag.Hash, expCache)
		expCache = exc
		if dch := graph.GetDagChainHashes(); dch != nil {
			bdag.DagChainHashes = *dch
		}
		bc.WriteBlockDag(&bdag)
	}
	// update tips
	tips := bc.GetTips()
	for _, tip := range tips {
		_, _, _, graph, exc, _ := bc.ExploreChainRecursive(tip.Hash, expCache)
		expCache = exc
		if dch := graph.GetDagChainHashes(); dch != nil {
			tip.DagChainHashes = *dch
		}
		bc.AddTips(tip)

	}
	bc.WriteCurrentTips()
	// save creators
	hs.eth.BlockChain().WriteCreators(checkpoint.Slot, checkpoint.Creators)
	// set ready state
	atomic.StoreInt32(&hs.ready, 1)
	hs.lastSyncData = checkpoint
	return true, nil
}

// Sync run head sync with coordinating network.
func (hs *Headsync) Sync(data []types.ConsensusInfo) (bool, error) {
	//skip if head synchronising is not active
	if !hs.eth.Downloader().HeadSynchronising() {
		log.Warn("⌛ Head synchronising is skipped (is not active)")
		return false, nil
	}

	if atomic.LoadInt32(&hs.busy) == 1 {
		log.Warn("⌛ Head synchronising is skipped (process busy)")
		return false, ErrBusy
	}
	atomic.StoreInt32(&hs.busy, 1)
	defer atomic.StoreInt32(&hs.busy, 0)

	//check ready state
	if atomic.LoadInt32(&hs.ready) == 1 {
		log.Info("⌛ Head synchronising is skipped (not ready)")
		return false, ErrNotReady
	}
	defer atomic.StoreInt32(&hs.ready, 0)

	//sort data by slots
	dataBySlots := map[uint64]*types.ConsensusInfo{}
	slots := common.SorterAskU64{}
	for _, d := range data {
		sl := d.Slot
		if dataBySlots[sl] != nil {
			log.Warn("Head synchronising has received duplicated data")
		}
		dataBySlots[sl] = &d
		slots = append(slots, sl)
	}
	sort.Sort(slots)
	// apply data
	for _, slot := range slots {
		d := dataBySlots[slot]
		// save creators
		hs.eth.BlockChain().WriteCreators(d.Slot, d.Creators)
		// finalize spines
		err := hs.finalizer.Finalize(&d.Finalizing, true)
		if err != nil {
			log.Warn("☠ Head synchronising failed", "err", err)
			return false, err
		}
	}
	return true, nil
}

// validateCheckpoint checkpoint validation
func (hs *Headsync) validateCheckpoint(checkpoint *types.ConsensusInfo) (bool, error) {
	bc := hs.eth.BlockChain()
	cpSpineHash := checkpoint.Finalizing[len(checkpoint.Finalizing)]
	block := bc.GetBlock(cpSpineHash)
	// block exists
	if block == nil {
		log.Info("☠ Handling prepare to head synchronising is skipped (checkpoint not found)")
		return false, ErrUnknownHash
	}
	// block is finalized
	if block.Height() > 0 && block.Nr() == 0 {
		return false, ErrCheckpointNotFin
	}
	// accordance height to number
	if block.Height() == block.Nr() {
		return false, ErrCheckpointBadNr
	}
	// state exists
	state, err := bc.StateAt(block.Root())
	if err == nil {
		return false, err
	}
	if state == nil {
		return false, ErrCheckpointNoState
	}
	return true, nil
}
