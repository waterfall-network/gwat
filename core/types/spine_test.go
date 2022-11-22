package types

import (
	"math"
	"math/rand"
	"testing"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
)

type testSpineCase struct {
	input          Blocks
	expectedOutput *Block
	description    string
}

type BlockChainMock struct {
	blocks map[common.Hash]*Block
}

func (bc *BlockChainMock) GetBlockByHash(hash common.Hash) *Block {
	if _, ok := bc.blocks[hash]; ok {
		return bc.blocks[hash]
	}
	return nil
}

func (bc *BlockChainMock) GetBlocksByHashes(hashes common.HashArray) BlockMap {
	res := make(BlockMap, len(hashes))
	for _, hash := range hashes {
		if _, ok := bc.blocks[hash]; ok {
			res[hash] = bc.blocks[hash]
			continue
		}
		res[hash] = nil
	}
	return res
}

func TestCalculateSpine(t *testing.T) {
	maxHeightExpectedOutput := NewBlock(&Header{Height: 2}, nil, nil, nil)
	maxParentHashesLenExpectedOutput := NewBlock(&Header{
		Height:       1,
		ParentHashes: make(common.HashArray, 2),
	}, nil, nil, nil)

	cases := []*testSpineCase{
		{
			input:          Blocks{},
			expectedOutput: nil,
			description:    "empty blocks slice",
		},
		{
			input:          nil,
			expectedOutput: nil,
			description:    "nil blocks slice",
		},
		{
			input: Blocks{
				NewBlock(&Header{
					Height: 1,
				}, nil, nil, nil),
				maxHeightExpectedOutput,
			},
			expectedOutput: maxHeightExpectedOutput,
			description:    "maxHeight block",
		},
		{
			input: Blocks{
				NewBlock(&Header{
					Height:       1,
					ParentHashes: make(common.HashArray, 1),
				}, nil, nil, nil),
				maxParentHashesLenExpectedOutput,
			},
			expectedOutput: maxParentHashesLenExpectedOutput,
			description:    "maxParentHashesLen block",
		},
	}

	maxHashBlockCase := &testSpineCase{
		input: Blocks{
			NewBlock(&Header{
				Height: 1,
				ParentHashes: common.HashArray{
					common.Hash{1},
				},
			}, nil, nil, nil),
			NewBlock(&Header{
				Height: 1,
				ParentHashes: common.HashArray{
					common.Hash{2},
				},
			}, nil, nil, nil),
		},
		description: "maxHash block",
	}
	if maxHashBlockCase.input[0].Hash().String() > maxHashBlockCase.input[1].Hash().String() {
		maxHashBlockCase.expectedOutput = maxHashBlockCase.input[0]
	} else {
		maxHashBlockCase.expectedOutput = maxHashBlockCase.input[1]
	}
	cases = append(cases, maxHashBlockCase)

	for _, testCase := range cases {
		if output := CalculateSpine(testCase.input); output != testCase.expectedOutput {
			t.Errorf("Failed \"%s\" \n", testCase.description)
		}
	}
}

func TestGroupBySlot(t *testing.T) {
	blocks := Blocks{
		NewBlock(&Header{
			Slot: 1,
		}, nil, nil, nil),
		NewBlock(&Header{
			Slot: 2,
		}, nil, nil, nil),
		NewBlock(&Header{
			Slot: 1,
		}, nil, nil, nil),
	}
	res, err := blocks.GroupBySlot()
	if err != nil {
		t.Fatal(err.Error())
	}

	blocksCounter := 0
	addedBlocks := make(map[*Block]struct{})
	for k, v := range res {
		for _, block := range v {
			if block.Slot() != k {
				t.Error("invalid slot")
			}
			if _, exists := addedBlocks[block]; exists {
				t.Error("block was added two or more times")
			} else {
				addedBlocks[block] = struct{}{}
				blocksCounter++
			}
		}
	}

	if blocksCounter != len(blocks) {
		t.Fatal("invalid result len")
	}
}

func TestSortBlocks(t *testing.T) {
	blocks := Blocks{
		NewBlock(&Header{
			Height: 1,
			ParentHashes: common.HashArray{
				common.Hash{},
			},
		}, nil, nil, nil),
		NewBlock(&Header{
			Height: 1,
			ParentHashes: common.HashArray{
				common.Hash{},
				common.Hash{1},
			},
		}, nil, nil, nil),
		NewBlock(&Header{
			Height: 1,
			ParentHashes: common.HashArray{
				common.Hash{},
				common.Hash{2},
			},
		}, nil, nil, nil),
		NewBlock(&Header{
			Height: 2,
			ParentHashes: common.HashArray{
				common.Hash{},
			},
		}, nil, nil, nil),
		NewBlock(&Header{
			Height: 2,
			ParentHashes: common.HashArray{
				common.Hash{},
				common.Hash{1},
			},
		}, nil, nil, nil),
		NewBlock(&Header{
			Height: 2,
			ParentHashes: common.HashArray{
				common.Hash{},
				common.Hash{2},
			},
		}, nil, nil, nil),
		NewBlock(&Header{
			Height: 3,
			ParentHashes: common.HashArray{
				common.Hash{},
			},
		}, nil, nil, nil),
		NewBlock(&Header{
			Height: 3,
			ParentHashes: common.HashArray{
				common.Hash{},
				common.Hash{1},
			},
		}, nil, nil, nil),
		NewBlock(&Header{
			Height: 3,
			ParentHashes: common.HashArray{
				common.Hash{},
				common.Hash{2},
			},
		}, nil, nil, nil),
	}

	orderedBlocks := SpineSortBlocks(blocks)
	for i, block := range orderedBlocks {
		if i == 0 {
			continue
		}
		if block.Height() > orderedBlocks[i-1].Height() {
			t.Error("invalid sort by height")
		}
		if block.Height() == orderedBlocks[i-1].Height() && len(block.ParentHashes()) > len(orderedBlocks[i-1].ParentHashes()) {
			t.Error("invalid sort by parent hashes")
		}
		if block.Height() == orderedBlocks[i-1].Height() &&
			len(block.ParentHashes()) == len(orderedBlocks[i-1].ParentHashes()) &&
			block.Hash().String() < orderedBlocks[i-1].Hash().String() {
			t.Error("invalid sort by hash")
		}
	}
}

func addParents(blocks SlotSpineMap, blocksInChain map[common.Hash]struct{}, bc *BlockChainMock) {
	for _, block := range blocks {
		blocksInChain[block.Hash()] = struct{}{}
		for _, ph := range block.ParentHashes() {
			addParents(SlotSpineMap{0: bc.GetBlockByHash(ph)}, blocksInChain, bc)
		}
	}
}

func TestOrderChain(t *testing.T) {

	bc := BlockChainMock{
		blocks: make(map[common.Hash]*Block),
	}

	blocksPerSlot := 3
	slots := 3

	totalBlocksCount := blocksPerSlot * slots

	blocks := make(Blocks, 0, totalBlocksCount)

	for i := 0; i < totalBlocksCount; i++ {
		parentHashes := int(math.Min(float64(blocksPerSlot-1), float64(len(blocks))))

		ph := common.HashArray{}

		for j := 0; j < parentHashes; j++ {
			parentIndex := rand.Int() % len(blocks)
			parentHash := blocks[parentIndex].Hash()
			ph = append(ph, parentHash)
		}

		newBlock := NewBlock(
			&Header{
				Slot:         uint64(i / blocksPerSlot),
				Height:       uint64(i),
				ParentHashes: ph,
				Number:       nil,
			}, nil, nil, nil,
		)

		blocks = append(blocks, newBlock)
	}

	for _, block := range blocks {
		bc.blocks[block.Hash()] = block
	}

	blocksInChain := make(map[common.Hash]struct{})
	spines, _ := CalculateSpines(blocks)

	addParents(spines, blocksInChain, &bc)

	orderedChain, err := OrderChain(&bc, blocks)
	if err != nil {
		t.Fatal(err)
	}

	if len(orderedChain) != len(blocksInChain) {
		t.Fatal("invalid ordered chain len")
	}

	resultBlocks := make(map[common.Hash]struct{})

	for _, block := range orderedChain {
		if _, exists := resultBlocks[block.Hash()]; exists {
			t.Errorf("hash %v was found twise", block.Hash())
		}
		resultBlocks[block.Hash()] = struct{}{}
	}

}
