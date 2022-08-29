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
		return (blocks[i].Slot() > blocks[j].Slot()) ||
			(blocks[i].Height() > blocks[j].Height()) ||
			(blocks[i].Height() == blocks[j].Height() && len(blocks[i].ParentHashes()) > len(blocks[j].ParentHashes())) ||
			(blocks[i].Height() == blocks[j].Height() && len(blocks[i].ParentHashes()) == len(blocks[j].ParentHashes()) &&
				ibn.Cmp(jbn) < 0)
	})
	return blocks
}

func CalculateSpines(blocks Blocks) (SlotSpineMap, error) {
	blocksBySlot, err := blocks.GroupBySlot()
	if err != nil {
		return nil, err
	}
	spines := make(SlotSpineMap)
	for slot, bs := range blocksBySlot {
		sortedBlocks := SpineSortBlocks(bs)
		spines[slot] = sortedBlocks[0]
	}
	return spines, nil
}

func SpineGetDagChain(bc BlockChain, spine *Block) Blocks {
	candidatesInChain := make(map[common.Hash]struct{})
	chain := make(Blocks, 0)
	spineProcessBlock(bc, spine, candidatesInChain, &chain)
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
