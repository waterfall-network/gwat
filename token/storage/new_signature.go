package storage

import (
	"encoding"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"strconv"
)

var (
	ErrNoKeyProperties   = errors.New("no key properties for this type")
	ErrNoValueProperties = errors.New("no value properties for this type")
	ErrNoLength          = errors.New("no length for this type")
	ErrWrongType         = errors.New("wrong type")
)

type Type uint16

func (t Type) String() string {
	return strconv.FormatUint(t, 10)
}

const (
	uint8Type   Type = 0x0101
	uint16Type  Type = 0x0102
	uint32Type  Type = 0x0104
	uint64Type  Type = 0x0108
	uint256Type Type = 0x0120
	int32Type   Type = 0x0204
	int64Type   Type = 0x0208
	arrayType   Type = 0x0200
	mapType     Type = 0x0300

	signatureV1 SignatureVersion = 0x0001
)

var (
	uint8Size   = 1
	uint16Size  = 2
	uint32Size  = 4
	uint64Size  = 8
	uint256Size = 32
	typeSizes   = map[Type]int{
		uint8Type:   uint8Size,
		uint16Type:  uint16Size,
		uint32Type:  uint32Size,
		uint64Type:  uint64Size,
		uint256Type: uint256Size,
		int32Type:   uint32Size,
		int64Type:   uint64Size,
		arrayType:   0,
		mapType:     0,
	}
)

type ValueProperties interface {
	Type() Type
	Length() (int64, error)
	KeyProperties() (ValueProperties, error)
	ValueProperties() (ValueProperties, error)
	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler
}

type ScalarProperties Type

func (s ScalarProperties) Type() Type {
	return s
}

func (s ScalarProperties) Length() (int64, error) {
	return 0, ErrNoLength
}

func (s ScalarProperties) KeyProperties() (ValueProperties, error) {
	return nil, ErrNoKeyProperties
}

func (s ScalarProperties) ValueProperties() (ValueProperties, error) {
	return nil, ErrNoValueProperties
}

func (s ScalarProperties) MarshalBinary() (data []byte, err error) {
	buf := make([]byte, typeSizes[s])
	binary.BigEndian.PutUint16(buf, uint16(s))

	return buf, nil
}

func (s *ScalarProperties) UnmarshalBinary(data []byte) error {
	*s = ScalarProperties(binary.BigEndian.Uint16(data))
	return nil
}

type ArrayProperties struct {
	valueProperties ValueProperties
	len             int64
}

func NewArrayProperties(t *ScalarProperties, l int64) *ArrayProperties {
	return &ArrayProperties{
		valueProperties: t,
		len:             l,
	}
}

func (a *ArrayProperties) Type() Type {
	return arrayType
}

func (a *ArrayProperties) Length() (int64, error) {
	return a.len, nil
}

func (a *ArrayProperties) KeyProperties() (ValueProperties, error) {
	return nil, ErrNoKeyProperties
}

func (a *ArrayProperties) ValueProperties() (ValueProperties, error) {
	return a.valueProperties, nil
}

func (a *ArrayProperties) MarshalBinary() ([]byte, error) {
	res, err := a.valueProperties.MarshalBinary()
	if err != nil {
		return nil, err
	}

	typeSize := typeSizes[a.Type()]
	buf := make([]byte, typeSize+uint64Size+len(res))

	// type
	binary.BigEndian.PutUint16(buf, a.Type())

	// length
	binary.BigEndian.PutUint64(buf[typeSize:uint64Size], uint64(a.len))

	// valueProperties
	copy(buf[typeSize+uint64Size:], res)

	return buf, nil
}

func (a *ArrayProperties) UnmarshalBinary(data []byte) error {
	// type
	tp := int64(binary.BigEndian.Uint16(data[:uint16Size]))
	if tp != arrayType {
		return ErrWrongType
	}

	// length
	data = data[uint16Size:]
	l := int64(binary.BigEndian.Uint64(data[:uint64Size]))

	// valueProperties
	data = data[uint64Size:]
	err := a.valueProperties.UnmarshalBinary(data[:])
	if err != nil {
		return err
	}

	a.len = l
	return nil
}

type MapProperties struct {
	keyProperties   ValueProperties
	valueProperties ValueProperties
}

func NewMapProperties(key, value ValueProperties) (*MapProperties, error) {
	return &MapProperties{
		keyProperties:   key,
		valueProperties: value,
	}, nil
}

func (m *MapProperties) Type() Type {
	return mapType
}

func (m *MapProperties) Length() (int64, error) {
	return 0, ErrNoLength
}

func (m *MapProperties) KeyProperties() (ValueProperties, error) {
	return m.keyProperties, nil
}

func (m *MapProperties) ValueProperties() (ValueProperties, error) {
	return m.valueProperties, nil
}

func (m *MapProperties) MarshalBinary() (data []byte, err error) {
	// unmarshal keys
	keysB, err := m.keyProperties.MarshalBinary()
	if err != nil {
		return nil, err
	}

	// marshal values
	valuesB, err := m.valueProperties.MarshalBinary()
	if err != nil {
		return nil, err
	}

	typeSize := typeSizes[m.Type()]
	buf := make([]byte, 0, uint16Size+len(keysB)+len(valuesB))

	// type
	binary.BigEndian.PutUint16(buf[:typeSize], m.Type())

	// put keys
	off := typeSize
	buf = append(buf[off:], keysB...)

	// put values
	off += len(keysB)
	buf = append(buf[off:], valuesB...)

	return buf, nil
}

func (m *MapProperties) UnmarshalBinary(data []byte) error {
	newM := *m

	// type
	tp := int64(binary.BigEndian.Uint16(data[:uint16Size]))
	if tp != mapType {
		return ErrWrongType
	}

	// unmarshal key properties
	off := uint16Size
	err := newM.keyProperties.UnmarshalBinary(data[off:])
	if err != nil {
		return err
	}

	// add size of key type
	off += uint16Size
	if newM.Type() == arrayType {
		// add size of key length
		off += uint64Size
	}

	// unmarshal value properties
	err = newM.valueProperties.UnmarshalBinary(data[off:])
	if err != nil {
		return err
	}

	*m = newM
	return nil
}

type SignatureVersion uint16

func (sv SignatureVersion) String() string {
	return fmt.Sprintf("%d", sv)
}

type Signature interface {
	Fields() []FieldProperties
	ReadFromStream(*StorageStream) error
	WriteToStream(*StorageStream) error
	Version() SignatureVersion
}

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

type FieldDescriptor struct {
	name      string
	valueType reflect.Type
}

type FieldProperties struct {
	nameLen uint8
	name    string

	vp ValueProperties
}

func (fp FieldProperties) MarshalBinary() (data []byte, err error) {
	// marshal value properties
	res, err := fp.vp.MarshalBinary()
	if err != nil {
		return nil, err
	}

	buf := make([]byte, uint8Size+int(fp.nameLen)+len(res))

	// put size of name
	buf[0] = fp.nameLen

	// put name
	off := uint8Size
	copy(buf[off:], fp.name)

	// copy value properties
	off += int(fp.nameLen)
	copy(buf[off:], res)

	return buf, nil
}

func (fp *FieldProperties) UnmarshalBinary(data []byte) error {
	// firstly unmarshal value properties to avoid setting another fields if there is an error
	err := fp.vp.UnmarshalBinary(data[uint8Size+int(fp.nameLen):])
	if err != nil {
		return err
	}

	// len of name
	fp.nameLen = data[0]

	// name
	data = data[uint8Size:]
	fp.name = string(data[:fp.nameLen])

	return nil
}

func NewFieldProperties(fieldName string, v ValueProperties) FieldProperties {
	return FieldProperties{
		nameLen: uint8(len(fieldName)),
		name:    fieldName,
		vp:      v,
	}
}

type SignatureV1 struct {
	version     SignatureVersion
	fieldsProps []FieldProperties

	fields []Field
}

func NewSignatureV1(fd []FieldDescriptor) (*SignatureV1, error) {
	fieldProps := make([]FieldProperties, len(fd))
	for i, field := range fd {
		vp, err := newValueProperty(field.valueType)
		if err != nil {
			return nil, err
		}

		fieldProps[i] = NewFieldProperties(field.name, vp)
	}

	return &SignatureV1{
		version:     signatureV1,
		fieldsProps: fieldProps,
		fields:      make([]Field, len(fieldProps)),
	}, nil
}

func (s SignatureV1) Fields() []FieldProperties {
	return s.fieldsProps
}

func (s SignatureV1) ReadFromStream(stream *StorageStream) error {
	off := big.NewInt(0)

	// read signature size
	signSizeBuf := make([]byte, uint64Size)
	n, err := stream.ReadAt(signSizeBuf, off)
	if err != nil {
		return err
	}

	signBuf := make([]byte, binary.BigEndian.Uint64(signSizeBuf))

	off.Add(off, big.NewInt(int64(n)))

	// read the full signature
	n, err = stream.ReadAt(signBuf, off)
	if err != nil {
		return err
	}

	totalFieldSize := int64(0)
	fieldsProps := make([]FieldProperties, 0)
	fields := make([]Field, 0)
	for len(signBuf) > 0 {
		// read the size of the fieldProps
		fieldSize := binary.BigEndian.Uint16(signBuf[:uint16Size])

		// unmarshal the fieldProps
		fieldProps := FieldProperties{}
		err = fieldProps.UnmarshalBinary(signBuf[uint16Size:fieldSize])
		if err != nil {
			return err
		}

		fieldsProps = append(fieldsProps, fieldProps)

		totalFieldSize = int64(uint16Size + int(fieldSize))

		// fill the field info
		fields = append(fields, Field{
			offset: off.Add(off, big.NewInt(totalFieldSize)),
			length: n,
		})

		signBuf = signBuf[totalFieldSize:]
	}

	s.fields = fields
	s.fieldsProps = fieldsProps

	return nil
}

func (s SignatureV1) WriteToStream(stream *StorageStream) error {
	off := big.NewInt(0)
	buf := make([]byte, uint64Size)

	signSize := uint64(0)
	sizeBuf := make([]byte, uint16Size)
	for i, field := range s.fieldsProps {
		// marshal the fieldProps
		b, err := field.MarshalBinary()
		if err != nil {
			return err
		}

		// write size of the fieldProps
		binary.BigEndian.PutUint16(sizeBuf, uint16(len(b)))
		buf = append(buf, sizeBuf...)

		// write the fieldProps
		buf = append(buf, b...)

		s.fields[i] = Field{
			offset: *(&off),
			length: uint16Size + len(b),
		}

		off.Add(off, big.NewInt(int64(s.fields[i].length)))
		signSize += uint64(s.fields[i].length)
		sizeBuf = []byte{0x00, 0x00}
	}

	// write signature size
	binary.BigEndian.PutUint64(buf[:uint64Size], signSize)

	_, err := stream.WriteAt(buf, big.NewInt(0))
	return err
}

func (s SignatureV1) Version() SignatureVersion {
	return s.version
}

func newValueProperty(tp reflect.Type) (ValueProperties, error) {
	switch tp.Kind() {
	case reflect.Map:
		kp, err := newValueProperty(tp.Elem())
		if err != nil {
			return nil, err
		}

		vp, err := newValueProperty(tp.Key())
		if err != nil {
			return nil, err
		}

		return NewMapProperties(kp, vp)
	case reflect.Slice:
		fallthrough
	case reflect.Array:
		vp, err := newValueProperty(tp.Elem())
		if err != nil {
			return nil, err
		}

		sp := vp.(*ScalarProperties)
		return NewArrayProperties(sp, int64(tp.Len())), nil
	case reflect.Uint8:
		return uint8Type, nil
	case reflect.Uint16:
		return uint16Type, nil
	case reflect.Uint32:
		return uint32Type, nil
	case reflect.Uint64:
		return uint64Type, nil
	case reflect.Int32:
		return int32Type, nil
	case reflect.Int64:
		return int64Type, nil
	case reflect.Ptr:
		return uint256Type, nil
	}

	return nil, ErrWrongType
}
