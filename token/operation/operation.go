package operation

// Marshaling and unmarshaling of operations in the package is impelemented using Ethereum rlp encoding.
// All marshal and unmarshal methods of operations suppose that an encoding prefix has already handled.
// Byte encoding of the operation should be given to the methods without the prefix.
//
// The operations are implement using a philosophy of immutable data structures. Every method that returns
// a data field of an operation always make its copy before returning. That prevents situations when the
// operation structure can be mutated accidentally.

import (
	"encoding"
)

const (
	DefaultDecimals = 18
)

// Token standard
type Std uint16

const (
	StdWRC20  = 20
	StdWRC721 = 721
)

// Token operation code
type Code byte

// Token operation codes use invalid op codes of EVM instructions to prevent clashes.
const (
	Create              = 0x0C
	Approve             = 0x0D
	Transfer            = 0x1E
	TransferFrom        = 0x1F
	Properties          = 0x21
	BalanceOf           = 0x22
	Allowance           = 0x23
	IsApprovedForAll    = 0x24
	SetApprovalForAll   = 0x25
	Mint                = 0x26
	Burn                = 0x27
	TokenOfOwnerByIndex = 0x28
	SafeTransferFrom    = 0x29
)

// Prefix for the encoded data field of a token operation
const (
	Prefix = 0xF3
)

// Operation is a token operation
// Every specific operation should implement this interface
type Operation interface {
	OpCode() Code
	Standard() Std

	encoding.BinaryUnmarshaler
	encoding.BinaryMarshaler
}

// GetOpCode gets op code of an encoded token operation
// It also checks the encoding for length and prefix
func GetOpCode(b []byte) (Code, error) {
	if len(b) < 2 {
		return 0, ErrRawDataShort
	}

	prefix := b[0]
	if prefix != Prefix {
		return 0, ErrPrefixNotValid
	}

	return Code(b[1]), nil
}

// DecodeBytes decodes an encoded token operation
// It does same checks as GetOpCode.
// Returns the decoded operation as Operation interface.
func DecodeBytes(b []byte) (Operation, error) {
	opCode, err := GetOpCode(b)
	if err != nil {
		return nil, err
	}

	var op Operation
	switch opCode {
	case Create:
		op = &createOperation{}
	case Approve:
		op = &approveOperation{}
	case Transfer:
		op = &transferOperation{}
	case TransferFrom:
		op = &transferFromOperation{}
	case Properties:
		op = &propertiesOperation{}
	case BalanceOf:
		op = &balanceOfOperation{}
	case Allowance:
		op = &allowanceOperation{}
	case IsApprovedForAll:
		op = &isApprovedForAllOperation{}
	case SetApprovalForAll:
		op = &setApprovalForAllOperation{}
	case Mint:
		op = &mintOperation{}
	case Burn:
		op = &burnOperation{}
	case TokenOfOwnerByIndex:
		op = &tokenOfOwnerByIndexOperation{}
	case SafeTransferFrom:
		op = &safeTransferFromOperation{}
	default:
		return nil, ErrOpNotValid
	}

	err = op.UnmarshalBinary(b[2:])
	return op, err
}

// EncodeToBytes encodes a token operation
// Returns byte representation of the encoded operation.
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
		buf[1] = Create
	case *approveOperation:
		buf[1] = Approve
	case *transferOperation:
		buf[1] = Transfer
	case *transferFromOperation:
		buf[1] = TransferFrom
	case *propertiesOperation:
		buf[1] = Properties
	case *balanceOfOperation:
		buf[1] = BalanceOf
	case *allowanceOperation:
		buf[1] = Allowance
	case *isApprovedForAllOperation:
		buf[1] = IsApprovedForAll
	case *setApprovalForAllOperation:
		buf[1] = SetApprovalForAll
	case *mintOperation:
		buf[1] = Mint
	case *burnOperation:
		buf[1] = Burn
	case *tokenOfOwnerByIndexOperation:
		buf[1] = TokenOfOwnerByIndex
	case *safeTransferFromOperation:
		buf[1] = SafeTransferFrom
	}

	buf = append(buf, b...)
	return buf, nil
}
