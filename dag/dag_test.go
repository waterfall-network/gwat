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

func TestGetOptimisticSpines(t *testing.T) {
	ctrl := gomock.NewController(t)
	bc := NewMockblockChain(ctrl)
	db := rawdb.NewMemoryDatabase()
	prepareTestBlocks(t, db)

	down := NewMockethDownloader(ctrl)
	down.EXPECT().Synchronising().Return(false)

	backend := NewMockBackend(ctrl)
	backend.EXPECT().Downloader().Return(&downloader.Downloader{})

	bc.EXPECT().Database().AnyTimes().Return(db)
	bc.EXPECT().GetBlock(spineBlock.Hash()).Return(spineBlock)
	bc.EXPECT().GetSlotInfo().AnyTimes().Return(&types.SlotInfo{
		GenesisTime:    uint64(time.Now().Unix() - 600),
		SecondsPerSlot: 4,
		SlotsPerEpoch:  32,
	})

	for _, testBlock := range testBlocks {
		bc.EXPECT().GetBlock(testBlock.Hash()).Return(testBlock)
	}

	dag := Dag{eth: backend, bc: bc, downloader: down}
	result := dag.HandleGetOptimisticSpines(spineBlock.Hash())

	expectedResult := types.OptimisticSpinesResult{
		Data:  []common.HashArray{{block.Hash()}, {block3.Hash()}, {block5.Hash(), block6.Hash()}, {block7.Hash()}},
		Error: nil,
	}
	testutils.AssertEqual(t, expectedResult, *result)

	//	Check if downloader is synchronising
	bc.EXPECT().GetBlock(spineBlock.Hash()).Return(spineBlock)
	down.EXPECT().Synchronising().Return(true)
	result = dag.HandleGetOptimisticSpines(spineBlock.Hash())
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
