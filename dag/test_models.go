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

package dag

import (
	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/types"
	"gitlab.waterfall.network/waterfall/protocol/gwat/tests/testutils"
)

var (
	spineBlockTest = types.NewBlock(&types.Header{
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
