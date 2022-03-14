// Marshaling and unmarshaling of operations in the package is impelemented using Ethereum rlp encoding.
// All marshal and unmarshal methods of operations suppose that an encoding prefix has already handled.
// Byte encoding of the operation should be given to the methods without the prefix.
//
// The operations are implement using a philosophy of immutable data structures. Every method that returns
// a data field of an operation always make its copy before returning. That prevents situations when the
// operation structure can be mutated accidentally.
package token

import (
	"errors"
	"math/big"
	"reflect"
	"regexp"

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
	ErrNoIndex          = errors.New("token index is required")
	ErrStandardNotValid = errors.New("not valid value for token standard")
)

type operation struct {
	Std
}

// Standard returns token standard
func (op *operation) Standard() Std {
	return op.Std
}

// CreateOperation contatins all attributes for creating WRC-20 or WRC-721 token
// Methods for getting optional attributes also return boolean values which indicates if the attribute was set
type CreateOperation interface {
	Operation
	Name() []byte
	Symbol() []byte
	// That's not required arguments
	// WRC-20 arguments
	Decimals() uint8
	TotalSupply() (*big.Int, bool)
	// WRC-721 arguments
	BaseURI() ([]byte, bool)
}

type createOperation struct {
	operation
	name        []byte
	symbol      []byte
	decimals    uint8
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
		if decimals == nil {
			op.decimals = DefaultDecimals
		} else {
			op.decimals = *decimals
		}
		op.totalSupply = totalSupply
	case StdWRC721:
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
func NewWrc20CreateOperation(name []byte, symbol []byte, decimals *uint8, totalSupply *big.Int) (CreateOperation, error) {
	op := createOperation{}
	if err := op.init(StdWRC20, name, symbol, decimals, totalSupply, nil); err != nil {
		return nil, err
	}
	return &op, nil
}

// NewWRC721CreateOperation creates an operation for creating WRC-721 token
// It sets Standard of the operation to StdWRC721 and all other WRC-721 related fields
func NewWrc721CreateOperation(name []byte, symbol []byte, baseURI []byte) (CreateOperation, error) {
	op := createOperation{}
	if err := op.init(StdWRC721, name, symbol, nil, nil, baseURI); err != nil {
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
}

// UnmarshalBinary unmarshals a create operation from byte encoding
func (op *createOperation) UnmarshalBinary(b []byte) error {
	opData := createOpData{}
	if err := rlp.DecodeBytes(b, &opData); err != nil {
		return err
	}
	op.init(opData.Std, opData.Name, opData.Symbol, opData.Decimals, opData.TotalSupply, opData.BaseURI)
	return nil
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

	return rlp.EncodeToBytes(&opData)
}

// OpCode returns op code of a create operation
func (op *createOperation) OpCode() OpCode {
	return OpCreate
}

// OpCode always returns an empty address
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

func makeCopy(src []byte) []byte {
	dst := make([]byte, len(src))
	copy(dst, src)
	return dst
}

type addresser interface {
	Address() common.Address
}

// PropertiesOperation contatins attributes for a token properties call
type PropertiesOperation interface {
	Operation
	addresser
	TokenId() (*big.Int, bool)
}

type addressOperation struct {
	TokenAddress common.Address
}

// Address returns copy of the address field
func (a *addressOperation) Address() common.Address {
	// It's safe to return common.Address by value, cause it's an array
	return a.TokenAddress
}

type propertiesOperation struct {
	addressOperation
	Id *big.Int
}

// NewPropertiesOperation creates a token properties operation
// tokenId parameters isn't required.
func NewPropertiesOperation(address common.Address, tokenId *big.Int) (PropertiesOperation, error) {
	if address == (common.Address{}) {
		return nil, ErrNoAddress
	}
	return &propertiesOperation{
		addressOperation: addressOperation{
			TokenAddress: address,
		},
		Id: tokenId,
	}, nil
}

// Standard always returns zero value
// It's just a stub for the Operation interface.
func (op *propertiesOperation) Standard() Std {
	return 0
}

// UnmarshalBinary unmarshals a properties operation from byte encoding
func (op *propertiesOperation) UnmarshalBinary(b []byte) error {
	return rlpDecode(b, op)
}

// MarshalBinary marshals a properties operation to byte encoding
func (op *propertiesOperation) MarshalBinary() ([]byte, error) {
	return rlpEncode(op)
}

// OpCode returns op code of a create operation
func (op *propertiesOperation) OpCode() OpCode {
	return OpProperties
}

// TokenId returns copy of the token id if the field is set.
// Otherwise it returns nil.
func (op *propertiesOperation) TokenId() (*big.Int, bool) {
	if op.Id == nil {
		return nil, false
	}
	return new(big.Int).Set(op.Id), true
}

// BalanceOfOperation contains attrubutes for token balance of call
type BalanceOfOperation interface {
	Operation
	addresser
	Owner() common.Address
}

type ownerOperation struct {
	OwnerAddress common.Address
}

// Owner returns copy of the owner address field
func (op *ownerOperation) Owner() common.Address {
	// It's safe to return common.Address by value, cause it's an array
	return op.OwnerAddress
}

type balanceOfOperation struct {
	addressOperation
	ownerOperation
}

// NewBalanceOfOperation creates a balance of operation
func NewBalanceOfOperation(address common.Address, owner common.Address) (BalanceOfOperation, error) {
	if address == (common.Address{}) {
		return nil, ErrNoAddress
	}
	if owner == (common.Address{}) {
		return nil, ErrNoOwner
	}
	return &balanceOfOperation{
		addressOperation: addressOperation{
			TokenAddress: address,
		},
		ownerOperation: ownerOperation{
			OwnerAddress: owner,
		},
	}, nil
}

// OpCode returns op code of a balance of operation
func (op *balanceOfOperation) OpCode() OpCode {
	return OpBalanceOf
}

// Standard method is just a stub to implement Operation interface.
// Always returns 0.
func (op *balanceOfOperation) Standard() Std {
	return 0
}

// UnmarshalBinary unmarshals a balance of operation from byte encoding
func (op *balanceOfOperation) UnmarshalBinary(b []byte) error {
	return rlpDecode(b, op)
}

// MarshalBinary marshals a balance of operation to byte encoding
func (op *balanceOfOperation) MarshalBinary() ([]byte, error) {
	return rlpEncode(op)
}

// TransferOperation contains attributes for token transfer call
type TransferOperation interface {
	Operation
	To() common.Address
	Value() *big.Int
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

type transferOperation struct {
	operation
	valueOperation
	toOperation
}

// NewTransferOperation creates a token trasnsfer operation
// Only WRC-20 tokens support the operation so its Standard alwasys sets to StdWRC20.
func NewTransferOperation(to common.Address, value *big.Int) (TransferOperation, error) {
	return newTransferOperation(StdWRC20, to, value)
}

func newTransferOperation(standard Std, to common.Address, value *big.Int) (*transferOperation, error) {
	if to == (common.Address{}) {
		return nil, ErrNoTo
	}
	if value == nil {
		return nil, ErrNoValue
	}
	return &transferOperation{
		operation: operation{
			Std: standard,
		},
		toOperation: toOperation{
			ToAddress: to,
		},
		valueOperation: valueOperation{
			TokenValue: value,
		},
	}, nil
}

// OpCode returns op code of a balance of operation
func (op *transferOperation) OpCode() OpCode {
	return OpTransfer
}

// UnmarshalBinary unmarshals a token transfer operation from byte encoding
func (op *transferOperation) UnmarshalBinary(b []byte) error {
	return rlpDecode(b, op)
}

// MarshalBinary marshals a token transfer operation to byte encoding
func (op *transferOperation) MarshalBinary() ([]byte, error) {
	return rlpEncode(op)
}

// TransferFromOperation contains attrubute for WRC-20 or WRC-721 transfer from operation
type TransferFromOperation interface {
	TransferOperation
	From() common.Address
}

type transferFromOperation struct {
	transferOperation
	FromAddress common.Address
}

// NewTransferFromOperation creates a token transfer operation.
// Standard of the token is selected using the standard parameter.
// For WRC-20 tokens the value parameter is value itself. For WRC-721 tokens the value parameter is a token id.
func NewTransferFromOperation(standard Std, from common.Address, to common.Address, value *big.Int) (TransferFromOperation, error) {
	if from == (common.Address{}) {
		return nil, ErrNoFrom
	}
	transferOp, err := newTransferOperation(standard, to, value)
	if err != nil {
		return nil, err
	}
	return &transferFromOperation{
		transferOperation: *transferOp,
		FromAddress:       from,
	}, nil
}

// From returns copy of the from field
func (op *transferFromOperation) From() common.Address {
	// It's safe to return common.Address by value, cause it's an array
	return op.FromAddress
}

// UnmarshalBinary unmarshals a token transfer from operation from byte encoding
func (op *transferFromOperation) UnmarshalBinary(b []byte) error {
	return rlpDecode(b, op)
}

// MarshalBinary marshals a token transfer from operation to byte encoding
func (op *transferFromOperation) MarshalBinary() ([]byte, error) {
	return rlpEncode(op)
}

// SafeTransferFromOperation contains attributes for WRC-721 safe transfer from operation
type SafeTransferFromOperation interface {
	TransferFromOperation
	Data() ([]byte, bool)
}

type safeTransferFromOperation struct {
	transferFromOperation
	data []byte
}

// NewSafeTransferFromOperation creates a safe token transfer operation.
// The operation only supports WRC-721 tokens so its Standard field sets to StdWRC721.
func NewSafeTransferFromOperation(from common.Address, to common.Address, value *big.Int, data []byte) (SafeTransferFromOperation, error) {
	transferOp, err := NewTransferFromOperation(StdWRC721, from, to, value)
	if err != nil {
		return nil, err
	}
	return &safeTransferFromOperation{
		transferFromOperation: *transferOp.(*transferFromOperation),
		data:                  data,
	}, nil
}

// Data returns copy of the data bytes if the field is set.
// Otherwise it returns nil.
func (op *safeTransferFromOperation) Data() ([]byte, bool) {
	if len(op.data) == 0 {
		return nil, false
	}
	return makeCopy(op.data), true
}

// UnmarshalBinary unmarshals a token safe transfer from operation from byte encoding
func (op *safeTransferFromOperation) UnmarshalBinary(b []byte) error {
	return rlpDecode(b, op)
}

// MarshalBinary marshals a token safe transfer from operation to byte encoding
func (op *safeTransferFromOperation) MarshalBinary() ([]byte, error) {
	return rlpEncode(op)
}

// ApproveOperation contains attributes for a token approve operation
type ApproveOperation interface {
	Operation
	Spender() common.Address
	Value() *big.Int
}

type spenderOperation struct {
	SpenderAddress common.Address
}

// Spender returns copy of the spender address field
func (op *spenderOperation) Spender() common.Address {
	// It's safe to return common.Address by value, cause it's an array
	return op.SpenderAddress
}

type approveOperation struct {
	operation
	valueOperation
	spenderOperation
}

// NewApproveOperation creates a token approve operation.
// Same logic with standard parameter applies as with the transfer from factory function.
func NewApproveOperation(standard Std, spender common.Address, value *big.Int) (ApproveOperation, error) {
	if spender == (common.Address{}) {
		return nil, ErrNoSpender
	}
	if value == nil {
		return nil, ErrNoValue
	}
	return &approveOperation{
		operation: operation{
			Std: standard,
		},
		valueOperation: valueOperation{
			TokenValue: value,
		},
		spenderOperation: spenderOperation{
			SpenderAddress: spender,
		},
	}, nil
}

// OpCode returns op code of a balance of operation
func (op *approveOperation) OpCode() OpCode {
	return OpApprove
}

// UnmarshalBinary unmarshals a token approve operation from byte encoding
func (op *approveOperation) UnmarshalBinary(b []byte) error {
	return rlpDecode(b, op)
}

// MarshalBinary marshals a token approve operation to byte encoding
func (op *approveOperation) MarshalBinary() ([]byte, error) {
	return rlpEncode(op)
}

// AllowanceOperation contains attributes for an allowance operation
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

// NewAllowanceOperation creates a token allowance operation.
// The operation only supports WRC-20 tokens so its Standard field sets to StdWRC20.
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
			Std: StdWRC20,
		},
		addressOperation: addressOperation{
			TokenAddress: address,
		},
		ownerOperation: ownerOperation{
			OwnerAddress: owner,
		},
		spenderOperation: spenderOperation{
			SpenderAddress: spender,
		},
	}, nil
}

// OpCode returns op code of an allowance operation
func (op *allowanceOperation) OpCode() OpCode {
	return OpAllowance
}

// UnmarshalBinary unmarshals a token allowance operation from byte encoding
func (op *allowanceOperation) UnmarshalBinary(b []byte) error {
	return rlpDecode(b, op)
}

// MarshalBinary marshals a token allowance operation to byte encoding
func (op *allowanceOperation) MarshalBinary() ([]byte, error) {
	return rlpEncode(op)
}

// IsApprovedForAllOperation contains attributes for WRC-721 is approved for all operation
type IsApprovedForAllOperation interface {
	Operation
	addresser
	Owner() common.Address
	Operator() common.Address
}

type operatorOperation struct {
	OperatorAddress common.Address
}

// Operator returns copy of the operator address field
func (op *operatorOperation) Operator() common.Address {
	// It's safe to return common.Address by value, cause it's an array
	return op.OperatorAddress
}

type isApprovedForAllOperation struct {
	operation
	addressOperation
	ownerOperation
	operatorOperation
}

// NewIsApprovedForAllOperation creates an approved for all operation.
// The operation only supports WRC-721 tokens so its Standard field sets to StdWRC721.
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
			Std: StdWRC721,
		},
		addressOperation: addressOperation{
			TokenAddress: address,
		},
		ownerOperation: ownerOperation{
			OwnerAddress: owner,
		},
		operatorOperation: operatorOperation{
			OperatorAddress: operator,
		},
	}, nil
}

// OpCode returns op code of an opproved for all operation
func (op *isApprovedForAllOperation) OpCode() OpCode {
	return OpIsApprovedForAll
}

// UnmarshalBinary unmarshals a token allowance operation from byte encoding
func (op *isApprovedForAllOperation) UnmarshalBinary(b []byte) error {
	return rlpDecode(b, op)
}

// MarshalBinary marshals a token allowance operation to byte encoding
func (op *isApprovedForAllOperation) MarshalBinary() ([]byte, error) {
	return rlpEncode(op)
}

// SetApprovalForAllOperation contains attributes for set approval for all operation
type SetApprovalForAllOperation interface {
	Operation
	Operator() common.Address
	IsApproved() bool
}

type setApprovalForAllOperation struct {
	operation
	operatorOperation
	Approved bool
}

// NewSetApprovalForAllOperation creates a set approval for all operation.
// The operation only supports WRC-721 tokens so its Standard field sets to StdWRC721.
func NewSetApprovalForAllOperation(operator common.Address, isApproved bool) (SetApprovalForAllOperation, error) {
	if operator == (common.Address{}) {
		return nil, ErrNoOperator
	}
	return &setApprovalForAllOperation{
		operation: operation{
			Std: StdWRC721,
		},
		operatorOperation: operatorOperation{
			OperatorAddress: operator,
		},
		Approved: isApproved,
	}, nil
}

// OpCode returns op code of an set approval for all operation
func (op *setApprovalForAllOperation) OpCode() OpCode {
	return OpSetApprovalForAll
}

// Returns flag whether operations on NFT are approved or not
func (op *setApprovalForAllOperation) IsApproved() bool {
	return op.Approved
}

// UnmarshalBinary unmarshals a set approval for all operation from byte encoding
func (op *setApprovalForAllOperation) UnmarshalBinary(b []byte) error {
	return rlpDecode(b, op)
}

// MarshalBinary marshals a set approval for all operation to byte encoding
func (op *setApprovalForAllOperation) MarshalBinary() ([]byte, error) {
	return rlpEncode(op)
}

// MintOperation contains attributes for a mint operation
type MintOperation interface {
	Operation
	To() common.Address
	TokenId() *big.Int
	Metadata() ([]byte, bool)
}

type tokenIdOperation struct {
	Id *big.Int
}

// Returns copy of the token id field
func (op *tokenIdOperation) TokenId() *big.Int {
	return new(big.Int).Set(op.Id)
}

type mintOperation struct {
	operation
	toOperation
	tokenIdOperation
	TokenMetadata []byte
}

// NewMintOperation creates a mint operation.
// The operation only supports WRC-721 tokens so its Standard field sets to StdWRC721.
func NewMintOperation(to common.Address, tokenId *big.Int, metadata []byte) (MintOperation, error) {
	if to == (common.Address{}) {
		return nil, ErrNoTo
	}
	if tokenId == nil {
		return nil, ErrNoTokenId
	}
	return &mintOperation{
		operation: operation{
			Std: StdWRC721,
		},
		toOperation: toOperation{
			ToAddress: to,
		},
		tokenIdOperation: tokenIdOperation{
			Id: tokenId,
		},
		TokenMetadata: metadata,
	}, nil
}

// OpCode returns op code of a mint token operation
func (op *mintOperation) OpCode() OpCode {
	return OpMint
}

// Metadata returns copy of the metadata bytes if the field is set.
// Otherwise it returns nil.
func (op *mintOperation) Metadata() ([]byte, bool) {
	if len(op.TokenMetadata) == 0 {
		return nil, false
	}
	return makeCopy(op.TokenMetadata), true
}

// UnmarshalBinary unmarshals a mint operation from byte encoding
func (op *mintOperation) UnmarshalBinary(b []byte) error {
	return rlpDecode(b, op)
}

// MarshalBinary marshals a mint operation to byte encoding
func (op *mintOperation) MarshalBinary() ([]byte, error) {
	return rlpEncode(op)
}

// BurnOperation contains attributes for a burn token operation
type BurnOperation interface {
	Operation
	TokenId() *big.Int
}

type burnOperation struct {
	operation
	tokenIdOperation
}

// NewBurnOperation creates a burn operation.
// The operation only supports WRC-721 tokens so its Standard field sets to StdWRC721.
func NewBurnOperation(tokenId *big.Int) (BurnOperation, error) {
	if tokenId == nil {
		return nil, ErrNoTokenId
	}
	return &burnOperation{
		operation: operation{
			Std: StdWRC721,
		},
		tokenIdOperation: tokenIdOperation{
			Id: tokenId,
		},
	}, nil
}

// OpCode returns op code of a burn token operation
func (op *burnOperation) OpCode() OpCode {
	return OpBurn
}

// UnmarshalBinary unmarshals a burn operation from byte encoding
func (op *burnOperation) UnmarshalBinary(b []byte) error {
	return rlpDecode(b, op)
}

// MarshalBinary marshals a burn operation to byte encoding
func (op *burnOperation) MarshalBinary() ([]byte, error) {
	return rlpEncode(op)
}

// TokenOfOwnerByIndexOperation contatins attributes for WRC-721 token of owner by index operation
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

// NewBurnOperation creates a token of owner by index operation.
// The operation only supports WRC-721 tokens so its Standard field sets to StdWRC721.
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
			TokenAddress: address,
		},
		ownerOperation: ownerOperation{
			OwnerAddress: owner,
		},
		index: index,
	}, nil
}

// OpCode returns op code of a token of owner by index token operation
func (op *tokenOfOwnerByIndexOperation) OpCode() OpCode {
	return OpTokenOfOwnerByIndex
}

// Index returns copy of the index field
func (op *tokenOfOwnerByIndexOperation) Index() *big.Int {
	return new(big.Int).Set(op.index)
}

// UnmarshalBinary unmarshals a token of owner by index operation from byte encoding
func (op *tokenOfOwnerByIndexOperation) UnmarshalBinary(b []byte) error {
	return rlpDecode(b, op)
}

// MarshalBinary marshals a token of owner by index operation to byte encoding
func (op *tokenOfOwnerByIndexOperation) MarshalBinary() ([]byte, error) {
	return rlpEncode(op)
}

type opData struct {
	Std
	common.Address
	TokenId    *big.Int
	Owner      common.Address
	Spender    common.Address
	Operator   common.Address
	From       common.Address
	To         common.Address
	Value      *big.Int
	Index      *big.Int
	IsApproved bool
	Data       []byte `rlp:"tail"`
}

func rlpDecode(b []byte, op interface{}) error {
	data := opData{}
	if err := rlp.DecodeBytes(b, &data); err != nil {
		return err
	}

	// op is passed by pointer
	value := reflect.ValueOf(op).Elem()

	setFieldValue := func(name string, isValid func() bool, v interface{}) bool {
		field := value.FieldByName(name)
		// Check if the field is exists in the struct
		if field != (reflect.Value{}) {
			if !isValid() {
				return false
			}
			field.Set(reflect.ValueOf(v))
		}
		return true
	}

	if !setFieldValue("Std", func() bool {
		return data.Std == StdWRC20 || data.Std == StdWRC721 || data.Std == 0
	}, data.Std) {
		return ErrStandardNotValid
	}

	if !setFieldValue("TokenAddress", func() bool {
		return data.Address != (common.Address{})
	}, data.Address) {
		return ErrNoAddress
	}

	setFieldValue("TokenValue", func() bool {
		return true
	}, data.Value)

	if !setFieldValue("OwnerAddress", func() bool {
		return data.Owner != (common.Address{})
	}, data.Owner) {
		return ErrNoOwner
	}

	if !setFieldValue("FromAddress", func() bool {
		return data.From != (common.Address{})
	}, data.From) {
		return ErrNoFrom
	}

	if !setFieldValue("ToAddress", func() bool {
		return data.To != (common.Address{})
	}, data.To) {
		return ErrNoTo
	}

	if !setFieldValue("OperatorAddress", func() bool {
		return data.Operator != (common.Address{})
	}, data.Operator) {
		return ErrNoOperator
	}

	if !setFieldValue("SpenderAddress", func() bool {
		return data.Spender != (common.Address{})
	}, data.Spender) {
		return ErrNoSpender
	}

	if !setFieldValue("Id", func() bool {
		return true
	}, data.TokenId) {
		return nil
	}

	switch v := op.(type) {
	case *safeTransferFromOperation:
		v.data = data.Data
	case *setApprovalForAllOperation:
		v.Approved = data.IsApproved
	case *mintOperation:
		v.TokenMetadata = data.Data
	case *tokenOfOwnerByIndexOperation:
		v.index = data.Index
	}

	return nil
}

func rlpEncode(op interface{}) ([]byte, error) {
	data := opData{}
	dataValue := reflect.ValueOf(&data).Elem()

	opValue := reflect.ValueOf(op).Elem()
	opType := opValue.Type()

	re := regexp.MustCompile(`^(.*)Address$`)

	var fillData func(reflect.Type, reflect.Value)
	fillData = func(t reflect.Type, v reflect.Value) {
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)

			// If the field is embedded then traverse all fields in the field recursively
			if f.Anonymous && f.Type.Kind() == reflect.Struct {
				fillData(f.Type, v.FieldByName(f.Name))
				continue
			}

			var name string = ""
			switch f.Name {
			case "Std":
				name = f.Name
			case "Id":
				name = "TokenId"
			case "TokenValue":
				name = "Value"
			case "TokenAddress":
				name = "Address"
			case "TokenMetadata":
				name = "Data"
			case "data":
				name = "Data"
			case "Approved":
				name = "IsApproved"
			case "index":
				name = "Index"
			default:
				if re.MatchString(f.Name) {
					name = re.ReplaceAllString(f.Name, "$1")
				}
			}

			if name != "" {
				dv := dataValue.FieldByName(name)
				ov := opValue.FieldByName(f.Name)
				dv.Set(ov)
			}
		}
	}

	fillData(opType, opValue)
	return rlp.EncodeToBytes(data)
}
