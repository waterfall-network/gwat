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
	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/types"
	"gitlab.waterfall.network/waterfall/protocol/gwat/rlp"
)

type ActivateLogData struct {
	InitTxHash     common.Hash
	CreatorAddress common.Address
	ProcEpoch      uint64
	ValidatorIndex uint64
}

// MarshalBinary marshals a create operation to byte encoding
func (d *ActivateLogData) MarshalBinary() ([]byte, error) {
	cmp := d.Copy()
	if cmp == nil {
		cmp = &ActivateLogData{}
	}
	return rlp.EncodeToBytes(cmp)
}

// UnmarshalBinary unmarshals a create operation from byte encoding
func (d *ActivateLogData) UnmarshalBinary(b []byte) error {
	return rlp.DecodeBytes(b, d)
}

func (d *ActivateLogData) Copy() *ActivateLogData {
	if d == nil {
		return nil
	}
	return &ActivateLogData{
		InitTxHash:     common.BytesToHash(d.InitTxHash.Bytes()),
		CreatorAddress: common.BytesToAddress(d.CreatorAddress.Bytes()),
		ProcEpoch:      d.ProcEpoch,
		ValidatorIndex: d.ValidatorIndex,
	}
}

// PackActivateLogData packs the deposit log.
func PackActivateLogData(
	initTxHash common.Hash,
	creatorAddress common.Address,
	procEpoch uint64,
	validatorIndex uint64,
) ([]byte, error) {
	logData := &ActivateLogData{
		InitTxHash:     initTxHash,
		CreatorAddress: creatorAddress,
		ProcEpoch:      procEpoch,
		ValidatorIndex: validatorIndex,
	}
	return logData.MarshalBinary()
}

// UnpackActivateLogData unpacks the data from a deposit log using the ABI decoder.
func UnpackActivateLogData(bin []byte) (
	initTxHash common.Hash,
	creatorAddress common.Address,
	procEpoch uint64,
	validatorIndex uint64,
	err error,
) {
	logData := &ActivateLogData{}
	err = logData.UnmarshalBinary(bin)
	if err != nil {
		return
	}
	initTxHash = logData.InitTxHash
	creatorAddress = logData.CreatorAddress
	procEpoch = logData.ProcEpoch
	validatorIndex = logData.ValidatorIndex
	return
}

func (e *EventEmmiter) AddActivateLog(stateValAdr common.Address, data []byte, creatorAdr common.Address, initTxHash common.Hash) {
	topics := []common.Hash{
		EvtActivateLogSignature,
		creatorAdr.Hash(),
		initTxHash,
	}

	e.state.AddLog(&types.Log{
		Address: stateValAdr,
		Topics:  topics,
		Data:    data,
	})
}
