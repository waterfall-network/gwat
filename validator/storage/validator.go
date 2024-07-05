package storage

import (
	"encoding/binary"
	"fmt"
	"math"
	"math/big"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/rlp"
	"gitlab.waterfall.network/waterfall/protocol/gwat/validator/operation"
)

type ValidatorVersion uint16

const (
	NoVer ValidatorVersion = iota
	Ver1
)

type Validator struct {
	// the Address property must be the first for IsValidatorAddress
	Address           common.Address                 `json:"address"`
	PubKey            common.BlsPubKey               `json:"pubKey"`
	WithdrawalAddress *common.Address                `json:"withdrawalAddress"`
	Index             uint64                         `json:"index"`
	ActivationEra     uint64                         `json:"activationEra"`
	ExitEra           uint64                         `json:"exitEra"`
	Stake             []*StakeByAddress              `json:"stake"`
	DelegatingStake   *operation.DelegatingStakeData `json:"delegatingStake"`
	Ver               *ValidatorVersion              `json:"version"`
	DepositTxs        common.HashArray               `json:"depositTxs"`
	WithdrawalTxs     common.HashArray               `json:"withdrawalTxs"`
	ExitTx            *common.Hash                   `json:"exitTx"`
}

func NewValidator(pubKey common.BlsPubKey, address common.Address, withdrawal *common.Address) *Validator {
	return &Validator{
		PubKey:            pubKey,
		Address:           address,
		WithdrawalAddress: withdrawal,
		Index:             math.MaxUint64,
		ActivationEra:     math.MaxUint64,
		ExitEra:           math.MaxUint64,
		Stake:             []*StakeByAddress{},
		DelegatingStake:   nil,
	}
}

func (v *Validator) Copy() *Validator {
	if v == nil {
		return nil
	}
	cpy := &Validator{
		Index:         v.Index,
		ActivationEra: v.ActivationEra,
		ExitEra:       v.ExitEra,
	}
	copy(cpy.PubKey[:], v.PubKey[:])
	copy(cpy.Address[:], v.Address[:])
	if v.WithdrawalAddress != nil {
		cpy.WithdrawalAddress = &common.Address{}
		copy(cpy.WithdrawalAddress[:], v.WithdrawalAddress[:])
	}
	cpy.Stake = make([]*StakeByAddress, len(v.Stake))
	for i, stake := range v.Stake {
		cpy.Stake[i] = stake.Copy()
	}
	return cpy
}

func (v *Validator) hasExtendedData() bool {
	return v.DelegatingStake != nil || v.Version() != NoVer
}

type rlpBaseValidator struct {
	//Address           common.Address    `json:"address"`
	PubKey            common.BlsPubKey  `json:"pubKey"`
	WithdrawalAddress *common.Address   `json:"withdrawalAddress"`
	Index             uint64            `json:"index"`
	ActivationEra     uint64            `json:"activationEra"`
	ExitEra           uint64            `json:"exitEra"`
	Stake             []*StakeByAddress `json:"stake"`
}

type rlpVer1 struct {
	DepositTxs    common.HashArray `json:"depositTxs"`
	WithdrawalTxs common.HashArray `json:"withdrawalTxs"`
	ExitTx        *common.Hash     `json:"exitTx"`
}

func (v *Validator) MarshalBinary() ([]byte, error) {
	rlpData := rlpBaseValidator{
		PubKey:            v.PubKey,
		WithdrawalAddress: v.WithdrawalAddress,
		Index:             v.Index,
		ActivationEra:     v.ActivationEra,
		ExitEra:           v.ExitEra,
		Stake:             v.Stake,
	}
	if rlpData.WithdrawalAddress == nil {
		rlpData.WithdrawalAddress = &common.Address{}
	}
	if rlpData.Stake == nil {
		rlpData.Stake = make([]*StakeByAddress, 0)
	}
	baseData, err := rlp.EncodeToBytes(rlpData)
	if err != nil {
		return nil, err
	}

	// marshal binary extended data
	var (
		delegateBin, versionData []byte
		extenedDataLen           int
	)
	//prevent modify state before any forks
	if v.hasExtendedData() {
		// delegate stake data
		delegateBin, err = v.DelegatingStake.MarshalBinary()
		if err != nil {
			return nil, err
		}
		extenedDataLen = len(delegateBin) + common.Uint32Size

		// if version set
		if v.Version() > NoVer {
			extenedDataLen += common.Uint16Size

			//prepare rlp data
			exitTx := v.GetExitTx()
			if exitTx == nil {
				exitTx = &common.Hash{}
			}

			// make binary versioned data
			switch v.Version() {
			case Ver1:
				dataVer1 := rlpVer1{
					DepositTxs:    v.GetDepositTxs(),
					WithdrawalTxs: v.GetWithdrawalTxs(),
					ExitTx:        exitTx,
				}
				versionData, err = rlp.EncodeToBytes(dataVer1)
			default:
				err = fmt.Errorf("invalid Validator version: %d", v.Version())
			}
			if err != nil {
				return nil, err
			}
			extenedDataLen += len(versionData)
		}
	}

	//create bin data
	binDataLen := common.AddressLength + common.Uint32Size + len(baseData) + extenedDataLen
	binData := make([]byte, binDataLen)

	var startOffset, endOfset int

	// set validator address at the first position
	startOffset = endOfset
	endOfset = startOffset + common.AddressLength
	copy(binData[startOffset:endOfset], v.Address.Bytes())

	// set len of base validator data
	startOffset = endOfset
	endOfset = startOffset + common.Uint32Size
	binary.BigEndian.PutUint32(binData[startOffset:endOfset], uint32(len(baseData)))

	// set base validator data
	startOffset = endOfset
	endOfset = startOffset + len(baseData)
	copy(binData[startOffset:endOfset], baseData)

	//prevent modify state before any forks
	if !v.hasExtendedData() {
		return binData, nil
	}

	// set extended data
	//set len of delegate stake data
	startOffset = endOfset
	endOfset = startOffset + common.Uint32Size
	binary.BigEndian.PutUint32(binData[startOffset:endOfset], uint32(len(delegateBin)))
	// delegate stake data
	startOffset = endOfset
	endOfset = startOffset + len(delegateBin)
	copy(binData[startOffset:endOfset], delegateBin)

	if v.Version() == NoVer {
		//if not versioned data
		return binData, nil
	}

	//set version of data
	startOffset = endOfset
	endOfset = startOffset + common.Uint16Size
	binary.BigEndian.PutUint16(binData[startOffset:endOfset], uint16(v.Version()))
	//set versioned data
	startOffset = endOfset
	endOfset = startOffset + len(versionData)
	copy(binData[startOffset:endOfset], versionData)

	return binData, nil
}

func (v *Validator) UnmarshalBinary(data []byte) error {
	var startOffset, endOfset int

	// get validator address at the first position
	v.Address = common.Address{}
	startOffset = endOfset
	endOfset = startOffset + common.AddressLength
	copy(v.Address[:], data[startOffset:endOfset])

	// get len of base validator data
	startOffset = endOfset
	endOfset = startOffset + common.Uint32Size
	baseDataLen := int(binary.BigEndian.Uint32(data[startOffset:endOfset]))
	if baseDataLen > len(data[endOfset:]) {
		return errBadBinaryData
	}
	// get base validator data
	startOffset = endOfset
	endOfset = startOffset + baseDataLen
	baseValData := &rlpBaseValidator{}
	if err := rlp.DecodeBytes(data[startOffset:endOfset], baseValData); err != nil {
		return err
	}

	v.PubKey = baseValData.PubKey
	v.WithdrawalAddress = baseValData.WithdrawalAddress
	v.Index = baseValData.Index
	v.ActivationEra = baseValData.ActivationEra
	v.ExitEra = baseValData.ExitEra
	v.Stake = baseValData.Stake

	if *v.WithdrawalAddress == (common.Address{}) {
		v.WithdrawalAddress = nil
	}

	// retrieve extended data
	extendedData := data[endOfset:]
	if len(extendedData) > 0 {
		// delegate stake data
		// retrieve data len
		startOffset = 0
		endOfset = startOffset + common.Uint32Size
		delegateDataLen := int(binary.BigEndian.Uint32(extendedData[startOffset:endOfset]))
		if delegateDataLen > len(extendedData[endOfset:]) {
			return errBadBinaryData
		}
		// get delegate data
		startOffset = endOfset
		endOfset = startOffset + delegateDataLen
		delegatingStake, err := operation.NewDelegatingStakeDataFromBinary(extendedData[startOffset:endOfset])
		if err != nil {
			return err
		}
		v.DelegatingStake = delegatingStake

		versionedData := extendedData[endOfset:]
		if len(versionedData) > 0 {
			//decode versioned data
			startOffset = 0
			endOfset = startOffset + common.Uint16Size
			version := ValidatorVersion(binary.BigEndian.Uint16(versionedData[startOffset:endOfset]))
			v.SetVersion(version)

			// decode versioned data
			switch v.Version() {
			case Ver1:
				ver1Data := &rlpVer1{}
				startOffset = endOfset
				if err = rlp.DecodeBytes(versionedData[startOffset:], ver1Data); err != nil {
					return err
				}
				//set data ver1
				v.setDepositTxs(ver1Data.DepositTxs)
				v.SetWithdrawalTxs(ver1Data.WithdrawalTxs)
				v.SetExitTx(ver1Data.ExitTx)
			default:
				return fmt.Errorf("invalid Validator version: %d", v.Version())
			}
		}
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

func (v *Validator) RmStakeByAddress(address common.Address) {
	stake := make([]*StakeByAddress, 0, len(v.Stake))
	for _, s := range v.Stake {
		if s.Address != address {
			stake = append(stake, s)
		}
	}
	v.Stake = stake
}

func (v *Validator) UnsetStake() {
	v.Stake = []*StakeByAddress{}
}

// HasDelegatingStake returns true if validator has delegating rules to apply.
func (v *Validator) HasDelegatingStake() bool {
	return !v.DelegatingStake.IsEmpty()
}

func (v *Validator) Version() ValidatorVersion {
	if v.Ver == nil {
		return 0
	}
	return *v.Ver
}
func (v *Validator) SetVersion(version ValidatorVersion) {
	if version == NoVer {
		v.Ver = nil
	}
	v.Ver = &version
}

func (v *Validator) GetExitTx() *common.Hash {
	if v.Version() == NoVer || v.ExitTx != nil && *v.ExitTx == (common.Hash{}) {
		return nil
	}
	return v.ExitTx
}

func (v *Validator) SetExitTx(tx *common.Hash) {
	if v.Version() == NoVer || tx != nil && *tx == (common.Hash{}) {
		v.ExitTx = nil
		return
	}
	v.ExitTx = tx
}

func (v *Validator) GetDepositTxs() common.HashArray {
	if v.Version() == NoVer {
		return common.HashArray{}
	}
	return v.DepositTxs.Copy()
}

func (v *Validator) setDepositTxs(txs common.HashArray) {
	if v.Version() == NoVer {
		v.DepositTxs = common.HashArray{}
		return
	}
	v.DepositTxs = txs
}

func (v *Validator) AddDepositTxs(txHash common.Hash) {
	if v.Version() == NoVer {
		v.DepositTxs = common.HashArray{}
		return
	}
	v.DepositTxs = append(v.GetDepositTxs(), txHash)
}

func (v *Validator) ResetDepositTxs() {
	v.DepositTxs = common.HashArray{}
}

func (v *Validator) GetWithdrawalTxs() common.HashArray {
	if v.Version() == NoVer {
		return common.HashArray{}
	}
	return v.WithdrawalTxs.Copy()
}

func (v *Validator) SetWithdrawalTxs(txs common.HashArray) {
	if v.Version() == NoVer {
		v.WithdrawalTxs = common.HashArray{}
		return
	}
	v.WithdrawalTxs = txs
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

func (s *StakeByAddress) Copy() *StakeByAddress {
	if s == nil {
		return nil
	}
	cpy := &StakeByAddress{}
	copy(cpy.Address[:], s.Address[:])
	if s.Sum != nil {
		cpy.Sum = new(big.Int).Set(s.Sum)
	}
	return cpy
}
