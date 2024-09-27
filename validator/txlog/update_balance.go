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
	"math/big"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/types"
	"gitlab.waterfall.network/waterfall/protocol/gwat/rlp"
)

type UpdateBalanceLogData struct {
	InitTxHash     common.Hash
	CreatorAddress common.Address
	ProcEpoch      uint64
	Amount         *big.Int
}

// MarshalBinary marshals a create operation to byte encoding
func (d *UpdateBalanceLogData) MarshalBinary() ([]byte, error) {
	cmp := d.Copy()
	if cmp == nil {
		cmp = &UpdateBalanceLogData{}
	}
	if cmp.Amount == nil {
		cmp.Amount = new(big.Int)
	}
	return rlp.EncodeToBytes(cmp)
}

// UnmarshalBinary unmarshals a create operation from byte encoding
func (d *UpdateBalanceLogData) UnmarshalBinary(b []byte) error {
	return rlp.DecodeBytes(b, d)
}

func (d *UpdateBalanceLogData) Copy() *UpdateBalanceLogData {
	if d == nil {
		return nil
	}
	var amt *big.Int
	if d.Amount != nil {
		amt = new(big.Int).Set(d.Amount)
	}
	return &UpdateBalanceLogData{
		InitTxHash:     common.BytesToHash(d.InitTxHash.Bytes()),
		CreatorAddress: common.BytesToAddress(d.CreatorAddress.Bytes()),
		ProcEpoch:      d.ProcEpoch,
		Amount:         amt,
	}
}

// PackUpdateBalanceLogData packs the deposit log.
func PackUpdateBalanceLogData(
	initTxHash common.Hash,
	creatorAddress common.Address,
	procEpoch uint64,
	amount *big.Int,
) ([]byte, error) {
	if amount == nil {
		return nil, ErrNoAmount
	}
	logData := &UpdateBalanceLogData{
		InitTxHash:     initTxHash,
		CreatorAddress: creatorAddress,
		ProcEpoch:      procEpoch,
		Amount:         amount,
	}
	return logData.MarshalBinary()
}

// UnpackUpdateBalanceLogData unpacks the data from a deposit log using the ABI decoder.
func UnpackUpdateBalanceLogData(bin []byte) (
	initTxHash common.Hash,
	creatorAddress common.Address,
	procEpoch uint64,
	amount *big.Int,
	err error,
) {
	logData := &UpdateBalanceLogData{}
	err = logData.UnmarshalBinary(bin)
	if err != nil {
		return
	}
	initTxHash = logData.InitTxHash
	creatorAddress = logData.CreatorAddress
	procEpoch = logData.ProcEpoch
	amount = logData.Amount
	return
}

func (e *EventEmmiter) AddUpdateBalanceLog(stateValAdr common.Address, data []byte, creatorAdr common.Address, initTxHash common.Hash, withdrawalAdr *common.Address) {
	topics := []common.Hash{
		EvtUpdateBalanceLogSignature,
		creatorAdr.Hash(),
		initTxHash,
	}
	//skipping for delegate stake case
	if withdrawalAdr != nil {
		topics = append(topics, withdrawalAdr.Hash())
	}
	e.state.AddLog(&types.Log{
		Address: stateValAdr,
		Topics:  topics,
		Data:    data,
	})
}
