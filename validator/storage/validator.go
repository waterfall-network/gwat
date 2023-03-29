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

// ValidatorInfo is a Validator represented as an array of bytes.
type ValidatorInfo []byte

func (vi ValidatorInfo) GetPubKey() common.BlsPubKey {
	return common.BytesToBlsPubKey(vi[:common.BlsPubKeyLength])
}

func (vi ValidatorInfo) SetPubKey(key common.BlsPubKey) {
	copy(vi[:common.BlsPubKeyLength], key[:])
}

func (vi ValidatorInfo) GetAddress() common.Address {
	return common.BytesToAddress(vi[creatorAddressOffset : creatorAddressOffset+common.AddressLength])
}

func (vi ValidatorInfo) SetAddress(address common.Address) {
	copy(vi[creatorAddressOffset:creatorAddressOffset+common.AddressLength], address[:])
}

func (vi ValidatorInfo) GetWithdrawalAddress() common.Address {
	return common.BytesToAddress(vi[withdrawalAddressOffset:validatorIndexOffset])
}

func (vi ValidatorInfo) SetWithdrawalAddress(address common.Address) {
	copy(vi[withdrawalAddressOffset:validatorIndexOffset], address[:])
}

func (vi ValidatorInfo) GetIndex() uint64 {
	return binary.BigEndian.Uint64(vi[validatorIndexOffset:activationEpochOffset])
}

func (vi ValidatorInfo) SetIndex(index uint64) {
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

func (vi ValidatorInfo) GetBalance() *big.Int {
	balanceLength := binary.BigEndian.Uint64(vi[balanceLengthOffset:balanceOffset])

	bal := vi[balanceOffset : balanceOffset+balanceLength]

	return new(big.Int).SetBytes(bal)
}

func SetValidatorBalance(valInfo ValidatorInfo, balance *big.Int) ValidatorInfo {
	balanceLen := len(balance.Bytes())
	binary.BigEndian.PutUint64(valInfo[balanceLengthOffset:balanceOffset], uint64(balanceLen))

	newVi := make(ValidatorInfo, len(valInfo)+balanceLen)
	copy(newVi, valInfo[:balanceOffset])
	copy(newVi[balanceOffset:], balance.Bytes())

	return newVi
}
