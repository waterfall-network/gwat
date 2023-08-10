package operation

import (
	"encoding/binary"
	"math/big"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/types"
)

const valSyncOpDataMinLen = 8 + 8 + 8 + common.AddressLength + common.HashLength

type validatorSyncOperation struct {
	initTxHash        common.Hash
	opType            types.ValidatorSyncOp
	procEpoch         uint64
	index             uint64
	creator           common.Address
	amount            *big.Int
	withdrawalAddress *common.Address
}

func (op *validatorSyncOperation) init(
	initTxHash common.Hash,
	opType types.ValidatorSyncOp,
	procEpoch uint64,
	index uint64,
	creator common.Address,
	amount *big.Int,
	withdrawalAddress *common.Address,
) error {
	if initTxHash == (common.Hash{}) {
		return ErrNoInitTxHash
	}
	if creator == (common.Address{}) {
		return ErrNoCreatorAddress
	}
	if opType == types.UpdateBalance {
		if amount == nil {
			return ErrNoAmount
		}
		if withdrawalAddress == nil {
			return ErrNoWithdrawalAddress
		}
	}
	op.initTxHash = initTxHash
	op.opType = opType
	op.procEpoch = procEpoch
	op.index = index
	op.creator = creator
	op.amount = amount
	op.withdrawalAddress = withdrawalAddress
	return nil
}

// NewValidatorSyncOperation creates an operation for creating validator sync operation.
func NewValidatorSyncOperation(
	initTxHash common.Hash,
	opType types.ValidatorSyncOp,
	procEpoch uint64,
	index uint64,
	creator common.Address,
	amount *big.Int,
	withdrawalAddress *common.Address,
) (ValidatorSync, error) {
	op := validatorSyncOperation{}
	if err := op.init(initTxHash, opType, procEpoch, index, creator, amount, withdrawalAddress); err != nil {
		return nil, err
	}
	return &op, nil
}

// UnmarshalBinary unmarshals a create operation from byte encoding
func (op *validatorSyncOperation) UnmarshalBinary(b []byte) error {
	if len(b) < valSyncOpDataMinLen {
		return ErrBadDataLen
	}
	startOffset := 0
	endOffset := startOffset + 8
	opType := types.ValidatorSyncOp(binary.BigEndian.Uint64(b[startOffset:endOffset]))

	startOffset = endOffset
	endOffset = startOffset + common.HashLength
	initTxHash := common.BytesToHash(b[startOffset:endOffset])

	startOffset = endOffset
	endOffset = startOffset + 8
	procEpoch := binary.BigEndian.Uint64(b[startOffset:endOffset])

	startOffset = endOffset
	endOffset = startOffset + 8
	index := binary.BigEndian.Uint64(b[startOffset:endOffset])

	startOffset = endOffset
	endOffset = startOffset + common.AddressLength
	creator := common.BytesToAddress(b[startOffset:endOffset])
	var (
		amount     *big.Int
		withdrawal common.Address
	)
	if opType == types.UpdateBalance {
		startOffset = endOffset
		endOffset = startOffset + common.AddressLength
		withdrawal = common.BytesToAddress(b[startOffset:endOffset])

		startOffset = endOffset
		amount = new(big.Int).SetBytes(b[startOffset:])
	}
	return op.init(initTxHash, opType, procEpoch, index, creator, amount, &withdrawal)
}

// MarshalBinary marshals a create operation to byte encoding
func (op *validatorSyncOperation) MarshalBinary() ([]byte, error) {
	bin := make([]byte, 0, valSyncOpDataMinLen)

	enc := make([]byte, 8)
	binary.BigEndian.PutUint64(enc, uint64(op.opType))
	bin = append(bin, enc...)

	bin = append(bin, op.initTxHash.Bytes()...)

	enc = make([]byte, 8)
	binary.BigEndian.PutUint64(enc, op.procEpoch)
	bin = append(bin, enc...)

	enc = make([]byte, 8)
	binary.BigEndian.PutUint64(enc, op.index)
	bin = append(bin, enc...)

	bin = append(bin, op.creator.Bytes()...)

	if op.withdrawalAddress != nil {
		bin = append(bin, op.withdrawalAddress.Bytes()...)
	}
	if op.amount != nil {
		bin = append(bin, op.amount.Bytes()...)
	}
	return bin, nil
}

// Code returns op code of a deposit operation
func (op *validatorSyncOperation) OpCode() Code {
	var code Code
	switch op.opType {
	case types.Activate:
		code = ActivateCode
	case types.Deactivate:
		code = DeactivateCode
	case types.UpdateBalance:
		code = UpdateBalanceCode
	}
	return code
}

// Code always returns an empty address
// It's just a stub for the Operation interface.
func (op *validatorSyncOperation) Address() common.Address {
	return common.Address{}
}

func (op *validatorSyncOperation) InitTxHash() common.Hash {
	return op.initTxHash
}

func (op *validatorSyncOperation) OpType() types.ValidatorSyncOp {
	return op.opType
}

func (op *validatorSyncOperation) ProcEpoch() uint64 {
	return op.procEpoch
}

func (op *validatorSyncOperation) Index() uint64 {
	return op.index
}

func (op *validatorSyncOperation) Creator() common.Address {
	return common.BytesToAddress(op.creator.Bytes())
}

func (op *validatorSyncOperation) Amount() *big.Int {
	if op.amount == nil {
		return nil
	}
	return new(big.Int).Set(op.amount)
}

func (op *validatorSyncOperation) WithdrawalAddress() *common.Address {
	if op.withdrawalAddress == nil {
		return nil
	}
	cpy := common.BytesToAddress(op.withdrawalAddress.Bytes())
	return &cpy
}
