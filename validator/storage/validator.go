package storage

import (
	"encoding/binary"
	"math"
	"math/big"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
)

const (
	uint64Size              = 8
	withdrawalAddressOffset = common.AddressLength
	validatorIndexOffset    = withdrawalAddressOffset + common.AddressLength
	activationEpochOffset   = validatorIndexOffset + uint64Size
	exitEpochOffset         = activationEpochOffset + uint64Size
	balanceLengthOffset     = exitEpochOffset + uint64Size
	balanceOffset           = balanceLengthOffset + uint64Size
	metricOffset            = balanceOffset // TODO: add balance length to calculate offset
)

type Validator struct {
	Address           common.Address
	WithdrawalAddress *common.Address
	Index             uint64
	ActivationEpoch   uint64
	ExitEpoch         uint64
	Balance           *big.Int
}

func NewValidator(address common.Address, withdrawal *common.Address) *Validator {
	return &Validator{
		Address:           address,
		WithdrawalAddress: withdrawal,
		Index:             math.MaxUint64,
		ActivationEpoch:   math.MaxUint64,
		ExitEpoch:         math.MaxUint64,
		Balance:           new(big.Int),
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

	data := make([]byte, common.AddressLength*2+uint64Size*4+len(balance))

	copy(data[:common.AddressLength], address)

	copy(data[withdrawalAddressOffset:withdrawalAddressOffset+common.AddressLength], withdrawalAddress)

	binary.BigEndian.PutUint64(data[validatorIndexOffset:validatorIndexOffset+uint64Size], v.Index)

	binary.BigEndian.PutUint64(data[activationEpochOffset:activationEpochOffset+uint64Size], v.ActivationEpoch)

	binary.BigEndian.PutUint64(data[exitEpochOffset:exitEpochOffset+uint64Size], v.ExitEpoch)

	binary.BigEndian.PutUint64(data[balanceLengthOffset:balanceOffset], uint64(len(balance)))

	copy(data[balanceOffset:balanceOffset+len(balance)], balance)
	return data, nil
}

func (v *Validator) UnmarshalBinary(data []byte) error {
	v.Address = common.Address{}
	copy(v.Address[:], data[:common.AddressLength])

	v.WithdrawalAddress = new(common.Address)
	copy(v.WithdrawalAddress[:], data[withdrawalAddressOffset:withdrawalAddressOffset+common.AddressLength])

	v.Index = binary.BigEndian.Uint64(data[validatorIndexOffset : validatorIndexOffset+uint64Size])

	v.ActivationEpoch = binary.BigEndian.Uint64(data[activationEpochOffset : activationEpochOffset+uint64Size])

	v.ExitEpoch = binary.BigEndian.Uint64(data[exitEpochOffset : exitEpochOffset+uint64Size])

	balanceLen := binary.BigEndian.Uint64(data[balanceLengthOffset:balanceOffset])

	v.Balance = new(big.Int).SetBytes(data[balanceOffset : balanceOffset+balanceLen])
	return nil
}

// ValidatorInfo is a Validator represented as an array of bytes.
type ValidatorInfo []byte

func (vi ValidatorInfo) GetAddress() common.Address {
	return common.BytesToAddress(vi[:common.AddressLength])
}

func (vi ValidatorInfo) SetAddress(address common.Address) {
	copy(vi[:common.AddressLength], address[:])
}

func (vi ValidatorInfo) GetWithdrawalAddress() common.Address {
	return common.BytesToAddress(vi[withdrawalAddressOffset:validatorIndexOffset])
}

func (vi ValidatorInfo) SetWithdrawalAddress(address common.Address) {
	copy(vi[withdrawalAddressOffset:validatorIndexOffset], address[:])
}

func (vi ValidatorInfo) GetValidatorIndex() uint64 {
	return binary.BigEndian.Uint64(vi[validatorIndexOffset:activationEpochOffset])
}

func (vi ValidatorInfo) SetValidatorIndex(index uint64) {
	binary.BigEndian.PutUint64(vi[validatorIndexOffset:activationEpochOffset], index)
}

func (vi ValidatorInfo) GetActivationEpoch() uint64 {
	return binary.BigEndian.Uint64(vi[activationEpochOffset:exitEpochOffset])
}

func (vi ValidatorInfo) SetActivationEpoch(epoch uint64) {
	binary.BigEndian.PutUint64(vi[activationEpochOffset:exitEpochOffset], epoch)
}

func (vi ValidatorInfo) GetExitEpoch() uint64 {
	return binary.BigEndian.Uint64(vi[exitEpochOffset:balanceLengthOffset])
}

func (vi ValidatorInfo) SetExitEpoch(epoch uint64) {
	binary.BigEndian.PutUint64(vi[exitEpochOffset:balanceLengthOffset], epoch)
}

func (vi ValidatorInfo) GetValidatorBalance() *big.Int {
	balanceLength := binary.BigEndian.Uint64(vi[balanceLengthOffset:balanceOffset])

	bal := vi[balanceOffset : balanceOffset+balanceLength]

	return new(big.Int).SetBytes(bal)
}

func (vi ValidatorInfo) SetValidatorBalance(balance *big.Int) {
	newLen := len(balance.Bytes())

	binary.BigEndian.PutUint64(vi[balanceLengthOffset:balanceOffset], uint64(newLen))

	copy(vi[balanceOffset:balanceOffset+newLen], balance.Bytes())
}
