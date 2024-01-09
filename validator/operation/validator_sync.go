package operation

import (
	"encoding/binary"
	"fmt"
	"math/big"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/types"
	"gitlab.waterfall.network/waterfall/protocol/gwat/rlp"
)

const valSyncOpDataMinLen = 8 + 8 + common.AddressLength + common.HashLength

type VersionValSyncOp uint16

const (
	NoVer VersionValSyncOp = iota
	Ver1
)

type validatorSyncOperation struct {
	version           VersionValSyncOp
	opType            types.ValidatorSyncOp
	initTxHash        common.Hash
	procEpoch         uint64
	index             uint64
	creator           common.Address
	amount            *big.Int
	withdrawalAddress *common.Address
	balance           *big.Int
}

// rlpValSyncOpVer1 rlp representation of ValidatorSyncOperation op ver 1.
type rlpValSyncOpVer1 struct {
	OpType            types.ValidatorSyncOp
	InitTxHash        common.Hash
	ProcEpoch         uint64
	Index             uint64
	Creator           common.Address
	WithdrawalAddress common.Address
	Amount            big.Int
	Balance           big.Int
}

func (op *validatorSyncOperation) init(
	version VersionValSyncOp,
	opType types.ValidatorSyncOp,
	initTxHash common.Hash,
	procEpoch uint64,
	index uint64,
	creator common.Address,
	amount *big.Int,
	withdrawalAddress *common.Address,
	balance *big.Int,
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
		if version > NoVer && balance == nil {
			return ErrNoBalance
		}
	}
	op.initTxHash = initTxHash
	op.opType = opType
	op.procEpoch = procEpoch
	op.index = index
	op.creator = creator
	op.amount = amount
	if opType == types.UpdateBalance {
		op.withdrawalAddress = withdrawalAddress
		op.balance = balance
	}
	op.version = version
	return nil
}

// NewValidatorSyncOperation creates an operation for creating validator sync operation.
func NewValidatorSyncOperation(
	version VersionValSyncOp,
	opType types.ValidatorSyncOp,
	initTxHash common.Hash,
	procEpoch uint64,
	index uint64,
	creator common.Address,
	amount *big.Int,
	withdrawalAddress *common.Address,
	balance *big.Int,
) (ValidatorSync, error) {
	op := validatorSyncOperation{}
	if err := op.init(version, opType, initTxHash, procEpoch, index, creator, amount, withdrawalAddress, balance); err != nil {
		return nil, err
	}
	return &op, nil
}

// UnmarshalBinary unmarshals a create operation from byte encoding
func (op *validatorSyncOperation) UnmarshalBinary(b []byte) error {
	version, binData, err := unwrapVersionedData(b)
	if err != nil {
		return op.unmarshalBinaryLegacy(b)
	}
	switch version {
	case Ver1:
		dec := &rlpValSyncOpVer1{}
		err = rlp.DecodeBytes(binData, dec)
		if err != nil {
			return err
		}
		return op.init(version, dec.OpType, dec.InitTxHash, dec.ProcEpoch, dec.Index, dec.Creator, &dec.Amount, &dec.WithdrawalAddress, &dec.Balance)
	default:
		return ErrOpBadVersion
	}
}

// MarshalBinary marshals a create operation to byte encoding
func (op *validatorSyncOperation) MarshalBinary() ([]byte, error) {
	var (
		binData           []byte
		err               error
		withdrawalAddress = &common.Address{}
		amount            = new(big.Int)
		balance           = new(big.Int)
	)
	if op.opType == types.UpdateBalance {
		withdrawalAddress = op.withdrawalAddress
		amount = op.amount
		balance = op.balance
	}
	switch op.version {
	case NoVer:
		return op.marshalBinaryLegacy()
	case Ver1:
		binData, err = rlp.EncodeToBytes(&rlpValSyncOpVer1{
			OpType:            op.opType,
			InitTxHash:        op.initTxHash,
			ProcEpoch:         op.procEpoch,
			Index:             op.index,
			Creator:           op.creator,
			WithdrawalAddress: *withdrawalAddress,
			Amount:            *amount,
			Balance:           *balance,
		})
	default:
		return nil, ErrOpBadVersion
	}
	if err != nil {
		return nil, err
	}
	return wrapVersionedData(op.version, binData)
}

// UnmarshalBinary unmarshals deprecated validator sync operation from byte encoding.
func (op *validatorSyncOperation) unmarshalBinaryLegacy(b []byte) error {
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
	return op.init(NoVer, opType, initTxHash, procEpoch, index, creator, amount, &withdrawal, nil)
}

// marshalBinaryLegacy marshals deprecated validator sync operation to byte encoding.
func (op *validatorSyncOperation) marshalBinaryLegacy() ([]byte, error) {
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

func (op *validatorSyncOperation) Balance() *big.Int {
	if op.balance == nil {
		return nil
	}
	return new(big.Int).Set(op.balance)
}

func (op *validatorSyncOperation) Version() VersionValSyncOp {
	return op.version
}
func (op *validatorSyncOperation) SetVersion(ver VersionValSyncOp) {
	op.version = ver
}

func (op *validatorSyncOperation) Print() string {
	if op == nil {
		return "{nil}"
	}
	return fmt.Sprintf("{InitTxHash: %#x, OpType: %d, ProcEpoch: %d, Index: %d, Creator: %#x, Amount: %s, Balance: %s, WithdrawalAddress: %#x, ver: %d}",
		op.initTxHash,
		op.opType,
		op.procEpoch,
		op.index,
		op.creator,
		op.amount.String(),
		op.balance.String(),
		op.withdrawalAddress.Hex(),
		op.version,
	)
}

type rlpVerWrapper struct {
	Version VersionValSyncOp
	Data    []byte
}

func wrapVersionedData(ver VersionValSyncOp, bin []byte) ([]byte, error) {
	return rlp.EncodeToBytes(rlpVerWrapper{
		Version: ver,
		Data:    bin,
	})
}

func unwrapVersionedData(bin []byte) (ver VersionValSyncOp, data []byte, err error) {
	verWrap := &rlpVerWrapper{}
	err = rlp.DecodeBytes(bin, verWrap)
	if err != nil {
		return
	}
	return verWrap.Version, verWrap.Data, nil
}
