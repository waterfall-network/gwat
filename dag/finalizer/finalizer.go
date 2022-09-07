// Package finalizer implements chain finalization of DAG network:
// - ordering (defines order of blocks in finalized chain)
// - apply transactions
// - state propagation
package finalizer

import (
	"sort"
	"sync/atomic"

	"github.com/waterfall-foundation/gwat/common"
	"github.com/waterfall-foundation/gwat/core"
	"github.com/waterfall-foundation/gwat/core/state"
	"github.com/waterfall-foundation/gwat/core/types"
	"github.com/waterfall-foundation/gwat/dag/creator"
	"github.com/waterfall-foundation/gwat/eth/downloader"
	"github.com/waterfall-foundation/gwat/event"
	"github.com/waterfall-foundation/gwat/log"
	"github.com/waterfall-foundation/gwat/params"
)

const (
	FinalisationDelaySlots = 3
)

// Backend wraps all methods required for finalizing.
type Backend interface {
	BlockChain() *core.BlockChain
	Downloader() *downloader.Downloader
	DagCreator() *creator.Creator
}

// Finalizer creates blocks and searches for proof-of-work values.
type Finalizer struct {
	chainConfig *params.ChainConfig

	// events
	mux *event.TypeMux

	eth     Backend
	running int32 // The indicator whether the finalizer is running or not.
	busy    int32 // The indicator whether the finalizer is finalizing blocks.
}

// New create new instance of Finalizer
func New(chainConfig *params.ChainConfig, eth Backend, mux *event.TypeMux) *Finalizer {
	f := &Finalizer{
		chainConfig: chainConfig,
		eth:         eth,
		mux:         mux,
	}
	atomic.StoreInt32(&f.running, 0)
	atomic.StoreInt32(&f.busy, 0)

	return f
}

// Finalize start finalization procedure
func (f *Finalizer) Finalize(spines *common.HashArray) error {

	if f.isSyncing() {
		return ErrSyncing
	}
	if atomic.LoadInt32(&f.busy) == 1 {
		log.Info("⌛ Finalization is skipped: process busy")
		return ErrBusy
	}

	atomic.StoreInt32(&f.busy, 1)
	defer atomic.StoreInt32(&f.busy, 0)

	if len(*spines) == 0 {
		log.Info("⌛ Finalization is skipped: received spines empty")
		return nil
	}

	bc := f.eth.BlockChain()
	lastFinBlock := bc.GetLastFinalizedBlock()
	lastFinNr := lastFinBlock.Nr()

	//collect and check finalizing blocks
	spinesMap := make(types.SlotSpineMap, len(*spines))
	for _, spineHash := range *spines {
		block := bc.GetBlockByHash(spineHash)
		if block == nil {
			log.Error("unknown spine hash", "spineHash", spineHash.Hex())
			return ErrUnknownHash
		}
		spinesMap[block.Slot()] = block
	}

	//sort by slots
	slots := common.SorterAskU64{}
	for sl, _ := range spinesMap {
		slots = append(slots, sl)
	}
	sort.Sort(slots)

	var headBlock *types.Block
	for _, slot := range slots {
		spine := spinesMap[slot]
		orderedChain := types.SpineGetDagChain(f.eth.BlockChain(), spine)
		orderedChain, unassigned := f.eth.DagCreator().GetChainWithoutUnassignedCreators(orderedChain)

		for _, unassignedBlock := range unassigned {
			log.Warn("remove block with unassigned creator", "creator", unassignedBlock.Header().Coinbase.Hex(), "blockHash", unassignedBlock.Hash().Hex())
			bc.DeleteBlockDag(unassignedBlock.Hash())
		}

		if len(orderedChain) == 0 {
			log.Info("⌛ Finalization skip finalized spine:", "slot", spine.Slot(), "nr", spine.Nr(), "height", spine.Height(), "hash", spine.Hash().Hex())
			continue
		}
		log.Info("Finalization spine chain calculated", "slot", spine.Slot(), "nr", spine.Nr(), "height", spine.Height(), "hash", spine.Hash().Hex(), "chain", orderedChain.GetHashes())

		// blocks finalizing
		for i, block := range orderedChain {
			nr := lastFinNr + uint64(i) + 1
			block.SetNumber(&nr)
			isHead := i == len(orderedChain)-1
			if err := f.finalizeBlock(nr, *block, isHead); err != nil {
				log.Error("block finalization failed", "nr", i, "slot", block.Slot(), "height", block.Height(), "hash", block.Hash().Hex(), "err", err)
				return err
			}
		}
		lastBlock := orderedChain[len(orderedChain)-1]
		f.updateTips(*orderedChain.GetHashes(), *lastBlock)
		lastFinNr = lastBlock.Nr()
		log.Info("⛓ Finalization of spine completed", "blocks", len(orderedChain), "slot", lastBlock.Slot(), "nr", lastBlock.Nr(), "height", lastBlock.Height(), "hash", lastBlock.Hash().Hex())
		headBlock = lastBlock

		if headBlock.Height() != headBlock.Nr() {
			log.Error("☠ finalizing: mismatch nr and height", "slot", headBlock.Slot(), "nr", headBlock.Nr(), "height", headBlock.Height(), "hash", headBlock.Hash().Hex())
		}
	}
	return nil
}

// updateTips update tips in accordance of finalized blocks.
func (f *Finalizer) updateTips(finHashes common.HashArray, lastBlock types.Block) {
	bc := f.eth.BlockChain()
	bc.FinalizeTips(finHashes, lastBlock.Hash(), lastBlock.Height())
	//remove stale blockDags
	for _, h := range finHashes {
		bc.DeleteBlockDag(h)
	}
}

// finalizeBlock finalize block
func (f *Finalizer) finalizeBlock(finNr uint64, block types.Block, isHead bool) error {
	bc := f.eth.BlockChain()
	nr := bc.ReadFinalizedNumberByHash(block.Hash())
	if nr != nil && *nr == finNr {
		log.Warn("Block already finalized", "finNr", "nr", nr, "height", block.Height(), "hash", block.Hash().Hex())
		return nil
	}
	if err := bc.WriteFinalizedBlock(finNr, &block, []*types.Receipt{}, []*types.Log{}, &state.StateDB{}, isHead); err != nil {
		return err
	}

	log.Info("🔗 block finalized", "Number", finNr, "Slot", block.Slot(), "Height", block.Height(), "hash", block.Hash().Hex())
	return nil
}

// isSyncing returns true if sync process is running.
func (f *Finalizer) isSyncing() bool {
	return f.eth.Downloader().Synchronising()
}

// GetFinalizingCandidates returns the ordered dag block hashes for finalization
func (f *Finalizer) GetFinalizingCandidates(lteSlot *uint64) (common.HashArray, error) {
	bc := f.eth.BlockChain()
	tips, unloaded := bc.ReviseTips()
	if len(unloaded) > 0 || tips == nil || len(*tips) == 0 {
		if tips == nil {
			log.Warn("Get finalized candidates received bad tips", "unloaded", unloaded, "tips", tips)
		} else {
			log.Warn("Get finalized candidates received bad tips", "unloaded", unloaded, "tips", tips.Print())
		}
		return common.HashArray{}, ErrBadDag
	}
	finChain := []*types.Block{}
	for _, block := range bc.GetBlocksByHashes(tips.GetOrderedDagChainHashes().Uniq()).ToArray() {
		if block.Height() > 0 && block.Nr() == 0 {
			finChain = append(finChain, block)
		}
	}

	if len(finChain) == 0 {
		return common.HashArray{}, nil
	}

	lastFinSlot := bc.GetLastFinalizedBlock().Slot()
	spines, err := types.CalculateSpines(finChain, lastFinSlot)
	if err != nil {
		return common.HashArray{}, err
	}

	if lteSlot != nil {
		spinesOfSlot := types.SlotSpineMap{}
		for slot, spine := range spines {
			if *lteSlot >= slot {
				spinesOfSlot[slot] = spine
			}
		}
		spineHashes := spinesOfSlot.GetOrderedHashes()
		return *spineHashes, nil
	}
	spineHashes := spines.GetOrderedHashes()
	return *spineHashes, nil
}
