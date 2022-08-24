package operation

import (
	"github.com/waterfall-foundation/gwat/common"

	"math/big"
)

type approveOperation struct {
	operation
	valueOperation
	spenderOperation
}

// NewApproveOperation creates a token approve operation.
// Same logic with standard parameter applies as with the transfer from factory function.
func NewApproveOperation(standard Std, spender common.Address, value *big.Int) (Approve, error) {
	if spender == (common.Address{}) {
		return nil, ErrNoSpender
	}
	if value == nil {
		return nil, ErrNoValue
	}
	return &approveOperation{
		operation: operation{
			Std: standard,
		},
		valueOperation: valueOperation{
			TokenValue: value,
		},
		spenderOperation: spenderOperation{
			SpenderAddress: spender,
		},
	}, nil
}

// Code returns op code of a balance of operation
func (op *approveOperation) OpCode() Code {
	return ApproveCode
}

// UnmarshalBinary unmarshals a token approve operation from byte encoding
func (op *approveOperation) UnmarshalBinary(b []byte) error {
	return rlpDecode(b, op)
}

// MarshalBinary marshals a token approve operation to byte encoding
func (op *approveOperation) MarshalBinary() ([]byte, error) {
	return rlpEncode(op)
}

type balanceOfOperation struct {
	addressOperation
	ownerOperation
}

// NewBalanceOfOperation creates a balance of operation
func NewBalanceOfOperation(address common.Address, owner common.Address) (BalanceOf, error) {
	if address == (common.Address{}) {
		return nil, ErrNoAddress
	}
	if owner == (common.Address{}) {
		return nil, ErrNoOwner
	}
	return &balanceOfOperation{
		addressOperation: addressOperation{
			TokenAddress: address,
		},
		ownerOperation: ownerOperation{
			OwnerAddress: owner,
		},
	}, nil
}

// Code returns op code of a balance of operation
func (op *balanceOfOperation) OpCode() Code {
	return BalanceOfCode
}

// Standard method is just a stub to implement Operation interface.
// Always returns 0.
func (op *balanceOfOperation) Standard() Std {
	return 0
}

// UnmarshalBinary unmarshals a balance of operation from byte encoding
func (op *balanceOfOperation) UnmarshalBinary(b []byte) error {
	return rlpDecode(b, op)
}

// MarshalBinary marshals a balance of operation to byte encoding
func (op *balanceOfOperation) MarshalBinary() ([]byte, error) {
	return rlpEncode(op)
}

type propertiesOperation struct {
	addressOperation
	Id *big.Int
}

// NewPropertiesOperation creates a token properties operation
// tokenId parameters isn't required.
func NewPropertiesOperation(address common.Address, tokenId *big.Int) (Properties, error) {
	if address == (common.Address{}) {
		return nil, ErrNoAddress
	}
	return &propertiesOperation{
		addressOperation: addressOperation{
			TokenAddress: address,
		},
		Id: tokenId,
	}, nil
}

// Standard always returns zero value
// It's just a stub for the Operation interface.
func (op *propertiesOperation) Standard() Std {
	return 0
}

// UnmarshalBinary unmarshals a properties operation from byte encoding
func (op *propertiesOperation) UnmarshalBinary(b []byte) error {
	return rlpDecode(b, op)
}

// MarshalBinary marshals a properties operation to byte encoding
func (op *propertiesOperation) MarshalBinary() ([]byte, error) {
	return rlpEncode(op)
}

// Code returns op code of a create operation
func (op *propertiesOperation) OpCode() Code {
	return PropertiesCode
}

// TokenId returns copy of the token id if the field is set.
// Otherwise it returns nil.
func (op *propertiesOperation) TokenId() (*big.Int, bool) {
	if op.Id == nil {
		return nil, false
	}
	return new(big.Int).Set(op.Id), true
}

type transferFromOperation struct {
	transferOperation
	FromAddress common.Address
}

// NewTransferFromOperation creates a token transfer operation.
// Standard of the token is selected using the standard parameter.
// For WRC-20 tokens the value parameter is value itself. For WRC-721 tokens the value parameter is a token id.
func NewTransferFromOperation(standard Std, from common.Address, to common.Address, value *big.Int) (TransferFrom, error) {
	if from == (common.Address{}) {
		return nil, ErrNoFrom
	}
	transferOp, err := newTransferOperation(standard, to, value)
	if err != nil {
		return nil, err
	}
	return &transferFromOperation{
		transferOperation: *transferOp,
		FromAddress:       from,
	}, nil
}

// From returns copy of the from field
func (op *transferFromOperation) From() common.Address {
	// It's safe to return common.Address by value, cause it's an array
	return op.FromAddress
}

// Code returns op code of a balance of operation
func (op *transferFromOperation) OpCode() Code {
	return TransferFromCode
}

// UnmarshalBinary unmarshals a token transfer from operation from byte encoding
func (op *transferFromOperation) UnmarshalBinary(b []byte) error {
	return rlpDecode(b, op)
}

// MarshalBinary marshals a token transfer from operation to byte encoding
func (op *transferFromOperation) MarshalBinary() ([]byte, error) {
	return rlpEncode(op)
}
