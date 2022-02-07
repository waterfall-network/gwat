package token

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
)

var (
	ErrNoTokenSupply    = errors.New("token supply is required")
	ErrNoName           = errors.New("token name is required")
	ErrNoSymbol         = errors.New("token symbol is required")
	ErrNoBaseURI        = errors.New("token baseURI is required")
	ErrNoAddress        = errors.New("token address is required")
	ErrNoOwner          = errors.New("token owner address is required")
	ErrNoValue          = errors.New("value is required")
	ErrNoTo             = errors.New("to address is required")
	ErrNoFrom           = errors.New("from address is required")
	ErrNoSpender        = errors.New("spender address is required")
	ErrNoOperator       = errors.New("operator address is required")
	ErrNoTokenId        = errors.New("token id is required")
	ErrStandardNotValid = errors.New("not valid value for token standard")
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

func (op *createOperation) init(std Std, name []byte, symbol []byte, decimals *uint8, totalSupply *big.Int, baseURI []byte) error {
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
		op.decimals = decimals
		op.totalSupply = totalSupply
	case StdWRC721:
		if len(baseURI) == 0 {
			return ErrNoBaseURI
		}
		op.baseURI = baseURI
	default:
		return ErrStandardNotValid
	}

	op.standard = std
	op.name = name
	op.symbol = symbol

	return nil
}

func NewWrc20CreateOperation(name []byte, symbol []byte, decimals *uint8, totalSupply *big.Int) (CreateOperation, error) {
	op := createOperation{}
	if err := op.init(StdWRC20, name, symbol, decimals, totalSupply, nil); err != nil {
		return nil, err
	}
	return &op, nil
}

func NewWrc721CreateOperation(name []byte, symbol []byte, baseURI []byte) (CreateOperation, error) {
	op := createOperation{}
	if err := op.init(StdWRC721, name, symbol, nil, nil, baseURI); err != nil {
		return nil, err
	}
	return &op, nil
}

func (op *createOperation) UnmarshalBinary(b []byte) error {
	opData := struct {
		Std
		Name        []byte
		Symbol      []byte
		TotalSupply *big.Int `rlp:"nil"`
		Decimals    *uint8   `rlp:"nil"`
		BaseURI     []byte   `rlp:"optional"`
	}{}

	if err := rlp.DecodeBytes(b, &opData); err != nil {
		return err
	}
	op.init(opData.Std, opData.Name, opData.Symbol, opData.Decimals, opData.TotalSupply, opData.BaseURI)
	return nil
}

func (op *createOperation) MarshalBinary() ([]byte, error) {
	return rlp.EncodeToBytes(op)
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

type toOperation struct {
	to common.Address
}

func (op *toOperation) To() common.Address {
	// It's safe to return common.Address by value, cause it's an array
	return op.to
}

type transferOperation struct {
	operation
	addressOperation
	valueOperation
	toOperation
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
		toOperation: toOperation{
			to: to,
		},
		valueOperation: valueOperation{
			value: value,
		},
	}, nil
}

func (op *transferOperation) OpCode() OpCode {
	return OpTransfer
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

type MintOperation interface {
	Operation
	addresser
	To() common.Address
	TokenId() *big.Int
	Metadata() ([]byte, bool)
}

type tokenIdOperation struct {
	tokenId *big.Int
}

func (op *tokenIdOperation) TokenId() *big.Int {
	return new(big.Int).Set(op.tokenId)
}

type mintOperation struct {
	operation
	addressOperation
	toOperation
	tokenIdOperation
	metadata []byte
}

func NewMintOperation(address common.Address, to common.Address, tokenId *big.Int, metadata []byte) (MintOperation, error) {
	if address == (common.Address{}) {
		return nil, ErrNoAddress
	}
	if to == (common.Address{}) {
		return nil, ErrNoTo
	}
	if tokenId == nil {
		return nil, ErrNoTokenId
	}
	return &mintOperation{
		operation: operation{
			standard: StdWRC721,
		},
		addressOperation: addressOperation{
			address: address,
		},
		toOperation: toOperation{
			to: to,
		},
		tokenIdOperation: tokenIdOperation{
			tokenId: tokenId,
		},
		metadata: metadata,
	}, nil
}

func (op *mintOperation) OpCode() OpCode {
	return OpMint
}

func (op *mintOperation) Metadata() ([]byte, bool) {
	if len(op.metadata) == 0 {
		return nil, false
	}
	return makeCopy(op.metadata), true
}

type BurnOperation interface {
	Operation
	addresser
	TokenId() *big.Int
}

type burnOperation struct {
	operation
	addressOperation
	tokenIdOperation
}

func NewBurnOperation(address common.Address, tokenId *big.Int) (BurnOperation, error) {
	if address == (common.Address{}) {
		return nil, ErrNoAddress
	}
	if tokenId == nil {
		return nil, ErrNoTokenId
	}
	return &burnOperation{
		operation: operation{
			standard: StdWRC721,
		},
		addressOperation: addressOperation{
			address: address,
		},
		tokenIdOperation: tokenIdOperation{
			tokenId: tokenId,
		},
	}, nil
}

func (op *burnOperation) OpCode() OpCode {
	return OpBurn
}

type TokenOfOwnerByIndexOperation interface {
	Operation
	addresser
	Owner() common.Address
	Index() *big.Int
}

type tokenOfOwnerByIndexOperation struct {
	operation
	addressOperation
	ownerOperation
	index *big.Int
}

func NewTokenOfOwnerByIndexOperation(address common.Address, owner common.Address, index *big.Int) (TokenOfOwnerByIndexOperation, error) {
	if address == (common.Address{}) {
		return nil, ErrNoAddress
	}
	if owner == (common.Address{}) {
		return nil, ErrNoOwner
	}
	if index == nil {
		return nil, ErrNoIndex
	}
	return &tokenOfOwnerByIndexOperation{
		operation: operation{
			Std: StdWRC721,
		},
		addressOperation: addressOperation{
			address: address,
		},
		ownerOperation: ownerOperation{
			owner: owner,
		},
		index: index,
	}, nil
}

func (op *tokenOfOwnerByIndexOperation) OpCode() OpCode {
	return OpTokenOfOwnerByIndex
}

func (op *tokenOfOwnerByIndexOperation) Index() *big.Int {
	return new(big.Int).Set(op.index)
}
