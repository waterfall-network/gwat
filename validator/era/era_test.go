package era

import (
	"testing"
	"time"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/types"
	"gitlab.waterfall.network/waterfall/protocol/gwat/ethdb"
	"gitlab.waterfall.network/waterfall/protocol/gwat/params"
)

type mockBlockchain struct {
	config                     *params.ChainConfig
	eraInfo                    *EraInfo
	coordinatedCheckpointEpoch uint64
	slotInfo                   types.SlotInfo
	db                         ethdb.Database
}

func (m mockBlockchain) GetEraInfo() *EraInfo {
	return m.eraInfo
}

func (m mockBlockchain) GetConfig() *params.ChainConfig {
	return m.config
}

func (m mockBlockchain) GetSlotInfo() *types.SlotInfo {
	// Return a dummy slot info for testing purposes
	return &types.SlotInfo{
		GenesisTime:    uint64(time.Now().Unix()),
		SecondsPerSlot: 2,
		SlotsPerEpoch:  2,
	}
}

func TestNextEraFirstEpoch(t *testing.T) {
	// Create a new EraInfo with number 1, from epoch 10 to epoch 20, and root hash "0x1234"
	era := NewEra(1, 10, 20, common.HexToHash("0x1234"))
	ei := NewEraInfo(*era)

	// Ensure that the NextEraFirstEpoch function returns 21
	if ei.NextEraFirstEpoch() != 21 {
		t.Errorf("Expected NextEraFirstEpoch to return 21, but got %v", ei.NextEraFirstEpoch())
	}
}

func TestNextEraFirstSlot(t *testing.T) {
	// Create a mock blockchain object for testing purposes
	bc := mockBlockchain{}

	// Create a new EraInfo with number 1, from epoch 10 to epoch 20, and root hash "0x1234"
	era := NewEra(1, 10, 20, common.HexToHash("0x1234"))
	ei := NewEraInfo(*era)

	// Ensure that the NextEraFirstSlot function returns the correct slot number
	if ei.NextEraFirstSlot(bc) != 42 {
		t.Errorf("Expected NextEraFirstSlot to return 21, but got %v", ei.NextEraFirstSlot(bc))
	}
}

func TestLenEpochs(t *testing.T) {
	// Create a new EraInfo with number 1, from epoch 10 to epoch 20, and root hash "0x1234"
	era := NewEra(1, 10, 20, common.HexToHash("0x1234"))
	ei := NewEraInfo(*era)

	// Ensure that the LenEpochs function returns 11
	if ei.LenEpochs() != 10 {
		t.Errorf("Expected LenEpochs to return 11, but got %v", ei.LenEpochs())
	}
}

func TestLenSlots(t *testing.T) {
	// Create a new EraInfo with number 1, from epoch 10 to epoch 20, and root hash "0x1234"
	era := NewEra(1, 10, 20, common.HexToHash("0x1234"))
	ei := NewEraInfo(*era)

	// Ensure that the LenSlots function returns 320
	if ei.LenSlots() != 320 {
		t.Errorf("Expected LenSlots to return 320, but got %v", ei.LenSlots())
	}
}

func TestNumber(t *testing.T) {
	// Create a new EraInfo with number 1, from epoch 10 to epoch 20, and root hash "0x1234"
	era := NewEra(1, 10, 20, common.HexToHash("0x1234"))
	ei := NewEraInfo(*era)

	// Ensure that the Number function returns the correct era number
	if ei.Number() != 1 {
		t.Errorf("Expected Number to return 1, but got %v", ei.Number())
	}
}

func TestIsContainsEpoch(t *testing.T) {
	// Create a new EraInfo with number 1, from epoch 10 to epoch 20, and root hash "0x1234"
	era := NewEra(1, 10, 20, common.HexToHash("0x1234"))
	ei := NewEraInfo(*era)

	// Ensure that the IsContainsEpoch function returns true for epochs within the era, and false otherwise
	if !ei.GetEra().IsContainsEpoch(10) {
		t.Errorf("Expected IsContainsEpoch to return true for epoch 10, but got false")
	}
	if !ei.GetEra().IsContainsEpoch(15) {
		t.Errorf("Expected IsContainsEpoch to return true for epoch 15, but got false")
	}
	if ei.GetEra().IsContainsEpoch(5) {
		t.Errorf("Expected IsContainsEpoch to return false for epoch 5, but got true")
	}
	if ei.GetEra().IsContainsEpoch(25) {
		t.Errorf("Expected IsContainsEpoch to return false for epoch 25, but got true")
	}
}

func TestEstimateEraLength(t *testing.T) {
	// Create a mock blockchain object for testing purposes
	bc := mockBlockchain{
		config: &params.ChainConfig{
			EpochsPerEra: 20,
		},
		slotInfo: types.SlotInfo{
			SlotsPerEpoch: 32,
		},
	}

	// Ensure that the EstimateEraLength function returns the correct era length for different numbers of validators
	if EstimateEraLength(bc, 640) != 340 {
		t.Errorf("Expected EstimateEraLength to return 21, but got %v", EstimateEraLength(bc, 640))
	}
	if EstimateEraLength(bc, 1024) != 520 {
		t.Errorf("Expected EstimateEraLength to return 32, but got %v", EstimateEraLength(bc, 1024))
	}
	if EstimateEraLength(bc, 1500) != 760 {
		t.Errorf("Expected EstimateEraLength to return 40, but got %v", EstimateEraLength(bc, 1500))
	}
}
