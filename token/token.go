package token

import (
	"encoding"
	"errors"

	"github.com/ethereum/go-ethereum/common"
)

var (
	ErrPrefixNotValid = errors.New("not valid value for prefix")
	ErrRawDataShort   = errors.New("binary data for token operation is short")
	ErrOpNotValid     = errors.New("not valid op code for token operation")
)

// Token standard
type Std uint16

const (
	StdWRC20  = 20
	StdWRC721 = 721
)

// Token operation code
type OpCode byte

// Token operation codes use invalid op codes of EVM instructions to prevent clashes.
const (
	OpCreate              = 0x0C
	OpApprove             = 0x0D
	OpTransfer            = 0x1E
	OpTransferFrom        = 0x1F
	OpProperties          = 0x21
	OpBalanceOf           = 0x22
	OpAllowance           = 0x23
	OpIsApprovedForAll    = 0x24
	OpSetApprovalForAll   = 0x25
	OpMint                = 0x26
	OpBurn                = 0x27
	OpTokenOfOwnerByIndex = 0x28
	OpSafeTransferFrom    = 0x29
)

const (
	Prefix = 0xF3
)

type Operation interface {
	OpCode() OpCode
	Standard() Std
	Address() common.Address // Token address

	encoding.BinaryUnmarshaler
	encoding.BinaryMarshaler
}

func DecodeBytes(b []byte) (Operation, error) {
	if len(b) < 2 {
		return nil, ErrRawDataShort
	}

	prefix := b[0]
	if prefix != Prefix {
		return nil, ErrPrefixNotValid
	}

	var op Operation
	opCode := b[1]
	switch opCode {
	case OpCreate:
		op = &createOperation{}
	case OpApprove:
		op = &approveOperation{}
	case OpTransfer:
		op = &transferOperation{}
	case OpTransferFrom:
		op = &transferFromOperation{}
	case OpProperties:
		op = &propertiesOperation{}
	case OpBalanceOf:
		op = &balanceOfOperation{}
	case OpAllowance:
		op = &allowanceOperation{}
	case OpIsApprovedForAll:
		op = &isApprovedForAllOperation{}
	case OpSetApprovalForAll:
		op = &setApprovalForAllOperation{}
	case OpMint:
		op = &mintOperation{}
	case OpBurn:
		op = &burnOperation{}
	case OpTokenOfOwnerByIndex:
		op = &tokenOfOwnerByIndexOperation{}
	case OpSafeTransferFrom:
		op = &safeTransferFromOperation{}
	default:
		return nil, ErrOpNotValid
	}

	err := op.UnmarshalBinary(b[2:])
	return op, err
}

func EncodeToBytes(op Operation) ([]byte, error) {
	b, err := op.MarshalBinary()
	if err != nil {
		return nil, err
	}

	// len = 2 for prefix and op code
	buf := make([]byte, 2, len(b)+2)
	buf[0] = Prefix

	switch op.(type) {
	case *createOperation:
		buf[1] = OpCreate
	case *approveOperation:
		buf[1] = OpApprove
	case *transferOperation:
		buf[1] = OpTransfer
	case *transferFromOperation:
		buf[1] = OpTransferFrom
	case *propertiesOperation:
		buf[1] = OpProperties
	case *balanceOfOperation:
		buf[1] = OpBalanceOf
	case *allowanceOperation:
		buf[1] = OpAllowance
	case *isApprovedForAllOperation:
		buf[1] = OpIsApprovedForAll
	case *setApprovalForAllOperation:
		buf[1] = OpSetApprovalForAll
	case *mintOperation:
		buf[1] = OpMint
	case *burnOperation:
		buf[1] = OpBurn
	case *tokenOfOwnerByIndexOperation:
		buf[1] = OpTokenOfOwnerByIndex
	case *safeTransferFromOperation:
		buf[1] = OpSafeTransferFrom
	}

	buf = append(buf, b...)
	return buf, nil
}
