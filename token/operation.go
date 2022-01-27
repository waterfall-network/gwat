package token

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

var (
	ErrNoTokenSupply = errors.New("token supply is required")
	ErrNoName        = errors.New("token name is required")
	ErrNoSymbol      = errors.New("token symbol is required")
	ErrNoBaseURI     = errors.New("token baseURI is required")
	ErrNoAddress     = errors.New("token address is required")
)

type operation struct {
	standard Std
}

func (op *operation) Standard() Std {
	return op.standard
}

type CreateOperation interface {
	Operation
	Name() []byte
	Symbol() []byte
	// That's not required arguments
	// WRC-20 arguments
	Decimals() (uint8, bool)
	TotalSupply() (*big.Int, bool)
	// WRC-721 arguments
	BaseURI() ([]byte, bool)
}

type createOperation struct {
	operation
	name        []byte
	symbol      []byte
	decimals    *uint8
	totalSupply *big.Int
	baseURI     []byte
}

func NewWrc20CreateOperation(name []byte, symbol []byte, decimals uint8, totalSupply *big.Int) (CreateOperation, error) {
	if len(name) == 0 {
		return nil, ErrNoName
	}
	if len(symbol) == 0 {
		return nil, ErrNoSymbol
	}
	if totalSupply == nil {
		return nil, ErrNoTokenSupply
	}

	return &createOperation{
		operation: operation{
			standard: StdWRC20,
		},
		name:        name,
		symbol:      symbol,
		decimals:    &decimals,
		totalSupply: totalSupply,
	}, nil
}

func NewWrc721CreateOperation(name []byte, symbol []byte, baseURI []byte) (CreateOperation, error) {
	if len(name) == 0 {
		return nil, ErrNoName
	}
	if len(symbol) == 0 {
		return nil, ErrNoSymbol
	}
	if len(baseURI) == 0 {
		return nil, ErrNoBaseURI
	}

	return &createOperation{
		operation: operation{
			standard: StdWRC721,
		},
		name:    name,
		symbol:  symbol,
		baseURI: baseURI,
	}, nil
}

func (op *createOperation) OpCode() OpCode {
	return OpCreate
}

func (op *createOperation) Address() common.Address {
	return common.Address{}
}

func (op *createOperation) Name() []byte {
	return makeCopy(op.name)
}

func (op *createOperation) Symbol() []byte {
	return makeCopy(op.name)
}

func (op *createOperation) Decimals() (uint8, bool) {
	if op.decimals == nil {
		return 0, false
	}
	return *op.decimals, true
}

func (op *createOperation) TotalSupply() (*big.Int, bool) {
	if op.totalSupply == nil {
		return nil, false
	}
	return new(big.Int).Set(op.totalSupply), true
}

func (op *createOperation) BaseURI() ([]byte, bool) {
	if op.baseURI == nil {
		return nil, false
	}
	return makeCopy(op.baseURI), true
}

func makeCopy(src []byte) []byte {
	dst := make([]byte, len(src))
	copy(dst, src)
	return dst
}

type addresser interface {
	Address() common.Address
}

type PropertiesOperation interface {
	Operation
	addresser
	TokenId() (*big.Int, bool)
}

type addressOperation struct {
	address common.Address
}

func (a *addressOperation) Address() common.Address {
	// It's safe to return common.Address by value, cause it's an array
	return a.address
}

type propertiesOperation struct {
	operation
	addressOperation
	tokenId *big.Int
}

func NewPropertiesOperation(address common.Address, tokenId *big.Int) (PropertiesOperation, error) {
	if address == (common.Address{}) {
		return nil, ErrNoAddress
	}
	var standard Std = StdWRC20
	if tokenId != nil {
		standard = StdWRC721
	}
	return &propertiesOperation{
		operation: operation{
			standard: standard,
		},
		addressOperation: addressOperation{
			address: address,
		},
		tokenId: tokenId,
	}, nil
}

func (op *propertiesOperation) OpCode() OpCode {
	return OpProperties
}

func (op *propertiesOperation) TokenId() (*big.Int, bool) {
	if op.tokenId == nil {
		return nil, false
	}
	return new(big.Int).Set(op.tokenId), true
}
