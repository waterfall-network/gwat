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
	ErrNoOwner       = errors.New("token owner address is required")
	ErrNoValue       = errors.New("value is required")
	ErrNoTo          = errors.New("to address is required")
	ErrNoFrom        = errors.New("from address is required")
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

type BalanceOfOperation interface {
	Operation
	addresser
	Owner() common.Address
}

type balanceOfOperation struct {
	addressOperation
	owner common.Address
}

func NewBalanceOfOperation(address common.Address, owner common.Address) (BalanceOfOperation, error) {
	if address == (common.Address{}) {
		return nil, ErrNoAddress
	}
	if owner == (common.Address{}) {
		return nil, ErrNoOwner
	}
	return &balanceOfOperation{
		addressOperation: addressOperation{
			address: address,
		},
		owner: owner,
	}, nil
}

func (op *balanceOfOperation) OpCode() OpCode {
	return OpBalanceOf
}

// Standard method is just a stub to implement Operation interface.
// Always returns 0.
func (op *balanceOfOperation) Standard() Std {
	return 0
}

func (op *balanceOfOperation) Owner() common.Address {
	// It's safe to return common.Address by value, cause it's an array
	return op.owner
}

type TransferOperation interface {
	Operation
	addresser
	To() common.Address
	Value() *big.Int
}

type transferOperation struct {
	operation
	addressOperation
	to    common.Address
	value *big.Int
}

func NewTransferOperation(address common.Address, to common.Address, value *big.Int) (TransferOperation, error) {
	return newTransferOperation(StdWRC20, address, to, value)
}

func newTransferOperation(standard Std, address common.Address, to common.Address, value *big.Int) (*transferOperation, error) {
	if address == (common.Address{}) {
		return nil, ErrNoAddress
	}
	if to == (common.Address{}) {
		return nil, ErrNoTo
	}
	if value == nil {
		return nil, ErrNoValue
	}
	return &transferOperation{
		operation: operation{
			standard: standard,
		},
		addressOperation: addressOperation{
			address: address,
		},
		to:    to,
		value: value,
	}, nil
}

func (op *transferOperation) OpCode() OpCode {
	return OpTransfer
}

func (op *transferOperation) To() common.Address {
	// It's safe to return common.Address by value, cause it's an array
	return op.to
}

func (op *transferOperation) Value() *big.Int {
	return new(big.Int).Set(op.value)
}

type TransferFromOperation interface {
	TransferOperation
	From() common.Address
}

type transferFromOperation struct {
	transferOperation
	from common.Address
}

func NewTransferFromOperation(standard Std, address common.Address, from common.Address, to common.Address, value *big.Int) (TransferFromOperation, error) {
	if from == (common.Address{}) {
		return nil, ErrNoFrom
	}
	transferOp, err := newTransferOperation(standard, address, to, value)
	if err != nil {
		return nil, err
	}
	return &transferFromOperation{
		transferOperation: *transferOp,
		from:              from,
	}, nil
}

func (op *transferFromOperation) From() common.Address {
	// It's safe to return common.Address by value, cause it's an array
	return op.from
}

type SafeTransferFromOperation interface {
	TransferFromOperation
	Data() ([]byte, bool)
}

type safeTransferFromOperation struct {
	transferFromOperation
	data []byte
}

func NewSafeTransferFromOperation(address common.Address, from common.Address, to common.Address, value *big.Int, data []byte) (SafeTransferFromOperation, error) {
	transferOp, err := NewTransferFromOperation(StdWRC721, address, from, to, value)
	if err != nil {
		return nil, err
	}
	return &safeTransferFromOperation{
		transferFromOperation: *transferOp.(*transferFromOperation),
		data:                  data,
	}, nil
}

func (op *safeTransferFromOperation) Data() ([]byte, bool) {
	if len(op.data) == 0 {
		return nil, false
	}
	return makeCopy(op.data), true
}
