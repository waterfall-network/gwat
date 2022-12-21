package types

import (
	"database/sql/driver"
	"fmt"
	"reflect"
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

//func TestCalculateSpine(t *testing.T) {
//	maxHeightExpectedOutput := NewBlock(&Header{Height: 2}, nil, nil, nil)
//	maxParentHashesLenExpectedOutput := NewBlock(&Header{
//		Height:       1,
//		ParentHashes: make(common.HashArray, 2),
//	}, nil, nil, nil)
//
//	cases := []*testSpineCase{
//		{
//			input:          Blocks{},
//			expectedOutput: nil,
//			description:    "empty blocks slice",
//		},
//		{
//			input:          nil,
//			expectedOutput: nil,
//			description:    "nil blocks slice",
//		},
//		{
//			input: Blocks{
//				NewBlock(&Header{
//					Height: 1,
//				}, nil, nil, nil),
//				maxHeightExpectedOutput,
//			},
//			expectedOutput: maxHeightExpectedOutput,
//			description:    "maxHeight block",
//		},
//		{
//			input: Blocks{
//				NewBlock(&Header{
//					Height:       1,
//					ParentHashes: make(common.HashArray, 1),
//				}, nil, nil, nil),
//				maxParentHashesLenExpectedOutput,
//			},
//			expectedOutput: maxParentHashesLenExpectedOutput,
//			description:    "maxParentHashesLen block",
//		},
//	}
//
//	maxHashBlockCase := &testSpineCase{
//		input: Blocks{
//			NewBlock(&Header{
//				Height: 1,
//				ParentHashes: common.HashArray{
//					common.Hash{1},
//				},
//			}, nil, nil, nil),
//			NewBlock(&Header{
//				Height: 1,
//				ParentHashes: common.HashArray{
//					common.Hash{2},
//				},
//			}, nil, nil, nil),
//		},
//		description: "maxHash block",
//	}
//	if maxHashBlockCase.input[0].Hash().String() > maxHashBlockCase.input[1].Hash().String() {
//		maxHashBlockCase.expectedOutput = maxHashBlockCase.input[0]
//	} else {
//		maxHashBlockCase.expectedOutput = maxHashBlockCase.input[1]
//	}
//	cases = append(cases, maxHashBlockCase)
//
//	for _, testCase := range cases {
//		if output := CalculateSpine(testCase.input); output != testCase.expectedOutput {
//			t.Errorf("Failed \"%s\" \n", testCase.description)
//		}
//	}
//}

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
	// hash: 0x1022e40d6eb8f76ac736da5955b5feefa2485d972159dccf5d1aac1bb8afde02
	bl_1_1_10 := NewBlock(&Header{
		Height: 1,
		ParentHashes: common.HashArray{
			common.Hash{0x00},
		},
		Slot:     1,
		Coinbase: common.Address{0x10, 0x10},
	}, nil, nil, nil)
	fmt.Println(bl_1_1_10.Hash().Hex())

	// hash: 0xeb1398459d792dc162b29fb6a39fc01f048c22aa3db0ac292ac90d6d4ebf67bd
	bl_1_1_eb := NewBlock(&Header{
		Height: 1,
		ParentHashes: common.HashArray{
			common.Hash{0x00},
		},
		Slot:     1,
		Coinbase: common.Address{0x99, 0x99},
	}, nil, nil, nil)
	fmt.Println(bl_1_1_eb.Hash().Hex())

	// hash: 0xf0a3dc02fd91e096e497bf3b3e24c58b2cb36b1796da470da7558fbbd212f41b
	bl_1_1_f0 := NewBlock(&Header{
		Height: 1,
		ParentHashes: common.HashArray{
			common.Hash{0x00},
		},
		Slot:     1,
		Coinbase: common.Address{0x77, 0x77},
	}, nil, nil, nil)
	fmt.Println(bl_1_1_f0.Hash().Hex())

	// hash: 0x0f3920e993ac0819034c82804860c2116154ad1d2bb1153af61140229b6b4722
	bl_1_2_0f := NewBlock(&Header{
		Height: 1,
		ParentHashes: common.HashArray{
			common.Hash{0x00},
			common.Hash{0x01},
		},
		Slot:     1,
		Coinbase: common.Address{0x22, 0x22},
	}, nil, nil, nil)
	fmt.Println(bl_1_2_0f.Hash().Hex())

	// hash: 0x1eb775dd4986d2c6d5d376491931ab66c3d94d25994a9d6ce4e3f4710054bf43
	bl_1_2_1e := NewBlock(&Header{
		Height: 1,
		ParentHashes: common.HashArray{
			common.Hash{0x00},
			common.Hash{0x01},
		},
		Slot:     1,
		Coinbase: common.Address{0x10, 0x10},
	}, nil, nil, nil)
	fmt.Println(bl_1_2_1e.Hash().Hex())

	// hash: 0x9a8be4e7dc146fcc3401dea07c50dc70e4e68d64ab867a831d2fd55ab172bb73
	bl_1_2_9a := NewBlock(&Header{
		Height: 1,
		ParentHashes: common.HashArray{
			common.Hash{0x00},
			common.Hash{0x01},
		},
		Slot:     1,
		Coinbase: common.Address{0x55, 0x55},
	}, nil, nil, nil)
	fmt.Println(bl_1_2_9a.Hash().Hex())

	// hash: 0x3278bfffa0aabca5653e4419bfd5673ab196ae3690e7272cabfeaef16f3b624f
	bl_1_3_32 := NewBlock(&Header{
		Height: 1,
		ParentHashes: common.HashArray{
			common.Hash{0x00},
			common.Hash{0x01},
			common.Hash{0x02},
		},
		Slot:     1,
		Coinbase: common.Address{0x77, 0x77},
	}, nil, nil, nil)
	fmt.Println(bl_1_3_32.Hash().Hex())

	// hash: 0x37ad7f596ceb520f06fb5e8e66e1e28ca574895f45d36e2b6ec9fcde5f3d4f23
	bl_1_3_37 := NewBlock(&Header{
		Height: 1,
		ParentHashes: common.HashArray{
			common.Hash{0x00},
			common.Hash{0x01},
			common.Hash{0x02},
		},
		Slot:     1,
		Coinbase: common.Address{0x10, 0x10},
	}, nil, nil, nil)
	fmt.Println(bl_1_3_37.Hash().Hex())

	// hash: 0xa5b6939a5c9c2ac36bbc0dfbab6158e67f9253f84c90c1ef9f161d4e2509469d
	bl_1_3_a5 := NewBlock(&Header{
		Height: 1,
		ParentHashes: common.HashArray{
			common.Hash{0x00},
			common.Hash{0x01},
			common.Hash{0x02},
		},
		Slot:     1,
		Coinbase: common.Address{0x33, 0x33},
	}, nil, nil, nil)
	fmt.Println(bl_1_3_a5.Hash().Hex())

	// hash: 0xa96581afc4e419ba31db2d5c1e000ee3a8ee7e9df343e338e573738ee33d6103
	bl_2_1_a9 := NewBlock(&Header{
		Height: 2,
		ParentHashes: common.HashArray{
			common.Hash{0x00},
		},
		Slot:     1,
		Coinbase: common.Address{0x77},
	}, nil, nil, nil)
	fmt.Println(bl_2_1_a9.Hash().Hex())

	// hash: 0xc5cf17853f214139ed23d3785adbffb866632732d3dc7ba0f3a9796fcfe65a54
	bl_2_1_c5 := NewBlock(&Header{
		Height: 2,
		ParentHashes: common.HashArray{
			common.Hash{0x00},
		},
		Slot:     1,
		Coinbase: common.Address{0x10},
	}, nil, nil, nil)
	fmt.Println(bl_2_1_c5.Hash().Hex())

	// hash: 0xf50297a79dc1bccea0d58876cf2793ef0fdb93fc20510bdda8b0055c353d3d4a
	bl_2_1_f5 := NewBlock(&Header{
		Height: 2,
		ParentHashes: common.HashArray{
			common.Hash{0x00},
		},
		Slot:     1,
		Coinbase: common.Address{0x99},
	}, nil, nil, nil)
	fmt.Println(bl_2_1_f5.Hash().Hex())

	// hash: 0x27f1c39b0905151984589e3a47685979298048e019eeeb04fb3880f875996db8
	bl_2_2_27 := NewBlock(&Header{
		Height: 2,
		ParentHashes: common.HashArray{
			common.Hash{0x00},
			common.Hash{0x01},
		},
		Slot:     1,
		Coinbase: common.Address{0x44},
	}, nil, nil, nil)
	fmt.Println(bl_2_2_27.Hash().Hex())

	// hash: 0x5a1914dc7c0a70b5359fceabf98a6aa5b5df0b2ca3e8dcb370cf529aa7af38a3
	bl_2_2_5a := NewBlock(&Header{
		Height: 2,
		ParentHashes: common.HashArray{
			common.Hash{0x00},
			common.Hash{0x01},
		},
		Slot:     1,
		Coinbase: common.Address{0x22},
	}, nil, nil, nil)
	fmt.Println(bl_2_2_5a.Hash().Hex())

	// hash: 0xaf1274d8c3aeafbffa48c5139450efcd5e337e8ac9c7bbe5e6840988f17fa783
	bl_2_2_af := NewBlock(&Header{
		Height: 2,
		ParentHashes: common.HashArray{
			common.Hash{0x00},
			common.Hash{0x01},
		},
		Slot:     1,
		Coinbase: common.Address{0x10, 0x10},
	}, nil, nil, nil)
	fmt.Println(bl_2_2_af.Hash().Hex())

	// hash: 0xe4aef5b721840e99b399409fd1406e2cccba3143ed4ad5e8ce3b2f732380ecff
	bl_2_3_e4 := NewBlock(&Header{
		Height: 2,
		ParentHashes: common.HashArray{
			common.Hash{0x00},
			common.Hash{0x01},
			common.Hash{0x02},
		},
		Slot:     1,
		Coinbase: common.Address{0x10, 0x10},
	}, nil, nil, nil)
	fmt.Println(bl_2_3_e4.Hash().Hex())

	// hash: 0xea8664fd377c38eb2dbac1b0c90d98b8275c5b97a8ef5974e2b11e72c37332b7
	bl_2_3_ea := NewBlock(&Header{
		Height: 2,
		ParentHashes: common.HashArray{
			common.Hash{0x00},
			common.Hash{0x01},
			common.Hash{0x02},
		},
		Slot:     1,
		Coinbase: common.Address{0x55},
	}, nil, nil, nil)
	fmt.Println(bl_2_3_ea.Hash().Hex())

	// hash: 0xd3e7c1eb394d390c53e2b2d97ff0e2e18bcf086215d941f7220530ad111746c6
	bl_2_3_d3 := NewBlock(&Header{
		Height: 2,
		ParentHashes: common.HashArray{
			common.Hash{0x00},
			common.Hash{0x01},
			common.Hash{0x02},
		},
		Slot:     1,
		Coinbase: common.Address{0x66},
	}, nil, nil, nil)
	fmt.Println(bl_2_3_d3.Hash().Hex())

	corectOrder := []*Block{
		bl_2_3_e4,
		bl_2_3_ea,
		bl_2_3_d3,

		bl_2_2_27,
		bl_2_2_5a,
		bl_2_2_af,

		bl_2_1_a9,
		bl_2_1_c5,
		bl_2_1_f5,

		bl_1_3_32,
		bl_1_3_37,
		bl_1_3_a5,

		bl_1_2_0f,
		bl_1_2_1e,
		bl_1_2_9a,

		bl_1_1_10,
		bl_1_1_eb,
		bl_1_1_f0,
	}

	tests := []struct {
		name string
		seq  []*Block
		want driver.Value
	}{
		{
			name: "sequence-1",

			seq: corectOrder,

			want: fmt.Sprintf("%#v", getBlocksHashes(corectOrder)),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fmt.Sprintf("%#v", getBlocksHashes(SpineSortBlocks(tt.seq)))

			fmt.Println(got)

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got:  %v\nwant: %v", got, tt.want)
			}
		})
	}
}

func getBlocksHashes(blocks []*Block) common.HashArray {
	ha := make(common.HashArray, len(blocks))
	for i, block := range blocks {
		ha[i] = block.Hash()
	}
	return ha
}

func addParents(blocks SlotSpineMap, blocksInChain map[common.Hash]struct{}, bc *BlockChainMock) {
	for _, block := range blocks {
		blocksInChain[block.Hash()] = struct{}{}
		for _, ph := range block.ParentHashes() {
			addParents(SlotSpineMap{0: bc.GetBlockByHash(ph)}, blocksInChain, bc)
		}
	}
}

//func TestOrderChain(t *testing.T) {
//
//	bc := BlockChainMock{
//		blocks: make(map[common.Hash]*Block),
//	}
//
//	blocksPerSlot := 3
//	slots := 3
//
//	totalBlocksCount := blocksPerSlot * slots
//
//	blocks := make(Blocks, 0, totalBlocksCount)
//
//	for i := 0; i < totalBlocksCount; i++ {
//		parentHashes := int(math.Min(float64(blocksPerSlot-1), float64(len(blocks))))
//
//		ph := common.HashArray{}
//
//		for j := 0; j < parentHashes; j++ {
//			parentIndex := rand.Int() % len(blocks)
//			parentHash := blocks[parentIndex].Hash()
//			ph = append(ph, parentHash)
//		}
//
//		newBlock := NewBlock(
//			&Header{
//				Slot:         uint64(i / blocksPerSlot),
//				Height:       uint64(i),
//				ParentHashes: ph,
//				Number:       nil,
//			}, nil, nil, nil,
//		)
//
//		blocks = append(blocks, newBlock)
//	}
//
//	for _, block := range blocks {
//		bc.blocks[block.Hash()] = block
//	}
//
//	blocksInChain := make(map[common.Hash]struct{})
//	spines, _ := CalculateSpines(blocks)
//
//	addParents(spines, blocksInChain, &bc)
//
//	orderedChain, err := OrderChain(&bc, blocks)
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	if len(orderedChain) != len(blocksInChain) {
//		t.Fatal("invalid ordered chain len")
//	}
//
//	resultBlocks := make(map[common.Hash]struct{})
//
//	for _, block := range orderedChain {
//		if _, exists := resultBlocks[block.Hash()]; exists {
//			t.Errorf("hash %v was found twise", block.Hash())
//		}
//		resultBlocks[block.Hash()] = struct{}{}
//	}
//
//}
