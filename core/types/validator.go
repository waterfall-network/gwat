package types

import (
	"encoding/binary"
	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"math/big"
)

const (
	Uint64Size              = 8
	BigIntSize              = 32
	WithdrawalAddressOffset = common.AddressLength
	ValidatorIndexOffset    = WithdrawalAddressOffset + common.AddressLength
	ActivationEpochOffset   = ValidatorIndexOffset + Uint64Size
	ExitEpochOffset         = ActivationEpochOffset + Uint64Size
	BalanceOffset           = ExitEpochOffset + Uint64Size
	MetricOffset            = BalanceOffset + BigIntSize
)

type Validator struct {
	Address           common.Address
	WithdrawalAddress *common.Address
	ValidatorIndex    uint64
	ActivationEpoch   uint64
	ExitEpoch         uint64
	Balance           *big.Int
}

func NewValidator(address common.Address, withdrawal *common.Address, balance *big.Int, validatorIndex, activationEpoch, exitEpoch uint64) *Validator {
	if balance == nil {
		balance = new(big.Int)
	}

	return &Validator{
		Address:           address,
		WithdrawalAddress: withdrawal,
		ValidatorIndex:    validatorIndex,
		ActivationEpoch:   activationEpoch,
		ExitEpoch:         exitEpoch,
		Balance:           balance,
	}
}

func (v *Validator) MarshalBinary() ([]byte, error) {
	address := make([]byte, common.AddressLength)
	withdrawalAddress := make([]byte, common.AddressLength)
	copy(address, v.Address[:])

	if v.WithdrawalAddress != nil {
		copy(withdrawalAddress, v.WithdrawalAddress[:])
	}

	balance := v.Balance.Bytes()

	data := make([]byte, common.AddressLength*2+Uint64Size*3+len(balance))

	copy(data[:common.AddressLength], address)

	copy(data[WithdrawalAddressOffset:WithdrawalAddressOffset+common.AddressLength], withdrawalAddress)

	binary.BigEndian.PutUint64(data[ValidatorIndexOffset:ValidatorIndexOffset+Uint64Size], v.ValidatorIndex)

	binary.BigEndian.PutUint64(data[ActivationEpochOffset:ActivationEpochOffset+Uint64Size], v.ActivationEpoch)

	binary.BigEndian.PutUint64(data[ExitEpochOffset:ExitEpochOffset+Uint64Size], v.ExitEpoch)

	copy(data[BalanceOffset:BalanceOffset+len(balance)], balance)
	return data, nil
}

func (v *Validator) UnmarshalBinary(data []byte) error {
	v.Address = common.Address{}
	copy(v.Address[:], data[:common.AddressLength])

	v.WithdrawalAddress = new(common.Address)
	copy(v.WithdrawalAddress[:], data[WithdrawalAddressOffset:WithdrawalAddressOffset+common.AddressLength])

	v.ValidatorIndex = binary.BigEndian.Uint64(data[ValidatorIndexOffset : ValidatorIndexOffset+Uint64Size])

	v.ActivationEpoch = binary.BigEndian.Uint64(data[ActivationEpochOffset : ActivationEpochOffset+Uint64Size])

	v.ExitEpoch = binary.BigEndian.Uint64(data[ExitEpochOffset : ExitEpochOffset+Uint64Size])

	v.Balance = new(big.Int).SetBytes(data[BalanceOffset:])
	return nil
}

type ValidatorInfo []byte

func (vi ValidatorInfo) GetAddress() common.Address {
	return common.BytesToAddress(vi[:common.AddressLength])
}

func (vi ValidatorInfo) SetAddress(address common.Address) {
	copy(vi[:common.AddressLength], address[:])
}

func (vi ValidatorInfo) GetWithdrawalAddress() common.Address {
	return common.BytesToAddress(vi[WithdrawalAddressOffset:ValidatorIndexOffset])
}

func (vi ValidatorInfo) SetWithdrawalAddress(address common.Address) {
	copy(vi[WithdrawalAddressOffset:ValidatorIndexOffset], address[:])
}

func (vi ValidatorInfo) GetValidatorIndex() uint64 {
	return binary.BigEndian.Uint64(vi[ValidatorIndexOffset:ActivationEpochOffset])
}

func (vi ValidatorInfo) SetValidatorIndex(index uint64) {
	binary.BigEndian.PutUint64(vi[ValidatorIndexOffset:ActivationEpochOffset], index)
}

func (vi ValidatorInfo) GetActivationEpoch() uint64 {
	return binary.BigEndian.Uint64(vi[ActivationEpochOffset:ExitEpochOffset])
}

func (vi ValidatorInfo) SetActivationEpoch(epoch uint64) {
	binary.BigEndian.PutUint64(vi[ActivationEpochOffset:ExitEpochOffset], epoch)
}

func (vi ValidatorInfo) GetExitEpoch() uint64 {
	return binary.BigEndian.Uint64(vi[ExitEpochOffset:BalanceOffset])
}

func (vi ValidatorInfo) SetExitEpoch(epoch uint64) {
	binary.BigEndian.PutUint64(vi[ExitEpochOffset:BalanceOffset], epoch)
}

func (vi ValidatorInfo) GetValidatorBalance() uint64 {
	return binary.BigEndian.Uint64(vi[BalanceOffset:MetricOffset])
}

func (vi ValidatorInfo) SetValidatorBalance(balance *big.Int) {
	copy(vi[BalanceOffset:MetricOffset], balance.Bytes())
}
