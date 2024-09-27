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

package storage

import (
	"context"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/state"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/types"
	"gitlab.waterfall.network/waterfall/protocol/gwat/tests/testutils"
	"gitlab.waterfall.network/waterfall/protocol/gwat/validator/era"
	"gitlab.waterfall.network/waterfall/protocol/gwat/validator/testmodels"
)

func TestConsensus_breakByValidatorsBySlotCount(t *testing.T) {
	tests := []struct {
		name              string
		validatorsPerSlot uint64
		want              [][]common.Address
	}{
		{
			name:              "5 validators per slot",
			validatorsPerSlot: 5,
			want: [][]common.Address{
				{testmodels.Addr1, testmodels.Addr2, testmodels.Addr3, testmodels.Addr4, testmodels.Addr5},
				{testmodels.Addr6, testmodels.Addr7, testmodels.Addr8, testmodels.Addr9, testmodels.Addr10},
			},
		},
		{
			name:              "2 validators per slot",
			validatorsPerSlot: 2,
			want: [][]common.Address{
				{testmodels.Addr1, testmodels.Addr2},
				{testmodels.Addr3, testmodels.Addr4},
				{testmodels.Addr5, testmodels.Addr6},
				{testmodels.Addr7, testmodels.Addr8},
				{testmodels.Addr9, testmodels.Addr10},
				{testmodels.Addr11, testmodels.Addr12},
			},
		},
		{
			name:              "1 validator per slot",
			validatorsPerSlot: 1,
			want: [][]common.Address{
				{testmodels.Addr1},
				{testmodels.Addr2},
				{testmodels.Addr3},
				{testmodels.Addr4},
				{testmodels.Addr5},
				{testmodels.Addr6},
				{testmodels.Addr7},
				{testmodels.Addr8},
				{testmodels.Addr9},
				{testmodels.Addr10},
				{testmodels.Addr11},
				{testmodels.Addr12},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			validators := breakByValidatorsBySlotCount(testmodels.InputValidators, test.validatorsPerSlot, testmodels.TestChainConfig.SlotsPerEpoch)
			testutils.AssertEqual(t, test.want, validators)
		})
	}
}

func TestGetValidators(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	bc := NewMockblockchain(ctrl)

	validatorsList := make([]*Validator, 0)
	blockHash := common.HexToHash("0x1234")
	block := types.NewBlock(&types.Header{Slot: slot}, nil, nil, nil)

	stateDb, _ := state.New(common.Hash{}, state.NewDatabase(testmodels.TestDb), nil)

	// Create a new storage object
	store := NewStorage(testmodels.TestChainConfig)
	store.SetValidatorsList(stateDb, testmodels.InputValidators)

	bc.EXPECT().GetSlotInfo().AnyTimes().Return(&types.SlotInfo{
		GenesisTime:    uint64(time.Now().Unix()),
		SecondsPerSlot: testmodels.TestChainConfig.SecondsPerSlot,
		SlotsPerEpoch:  testmodels.TestChainConfig.SlotsPerEpoch,
	})
	bc.EXPECT().EpochToEra(gomock.AssignableToTypeOf(uint64(0))).AnyTimes().Return(&era.Era{
		Number: 0,
		From:   0,
		To:     0,
		Root:   common.Hash{},
	})

	bc.EXPECT().GetBlock(
		gomock.AssignableToTypeOf(context.Background()),
		gomock.AssignableToTypeOf(blockHash),
	).AnyTimes().Return(block)
	bc.EXPECT().StateAt(gomock.AssignableToTypeOf(blockHash)).AnyTimes().Return(stateDb, nil)
	bc.EXPECT().GetLastCoordinatedCheckpoint().AnyTimes().Return(&types.Checkpoint{
		Epoch: uint64(testutils.RandomInt(0, 99999)),
		Root:  common.BytesToHash(testutils.RandomData(32)),
		Spine: common.BytesToHash(testutils.RandomData(32)),
	})

	for i, address := range testmodels.InputValidators {
		validator := &Validator{
			Address:       address,
			Index:         uint64(i),
			ActivationEra: 0,
			ExitEra:       math.MaxUint64,
		}
		validatorsList = append(validatorsList, validator)
		err := store.SetValidator(stateDb, validator)
		testutils.AssertNoError(t, err)
	}

	tests := []struct {
		name          string
		slot          uint64
		wantAddresses []common.Address
	}{
		{
			name:          "correct test",
			slot:          uint64(testutils.RandomInt(0, len(testmodels.InputValidators)*int(testmodels.TestChainConfig.SlotsPerEpoch))),
			wantAddresses: testmodels.InputValidators,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			addresses, err := store.GetValidators(bc, test.slot, "Tests")
			testutils.AssertNoError(t, err)
			testutils.AssertEqual(t, test.wantAddresses, addresses)
		})
	}
}

func TestGetShuffledValidators(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	blockHash := common.HexToHash("0x1234")
	block := types.NewBlock(&types.Header{Slot: slot}, nil, nil, nil)
	slot := uint64(200)

	stateDb, err := state.New(common.Hash{}, state.NewDatabase(testmodels.TestDb), nil)
	testutils.AssertNoError(t, err)

	bc := NewMockblockchain(ctrl)
	bc.EXPECT().GetSlotInfo().AnyTimes().Return(&types.SlotInfo{
		GenesisTime:    uint64(time.Now().Unix()),
		SecondsPerSlot: testmodels.TestChainConfig.SecondsPerSlot,
		SlotsPerEpoch:  testmodels.TestChainConfig.SlotsPerEpoch,
	})
	bc.EXPECT().GetBlock(gomock.AssignableToTypeOf(context.Background()), gomock.AssignableToTypeOf(blockHash)).AnyTimes().Return(block)
	bc.EXPECT().StateAt(gomock.AssignableToTypeOf(blockHash)).AnyTimes().Return(stateDb, nil)
	bc.EXPECT().GetLastCoordinatedCheckpoint().AnyTimes().Return(&types.Checkpoint{
		Epoch: 10,
		Root:  common.HexToHash("0xe46fb9c7774e3189b822353c521183f637560dfa199695ed5157d49f989d0c52"),
		Spine: common.HexToHash("0x5e44e252e7b239ea389a3cb95b112ffccd349852dcfd5b4c5e8f7857f1e730e5"),
	})
	bc.EXPECT().EpochToEra(gomock.AssignableToTypeOf(uint64(0))).AnyTimes().Return(&era.Era{
		Number: 10,
		From:   0,
		To:     0,
		Root:   common.Hash{},
	})
	bc.EXPECT().GetEpoch(gomock.AssignableToTypeOf(uint64(0))).AnyTimes().Return(common.Hash{0x11})

	store := NewStorage(testmodels.TestChainConfig)
	store.SetValidatorsList(stateDb, testmodels.InputValidators)

	for i, address := range testmodels.InputValidators {
		validator := &Validator{
			Address:       address,
			Index:         uint64(i),
			ActivationEra: 0,
			ExitEra:       math.MaxUint64,
		}
		err := store.SetValidator(stateDb, validator)
		testutils.AssertNoError(t, err)
	}

	// Test case 1: Invalid filter error
	result, err := store.GetCreatorsBySlot(bc, slot, epoch, slot)
	testutils.AssertError(t, err, ErrInvalidValidatorsFilter)
	testutils.AssertNil(t, result)

	// Test case 2: Validators available in cache
	result, err = store.GetCreatorsBySlot(bc, slot)
	testutils.AssertNoError(t, err)
	testutils.AssertEqual(t, []common.Address{
		testmodels.Addr5,
		testmodels.Addr4}, result)
}

func BenchmarkPrepareNextEraValidators(b *testing.B) {
	validators := make([]common.Address, 1000000)

	stateDb, err := state.New(common.Hash{}, state.NewDatabase(testmodels.TestDb), nil)
	if err != nil {
		b.Fatal(err)
	}

	store := NewStorage(testmodels.TestChainConfig)
	for i := range validators {
		addr := common.HexToAddress(fmt.Sprintf("0x%x", i))
		validators[i] = addr
		err = store.SetValidator(stateDb, &Validator{
			Address:       addr,
			Index:         uint64(i),
			ActivationEra: 0,
			ExitEra:       math.MaxUint64,
		})
		if err != nil {
			b.Fatal(err)
		}
	}
	store.SetValidatorsList(stateDb, validators)

	ctrl := gomock.NewController(b)
	defer ctrl.Finish()

	bc := NewMockblockchain(ctrl)

	bc.EXPECT().GetSlotInfo().AnyTimes().Return(&types.SlotInfo{
		GenesisTime:    uint64(time.Now().Unix()),
		SecondsPerSlot: testmodels.TestChainConfig.SecondsPerSlot,
		SlotsPerEpoch:  testmodels.TestChainConfig.SlotsPerEpoch,
	})
	bc.EXPECT().EpochToEra(gomock.AssignableToTypeOf(uint64(0))).AnyTimes().Return(&era.Era{
		Number: 0,
		From:   0,
		To:     0,
		Root:   common.Hash{},
	})
	bc.EXPECT().StateAt(gomock.AssignableToTypeOf(common.Hash{})).AnyTimes().Return(stateDb, nil)

	sizes := []int{1000, 10000, 100000, 500000, 1000000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size_%d", size), func(b *testing.B) {
			testValidators := validators[:size]
			store.SetValidatorsList(stateDb, testValidators)

			era := &era.Era{
				Number:    0,
				From:      0,
				To:        0,
				Root:      common.Hash{},
				BlockHash: common.Hash{},
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				store.PrepareNextEraValidators(bc, era)
			}
		})
	}
}
