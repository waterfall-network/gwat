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

package types

import (
	"bytes"
	"fmt"
	"sort"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/log"
)

type BlockChain interface {
	GetBlockByHash(hash common.Hash) *Block
	GetBlocksByHashes(hashes common.HashArray) BlockMap
	GetLastFinalizedBlock() *Block
	GetBlockFinalizedNumber(hash common.Hash) *uint64
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
	heightKeys := make(common.SorterDescU64, 0, len(heightPlenMap))
	for k := range heightPlenMap {
		heightKeys = append(heightKeys, k)
	}
	sort.Sort(heightKeys)

	sortedBlocks := []*Block{}
	for _, hk := range heightKeys {
		plenMap := heightPlenMap[hk]
		// sort by number of parents
		plenKeys := make(common.SorterDescU64, 0, len(plenMap))
		for plk := range plenMap {
			plenKeys = append(plenKeys, plk)
		}
		sort.Sort(plenKeys)

		for _, k := range plenKeys {
			hashMap := plenMap[k]
			// sort by hash
			hashKeys := make(common.HashArray, 0, len(hashMap))
			for h := range hashMap {
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
	slots := common.SorterAscU64{}
	for sl := range blocksBySlot {
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

func CalculateOptimisticSpines(slotBlocks Headers) (common.HashArray, error) {
	optimisticSpines := make(common.HashArray, 0)
	slotSpines := selectSpinesByMaxHeight(slotBlocks)

	if len(slotSpines) > 1 {
		slotSpines = selectSpinesByMaxParentsCount(slotSpines)
	}

	if len(slotSpines) > 1 {
		sortByHash(slotSpines)
	}

	optimisticSpines = append(optimisticSpines, *slotSpines.GetHashes()...)

	return optimisticSpines, nil
}

// OptimisticSortSlotHeaders sorts headers of the same slot by optimistic order.
func OptimisticSortSlotHeaders(slotHeaders []*Header) common.HashArray {
	heightPlenMap := map[uint64]map[uint64]common.HashArray{}
	for _, bl := range slotHeaders {
		h := bl.Height
		plen := uint64(len(bl.ParentHashes))
		hash := bl.Hash()
		if heightPlenMap[h] == nil {
			heightPlenMap[h] = map[uint64]common.HashArray{}
		}
		heightPlenMap[h][plen] = append(heightPlenMap[h][plen], hash)
	}

	//sort by height
	heightKeys := make(common.SorterDescU64, 0, len(heightPlenMap))
	for k := range heightPlenMap {
		heightKeys = append(heightKeys, k)
	}
	sort.Sort(heightKeys)

	sortedHashes := make(common.HashArray, 0, len(slotHeaders))
	for _, hk := range heightKeys {
		plenMap := heightPlenMap[hk]
		// sort by number of parents
		plenKeys := make(common.SorterDescU64, 0, len(plenMap))
		for plk := range plenMap {
			plenKeys = append(plenKeys, plk)
		}
		sort.Sort(plenKeys)
		for _, k := range plenKeys {
			hashKeys := plenMap[k]
			// sort by hash
			sort.Slice(hashKeys, func(i, j int) bool {
				cmp := bytes.Compare(hashKeys[i][:], hashKeys[j][:]) < 0
				return cmp
			})
			sortedHashes = append(sortedHashes, hashKeys...)
		}
	}
	return sortedHashes
}

func selectSpinesByMaxParentsCount(headers Headers) Headers {
	var maxParents uint64
	maxParentsBlocks := make(Headers, 0)
	for _, header := range headers {
		blockParents := uint64(len(header.ParentHashes))

		if blockParents < maxParents {
			continue
		}

		if blockParents > maxParents {
			maxParents = blockParents
			maxParentsBlocks = make(Headers, 0)
		}

		maxParentsBlocks = append(maxParentsBlocks, header)
	}

	return maxParentsBlocks
}

func sortByHash(headers Headers) {
	sort.Slice(headers, func(i, j int) bool {
		cmp := bytes.Compare(headers[i].Hash().Bytes(), headers[j].Hash().Bytes()) < 0
		return cmp
	})
}

func selectSpinesByMaxHeight(blocks Headers) Headers {
	if len(blocks) == 0 {
		return Headers{}
	}

	var maxHeight uint64
	maxHeightBlocks := make(Headers, 0)
	for _, header := range blocks {
		blockHeight := header.Height
		if blockHeight < maxHeight {
			continue
		}
		if blockHeight > maxHeight {
			maxHeight = blockHeight
			maxHeightBlocks = make(Headers, 0)
		}
		maxHeightBlocks = append(maxHeightBlocks, header)
	}
	return maxHeightBlocks
}

func SpineGetDagChain(bc BlockChain, spine *Block) (Blocks, error) {
	// collect all ancestors in dag (not finalized)
	candidatesInChain := make(map[common.Hash]struct{})
	dagBlocks := make(Blocks, 0)
	err := spineProcessBlock(bc, spine, candidatesInChain, &dagBlocks)
	if err != nil {
		log.Error("☠ Calculate dag chain failed (process block)", "err", err)
		return nil, err
	}
	// sort by slot
	blocksBySlot, err := dagBlocks.GroupBySlot()
	if err != nil {
		log.Error("☠ Calculate dag chain failed (group by slot)", "err", err)
		return nil, err
	}
	//sort by slots
	slots := common.SorterAscU64{}
	for sl := range blocksBySlot {
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

	// check that spine is the last in chain
	if len(orderedBlocks) > 0 {
		if orderedBlocks[len(orderedBlocks)-1].Hash() != spine.Hash() {
			err = fmt.Errorf("bad sequence")
			log.Error("☠ Calculate dag chain failed (bad sequence)", "err", err)
			return nil, err
		}
	}
	return orderedBlocks, nil
}

func spineCalculateChain(bc BlockChain, block *Block, candidatesInChain map[common.Hash]struct{}) (Blocks, error) {
	chain := make(Blocks, 0, len(block.ParentHashes()))
	err := spineProcessBlock(bc, block, candidatesInChain, &chain)
	if err != nil {
		return nil, err
	}
	return chain, nil
}

func spineProcessBlock(bc BlockChain, block *Block, candidatesInChain map[common.Hash]struct{}, chain *Blocks) error {
	if _, wasProcessed := candidatesInChain[block.Hash()]; wasProcessed || block.Number() != nil {
		return nil
	}
	parentsMap := bc.GetBlocksByHashes(block.ParentHashes())
	parents := make([]*Block, 0, len(parentsMap))
	for h, b := range parentsMap {
		if b == nil {
			return fmt.Errorf("block not found hash=%#x", h)
		}
		parents = append(parents, b)
	}
	sortedParents := SpineSortBlocks(parents)

	candidatesInChain[block.Hash()] = struct{}{}
	for _, parent := range sortedParents {
		nr := bc.GetBlockFinalizedNumber(parent.Hash())
		if _, wasProcessed := candidatesInChain[parent.Hash()]; !wasProcessed && nr == nil {
			chainPart, err := spineCalculateChain(bc, parent, candidatesInChain)
			if err != nil {
				return err
			}
			if len(chainPart) != 0 {
				*chain = append(*chain, chainPart...)
			}
			candidatesInChain[parent.Hash()] = struct{}{}
		}
	}
	*chain = append(*chain, block)
	return nil
}
