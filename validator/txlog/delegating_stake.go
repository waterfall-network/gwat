package txlog

import (
	"math/big"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/rlp"
)

type UpdateBalanceRuleType uint8

const (
	NoRule UpdateBalanceRuleType = iota
	ProfitShare
	StakeShare
)

type ShareRuleApplying struct {
	Address  common.Address
	RuleType UpdateBalanceRuleType
	IsTrial  bool
	Amount   *big.Int
}

func (d *ShareRuleApplying) Copy() *ShareRuleApplying {
	if d == nil {
		return nil
	}
	var amt *big.Int
	if d.Amount != nil {
		amt = new(big.Int).Set(d.Amount)
	}
	return &ShareRuleApplying{
		Address:  d.Address,
		RuleType: d.RuleType,
		IsTrial:  d.IsTrial,
		Amount:   amt,
	}
}

type DelegatingStakeLogData []*ShareRuleApplying

func (d *DelegatingStakeLogData) Copy() *DelegatingStakeLogData {
	if d == nil {
		return nil
	}
	cpy := make(DelegatingStakeLogData, len(*d))
	for i, v := range *d {
		cpy[i] = v.Copy()
	}
	return &cpy
}

// MarshalBinary marshals a create operation to byte encoding
func (d *DelegatingStakeLogData) MarshalBinary() ([]byte, error) {
	cmp := d.Copy()
	if cmp == nil {
		cmp = &DelegatingStakeLogData{}
	}
	for _, v := range *d {
		if v.Amount == nil {
			v.Amount = new(big.Int)
		}
	}
	return rlp.EncodeToBytes(cmp)
}

// UnmarshalBinary unmarshals a create operation from byte encoding
func (d *DelegatingStakeLogData) UnmarshalBinary(b []byte) error {
	if err := rlp.DecodeBytes(b, d); err != nil {
		return err
	}
	return nil
}

// PackDelegatingStakeLogData packs the deposit log.
func PackDelegatingStakeLogData(data *DelegatingStakeLogData) ([]byte, error) {
	if data == nil {
		return []byte{}, nil
	}
	for _, v := range *data {
		if v.Amount == nil {
			return nil, ErrNoAmount
		}
	}
	return data.MarshalBinary()
}

// UnpackDelegatingStakeLogData unpacks the data from a deposit log using the ABI decoder.
func UnpackDelegatingStakeLogData(bin []byte) (*DelegatingStakeLogData, error) {
	if len(bin) == 0 {
		return nil, nil
	}
	logData := &DelegatingStakeLogData{}
	err := logData.UnmarshalBinary(bin)
	if err != nil {
		return nil, err
	}
	return logData, nil
}
