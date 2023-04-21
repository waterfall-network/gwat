// Package headsync implements head synchronising of DAG with coordinating network:

package headsync

import (
	"sync/atomic"
	"time"

	"gitlab.waterfall.network/waterfall/protocol/gwat/core"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/types"
	"gitlab.waterfall.network/waterfall/protocol/gwat/dag/finalizer"
	"gitlab.waterfall.network/waterfall/protocol/gwat/eth/downloader"
	"gitlab.waterfall.network/waterfall/protocol/gwat/event"
	"gitlab.waterfall.network/waterfall/protocol/gwat/log"
	"gitlab.waterfall.network/waterfall/protocol/gwat/params"
)

// Backend wraps all methods required for finalizing.
type Backend interface {
	BlockChain() *core.BlockChain
	Downloader() *downloader.Downloader
}

const HeadSyncTimeoutSec = 120

type Headsync struct {
	chainConfig  *params.ChainConfig
	mux          *event.TypeMux
	eth          Backend
	finalizer    *finalizer.Finalizer
	ready        int32             // The indicator whether the headsync is ready to synck (SetReadyState method has been called).
	mainProc     int32             // The indicator whether the headsync is finalizing blocks.
	lastSyncData *types.Checkpoint // last checkpoint
	timeout      *time.Timer       // reset head sync timer
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
	atomic.StoreInt32(&f.mainProc, 0)

	return f
}

// SetReadyState  set initial state to start head sync with coordinating network.
func (hs *Headsync) SetReadyState(checkpoint *types.Checkpoint) (bool, error) {
	if checkpoint == nil {
		log.Warn("☠ Prepare to head synchronising is skipped (checkpoint is nil)", "checkpoint", checkpoint)
		return false, ErrBadParams
	}
	// TODO: check
	//if len(checkpoint.Spine) == 0 && checkpoint.Epoch > 0 {
	//	log.Warn("☠ Prepare to head synchronising is skipped (spines empty)", "checkpoint", checkpoint)
	//	return false, ErrBadParams
	//}

	//skip if other synchronising type is running
	if hs.eth.Downloader().FinSynchronising() || hs.eth.Downloader().DagSynchronising() {
		log.Warn("⌛ Prepare to head synchronising is skipped (process is locked)")
		return false, ErrLocked
	}
	//skip if head synchronising is running
	if err := hs.HeadSyncSet(); err != nil {
		log.Warn("⌛ Prepare to head synchronising is skipped (process is running)")
		return false, err
	}
	// reset ready state
	atomic.StoreInt32(&hs.ready, 0)

	// if genesis checkpoint -
	if checkpoint.Epoch == 0 {
		log.Info("Prepare to head synchronising set to genesis", "checkpoint", checkpoint)
		atomic.StoreInt32(&hs.ready, 1)
		hs.lastSyncData = checkpoint
		return true, nil
	}

	if ok, err := hs.validateCheckpoint(checkpoint); !ok {
		log.Warn("☠ Prepare to head synchronising is skipped (bad checkpoint)", "err", err, "checkpoint", checkpoint)
		hs.headSyncReset()
		return false, err
	}

	//reorg finalized and dag chains in accordance with checkpoint
	bc := hs.eth.BlockChain()
	cpSpineHash := checkpoint.Spine
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
			LastFinalizedHash:   block.CpHash(),
			LastFinalizedHeight: block.CpNumber(),
			DagChainHashes:      block.ParentHashes(),
		})
		err := bc.RollbackFinalization(i)
		if err != nil {
			log.Error("Prepare to head synchronising error (rollback)", "finNr", i, "hash", block.Hash().Hex(), "err", err)
		}
	}
	// update head of finalized chain
	if err := bc.WriteFinalizedBlock(cpBlock.Nr(), cpBlock, nil, nil, nil, true); err != nil {
		hs.headSyncReset()
		return false, err
	}
	// update BlockDags
	expCache := core.ExploreResultMap{}
	for _, bdag := range blockDagList {
		_, loaded, _, _, exc, _ := bc.ExploreChainRecursive(bdag.Hash, expCache)
		expCache = exc
		bdag.DagChainHashes = loaded
		//if dch := graph.GetDagChainHashes(); dch != nil {
		//	bdag.DagChainHashes = *dch
		//}
		bc.WriteBlockDag(&bdag)
	}
	// update tips
	tips := bc.GetTips()
	for _, tip := range tips {
		_, loaded, _, _, exc, _ := bc.ExploreChainRecursive(tip.Hash, expCache)
		expCache = exc
		tip.DagChainHashes = loaded
		//if dch := graph.GetDagChainHashes(); dch != nil {
		//	tip.DagChainHashes = *dch
		//}
		bc.AddTips(tip)
	}
	bc.WriteCurrentTips()

	// TODO: rm deprecated
	// update LastCoordinatedHash to checkpoint
	//bc.WriteLastCoordinatedHash(cpBlock.Hash())

	// set ready state
	atomic.StoreInt32(&hs.ready, 1)
	hs.lastSyncData = checkpoint
	return true, nil
}

// validateCheckpoint checkpoint validation
func (hs *Headsync) validateCheckpoint(checkpoint *types.Checkpoint) (bool, error) {
	bc := hs.eth.BlockChain()
	cpSpineHash := checkpoint.Spine
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
	if block.Height() > block.Nr() {
		return false, ErrCheckpointBadNr
	}
	// state exists
	state, err := bc.StateAt(block.Root())
	if err != nil {
		return false, err
	}
	if state == nil {
		return false, ErrCheckpointNoState
	}
	return true, nil
}

// HeadSyncSet set status of head sync.
func (hs *Headsync) HeadSyncSet() error {
	//skip if head synchronising is running
	if !hs.eth.Downloader().HeadSyncSet() {
		return ErrLocked
	}

	// set timeout of head sync procedure
	tot := HeadSyncTimeoutSec * time.Second
	if hs.timeout == nil {
		hs.timeout = time.NewTimer(tot)
	} else {
		hs.timeout.Reset(tot)
	}

	go func() {
		select {
		case <-hs.timeout.C:
			log.Warn("Head sync: timeout reset")
			hs.headSyncReset()
			return
		}
	}()
	return nil
}

// headSyncReset reset status of head sync.
func (hs *Headsync) headSyncReset() bool {
	// skip if main process is running
	if atomic.LoadInt32(&hs.mainProc) == 1 {
		log.Warn("⌛ Head synchronising is skipped (is running)")
		return false
	}

	log.Warn("Head sync: reset")
	if hs.timeout != nil {
		hs.timeout.Stop()
	}
	hs.eth.Downloader().HeadSyncReset()
	// set ready state
	atomic.StoreInt32(&hs.ready, 0)
	return true
}
