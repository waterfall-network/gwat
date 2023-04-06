package core

import (
	"testing"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/rawdb"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/types"
	"gitlab.waterfall.network/waterfall/protocol/gwat/tests/testutils"
)

func TestUpdateSlotBlocks(t *testing.T) {
	slot := uint64(testutils.RandomInt(0, 100))
	hashesCount := testutils.RandomInt(5, 20)
	newBlock := types.NewBlock(&types.Header{
		Slot:   slot,
		TxHash: common.BytesToHash(testutils.RandomData(32)),
		Root:   common.BytesToHash(testutils.RandomData(32)),
	}, nil, nil, nil)

	db := rawdb.NewMemoryDatabase()
	bc := BlockChain{db: db}

	blocksHashes := common.HashArray{}
	for i := 0; i < hashesCount; i++ {
		blocksHashes = append(blocksHashes, common.BytesToHash(testutils.RandomData(32)))
	}

	rawdb.WriteSlotBlocksHashes(bc.Database(), slot, blocksHashes)
	slotBlocks := rawdb.ReadSlotBlocksHashes(bc.Database(), slot)
	testutils.AssertEqual(t, blocksHashes, slotBlocks)

	bc.UpdateSlotBlocks(newBlock)
	updatedSlotBlocks := rawdb.ReadSlotBlocksHashes(bc.Database(), slot)
	testutils.AssertEqual(t, append(blocksHashes, newBlock.Hash()), updatedSlotBlocks)
}
