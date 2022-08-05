package storage

import (
	"encoding/binary"
	"errors"
	"fmt"
	"reflect"
)

var (
	ErrBadType             = errors.New("bad type")
	ErrMapIsNil            = errors.New("map is nil")
	ErrMapKeyIsNil         = errors.New("map key is nil")
	ErrDecoderIsNil        = errors.New("decoder is nil")
	ErrEncoderIsNil        = errors.New("encoder is nil")
	ErrFieldNotFound       = errors.New("field not found")
	ErrUnsupportedMapKey   = errors.New("unsupported map key type")
	ErrUnsupportedMapValue = errors.New("unsupported map value type")
	ErrValueTooLarge       = errors.New("value too large")
)

type Storage interface {
	ReadField(string, interface{}, ...Option) error
	WriteField(string, interface{}, ...Option) error
}

type Encoder func(interface{}) ([]byte, error)
type Decoder func([]byte, interface{}) error

var (
	ScalarEncoder = encodeScalar
	ScalarDecoder = decodeScalar

	ArrayEncoder = encodeArray
	ArrayDecoder = decodeArray
)

type options struct {
	mapKey interface{}

	decoder Decoder
	encoder Encoder
}

type Option func(*options)

func MapKey(key interface{}) func(opt *options) {
	return func(opt *options) {
		opt.mapKey = key
	}
}

func WithDecoder(decoder Decoder) func(opt *options) {
	return func(opt *options) {
		opt.decoder = decoder
	}
}

func WithEncoder(encoder Encoder) func(opt *options) {
	return func(opt *options) {
		opt.encoder = encoder
	}
}

type fieldsHolder map[string]Field
type mapsHolder map[string]*ByteMap

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

	_, err = sign.ReadFromStream(stream)
	if err != nil {
		return nil, err
	}

	fieldsHolder := fieldsHolder{}
	mapsHolder := mapsHolder{}

	fields := sign.Fields()
	for i, descriptor := range descriptors {
		switch {
		case isScalar(descriptor.vp.Type().Id()) || isArray(descriptor.vp.Type().Id()):
			fieldsHolder[string(descriptor.name)] = fields[i]
		case isMap(descriptor.vp.Type().Id()):
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

			var keyEncoder Encoder
			if isScalar(kp.Type().Id()) {
				keyEncoder = ScalarEncoder
			} else if isArray(kp.Type().Id()) {
				keyEncoder = ArrayEncoder
			} else {
				return nil, errors.New(fmt.Sprintf("%s: %s", ErrUnsupportedMapKey, kp.Type().String()))
			}

			var valEncoder Encoder
			var valDecoder Decoder
			if isScalar(vp.Type().Id()) {
				valEncoder = ScalarEncoder
				valDecoder = ScalarDecoder
			} else if isArray(vp.Type().Id()) {
				valEncoder = ArrayEncoder
				valDecoder = ArrayDecoder
			} else {
				return nil, errors.New(fmt.Sprintf("%s: %s", ErrUnsupportedMapValue, kp.Type().String()))
			}

			mapsHolder[string(descriptor.name)], err = newByteMap(
				descriptor.name,
				stream,
				keyEncoder,
				valEncoder,
				valDecoder,
				keySize,
				valueSize,
			)
			if err != nil {
				return nil, err
			}
		}
	}

	return &storage{
		stream:    stream,
		signature: nil,
		fields:    fieldsHolder,
		maps:      mapsHolder,
	}, nil
}

// ReadField reads a field from stream.
// Note: Requires Decoder Option if it is not a map. If map then requires MapKey
// Note: Support only one-dimensional arrays.
func (s *storage) ReadField(fieldName string, toPtr interface{}, opts ...Option) error {
	options := &options{}
	for _, opt := range opts {
		opt(options)
	}

	var buf []byte
	field, ok := s.fields[fieldName]
	if ok {
		buf = make([]byte, field.length)
		_, err := s.stream.ReadAt(buf, field.offset)
		if err != nil {
			return err
		}

		if options.decoder == nil {
			return ErrDecoderIsNil
		}

		return options.decoder(buf, toPtr)
	}

	mapField, ok := s.maps[fieldName]
	if ok {
		if mapField == nil {
			return ErrMapIsNil
		}

		if options.mapKey == nil {
			return ErrMapKeyIsNil
		}

		err := mapField.Load(options.mapKey, toPtr)
		if err != nil {
			return err
		}

		return nil
	}

	return ErrFieldNotFound
}

// WriteField writes a field to stream.
// Note: Requires Encoder Option if it is not a map. If map then requires MapKey
// Note: Support only one-dimensional arrays.
func (s *storage) WriteField(fieldName string, val interface{}, opts ...Option) error {
	options := &options{}
	for _, opt := range opts {
		opt(options)
	}

	field, ok := s.fields[fieldName]
	if ok {
		if options.encoder == nil {
			return ErrEncoderIsNil
		}

		buf, err := options.encoder(val)
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
		if mapField == nil {
			return ErrMapIsNil
		}

		if options.mapKey == nil {
			return ErrMapKeyIsNil
		}

		return mapField.Put(options.mapKey, val)
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
