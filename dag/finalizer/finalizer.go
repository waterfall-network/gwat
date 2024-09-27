// Copyright 2024   Blue Wave Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package finalizer implements chain finalization of DAG network:
// - ordering (defines order of blocks in finalized chain)
// - apply transactions
// - state propagation
package finalizer

import (
	"context"
	"fmt"
	"sort"
	"sync/atomic"
	"time"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/types"
	"gitlab.waterfall.network/waterfall/protocol/gwat/eth/downloader"
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

type BlockChain interface {
	Config() *params.ChainConfig
	GetLastFinalizedBlock() *types.Block
	GetBlockByHash(hash common.Hash) *types.Block
	GetBlocksByHashes(hashes common.HashArray) types.BlockMap
	UpdateFinalizingState(block *types.Block, stateBlock *types.Block) error
	GetBlock(ctx context.Context, hash common.Hash) *types.Block
	FinalizeTips(finHashes common.HashArray, lastFinHash common.Hash, lastFinNr uint64)
	ReadFinalizedNumberByHash(hash common.Hash) *uint64
	ReadFinalizedHashByNumber(number uint64) common.Hash
	GetBlockFinalizedNumber(hash common.Hash) *uint64
	WriteFinalizedBlock(finNr uint64, block *types.Block, isHead bool) error
	GetTips() types.Tips
	GetHeadersByHashes(hashes common.HashArray) types.HeaderMap
	GetOptimisticSpines(gtSlot uint64) ([]common.HashArray, error)
	GetLastFinalizedHeader() *types.Header
	GetHeaderByNumber(number uint64) *types.Header
	SetRollbackActive()
	ResetRollbackActive()
	GetBlockDag(hash common.Hash) *types.BlockDAG
	CollectAncestorsAftCpByTips(parents common.HashArray, cpHash common.Hash) (isCpAncestor bool, ancestors types.HeaderMap, unloaded common.HashArray, tips types.Tips)
	GetHeader(hash common.Hash) *types.Header
	SaveBlockDag(blockDag *types.BlockDAG)
	RollbackFinalization(spineHash common.Hash, lfNr uint64) error
	GetLastFinalizedNumber() uint64
}

// Finalizer
type Finalizer struct {
	bc   BlockChain
	eth  Backend
	busy int32 // The indicator whether the finalizer is finalizing blocks.
}

// New create new instance of Finalizer
func New(eth Backend) *Finalizer {
	f := &Finalizer{
		eth: eth,
		bc:  eth.BlockChain(),
	}
	atomic.StoreInt32(&f.busy, 0)

	return f
}

// Finalize start finalization procedure
func (f *Finalizer) Finalize(spines *common.HashArray, baseSpine *common.Hash) error {
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

	ctx := context.Background()
	lastFinBlock := f.bc.GetLastFinalizedBlock()

	if err := f.SetSpineState(baseSpine, lastFinBlock.Nr()); err != nil {
		return err
	}

	lastFinBlock = f.bc.GetLastFinalizedBlock()
	lastFinNr := lastFinBlock.Nr()
	successSpine := lastFinBlock.Hash()

	//collect and check finalizing blocks
	spinesMap := make(types.SlotSpineMap, len(*spines))
	for _, spineHash := range *spines {
		block := f.bc.GetBlockByHash(spineHash)
		if block == nil {
			log.Error("Finalization failed (spine not found)", "spineHash", spineHash.Hex(), "err", ErrSpineNotFound)
			return ErrSpineNotFound
		}
		spinesMap[block.Slot()] = block
	}

	//sort by slots
	slots := common.SorterAscU64{}
	for sl := range spinesMap {
		slots = append(slots, sl)
	}
	sort.Sort(slots)

	for _, slot := range slots {
		spine := spinesMap[slot]
		orderedChain, err := types.SpineGetDagChain(f.bc, spine)
		if err != nil {
			log.Error("Finalization failed (calc fin seq)", "err", err)
			return err
		}

		log.Info("Finalization spine chain calculated", "isSync", f.isSyncing(), "lfNr", lastFinNr, "slot", spine.Slot(), "height", spine.Height(), "hash", spine.Hash().Hex(), "chain", orderedChain.GetHashes())

		if len(orderedChain) == 0 {
			log.Warn("⌛ Finalization skip finalized spine: (must never happened)", "slot", spine.Slot(), "nr", spine.Nr(), "height", spine.Height(), "hash", spine.Hash().Hex())
			continue
		}

		// blocks finalizing
		for i, block := range orderedChain {
			nr := lastFinNr + uint64(i) + 1
			block.SetNumber(&nr)
			err = f.bc.UpdateFinalizingState(block, lastFinBlock)
			if err != nil {
				log.Error("Finalization failed (state transition)", "err", err)
				return err
			}
			isHead := i == len(orderedChain)-1
			if err := f.finalizeBlock(nr, *block, isHead); err != nil {
				log.Error("Block finalization failed", "isHead", isHead, "calc.nr", nr, "b.nr", block.Nr(), "slot", block.Slot(), "height", block.Height(), "hash", block.Hash().Hex(), "err", err)
				if err := f.SetSpineState(&successSpine, nr); err != nil {
					return err
				}
				return err
			}
			lastFinBlock = block
		}
		lastBlock := f.bc.GetBlock(ctx, orderedChain[len(orderedChain)-1].Hash())
		log.Info("⛓ Finalization of spine completed", "blocks", len(orderedChain), "slot", lastBlock.Slot(), "calc.nr", lastFinNr, "nr", lastBlock.Nr(), "height", lastBlock.Height(), "hash", lastBlock.Hash().Hex())

		lastFinNr = lastBlock.Nr()
		f.updateTips(*orderedChain.GetHashes(), *lastBlock)
		log.Info("⛓ Finalization of spine completed (updateTips)", "blocks", len(orderedChain), "slot", lastBlock.Slot(), "calc.nr", lastFinNr, "nr", lastBlock.Nr(), "height", lastBlock.Height(), "hash", lastBlock.Hash().Hex())

		successSpine = spine.Hash()
	}
	return nil
}

// updateTips update tips in accordance of finalized blocks.
func (f *Finalizer) updateTips(finHashes common.HashArray, lastBlock types.Block) {
	f.bc.FinalizeTips(finHashes, lastBlock.Hash(), lastBlock.Height())
}

// finalizeBlock finalize block
func (f *Finalizer) finalizeBlock(finNr uint64, block types.Block, isHead bool) error {
	nr := f.bc.ReadFinalizedNumberByHash(block.Hash())
	if nr != nil && *nr == finNr {
		log.Warn("Block already finalized", "finNr", "nr", nr, "height", block.Height(), "hash", block.Hash().Hex())
		return nil
	}
	usedHash := f.bc.ReadFinalizedHashByNumber(finNr)
	if usedHash != (common.Hash{}) {
		log.Warn("Fin Nr is already used", "finNr", finNr, "usedHash", usedHash.Hex(), "height", block.Height(), "hash", block.Hash().Hex())
		return ErrFinNrrUsed
	}

	if err := f.bc.WriteFinalizedBlock(finNr, &block, isHead); err != nil {
		return err
	}

	log.Info("🔗 block finalized",
		"Number", finNr,
		"b.nr", block.Nr(),
		"Slot", block.Slot(),
		"Height", block.Height(),
		"txs", len(block.Transactions()),
		"hash", block.Hash().Hex(),
	)
	return nil
}

// isSyncing returns true if sync process is running.
func (f *Finalizer) isSyncing() bool {
	return f.eth.Downloader().Synchronising()
}

// GetFinalizingCandidates returns the ordered dag block hashes for finalization
func (f *Finalizer) GetFinalizingCandidates(lteSlot *uint64) (common.HashArray, error) {
	tips := f.bc.GetTips()

	finChain := make(types.Blocks, 0)
	for _, block := range f.bc.GetBlocksByHashes(tips.GetOrderedAncestorsHashes().Uniq()).ToArray() {
		if block.Height() > 0 && block.Nr() == 0 {
			finChain = append(finChain, block)
		}
	}

	if len(finChain) == 0 {
		return common.HashArray{}, nil
	}

	lastFinSlot := f.bc.GetLastFinalizedBlock().Slot()
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
	defer func(start time.Time) {
		log.Info("^^^^^^^^^^^^ TIME",
			"elapsed", common.PrettyDuration(time.Since(start)),
			"func:", "ValidateCandidates",
			"spines", len(spines),
		)
	}(time.Now())

	var (
		hasNotFin     = false          //has any not finalized items
		dagCandidates common.HashArray // current dag candidates
		optSpines     []common.HashArray
		err           error
	)

	mapHeaders := f.bc.GetHeadersByHashes(spines)
	for _, b := range mapHeaders {
		// if not found
		if b == nil {
			log.Error("IsValidSequenceOfSpines header not found", "headers", mapHeaders)
			return false, nil
		}
		// if block is not finalized
		if b.Nr() == 0 && b.Height > 0 {
			hasNotFin = true
		} else {
			// if block is finalized - check is it spine
			// TODO: check
			if b.Nr() != b.Height {
				log.Warn("IsValidSequenceOfSpines b.Nr() != b.Height", "b.Nr()", b.Nr(), "b.Nr()", b.Height, "hash", b.Hash())
				//return false, nil
			}
		}
	}

	if hasNotFin {
		fromSlot := f.bc.GetLastFinalizedHeader().Slot
		optSpines, err = f.bc.GetOptimisticSpines(fromSlot)
		if err != nil {
			log.Error("GetOptimisticSpines error", "slot", fromSlot, "err", err)
			return false, err
		}
		if len(optSpines) == 0 {
			log.Info("No spines found", "tips", f.bc.GetTips().Print(), "slot", fromSlot)
		}

		for _, candidate := range optSpines {
			if len(candidate) > 0 {
				//header := d.bc.GetHeaderByHash(candidate[0])
				dagCandidates = append(dagCandidates, candidate[0])
			}
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
				nxt := f.bc.GetHeaderByNumber(fnr)
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

// SetSpineState set state by past spine.
func (f *Finalizer) SetSpineState(spineHash *common.Hash, lfNr uint64) error {
	if spineHash == nil {
		log.Error("Set spine state: bad param", "spineHash", nil)
		return ErrBadParams
	}
	return f.bc.RollbackFinalization(*spineHash, lfNr)
}

// ForwardFinalization recalculate finalization params by skipping correctly finalized spines.
func (f *Finalizer) ForwardFinalization(spines common.HashArray, baseSpine common.Hash) (common.HashArray, common.Hash, error) {
	spn, bSpine, err := f.forwardFinalization(&spines, &baseSpine)
	return *spn, *bSpine, err
}

// forwardFinalization recalculate finalization params by skipping correctly finalized spines.
func (f *Finalizer) forwardFinalization(spines *common.HashArray, baseSpine *common.Hash) (*common.HashArray, *common.Hash, error) {
	if baseSpine == nil || spines == nil || len(*spines) == 0 {
		return spines, baseSpine, nil
	}

	lfNr := f.bc.GetLastFinalizedNumber()
	baseHeader := f.bc.GetHeader(*baseSpine)
	if baseHeader == nil || baseHeader.Height > 0 && baseHeader.Nr() == 0 {
		return spines, baseSpine, downloader.ErrInvalidBaseSpine
	}

	curSlot := baseHeader.Slot
	curNr := baseHeader.Nr()
	curIndex := 0

	for nr := curNr + 1; nr <= lfNr && curIndex < len(*spines); nr++ {
		header := f.bc.GetHeaderByNumber(nr)
		if header == nil {
			return spines, baseSpine, fmt.Errorf("header not found by nr")
		}
		//each slot increasing have to
		//correspond to the next spine
		if header.Slot > curSlot {
			curSlot = header.Slot
			curSpine := (*spines)[curIndex]
			if curSpine != header.Hash() {
				break
			}
			curIndex++
			baseSpine = &curSpine
		}
	}

	resSpine := (*spines)[curIndex:]

	log.Info("forward finalization: result",
		"baseSpine", fmt.Sprintf("%#x", baseSpine),
		"spines", resSpine,
	)

	return &resSpine, baseSpine, nil
}
