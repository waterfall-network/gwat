package txlog

import (
	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
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
