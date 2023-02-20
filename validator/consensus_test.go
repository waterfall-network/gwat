package validator

import (
	"math/big"
	"testing"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/rawdb"
	"gitlab.waterfall.network/waterfall/protocol/gwat/params"
	"gitlab.waterfall.network/waterfall/protocol/gwat/tests/testutils"
	"gitlab.waterfall.network/waterfall/protocol/gwat/validator/testmodels"
)

func TestConsensus_breakByValidatorsBySlotCount(t *testing.T) {
	db := rawdb.NewMemoryDatabase()

	config := &params.ChainConfig{
		ChainID:                big.NewInt(111111),
		SecondsPerSlot:         4,
		SlotsPerEpoch:          32,
		ForkSlotSubNet1:        9999999,
		ValidatorsStateAddress: nil,
	}

	consensus := consensus{
		db:     db,
		config: config,
	}

	tests := []struct {
		name              string
		validatorsPerSlot int
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
			validators := consensus.breakByValidatorsBySlotCount(testmodels.InputValidators, test.validatorsPerSlot)
			testutils.AssertEqual(t, test.want, validators)

		})
	}
}
