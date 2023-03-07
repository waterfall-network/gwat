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
	"math"
	"math/big"
	"testing"

	"gitlab.waterfall.network/waterfall/protocol/gwat/core/types"
	"gitlab.waterfall.network/waterfall/protocol/gwat/params"
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
	}
}

func config() *params.ChainConfig {
	config := copyConfig(params.TestChainConfig)
	return config
}

// TestBlockGasLimits tests the gasLimit checks for blocks both across
// the EIP-1559 boundary and post-1559 blocks
func TestBlockGasLimits(t *testing.T) {
	initial := new(big.Int).SetUint64(params.InitialBaseFee)

	for i, tc := range []struct {
		pGasLimit uint64
		pNum      int64
		gasLimit  uint64
		ok        bool
	}{
		// Transitions from non-london to london
		{10000000, 4, 20000000, true},  // No change
		{10000000, 4, 20019530, true},  // Upper limit
		{10000000, 4, 20019531, false}, // Upper +1
		{10000000, 4, 19980470, true},  // Lower limit
		{10000000, 4, 19980469, false}, // Lower limit -1
		// London to London
		{20000000, 5, 20000000, true},
		{20000000, 5, 20019530, true},  // Upper limit
		{20000000, 5, 20019531, false}, // Upper limit +1
		{20000000, 5, 19980470, true},  // Lower limit
		{20000000, 5, 19980469, false}, // Lower limit -1
		{40000000, 5, 40039061, true},  // Upper limit
		{40000000, 5, 40039062, false}, // Upper limit +1
		{40000000, 5, 39960939, true},  // lower limit
		{40000000, 5, 39960938, false}, // Lower limit -1
	} {
		nrPt := uint64(tc.pNum)
		parent := &types.Header{
			GasUsed:  tc.pGasLimit / 2,
			GasLimit: tc.pGasLimit,
			BaseFee:  initial,
			Number:   &nrPt,
		}

		nrHd := uint64(tc.pNum + 1)
		header := &types.Header{
			GasUsed:  tc.gasLimit / 2,
			GasLimit: tc.gasLimit,
			BaseFee:  initial,
			Number:   &nrHd,
		}
		err := VerifyEip1559Header(config(), parent, header, 2048, 100000000)
		if tc.ok && err != nil {
			t.Errorf("test %d: Expected valid header: %s", i, err)
		}
		if !tc.ok && err == nil {
			t.Errorf("test %d: Expected invalid header", i)
		}
	}
}

// TestCalcBaseFee assumes all blocks are 1559-blocks
func TestCalcBaseFee(t *testing.T) {
	tests := []struct {
		parentBaseFee   int64
		parentGasLimit  uint64
		parentGasUsed   uint64
		expectedBaseFee int64
	}{
		{params.InitialBaseFee, 20000000, 10000000, params.InitialBaseFee}, // usage == target
		{params.InitialBaseFee, 20000000, 9000000, 987500000},              // usage below target
		{params.InitialBaseFee, 20000000, 11000000, 1012500000},            // usage above target
	}
	for i, test := range tests {
		parent := &types.Header{
			Number:   new(uint64),
			GasLimit: test.parentGasLimit,
			GasUsed:  test.parentGasUsed,
			BaseFee:  big.NewInt(test.parentBaseFee),
		}
		if have, want := CalcBaseFee(config(), parent), big.NewInt(test.expectedBaseFee); have.Cmp(want) != 0 {
			t.Errorf("test %d: have %d  want %d, ", i, have, want)
		}
	}
}

// TestCalcSlotBaseFee assumes all blocks are 1559-blocks
func TestCalcSlotBaseFee(t *testing.T) {
	tests := []struct {
		gasUsed         uint64
		validatorsNum   uint64
		maxGasPerBlock  uint64
		burnMultiplier  float64
		expectedBaseFee int64
	}{
		{10000, 2048, 210000000, 1, 478753},
		{10000, 4096, 210000000, 1, 677060},
		{10000, 32000, 210000000, 1, 1892440},
		{10000, 300000, 210000000, 1, 5794393},
		{2100000, 2048, 210000000, 1, 100538315},
		{2100000, 300000, 210000000, 1, 1216822572},
		{2100000, 300000, 210000000, 0.75, 912616929},
		{10000, 2048, 210000000, 1, 478753},
		{10000, 2048, 210000000, 0.25, 119688},
		{90000, 2048, 100000000, 1, 9048448},
		{110000, 2048, 100000000, 1, 11059214},
		{110000, 2048, 100000000, 0.25, 2764803},
		{110000, 2048, 100000000, 0.5, 5529607},
		{110000, 2048, 100000000, 0.75, 8294411},
	}
	for i, test := range tests {
		current := &types.Header{
			Number:  new(uint64),
			GasUsed: test.gasUsed,
		}
		if have, want := CalcSlotBaseFee(config(), current, test.validatorsNum, test.maxGasPerBlock, test.burnMultiplier), big.NewInt(test.expectedBaseFee); have.Cmp(want) != 0 {
			t.Errorf("test %d: have %d  want %d, ", i, have, want)
		}
	}
}
