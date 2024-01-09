package txlog

import (
	"math/big"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
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
	if err := rlp.DecodeBytes(b, d); err != nil {
		return err
	}
	return nil
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
