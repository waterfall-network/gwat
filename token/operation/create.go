package operation

import (
	"github.com/waterfall-foundation/gwat/common"
	"github.com/waterfall-foundation/gwat/rlp"
	"math/big"
)

type createOperation struct {
	operation
	name        []byte
	symbol      []byte
	decimals    uint8
	totalSupply *big.Int
	baseURI     []byte
	percentFee  uint8
}

func (op *createOperation) init(std Std, name []byte, symbol []byte, decimals, percentFee *uint8, totalSupply *big.Int, baseURI []byte) error {
	if len(name) == 0 {
		return ErrNoName
	}
	if len(symbol) == 0 {
		return ErrNoSymbol
	}

	switch std {
	case StdWRC20:
		if totalSupply == nil {
			return ErrNoTokenSupply
		}
		if decimals == nil {
			op.decimals = DefaultDecimals
		} else {
			op.decimals = *decimals
		}
		op.totalSupply = totalSupply
	case StdWRC721:
		if percentFee != nil {
			op.percentFee = *percentFee
		}

		if len(baseURI) == 0 {
			return ErrNoBaseURI
		}
		op.baseURI = baseURI
	default:
		return ErrStandardNotValid
	}

	op.Std = std
	op.name = name
	op.symbol = symbol

	return nil
}

// NewWrc20CreateOperation creates an operation for creating WRC-20 token
// It sets Standard of the operation to StdWRC20 and all other WRC-20 related fields
func NewWrc20CreateOperation(name []byte, symbol []byte, decimals *uint8, totalSupply *big.Int) (Create, error) {
	op := createOperation{}
	if err := op.init(StdWRC20, name, symbol, decimals, nil, totalSupply, nil); err != nil {
		return nil, err
	}
	return &op, nil
}

// NewWRC721CreateOperation creates an operation for creating WRC-721 token
// It sets Standard of the operation to StdWRC721 and all other WRC-721 related fields
func NewWrc721CreateOperation(name []byte, symbol []byte, baseURI []byte, percentFee *uint8) (Create, error) {
	op := createOperation{}
	if err := op.init(StdWRC721, name, symbol, nil, percentFee, nil, baseURI); err != nil {
		return nil, err
	}
	return &op, nil
}

type createOpData struct {
	Std
	Name        []byte
	Symbol      []byte
	TotalSupply *big.Int `rlp:"nil"`
	Decimals    *uint8   `rlp:"nil"`
	BaseURI     []byte   `rlp:"optional"`
	PercentFee  *uint8   `rlp:"optional"`
}

// UnmarshalBinary unmarshals a create operation from byte encoding
func (op *createOperation) UnmarshalBinary(b []byte) error {
	opData := createOpData{}
	if err := rlp.DecodeBytes(b, &opData); err != nil {
		return err
	}

	return op.init(opData.Std, opData.Name, opData.Symbol, opData.Decimals, opData.PercentFee, opData.TotalSupply, opData.BaseURI)
}

// MarshalBinary marshals a create operation to byte encoding
func (op *createOperation) MarshalBinary() ([]byte, error) {
	opData := createOpData{}
	opData.Std = op.Std
	opData.Name = op.name
	opData.Symbol = op.symbol
	opData.TotalSupply = op.totalSupply
	opData.Decimals = &op.decimals
	opData.BaseURI = op.baseURI
	opData.PercentFee = &op.percentFee

	return rlp.EncodeToBytes(&opData)
}

// Code returns op code of a create operation
func (op *createOperation) OpCode() Code {
	return CreateCode
}

// Code always returns an empty address
// It's just a stub for the Operation interface.
func (op *createOperation) Address() common.Address {
	return common.Address{}
}

// Name returns copy of the name field
func (op *createOperation) Name() []byte {
	return makeCopy(op.name)
}

// Symbol returns copy of the symbol field
func (op *createOperation) Symbol() []byte {
	return makeCopy(op.symbol)
}

// Decimals returns copy of the decimals field
func (op *createOperation) Decimals() uint8 {
	return op.decimals
}

// TotalSupply returns copy of the total supply field
func (op *createOperation) TotalSupply() (*big.Int, bool) {
	if op.totalSupply == nil {
		return nil, false
	}
	return new(big.Int).Set(op.totalSupply), true
}

// BaseURI returns copy of the base uri field if the field is set
// If it isn't set the method returns nil byte slice.
func (op *createOperation) BaseURI() ([]byte, bool) {
	if op.baseURI == nil {
		return nil, false
	}
	return makeCopy(op.baseURI), true
}

// PercentFee returns copy of the percentFee field
func (op *createOperation) PercentFee() uint8 {
	return op.percentFee
}
