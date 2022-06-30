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
	baseUriField    Field = 2
	decimalsSize          = 1
	totalSupplySize       = 32
)

var (
	stdSize                        = int(reflect.TypeOf(operation.Std(0)).Size())
	versionSize                    = int(reflect.TypeOf(uint16(0)).Size())
	ErrUnsupportedSignatureVersion = errors.New("unsupported version of signature")
)

type field struct {
	offset int
	length int
}

type wrc20Signature interface {
	Decimals() (int, int)
	TotalSupply() (int, int)
	Version() uint16
	Name() (int, int)
	Symbol() (int, int)
	BytesSize() int
	ReadFromStream(stream *StorageStream) error
	WriteToStream(stream *StorageStream) error
	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler
}

type wrc721Signature interface {
	BaseUri() (int, int)
	Version() uint16
	Name() (int, int)
	Symbol() (int, int)
	BytesSize() int
	ReadFromStream(stream *StorageStream) error
	WriteToStream(stream *StorageStream) error
	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler
}

type signatureV1 struct {
	fields      []field
	totalLength int
	std         operation.Std
}

func newSignatureV1(buf []int, fieldsAmount int, std operation.Std) *signatureV1 {
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

	return &signatureV1{
		fields:      sign,
		totalLength: total,
		std:         std,
	}
}

func (s *signatureV1) BytesSize() int {
	return len(s.fields) + stdSize + versionSize
}

func (s *signatureV1) MarshalBinary() ([]byte, error) {
	buf := make([]byte, len(s.fields))

	for i, f := range s.fields {
		buf[i] = uint8(f.length)
	}

	return buf, nil
}

func (s *signatureV1) UnmarshalBinary(buf []byte) error {
	var std operation.Std
	err := std.UnmarshalBinary(buf[:stdSize])
	if err != nil {
		return err
	}

	ver := binary.BigEndian.Uint16(buf[stdSize : stdSize+versionSize])

	if ver != 1 {
		return ErrUnsupportedSignatureVersion
	}

	if std != s.std {
		return operation.ErrStandardNotValid
	}

	for i, b := range buf[stdSize+versionSize:] {
		s.fields[i].length = int(b)
	}

	return nil
}

func (s *signatureV1) Name() (int, int) {
	return s.fields[nameField].offset, s.fields[nameField].length
}

func (s *signatureV1) Symbol() (int, int) {
	return s.fields[symbolField].offset, s.fields[symbolField].length
}

func (s *signatureV1) Version() uint16 {
	return 1
}

func (s *signatureV1) WriteToStream(stream *StorageStream) error {
	buf := make([]byte, stdSize+versionSize)
	binary.BigEndian.PutUint16(buf[stdSize:], s.Version())
	err := writeToStream(stream, s, buf, s.std)
	if err != nil {
		return err
	}

	return nil
}

func (s *signatureV1) ReadFromStream(stream *StorageStream) error {
	buf := make([]byte, s.BytesSize())
	_, err := stream.ReadAt(buf, 0)
	if err != nil {
		return err
	}

	_ = s.UnmarshalBinary(buf)

	return nil
}

type wrc20SignatureV1 struct {
	*signatureV1
}

func readWrc20Signature(stream *StorageStream) (wrc20Signature, error) {
	newSign := func(ver uint16) (streamReader, error) {
		switch ver {
		case 1:
			return newWrc20SignatureV1(0, 0), nil

		default:
			return nil, ErrUnsupportedSignatureVersion
		}
	}

	reader, err := readSignature(stream, newSign, operation.StdWRC20)
	if err != nil {
		return nil, err
	}

	return reader.(wrc20Signature), nil
}

func newWrc20SignatureV1(name, symbol int) *wrc20SignatureV1 {
	input := []int{
		name, symbol, decimalsSize, totalSupplySize,
	}

	sign := newSignatureV1(input, len(input), operation.StdWRC20)

	return &wrc20SignatureV1{sign}
}

func lastWrc20Signature(name, symbol int) wrc20Signature {
	return newWrc20SignatureV1(name, symbol)

}

func (s *wrc20SignatureV1) Decimals() (int, int) {
	return s.fields[decimalsField].offset, s.fields[decimalsField].length
}

func (s *wrc20SignatureV1) TotalSupply() (int, int) {
	return s.fields[totalSupplyField].offset, s.fields[totalSupplyField].length
}

type wrc721SignatureV1 struct {
	*signatureV1
}

func readWrc721Signature(stream *StorageStream) (wrc721Signature, error) {
	newSign := func(ver uint16) (streamReader, error) {
		switch ver {
		case 1:
			return newWrc721SignatureV1(0, 0, 0), nil

		default:
			return nil, ErrUnsupportedSignatureVersion
		}
	}

	reader, err := readSignature(stream, newSign, operation.StdWRC721)
	if err != nil {
		return nil, err
	}

	return reader.(wrc721Signature), nil
}

func newWrc721SignatureV1(name, symbol, baseUri int) *wrc721SignatureV1 {
	input := []int{
		name, symbol, baseUri,
	}
	sign := newSignatureV1(input, len(input), operation.StdWRC721)

	return &wrc721SignatureV1{sign}
}

func lastWrc721Signature(name, symbol, baseUri int) wrc721Signature {
	return newWrc721SignatureV1(name, symbol, baseUri)
}

func (s *wrc721SignatureV1) BaseUri() (int, int) {
	return s.fields[baseUriField].offset, s.fields[baseUriField].length
}

type streamReader interface {
	ReadFromStream(*StorageStream) error
}

func readSignature(stream *StorageStream, f func(ver uint16) (streamReader, error), std operation.Std) (streamReader, error) {
	ver, err := readVersionAndCheckStd(stream, std)
	if err != nil {
		return nil, err
	}

	reader, err := f(ver)
	if err != nil {
		return nil, err
	}

	err = reader.ReadFromStream(stream)
	if err != nil {
		return nil, err
	}

	return reader, nil
}

func writeToStream(stream *StorageStream, sign encoding.BinaryMarshaler, buf []byte, std operation.Std) error {
	stdBuf, _ := std.MarshalBinary()
	copy(buf[:stdSize], stdBuf)

	_ = std.UnmarshalBinary(buf[:stdSize])

	off, _ := stream.WriteAt(buf, 0)
	b, _ := sign.MarshalBinary()
	_, err := stream.WriteAt(b, off)
	if err != nil {
		return err
	}

	return nil
}

func readVersion(stream *StorageStream) (uint16, error) {
	b := make([]byte, versionSize)
	_, err := stream.ReadAt(b, stdSize)
	if err != nil {
		return 0, err
	}

	ver := binary.BigEndian.Uint16(b)

	return ver, nil
}

func readVersionAndCheckStd(stream *StorageStream, std operation.Std) (uint16, error) {
	var s operation.Std
	b := make([]byte, stdSize)

	_, err := stream.ReadAt(b, 0)
	if err != nil {
		return 0, err
	}

	_ = s.UnmarshalBinary(b)

	if s != std {
		return 0, operation.ErrStandardNotValid
	}

	ver, err := readVersion(stream)
	if err != nil {
		return 0, err
	}

	return ver, nil
}
