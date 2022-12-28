package types

import (
	"sort"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
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
