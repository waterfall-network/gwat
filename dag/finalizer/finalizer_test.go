package finalizer

import (
	"testing"

	"github.com/golang/mock/gomock"
	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/types"
	"gitlab.waterfall.network/waterfall/protocol/gwat/tests/testutils"
)

func TestSetSpineState(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	bc := NewMockBlockChain(ctrl)
	backend := NewMockBackend(ctrl)
	finalizer := &Finalizer{
		bc:  bc,
		eth: backend,
	}

	spineHash := common.HexToHash("0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421")
	lfNr := uint64(59)
	header := &types.Header{
		Number: &lfNr,
		Root:   spineHash,
	}
	num := uint64(50)
	num1 := uint64(60)
	num2 := uint64(61)
	num3 := uint64(62)
	unloaded := map[common.Hash]*types.Header{
		common.HexToHash("0x448172babe559718913088c08e2e28a04423ec9cac88e4d2eb58c7b907f9d9c5"): &types.Header{
			ParentHashes:  common.HashArray{common.HexToHash("0xea24f85eb3e8ffe7d0c1eb562bf56909495834ed766a4e2e08aec396643e81a9")},
			Slot:          3,
			Era:           0,
			Height:        1,
			Coinbase:      common.HexToAddress("0xa7062A2Bd7270740f1d15ab70B3Dee189A87b6DE"),
			TxHash:        common.HexToHash("0xad51b9a26e5b255d0e443389e598bcbcf2138fd2044dc2573d6d6fe00d383ec2"),
			BodyHash:      common.HexToHash("0x7e0c7bd288836a2c3b4672e3e75fdc94ee50204d79f406139be82149599e6e48"),
			CpHash:        common.HexToHash("0xea24f85eb3e8ffe7d0c1eb562bf56909495834ed766a4e2e08aec396643e81a9"),
			CpNumber:      0,
			CpRoot:        common.HexToHash("0x9230e5a1b66ae52a5249d2fee31149faa58d468abf184cc393a8cf296dad694d"),
			CpReceiptHash: common.HexToHash("0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421"),
			Number:        &num1,
			Root:          common.HexToHash("0x677c095ac4ea629570d2df78cc21123d83cd8287d0a347d2eb330db5c69b87f6"),
			ReceiptHash:   common.HexToHash("0x25e6b7af647c519a27cc13276a1e6abc46154b51414d174b072698df1f6c19df"),
		},
		common.HexToHash("0xad51b9a26e5b255d0e443389e598bcbcf2138fd2044dc2573d6d6fe00d383ec2"): &types.Header{
			ParentHashes:  common.HashArray{common.HexToHash("0xea24f85eb3e8ffe7d0c1eb562bf56909495834ed766a4e2e08aec396643e81a9")},
			Slot:          3,
			Era:           0,
			Height:        1,
			Coinbase:      common.HexToAddress("0xa7062A2Bd7270740f1d15ab70B3Dee189A87b6DE"),
			TxHash:        common.HexToHash("0x448172babe559718913088c08e2e28a04423ec9cac88e4d2eb58c7b907f9d9c5"),
			BodyHash:      common.HexToHash("0x7e0c7bd288836a2c3b4672e3e75fdc94ee50204d79f406139be82149599e6e48"),
			CpHash:        common.HexToHash("0xea24f85eb3e8ffe7d0c1eb562bf56909495834ed766a4e2e08aec396643e81a9"),
			CpNumber:      0,
			CpRoot:        common.HexToHash("0x9230e5a1b66ae52a5249d2fee31149faa58d468abf184cc393a8cf296dad694d"),
			CpReceiptHash: common.HexToHash("0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421"),
			Number:        &num2,
			Root:          common.HexToHash("0x677c095ac4ea629570d2df78cc21123d83cd8287d0a347d2eb330db5c69b87f6"),
			ReceiptHash:   common.HexToHash("0x25e6b7af647c519a27cc13276a1e6abc46154b51414d174b072698df1f6c19df"),
		},
		common.HexToHash("0x25e6b7af647c519a27cc13276a1e6abc46154b51414d174b072698df1f6c19df"): &types.Header{
			ParentHashes:  common.HashArray{common.HexToHash("0xea24f85eb3e8ffe7d0c1eb562bf56909495834ed766a4e2e08aec396643e81a9")},
			Slot:          3,
			Era:           0,
			Height:        1,
			Coinbase:      common.HexToAddress("0xa7062A2Bd7270740f1d15ab70B3Dee189A87b6DE"),
			TxHash:        common.HexToHash("0x448172babe559718913088c08e2e28a04423ec9cac88e4d2eb58c7b907f9d9c5"),
			BodyHash:      common.HexToHash("0x7e0c7bd288836a2c3b4672e3e75fdc94ee50204d79f406139be82149599e6e48"),
			CpHash:        common.HexToHash("0xea24f85eb3e8ffe7d0c1eb562bf56909495834ed766a4e2e08aec396643e81a9"),
			CpNumber:      0,
			CpRoot:        common.HexToHash("0x9230e5a1b66ae52a5249d2fee31149faa58d468abf184cc393a8cf296dad694d"),
			CpReceiptHash: common.HexToHash("0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421"),
			Number:        &num3,
			Root:          common.HexToHash("0x677c095ac4ea629570d2df78cc21123d83cd8287d0a347d2eb330db5c69b87f6"),
			ReceiptHash:   common.HexToHash("0xad51b9a26e5b255d0e443389e598bcbcf2138fd2044dc2573d6d6fe00d383ec2"),
		},
	}

	block := types.NewBlock(header, nil, nil, nil)
	bc.EXPECT().GetBlock(spineHash).AnyTimes().Return(block)
	bc.EXPECT().GetLastFinalizedBlock().AnyTimes().Return(block)
	bc.EXPECT().GetLastFinalizedHeader().AnyTimes().Return(header)
	bc.EXPECT().RollbackFinalization(gomock.AssignableToTypeOf(spineHash), gomock.AssignableToTypeOf(uint64(0))).AnyTimes()

	err := finalizer.SetSpineState(&spineHash, lfNr)
	testutils.AssertNoError(t, err)

	err = finalizer.SetSpineState(nil, lfNr)
	testutils.AssertError(t, err, ErrBadParams)

	bc.EXPECT().GetHeaderByNumber(gomock.AssignableToTypeOf(num)).AnyTimes().Return(header)
	bc.EXPECT().GetBlockDag(header.Hash()).AnyTimes().Return(nil)
	bc.EXPECT().CollectAncestorsAftCpByTips(gomock.AssignableToTypeOf(common.HashArray{}), gomock.AssignableToTypeOf(common.Hash{})).AnyTimes().Return(true, unloaded, nil, nil)
	bc.EXPECT().GetHeader(gomock.AssignableToTypeOf(common.Hash{})).AnyTimes().Return(header)
	bc.EXPECT().SaveBlockDag(gomock.AssignableToTypeOf(&types.BlockDAG{})).AnyTimes()

	err = finalizer.SetSpineState(&spineHash, num3)
	testutils.AssertNoError(t, err)
}
