package storage

import (
	"math"
	"testing"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/state"
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
	validatorsList := make([]Validator, 0)

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
		epoch          int
		activeOnly     bool
		needAddresses  bool
		wantValidators []Validator
		wantAddresses  []common.Address
	}{
		{
			name:           "activeOnly and needAddresses are both false",
			epoch:          testutils.RandomInt(0, len(testmodels.InputValidators)),
			activeOnly:     false,
			needAddresses:  false,
			wantValidators: validatorsList,
			wantAddresses:  nil,
		},
		{
			name:           "activeOnly is false and needAddresses is true",
			epoch:          testutils.RandomInt(0, len(testmodels.InputValidators)),
			activeOnly:     false,
			needAddresses:  true,
			wantValidators: validatorsList,
			wantAddresses:  testmodels.InputValidators,
		},
		{
			name:           "activeOnly is true and needAddresses is false",
			epoch:          testutils.RandomInt(0, len(testmodels.InputValidators)),
			activeOnly:     true,
			needAddresses:  false,
			wantValidators: make([]Validator, 0),
			wantAddresses:  nil,
		},
		{
			name:           "activeOnly and needAddresses are both true",
			epoch:          testutils.RandomInt(0, len(testmodels.InputValidators)),
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
					if validator.ActivationEpoch <= uint64(test.epoch) && validator.ExitEpoch > uint64(test.epoch) {
						test.wantValidators = append(test.wantValidators, validator)
					}
				}

				if test.needAddresses {
					for _, val := range test.wantValidators {
						test.wantAddresses = append(test.wantAddresses, val.Address)
					}
				}
			}

			validators, addresses := store.GetValidators(stateDb, uint64(test.epoch), test.activeOnly, test.needAddresses)
			testutils.AssertEqual(t, test.wantValidators, validators)
			testutils.AssertEqual(t, test.wantAddresses, addresses)

		})
	}
}

func TestGetShuffledValidators(t *testing.T) {
	blockHash := common.HexToHash("0x1234")
	epoch := uint64(len(testmodels.InputValidators))
	slot := uint64(0)

	stateDb, err := state.New(common.Hash{}, state.NewDatabase(testmodels.TestDb), nil)
	testutils.AssertNoError(t, err)

	store := NewStorage(testmodels.TestDb, testmodels.TestChainConfig)

	store.SetValidatorsList(stateDb, testmodels.InputValidators)
	for i, address := range testmodels.InputValidators {
		validator := NewValidator(address, &common.Address{0x0000000000000000000000000000000000000000}, uint64(i), uint64(i), uint64(math.MaxUint64), nil)
		info, err := validator.MarshalBinary()
		testutils.AssertNoError(t, err)

		store.SetValidatorInfo(stateDb, info)
	}

	// Test case 1: Invalid filter error
	filter := []uint64{epoch, slot}
	result, err := store.GetShuffledValidators(stateDb, blockHash, filter[0:1]...)
	testutils.AssertError(t, err, ErrInvalidValidatorsFilter)
	testutils.AssertNil(t, result)

	// Test case 2: Validators available in cache
	result, err = store.GetShuffledValidators(stateDb, blockHash, filter...)
	testutils.AssertNoError(t, err)
	testutils.AssertEqual(t, []common.Address{
		testmodels.Addr6,
		testmodels.Addr12,
		testmodels.Addr7,
		testmodels.Addr10,
		testmodels.Addr4,
		testmodels.Addr3}, result)
}
