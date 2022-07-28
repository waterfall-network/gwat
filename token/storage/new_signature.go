package storage

import (
	"encoding"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
)

const signatureV1 SignatureVersion = 0x0001

var (
	ErrNoKeyProperties   = errors.New("no key properties for this type")
	ErrNoValueProperties = errors.New("no value properties for this type")
	ErrNoLength          = errors.New("no length for this type")
	ErrWrongType         = errors.New("wrong type")
	ErrWrongKeyType      = errors.New("wrong key type")
)

var (
	uint8Type   = Type{0x01, 0x01}
	uint16Type  = Type{0x01, 0x02}
	uint32Type  = Type{0x01, 0x04}
	uint64Type  = Type{0x01, 0x08}
	uint256Type = Type{0x01, 0x20}
	int32Type   = Type{0x02, 0x04}
	int64Type   = Type{0x02, 0x08}
	arrayType   = Type{0x03, 0x00}
	mapType     = Type{0x04, 0x00}

	uint8Size   = 1
	uint16Size  = 2
	uint32Size  = 4
	uint64Size  = 8
	uint256Size = 32
)

type Type [2]byte

func (t Type) String() string {
	s := "Unknown"
	switch t {
	case uint8Type:
		s = "Uint8"
	case uint16Type:
		s = "Uint16"
	case uint32Type:
		s = "Uint32"
	case uint64Type:
		s = "Uint64"
	case uint256Type:
		s = "Uint256"
	case int32Type:
		s = "Int32"
	case int64Type:
		s = "Int64"
	case arrayType:
		s = "Array"
	case mapType:
		s = "Map"
	}

	return s
}

func (t Type) InnerType() uint8 {
	return t[0]
}

func (t Type) Size() uint8 {
	return t[1]
}

func (t Type) MarshalBinary() ([]byte, error) {
	return t[:], nil
}

func (t *Type) UnmarshalBinary(data []byte) error {
	copy(t[:], data[:2])
	return nil
}

type ValueProperties interface {
	Type() Type
	Length() (int64, error)
	KeyProperties() (ValueProperties, error)
	ValueProperties() (ValueProperties, error)

	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler
}

type ScalarProperties struct {
	internalType Type
}

func NewScalarProperties(internalType Type) *ScalarProperties {
	return &ScalarProperties{internalType: internalType}
}

func (s ScalarProperties) Type() Type {
	return s.internalType
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
	return s.internalType.MarshalBinary()
}

func (s *ScalarProperties) UnmarshalBinary(data []byte) error {
	return s.internalType.UnmarshalBinary(data)
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
	// marshal type
	typeB := a.Type().InnerType()

	// marshal valueProperties
	valueB, err := a.valueProperties.MarshalBinary()
	if err != nil {
		return nil, err
	}

	buf := make([]byte, uint8Size+uint64Size+len(valueB))

	// type
	buf[0] = typeB

	// length
	binary.BigEndian.PutUint64(buf[uint8Size:uint8Size+uint64Size], uint64(a.len))

	// valueProperties
	copy(buf[uint8Size+uint64Size:], valueB)

	return buf, nil
}

func (a *ArrayProperties) UnmarshalBinary(data []byte) error {
	// type
	tp := data[0]
	if tp != arrayType.InnerType() {
		return ErrWrongType
	}

	// length
	data = data[uint8Size:]
	l := int64(binary.BigEndian.Uint64(data[:uint64Size]))

	// valueProperties
	data = data[uint64Size:]
	vp := newProperties(data[0])
	err := vp.UnmarshalBinary(data[:])
	if err != nil {
		return err
	}

	a.len = l
	a.valueProperties = vp
	return nil
}

type MapProperties struct {
	keyProperties   ValueProperties
	valueProperties ValueProperties
}

func NewMapProperties(key, value ValueProperties) (*MapProperties, error) {
	if key.Type() == mapType {
		return nil, ErrWrongKeyType
	}

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
	// marshal type
	typeB := m.Type().InnerType()

	// marshal keyProperties
	keyB, err := m.keyProperties.MarshalBinary()
	if err != nil {
		return nil, err
	}

	// marshal valueProperties
	valueB, err := m.valueProperties.MarshalBinary()
	if err != nil {
		return nil, err
	}

	buf := make([]byte, uint8Size+len(keyB)+len(valueB))

	// put type
	buf[0] = typeB

	// put key
	off := uint8Size
	copy(buf[off:], keyB)

	// put value
	off += len(keyB)
	copy(buf[off:], valueB)

	return buf, nil
}

func (m *MapProperties) UnmarshalBinary(data []byte) error {
	// type
	tp := data[0]
	if tp != mapType.InnerType() {
		return ErrWrongType
	}

	// unmarshal key properties
	data = data[uint8Size:]
	kp := newProperties(data[0])
	err := kp.UnmarshalBinary(data)
	if err != nil {
		return err
	}

	// add size of key type
	off, err := calculatePropertiesSize(kp)
	if err != nil {
		return err
	}
	data = data[off:]

	// unmarshal value properties
	vp := newProperties(data[0])
	err = vp.UnmarshalBinary(data)
	if err != nil {
		return err
	}

	m.keyProperties = kp
	m.valueProperties = vp
	return nil
}

type SignatureVersion uint16

func (sv SignatureVersion) String() string {
	return fmt.Sprintf("%d", sv)
}

type Signature interface {
	Fields() []FieldDescriptor
	ReadFromStream(*StorageStream) (int, error)
	WriteToStream(*StorageStream) (int, error)
	Version() SignatureVersion
}

type Field struct {
	offset *big.Int
	length int64
}

func (f *Field) Offset() *big.Int {
	return f.offset
}

func (f *Field) Length() int64 {
	return f.length
}

type FieldDescriptor struct {
	name []byte
	vp   ValueProperties

	field Field
}

func (fp FieldDescriptor) MarshalBinary() (data []byte, err error) {
	// marshal value properties
	res, err := fp.vp.MarshalBinary()
	if err != nil {
		return nil, err
	}

	nameLen := len(fp.name)
	buf := make([]byte, uint8Size+nameLen+len(res))

	// put length of name
	buf[0] = byte(nameLen)

	// put name
	off := uint8Size
	copy(buf[off:], fp.name)

	// put value properties
	off += len(fp.name)
	copy(buf[off:], res)

	return buf, nil
}

func (fp *FieldDescriptor) UnmarshalBinary(data []byte) error {
	// get len of name
	nameLen := data[0]
	data = data[uint8Size:]

	// get name
	name := data[:nameLen]

	// get valueProperties
	data = data[nameLen:]
	vp := newProperties(data[0])
	err := vp.UnmarshalBinary(data)
	if err != nil {
		return err
	}

	fp.name = name
	fp.vp = vp
	return nil
}

func NewFieldDescriptor(fieldName []byte, v ValueProperties) FieldDescriptor {
	return FieldDescriptor{
		name: fieldName,
		vp:   v,
	}
}

type SignatureV1 struct {
	version           SignatureVersion
	fieldsDescriptors []FieldDescriptor
}

func NewSignatureV1(fd []FieldDescriptor) (*SignatureV1, error) {
	// version + fields count
	off := big.NewInt(int64(uint16Size + uint8Size))
	for _, descriptor := range fd {
		s, err := calculatePropertiesSize(descriptor.vp)
		if err != nil {
			return nil, err
		}

		descriptor.field = Field{
			offset: off,
			length: s,
		}
		off.Add(off, big.NewInt(s))
	}

	return &SignatureV1{
		version:           signatureV1,
		fieldsDescriptors: fd,
	}, nil
}

func (s SignatureV1) Fields() []FieldDescriptor {
	return s.fieldsDescriptors
}

func (s *SignatureV1) ReadFromStream(stream *StorageStream) (int, error) {
	pos := big.NewInt(0)

	// read the full signatureVersion
	sigVersionBuf := make([]byte, uint16Size)
	n, err := stream.ReadAt(sigVersionBuf, pos)
	if err != nil {
		return 0, err
	}
	pos.Add(pos, big.NewInt(int64(n)))

	// read the count of fields
	countBuf := make([]byte, uint8Size)
	n, err = stream.ReadAt(countBuf, pos)
	if err != nil {
		return 0, err
	}
	pos.Add(pos, big.NewInt(int64(n)))

	fieldsDescriptors := make([]FieldDescriptor, countBuf[0])
	uint8Buf := make([]byte, uint8Size)
	for i, _ := range fieldsDescriptors {
		currPos := new(big.Int).Set(pos)

		// read length of filed name
		n, err = stream.ReadAt(uint8Buf, currPos)
		if err != nil {
			return 0, err
		}
		currPos.Add(currPos, big.NewInt(int64(n)))

		// read name of filed
		nameBuf := make([]byte, uint8Buf[0])
		n, err = stream.ReadAt(nameBuf, currPos)
		if err != nil {
			return 0, err
		}
		currPos.Add(currPos, big.NewInt(int64(n)))

		err := calculatePropsEnd(stream, currPos)
		if err != nil {
			return 0, err
		}

		fieldBuf := make([]byte, currPos.Sub(currPos, pos).Uint64())
		n, err = stream.ReadAt(fieldBuf, pos)
		if err != nil {
			return 0, err
		}
		pos.Add(pos, big.NewInt(int64(n)))

		err = fieldsDescriptors[i].UnmarshalBinary(fieldBuf)
		if err != nil {
			return 0, err
		}

		uint8Buf = []byte{0x00}
	}

	s.version = SignatureVersion(binary.BigEndian.Uint16(sigVersionBuf))
	s.fieldsDescriptors = fieldsDescriptors

	return int(pos.Uint64()), nil
}

func (s SignatureV1) WriteToStream(stream *StorageStream) (int, error) {
	// SignatureVersion size + fields count
	buf := make([]byte, uint16Size+uint8Size)

	// write SignatureVersion
	binary.BigEndian.PutUint16(buf[:uint16Size], uint16(s.version))

	// write count of fieldsDescriptors
	buf[uint16Size] = uint8(len(s.fieldsDescriptors))

	// write fieldsDescriptors
	for _, field := range s.fieldsDescriptors {
		// marshal the fieldDescriptor
		b, err := field.MarshalBinary()
		if err != nil {
			return 0, err
		}

		// write the fieldDescriptor
		buf = append(buf, b...)
	}

	return stream.WriteAt(buf, big.NewInt(0))
}

func (s SignatureV1) Version() SignatureVersion {
	return s.version
}

func newProperties(tp uint8) ValueProperties {
	switch {
	case tp < arrayType.InnerType():
		return &ScalarProperties{}
	case tp == arrayType.InnerType():
		return &ArrayProperties{}
	case tp == mapType.InnerType():
		return &MapProperties{}
	default:
		return nil
	}
}

func isScalar(tp uint8) bool {
	return tp < arrayType.InnerType()
}

func isArray(tp uint8) bool {
	return tp == arrayType.InnerType()
}

func isMap(tp uint8) bool {
	return tp == mapType.InnerType()
}

func calculatePropsEnd(stream *StorageStream, currPos *big.Int) error {
	uint8Buf := make([]byte, uint8Size)
	// read Type
	n, err := stream.ReadAt(uint8Buf, currPos)
	if err != nil {
		return err
	}
	currPos.Add(currPos, big.NewInt(int64(n)))

	switch {
	case isScalar(uint8Buf[0]):
		// add size of Type
		currPos.Add(currPos, big.NewInt(int64(uint8Size)))
		return nil
	case isArray(uint8Buf[0]):
		// add array length
		currPos.Add(currPos, big.NewInt(int64(uint64Size)))
		return calculatePropsEnd(stream, currPos)
	case isMap(uint8Buf[0]):
		// get end of map key
		err := calculatePropsEnd(stream, currPos)
		if err != nil {
			return err
		}

		// get end of map value
		return calculatePropsEnd(stream, currPos)
	default:
		return ErrWrongType
	}
}

func calculatePropertiesSize(vp ValueProperties) (int64, error) {
	switch {
	case isScalar(vp.Type().InnerType()):
		// type
		return int64(uint16Size), nil
	case isArray(vp.Type().InnerType()):
		value, err := vp.ValueProperties()
		if err != nil {
			return 0, err
		}

		vpSize, err := calculatePropertiesSize(value)
		if err != nil {
			return 0, err
		}

		// type + length + element size
		return int64(uint8Size+uint64Size) + vpSize, nil
	case isMap(vp.Type().InnerType()):

		key, err := vp.KeyProperties()
		if err != nil {
			return 0, err
		}

		kSize, err := calculatePropertiesSize(key)
		if err != nil {
			return 0, err
		}

		value, err := vp.ValueProperties()
		if err != nil {
			return 0, err
		}

		vSize, err := calculatePropertiesSize(value)
		if err != nil {
			return 0, err
		}

		return kSize + vSize, nil
	default:
		return 0, ErrWrongType
	}
}
