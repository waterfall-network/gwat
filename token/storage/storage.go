package storage

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"reflect"

	"github.com/holiman/uint256"
)

var (
	ErrBadType              = errors.New("bad type")
	ErrCannotEncodeNilValue = errors.New("cannot encode nil value")
	ErrCannotDecodeNilValue = errors.New("cannot decode nil value")
	ErrDecoderIsNotSet      = errors.New("decoder is not set")
	ErrEncoderIsNotSet      = errors.New("encoder is not set")
	ErrFieldNotFound        = errors.New("field not found")
	ErrValueTooLarge        = errors.New("value too large")
	ErrBadFieldDescriptor   = errors.New("cannot create field from field descriptor")
	ErrUnknownType          = errors.New("unknown type")
)

type FieldEntry interface {
	Read(*StorageStream, interface{}) error
	Write(*StorageStream, interface{}) error
}

type Storage interface {
	ReadField(string, interface{}) error
	WriteField(string, interface{}) error
}

type Encoder func(interface{}) ([]byte, error)
type Decoder func([]byte, interface{}) error

var (
	DefaultScalarEncoder = encodeScalar
	DefaultScalarDecoder = decodeScalar

	DefaultArrayEncoder = encodeArray
	DefaultArrayDecoder = decodeArray

	BigIntEncoder = encodeBigInt
	BigIntDecoder = decodeBigInt

	Uint256Encoder = encodeUint256
	Uint256Decoder = decodeUint256
)

type KeyValuePair struct {
	key, value interface{}
}

type storage struct {
	stream    *StorageStream
	signature Signature

	fields fieldsHolder
}

func NewStorage(stream *StorageStream, descriptors []FieldDescriptor) (*storage, error) {
	sign, err := NewSignatureV1(descriptors)
	if err != nil {
		return nil, err
	}

	_, err = sign.WriteToStream(stream)
	if err != nil {
		return nil, err
	}

	fields, err := fillFields(sign)
	if err != nil {
		return nil, err
	}

	return &storage{
		stream:    stream,
		signature: sign,
		fields:    fields,
	}, nil
}

func ReadStorage(stream *StorageStream) (*storage, error) {
	sign := new(SignatureV1)
	_, err := sign.ReadFromStream(stream)
	if err != nil {
		return nil, err
	}

	fields, err := fillFields(sign)
	if err != nil {
		return nil, err
	}

	return &storage{
		stream:    stream,
		signature: sign,
		fields:    fields,
	}, nil
}

func fillFields(sign Signature) (fieldsHolder, error) {
	var err error
	fields := fieldsHolder{}
	for _, info := range sign.Fields() {
		fields[info.Descriptor().Name()], err = NewFieldEntry(info)
		if err != nil {
			return nil, err
		}
	}

	return fields, nil
}

// ReadField reads a field from stream.
// Note: Support only one-dimensional arrays.
// Note: for maps expects pointer to KeyValuePair struct
func (s *storage) ReadField(fieldName string, toPtr interface{}) error {
	field, ok := s.fields[fieldName]
	if !ok {
		return ErrFieldNotFound
	}

	return field.Read(s.stream, toPtr)
}

// WriteField writes a field to stream.
// Note: Support only one-dimensional arrays.
// Note: for maps expects pointer to KeyValuePair struct
func (s *storage) WriteField(fieldName string, val interface{}) error {
	field, ok := s.fields[fieldName]
	if !ok {
		return ErrFieldNotFound
	}

	return field.Write(s.stream, val)
}

func NewFieldEntry(fieldInfo FieldInfo) (FieldEntry, error) {
	descriptor := fieldInfo.Descriptor()
	tp := descriptor.vp.Type()
	id := tp.Id()
	switch {
	case isMap(id):
		kp, err := descriptor.vp.KeyProperties()
		if err != nil {
			return nil, err
		}
		keySize, err := calculateFieldLength(kp)
		if err != nil {
			return nil, err
		}

		vp, err := descriptor.vp.ValueProperties()
		if err != nil {
			return nil, err
		}
		valueSize, err := calculateFieldLength(vp)
		if err != nil {
			return nil, err
		}

		keyEncoder, keyDecoder, err := getDefaultEncoderAndDecoder(kp.Type())
		if err != nil {
			return nil, err
		}

		valEncoder, valDecoder, err := getDefaultEncoderAndDecoder(vp.Type())
		if err != nil {
			return nil, err
		}

		byteMap, err := newByteMap([]byte(descriptor.Name()), keySize, valueSize)
		if err != nil {
			return nil, err
		}

		return &mapEntry{
			ByteMap:      byteMap,
			keyEncoder:   keyEncoder,
			valueEncoder: valEncoder,
			keyDecoder:   keyDecoder,
			valueDecoder: valDecoder,
		}, nil
	default:
		encoder, decoder, err := getDefaultEncoderAndDecoder(tp)
		if err != nil {
			return nil, err
		}

		return &fieldEntry{
			FieldInfo: fieldInfo,
			encoder:   encoder,
			decoder:   decoder,
		}, nil
	}
}

type fieldsHolder map[string]FieldEntry

type fieldEntry struct {
	FieldInfo
	encoder Encoder
	decoder Decoder
}

func (f *fieldEntry) Read(stream *StorageStream, toPtr interface{}) error {
	buf := make([]byte, f.Length())
	_, err := stream.ReadAt(buf, f.offset)
	if err != nil {
		return err
	}

	return f.decode(buf, toPtr)
}

func (f *fieldEntry) Write(stream *StorageStream, val interface{}) error {
	buf, err := f.encode(val)
	if err != nil {
		return err
	}

	if uint64(len(buf)) > f.Length() {
		return ErrValueTooLarge
	}

	_, err = stream.WriteAt(buf, f.offset)
	return err
}

func (f *fieldEntry) encode(v interface{}) ([]byte, error) {
	return encode(f.encoder, v)
}

func (f *fieldEntry) decode(buf []byte, ptr interface{}) error {
	return decode(f.decoder, buf, ptr)
}

type mapEntry struct {
	*ByteMap
	keyEncoder, valueEncoder Encoder
	keyDecoder, valueDecoder Decoder
}

//Read expects pointer to KeyValuePair struct
func (m *mapEntry) Read(s *StorageStream, toPtr interface{}) error {
	kvPair, ok := toPtr.(*KeyValuePair)
	if !ok {
		return ErrBadType
	}

	keyB, err := m.encodeKey(kvPair.key)
	if err != nil {
		return err
	}

	res, err := m.Get(s, keyB)
	if err != nil {
		return err
	}

	return m.decodeValue(res, kvPair.value)
}

//Write expects pointer to KeyValuePair struct
func (m *mapEntry) Write(s *StorageStream, val interface{}) error {
	kvPair, ok := val.(*KeyValuePair)
	if !ok {
		return ErrBadType
	}

	keyB, err := m.encodeKey(kvPair.key)
	if err != nil {
		return err
	}

	valueB, err := m.encodeValue(kvPair.value)
	if err != nil {
		return err
	}

	return m.Put(s, keyB, valueB)
}

func (m *mapEntry) encodeKey(v interface{}) ([]byte, error) {
	return encode(m.keyEncoder, v)
}

func (m *mapEntry) decodeKey(buf []byte, ptr interface{}) error {
	return decode(m.keyDecoder, buf, ptr)
}

func (m *mapEntry) encodeValue(v interface{}) ([]byte, error) {
	return encode(m.valueEncoder, v)
}

func (m *mapEntry) decodeValue(buf []byte, ptr interface{}) error {
	return decode(m.valueDecoder, buf, ptr)
}

func encodeScalar(v interface{}) ([]byte, error) {
	var err error
	var buf []byte
	switch v.(type) {
	case uint8:
		buf = []byte{v.(uint8)}
	case uint16:
		buf = make([]byte, 2)
		binary.BigEndian.PutUint16(buf, v.(uint16))
	case uint32:
		buf = make([]byte, 4)
		binary.BigEndian.PutUint32(buf, v.(uint32))
	case uint64:
		buf = make([]byte, 8)
		binary.BigEndian.PutUint64(buf, v.(uint64))
	case int32:
		buf = make([]byte, 4)
		binary.BigEndian.PutUint64(buf, uint64(v.(int64)))
	case int64:
		buf = make([]byte, 8)
		binary.BigEndian.PutUint64(buf, uint64(v.(int64)))
	case string:
		buf = []byte(v.(string))
	default:
		buf, err = encodeUint256(v)
		if err == nil {
			break
		}

		return nil, fmt.Errorf("%s: %T", ErrBadType, v)
	}

	return buf, nil
}

func decodeScalar(buf []byte, vPtr interface{}) error {
	tp := reflect.TypeOf(vPtr)
	if tp.Kind() != reflect.Ptr {
		return ErrBadType
	}

	switch vPtr.(type) {
	case *uint8:
		*vPtr.(*uint8) = buf[0]
	case *uint16:
		*vPtr.(*uint16) = binary.BigEndian.Uint16(buf)
	case *uint32:
		*vPtr.(*uint32) = binary.BigEndian.Uint32(buf)
	case *uint64:
		*vPtr.(*uint64) = binary.BigEndian.Uint64(buf)
	case *int32:
		*vPtr.(*int32) = int32(binary.BigEndian.Uint32(buf))
	case *int64:
		*vPtr.(*int64) = int64(binary.BigEndian.Uint64(buf))
	case *string:
		*vPtr.(*string) = string(buf)
	default:
		err := decodeUint256(buf, vPtr)
		if err == nil {
			break
		}

		return fmt.Errorf("%s: %T", ErrBadType, vPtr)
	}

	return nil
}

func encodeArray(arr interface{}) ([]byte, error) {
	tp := reflect.TypeOf(arr)
	if !(tp.Kind() == reflect.Slice || tp.Kind() == reflect.Array) {
		return nil, ErrBadType
	}

	switch arr.(type) {
	case []byte:
		return arr.([]byte), nil
	default:
		val := reflect.ValueOf(arr)
		buf := make([]byte, 0, int(tp.Elem().Size())*val.Len())

		var res []byte
		var err error
		for i := 0; i < val.Len(); i++ {
			res, err = encodeScalar(val.Index(i).Interface())
			if err != nil {
				return nil, err
			}

			buf = append(buf, res...)
		}

		return buf, nil
	}
}

func decodeArray(buf []byte, arrPtr interface{}) error {
	tp := reflect.TypeOf(arrPtr)
	if tp.Kind() != reflect.Ptr {
		return ErrBadType
	}

	valuePtr := reflect.ValueOf(arrPtr)
	value := valuePtr.Elem()

	switch arrPtr.(type) {
	case *[]byte:
		*arrPtr.(*[]byte) = buf
		return nil
	default:
		tp := reflect.TypeOf(value.Interface())
		if !(tp.Kind() == reflect.Slice || tp.Kind() == reflect.Array) {
			return ErrBadType
		}

		elemType := tp.Elem()
		elemSize := elemType.Size()

		if len(buf)%int(elemSize) != 0 {
			return ErrBadType
		}

		var newElem reflect.Value
		if elemType.Kind() == reflect.Ptr {
			newElem = reflect.New(elemType.Elem())
		} else {
			newElem = reflect.New(elemType)
		}

		var err error
		for len(buf) > 0 {
			err = decodeScalar(buf[:elemSize], newElem.Interface())
			if err != nil {
				arrPtr = nil
				return err
			}

			if elemType.Kind() == reflect.Ptr {
				value.Set(reflect.Append(value, newElem))
			} else {
				value.Set(reflect.Append(value, newElem.Elem()))
			}
			buf = buf[elemSize:]
		}

		return nil
	}
}

func encodeUint256(val interface{}) ([]byte, error) {
	v, ok := val.(uint256.Int)
	if !ok {
		vr, ok := val.(*uint256.Int)
		if !ok {
			return nil, ErrBadType
		}
		v = *vr
	}

	return v.Bytes(), nil
}

func decodeUint256(buf []byte, toPtr interface{}) error {
	_, ok := toPtr.(*uint256.Int)
	if !ok {
		return ErrBadType
	}

	toPtr = new(uint256.Int).SetBytes(buf)
	return nil
}

func encodeBigInt(val interface{}) ([]byte, error) {
	v, ok := val.(big.Int)
	if !ok {
		return nil, ErrBadType
	}

	return v.Bytes(), nil
}

func decodeBigInt(buf []byte, toPtr interface{}) error {
	ptr, ok := toPtr.(*big.Int)
	if !ok {
		return ErrBadType
	}

	v := new(big.Int).SetBytes(buf)
	*ptr = *v

	return nil
}

func getDefaultEncoderAndDecoder(tp Type) (Encoder, Decoder, error) {
	if Uint256Type.Equal(tp) {
		return Uint256Encoder, Uint256Decoder, nil
	}

	if isScalar(tp.Id()) {
		return DefaultScalarEncoder, DefaultScalarDecoder, nil
	}

	if isArray(tp.Id()) {
		return DefaultArrayEncoder, DefaultArrayDecoder, nil
	}

	return nil, nil, ErrUnknownType
}

func encode(encoder Encoder, v interface{}) ([]byte, error) {
	if v == nil {
		return nil, ErrCannotEncodeNilValue
	}

	if encoder == nil {
		return nil, ErrEncoderIsNotSet
	}

	return encoder(v)
}

func decode(decoder Decoder, buf []byte, ptr interface{}) error {
	if ptr == nil {
		return ErrCannotDecodeNilValue
	}

	if decoder == nil {
		return ErrDecoderIsNotSet
	}

	return decoder(buf, ptr)
}
