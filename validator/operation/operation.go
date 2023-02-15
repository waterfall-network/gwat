package operation

import (
	"encoding"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
)

// Validator operation code
type Code byte

// Token operation codes use invalid op codes of EVM instructions to prevent clashes.
const (
	DepositCode     = 0x01
	ActivationCode  = 0x02
	RequestExitCode = 0x03
	ExitCode        = 0x04
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

// IsValidatorOp returns true if tx is validator operation
func IsValidatorOp(addrTo *common.Address) bool {
	if addrTo == nil {
		return false
	}
	//todo get the address from genesis config
	validatorsStateAddress := common.HexToAddress("0x1111111111111111111111111111111111111111")
	return *addrTo == validatorsStateAddress
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
	// todo implement
	//case ActivationCode:
	//	op = &ActivationOperation{}
	// todo implement
	//case RequestExitCode:
	//	op = &RequestExitOperation{}
	// todo implement
	//case ExitCode:
	//	op = &ExitOperation{}

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

		// todo implement
		//case *activationCodeOperation:
		//	buf[1] = ActivationCode
		// todo implement
		//case *requestExitOperation:
		//	buf[1] = RequestExitCode
		// todo implement
		//case *exitCodeOperation:
		//	buf[1] = ExitCode
	}

	buf = append(buf, b...)
	return buf, nil
}
