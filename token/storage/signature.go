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

type signatureV0 struct {
	fields      []field
	totalLength int
}

func newSignatureV0(buf []int, fieldsAmount int) *signatureV0 {
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

	return &signatureV0{
		fields:      sign,
		totalLength: total,
	}
}

func (s *signatureV0) BytesSize() int {
	return len(s.fields)
}

func (s *signatureV0) MarshalBinary() ([]byte, error) {
	buf := make([]byte, len(s.fields))

	for i, f := range s.fields {
		buf[i] = uint8(f.length)
	}

	return buf, nil
}

func (s *signatureV0) UnmarshalBinary(buf []byte) error {
	for i, b := range buf {
		s.fields[i].length = int(b)
	}

	return nil
}

func (s *signatureV0) Name() (int, int) {
	return s.fields[nameField].offset, s.fields[nameField].length
}

func (s *signatureV0) Symbol() (int, int) {
	return s.fields[symbolField].offset, s.fields[symbolField].length
}

func (s *signatureV0) Version() uint16 {
	return 0
}

func (s *signatureV0) readFromStream(stream *StorageStream) error {
	buf := make([]byte, s.BytesSize())
	_, err := stream.ReadAt(buf, stdSize+versionSize)
	if err != nil {
		return err
	}

	_ = s.UnmarshalBinary(buf)

	return nil
}

type wrc20SignatureV0 struct {
	*signatureV0
}

func newWrc20SignatureV0(name, symbol int) *wrc20SignatureV0 {
	input := []int{
		name, symbol, decimalsSize, totalSupplySize,
	}

	sign := newSignatureV0(input, len(input))

	return &wrc20SignatureV0{signatureV0: sign}
}

func lastWrc20Signature(name, symbol int) wrc20Signature {
	return newWrc20SignatureV0(name, symbol)
}

func (s *wrc20SignatureV0) Decimals() (int, int) {
	return s.fields[decimalsField].offset, s.fields[symbolField].length
}

func (s *wrc20SignatureV0) TotalSupply() (int, int) {
	return s.fields[totalSupplyField].offset, s.fields[totalSupplyField].length
}

func (s *wrc20SignatureV0) ReadFromStream(stream *StorageStream) error {
	ver, err := readStdAndVersion(stream, operation.StdWRC20)
	if err != nil {
		return err
	}

	switch ver {
	case 0:
		s := newWrc20SignatureV0(0, 0)
		err := s.readFromStream(stream)
		if err != nil {
			return err
		}

		return nil

	default:
		return ErrUnsupportedSignatureVersion
	}

}

func (s *wrc20SignatureV0) WriteToStream(stream *StorageStream) error {
	buf := make([]byte, stdSize+versionSize)
	binary.BigEndian.PutUint16(buf[stdSize:], s.Version())
	err := writeToStream(stream, s, buf, operation.StdWRC20)
	if err != nil {
		return err
	}

	return nil
}

type wrc721SignatureV0 struct {
	*signatureV0
}

func newWrc721SignatureV0(name, symbol, baseUri int) *wrc721SignatureV0 {
	input := []int{
		name, symbol, baseUri,
	}
	sign := newSignatureV0(input, len(input))

	return &wrc721SignatureV0{signatureV0: sign}
}

func lastWrc721Signature(name, symbol, baseUri int) wrc721Signature {
	return newWrc721SignatureV0(name, symbol, baseUri)
}

func (s *wrc721SignatureV0) BaseUri() (int, int) {
	return s.fields[baseUriField].offset, s.fields[baseUriField].length
}

func (s *wrc721SignatureV0) ReadFromStream(stream *StorageStream) error {
	ver, err := readStdAndVersion(stream, operation.StdWRC721)
	if err != nil {
		return err
	}

	switch ver {
	case 0:
		s := newWrc721SignatureV0(0, 0, 0)
		err := s.readFromStream(stream)
		if err != nil {
			return err
		}

		return nil

	default:
		return ErrUnsupportedSignatureVersion
	}

}

func (s *wrc721SignatureV0) WriteToStream(stream *StorageStream) error {
	buf := make([]byte, stdSize+versionSize)
	binary.BigEndian.PutUint16(buf[stdSize:], s.Version())
	err := writeToStream(stream, s, buf, operation.StdWRC721)
	if err != nil {
		return err
	}

	return nil
}

func writeToStream(stream *StorageStream, sign encoding.BinaryMarshaler, buf []byte, std operation.Std) error {
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

func readStdAndVersion(stream *StorageStream, std int) (int, error) {
	b := make([]byte, stdSize)

	_, err := stream.ReadAt(b, 0)
	if err != nil {
		return -1, err
	}

	s := int(binary.BigEndian.Uint16(b))

	if s != std {
		return -1, operation.ErrStandardNotValid
	}

	ver, err := readVersion(stream)
	if err != nil {
		return -1, err
	}

	return ver, nil
}
