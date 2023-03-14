package operation

import (
	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"math/big"
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
) (WithdrawalRequest, error) {
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
	return WithdrawalRequestCode
}

func (op *withdrawalOperation) CreatorAddress() common.Address {
	return op.creatorAddress
}

func (op *withdrawalOperation) Amount() *big.Int {
	return op.amount
}
