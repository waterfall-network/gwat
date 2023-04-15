package dag

import (
	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/types"
	"gitlab.waterfall.network/waterfall/protocol/gwat/tests/testutils"
)

var (
	spineBlock = types.NewBlock(&types.Header{
		Slot:   11,
		Height: 11,
		TxHash: common.BytesToHash(testutils.RandomData(32)),
		Root:   common.BytesToHash(testutils.RandomData(32)),
	}, nil, nil, nil)

	block = types.NewBlock(&types.Header{
		Slot:   12,
		Height: 13,
		TxHash: common.BytesToHash(testutils.RandomData(32)),
		Root:   common.BytesToHash(testutils.RandomData(32)),
	}, nil, nil, nil)

	block2 = types.NewBlock(&types.Header{
		Slot:   12,
		Height: 12,
		TxHash: common.BytesToHash(testutils.RandomData(32)),
		Root:   common.BytesToHash(testutils.RandomData(32)),
	}, nil, nil, nil)

	block3 = types.NewBlock(&types.Header{
		Slot:   13,
		Height: 14,
		ParentHashes: common.HashArray{
			common.BytesToHash(testutils.RandomData(32)),
			common.BytesToHash(testutils.RandomData(32)),
			common.BytesToHash(testutils.RandomData(32)),
			common.BytesToHash(testutils.RandomData(32)),
		},
		TxHash: common.BytesToHash(testutils.RandomData(32)),
		Root:   common.BytesToHash(testutils.RandomData(32)),
	}, nil, nil, nil)

	block4 = types.NewBlock(&types.Header{
		Slot:   13,
		Height: 14,
		ParentHashes: common.HashArray{
			common.BytesToHash(testutils.RandomData(32)),
			common.BytesToHash(testutils.RandomData(32)),
		},
		TxHash: common.BytesToHash(testutils.RandomData(32)),
		Root:   common.BytesToHash(testutils.RandomData(32)),
	}, nil, nil, nil)

	block5 = types.NewBlock(&types.Header{
		Slot:   14,
		Height: 10,
		ParentHashes: common.HashArray{
			common.Hash{0x01},
			common.Hash{0x02},
			common.Hash{0x03},
			common.Hash{0x04},
			common.Hash{0x05},
		},
		TxHash: common.Hash{0x025, 255, 255, 125},
		Root:   common.Hash{0x025, 255, 255, 156},
	}, nil, nil, nil)

	block6 = types.NewBlock(&types.Header{
		Slot:   14,
		Height: 10,
		ParentHashes: common.HashArray{
			common.Hash{0x07},
			common.Hash{0x08},
			common.Hash{0x09},
			common.Hash{0x10},
			common.Hash{0x11},
		},
		TxHash: common.Hash{0x01},
		Root:   common.Hash{0x01},
	}, nil, nil, nil)

	block7 = types.NewBlock(&types.Header{
		Slot:   15,
		Height: 35,
		ParentHashes: common.HashArray{
			common.Hash{0x01},
		},
		TxHash: common.Hash{0x025, 255, 255},
		Root:   common.Hash{0x025, 255, 255},
	}, nil, nil, nil)

	testBlocks = types.Blocks{block, block2, block3, block4, block5, block6, block7}
)
