package txlog

import (
	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/types"
	"gitlab.waterfall.network/waterfall/protocol/gwat/rlp"
)

type DeactivateLogData struct {
	InitTxHash     common.Hash
	CreatorAddress common.Address
	ProcEpoch      uint64
	ValidatorIndex uint64
}

// MarshalBinary marshals a create operation to byte encoding
func (d *DeactivateLogData) MarshalBinary() ([]byte, error) {
	cmp := d.Copy()
	if cmp == nil {
		cmp = &DeactivateLogData{}
	}
	return rlp.EncodeToBytes(cmp)
}

// UnmarshalBinary unmarshals a create operation from byte encoding
func (d *DeactivateLogData) UnmarshalBinary(b []byte) error {
	return rlp.DecodeBytes(b, d)
}

func (d *DeactivateLogData) Copy() *DeactivateLogData {
	if d == nil {
		return nil
	}
	return &DeactivateLogData{
		InitTxHash:     common.BytesToHash(d.InitTxHash.Bytes()),
		CreatorAddress: common.BytesToAddress(d.CreatorAddress.Bytes()),
		ProcEpoch:      d.ProcEpoch,
		ValidatorIndex: d.ValidatorIndex,
	}
}

// PackDeactivateLogData packs the deposit log.
func PackDeactivateLogData(
	initTxHash common.Hash,
	creatorAddress common.Address,
	procEpoch uint64,
	validatorIndex uint64,
) ([]byte, error) {
	logData := &DeactivateLogData{
		InitTxHash:     initTxHash,
		CreatorAddress: creatorAddress,
		ProcEpoch:      procEpoch,
		ValidatorIndex: validatorIndex,
	}
	return logData.MarshalBinary()
}

// UnpackDeactivateLogData unpacks the data from a deposit log using the ABI decoder.
func UnpackDeactivateLogData(bin []byte) (
	initTxHash common.Hash,
	creatorAddress common.Address,
	procEpoch uint64,
	validatorIndex uint64,
	err error,
) {
	logData := &DeactivateLogData{}
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

func (e *EventEmmiter) AddDeactivateLog(stateValAdr common.Address, data []byte, creatorAdr common.Address, initTxHash common.Hash) {
	topics := []common.Hash{
		EvtDeactivateLogSignature,
		creatorAdr.Hash(),
		initTxHash,
	}

	e.state.AddLog(&types.Log{
		Address: stateValAdr,
		Topics:  topics,
		Data:    data,
	})
}
