package types

import (
	"database/sql/driver"
	"fmt"
	"reflect"
	"testing"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/tests/testutils"
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

func TestSelectBlocks(t *testing.T) {
	block1 := &Header{
		ParentHashes: nil,
		Height:       uint64(56),
	}
	block2 := &Header{
		ParentHashes: nil,
		Height:       uint64(24),
	}
	block3 := &Header{
		ParentHashes: nil,
		Height:       uint64(175),
	}
	block4 := &Header{
		ParentHashes: nil,
		Height:       uint64(1),
	}
	block5 := &Header{
		ParentHashes: nil,
		Height:       uint64(10),
	}
	block6 := &Header{
		ParentHashes: common.HashArray{
			common.Hash{0x01},
			common.Hash{0x02},
		},
		Height: uint64(100),
	}
	block7 := &Header{
		ParentHashes: common.HashArray{
			common.Hash{0x01},
		},
		Height: uint64(100),
	}
	block8 := &Header{
		ParentHashes: common.HashArray{
			common.Hash{0x01},
			common.Hash{0x02},
			common.Hash{0x03},
			common.Hash{0x04},
		},
		Height: uint64(100),
	}
	block9 := &Header{
		ParentHashes: common.HashArray{
			common.Hash{0x01},
			common.Hash{0x02},
			common.Hash{0x03},
			common.Hash{0x04},
			common.Hash{0x05},
		},
		Height: uint64(100),
	}
	block10 := &Header{
		ParentHashes: common.HashArray{
			common.Hash{0x01},
			common.Hash{0x02},
			common.Hash{0x03},
		},
		Height: uint64(100),
	}
	//0x928d024a377a5c37c4b9b8d87578ff3b7a3f5af03d504db6d04c7ac22bfe27dd
	block_92 := &Header{
		TxHash: common.Hash{0x01},
		ParentHashes: common.HashArray{
			common.Hash{0x01},
			common.Hash{0x02},
			common.Hash{0x03},
		},
		Height: uint64(100),
	}
	//0x3469716b8c69a8f5ec4e5c84e61e1e19b4f675f4902dd2b480368f0887391e27
	block_34 := &Header{
		TxHash: common.Hash{0x02},
		ParentHashes: common.HashArray{
			common.Hash{0x01},
			common.Hash{0x02},
			common.Hash{0x03},
		},
		Height: uint64(100),
	}
	//0x59d8e941c7fd22f73459b7dfdd3553857a1cba5a5d582081833ff7e020fa7c05
	block_59 := &Header{
		TxHash: common.Hash{0x03},
		ParentHashes: common.HashArray{
			common.Hash{0x01},
			common.Hash{0x02},
			common.Hash{0x03},
		},
		Height: uint64(100),
	}

	testCases := []struct {
		name           string
		blocks         Headers
		expectedBlocks Headers
		selectFunc     func(headers Headers) Headers
	}{
		{
			name:           "Missing blocks",
			blocks:         Headers{},
			expectedBlocks: Headers{},
			selectFunc: func(blocks Headers) Headers {
				return selectSpinesByMaxHeight(blocks)
			},
		},
		{
			name:           "Select blocks by height",
			blocks:         Headers{block1, block2, block3, block4, block5},
			expectedBlocks: Headers{block3},
			selectFunc: func(blocks Headers) Headers {
				return selectSpinesByMaxHeight(blocks)
			},
		}, {
			name:           "Select blocks by parents count",
			blocks:         Headers{block6, block7, block8, block9, block10},
			expectedBlocks: Headers{block9},
			selectFunc: func(blocks Headers) Headers {
				return selectSpinesByMaxParentsCount(blocks)
			},
		},
		{
			name:           "Sort by hash",
			blocks:         Headers{block_92, block_34, block_59},
			expectedBlocks: Headers{block_34, block_59, block_92},
			selectFunc: func(blocks Headers) Headers {
				sortByHash(blocks)
				return blocks
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			sortBlocks := testCase.selectFunc(testCase.blocks)
			testutils.AssertEqual(t, testCase.expectedBlocks.GetHashes(), sortBlocks.GetHashes())
		})
	}
}

func TestCalculateOptimisticCandidates(t *testing.T) {
	block1 := &Header{
		Slot:         1,
		ParentHashes: nil,
		Height:       10,
	}
	block2 := &Header{
		Slot:         1,
		ParentHashes: nil,
		Height:       8,
	}
	block3 := &Header{
		Slot: 2,
		ParentHashes: common.HashArray{
			common.Hash{0x01},
			common.Hash{0x02},
			common.Hash{0x03},
			common.Hash{0x04},
		},
		Height: 25,
	}
	block4 := &Header{
		Slot: 2,
		ParentHashes: common.HashArray{
			common.Hash{0x01},
			common.Hash{0x02},
			common.Hash{0x03},
			common.Hash{0x04},
			common.Hash{0x05},
		},
		Height: 25,
	}
	block5 := &Header{
		Slot: 2,
		ParentHashes: common.HashArray{
			common.Hash{0x01},
			common.Hash{0x02},
		},
		Height: 25,
	}
	//0x1144fb74d443ac36555080958ba8568d223126d178233f6a50f20b09a6f99ef3
	block_11 := &Header{
		Slot:   3,
		TxHash: common.Hash{0x015, 8},
		ParentHashes: common.HashArray{
			common.Hash{0x01},
			common.Hash{0x02},
		},
		Height: 30,
	}
	//0x2ae27f2e761ba1bf5ea57cab7e06b974092c01ad73bc216f3a68e82044a95a4a
	block_2a := &Header{
		Slot:   3,
		TxHash: common.Hash{0x02},
		ParentHashes: common.HashArray{
			common.Hash{0x01},
			common.Hash{0x02},
		},
		Height: 30,
	}
	//0x1cbdf17a23bf340113f21efb1c5ae080688017777e8f0acf3a2e10842a00a851
	block_1c := &Header{
		Slot:   3,
		TxHash: common.Hash{0x01},
		ParentHashes: common.HashArray{
			common.Hash{0x01},
			common.Hash{0x02},
		},
		Height: 30,
	}
	testCases := []struct {
		name           string
		blocks         Headers
		expectedBlocks common.HashArray
	}{
		{
			name:           "Sort by height",
			blocks:         Headers{block1, block2},
			expectedBlocks: common.HashArray{block1.Hash()},
		},
		{
			name:           "Sort by parents count",
			blocks:         Headers{block3, block4, block5},
			expectedBlocks: common.HashArray{block4.Hash()},
		},
		{
			name:           "Sort by hashes",
			blocks:         Headers{block_11, block_2a, block_1c},
			expectedBlocks: common.HashArray{block_11.Hash(), block_1c.Hash(), block_2a.Hash()},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			candidates, err := CalculateOptimisticSpines(testCase.blocks)
			testutils.AssertNoError(t, err)
			testutils.AssertEqual(t, testCase.expectedBlocks, candidates)
		})
	}

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

func TestSortBySlot(t *testing.T) {
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
	// hash: 0x6f27c060e9b7a2340704d6d80276ef5b4c0fc7a74be0af9db7935cbf45f979c0
	bl_1_1_6f := NewBlock(&Header{
		Height: 1,
		ParentHashes: common.HashArray{
			common.Hash{0x00},
		},
		Slot:     1,
		Coinbase: common.Address{0x10, 0x10},
	}, nil, nil, nil)
	//fmt.Println(bl_1_1_6f.Hash().Hex())

	// hash: 0x72c078056ddcbf6471b3e66c514253877313c1c0be760514b66cfd22b927a04a
	bl_1_1_72 := NewBlock(&Header{
		Height: 1,
		ParentHashes: common.HashArray{
			common.Hash{0x00},
		},
		Slot:     1,
		Coinbase: common.Address{0x99, 0x99},
	}, nil, nil, nil)
	//fmt.Println(bl_1_1_72.Hash().Hex())

	// hash: 0x46e2f5d29a22d540c2a43cfd48964b6ea8e17ef6f114ce66b423b2f19a1b4947
	bl_1_1_46 := NewBlock(&Header{
		Height: 1,
		ParentHashes: common.HashArray{
			common.Hash{0x00},
		},
		Slot:     1,
		Coinbase: common.Address{0x77, 0x77},
	}, nil, nil, nil)
	//fmt.Println(bl_1_1_46.Hash().Hex())

	// hash: 0xc33a3313ceb33701f05b90fd53225d654d1544a27ee6e9892f5e6e6d9f478b0a
	bl_1_2_c3 := NewBlock(&Header{
		Height: 1,
		ParentHashes: common.HashArray{
			common.Hash{0x00},
			common.Hash{0x01},
		},
		Slot:     1,
		Coinbase: common.Address{0x22, 0x22},
	}, nil, nil, nil)
	//fmt.Println(bl_1_2_c3.Hash().Hex())

	// hash: 0x94380592b3371e64a71b323366b23a26f3451f3f7e68abd8bd0084360f2b8a40
	bl_1_2_94 := NewBlock(&Header{
		Height: 1,
		ParentHashes: common.HashArray{
			common.Hash{0x00},
			common.Hash{0x01},
		},
		Slot:     1,
		Coinbase: common.Address{0x10, 0x10},
	}, nil, nil, nil)
	//fmt.Println(bl_1_2_94.Hash().Hex())

	// hash: 0xe17cde4767a3447b5dc3a5c167805a7fdc87bbbb8e535df2969121f410a1764e
	bl_1_2_e1 := NewBlock(&Header{
		Height: 1,
		ParentHashes: common.HashArray{
			common.Hash{0x00},
			common.Hash{0x01},
		},
		Slot:     1,
		Coinbase: common.Address{0x55, 0x55},
	}, nil, nil, nil)
	//fmt.Println(bl_1_2_e1.Hash().Hex())

	// hash: 0xdaded57ac08b50a31e0f328dc19b91e7242aef1aaf24ec6f1bebf53c75a0eb87
	bl_1_3_da := NewBlock(&Header{
		Height: 1,
		ParentHashes: common.HashArray{
			common.Hash{0x00},
			common.Hash{0x01},
			common.Hash{0x02},
		},
		Slot:     1,
		Coinbase: common.Address{0x77, 0x77},
	}, nil, nil, nil)
	//fmt.Println(bl_1_3_da.Hash().Hex())

	// hash: 0x5127a17866512da98b4cdec28c6d896985d7156e309a90a5c9a549ff9472d1cf
	bl_1_3_51 := NewBlock(&Header{
		Height: 1,
		ParentHashes: common.HashArray{
			common.Hash{0x00},
			common.Hash{0x01},
			common.Hash{0x02},
		},
		Slot:     1,
		Coinbase: common.Address{0x10, 0x10},
	}, nil, nil, nil)
	//fmt.Println(bl_1_3_51.Hash().Hex())

	// hash: 0x5c5a44ac1ca0d35d2e5492fee5b7303daff6698b2e15fcb1a2a84924b5228892
	bl_1_3_5c := NewBlock(&Header{
		Height: 1,
		ParentHashes: common.HashArray{
			common.Hash{0x00},
			common.Hash{0x01},
			common.Hash{0x02},
		},
		Slot:     1,
		Coinbase: common.Address{0x33, 0x33},
	}, nil, nil, nil)
	//fmt.Println(bl_1_3_5c.Hash().Hex())

	// hash: 0x1684d8ac661f71ce5a4cb1d6ee27f148c1d5790eb8b6478187e27fa2b8228b19
	bl_2_1_16 := NewBlock(&Header{
		Height: 2,
		ParentHashes: common.HashArray{
			common.Hash{0x00},
		},
		Slot:     1,
		Coinbase: common.Address{0x77},
	}, nil, nil, nil)
	//fmt.Println(bl_2_1_16.Hash().Hex())

	// hash: 0x4805387af643b2396f11e83442c9e52bfa7f2f6d925fb423fa929d6f8f40a285
	bl_2_1_48 := NewBlock(&Header{
		Height: 2,
		ParentHashes: common.HashArray{
			common.Hash{0x00},
		},
		Slot:     1,
		Coinbase: common.Address{0x10},
	}, nil, nil, nil)
	//fmt.Println(bl_2_1_48.Hash().Hex())

	// hash: 0x459f2d8cd49d0bdc0081952f052b8e09187ef6e593464b5ca05d9145262547c2
	bl_2_1_45 := NewBlock(&Header{
		Height: 2,
		ParentHashes: common.HashArray{
			common.Hash{0x00},
		},
		Slot:     1,
		Coinbase: common.Address{0x99},
	}, nil, nil, nil)
	//fmt.Println(bl_2_1_45.Hash().Hex())

	// hash: 0x0df4abd0f825d016a1892fe33b095928328b0917e21e11dac72bc1e0b54b7cec
	bl_2_2_0d := NewBlock(&Header{
		Height: 2,
		ParentHashes: common.HashArray{
			common.Hash{0x00},
			common.Hash{0x01},
		},
		Slot:     1,
		Coinbase: common.Address{0x44},
	}, nil, nil, nil)
	//fmt.Println(bl_2_2_0d.Hash().Hex())

	// hash: 0x4cac330c5162c9e23aab9524029e8c77b7ac49961f5429fa74c751a3a9f24789
	bl_2_2_4c := NewBlock(&Header{
		Height: 2,
		ParentHashes: common.HashArray{
			common.Hash{0x00},
			common.Hash{0x01},
		},
		Slot:     1,
		Coinbase: common.Address{0x22},
	}, nil, nil, nil)
	//fmt.Println(bl_2_2_4c.Hash().Hex())

	// hash: 0xe51b7eaa25f65e290876fcbd00beb5ca3335cd435c3471fb2ef95640fac46d38
	bl_2_2_e5 := NewBlock(&Header{
		Height: 2,
		ParentHashes: common.HashArray{
			common.Hash{0x00},
			common.Hash{0x01},
		},
		Slot:     1,
		Coinbase: common.Address{0x10, 0x10},
	}, nil, nil, nil)
	//fmt.Println(bl_2_2_e5.Hash().Hex())

	// hash: 0x6fc878a06fe890317c0161c62b2e92c190ea5b7176974e2402d2e49117572641
	bl_2_3_6f := NewBlock(&Header{
		Height: 2,
		ParentHashes: common.HashArray{
			common.Hash{0x00},
			common.Hash{0x01},
			common.Hash{0x02},
		},
		Slot:     1,
		Coinbase: common.Address{0x10, 0x10},
	}, nil, nil, nil)
	//fmt.Println(bl_2_3_6f.Hash().Hex())

	// hash: 0xb77ced335affa947b189271f33468dd2d74011a9bbe58ee9b77727eb127b20a6
	bl_2_3_b7 := NewBlock(&Header{
		Height: 2,
		ParentHashes: common.HashArray{
			common.Hash{0x00},
			common.Hash{0x01},
			common.Hash{0x02},
		},
		Slot:     1,
		Coinbase: common.Address{0x55},
	}, nil, nil, nil)
	//fmt.Println(bl_2_3_b7.Hash().Hex())

	// hash: 0x523d75c49a277a76871f972c3154be3ad526e3971897acb6532a2b8badcdc7e1
	bl_2_3_52 := NewBlock(&Header{
		Height: 2,
		ParentHashes: common.HashArray{
			common.Hash{0x00},
			common.Hash{0x01},
			common.Hash{0x02},
		},
		Slot:     1,
		Coinbase: common.Address{0x66},
	}, nil, nil, nil)
	//fmt.Println(bl_2_3_52.Hash().Hex())

	corectOrder := []*Block{
		bl_2_3_52,
		bl_2_3_6f,
		bl_2_3_b7,

		bl_2_2_0d,
		bl_2_2_4c,
		bl_2_2_e5,

		bl_2_1_16,
		bl_2_1_45,
		bl_2_1_48,

		bl_1_3_51,
		bl_1_3_5c,
		bl_1_3_da,

		bl_1_2_94,
		bl_1_2_c3,
		bl_1_2_e1,

		bl_1_1_46,
		bl_1_1_6f,
		bl_1_1_72,
	}

	tests := []struct {
		name string
		seq  []*Block
		want driver.Value
	}{
		{
			name: "sequence-1",
			seq: []*Block{
				bl_2_2_0d,
				bl_2_3_6f,
				bl_1_1_46,
				bl_2_2_4c,
				bl_1_1_6f,
				bl_2_2_e5,
				bl_1_1_72,
				bl_1_2_c3,
				bl_2_1_16,
				bl_1_2_94,
				bl_2_1_48,
				bl_2_1_45,
				bl_1_2_e1,
				bl_2_3_52,
				bl_1_3_51,
				bl_2_3_b7,
				bl_1_3_da,
				bl_1_3_5c,
			},
			want: fmt.Sprintf("%#v", getBlocksHashes(corectOrder)),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fmt.Sprintf("%#v", getBlocksHashes(SpineSortBlocks(tt.seq)))
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
