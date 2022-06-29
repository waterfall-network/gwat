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

var (
	stdSize     = int(reflect.TypeOf(operation.Std(0)).Size())
	versionSize = int(reflect.TypeOf(uint16(0)).Size())
)

type field struct {
	offset int
	length int
}

type Signature interface {
	Version() uint16
	Name() (int, int)
	Symbol() (int, int)
	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler
}

type WRC20Signature interface {
	Decimals() (int, int)
	TotalSupply() (int, int)
	Signature
}

type WRC721Signature interface {
	BaseUri() (int, int)
	Signature
}

type wrc20SignatureV0 struct {
	fields      []field
	totalLength int
}

func newWrc20SignatureV0(name, symbol int) WRC20Signature {
	input := []int{
		name, symbol, decimalsSize, totalSupplySize,
	}

	sign, total := newSignature(input, wrc20V0FieldsAmount)

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
	input := []int{
		name, symbol, baseUri,
	}
	sign, total := newSignature(input, wrc721V0FieldsAmount)

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

func readSignatureFromStream(stream *StorageStream) (Signature, error) {
	b := make([]byte, stdSize)

	_, err := stream.ReadAt(b, 0)
	if err != nil {
		return nil, err
	}

	std := int(binary.BigEndian.Uint16(b))
	switch std {
	case operation.StdWRC20:
		ver, err := readVersion(stream)
		if err != nil {
			return nil, err
		}

		switch ver {
		case 0:
			s, err := readSignature(stream, wrc20V0FieldsAmount)
			if err != nil {
				return nil, err
			}
			return s, nil

		default:
			return nil, errors.New("unsupported version of wrc20 signature")
		}

	case operation.StdWRC721:
		ver, err := readVersion(stream)
		if err != nil {
			return nil, err
		}

		switch ver {
		case 0:
			s, err := readSignature(stream, wrc721V0FieldsAmount)
			if err != nil {
				return nil, err
			}

			return s, nil

		default:
			return nil, errors.New("unsupported version of wrc721 signature")
		}

	default:
		return nil, operation.ErrStandardNotValid
	}
}

func writeSignatureToStream(stream *StorageStream, sign Signature) error {
	buf := make([]byte, stdSize+versionSize)
	binary.BigEndian.PutUint16(buf[stdSize:], sign.Version())

	switch sign.(type) {
	case WRC20Signature:
		err := writeToStream(stream, sign, buf, operation.StdWRC20)
		if err != nil {
			return err
		}

	case WRC721Signature:
		err := writeToStream(stream, sign, buf, operation.StdWRC721)
		if err != nil {
			return err
		}
	}

	return nil
}

func writeToStream(stream *StorageStream, sign Signature, buf []byte, std operation.Std) error {
	binary.BigEndian.PutUint16(buf[:stdSize], uint16(std))
	off, _ := stream.WriteAt(buf, 0)
	b, _ := sign.MarshalBinary()
	_, err := stream.WriteAt(b, off)
	if err != nil {
		return err
	}

	return nil
}

func readVersion(stream *StorageStream) (int, error) {
	b := make([]byte, versionSize)
	_, err := stream.ReadAt(b, stdSize)
	if err != nil {
		return -1, err
	}

	ver := int(binary.BigEndian.Uint16(b))

	return ver, nil
}

func readSignature(stream *StorageStream, l int) (Signature, error) {
	buf := make([]byte, l)
	var s wrc721SignatureV0
	_, err := stream.ReadAt(buf, stdSize+versionSize)
	if err != nil {
		return nil, err
	}

	_ = s.UnmarshalBinary(buf)

	return &s, nil
}

func newSignature(buf []int, fieldsAmount int) ([]field, int) {
	shift := stdSize + versionSize + fieldsAmount
	sign := make([]field, fieldsAmount)

	total := buf[0]
	sign[0].offset = shift
	sign[0].length = buf[0]
	for i := 0; i < len(buf)-1; i++ {
		sign[i+1].offset = sign[i].offset + buf[i]
		sign[i+1].length = buf[i+1]
		total += buf[i+1]
	}

	return sign, total
}
