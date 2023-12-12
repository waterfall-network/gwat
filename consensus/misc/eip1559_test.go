// Copyright 2021 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package misc

import (
	"fmt"
	"math"
	"math/big"
	"testing"

	"gitlab.waterfall.network/waterfall/protocol/gwat/core/types"
	"gitlab.waterfall.network/waterfall/protocol/gwat/params"
	"gitlab.waterfall.network/waterfall/protocol/gwat/tests/testutils"
)

// copyConfig does a _shallow_ copy of a given config. Safe to set new values, but
// do not use e.g. SetInt() on the numbers. For testing only
func copyConfig(original *params.ChainConfig) *params.ChainConfig {
	return &params.ChainConfig{
		ChainID:           original.ChainID,
		SecondsPerSlot:    4,
		SlotsPerEpoch:     32,
		ForkSlotSubNet1:   math.MaxUint64,
		ValidatorsPerSlot: 4,
		EffectiveBalance:  big.NewInt(3200),
	}
}

func config() *params.ChainConfig {
	config := copyConfig(params.TestChainConfig)
	return config
}

// TestBlockGasLimits tests the gasLimit checks for blocks both across
// the EIP-1559 boundary and post-1559 blocks
func TestBlockGasLimits(t *testing.T) {
	for i, tc := range []struct {
		baseFee   uint64
		pGasLimit uint64
		pNum      int64
		gasLimit  uint64
		ok        bool
	}{
		// Transitions from non-london to london
		{47689510544, 10000000, 4, 10000000, true}, // No change
		{47689510544, 10000000, 4, 10000000, true}, // Upper limit
		{1000000000, 10000000, 4, 20019531, false}, // Upper +1
		{47689510544, 10000000, 4, 10000000, true}, // Lower limit
		{1000000000, 10000000, 4, 19980469, false}, // Lower limit -1
		// London to London
		{47689510544, 20000000, 5, 20000000, true},
		{47689510544, 20000000, 5, 20019530, true}, // Upper limit
		{1000000000, 20000000, 5, 20019531, false}, // Upper limit +1
		{47689510544, 20000000, 5, 19980470, true}, // Lower limit
		{1000000000, 20000000, 5, 19980469, false}, // Lower limit -1
		{47689510544, 40000000, 5, 40039061, true}, // Upper limit
		{1000000000, 40000000, 5, 40039062, false}, // Upper limit +1
		{47689510544, 40000000, 5, 39960939, true}, // lower limit
		{1000000000, 40000000, 5, 39960938, false}, // Lower limit -1
	} {
		nrPt := uint64(tc.pNum)
		parent := &types.Header{
			GasUsed:  tc.pGasLimit / 2,
			GasLimit: tc.pGasLimit,
			BaseFee:  new(big.Int).SetUint64(tc.baseFee),
			Number:   &nrPt,
		}

		nrHd := uint64(tc.pNum + 1)
		header := &types.Header{
			GasUsed:  tc.gasLimit / 2,
			GasLimit: tc.gasLimit,
			BaseFee:  new(big.Int).SetUint64(tc.baseFee),
			Number:   &nrHd,
		}
		err := VerifyEip1559Header(config(), parent, header, 2048, 100000000, 4)
		if tc.ok && err != nil {
			t.Errorf("test %d: Expected valid header: %s", i, err)
		}
		if !tc.ok && err == nil {
			t.Errorf("test %d: Expected invalid header", i)
		}
	}
}

// TestCalcSlotBaseFee assumes all blocks are 1559-blocks
func TestCalcSlotBaseFee(t *testing.T) {
	conf := &params.ChainConfig{
		SecondsPerSlot:    4,
		ValidatorsPerSlot: 4,
		EffectiveBalance:  big.NewInt(3200),
	}

	maxGasAmountPerBlock := uint64(105000000)

	testCases := []struct {
		validatorsCount uint64
		expectedBaseFee *big.Int
	}{
		{
			validatorsCount: 8,
			expectedBaseFee: big.NewInt(2838661342),
		},
		{
			validatorsCount: 248,
			expectedBaseFee: big.NewInt(17031968050),
		},
		{
			validatorsCount: 2048,
			expectedBaseFee: big.NewInt(45418581470),
		},
		{
			validatorsCount: 5000,
			expectedBaseFee: big.NewInt(70966533550),
		},
		{
			validatorsCount: 10000,
			expectedBaseFee: big.NewInt(100361834200),
		},
		{
			validatorsCount: 20000,
			expectedBaseFee: big.NewInt(141933067100),
		},
		{
			validatorsCount: 50000,
			expectedBaseFee: big.NewInt(224415883700),
		},
		{
			validatorsCount: 100000,
			expectedBaseFee: big.NewInt(317371986300),
		},
		{
			validatorsCount: 200000,
			expectedBaseFee: big.NewInt(448831767300),
		},
		{
			validatorsCount: 500000,
			expectedBaseFee: big.NewInt(709665335500),
		},
		{
			validatorsCount: 1000000,
			expectedBaseFee: big.NewInt(1003618342000),
		},
		{
			validatorsCount: 3000000,
			expectedBaseFee: big.NewInt(1738317960000),
		},
		{
			validatorsCount: 5000000,
			expectedBaseFee: big.NewInt(2244158837000),
		},
	}

	for _, testCase := range testCases {
		t.Run(fmt.Sprintf("validators amount %d", testCase.validatorsCount), func(t *testing.T) {
			res := CalcSlotBaseFee(conf, 4, testCase.validatorsCount, maxGasAmountPerBlock)
			testutils.BigIntEquals(res, testCase.expectedBaseFee)
		})
	}
}

func TestCalcCreatorRewardForBaseTx(t *testing.T) {
	conf := &params.ChainConfig{
		SecondsPerSlot:    4,
		ValidatorsPerSlot: 4,
		EffectiveBalance:  big.NewInt(3200),
	}

	maxGasAmountPerBlock := uint64(105000000)

	testCases := []struct {
		validatorsCount uint64
		expectedReward  *big.Int
	}{
		{
			validatorsCount: 8,
			expectedReward:  big.NewInt(19870629390000),
		},
		{
			validatorsCount: 248,
			expectedReward:  big.NewInt(119223776400000),
		},
		{
			validatorsCount: 2048,
			expectedReward:  big.NewInt(317930070300000),
		},
		{
			validatorsCount: 5000,
			expectedReward:  big.NewInt(496765734800000),
		},
		{
			validatorsCount: 10000,
			expectedReward:  big.NewInt(702532839500000),
		},
		{
			validatorsCount: 20000,
			expectedReward:  big.NewInt(993531469700000),
		},
		{
			validatorsCount: 50000,
			expectedReward:  big.NewInt(1570911186000000),
		},
		{
			validatorsCount: 100000,
			expectedReward:  big.NewInt(2221603904000000),
		},
		{
			validatorsCount: 200000,
			expectedReward:  big.NewInt(3141822371000000),
		},
		{
			validatorsCount: 500000,
			expectedReward:  big.NewInt(4967657348000000),
		},
		{
			validatorsCount: 1000000,
			expectedReward:  big.NewInt(7025328395000000),
		},
		{
			validatorsCount: 3000000,
			expectedReward:  big.NewInt(12168225720000000),
		},
		{
			validatorsCount: 5000000,
			expectedReward:  big.NewInt(15709111860000000),
		},
	}

	for _, testCase := range testCases {
		t.Run(fmt.Sprintf("validators amount %d", testCase.validatorsCount), func(t *testing.T) {
			baseFee := CalcSlotBaseFee(conf, 4, testCase.validatorsCount, maxGasAmountPerBlock)
			reward := CalcCreatorReward(params.TxGas, baseFee)
			testutils.BigIntEquals(reward, testCase.expectedReward)
		})
	}
}
