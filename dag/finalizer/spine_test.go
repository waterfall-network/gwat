package finalizer_test

import (
	"math"
	"math/rand"
	"testing"

	"github.com/waterfall-foundation/gwat/common"
	"github.com/waterfall-foundation/gwat/core"
	"github.com/waterfall-foundation/gwat/core/types"
	"github.com/waterfall-foundation/gwat/dag/finalizer"
)

type testSpineCase struct {
	input          types.Blocks
	expectedOutput *types.Block
	description    string
}

type BlockChainMock struct {
	blocks map[common.Hash]*types.Block
}

func (bc *BlockChainMock) GetBlockByHash(hash common.Hash) *types.Block {
	if _, ok := bc.blocks[hash]; ok {
		return bc.blocks[hash]
	}
	return nil
}

func (bc *BlockChainMock) GetBlocksByHashes(hashes common.HashArray) types.BlockMap {
	res := make(types.BlockMap, len(hashes))
	for _, hash := range hashes {
		if _, ok := bc.blocks[hash]; ok {
			res[hash] = bc.blocks[hash]
			continue
		}
		res[hash] = nil
	}
	return res
}

func NewBlock(header *types.Header) *types.Block {
	return types.NewBlock(header, nil, nil, nil)
}

func TestCalculateSpine(t *testing.T) {
	maxHeightExpectedOutput := NewBlock(&types.Header{Height: 2})
	maxParentHashesLenExpectedOutput := NewBlock(&types.Header{
		Height:       1,
		ParentHashes: make(common.HashArray, 2),
	})

	cases := []*testSpineCase{
		{
			input:          types.Blocks{},
			expectedOutput: nil,
			description:    "empty blocks slice",
		},
		{
			input:          nil,
			expectedOutput: nil,
			description:    "nil blocks slice",
		},
		{
			input: types.Blocks{
				NewBlock(&types.Header{
					Height: 1,
				}),
				maxHeightExpectedOutput,
			},
			expectedOutput: maxHeightExpectedOutput,
			description:    "maxHeight block",
		},
		{
			input: types.Blocks{
				NewBlock(&types.Header{
					Height:       1,
					ParentHashes: make(common.HashArray, 1),
				}),
				maxParentHashesLenExpectedOutput,
			},
			expectedOutput: maxParentHashesLenExpectedOutput,
			description:    "maxParentHashesLen block",
		},
	}

	maxHashBlockCase := &testSpineCase{
		input: types.Blocks{
			NewBlock(&types.Header{
				Height: 1,
				ParentHashes: common.HashArray{
					common.Hash{1},
				},
			}),
			NewBlock(&types.Header{
				Height: 1,
				ParentHashes: common.HashArray{
					common.Hash{2},
				},
			}),
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
		if output := finalizer.CalculateSpine(testCase.input); output != testCase.expectedOutput {
			t.Errorf("Failed \"%s\" \n", testCase.description)
		}
	}
}

func TestGroupBySlot(t *testing.T) {
	blocks := types.Blocks{
		NewBlock(&types.Header{
			Slot: 1,
		}),
		NewBlock(&types.Header{
			Slot: 2,
		}),
		NewBlock(&types.Header{
			Slot: 1,
		}),
	}
	res, err := blocks.GroupBySlot()
	if err != nil {
		t.Fatal(err.Error())
	}

	blocksCounter := 0
	addedBlocks := make(map[*types.Block]struct{})
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
	blocks := types.Blocks{
		NewBlock(&types.Header{
			Height: 1,
			ParentHashes: common.HashArray{
				common.Hash{},
			},
		}),
		NewBlock(&types.Header{
			Height: 1,
			ParentHashes: common.HashArray{
				common.Hash{},
				common.Hash{1},
			},
		}),
		NewBlock(&types.Header{
			Height: 1,
			ParentHashes: common.HashArray{
				common.Hash{},
				common.Hash{2},
			},
		}),
		NewBlock(&types.Header{
			Height: 2,
			ParentHashes: common.HashArray{
				common.Hash{},
			},
		}),
		NewBlock(&types.Header{
			Height: 2,
			ParentHashes: common.HashArray{
				common.Hash{},
				common.Hash{1},
			},
		}),
		NewBlock(&types.Header{
			Height: 2,
			ParentHashes: common.HashArray{
				common.Hash{},
				common.Hash{2},
			},
		}),
		NewBlock(&types.Header{
			Height: 3,
			ParentHashes: common.HashArray{
				common.Hash{},
			},
		}),
		NewBlock(&types.Header{
			Height: 3,
			ParentHashes: common.HashArray{
				common.Hash{},
				common.Hash{1},
			},
		}),
		NewBlock(&types.Header{
			Height: 3,
			ParentHashes: common.HashArray{
				common.Hash{},
				common.Hash{2},
			},
		}),
	}

	orderedBlocks := core.SortBlocks(blocks)
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

func addParents(blocks types.SlotSpineHashMap, blocksInChain map[common.Hash]struct{}, bc *BlockChainMock) {
	for _, block := range blocks {
		blocksInChain[block.Hash()] = struct{}{}
		for _, ph := range block.ParentHashes() {
			addParents(types.SlotSpineHashMap{0: bc.GetBlockByHash(ph)}, blocksInChain, bc)
		}
	}
}

func TestOrderChain(t *testing.T) {

	bc := BlockChainMock{
		blocks: make(map[common.Hash]*types.Block),
	}

	blocksPerSlot := 3
	slots := 3

	totalBlocksCount := blocksPerSlot * slots

	blocks := make(types.Blocks, 0, totalBlocksCount)

	for i := 0; i < totalBlocksCount; i++ {
		parentHashes := int(math.Min(float64(blocksPerSlot-1), float64(len(blocks))))

		ph := common.HashArray{}

		for j := 0; j < parentHashes; j++ {
			parentIndex := rand.Int() % len(blocks)
			parentHash := blocks[parentIndex].Hash()
			ph = append(ph, parentHash)
		}

		newBlock := NewBlock(
			&types.Header{
				Slot:         uint64(i / blocksPerSlot),
				Height:       uint64(i),
				ParentHashes: ph,
				Number:       nil,
			},
		)

		blocks = append(blocks, newBlock)
	}

	for _, block := range blocks {
		bc.blocks[block.Hash()] = block
	}

	blocksInChain := make(map[common.Hash]struct{})
	spines, _ := finalizer.CalculateSpines(blocks)

	addParents(spines, blocksInChain, &bc)

	orderedChain, err := finalizer.OrderChain(blocks, &bc)
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
