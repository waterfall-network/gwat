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

func (m mockBlockchain) GetHeaderByHash(hash common.Hash) *types.Header {
	//TODO implement me
	panic("implement me")
}

func (m mockBlockchain) EnterNextEra(cp uint64, root, hash common.Hash) (*Era, error) {
	//TODO implement me
	panic("implement me")
}

func (m mockBlockchain) StartTransitionPeriod(cp *types.Checkpoint, spineRoot, spineHash common.Hash) error {
	//TODO implement me
	panic("implement me")
}

//func (m mockBlockchain) SyncEraToSlot(slot uint64) {
//	//TODO implement me
//	panic("implement me")
//}

func (m mockBlockchain) GetLastCoordinatedCheckpoint() *types.Checkpoint {
	//TODO implement me
	panic("implement me")
}

func (m mockBlockchain) GetEraInfo() *EraInfo {
	return m.eraInfo
}

func (m mockBlockchain) Config() *params.ChainConfig {
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
	era := NewEra(1, 10, 20, common.HexToHash("0x1234"), common.Hash{})
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
	era := NewEra(1, 10, 20, common.HexToHash("0x1234"), common.Hash{})
	ei := NewEraInfo(*era)

	// Ensure that the NextEraFirstSlot function returns the correct slot number
	if ei.NextEraFirstSlot(bc) != 42 {
		t.Errorf("Expected NextEraFirstSlot to return 21, but got %v", ei.NextEraFirstSlot(bc))
	}
}

func TestLenEpochs(t *testing.T) {
	// Create a new EraInfo with number 1, from epoch 10 to epoch 20, and root hash "0x1234"
	era := NewEra(1, 10, 20, common.HexToHash("0x1234"), common.Hash{})
	ei := NewEraInfo(*era)

	// Ensure that the LenEpochs function returns 11
	if ei.LenEpochs() != 10 {
		t.Errorf("Expected LenEpochs to return 11, but got %v", ei.LenEpochs())
	}
}

func TestLenSlots(t *testing.T) {
	// Create a new EraInfo with number 1, from epoch 10 to epoch 20, and root hash "0x1234"
	era := NewEra(1, 10, 20, common.HexToHash("0x1234"), common.Hash{})
	ei := NewEraInfo(*era)

	// Ensure that the LenSlots function returns 320
	if ei.LenSlots() != 320 {
		t.Errorf("Expected LenSlots to return 320, but got %v", ei.LenSlots())
	}
}

func TestNumber(t *testing.T) {
	// Create a new EraInfo with number 1, from epoch 10 to epoch 20, and root hash "0x1234"
	era := NewEra(1, 10, 20, common.HexToHash("0x1234"), common.Hash{})
	ei := NewEraInfo(*era)

	// Ensure that the Number function returns the correct era number
	if ei.Number() != 1 {
		t.Errorf("Expected Number to return 1, but got %v", ei.Number())
	}
}

func TestIsContainsEpoch(t *testing.T) {
	// Create a new EraInfo with number 1, from epoch 10 to epoch 20, and root hash "0x1234"
	era := NewEra(1, 10, 20, common.HexToHash("0x1234"), common.Hash{})
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
			EpochsPerEra:      20,
			SlotsPerEpoch:     32,
			ValidatorsPerSlot: 4,
			StartEpochsPerEra: 30,
		},
		slotInfo: types.SlotInfo{
			SlotsPerEpoch: 32,
		},
	}

	// Ensure that the EstimateEraLength function returns the correct era length for different numbers of validators
	if EstimateEraLength(bc.Config(), 1000, 0) != 20 {
		t.Errorf("Expected EstimateEraLength to return 20, but got %v", EstimateEraLength(bc.Config(), 1000, 0))
	}
	if EstimateEraLength(bc.Config(), 10000, 20) != 80 {
		t.Errorf("Expected EstimateEraLength to return 80, but got %v", EstimateEraLength(bc.Config(), 10000, 20))
	}
	if EstimateEraLength(bc.Config(), 20000, 30) != 20 {
		t.Errorf("Expected EstimateEraLength to return 160, but got %v", EstimateEraLength(bc.Config(), 20000, 30))
	}
}
