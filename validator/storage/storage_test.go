package storage

import (
	"math"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/state"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/types"
	"gitlab.waterfall.network/waterfall/protocol/gwat/tests/testutils"
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

	store := NewStorage(testmodels.TestDb, testmodels.TestChainConfig)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			validators := store.breakByValidatorsBySlotCount(testmodels.InputValidators, test.validatorsPerSlot)
			testutils.AssertEqual(t, test.want, validators)
		})
	}
}

func TestGetValidators(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	bc := NewMockblockchain(ctrl)

	validatorsList := make([]Validator, 0)
	blockHash := common.HexToHash("0x1234")
	block := types.NewBlock(&types.Header{Slot: slot}, nil, nil, nil)

	stateDb, _ := state.New(common.Hash{}, state.NewDatabase(testmodels.TestDb), nil)

	// Create a new storage object
	store := NewStorage(testmodels.TestDb, testmodels.TestChainConfig)
	store.SetValidatorsList(stateDb, testmodels.InputValidators)

	for i, address := range testmodels.InputValidators {
		validator := NewValidator(address, &common.Address{0x0000000000000000000000000000000000000000}, uint64(i), uint64(i), uint64(math.MaxUint64), nil)
		info, err := validator.MarshalBinary()
		testutils.AssertNoError(t, err)

		validatorsList = append(validatorsList, *validator)
		store.SetValidatorInfo(stateDb, info)
	}

	tests := []struct {
		name           string
		slot           uint64
		activeOnly     bool
		needAddresses  bool
		wantValidators []Validator
		wantAddresses  []common.Address
	}{
		{
			name:           "activeOnly and needAddresses are both false",
			slot:           uint64(testutils.RandomInt(0, len(testmodels.InputValidators)*int(testmodels.TestChainConfig.SlotsPerEpoch))),
			activeOnly:     false,
			needAddresses:  false,
			wantValidators: validatorsList,
			wantAddresses:  nil,
		},
		{
			name:           "activeOnly is false and needAddresses is true",
			slot:           uint64(testutils.RandomInt(0, len(testmodels.InputValidators)*int(testmodels.TestChainConfig.SlotsPerEpoch))),
			activeOnly:     false,
			needAddresses:  true,
			wantValidators: validatorsList,
			wantAddresses:  testmodels.InputValidators,
		},
		{
			name:           "activeOnly is true and needAddresses is false",
			slot:           uint64(testutils.RandomInt(0, len(testmodels.InputValidators)*int(testmodels.TestChainConfig.SlotsPerEpoch))),
			activeOnly:     true,
			needAddresses:  false,
			wantValidators: make([]Validator, 0),
			wantAddresses:  nil,
		},
		{
			name:           "activeOnly and needAddresses are both true",
			slot:           uint64(testutils.RandomInt(0, len(testmodels.InputValidators)*int(testmodels.TestChainConfig.SlotsPerEpoch))),
			activeOnly:     true,
			needAddresses:  true,
			wantValidators: make([]Validator, 0),
			wantAddresses:  make([]common.Address, 0),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.activeOnly {
				for _, validator := range validatorsList {
					if validator.ActivationEpoch <= bc.GetSlotInfo().SlotToEpoch(test.slot) && validator.ExitEpoch > bc.GetSlotInfo().SlotToEpoch(test.slot) {
						test.wantValidators = append(test.wantValidators, validator)
					}
				}

				if test.needAddresses {
					for _, val := range test.wantValidators {
						test.wantAddresses = append(test.wantAddresses, val.Address)
					}
				}
			}

			bc.EXPECT().GetSlotInfo().AnyTimes().Return(&types.SlotInfo{
				GenesisTime:    uint64(time.Now().Unix()),
				SecondsPerSlot: testmodels.TestChainConfig.SecondsPerSlot,
				SlotsPerEpoch:  testmodels.TestChainConfig.SlotsPerEpoch,
			})
			bc.EXPECT().SearchFirstEpochBlockHashRecursive(gomock.AssignableToTypeOf(test.slot)).AnyTimes().Return(blockHash, true)
			bc.EXPECT().GetBlock(gomock.AssignableToTypeOf(blockHash)).AnyTimes().Return(block)
			bc.EXPECT().StateAt(gomock.AssignableToTypeOf(blockHash)).AnyTimes().Return(stateDb, nil)
			bc.EXPECT().GetCoordinatedCheckpointEpoch(gomock.AssignableToTypeOf(test.slot)).AnyTimes().Return(epoch)

			validators, addresses := store.GetValidators(bc, test.slot, test.activeOnly, test.needAddresses)
			testutils.AssertEqual(t, test.wantValidators, validators)
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
	bc.EXPECT().SearchFirstEpochBlockHashRecursive(gomock.AssignableToTypeOf(slot)).AnyTimes().Return(blockHash, true)
	bc.EXPECT().GetBlock(gomock.AssignableToTypeOf(blockHash)).AnyTimes().Return(block)
	bc.EXPECT().StateAt(gomock.AssignableToTypeOf(blockHash)).AnyTimes().Return(stateDb, nil)
	bc.EXPECT().GetCoordinatedCheckpointEpoch(gomock.AssignableToTypeOf(slot)).AnyTimes().Return(epoch)

	store := NewStorage(testmodels.TestDb, testmodels.TestChainConfig)
	store.SetValidatorsList(stateDb, testmodels.InputValidators)

	validatorList := make([]Validator, len(testmodels.InputValidators))
	for i, address := range testmodels.InputValidators {
		validator := NewValidator(address, &common.Address{0x0000000000000000000000000000000000000000}, uint64(i), uint64(i), uint64(math.MaxUint64), nil)
		info, err := validator.MarshalBinary()
		testutils.AssertNoError(t, err)
		validatorList[i] = *validator
		store.SetValidatorInfo(stateDb, info)
	}

	// Test case 1: Invalid filter error
	result, err := store.GetCreatorsBySlot(bc, slot, epoch, slot)
	testutils.AssertError(t, err, ErrInvalidValidatorsFilter)
	testutils.AssertNil(t, result)

	// Test case 2: Validators available in cache
	result, err = store.GetCreatorsBySlot(bc, slot)
	testutils.AssertNoError(t, err)
	testutils.AssertEqual(t, []common.Address{
		testmodels.Addr1,
		testmodels.Addr3,
		testmodels.Addr2,
		testmodels.Addr4,
		testmodels.Addr5,
		testmodels.Addr6}, result)
}
