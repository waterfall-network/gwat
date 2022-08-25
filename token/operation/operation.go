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
	"encoding/binary"
)

const (
	DefaultDecimals = 18
)

// Token standard
type Std uint16

func (s Std) MarshalBinary() ([]byte, error) {
	buf := make([]byte, 2)
	binary.BigEndian.PutUint16(buf, uint16(s))

	return buf, nil
}

func (s *Std) UnmarshalBinary(buf []byte) error {
	*s = Std(binary.BigEndian.Uint16(buf))

	return nil
}

const (
	StdWRC20  = 20
	StdWRC721 = 721
)

// Token operation code
type Code byte

// Token operation codes use invalid op codes of EVM instructions to prevent clashes.
const (
	CreateCode              = 0x0C
	ApproveCode             = 0x0D
	TransferCode            = 0x1E
	TransferFromCode        = 0x1F
	PropertiesCode          = 0x21
	BalanceOfCode           = 0x22
	AllowanceCode           = 0x23
	IsApprovedForAllCode    = 0x24
	SetApprovalForAllCode   = 0x25
	MintCode                = 0x26
	BurnCode                = 0x27
	TokenOfOwnerByIndexCode = 0x28
	SafeTransferFromCode    = 0x29
	SetPriceCode            = 0x2a
	BuyCode                 = 0x2b
	CostCode                = 0x2c
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
	case CreateCode:
		op = &createOperation{}
	case ApproveCode:
		op = &approveOperation{}
	case TransferCode:
		op = &transferOperation{}
	case TransferFromCode:
		op = &transferFromOperation{}
	case PropertiesCode:
		op = &propertiesOperation{}
	case BalanceOfCode:
		op = &balanceOfOperation{}
	case AllowanceCode:
		op = &allowanceOperation{}
	case IsApprovedForAllCode:
		op = &isApprovedForAllOperation{}
	case SetApprovalForAllCode:
		op = &setApprovalForAllOperation{}
	case MintCode:
		op = &mintOperation{}
	case BurnCode:
		op = &burnOperation{}
	case TokenOfOwnerByIndexCode:
		op = &tokenOfOwnerByIndexOperation{}
	case SafeTransferFromCode:
		op = &safeTransferFromOperation{}
	case SetPriceCode:
		op = &setPriceOperation{}
	case BuyCode:
		op = &buyOperation{}
	case CostCode:
		op = &costOperation{}
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
		buf[1] = CreateCode
	case *approveOperation:
		buf[1] = ApproveCode
	case *transferOperation:
		buf[1] = TransferCode
	case *transferFromOperation:
		buf[1] = TransferFromCode
	case *propertiesOperation:
		buf[1] = PropertiesCode
	case *balanceOfOperation:
		buf[1] = BalanceOfCode
	case *allowanceOperation:
		buf[1] = AllowanceCode
	case *isApprovedForAllOperation:
		buf[1] = IsApprovedForAllCode
	case *setApprovalForAllOperation:
		buf[1] = SetApprovalForAllCode
	case *mintOperation:
		buf[1] = MintCode
	case *burnOperation:
		buf[1] = BurnCode
	case *tokenOfOwnerByIndexOperation:
		buf[1] = TokenOfOwnerByIndexCode
	case *safeTransferFromOperation:
		buf[1] = SafeTransferFromCode
	case *setPriceOperation:
		buf[1] = SetPriceCode
	case *buyOperation:
		buf[1] = BuyCode
	case *costOperation:
		buf[1] = CostCode
	}

	buf = append(buf, b...)
	return buf, nil
}
