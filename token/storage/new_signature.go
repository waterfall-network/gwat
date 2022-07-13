package storage

import (
	"encoding"
	"encoding/binary"
	"errors"
	"reflect"
	"strconv"
)

var (
	ErrNoKeyProperties   = errors.New("no key properties for this type")
	ErrNoValueProperties = errors.New("no value properties for this type")
	ErrNoLength          = errors.New("no length for this type")
)

type Type uint16

func (t *Type) String() string {
	return strconv.FormatUint(*t, 10)
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
)

type ValueProperties interface {
	Type() Type
	Length() (int, error)
	KeyProperties() (ValueProperties, error)
	ValueProperties() (ValueProperties, error)
	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler
}

type ScalarProperties Type

func (s ScalarProperties) Type() Type {
	return s
}

func (s ScalarProperties) Length() (int, error) {
	return 0, ErrNoLength
}

func (s ScalarProperties) KeyProperties() (ValueProperties, error) {
	return nil, ErrNoKeyProperties
}

func (s ScalarProperties) ValueProperties() (ValueProperties, error) {
	return nil, ErrNoValueProperties
}

func (s ScalarProperties) MarshalBinary() (data []byte, err error) {
	buf := make([]byte, int(reflect.TypeOf(uint16(0)).Size()))
	binary.BigEndian.PutUint16(buf, uint16(s))

	return buf, nil
}

func (s ScalarProperties) UnmarshalBinary(data []byte) error {
	s = ScalarProperties(binary.BigEndian.Uint16(data))

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

func (a *ArrayProperties) Length() (int, error) {
	return int(a.len), nil
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

	buf := make([]byte, len(res)+int(reflect.TypeOf(int64(0)).Size()))

	binary.BigEndian.PutUint64(buf, uint64(a.len))
	buf = append(buf, res...)

	return buf, nil
}

func (a *ArrayProperties) UnmarshalBinary(data []byte) error {
	a.len = int64(binary.BigEndian.Uint64(data[:int(reflect.TypeOf(int64(0)).Size())]))
	err := a.valueProperties.UnmarshalBinary(data[int(reflect.TypeOf(int64(0)).Size()):])
	if err != nil {
		return err
	}

	return nil
}

type MapProperties struct {
	keyProperties   ValueProperties
	valueProperties ValueProperties
}

func NewMapProperties(key, value ValueProperties) (*MapProperties, error) {
	m := MapProperties{
		keyProperties:   nil,
		valueProperties: nil,
	}

	arrayKey, ok := key.(*ArrayProperties)
	if !ok {
		scalarKey, ok := key.(ScalarProperties)
		if !ok {
			return nil, ErrWrongType
		}
		m.keyProperties = scalarKey
	}
	m.keyProperties = arrayKey

	arrayValue, ok := value.(*ArrayProperties)
	if !ok {
		scalarValue, ok := value.(ScalarProperties)
		if !ok {
			return nil, ErrWrongType
		}
		m.keyProperties = scalarValue
	}
	m.keyProperties = arrayValue

	return &m, nil
}

func (m *MapProperties) Type() Type {
	return mapType
}

func (m *MapProperties) Length() (int, error) {
	return 0, ErrNoLength
}

func (m *MapProperties) KeyProperties() (ValueProperties, error) {
	return m.keyProperties, nil
}

func (m *MapProperties) ValueProperties() (ValueProperties, error) {
	return m.valueProperties, nil
}

func (m *MapProperties) MarshalBinary() (data []byte, err error) {
	//TODO implement me
	panic("implement me")
}

func (m *MapProperties) UnmarshalBinary(data []byte) error {
	//TODO implement me
	panic("implement me")
}
