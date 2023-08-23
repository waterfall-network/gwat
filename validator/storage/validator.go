package storage

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"math/big"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
)

const (
	uint64Size              = 8
	creatorAddressOffset    = common.BlsPubKeyLength
	withdrawalAddressOffset = creatorAddressOffset + common.AddressLength
	validatorIndexOffset    = withdrawalAddressOffset + common.AddressLength
	activationEraOffset     = validatorIndexOffset + uint64Size
	exitEraOffset           = activationEraOffset + uint64Size
	balanceLengthOffset     = exitEraOffset + uint64Size
	balanceOffset           = balanceLengthOffset + uint64Size
	metricOffset            = balanceOffset // TODO: add balance length to calculate offset
)

type Validator struct {
	PubKey            common.BlsPubKey  `json:"pubKey"`
	Address           common.Address    `json:"address"`
	WithdrawalAddress *common.Address   `json:"withdrawalAddress"`
	Index             uint64            `json:"index"`
	ActivationEra     uint64            `json:"activationEra"`
	ExitEra           uint64            `json:"exitEra"`
	Balance           *big.Int          `json:"balance"`
	Stake             []*StakeByAddress `json:"stake"`
}

func NewValidator(pubKey common.BlsPubKey, address common.Address, withdrawal *common.Address) *Validator {
	return &Validator{
		PubKey:            pubKey,
		Address:           address,
		WithdrawalAddress: withdrawal,
		Index:             math.MaxUint64,
		ActivationEra:     math.MaxUint64,
		ExitEra:           math.MaxUint64,
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

	binary.BigEndian.PutUint64(data[activationEraOffset:activationEraOffset+uint64Size], v.ActivationEra)

	binary.BigEndian.PutUint64(data[exitEraOffset:exitEraOffset+uint64Size], v.ExitEra)

	binary.BigEndian.PutUint64(data[balanceLengthOffset:balanceOffset], uint64(len(balance)))

	copy(data[balanceOffset:balanceOffset+len(balance)], balance)

	var stakeBuffer bytes.Buffer
	for _, stake := range v.Stake {
		// Marshal each stake to binary
		stakeData := stake.MarshalBinary()

		// Write the stake length
		binary.Write(&stakeBuffer, binary.BigEndian, uint64(len(stakeData)))

		// Write the stake data
		stakeBuffer.Write(stakeData)
	}
	stakeData := stakeBuffer.Bytes()

	// Resize the main data buffer to hold the stake data
	newData := make([]byte, len(data)+len(stakeData))
	copy(newData, data)
	copy(newData[len(data):], stakeData)

	return newData, nil
}

func (v *Validator) UnmarshalBinary(data []byte) error {
	v.PubKey = common.BlsPubKey{}
	copy(v.PubKey[:], data[:common.BlsPubKeyLength])

	v.Address = common.Address{}
	copy(v.Address[:], data[creatorAddressOffset:creatorAddressOffset+common.AddressLength])

	v.WithdrawalAddress = new(common.Address)
	copy(v.WithdrawalAddress[:], data[withdrawalAddressOffset:withdrawalAddressOffset+common.AddressLength])

	v.Index = binary.BigEndian.Uint64(data[validatorIndexOffset : validatorIndexOffset+uint64Size])

	v.ActivationEra = binary.BigEndian.Uint64(data[activationEraOffset : activationEraOffset+uint64Size])

	v.ExitEra = binary.BigEndian.Uint64(data[exitEraOffset : exitEraOffset+uint64Size])

	balanceLen := binary.BigEndian.Uint64(data[balanceLengthOffset:balanceOffset])

	v.Balance = new(big.Int).SetBytes(data[balanceOffset : balanceOffset+balanceLen])

	stakeData := data[balanceOffset+balanceLen:]
	for len(stakeData) > 0 {
		// Ensure stakeData is long enough to contain stakeLen
		if len(stakeData) < uint64Size {
			return fmt.Errorf("stakeData is too short to contain stakeLen")
		}
		// Get the size of the stake
		stakeLen := binary.BigEndian.Uint64(stakeData[:uint64Size])

		// Ensure stakeData is long enough to contain the stake
		if len(stakeData) < uint64Size+int(stakeLen) {
			return fmt.Errorf("stakeData is too short to contain the stake")
		}
		// Unmarshal the stake
		stake := &StakeByAddress{}
		err := stake.UnmarshalBinary(stakeData[uint64Size : uint64Size+stakeLen])
		if err != nil {
			return err
		}
		v.Stake = append(v.Stake, stake)

		stakeData = stakeData[uint64Size+stakeLen:]
	}

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

func (v *Validator) GetActivationEra() uint64 {
	return v.ActivationEra
}

func (v *Validator) SetActivationEra(era uint64) {
	v.ActivationEra = era
}

func (v *Validator) GetExitEra() uint64 {
	return v.ExitEra
}

func (v *Validator) SetExitEra(era uint64) {
	v.ExitEra = era
}

func (v *Validator) GetBalance() *big.Int {
	return v.Balance
}

func (v *Validator) SetBalance(balance *big.Int) {
	v.Balance = balance
}

func (v *Validator) AddStake(address common.Address, sum *big.Int) *big.Int {
	for _, existingStake := range v.Stake {
		if existingStake.Address == address {
			existingStake.Sum.Add(existingStake.Sum, sum)
			return existingStake.Sum
		}
	}
	v.Stake = append(v.Stake, &StakeByAddress{
		Address: address,
		Sum:     sum,
	})
	return sum
}

func (v *Validator) SubtractStake(address common.Address, sum *big.Int) (*big.Int, error) {
	for _, existingStake := range v.Stake {
		if existingStake.Address == address {
			// Check if the subtraction would result in a negative stake
			if existingStake.Sum.Cmp(sum) < 0 {
				return nil, fmt.Errorf("cannot subtract more than the existing stake")
			}

			existingStake.Sum.Sub(existingStake.Sum, sum)
			return existingStake.Sum, nil
		}
	}
	return nil, fmt.Errorf("no stake found for the provided address")
}

func (v *Validator) TotalStake() *big.Int {
	total := big.NewInt(0)
	for _, stake := range v.Stake {
		total.Add(total, stake.Sum)
	}
	return total
}

func (v *Validator) StakeByAddress(address common.Address) *big.Int {
	for _, stake := range v.Stake {
		if stake.Address == address {
			return stake.Sum
		}
	}
	return big.NewInt(0)
}

func (v *Validator) UnsetStake() {
	v.Stake = []*StakeByAddress{}
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

type StakeByAddress struct {
	Address common.Address `json:"address"`
	Sum     *big.Int       `json:"sum"`
}

func (s *StakeByAddress) MarshalBinary() []byte {
	// Get the bytes for the Address and Sum
	address := s.Address[:]
	sum := s.Sum.Bytes()

	// Create the binary data
	data := make([]byte, common.AddressLength+len(sum))

	copy(data[:common.AddressLength], address)
	copy(data[common.AddressLength:], sum)

	return data
}

func (s *StakeByAddress) UnmarshalBinary(data []byte) error {
	if len(data) < common.AddressLength {
		return fmt.Errorf("data is too short for unmarshalling")
	}

	// Set the Address
	s.Address = common.Address{}
	copy(s.Address[:], data[:common.AddressLength])

	// Set the Sum
	s.Sum = new(big.Int).SetBytes(data[common.AddressLength:])

	return nil
}
