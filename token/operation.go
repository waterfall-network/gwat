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
	ErrNoSpender     = errors.New("spender address is required")
	ErrNoOperator    = errors.New("operator address is required")
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

type ownerOperation struct {
	owner common.Address
}

func (op *ownerOperation) Owner() common.Address {
	// It's safe to return common.Address by value, cause it's an array
	return op.owner
}

type balanceOfOperation struct {
	addressOperation
	ownerOperation
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
		ownerOperation: ownerOperation{
			owner: owner,
		},
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

type TransferOperation interface {
	Operation
	addresser
	To() common.Address
	Value() *big.Int
}

type valueOperation struct {
	value *big.Int
}

func (op *valueOperation) Value() *big.Int {
	return new(big.Int).Set(op.value)
}

type transferOperation struct {
	operation
	addressOperation
	valueOperation
	to common.Address
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
		to: to,
		valueOperation: valueOperation{
			value: value,
		},
	}, nil
}

func (op *transferOperation) OpCode() OpCode {
	return OpTransfer
}

func (op *transferOperation) To() common.Address {
	// It's safe to return common.Address by value, cause it's an array
	return op.to
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

type ApproveOperation interface {
	Operation
	Spender() common.Address
	Value() *big.Int
}

type spenderOperation struct {
	spender common.Address
}

func (op *spenderOperation) Spender() common.Address {
	// It's safe to return common.Address by value, cause it's an array
	return op.spender
}

type approveOperation struct {
	operation
	addressOperation
	valueOperation
	spenderOperation
}

func NewApproveOperation(address common.Address, spender common.Address, value *big.Int) (ApproveOperation, error) {
	if address == (common.Address{}) {
		return nil, ErrNoAddress
	}
	if spender == (common.Address{}) {
		return nil, ErrNoSpender
	}
	if value == nil {
		return nil, ErrNoValue
	}
	return &approveOperation{
		operation: operation{
			standard: StdWRC20,
		},
		addressOperation: addressOperation{
			address: address,
		},
		valueOperation: valueOperation{
			value: value,
		},
		spenderOperation: spenderOperation{
			spender: spender,
		},
	}, nil
}

func (op *approveOperation) OpCode() OpCode {
	return OpApprove
}

type AllowanceOperation interface {
	Operation
	addresser
	Owner() common.Address
	Spender() common.Address
}

type allowanceOperation struct {
	operation
	addressOperation
	ownerOperation
	spenderOperation
}

func NewAllowanceOperation(address common.Address, owner common.Address, spender common.Address) (AllowanceOperation, error) {
	if address == (common.Address{}) {
		return nil, ErrNoAddress
	}
	if spender == (common.Address{}) {
		return nil, ErrNoSpender
	}
	if owner == (common.Address{}) {
		return nil, ErrNoOwner
	}
	return &allowanceOperation{
		operation: operation{
			standard: StdWRC20,
		},
		addressOperation: addressOperation{
			address: address,
		},
		ownerOperation: ownerOperation{
			owner: owner,
		},
		spenderOperation: spenderOperation{
			spender: spender,
		},
	}, nil
}

func (op *allowanceOperation) OpCode() OpCode {
	return OpAllowance
}

type IsApprovedForAllOperation interface {
	Operation
	addresser
	Owner() common.Address
	Operator() common.Address
}

type operatorOperation struct {
	operator common.Address
}

func (op *operatorOperation) Operator() common.Address {
	// It's safe to return common.Address by value, cause it's an array
	return op.operator
}

type isApprovedForAllOperation struct {
	operation
	addressOperation
	ownerOperation
	operatorOperation
}

func NewIsApprovedForAllOperation(address common.Address, owner common.Address, operator common.Address) (IsApprovedForAllOperation, error) {
	if address == (common.Address{}) {
		return nil, ErrNoAddress
	}
	if owner == (common.Address{}) {
		return nil, ErrNoOwner
	}
	if operator == (common.Address{}) {
		return nil, ErrNoOperator
	}
	return &isApprovedForAllOperation{
		operation: operation{
			standard: StdWRC721,
		},
		addressOperation: addressOperation{
			address: address,
		},
		ownerOperation: ownerOperation{
			owner: owner,
		},
		operatorOperation: operatorOperation{
			operator: operator,
		},
	}, nil
}

func (op *isApprovedForAllOperation) OpCode() OpCode {
	return OpIsApprovedForAll
}

type SetApprovalForAllOperation interface {
	Operation
	addresser
	Operator() common.Address
	IsApproved() bool
}

type setApprovalForAllOperation struct {
	operation
	addressOperation
	operatorOperation
	isApproved bool
}

func NewSetApprovalForAllOperation(address common.Address, operator common.Address, isApproved bool) (SetApprovalForAllOperation, error) {
	if address == (common.Address{}) {
		return nil, ErrNoAddress
	}
	if operator == (common.Address{}) {
		return nil, ErrNoOperator
	}
	return &setApprovalForAllOperation{
		operation: operation{
			standard: StdWRC721,
		},
		addressOperation: addressOperation{
			address: address,
		},
		operatorOperation: operatorOperation{
			operator: operator,
		},
		isApproved: isApproved,
	}, nil
}

func (op *setApprovalForAllOperation) OpCode() OpCode {
	return OpSetApprovalForAll
}

func (op *setApprovalForAllOperation) IsApproved() bool {
	return op.isApproved
}
