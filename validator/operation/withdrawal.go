package operation

import (
	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"math/big"
)

type withdrawalRequestOperation struct {
	creatorAddress common.Address
	amount         *big.Int
}

func (op *withdrawalRequestOperation) init(
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
) (WithdrawalRequest, error) {
	op := &withdrawalRequestOperation{}
	if err := op.init(validatorAddress, amount); err != nil {
		return nil, err
	}

	return op, nil

}

func (op *withdrawalRequestOperation) MarshalBinary() ([]byte, error) {
	data := make([]byte, 0)

	data = append(data, op.creatorAddress.Bytes()...)

	data = append(data, op.amount.Bytes()...)

	return data, nil
}

func (op *withdrawalRequestOperation) UnmarshalBinary(data []byte) error {
	validatorAddress := common.BytesToAddress(data[:common.AddressLength])

	amount := new(big.Int).SetBytes(data[common.AddressLength:])

	return op.init(validatorAddress, amount)
}

func (op *withdrawalRequestOperation) OpCode() Code {
	return WithdrawalCode
}

func (op *withdrawalRequestOperation) CreatorAddress() common.Address {
	return op.creatorAddress
}

func (op *withdrawalRequestOperation) Amount() *big.Int {
	return op.amount
}
