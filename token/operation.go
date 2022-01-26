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
)

type commonOpArgs struct {
	standard Std
}

func (a *commonOpArgs) Standard() Std {
	return a.standard
}

type CreateOperation struct {
	commonOpArgs
	Name   []byte
	Symbol []byte
	// That's not required arguments
	// WRC-20 arguments
	Decimals    *uint8
	TotalSupply *big.Int
	// WRC-721 arguments
	BaseURI []byte
}

func NewWrc20CreateOperation(name []byte, symbol []byte, decimals uint8, totalSupply *big.Int) (Operation, error) {
	if len(name) == 0 {
		return nil, ErrNoName
	}
	if len(symbol) == 0 {
		return nil, ErrNoSymbol
	}
	if totalSupply == nil {
		return nil, ErrNoTokenSupply
	}

	return &CreateOperation{
		commonOpArgs: commonOpArgs{
			standard: StdWRC20,
		},
		Name:        name,
		Symbol:      symbol,
		Decimals:    &decimals,
		TotalSupply: totalSupply,
	}, nil
}

func NewWrc721CreateOperation(name []byte, symbol []byte, baseURI []byte) (Operation, error) {
	if len(name) == 0 {
		return nil, ErrNoName
	}
	if len(symbol) == 0 {
		return nil, ErrNoSymbol
	}
	if len(baseURI) == 0 {
		return nil, ErrNoBaseURI
	}

	return &CreateOperation{
		commonOpArgs: commonOpArgs{
			standard: StdWRC20,
		},
		Name:    name,
		Symbol:  symbol,
		BaseURI: baseURI,
	}, nil
}

func (op *CreateOperation) OpCode() OpCode {
	return OpCreate
}

func (op *CreateOperation) Address() common.Address {
	return common.Address{}
}
