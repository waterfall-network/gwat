package types

import (
	"sort"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/log"
)

type BlockChain interface {
	GetBlockByHash(hash common.Hash) *Block
	GetBlocksByHashes(hashes common.HashArray) BlockMap
	GetLastFinalizedBlock() *Block
}

// SpineSortBlocks sorts hashes by order of finalization
func SpineSortBlocks(blocks []*Block) []*Block {
	heightPlenMap := map[uint64]map[uint64]map[common.Hash]*Block{}
	for _, bl := range blocks {
		h := bl.Height()
		plen := uint64(len(bl.ParentHashes()))
		hash := bl.Hash()
		if heightPlenMap[h] == nil {
			heightPlenMap[h] = map[uint64]map[common.Hash]*Block{}
		}
		if heightPlenMap[h][plen] == nil {
			heightPlenMap[h][plen] = map[common.Hash]*Block{}
		}
		heightPlenMap[h][plen][hash] = bl
	}

	//sort by height
	heightKeys := make(common.SorterDeskU64, 0, len(heightPlenMap))
	for k := range heightPlenMap {
		heightKeys = append(heightKeys, k)
	}
	sort.Sort(heightKeys)

	sortedBlocks := []*Block{}
	for _, hk := range heightKeys {
		plenMap := heightPlenMap[hk]
		// sort by number of parents
		plenKeys := make(common.SorterDeskU64, 0, len(plenMap))
		for plk, _ := range plenMap {
			plenKeys = append(plenKeys, plk)
		}
		sort.Sort(plenKeys)

		for _, k := range plenKeys {
			hashMap := plenMap[k]
			// sort by hash
			hashKeys := make(common.HashArray, 0, len(hashMap))
			for h, _ := range hashMap {
				hashKeys = append(hashKeys, h)
			}
			hashKeys = hashKeys.Sort()
			//add to sorted blocks
			for _, hash := range hashKeys {
				sortedBlocks = append(sortedBlocks, hashMap[hash])
			}
		}
	}

	return sortedBlocks
}

func CalculateSpines(blocks Blocks, lastFinSlot uint64) (SlotSpineMap, error) {
	blocksBySlot, err := blocks.GroupBySlot()
	if err != nil {
		return nil, err
	}
	spines := make(SlotSpineMap)
	//sort by slots
	slots := common.SorterAskU64{}
	for sl, _ := range blocksBySlot {
		// exclude finalized slots
		if sl > lastFinSlot {
			slots = append(slots, sl)
		}
	}
	sort.Sort(slots)
	for _, slot := range slots {
		slotBlocks := SpineSortBlocks(blocksBySlot[slot])
		if len(slotBlocks) == 0 {
			continue
		}
		spines[slot] = slotBlocks[0]
	}
	return spines, nil
}

func SpineGetDagChain(bc BlockChain, spine *Block) Blocks {
	// collect all ancestors in dag (not finalized)
	candidatesInChain := make(map[common.Hash]struct{})
	dagBlocks := make(Blocks, 0)
	spineProcessBlock(bc, spine, candidatesInChain, &dagBlocks)
	// sort by slot
	blocksBySlot, err := dagBlocks.GroupBySlot()
	if err != nil {
		//todo
		log.Error("☠ Ordering dag chain failed", "err", err)
	}
	//sort by slots
	slots := common.SorterAskU64{}
	for sl, _ := range blocksBySlot {
		slots = append(slots, sl)
	}
	sort.Sort(slots)

	orderedBlocks := Blocks{}
	for _, slot := range slots {
		// sort slot blocks
		slotBlocks := SpineSortBlocks(blocksBySlot[slot])
		if len(slotBlocks) == 0 {
			continue
		}
		orderedBlocks = append(orderedBlocks, slotBlocks...)
	}
	// todo rm
	// check that spine is the last in chain
	if len(orderedBlocks) > 0 {
		if orderedBlocks[len(orderedBlocks)-1].Hash() != spine.Hash() {
			panic("☠ Ordering of spine finalization chain is bad")
		}
	}
	return orderedBlocks
}

func spineCalculateChain(bc BlockChain, block *Block, candidatesInChain map[common.Hash]struct{}) Blocks {
	chain := make(Blocks, 0, len(block.ParentHashes()))
	spineProcessBlock(bc, block, candidatesInChain, &chain)
	return chain
}

func spineProcessBlock(bc BlockChain, block *Block, candidatesInChain map[common.Hash]struct{}, chain *Blocks) {
	if _, wasProcessed := candidatesInChain[block.Hash()]; wasProcessed || block.Number() != nil {
		return
	}
	parents := bc.GetBlocksByHashes(block.ParentHashes()).ToArray()
	sortedParents := SpineSortBlocks(parents)

	candidatesInChain[block.Hash()] = struct{}{}
	for _, parent := range sortedParents {
		if _, wasProcessed := candidatesInChain[parent.Hash()]; !wasProcessed && parent.Number() == nil {
			if chainPart := spineCalculateChain(bc, parent, candidatesInChain); len(chainPart) != 0 {
				*chain = append(*chain, chainPart...)
			}
			candidatesInChain[parent.Hash()] = struct{}{}
		}
	}
	*chain = append(*chain, block)
}
