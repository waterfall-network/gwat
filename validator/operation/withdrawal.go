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

package operation

import (
	"math/big"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
)

type withdrawalOperation struct {
	creatorAddress common.Address
	amount         *big.Int
}

func (op *withdrawalOperation) init(
	creatorAddress common.Address,
	amount *big.Int,
) error {
	if creatorAddress == (common.Address{}) {
		return ErrNoCreatorAddress
	}

	if amount == nil {
		return ErrNoAmount
	}

	op.creatorAddress = creatorAddress
	op.amount = amount

	return nil
}

func NewWithdrawalOperation(
	validatorAddress common.Address,
	amount *big.Int,
) (Withdrawal, error) {
	op := &withdrawalOperation{}
	if err := op.init(validatorAddress, amount); err != nil {
		return nil, err
	}

	return op, nil
}

func (op *withdrawalOperation) MarshalBinary() ([]byte, error) {
	data := make([]byte, 0)

	data = append(data, op.creatorAddress.Bytes()...)

	data = append(data, op.amount.Bytes()...)

	return data, nil
}

func (op *withdrawalOperation) UnmarshalBinary(data []byte) error {
	validatorAddress := common.BytesToAddress(data[:common.AddressLength])

	amount := new(big.Int).SetBytes(data[common.AddressLength:])

	return op.init(validatorAddress, amount)
}

func (op *withdrawalOperation) OpCode() Code {
	return WithdrawalCode
}

func (op *withdrawalOperation) CreatorAddress() common.Address {
	return op.creatorAddress
}

func (op *withdrawalOperation) Amount() *big.Int {
	return op.amount
}
