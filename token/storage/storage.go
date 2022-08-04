package storage

import (
	"encoding/binary"
	"errors"
	"fmt"
	"reflect"
)

var (
	ErrBadType       = errors.New("bad type")
	ErrFieldNotFound = errors.New("field not found")
	ErrMapIsNil      = errors.New("map is nil")
)

type Storage interface {
	ReadScalar(string, interface{}) error
	WriteScalar(string, interface{}) error

	ReadArray(string, interface{}) error
	WriteArray(string, interface{}) error

	ReadFromMap(string, interface{}) (interface{}, error)
	WriteToMap(string, interface{}, interface{}) error
}

type scalarFields map[string]Field
type arrayFields map[string]Field
type mapFields map[string]*ByteMap

type storage struct {
	stream    *StorageStream
	signature Signature

	scalarFields scalarFields
	arrayFields  arrayFields
	mapFields    mapFields
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

	scalarFields := scalarFields{}
	arrayFields := arrayFields{}
	mapFields := mapFields{}

	fields := sign.Fields()
	for i, descriptor := range descriptors {
		switch {
		case isScalar(descriptor.vp.Type().Id()):
			scalarFields[string(descriptor.name)] = fields[i]
		case isArray(descriptor.vp.Type().Id()):
			arrayFields[string(descriptor.name)] = fields[i]
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

			mapFields[string(descriptor.name)], err = newByteMap(
				descriptor.name,
				stream,
				DefaultEncoder,
				DefaultEncoder,
				keySize,
				valueSize,
			)
			if err != nil {
				return nil, err
			}
		}
	}

	return &storage{
		stream:       stream,
		signature:    nil,
		scalarFields: scalarFields,
		arrayFields:  arrayFields,
		mapFields:    mapFields,
	}, nil
}

func (s *storage) ReadScalar(fieldName string, scalarPtr interface{}) error {
	field, ok := s.scalarFields[fieldName]
	if !ok {
		return ErrFieldNotFound
	}

	buf := make([]byte, field.length)
	_, err := s.stream.ReadAt(buf, field.offset)
	if err != nil {
		return err
	}

	return bytesToScalar(buf, scalarPtr)
}

func (s *storage) WriteScalar(fieldName string, v interface{}) error {
	field, ok := s.scalarFields[fieldName]
	if !ok {
		return ErrFieldNotFound
	}

	res, err := scalarToBytes(v)
	if err != nil {
		return err
	}

	_, err = s.stream.WriteAt(res, field.offset)
	return err
}

// ReadArray read bytes to an array.
// Note: support only one-dimensional arrays
func (s *storage) ReadArray(fieldName string, arrPtr interface{}) error {
	tp := reflect.TypeOf(arrPtr)
	if !(tp.Kind() == reflect.Slice || tp.Kind() == reflect.Array) {
		return ErrBadType
	}

	field, ok := s.arrayFields[fieldName]
	if !ok {
		return ErrFieldNotFound
	}

	buf := make([]byte, field.length)
	_, err := s.stream.ReadAt(buf, field.offset)
	if err != nil {
		return err
	}

	elemSize := tp.Elem().Size()
	val := reflect.ValueOf(arrPtr).Elem()
	for len(buf) > 0 {
		newElem := reflect.New(val.Type())
		err = bytesToScalar(buf[:elemSize], newElem.Interface())
		if err != nil {
			arrPtr = nil
			return err
		}
		val = reflect.Append(val, reflect.ValueOf(newElem))

		buf = buf[elemSize:]
	}

	return nil
}

func (s *storage) WriteArray(fieldName string, arr interface{}) error {
	tp := reflect.TypeOf(arr)
	if !(tp.Kind() == reflect.Slice || tp.Kind() == reflect.Array) {
		return ErrBadType
	}

	field, ok := s.arrayFields[fieldName]
	if !ok {
		return ErrFieldNotFound
	}

	val := reflect.ValueOf(arr)
	buf := make([]byte, 0, int(tp.Elem().Size())*val.Len())

	var res []byte
	var err error
	for i := 0; i < val.Len(); i++ {
		res, err = scalarToBytes(val.Index(i))
		if err != nil {
			return err
		}
		buf = append(buf, res...)
	}

	_, err = s.stream.WriteAt(buf, field.offset)
	return err
}

func (s *storage) ReadFromMap(fieldName string, key interface{}) (interface{}, error) {
	mapField, ok := s.mapFields[fieldName]
	if !ok {
		return nil, ErrFieldNotFound
	}

	if mapField == nil {
		return nil, ErrMapIsNil
	}

	return mapField.Get(key)
}

func (s *storage) WriteToMap(fieldName string, key, value interface{}) error {
	mapField, ok := s.mapFields[fieldName]
	if !ok {
		return ErrFieldNotFound
	}

	if mapField == nil {
		return ErrMapIsNil
	}

	return mapField.Put(key, value)
}

func scalarToBytes(v interface{}) ([]byte, error) {
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

func bytesToScalar(buf []byte, vr interface{}) error {
	switch vr.(type) {
	case *uint8:
		*vr.(*uint8) = vr.(uint8)
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
		*vr.(*string) = vr.(string)
	case []byte:
		vr = buf
	default:
		return errors.New(fmt.Sprintf("%s: %s", ErrBadType, vr))
	}

	return nil
}
