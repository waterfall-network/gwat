package types

import (
	"math/big"
	"sort"

	"github.com/waterfall-foundation/gwat/common"
)

type BlockChain interface {
	GetBlockByHash(hash common.Hash) *Block
	GetBlocksByHashes(hashes common.HashArray) BlockMap
}

// SpineSortBlocks sorts hashes by order of finalization
func SpineSortBlocks(blocks []*Block) []*Block {
	sort.Slice(blocks, func(i, j int) bool {
		ibn := new(big.Int).SetBytes(blocks[i].Hash().Bytes())
		jbn := new(big.Int).SetBytes(blocks[j].Hash().Bytes())
		return (blocks[i].Height() > blocks[j].Height()) ||
			(blocks[i].Height() == blocks[j].Height() && len(blocks[i].ParentHashes()) > len(blocks[j].ParentHashes())) ||
			(blocks[i].Height() == blocks[j].Height() && len(blocks[i].ParentHashes()) == len(blocks[j].ParentHashes()) &&
				ibn.Cmp(jbn) < 0)
	})
	return blocks
}

// GetOrderedParentHashes get parent hashes sorted by order of finalization
func GetOrderedParentHashes(bc BlockChain, b *Block) common.HashArray {
	ph := b.ParentHashes()
	return SpineSortHashes(bc, ph)
}

// SpineSortHashes sorts hashes by order of finalization
func SpineSortHashes(bc BlockChain, hashes common.HashArray) common.HashArray {
	blocks := bc.GetBlocksByHashes(hashes)
	blocksArr := blocks.ToArray()

	blocksArr = SpineSortBlocks(blocksArr)

	orderedHashes := make(common.HashArray, 0, len(blocksArr))
	for _, block := range blocksArr {
		orderedHashes = append(orderedHashes, block.Hash())
	}

	return orderedHashes
}

func CalculateSpine(blocks Blocks) *Block {
	if len(blocks) == 0 {
		return nil
	}
	if len(blocks) == 1 {
		return blocks[0]
	}
	maxHeightBlocks := blocks.GetMaxHeightBlocks()
	if len(maxHeightBlocks) == 1 {
		return maxHeightBlocks[0]
	}
	maxParentHashesBlocks := maxHeightBlocks.GetMaxParentHashesLenBlocks()
	if len(maxParentHashesBlocks) == 1 {
		return maxParentHashesBlocks[0]
	}
	var maxHashIndex int
	for i := range maxParentHashesBlocks {
		if maxParentHashesBlocks[i].Hash().String() > maxParentHashesBlocks[maxHashIndex].Hash().String() {
			maxHashIndex = i
		}
	}
	return maxParentHashesBlocks[maxHashIndex]
}

func CalculateSpines(blocks Blocks) (SlotSpineHashMap, error) {
	blocksBySlot, err := blocks.GroupBySlot()
	if err != nil {
		return nil, err
	}
	spines := make(SlotSpineHashMap)
	for slot, bs := range blocksBySlot {
		spines[slot] = CalculateSpine(bs)
	}
	return spines, nil
}

func OrderChain(bc BlockChain, blocks Blocks) (Blocks, error) {
	spines, err := CalculateSpines(blocks)
	if err != nil {
		return nil, err
	}
	orderedChain := SpinesToChain(bc, &spines)
	return orderedChain, nil
}

func SpinesToChain(bc BlockChain, spines *SlotSpineHashMap) Blocks {
	candidatesInChain := make(map[common.Hash]struct{})
	maxSlot := spines.GetMaxSlot()
	minSlot := spines.GetMinSlot()
	chain := make(Blocks, 0)
	for slot := int(minSlot); slot <= int(maxSlot); slot++ {
		if _, ok := (*spines)[uint64(slot)]; !ok {
			continue
		}
		spine := (*spines)[uint64(slot)]
		spineProcessBlock(bc, spine, candidatesInChain, &chain)
	}
	return chain
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
	orderedHashes := GetOrderedParentHashes(bc, block)
	for _, ph := range orderedHashes {
		parent := bc.GetBlockByHash(ph)
		if _, wasProcessed := candidatesInChain[ph]; !wasProcessed && parent.Number() == nil {
			candidatesInChain[block.Hash()] = struct{}{}
			if chainPart := spineCalculateChain(bc, parent, candidatesInChain); len(chainPart) != 0 {
				*chain = append(*chain, chainPart...)
			}
			candidatesInChain[ph] = struct{}{}
		}
	}
	*chain = append(*chain, block)
}
