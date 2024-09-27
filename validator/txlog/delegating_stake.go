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

package txlog

import (
	"bytes"
	"math/big"
	"sort"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/types"
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

func (d *DelegatingStakeLogData) Copy() DelegatingStakeLogData {
	if d == nil {
		return nil
	}
	cpy := make(DelegatingStakeLogData, len(*d))
	for i, v := range *d {
		cpy[i] = v.Copy()
	}
	return cpy
}

func (d *DelegatingStakeLogData) Sort() DelegatingStakeLogData {
	if d == nil {
		return nil
	}
	s := d.Copy()[:]
	sort.Slice(s, func(i, j int) bool {
		return bytes.Compare(s[i].Address.Bytes(), s[j].Address.Bytes()) < 0
	})
	return s
}

func (d *DelegatingStakeLogData) Topics() []common.Hash {
	if d == nil {
		return nil
	}
	accs := make(common.HashArray, len(*d))
	for i, v := range *d {
		accs[i] = v.Address.Hash()
	}
	accs.Deduplicate()
	return accs.Sort()
}

// MarshalBinary marshals a create operation to byte encoding
func (d *DelegatingStakeLogData) MarshalBinary() ([]byte, error) {
	cmp := d.Copy()
	if cmp == nil {
		cmp = DelegatingStakeLogData{}
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
func PackDelegatingStakeLogData(data DelegatingStakeLogData) ([]byte, error) {
	if data == nil {
		return []byte{}, nil
	}
	sorted := data.Sort()
	for _, v := range sorted {
		if v.Amount == nil {
			return nil, ErrNoAmount
		}
	}
	return sorted.MarshalBinary()
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

func (e *EventEmmiter) AddDelegatingStakeLog(stateValAdr common.Address, data []byte, topics []common.Hash) {
	allTopics := append([]common.Hash{EvtDelegatingStakeSignature}, topics...)
	e.state.AddLog(&types.Log{
		Address: stateValAdr,
		Topics:  allTopics,
		Data:    data,
	})
}
