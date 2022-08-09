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
	ErrNameTooBig        = errors.New("name too big")
	ErrTooManyFields     = errors.New("too many fields")
)

var (
	Uint8Type   = Type{0x01, 0x01}
	Uint16Type  = Type{0x01, 0x02}
	Uint32Type  = Type{0x01, 0x04}
	Uint64Type  = Type{0x01, 0x08}
	Uint256Type = Type{0x01, 0x20}
	Int32Type   = Type{0x02, 0x04}
	Int64Type   = Type{0x02, 0x08}
	ArrayType   = Type{0x03, 0x00}
	MapType     = Type{0x04, 0x00}

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
	case Uint8Type:
		s = "Uint8"
	case Uint16Type:
		s = "Uint16"
	case Uint32Type:
		s = "Uint32"
	case Uint64Type:
		s = "Uint64"
	case Uint256Type:
		s = "Uint256"
	case Int32Type:
		s = "Int32"
	case Int64Type:
		s = "Int64"
	case ArrayType:
		s = "Array"
	case MapType:
		s = "Map"
	}

	return s
}

func (t Type) Id() uint8 {
	return t[0]
}

func (t Type) Size() uint8 {
	return t[1]
}

func (t Type) Equal(gotT Type) bool {
	return t[0] == gotT[0] && t[1] == gotT[1]
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
	Length() (uint64, error)
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

func (s ScalarProperties) Length() (uint64, error) {
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
	len             uint64
}

func NewArrayProperties(t *ScalarProperties, l uint64) *ArrayProperties {
	return &ArrayProperties{
		valueProperties: t,
		len:             l,
	}
}

func (a *ArrayProperties) Type() Type {
	return ArrayType
}

func (a *ArrayProperties) Length() (uint64, error) {
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
	typeB := a.Type().Id()

	// marshal valueProperties
	valueB, err := a.valueProperties.MarshalBinary()
	if err != nil {
		return nil, err
	}

	buf := make([]byte, uint8Size+uint64Size+len(valueB))

	// type
	buf[0] = typeB

	// length
	binary.BigEndian.PutUint64(buf[uint8Size:uint8Size+uint64Size], a.len)

	// valueProperties
	copy(buf[uint8Size+uint64Size:], valueB)

	return buf, nil
}

func (a *ArrayProperties) UnmarshalBinary(data []byte) error {
	// type
	tp := data[0]
	if tp != ArrayType.Id() {
		return ErrWrongType
	}

	// length
	data = data[uint8Size:]
	l := binary.BigEndian.Uint64(data[:uint64Size])

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
	if key.Type() == MapType {
		return nil, ErrWrongKeyType
	}

	return &MapProperties{
		keyProperties:   key,
		valueProperties: value,
	}, nil
}

func (m *MapProperties) Type() Type {
	return MapType
}

func (m *MapProperties) Length() (uint64, error) {
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
	typeB := m.Type().Id()

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
	if tp != MapType.Id() {
		return ErrWrongType
	}

	// check that key is not map
	data = data[uint8Size:]
	if data[0] == MapType.Id() {
		return ErrWrongType
	}

	// unmarshal key properties
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
	Fields() []FieldInfo
	ReadFromStream(*StorageStream) (int, error)
	WriteToStream(*StorageStream) (int, error)
	Version() SignatureVersion
}

type FieldInfo struct {
	descriptor FieldDescriptor

	offset *big.Int
	length uint64
}

func (f *FieldInfo) Descriptor() FieldDescriptor {
	return f.descriptor
}

func (f *FieldInfo) Offset() *big.Int {
	return f.offset
}

func (f *FieldInfo) Length() uint64 {
	return f.length
}

type FieldDescriptor struct {
	name []byte
	vp   ValueProperties
}

func NewFieldDescriptor(fieldName []byte, v ValueProperties) (*FieldDescriptor, error) {
	if len(fieldName) > int(^uint8(0)) {
		return nil, ErrNameTooBig
	}

	return &FieldDescriptor{
		name: fieldName,
		vp:   v,
	}, nil
}

func (fd FieldDescriptor) Name() string {
	return string(fd.name)
}

func (fd FieldDescriptor) MarshalBinary() (data []byte, err error) {
	// marshal value properties
	res, err := fd.vp.MarshalBinary()
	if err != nil {
		return nil, err
	}

	nameLen := len(fd.name)
	buf := make([]byte, uint8Size+nameLen+len(res))

	// put length of name
	buf[0] = byte(nameLen)

	// put name
	off := uint8Size
	copy(buf[off:], fd.name)

	// put value properties
	off += len(fd.name)
	copy(buf[off:], res)

	return buf, nil
}

func (fd *FieldDescriptor) UnmarshalBinary(data []byte) error {
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

	fd.name = name
	fd.vp = vp
	return nil
}

type SignatureV1 struct {
	version           SignatureVersion
	fieldsDescriptors []FieldDescriptor
	fields            []FieldInfo
}

func NewSignatureV1(fd []FieldDescriptor) (Signature, error) {
	if len(fd) > int(^uint8(0)) {
		return nil, ErrTooManyFields
	}

	// version + fields count
	totalSize := uint64(uint16Size + uint8Size)
	fields := make([]FieldInfo, len(fd))
	for i, descriptor := range fd {
		fields[i].descriptor = descriptor

		l, err := calculateFieldLength(descriptor.vp)
		if err != nil {
			return nil, err
		}
		fields[i].length = l

		s, err := calculatePropertiesSize(descriptor.vp)
		if err != nil {
			return nil, err
		}
		totalSize += s + uint64(uint8Size+len(descriptor.name))
	}

	calculateFieldsOffset(totalSize, fields)

	return &SignatureV1{
		version:           signatureV1,
		fieldsDescriptors: fd,
		fields:            fields,
	}, nil
}

func (s SignatureV1) Fields() []FieldInfo {
	return s.fields
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

	uint8Buf := make([]byte, uint8Size)
	fields := make([]FieldInfo, countBuf[0])
	fieldsDescriptors := make([]FieldDescriptor, countBuf[0])
	for i := range fieldsDescriptors {
		// read length of field name
		n, err = stream.ReadAt(uint8Buf, pos)
		if err != nil {
			return 0, err
		}
		pos.Add(pos, big.NewInt(int64(n)))

		// read name of field
		nameBuf := make([]byte, uint8Buf[0])
		n, err = stream.ReadAt(nameBuf, pos)
		if err != nil {
			return 0, err
		}
		pos.Add(pos, big.NewInt(int64(n)))

		end, err := calculatePropsEnd(stream, pos)
		if err != nil {
			return 0, err
		}

		fieldBuf := make([]byte, uint64(uint8Size+len(nameBuf))+end.Sub(end, pos).Uint64())
		// copy name length
		fieldBuf[0] = uint8Buf[0]
		// copy name
		copy(fieldBuf[uint8Size:uint8Size+len(nameBuf)], nameBuf[:])
		// read fields
		n, err = stream.ReadAt(fieldBuf[uint8Size+len(nameBuf):], pos)
		if err != nil {
			return 0, err
		}
		pos.Add(pos, big.NewInt(int64(n)))

		err = fieldsDescriptors[i].UnmarshalBinary(fieldBuf)
		if err != nil {
			return 0, err
		}

		// save length of the field
		fields[i].length, err = calculateFieldLength(fieldsDescriptors[i].vp)
		if err != nil {
			return 0, err
		}

		uint8Buf = []byte{0x00}
	}

	// save offset of fields
	calculateFieldsOffset(pos.Uint64(), fields)

	*s = SignatureV1{
		version:           SignatureVersion(binary.BigEndian.Uint16(sigVersionBuf)),
		fieldsDescriptors: fieldsDescriptors,
		fields:            fields,
	}

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
	case isScalar(tp):
		return &ScalarProperties{}
	case isArray(tp):
		return &ArrayProperties{}
	case isMap(tp):
		return &MapProperties{}
	default:
		return nil
	}
}

func isScalar(tp uint8) bool {
	return tp < ArrayType.Id()
}

func isArray(tp uint8) bool {
	return tp == ArrayType.Id()
}

func isMap(tp uint8) bool {
	return tp == MapType.Id()
}

func calculatePropsEnd(stream *StorageStream, currPos *big.Int) (*big.Int, error) {
	end := new(big.Int).Set(currPos)

	uint8Buf := make([]byte, uint8Size)
	// read Type
	n, err := stream.ReadAt(uint8Buf, currPos)
	if err != nil {
		return nil, err
	}
	end.Add(end, big.NewInt(int64(n)))

	switch {
	case isScalar(uint8Buf[0]):
		// add size of Type
		end.Add(end, big.NewInt(int64(uint8Size)))
		return end, nil
	case isArray(uint8Buf[0]):
		// add array length
		end.Add(end, big.NewInt(int64(uint64Size)))
		return calculatePropsEnd(stream, end)
	case isMap(uint8Buf[0]):
		// get end of map key
		end, err := calculatePropsEnd(stream, end)
		if err != nil {
			return nil, err
		}

		// get end of map value
		return calculatePropsEnd(stream, end)
	default:
		return nil, ErrWrongType
	}
}

func calculatePropertiesSize(vp ValueProperties) (uint64, error) {
	switch {
	case isScalar(vp.Type().Id()):
		// type
		return uint64(uint16Size), nil
	case isArray(vp.Type().Id()):
		value, err := vp.ValueProperties()
		if err != nil {
			return 0, err
		}

		vpSize, err := calculatePropertiesSize(value)
		if err != nil {
			return 0, err
		}

		// type + length + element size
		return uint64(uint8Size+uint64Size) + vpSize, nil
	case isMap(vp.Type().Id()):
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

		return uint64(uint8Size) + kSize + vSize, nil
	default:
		return 0, ErrWrongType
	}
}

func calculateFieldLength(vp ValueProperties) (uint64, error) {
	switch {
	case isScalar(vp.Type().Id()):
		return uint64(vp.Type().Size()), nil
	case isArray(vp.Type().Id()):
		value, err := vp.ValueProperties()
		if err != nil {
			return 0, err
		}

		vpSize, err := calculateFieldLength(value)
		if err != nil {
			return 0, err
		}

		l, err := vp.Length()
		if err != nil {
			return 0, err
		}

		// length * element size
		return l * vpSize, nil
	case isMap(vp.Type().Id()):
		return 0, nil
	default:
		return 0, ErrWrongType
	}
}

func calculateFieldsOffset(start uint64, fields []FieldInfo) {
	for i := range fields {
		fields[i].offset = big.NewInt(int64(start))
		start += fields[i].length
	}
}
