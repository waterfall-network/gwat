package storage

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"reflect"

	"github.com/ethereum/go-ethereum/log"

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
)

type Storage interface {
	ApplyOptions(string, ...Option) error
	ReadField(string, interface{}) error
	WriteField(string, interface{}) error
}

type Encoder func(interface{}) ([]byte, error)
type Decoder func([]byte, interface{}) error

var (
	ScalarEncoder = encodeScalar
	ScalarDecoder = decodeScalar

	ArrayEncoder = encodeArray
	ArrayDecoder = decodeArray

	BigIntEncoder = encodeBigInt
	BigIntDecoder = decodeBigInt

	Uint256Encoder = encodeUint256
	Uint256Decoder = decodeUint256
)

type options struct {
	keyEncoder, valueEncoder Encoder
	keyDecoder, valueDecoder Decoder
}

type Option func(*options)

func WithDecoder(decoder Decoder) Option {
	return func(opt *options) {
		opt.valueDecoder = decoder
	}
}

func WithEncoder(encoder Encoder) Option {
	return func(opt *options) {
		opt.valueEncoder = encoder
	}
}

func WithMapDecoders(keyDecoder, valueDecoder Decoder) Option {
	return func(opt *options) {
		opt.keyDecoder = keyDecoder
		opt.valueDecoder = valueDecoder
	}
}

func WithMapEncoders(keyEncoder, valueEncoder Encoder) Option {
	return func(opt *options) {
		opt.keyEncoder = keyEncoder
		opt.valueEncoder = valueEncoder
	}
}

type KeyValuePair struct {
	key, value interface{}
}

type fieldWrapper struct {
	Field
	encoder Encoder
	decoder Decoder
}

func (f *fieldWrapper) encode(v interface{}) ([]byte, error) {
	return encode(f.encoder, v)
}

func (f *fieldWrapper) decode(buf []byte, ptr interface{}) error {
	return decode(f.decoder, buf, ptr)
}

type mapWrapper struct {
	*ByteMap
	keyEncoder, valueEncoder Encoder
	keyDecoder, valueDecoder Decoder
}

func (m *mapWrapper) encodeKey(v interface{}) ([]byte, error) {
	return encode(m.keyEncoder, v)
}

func (m *mapWrapper) decodeKey(buf []byte, ptr interface{}) error {
	return decode(m.keyDecoder, buf, ptr)
}

func (m *mapWrapper) encodeValue(v interface{}) ([]byte, error) {
	return encode(m.valueEncoder, v)
}

func (m *mapWrapper) decodeValue(buf []byte, ptr interface{}) error {
	return decode(m.valueDecoder, buf, ptr)
}

type fieldsHolder map[string]fieldWrapper
type mapsHolder map[string]mapWrapper

type storage struct {
	stream    *StorageStream
	signature Signature

	fields fieldsHolder
	maps   mapsHolder
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

	s := &storage{
		stream:    stream,
		signature: sign,
	}
	err = s.fillFields(sign)
	if err != nil {
		return nil, err
	}

	return s, nil
}

func ReadStorage(stream *StorageStream) (*storage, error) {
	sign := new(SignatureV1)
	_, err := sign.ReadFromStream(stream)
	if err != nil {
		return nil, err
	}

	s := &storage{
		stream:    stream,
		signature: sign,
	}
	err = s.fillFields(sign)
	if err != nil {
		return nil, err
	}

	return s, nil
}

func (s *storage) fillFields(sign Signature) error {
	fieldsHolder := fieldsHolder{}
	mapsHolder := mapsHolder{}

	fields := sign.Fields()
	for i, descriptor := range sign.FieldsDescriptors() {
		switch {
		case isScalar(descriptor.vp.Type().Id()) || isArray(descriptor.vp.Type().Id()):
			valEncoder, valDecoder := getDefaultEncoderAndDecoder(descriptor.vp.Type().Id())
			fieldsHolder[descriptor.Name()] = fieldWrapper{
				Field:   fields[i],
				encoder: valEncoder,
				decoder: valDecoder,
			}
		case isMap(descriptor.vp.Type().Id()):
			kp, err := descriptor.vp.KeyProperties()
			if err != nil {
				return err
			}

			keySize, err := calculateFieldLength(kp)
			if err != nil {
				return err
			}

			vp, err := descriptor.vp.ValueProperties()
			if err != nil {
				return err
			}
			valueSize, err := calculateFieldLength(vp)
			if err != nil {
				return err
			}

			keyEncoder, keyDecoder := getDefaultEncoderAndDecoder(kp.Type().Id())
			valEncoder, valDecoder := getDefaultEncoderAndDecoder(vp.Type().Id())

			byteMap, err := newByteMap(
				[]byte(descriptor.Name()),
				s.stream,
				keySize,
				valueSize,
			)
			if err != nil {
				return err
			}

			mapsHolder[descriptor.Name()] = mapWrapper{
				ByteMap:      byteMap,
				keyEncoder:   keyEncoder,
				valueEncoder: valEncoder,
				keyDecoder:   keyDecoder,
				valueDecoder: valDecoder,
			}
		}
	}

	s.fields = fieldsHolder
	s.maps = mapsHolder

	return nil
}

func (s *storage) ApplyOptions(fieldName string, opts ...Option) error {
	options := &options{}
	for _, opt := range opts {
		opt(options)
	}

	field, ok := s.fields[fieldName]
	if ok {
		if options.valueEncoder != nil {
			field.encoder = options.valueEncoder
		}
		if options.valueDecoder != nil {
			field.decoder = options.valueDecoder
		}
	}

	mapField, ok := s.maps[fieldName]
	if ok {
		if options.keyEncoder != nil {
			mapField.keyEncoder = options.keyEncoder
		}
		if options.keyDecoder != nil {
			mapField.keyDecoder = options.keyDecoder
		}
		if options.valueEncoder != nil {
			mapField.valueEncoder = options.valueEncoder
		}
		if options.valueDecoder != nil {
			mapField.valueDecoder = options.valueDecoder
		}
	}

	return ErrFieldNotFound
}

// ReadField reads a field from stream.
// Note: Support only one-dimensional arrays.
// Note: for maps expects pointer to KeyValuePair struct
func (s *storage) ReadField(fieldName string, toPtr interface{}) error {
	var buf []byte
	field, ok := s.fields[fieldName]
	if ok {
		buf = make([]byte, field.length)
		_, err := s.stream.ReadAt(buf, field.offset)
		if err != nil {
			return err
		}

		return field.decode(buf, toPtr)
	}

	mapField, ok := s.maps[fieldName]
	if ok {
		kvPair, ok := toPtr.(*KeyValuePair)
		if !ok {
			return ErrBadType
		}

		keyB, err := mapField.encodeKey(kvPair.key)
		if err != nil {
			return err
		}

		res, err := mapField.Get(keyB)
		if err != nil {
			return err
		}

		return mapField.decodeValue(res, toPtr)
	}

	return ErrFieldNotFound
}

// WriteField writes a field to stream.
// Note: Support only one-dimensional arrays.
// Note: for maps expects pointer to KeyValuePair struct
func (s *storage) WriteField(fieldName string, val interface{}) error {
	field, ok := s.fields[fieldName]
	if ok {
		buf, err := field.encode(val)
		if err != nil {
			return err
		}

		if uint64(len(buf)) > field.length {
			return ErrValueTooLarge
		}

		_, err = s.stream.WriteAt(buf, field.offset)
		return err
	}

	mapField, ok := s.maps[fieldName]
	if ok {
		kvPair, ok := val.(*KeyValuePair)
		if !ok {
			return ErrBadType
		}

		keyB, err := mapField.encodeKey(kvPair.key)
		if err != nil {
			return err
		}

		valueB, err := mapField.encodeValue(kvPair.value)
		if err != nil {
			return err
		}

		return mapField.Put(keyB, valueB)
	}

	return ErrFieldNotFound
}

func encodeScalar(v interface{}) ([]byte, error) {
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
	case []byte:
		buf = v.([]byte)
	default:
		return nil, errors.New(fmt.Sprintf("%s: %s", ErrBadType, v))
	}

	return buf, nil
}

func decodeScalar(buf []byte, vr interface{}) error {
	switch vr.(type) {
	case *uint8:
		*vr.(*uint8) = buf[0]
	case *uint16:
		*vr.(*uint16) = binary.BigEndian.Uint16(buf)
	case *uint32:
		*vr.(*uint32) = binary.BigEndian.Uint32(buf)
	case *uint64:
		*vr.(*uint64) = binary.BigEndian.Uint64(buf)
	case *int32:
		*vr.(*int32) = int32(binary.BigEndian.Uint32(buf))
	case *int64:
		*vr.(*int64) = int64(binary.BigEndian.Uint64(buf))
	case *string:
		*vr.(*string) = string(buf)
	case *[]byte:
		*vr.(*[]byte) = buf
	default:
		return errors.New(fmt.Sprintf("%s: %s", ErrBadType, vr))
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
			res, err = encodeScalar(val.Index(i))
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
	if !(tp.Kind() == reflect.Slice || tp.Kind() == reflect.Array) {
		return ErrBadType
	}

	switch arrPtr.(type) {
	case *[]byte:
		*arrPtr.(*[]byte) = buf
		return nil
	default:
		var err error
		elemSize := tp.Elem().Size()
		val := reflect.ValueOf(arrPtr).Elem()
		for len(buf) > 0 {
			newElem := reflect.New(val.Type())
			err = decodeScalar(buf[:elemSize], newElem.Interface())
			if err != nil {
				arrPtr = nil
				return err
			}
			val = reflect.Append(val, reflect.ValueOf(newElem))

			buf = buf[elemSize:]
		}

		return nil
	}
}

func encodeUint256(val interface{}) ([]byte, error) {
	v, ok := val.(uint256.Int)
	if !ok {
		return nil, ErrBadType
	}

	return v.Bytes(), nil
}

func decodeUint256(buf []byte, toPtr interface{}) error {
	ptr, ok := toPtr.(*uint256.Int)
	if !ok {
		return ErrBadType
	}

	v := new(uint256.Int).SetBytes(buf)
	*ptr = *v

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

func getDefaultEncoderAndDecoder(tp uint8) (Encoder, Decoder) {
	if isScalar(tp) {
		return ScalarEncoder, ScalarDecoder
	}

	if isArray(tp) {
		return ArrayEncoder, ArrayDecoder
	}

	log.Warn("Cannot set default encoder and decoder")
	return nil, nil
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
