package operation

import "encoding"

// Validator operation code
type Code byte

// Token operation codes use invalid op codes of EVM instructions to prevent clashes.
const (
	DepositCode       = 0x01
	ActivateCode      = 0x02
	ExitCode          = 0x03
	DeactivateCode    = 0x04
	UpdateBalanceCode = 0x05
	WithdrawalCode    = 0x06
	DelegateStakeCode = 0x20
)

// Prefix for the encoded data field of a validator operation
const (
	Prefix = 0xF4
)

// Operation is a validator operation
// Every specific operation should implement this interface
type Operation interface {
	OpCode() Code

	encoding.BinaryUnmarshaler
	encoding.BinaryMarshaler
}

// GetOpCode gets op code of an encoded validator operation
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

// DecodeBytes decodes an encoded validator operation
// It does same checks as GetOpCode.
// Returns the decoded operation as Operation interface.
func DecodeBytes(b []byte) (Operation, error) {
	opCode, err := GetOpCode(b)
	if err != nil {
		return nil, err
	}

	var op Operation
	switch opCode {
	case DepositCode:
		op = &depositOperation{}
	case ActivateCode:
		op = &validatorSyncOperation{}
	case DeactivateCode:
		op = &validatorSyncOperation{}
	case UpdateBalanceCode:
		op = &validatorSyncOperation{}
	case ExitCode:
		op = &exitOperation{}
	case WithdrawalCode:
		op = &withdrawalOperation{}
	case DelegateStakeCode:
		op = &delegateStakeOperation{}
	default:
		return nil, ErrOpNotValid
	}

	err = op.UnmarshalBinary(b[2:])
	return op, err
}

// EncodeToBytes encodes a validator operation
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
	case *depositOperation:
		buf[1] = DepositCode
	case *validatorSyncOperation:
		vs := new(validatorSyncOperation)
		err := vs.UnmarshalBinary(b)
		if err != nil {
			return nil, err
		}
		buf[1] = byte(vs.OpCode())
	case *exitOperation:
		buf[1] = ExitCode
	case *withdrawalOperation:
		buf[1] = WithdrawalCode
	case *delegateStakeOperation:
		buf[1] = DelegateStakeCode
	}

	buf = append(buf, b...)
	return buf, nil
}
