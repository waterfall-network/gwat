// Package finalizer implements chain finalization of DAG network:
// - ordering (defines order of blocks in finalized chain)
// - apply transactions
// - state propagation
package finalizer

import (
	"sort"
	"sync/atomic"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/state"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/types"
	"gitlab.waterfall.network/waterfall/protocol/gwat/eth/downloader"
	"gitlab.waterfall.network/waterfall/protocol/gwat/event"
	"gitlab.waterfall.network/waterfall/protocol/gwat/log"
	"gitlab.waterfall.network/waterfall/protocol/gwat/params"
)

const (
	//CoordDelaySlots number of slots of delay to retrieve candidates for coord network
	CoordDelaySlots uint64 = 2
	//CreateDagSlotsLimit limit of slots in dag chain to stop block creation
	CreateDagSlotsLimit = 8
)

// Backend wraps all methods required for finalizing.
type Backend interface {
	BlockChain() *core.BlockChain
	Downloader() *downloader.Downloader
}

// Finalizer
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
func (f *Finalizer) Finalize(spines *common.HashArray, baseSpine *common.Hash, isHeadSync bool) error {

	if f.isSyncing() && !isHeadSync {
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
			log.Error("Block finalization failed", "spineHash", spineHash.Hex(), "err", ErrSpineNotFound)
			return ErrSpineNotFound
		}
		spinesMap[block.Slot()] = block
	}

	//sort by slots
	slots := common.SorterAskU64{}
	for sl := range spinesMap {
		slots = append(slots, sl)
	}
	sort.Sort(slots)

	for _, slot := range slots {
		spine := spinesMap[slot]
		orderedChain := types.SpineGetDagChain(f.eth.BlockChain(), spine)

		if len(orderedChain) == 0 {
			log.Info("⌛ Finalization skip finalized spine:", "slot", spine.Slot(), "nr", spine.Nr(), "height", spine.Height(), "hash", spine.Hash().Hex())
			continue
		}
		log.Info("Finalization spine chain calculated", "isHeadSync", isHeadSync, "lfNr", lastFinNr, "slot", spine.Slot(), "height", spine.Height(), "hash", spine.Hash().Hex(), "chain", orderedChain.GetHashes())

		if isHeadSync {
			//validate blocks while head sync
			for _, block := range orderedChain {
				if ok, err := bc.VerifyBlock(block); !ok {
					if err == nil {
						bc.CacheInvalidBlock(block)
						err = ErrInvalidBlock
					}
					log.Error("Block finalization failed (validation)", "valid", ok, "slot", block.Slot(), "height", block.Height(), "hash", block.Hash().Hex(), "err", err)
					return err
				}
			}
		}

		// blocks finalizing
		for i, block := range orderedChain {
			nr := lastFinNr + uint64(i) + 1
			block.SetNumber(&nr)
			isHead := i == len(orderedChain)-1
			if err := f.finalizeBlock(nr, *block, isHead); err != nil {
				log.Error("Block finalization failed", "isHead", isHead, "calc.nr", nr, "b.nr", block.Nr(), "slot", block.Slot(), "height", block.Height(), "hash", block.Hash().Hex(), "err", err)
				return err
			}
		}
		lastBlock := bc.GetBlock(orderedChain[len(orderedChain)-1].Hash())
		log.Info("⛓ Finalization of spine completed", "blocks", len(orderedChain), "slot", lastBlock.Slot(), "calc.nr", lastFinNr, "nr", lastBlock.Nr(), "height", lastBlock.Height(), "hash", lastBlock.Hash().Hex())
		lastFinNr = lastBlock.Nr()
		//todo add logs to catch Reset Nr to update tips
		f.updateTips(*orderedChain.GetHashes(), *lastBlock)
		log.Info("⛓ Finalization of spine completed (updateTips)", "blocks", len(orderedChain), "slot", lastBlock.Slot(), "calc.nr", lastFinNr, "nr", lastBlock.Nr(), "height", lastBlock.Height(), "hash", lastBlock.Hash().Hex())
		if lastBlock.Height() != lastBlock.Nr() {
			log.Error("☠ finalizing: mismatch nr and height", "slot", lastBlock.Slot(), "nr", lastBlock.Nr(), "height", lastBlock.Height(), "hash", lastBlock.Hash().Hex())
		}
	}
	return nil
}

// updateTips update tips in accordance of finalized blocks.
func (f *Finalizer) updateTips(finHashes common.HashArray, lastBlock types.Block) {
	bc := f.eth.BlockChain()

	//todo reset nr log
	if lastBlock.Nr() == 0 {
		log.Error("☠☠☠ RESET NR DETECTED:updateTips 000 ☠☠☠", "slot", lastBlock.Slot(), "nr", lastBlock.Nr(), "height", lastBlock.Height(), "hash", lastBlock.Hash().Hex())
	}

	//todo reset nr rollback
	//bc.FinalizeTips(finHashes, lastBlock.Hash(), lastBlock.Height())
	bc.FinalizeTips(finHashes, lastBlock.Hash(), lastBlock.Height(), lastBlock)

	//todo reset nr log
	if lastBlock.Nr() == 0 {
		log.Error("☠☠☠ RESET NR DETECTED:updateTips 111 ☠☠☠", "slot", lastBlock.Slot(), "nr", lastBlock.Nr(), "height", lastBlock.Height(), "hash", lastBlock.Hash().Hex())
	}

	//remove stale blockDags
	for _, h := range finHashes {
		bc.DeleteBlockDag(h)
	}

	//todo reset nr log
	if lastBlock.Nr() == 0 {
		log.Error("☠☠☠ RESET NR DETECTED:updateTips 111 ☠☠☠", "slot", lastBlock.Slot(), "nr", lastBlock.Nr(), "height", lastBlock.Height(), "hash", lastBlock.Hash().Hex())
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
	usedHash := bc.ReadFinalizedHashByNumber(finNr)
	if usedHash != (common.Hash{}) {
		log.Warn("Fin Nr is already used", "finNr", finNr, "usedHash", usedHash.Hex(), "height", block.Height(), "hash", block.Hash().Hex())
		return ErrFinNrrUsed
	}

	if err := bc.WriteFinalizedBlock(finNr, &block, []*types.Receipt{}, []*types.Log{}, &state.StateDB{}, isHead); err != nil {
		return err
	}

	log.Info("🔗 block finalized", "Number", finNr, "b.nr", block.Nr(), "Slot", block.Slot(), "Height", block.Height(), "hash", block.Hash().Hex())
	return nil
}

// isSyncing returns true if sync process is running.
func (f *Finalizer) isSyncing() bool {
	return f.eth.Downloader().Synchronising()
}

// GetFinalizingCandidates returns the ordered dag block hashes for finalization
func (f *Finalizer) GetFinalizingCandidates(lteSlot *uint64) (common.HashArray, error) {
	bc := f.eth.BlockChain()
	tips := bc.GetTips()

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

// ValidateSequenceOfSpines check is sequence of spines is valid.
func (f *Finalizer) IsValidSequenceOfSpines(spines common.HashArray) (bool, error) {
	if len(spines) == 0 {
		return true, nil
	}
	var (
		bc            = f.eth.BlockChain()
		hasNotFin     = false          //has any not finalized items
		dagCandidates common.HashArray // current dag candidates
		err           error
	)

	mapHeaders := bc.GetHeadersByHashes(spines)
	for _, b := range mapHeaders {
		// if not found
		if b == nil {
			return false, nil
		}
		// if block is not finalized
		if b.Nr() == 0 && b.Height > 0 {
			hasNotFin = true
		} else {
			// if block is finalized - check is it spine
			if b.Nr() != b.Height {
				return false, nil
			}
		}
	}

	if hasNotFin {
		dagCandidates, err = f.GetFinalizingCandidates(nil)
		if err != nil {
			return false, err
		}
	}

	// check order by slots and gap in sequence
	var prevBlock *types.Header
	for i, sp := range spines {
		bl := mapHeaders[sp]
		if i == 0 {
			prevBlock = bl
			continue
		}
		// check that slot must be greater than previous
		if bl.Slot <= prevBlock.Slot {
			return false, nil
		}
		// if prev block is not finalized
		if prevBlock.Nr() == 0 && prevBlock.Height > 0 {
			// check in current candidates
			checkPart := common.HashArray{
				prevBlock.Hash(),
				bl.Hash(),
			}
			seqInter := dagCandidates.SequenceIntersection(checkPart)
			if !checkPart.IsEqualTo(seqInter) {
				return false, nil
			}
		} else {
			// if prev block is finalized - find following spine
			var followingSpine common.Hash
			for fnr := prevBlock.Nr(); ; {
				fnr++
				nxt := bc.GetHeaderByNumber(fnr)
				if nxt == nil {
					spBlock := mapHeaders[sp]
					// if spine block is not finalized - check first dag candidate
					if spBlock.Nr() == 0 && spBlock.Height > 0 && len(dagCandidates) > 0 {
						followingSpine = dagCandidates[0]
					}
					break
				}
				if nxt.Nr() == nxt.Height {
					followingSpine = nxt.Hash()
					break
				}
			}
			if sp != followingSpine {
				return false, nil
			}
		}
		prevBlock = bl
	}
	return true, nil
}
