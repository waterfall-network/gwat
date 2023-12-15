package dag

import (
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/rawdb"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/types"
	"gitlab.waterfall.network/waterfall/protocol/gwat/dag/creator"
	"gitlab.waterfall.network/waterfall/protocol/gwat/eth/downloader"
	"gitlab.waterfall.network/waterfall/protocol/gwat/ethdb"
	"gitlab.waterfall.network/waterfall/protocol/gwat/tests/testutils"
)

func TestHandleGetOptimisticSpines(t *testing.T) {
	ctrl := gomock.NewController(t)
	db := rawdb.NewMemoryDatabase()
	prepareTestBlocks(t, db)

	down := NewMockethDownloader(ctrl)
	down.EXPECT().Synchronising().Return(false)

	backend := NewMockBackend(ctrl)
	backend.EXPECT().Downloader().Return(&downloader.Downloader{})

	bc := NewMockblockChain(ctrl)
	bc.EXPECT().Database().AnyTimes().Return(db)
	bc.EXPECT().GetBlock(spineBlockTest.Hash()).Return(spineBlockTest)
	bc.EXPECT().GetSlotInfo().Return(&types.SlotInfo{
		GenesisTime:    uint64(time.Now().Unix() - 60),
		SecondsPerSlot: 4,
		SlotsPerEpoch:  32,
	})

	bc.EXPECT().GetHeaderByHash(spineBlockTest.Hash()).AnyTimes().Return(spineBlockTest.Header())
	bc.EXPECT().GetOptimisticSpines(spineBlockTest.Slot()).Return([]common.HashArray{{block.Hash()}, {block3.Hash()}, {block5.Hash(), block6.Hash()}, {block7.Hash()}}, nil)
	for _, testBlock := range testBlocks {
		bc.EXPECT().GetBlock(testBlock.Hash()).Return(testBlock)
	}

	dag := Dag{eth: backend, bc: bc, downloader: down}
	result := dag.HandleGetOptimisticSpines(spineBlockTest.Hash())

	expectedResult := types.OptimisticSpinesResult{
		Data:  []common.HashArray{{block.Hash()}, {block3.Hash()}, {block5.Hash(), block6.Hash()}, {block7.Hash()}},
		Error: nil,
	}
	testutils.AssertEqual(t, expectedResult, *result)

	//	Check if downloader is synchronising
	bc.EXPECT().GetBlock(spineBlockTest.Hash()).Return(spineBlockTest)
	down.EXPECT().Synchronising().Return(true)
	result = dag.HandleGetOptimisticSpines(spineBlockTest.Hash())
	expectedErr := creator.ErrSynchronization.Error()
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
