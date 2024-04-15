package dag

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/rawdb"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/types"
	"gitlab.waterfall.network/waterfall/protocol/gwat/eth/downloader"
	"gitlab.waterfall.network/waterfall/protocol/gwat/ethdb"
	"gitlab.waterfall.network/waterfall/protocol/gwat/params"
	"gitlab.waterfall.network/waterfall/protocol/gwat/tests/testutils"
)

func TestHandleGetOptimisticSpines(t *testing.T) {
	ctrl := gomock.NewController(t)
	db := rawdb.NewMemoryDatabase()
	prepareTestBlocks(t, db)

	down := NewMockethDownloader(ctrl)
	down.EXPECT().Synchronising().Return(false)

	backend := NewMockBackend(ctrl)
	backend.EXPECT().Downloader().Return(&downloader.Downloader{}).AnyTimes()

	bc := NewMockblockChain(ctrl)
	bc.EXPECT().Database().AnyTimes().Return(db)
	bc.EXPECT().GetBlock(context.Background(), spineBlockTest.Hash()).Return(spineBlockTest).AnyTimes()
	bc.EXPECT().GetSlotInfo().Return(&types.SlotInfo{
		GenesisTime:    uint64(time.Now().Unix() - 60),
		SecondsPerSlot: 4,
		SlotsPerEpoch:  32,
	}).AnyTimes()

	bc.EXPECT().GetHeaderByHash(spineBlockTest.Hash()).AnyTimes().Return(spineBlockTest.Header())
	bc.EXPECT().GetOptimisticSpines(spineBlockTest.Slot()).Return([]common.HashArray{{block.Hash()}, {block3.Hash()}, {block5.Hash(), block6.Hash()}, {block7.Hash()}}, nil)
	for _, testBlock := range testBlocks {
		bc.EXPECT().GetBlock(context.Background(), testBlock.Hash()).Return(testBlock).AnyTimes()
	}

	dag := Dag{eth: backend, bc: bc, downloader: down}
	result := dag.HandleGetOptimisticSpines(spineBlockTest.Hash())

	expectedResult := types.OptimisticSpinesResult{
		Data:  []common.HashArray{{block.Hash()}, {block3.Hash()}, {block5.Hash(), block6.Hash()}, {block7.Hash()}},
		Error: nil,
	}
	testutils.AssertEqual(t, expectedResult, *result)

	//	Check if downloader is synchronising
	bc.EXPECT().GetBlock(context.Background(), spineBlockTest.Hash()).Return(spineBlockTest).AnyTimes()
	down.EXPECT().Synchronising().Return(true)
	result = dag.HandleGetOptimisticSpines(spineBlockTest.Hash())
	expectedErr := errSynchronization.Error()
	expectedResult = types.OptimisticSpinesResult{
		Error: &expectedErr,
	}
	testutils.AssertEqual(t, expectedResult, *result)
}

func prepareTestBlocks(t *testing.T, db ethdb.Database) {
	blocksBySlots, err := testBlocks.GroupBySlot()
	testutils.AssertNoError(t, err)

	for slot, blocks := range blocksBySlots {
		rawdb.WriteSlotBlocksHashes(db, slot, *blocks.GetHashes())
	}
}

func TestIsCpUnacceptable(t *testing.T) {
	ctrl := gomock.NewController(t)

	down := NewMockethDownloader(ctrl)

	backend := NewMockBackend(ctrl)

	bc := NewMockblockChain(ctrl)
	bc.EXPECT().Config().AnyTimes().Return(&params.ChainConfig{
		AcceptCpRootOnFinEpoch: map[common.Hash][]uint64{common.Hash{0x11}: {456, 457}},
	})
	dag := Dag{eth: backend, bc: bc, downloader: down}

	var cp *types.Checkpoint

	// case: cp == nil (true)
	result := dag.isCpUnacceptable(cp)
	testutils.AssertEqual(t, true, result)

	// case: accepted cp (false)
	cp = &types.Checkpoint{
		FinEpoch: 456,
		Root:     common.Hash{0x11},
	}
	result = dag.isCpUnacceptable(cp)
	testutils.AssertEqual(t, false, result)

	// case: accepted cp (false)
	cp.FinEpoch = 457
	result = dag.isCpUnacceptable(cp)
	testutils.AssertEqual(t, false, result)

	// case: unaccepted cp (true)
	cp.FinEpoch = 458
	result = dag.isCpUnacceptable(cp)
	testutils.AssertEqual(t, true, result)

	// case: not defined root (false)
	cp.Root = common.Hash{0x22}
	result = dag.isCpUnacceptable(cp)
	testutils.AssertEqual(t, false, result)
}

func TestIsCpUnacceptable_NoConfParam(t *testing.T) {
	ctrl := gomock.NewController(t)

	down := NewMockethDownloader(ctrl)

	backend := NewMockBackend(ctrl)

	bc := NewMockblockChain(ctrl)
	bc.EXPECT().Config().AnyTimes().Return(&params.ChainConfig{})
	dag := Dag{eth: backend, bc: bc, downloader: down}

	var cp *types.Checkpoint

	// case: cp == nil (true)
	result := dag.isCpUnacceptable(cp)
	testutils.AssertEqual(t, true, result)

	// case: accepted cp (false)
	cp = &types.Checkpoint{
		FinEpoch: 456,
		Root:     common.Hash{0x11},
	}
	result = dag.isCpUnacceptable(cp)
	testutils.AssertEqual(t, false, result)
}
