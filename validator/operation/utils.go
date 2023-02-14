package operation

import (
	"math/big"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
)

type opData struct {
	common.Address
	TokenId    *big.Int
	Owner      common.Address
	Spender    common.Address
	Operator   common.Address
	From       common.Address
	To         common.Address
	NewCost    *big.Int
	Value      *big.Int
	Index      *big.Int
	IsApproved bool
	Data       []byte `rlp:"tail"`
}

type valueOperation struct {
	TokenValue *big.Int
}

// Value returns copy of the value field
func (op *valueOperation) Value() *big.Int {
	return new(big.Int).Set(op.TokenValue)
}

type toOperation struct {
	ToAddress common.Address
}

// To returns copy of the to address field
func (op *toOperation) To() common.Address {
	// It's safe to return common.Address by value, cause it's an array
	return op.ToAddress
}

type operatorOperation struct {
	OperatorAddress common.Address
}

// Operator returns copy of the operator address field
func (op *operatorOperation) Operator() common.Address {
	// It's safe to return common.Address by value, cause it's an array
	return op.OperatorAddress
}

type addresser interface {
	Address() common.Address
}

type addressOperation struct {
	TokenAddress common.Address
}

// Address returns copy of the address field
func (a *addressOperation) Address() common.Address {
	// It's safe to return common.Address by value, cause it's an array
	return a.TokenAddress
}

func makeCopy(src []byte) []byte {
	dst := make([]byte, len(src))
	copy(dst, src)
	return dst
}
