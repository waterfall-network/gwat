package core

import (
	"testing"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/types"
	"gitlab.waterfall.network/waterfall/protocol/gwat/internal/token/testutils"
)

func TestBreakByCreatorsPerSlotCount(t *testing.T) {
	bc := &BlockChain{}
	// Test input with 5 creators and 2 creators per slot
	creators := []common.Address{common.HexToAddress("0x1"), common.HexToAddress("0x2"), common.HexToAddress("0x3"), common.HexToAddress("0x4"), common.HexToAddress("0x5")}
	bc.slotInfo = &types.SlotInfo{SlotsPerEpoch: 2}
	expected := [][]common.Address{
		{common.HexToAddress("0x1"), common.HexToAddress("0x2")},
		{common.HexToAddress("0x3"), common.HexToAddress("0x4")},
	}

	result := bc.breakByCreatorsBySlotCount(creators, 2)

	testutils.AssertEqual(t, expected, result)
}
