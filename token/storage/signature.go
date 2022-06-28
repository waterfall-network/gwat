package storage

import (
	"encoding"
	"encoding/binary"
	"errors"
	"github.com/ethereum/go-ethereum/token/operation"
	"reflect"
)

type Field int

const (
	nameField Field = iota
	symbolField
	decimalsField
	totalSupplyField
	baseUriField         Field = 2
	wrc20V0FieldsAmount        = 4
	wrc721V0FieldsAmount       = 3
	decimalsSize               = 1
	totalSupplySize            = 32
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

type field struct {
	offset int
	length int
}

type wrc20SignatureV0 struct {
	fields      []field
	totalLength int
}

func newWrc20SignatureV0(name, symbol int) WRC20Signature {
	stdSize := int(reflect.TypeOf(operation.Std(0)).Size())
	versionSize := int(reflect.TypeOf(uint16(0)).Size())
	shift := stdSize + versionSize + wrc20V0FieldsAmount

	input := []int{
		name, symbol, decimalsSize, totalSupplySize,
	}

	sign := make([]field, wrc20V0FieldsAmount)

	total := name
	sign[0].offset = shift
	sign[0].length = name
	for i := 0; i < len(input)-1; i++ {
		sign[i+1].offset = sign[i].offset + input[i]
		sign[i+1].length = input[i+1]
		total += input[i+1]
	}

	return &wrc20SignatureV0{
		fields:      sign,
		totalLength: total,
	}
}

func lastWrc20Signature(name, symbol int) WRC20Signature {
	return newWrc20SignatureV0(name, symbol)
}

func (s *wrc20SignatureV0) MarshalBinary() ([]byte, error) {
	buf := make([]byte, wrc20V0FieldsAmount)

	for i, f := range s.fields {
		buf[i] = uint8(f.length)
	}

	return buf, nil
}

func (s *wrc20SignatureV0) UnmarshalBinary(buf []byte) error {
	for i, b := range buf {
		s.fields[i].length = int(b)
	}

	return nil
}

func (s *wrc20SignatureV0) Name() (int, int) {
	return s.fields[nameField].offset, s.fields[nameField].length
}

func (s *wrc20SignatureV0) Symbol() (int, int) {
	return s.fields[symbolField].offset, s.fields[symbolField].length
}

func (s *wrc20SignatureV0) Decimals() (int, int) {
	return s.fields[decimalsField].offset, s.fields[symbolField].length
}

func (s *wrc20SignatureV0) TotalSupply() (int, int) {
	return s.fields[totalSupplyField].offset, s.fields[totalSupplyField].length
}

func (s *wrc20SignatureV0) Version() uint16 {
	return 0
}

type wrc721SignatureV0 struct {
	fields      []field
	totalLength int
}

func newWrc721SignatureV0(name, symbol, baseUri int) WRC721Signature {
	stdSize := int(reflect.TypeOf(operation.Std(0)).Size())
	versionSize := int(reflect.TypeOf(uint16(0)).Size())
	shift := stdSize + versionSize + wrc721V0FieldsAmount

	input := []int{
		name, symbol, baseUri,
	}

	sign := make([]field, wrc721V0FieldsAmount)

	total := name
	sign[0].offset = shift
	sign[0].length = name
	for i := 0; i < len(input)-1; i++ {
		sign[i+1].offset = sign[i].offset + input[i]
		sign[i+1].length = input[i+1]
		total += input[i+1]
	}

	return &wrc721SignatureV0{
		fields:      sign,
		totalLength: total,
	}
}

func lastWrc721Signature(name, symbol, baseUri int) WRC721Signature {
	return newWrc721SignatureV0(name, symbol, baseUri)
}

func (s *wrc721SignatureV0) MarshalBinary() ([]byte, error) {
	buf := make([]byte, s.totalLength)

	for i, f := range s.fields {
		buf[i] = uint8(f.length)
	}

	return buf, nil
}

func (s *wrc721SignatureV0) UnmarshalBinary(buf []byte) error {
	for i, b := range buf {
		s.fields[i].length = int(b)
	}

	return nil
}

func (s *wrc721SignatureV0) Name() (int, int) {
	return s.fields[nameField].offset, s.fields[nameField].length
}

func (s *wrc721SignatureV0) Symbol() (int, int) {
	return s.fields[symbolField].offset, s.fields[symbolField].length
}

func (s *wrc721SignatureV0) BaseUri() (int, int) {
	return s.fields[baseUriField].offset, s.fields[baseUriField].length
}

func (s *wrc721SignatureV0) Version() uint16 {
	return 0
}

type Signature interface {
	Version() uint16
	Name() (int, int)
	Symbol() (int, int)
	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler
}

// Checks the standard and version.
// Based on the received data, call the desired factory function
func readSignatureFromStream(stream StorageStream) (Signature, error) {
	stdSize := int(reflect.TypeOf(operation.Std(0)).Size())
	b := make([]byte, stdSize)

	_, err := stream.ReadAt(b, 0)
	if err != nil {
		return nil, err
	}

	std := int(binary.BigEndian.Uint16(b))
	switch std {
	case operation.StdWRC20:
		versionSize := int(reflect.TypeOf(uint16(0)).Size())
		b := make([]byte, versionSize)
		_, err := stream.ReadAt(b, stdSize)
		if err != nil {
			return nil, err
		}

		ver := int(binary.BigEndian.Uint16(b))
		switch ver {
		case 0:
			buf := make([]byte, wrc20V0FieldsAmount)
			var s wrc20SignatureV0
			_, err := stream.ReadAt(buf, stdSize+versionSize)
			if err != nil {
				return nil, err
			}

			_ = s.UnmarshalBinary(buf)
			wrc20Signature := newWrc20SignatureV0(s.fields[nameField].length, s.fields[symbolField].length)
			return wrc20Signature, nil

		default:
			return nil, errors.New("unsupported version of wrc20 signature")
		}

	case operation.StdWRC721:
		versionSize := int(reflect.TypeOf(uint16(0)).Size())
		b := make([]byte, versionSize)
		_, err := stream.ReadAt(b, stdSize)
		if err != nil {
			return nil, err
		}

		ver := int(binary.BigEndian.Uint16(b))
		switch ver {
		case 0:
			buf := make([]byte, wrc721V0FieldsAmount)
			var s wrc721SignatureV0
			_, err := stream.ReadAt(buf, stdSize+versionSize)
			if err != nil {
				return nil, err
			}

			_ = s.UnmarshalBinary(buf)
			wrc721Signature := newWrc721SignatureV0(s.fields[nameField].length, s.fields[symbolField].length, s.fields[baseUriField].length)
			return wrc721Signature, nil

		default:
			return nil, errors.New("unsupported version of wrc721 signature")
		}

	default:
		return nil, operation.ErrStandardNotValid
	}
}

func writeSignatureToStream(stream *StorageStream, sign Signature) {
	stdSize := int(reflect.TypeOf(operation.Std(0)).Size())
	versionSize := int(reflect.TypeOf(uint16(0)).Size())
	buf := make([]byte, stdSize+versionSize)
	binary.BigEndian.PutUint16(buf[stdSize:], sign.Version())

	switch sign.(type) {
	case WRC20Signature:
		binary.BigEndian.PutUint16(buf[:stdSize], operation.StdWRC20)
		off, _ := stream.WriteAt(buf, 0)
		b, _ := sign.MarshalBinary()
		stream.WriteAt(b, off)

	case WRC721Signature:
		binary.BigEndian.PutUint16(buf[:stdSize], operation.StdWRC721)
		off, _ := stream.WriteAt(buf, 0)
		b, _ := sign.MarshalBinary()
		stream.WriteAt(b, off)
	}
}
