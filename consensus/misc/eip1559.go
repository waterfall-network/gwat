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
	"math/big"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/common/math"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/types"
	"gitlab.waterfall.network/waterfall/protocol/gwat/params"
)

// VerifyEip1559Header verifies some header attributes which were changed in EIP-1559,
// - gas limit check
// - basefee check
func VerifyEip1559Header(config *params.ChainConfig, parent, header *types.Header, validatorsNum uint64, maxGasPerBlock uint64, creatorsPerSlot uint64) error {
	// Verify that the gas limit remains within allowed bounds
	parentGasLimit := parent.GasLimit
	if err := VerifyGaslimit(parentGasLimit, header.GasLimit); err != nil {
		return err
	}
	// Verify the header is not malformed
	if header.BaseFee == nil {
		return fmt.Errorf("header is missing baseFee")
	}
	// Verify the baseFee is correct based on the parent header.
	expectedBaseFee := CalcSlotBaseFee(config, header, validatorsNum, maxGasPerBlock, params.BurnMultiplier, creatorsPerSlot)
	if header.BaseFee.Cmp(expectedBaseFee) != 0 {
		return fmt.Errorf("invalid baseFee: have %s, want %s, parentBaseFee %s, parentGasUsed %d",
			expectedBaseFee, header.BaseFee, parent.BaseFee, parent.GasUsed)
	}
	return nil
}

// CalcBaseFee calculates the basefee of the header.
func CalcBaseFee(config *params.ChainConfig, parent *types.Header) *big.Int {
	var (
		parentGasTarget          = parent.GasLimit / params.ElasticityMultiplier
		parentGasTargetBig       = new(big.Int).SetUint64(parentGasTarget)
		baseFeeChangeDenominator = new(big.Int).SetUint64(params.BaseFeeChangeDenominator)
	)
	// If the parent gasUsed is the same as the target, the baseFee remains unchanged.
	if parent.GasUsed == parentGasTarget {
		return new(big.Int).Set(parent.BaseFee)
	}
	if parent.GasUsed > parentGasTarget {
		// If the parent block used more gas than its target, the baseFee should increase.
		gasUsedDelta := new(big.Int).SetUint64(parent.GasUsed - parentGasTarget)
		x := new(big.Int).Mul(parent.BaseFee, gasUsedDelta)
		y := x.Div(x, parentGasTargetBig)
		baseFeeDelta := math.BigMax(
			x.Div(y, baseFeeChangeDenominator),
			common.Big1,
		)

		return x.Add(parent.BaseFee, baseFeeDelta)
	} else {
		// Otherwise if the parent block used less gas than its target, the baseFee should decrease.
		gasUsedDelta := new(big.Int).SetUint64(parentGasTarget - parent.GasUsed)
		x := new(big.Int).Mul(parent.BaseFee, gasUsedDelta)
		y := x.Div(x, parentGasTargetBig)
		baseFeeDelta := x.Div(y, baseFeeChangeDenominator)

		return math.BigMax(
			x.Sub(parent.BaseFee, baseFeeDelta),
			common.Big0,
		)
	}
}

// CalcSlotBaseFee calculates the base fee of the slot.
func CalcSlotBaseFee(config *params.ChainConfig, current *types.Header, validatorsNum uint64, maxGasPerBlock uint64, burnFactor float64, creatorsPerSlot uint64) *big.Int {
	var (
		gasUsedBig                = new(big.Float).SetUint64(current.GasUsed)       // G normal transaction gas used
		blocksPerSlotBig          = new(big.Float).SetUint64(creatorsPerSlot)       // b i-th in the formula - eq the number of creators\blocks for i-th slot
		secondsPerSlotBig         = new(big.Float).SetUint64(config.SecondsPerSlot) // t i-th in the formula - the time of i-th slot
		secondsInYear             = new(big.Float).SetUint64(60 * 60 * 24 * 365.25)
		maxAnnualizedReturnRate   = new(big.Float).SetFloat64(params.MaxAnnualizedReturnRate)                             // R0 in the formula - the maximum annualized return rate with Nopt
		coordinatorStakeWei       = new(big.Float).Mul(new(big.Float).SetInt(config.EffectiveBalance), big.NewFloat(1e9)) // s in the formula - 1 coordinator stake
		optValidatorsNumBig       = new(big.Float).SetUint64(params.OptValidatorsNum)                                     // Nopt in the formula - eq 300000, the optimal number of validators
		validatorsNumBig          = new(big.Float).SetUint64(validatorsNum)                                               // N in the formula - the current number of validators
		totalAllowableGasPerBlock = new(big.Float).SetUint64(maxGasPerBlock)                                              // Gmax in the formula - the total allowable gas amount per block, eq genesis gas limit
	)

	numOfBlocksPerYear := new(big.Float).Quo(new(big.Float).Mul(secondsInYear, blocksPerSlotBig), secondsPerSlotBig)
	x := new(big.Float).Sqrt(new(big.Float).Mul(optValidatorsNumBig, validatorsNumBig))
	y := new(big.Float).Mul(maxAnnualizedReturnRate, coordinatorStakeWei)
	annualMintedCoins := new(big.Float).Mul(y, x)
	rewardPerBlock := new(big.Float).Quo(annualMintedCoins, numOfBlocksPerYear)
	baseFee := new(big.Float).Mul(new(big.Float).Mul(new(big.Float).Quo(gasUsedBig, totalAllowableGasPerBlock), rewardPerBlock), big.NewFloat(params.PriceMultiplier))
	baseFeeWithBurnFactor := new(big.Int)
	new(big.Float).Mul(baseFee, new(big.Float).SetFloat64(burnFactor)).Int(baseFeeWithBurnFactor)
	return baseFeeWithBurnFactor
}
