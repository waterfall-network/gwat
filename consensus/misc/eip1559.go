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
	expectedBaseFee := CalcSlotBaseFee(config, creatorsPerSlot, validatorsNum, maxGasPerBlock, header.Slot)

	if header.BaseFee.Cmp(expectedBaseFee) != 0 {
		return fmt.Errorf("invalid baseFee: have %s, want %s, parentBaseFee %s, parentGasUsed %d",
			expectedBaseFee, header.BaseFee, parent.BaseFee, parent.GasUsed)
	}
	return nil
}

// CalcSlotBaseFee calculates the base fee of the slot.
func CalcSlotBaseFee(config *params.ChainConfig, creatorsPerSlotCount, validatorsCount, maxGasAmountPerBlock, slot uint64) *big.Int {
	var (
		effectiveBalanceInWei = new(big.Float).Mul(new(big.Float).SetInt(config.EffectiveBalance), big.NewFloat(1e9))
		annualizedRateBig     = new(big.Float).SetFloat64(params.MaxAnnualizedReturnRate)
		optValidatorsNumBig   = new(big.Float).SetUint64(params.OptValidatorsNum)
		validatorsAmountBig   = new(big.Float).SetUint64(validatorsCount)
		stakeAmount           = new(big.Float).Mul(new(big.Float).SetUint64(validatorsCount), effectiveBalanceInWei)
		secondsInYear         = new(big.Float).SetUint64(60 * 60 * 24 * 365.25)
		slotTimeBig           = new(big.Float).SetUint64(config.SecondsPerSlot)
		blocksPerSlotBig      = new(big.Float).SetUint64(creatorsPerSlotCount)
	)

	if !config.IsForkSlotReduceBaseFee(slot) {
		optValidatorsNumBig = new(big.Float).SetUint64(params.OptValidatorsNumBeforeReduceBaseFee)
	}

	annualizedReturnRate := new(big.Float).Mul(annualizedRateBig, new(big.Float).Sqrt(new(big.Float).Quo(optValidatorsNumBig, validatorsAmountBig)))
	annualizedMintedAmount := new(big.Float).Mul(stakeAmount, annualizedReturnRate)
	dagBlocksPerYear := new(big.Float).Mul(new(big.Float).Quo(secondsInYear, slotTimeBig), blocksPerSlotBig)
	feeForGas := new(big.Float).Mul(new(big.Float).Quo(new(big.Float).Mul(new(big.Float).Quo(annualizedMintedAmount, dagBlocksPerYear), big.NewFloat(params.PriceMultiplier)), new(big.Float).SetUint64(maxGasAmountPerBlock)), big.NewFloat(1e9))

	baseFee := new(big.Int)
	feeForGas.Int(baseFee)

	return baseFee
}

func CalcCreatorReward(gas uint64, baseFee *big.Int) *big.Int {
	fee := new(big.Int).Mul(new(big.Int).SetUint64(gas), baseFee)
	validatorPart := big.NewFloat(1 - params.BurnMultiplier)
	validatorReward := new(big.Int)
	new(big.Float).Mul(validatorPart, new(big.Float).SetInt(fee)).Int(validatorReward)

	return validatorReward
}
