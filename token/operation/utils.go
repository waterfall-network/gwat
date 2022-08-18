package operation

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
	"math/big"
	"reflect"
	"regexp"
)

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

type tokenIdOperation struct {
	Id *big.Int
}

// Returns copy of the token id field
func (op *tokenIdOperation) TokenId() *big.Int {
	return new(big.Int).Set(op.Id)
}

type operation struct {
	Std
}

// Standard returns token standard
func (op *operation) Standard() Std {
	return op.Std
}

type ownerOperation struct {
	OwnerAddress common.Address
}

// Owner returns copy of the owner address field
func (op *ownerOperation) Owner() common.Address {
	// It's safe to return common.Address by value, cause it's an array
	return op.OwnerAddress
}

type spenderOperation struct {
	SpenderAddress common.Address
}

// Spender returns copy of the spender address field
func (op *spenderOperation) Spender() common.Address {
	// It's safe to return common.Address by value, cause it's an array
	return op.SpenderAddress
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

	if !setFieldValue("TokenValue", func() bool {
		return data.Value != nil
	}, data.Value) {
		return ErrNoValue
	}

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
		return data.TokenId != nil
	}, data.TokenId) {
		return ErrNoTokenId
	}

	switch v := op.(type) {
	case *safeTransferFromOperation:
		v.OperationData = data.Data
	case *setApprovalForAllOperation:
		v.Approved = data.IsApproved
	case *mintOperation:
		v.TokenMetadata = data.Data
	case *tokenOfOwnerByIndexOperation:
		v.TokenIndex = data.Index
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

			name := ""
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
			case "OperationData":
				name = "Data"
			case "Approved":
				name = "IsApproved"
			case "TokenIndex":
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

func makeCopy(src []byte) []byte {
	dst := make([]byte, len(src))
	copy(dst, src)
	return dst
}
