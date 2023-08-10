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
	expectedBaseFee := CalcSlotBaseFee(config, validatorsNum, maxGasPerBlock, creatorsPerSlot)
	if header.BaseFee.Cmp(expectedBaseFee) != 0 {
		return fmt.Errorf("invalid baseFee: have %s, want %s, parentBaseFee %s, parentGasUsed %d",
			expectedBaseFee, header.BaseFee, parent.BaseFee, parent.GasUsed)
	}
	return nil
}

// CalcSlotBaseFee calculates the base fee of the slot.
func CalcSlotBaseFee(config *params.ChainConfig, validatorsNum uint64, maxGasPerBlock uint64, creatorsPerSlot uint64) *big.Int {
	var (
		gasUsedBig                = new(big.Float).SetUint64(params.TxGas)          // G normal transaction gas used
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
	fee := new(big.Int)
	baseFee.Int(fee)
	return fee
}
