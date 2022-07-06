package storage

import (
	"encoding"
	"encoding/binary"
	"errors"
	"github.com/ethereum/go-ethereum/token/operation"
	"math/big"
	"reflect"
)

type fieldSize int

const (
	nameField fieldSize = iota
	symbolField
	decimalsField
	totalSupplyField
	baseUriField    fieldSize = 2
	decimalsSize              = 1
	totalSupplySize           = 32
	mapSize                   = 1
)

var (
	stdSize                        = int(reflect.TypeOf(operation.Std(0)).Size())
	versionSize                    = int(reflect.TypeOf(uint16(0)).Size())
	ErrUnsupportedSignatureVersion = errors.New("unsupported version of signature")
	ErrEmptyBuf                    = errors.New("buffer is empty")
	initialOffset                  = big.NewInt(0)
)

type Field struct {
	offset *big.Int
	length int
}

func (f *Field) Offset() *big.Int {
	return f.offset
}

func (f *Field) Length() int {
	return f.length
}

type wrc20Signature interface {
	Decimals() Field
	TotalSupply() Field
	Version() uint16
	Name() Field
	Symbol() Field
	BytesSize() int
	TotalLength() *big.Int
	ReadFromStream(stream *StorageStream) error
	WriteToStream(stream *StorageStream) error
	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler
}

type wrc721Signature interface {
	BaseUri() Field
	Version() uint16
	Name() Field
	Symbol() Field
	BytesSize() int
	TotalLength() *big.Int
	ReadFromStream(stream *StorageStream) error
	WriteToStream(stream *StorageStream) error
	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler
}

type signatureV1 struct {
	fields []Field
	std    operation.Std
	total  *big.Int
}

func newSignatureV1(fieldSizes []int, std operation.Std) *signatureV1 {
	shift := stdSize + versionSize + len(fieldSizes)
	sign := make([]Field, len(fieldSizes))

	if len(fieldSizes) > 0 {
		total := big.NewInt(int64(shift + fieldSizes[0]))
		sign[0].offset = big.NewInt(int64(shift))
		sign[0].length = fieldSizes[0]
		for i := 0; i < len(fieldSizes)-1; i++ {
			sign[i+1].offset = big.NewInt(0).Add(sign[i].offset, big.NewInt(int64(fieldSizes[i])))
			sign[i+1].length = fieldSizes[i+1]
			total.Add(total, big.NewInt(int64(fieldSizes[i+1])))
		}

		return &signatureV1{
			fields: sign,
			std:    std,
			total:  total,
		}
	}

	return nil
}

func (s *signatureV1) BytesSize() int {
	return len(s.fields) + stdSize + versionSize
}

func (s *signatureV1) TotalLength() *big.Int {
	return s.total
}

func (s *signatureV1) MarshalBinary() ([]byte, error) {
	buf := make([]byte, len(s.fields)+stdSize+versionSize)
	stdBuf, err := s.std.MarshalBinary()
	if err != nil {
		return nil, err
	}

	copy(buf[:stdSize], stdBuf)
	binary.BigEndian.PutUint16(buf[stdSize:stdSize+versionSize], s.Version())

	for i, f := range s.fields {
		buf[stdSize+versionSize+i] = uint8(f.length)
	}

	return buf, nil
}

func (s *signatureV1) UnmarshalBinary(buf []byte) error {
	if len(buf) == 0 {
		return ErrEmptyBuf
	}

	var std operation.Std
	err := std.UnmarshalBinary(buf[:stdSize])
	if err != nil {
		return err
	}

	if std != s.std {
		return operation.ErrStandardNotValid
	}

	ver := binary.BigEndian.Uint16(buf[stdSize : stdSize+versionSize])

	if ver != s.Version() {
		return ErrUnsupportedSignatureVersion
	}

	s.fields[0].length = int(buf[stdSize+versionSize])
	s.total.Add(s.total, big.NewInt(int64(s.fields[0].length)))
	for i := 0; i < len(buf[stdSize+versionSize:])-1; i++ {
		s.fields[i+1].length = int(buf[stdSize+versionSize+i+1])
		s.fields[i+1].offset.Add(s.fields[i].offset, big.NewInt(int64(s.fields[i].length)))
		s.total.Add(s.total, big.NewInt(int64(s.fields[i+1].length)))
	}

	return nil
}

func (s *signatureV1) Name() Field {
	return s.fields[nameField]
}

func (s *signatureV1) Symbol() Field {
	return s.fields[symbolField]
}

func (s *signatureV1) Version() uint16 {
	return 1
}

func (s *signatureV1) WriteToStream(stream *StorageStream) error {
	err := writeToStream(stream, s)
	if err != nil {
		return err
	}

	return nil
}

func (s *signatureV1) ReadFromStream(stream *StorageStream) error {
	buf := make([]byte, s.BytesSize())
	_, err := stream.ReadAt(buf, initialOffset)
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

	//wrc20 token signature has 2 maps: balances and allowances.
	input := []int{
		name, symbol, decimalsSize, totalSupplySize, mapSize, mapSize,
	}

	sign := newSignatureV1(input, operation.StdWRC20)

	return &wrc20SignatureV1{sign}
}

func lastWrc20Signature(name, symbol int) wrc20Signature {
	return newWrc20SignatureV1(name, symbol)

}

func (s *wrc20SignatureV1) Decimals() Field {
	return s.fields[decimalsField]
}

func (s *wrc20SignatureV1) TotalSupply() Field {
	return s.fields[totalSupplyField]
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

	//wrc721 token signature has 4 maps: balances, owners, tokenApprovals, operatorApprovals.
	input := []int{
		name, symbol, baseUri, mapSize, mapSize, mapSize, mapSize,
	}
	sign := newSignatureV1(input, operation.StdWRC721)

	return &wrc721SignatureV1{sign}
}

func lastWrc721Signature(name, symbol, baseUri int) wrc721Signature {
	return newWrc721SignatureV1(name, symbol, baseUri)
}

func (s *wrc721SignatureV1) BaseUri() Field {
	return s.fields[baseUriField]
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

func writeToStream(stream *StorageStream, sign encoding.BinaryMarshaler) error {
	b, _ := sign.MarshalBinary()
	_, err := stream.WriteAt(b, initialOffset)
	if err != nil {
		return err
	}

	return nil
}

func readVersion(stream *StorageStream) (uint16, error) {
	b := make([]byte, versionSize)
	_, err := stream.ReadAt(b, big.NewInt(int64(stdSize)))
	if err != nil {
		return 0, err
	}

	ver := binary.BigEndian.Uint16(b)

	return ver, nil
}

func readVersionAndCheckStd(stream *StorageStream, std operation.Std) (uint16, error) {
	var s operation.Std
	b := make([]byte, stdSize)

	_, err := stream.ReadAt(b, initialOffset)
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
