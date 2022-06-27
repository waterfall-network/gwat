package storage

import (
	"github.com/ethereum/go-ethereum/token/operation"
	"reflect"
)

type Field int

const (
	nameField Field = iota
	symbolField
	decimalsField
	totalSupplyField
	baseUriField        Field = 2
	wrc20V0FieldsAmount       = 4
	wrc721FieldsAmount        = 3
	uint16Size                = 2
	decimalsSize              = 1
	totalSupplySize           = 32
)

type WRC20Signature interface {
	Decimals() (int, int)
	TotalSupply() (int, int)
	Signature
}

type WRC721Signature interface {
	BaseUri() (int, int)
	Signature
}

type pair struct {
	offset int
	length int
}

type wrc20SignatureV0 struct {
	offsets []pair
}

func (s *wrc20SignatureV0) Name() (int, int) {
	return s.offsets[nameField].offset, s.offsets[nameField].length
}

func (s *wrc20SignatureV0) Symbol() (int, int) {
	return s.offsets[symbolField].offset, s.offsets[symbolField].length
}

func (s *wrc20SignatureV0) Decimals() (int, int) {
	return s.offsets[decimalsField].offset, s.offsets[symbolField].length
}

func (s *wrc20SignatureV0) TotalSupply() (int, int) {
	return s.offsets[totalSupplyField].offset, s.offsets[totalSupplyField].length
}

func (s *wrc20SignatureV0) Version() uint16 {
	return 0
}

func newWrc20SignatureV0(name, symbol int) WRC20Signature {
	stdSize := int(reflect.TypeOf(operation.Std(0)).Size())
	versionSize := int(reflect.TypeOf(uint16(0)).Size())
	shift := stdSize + versionSize + wrc20V0FieldsAmount

	input := []int{
		name, symbol, decimalsSize, totalSupplySize,
	}

	sign := make([]pair, wrc20V0FieldsAmount)

	for i, off := range input {
		offset := off
		if i == 0 {
			offset = 0
		}

		sign[i].offset = shift + offset
		sign[i].length = off
	}

	return &wrc20SignatureV0{offsets: sign}
}

type wrc721SignatureV0 struct {
	offsets []pair
}

func (s *wrc721SignatureV0) Name() (int, int) {
	return s.offsets[nameField].offset, s.offsets[nameField].length
}

func (s *wrc721SignatureV0) Symbol() (int, int) {
	return s.offsets[symbolField].offset, s.offsets[symbolField].length
}

func (s *wrc721SignatureV0) BaseUri() (int, int) {
	return s.offsets[baseUriField].offset, s.offsets[baseUriField].length
}

func (s *wrc721SignatureV0) Version() uint16 {
	return 0
}

type Signature interface {
	Version() uint16
	Name() (int, int)
	Symbol() (int, int)
}

//проверяет стандарт и версию и взависимости от них дергает нужную фабричную функцию для signature
func NewSignature(stream StorageStream) Signature {
	return nil
}

func NewWRC20SignatureV0(stream StorageStream) WRC20Signature {
	return nil
}
