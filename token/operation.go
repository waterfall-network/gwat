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

func (op *operation) Standard() Std {
	return op.Std
}

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

type createOpData struct {
	Std
	Name        []byte
	Symbol      []byte
	TotalSupply *big.Int `rlp:"nil"`
	Decimals    *uint8   `rlp:"nil"`
	BaseURI     []byte   `rlp:"optional"`
}

func (op *createOperation) UnmarshalBinary(b []byte) error {
	opData := createOpData{}
	if err := rlp.DecodeBytes(b, &opData); err != nil {
		return err
	}
	op.init(opData.Std, opData.Name, opData.Symbol, opData.Decimals, opData.TotalSupply, opData.BaseURI)
	return nil
}

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
	return makeCopy(op.symbol)
}

func (op *createOperation) Decimals() uint8 {
	return op.decimals
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
	TokenAddress common.Address
}

func (a *addressOperation) Address() common.Address {
	// It's safe to return common.Address by value, cause it's an array
	return a.TokenAddress
}

type propertiesOperation struct {
	operation
	addressOperation
	Id *big.Int
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
			Std: standard,
		},
		addressOperation: addressOperation{
			TokenAddress: address,
		},
		Id: tokenId,
	}, nil
}

func (op *propertiesOperation) UnmarshalBinary(b []byte) error {
	return rlpDecode(b, op)
}

func (op *propertiesOperation) MarshalBinary() ([]byte, error) {
	return rlpEncode(op)
}

func (op *propertiesOperation) OpCode() OpCode {
	return OpProperties
}

func (op *propertiesOperation) TokenId() (*big.Int, bool) {
	if op.Id == nil {
		return nil, false
	}
	return new(big.Int).Set(op.Id), true
}

type BalanceOfOperation interface {
	Operation
	addresser
	Owner() common.Address
}

type ownerOperation struct {
	OwnerAddress common.Address
}

func (op *ownerOperation) Owner() common.Address {
	// It's safe to return common.Address by value, cause it's an array
	return op.OwnerAddress
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
			TokenAddress: address,
		},
		ownerOperation: ownerOperation{
			OwnerAddress: owner,
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

func (op *balanceOfOperation) UnmarshalBinary(b []byte) error {
	return rlpDecode(b, op)
}

func (op *balanceOfOperation) MarshalBinary() ([]byte, error) {
	return rlpEncode(op)
}

type TransferOperation interface {
	Operation
	To() common.Address
	Value() *big.Int
}

type valueOperation struct {
	TokenValue *big.Int
}

func (op *valueOperation) Value() *big.Int {
	return new(big.Int).Set(op.TokenValue)
}

type toOperation struct {
	ToAddress common.Address
}

func (op *toOperation) To() common.Address {
	// It's safe to return common.Address by value, cause it's an array
	return op.ToAddress
}

type transferOperation struct {
	operation
	valueOperation
	toOperation
}

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

func (op *transferOperation) OpCode() OpCode {
	return OpTransfer
}

func (op *transferOperation) UnmarshalBinary(b []byte) error {
	return rlpDecode(b, op)
}

func (op *transferOperation) MarshalBinary() ([]byte, error) {
	return rlpEncode(op)
}

type TransferFromOperation interface {
	TransferOperation
	From() common.Address
}

type transferFromOperation struct {
	transferOperation
	FromAddress common.Address
}

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

func (op *transferFromOperation) From() common.Address {
	// It's safe to return common.Address by value, cause it's an array
	return op.FromAddress
}

func (op *transferFromOperation) UnmarshalBinary(b []byte) error {
	return rlpDecode(b, op)
}

func (op *transferFromOperation) MarshalBinary() ([]byte, error) {
	return rlpEncode(op)
}

type SafeTransferFromOperation interface {
	TransferFromOperation
	Data() ([]byte, bool)
}

type safeTransferFromOperation struct {
	transferFromOperation
	data []byte
}

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

func (op *safeTransferFromOperation) Data() ([]byte, bool) {
	if len(op.data) == 0 {
		return nil, false
	}
	return makeCopy(op.data), true
}

func (op *safeTransferFromOperation) UnmarshalBinary(b []byte) error {
	return rlpDecode(b, op)
}

func (op *safeTransferFromOperation) MarshalBinary() ([]byte, error) {
	return rlpEncode(op)
}

type ApproveOperation interface {
	Operation
	Spender() common.Address
	Value() *big.Int
}

type spenderOperation struct {
	SpenderAddress common.Address
}

func (op *spenderOperation) Spender() common.Address {
	// It's safe to return common.Address by value, cause it's an array
	return op.SpenderAddress
}

type approveOperation struct {
	operation
	valueOperation
	spenderOperation
}

func NewApproveOperation(spender common.Address, value *big.Int) (ApproveOperation, error) {
	if spender == (common.Address{}) {
		return nil, ErrNoSpender
	}
	if value == nil {
		return nil, ErrNoValue
	}
	return &approveOperation{
		operation: operation{
			Std: StdWRC20,
		},
		valueOperation: valueOperation{
			TokenValue: value,
		},
		spenderOperation: spenderOperation{
			SpenderAddress: spender,
		},
	}, nil
}

func (op *approveOperation) OpCode() OpCode {
	return OpApprove
}

func (op *approveOperation) UnmarshalBinary(b []byte) error {
	return rlpDecode(b, op)
}

func (op *approveOperation) MarshalBinary() ([]byte, error) {
	return rlpEncode(op)
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

func (op *allowanceOperation) OpCode() OpCode {
	return OpAllowance
}

func (op *allowanceOperation) UnmarshalBinary(b []byte) error {
	return rlpDecode(b, op)
}

func (op *allowanceOperation) MarshalBinary() ([]byte, error) {
	return rlpEncode(op)
}

type IsApprovedForAllOperation interface {
	Operation
	addresser
	Owner() common.Address
	Operator() common.Address
}

type operatorOperation struct {
	OperatorAddress common.Address
}

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

func (op *isApprovedForAllOperation) OpCode() OpCode {
	return OpIsApprovedForAll
}

func (op *isApprovedForAllOperation) UnmarshalBinary(b []byte) error {
	return rlpDecode(b, op)
}

func (op *isApprovedForAllOperation) MarshalBinary() ([]byte, error) {
	return rlpEncode(op)
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
			Std: StdWRC721,
		},
		addressOperation: addressOperation{
			TokenAddress: address,
		},
		operatorOperation: operatorOperation{
			OperatorAddress: operator,
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

func (op *setApprovalForAllOperation) UnmarshalBinary(b []byte) error {
	return rlpDecode(b, op)
}

func (op *setApprovalForAllOperation) MarshalBinary() ([]byte, error) {
	return rlpEncode(op)
}

type MintOperation interface {
	Operation
	addresser
	To() common.Address
	TokenId() *big.Int
	Metadata() ([]byte, bool)
}

type tokenIdOperation struct {
	Id *big.Int
}

func (op *tokenIdOperation) TokenId() *big.Int {
	return new(big.Int).Set(op.Id)
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
			Std: StdWRC721,
		},
		addressOperation: addressOperation{
			TokenAddress: address,
		},
		toOperation: toOperation{
			ToAddress: to,
		},
		tokenIdOperation: tokenIdOperation{
			Id: tokenId,
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

func (op *mintOperation) UnmarshalBinary(b []byte) error {
	return rlpDecode(b, op)
}

func (op *mintOperation) MarshalBinary() ([]byte, error) {
	return rlpEncode(op)
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
			Std: StdWRC721,
		},
		addressOperation: addressOperation{
			TokenAddress: address,
		},
		tokenIdOperation: tokenIdOperation{
			Id: tokenId,
		},
	}, nil
}

func (op *burnOperation) OpCode() OpCode {
	return OpBurn
}

func (op *burnOperation) UnmarshalBinary(b []byte) error {
	return rlpDecode(b, op)
}

func (op *burnOperation) MarshalBinary() ([]byte, error) {
	return rlpEncode(op)
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
			TokenAddress: address,
		},
		ownerOperation: ownerOperation{
			OwnerAddress: owner,
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

func (op *tokenOfOwnerByIndexOperation) UnmarshalBinary(b []byte) error {
	return rlpDecode(b, op)
}

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
		v.isApproved = data.IsApproved
	case *mintOperation:
		v.metadata = data.Data
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
			case "metadata":
				name = "Data"
			case "data":
				name = "Data"
			case "isApproved":
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
