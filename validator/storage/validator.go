package storage

import (
	"encoding/binary"
	"math"
	"math/big"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
)

const (
	uint64Size              = 8
	creatorAddressOffset    = common.BlsPubKeyLength
	withdrawalAddressOffset = creatorAddressOffset + common.AddressLength
	validatorIndexOffset    = withdrawalAddressOffset + common.AddressLength
	activationEpochOffset   = validatorIndexOffset + uint64Size
	exitEpochOffset         = activationEpochOffset + uint64Size
	balanceLengthOffset     = exitEpochOffset + uint64Size
	balanceOffset           = balanceLengthOffset + uint64Size
	metricOffset            = balanceOffset // TODO: add balance length to calculate offset
)

type Validator struct {
	PubKey            common.BlsPubKey
	Address           common.Address
	WithdrawalAddress *common.Address
	Index             uint64
	ActivationEpoch   uint64
	ExitEpoch         uint64
	Balance           *big.Int
}

func NewValidator(pubKey common.BlsPubKey, address common.Address, withdrawal *common.Address) *Validator {
	return &Validator{
		PubKey:            pubKey,
		Address:           address,
		WithdrawalAddress: withdrawal,
		Index:             math.MaxUint64,
		ActivationEpoch:   math.MaxUint64,
		ExitEpoch:         math.MaxUint64,
		Balance:           new(big.Int),
	}
}

func (v *Validator) MarshalBinary() ([]byte, error) {
	pubKey := make([]byte, common.BlsPubKeyLength)
	copy(pubKey, v.PubKey[:])

	address := make([]byte, common.AddressLength)
	copy(address, v.Address[:])

	withdrawalAddress := make([]byte, common.AddressLength)
	if v.WithdrawalAddress != nil {
		copy(withdrawalAddress, v.WithdrawalAddress[:])
	}

	balance := v.Balance.Bytes()

	data := make([]byte, common.BlsPubKeyLength+common.AddressLength*2+uint64Size*4+len(balance))

	copy(data[:common.BlsPubKeyLength], pubKey)
	copy(data[creatorAddressOffset:creatorAddressOffset+common.AddressLength], address)
	copy(data[withdrawalAddressOffset:withdrawalAddressOffset+common.AddressLength], withdrawalAddress)

	binary.BigEndian.PutUint64(data[validatorIndexOffset:validatorIndexOffset+uint64Size], v.Index)

	binary.BigEndian.PutUint64(data[activationEpochOffset:activationEpochOffset+uint64Size], v.ActivationEpoch)

	binary.BigEndian.PutUint64(data[exitEpochOffset:exitEpochOffset+uint64Size], v.ExitEpoch)

	binary.BigEndian.PutUint64(data[balanceLengthOffset:balanceOffset], uint64(len(balance)))

	copy(data[balanceOffset:balanceOffset+len(balance)], balance)
	return data, nil
}

func (v *Validator) UnmarshalBinary(data []byte) error {
	v.PubKey = common.BlsPubKey{}
	copy(v.PubKey[:], data[:common.BlsPubKeyLength])

	v.Address = common.Address{}
	copy(v.Address[:], data[creatorAddressOffset:creatorAddressOffset+common.AddressLength])

	v.WithdrawalAddress = new(common.Address)
	copy(v.WithdrawalAddress[:], data[withdrawalAddressOffset:withdrawalAddressOffset+common.AddressLength])

	v.Index = binary.BigEndian.Uint64(data[validatorIndexOffset : validatorIndexOffset+uint64Size])

	v.ActivationEpoch = binary.BigEndian.Uint64(data[activationEpochOffset : activationEpochOffset+uint64Size])

	v.ExitEpoch = binary.BigEndian.Uint64(data[exitEpochOffset : exitEpochOffset+uint64Size])

	balanceLen := binary.BigEndian.Uint64(data[balanceLengthOffset:balanceOffset])

	v.Balance = new(big.Int).SetBytes(data[balanceOffset : balanceOffset+balanceLen])
	return nil
}

func (v *Validator) GetPubKey() common.BlsPubKey {
	return v.PubKey
}

func (v *Validator) SetPubKey(key common.BlsPubKey) {
	v.PubKey = key
}

func (v *Validator) GetAddress() common.Address {
	return v.Address
}

func (v *Validator) SetAddress(address common.Address) {
	v.Address = address
}

func (v *Validator) GetWithdrawalAddress() *common.Address {
	return v.WithdrawalAddress
}

func (v *Validator) SetWithdrawalAddress(address *common.Address) {
	v.WithdrawalAddress = address
}

func (v *Validator) GetIndex() uint64 {
	return v.Index
}

func (v *Validator) SetIndex(index uint64) {
	v.Index = index
}

func (v *Validator) GetActivationEpoch() uint64 {
	return v.ActivationEpoch
}

func (v *Validator) SetActivationEpoch(epoch uint64) {
	v.ActivationEpoch = epoch
}

func (v *Validator) GetExitEpoch() uint64 {
	return v.ExitEpoch
}

func (v *Validator) SetExitEpoch(epoch uint64) {
	v.ExitEpoch = epoch
}

func (v *Validator) GetBalance() *big.Int {
	return v.Balance
}

func (v *Validator) SetBalance(balance *big.Int) {
	v.Balance = balance
}

// ValidatorBinary is a Validator represented as an array of bytes.
type ValidatorBinary []byte

func (vb ValidatorBinary) ToValidator() (*Validator, error) {
	validator := new(Validator)
	err := validator.UnmarshalBinary(vb)
	if err != nil {
		return nil, err
	}

	return validator, nil
}
