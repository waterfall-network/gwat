package storage

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/internal/token/testutils"

	"github.com/holiman/uint256"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateStorage(t *testing.T) {
	descriptors := []FieldDescriptor{
		{
			[]byte("scalar"),
			NewScalarProperties(Uint8Type),
		},
		{
			[]byte("array"),
			NewArrayProperties(NewScalarProperties(Uint16Type), 10),
		},
		{
			[]byte("map"),
			newMapPropertiesPanic(NewScalarProperties(Uint32Type), NewScalarProperties(Uint32Type)),
		},
	}

	t.Run("NewStorage", func(t *testing.T) {
		db, _ := state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
		addr := common.BytesToAddress(testutils.RandomData(20))
		stream := NewStorageStream(addr, db)

		expectedSign, err := NewSignatureV1(descriptors)
		require.NoError(t, err, err)

		storage, err := NewStorage(stream, descriptors)
		assert.NoError(t, err, err)
		require.NotNil(t, storage)

		readSign := new(SignatureV1)
		_, err = readSign.ReadFromStream(stream)
		assert.NoError(t, err, err)
		assert.EqualValues(t, expectedSign, readSign)
	})

	t.Run("ReadStorage", func(t *testing.T) {
		db, _ := state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
		addr := common.BytesToAddress(testutils.RandomData(20))
		stream := NewStorageStream(addr, db)

		sig, err := NewSignatureV1(descriptors)
		require.NoError(t, err, err)

		_, err = sig.WriteToStream(stream)
		require.NoError(t, err, err)

		storage, err := ReadStorage(stream)
		assert.NoError(t, err, err)
		assert.NotNil(t, storage)
	})
}

func TestStorage_WriteReadField(t *testing.T) {
	tests := []struct {
		descriptor FieldDescriptor
		expValue   interface{}
		readTo     func() interface{}
		isEqual    func(interface{}, interface{}) bool
	}{
		{
			descriptor: FieldDescriptor{
				name: []byte("Scalar_Uint8"),
				vp:   NewScalarProperties(Uint8Type),
			},
			expValue: uint8(10),
			readTo: func() interface{} {
				v := uint8(0)
				return &v
			},
			isEqual: func(v interface{}, ptr interface{}) bool {
				return v.(uint8) == *ptr.(*uint8)
			},
		},
		{
			descriptor: FieldDescriptor{
				name: []byte("Uint256"),
				vp:   NewScalarProperties(Uint256Type),
			},
			expValue: new(uint256.Int).SetBytes([]byte{
				0x01, 0x02, 0x03, 0x04,
				0x05, 0x06, 0x07, 0x08,
				0x09, 0x0a, 0x0b, 0x0c,
				0x0d, 0x0e, 0x0f, 0x00,
				0x11, 0x12, 0x13, 0x14,
				0x15, 0x16, 0x17, 0x18,
				0x19, 0x1a, 0x1b, 0x1c,
				0x1d, 0x1e, 0x1f, 0x10,
			}),
			readTo: func() interface{} {
				v := uint256.Int{}
				return &v
			},
			isEqual: func(v interface{}, ptr interface{}) bool {
				exp, ok := v.(uint256.Int)
				if !ok {
					expR, ok := v.(*uint256.Int)
					if !ok {
						return ok
					}
					exp = *expR
				}

				got, ok := ptr.(*uint256.Int)
				if !ok {
					return ok
				}

				return exp.Eq(got)
			},
		},
		{
			descriptor: FieldDescriptor{
				name: []byte("Array_Uint16"),
				vp:   NewArrayProperties(NewScalarProperties(Uint16Type), 10),
			},
			expValue: []uint16{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
			readTo: func() interface{} {
				var v []uint16
				return &v
			},
			isEqual: func(v interface{}, ptr interface{}) bool {
				exp := v.([]uint16)
				got := *ptr.(*[]uint16)
				if len(exp) != len(got) {
					return false
				}

				for i, u := range exp {
					if u != got[i] {
						return false
					}
				}

				return true
			},
		},
		{
			descriptor: FieldDescriptor{
				name: []byte("Map_Uint16_Uint16"),
				vp:   newMapPropertiesPanic(NewScalarProperties(Uint16Type), NewScalarProperties(Uint16Type)),
			},
			expValue: NewKeyValuePair(uint16(111), uint16(222)),
			readTo: func() interface{} {
				v := uint16(0)
				return NewKeyValuePair(uint16(111), &v)
			},
			isEqual: func(v interface{}, ptr interface{}) bool {
				exp, ok := v.(*KeyValuePair)
				if !ok {
					return ok
				}

				got, ok := ptr.(*KeyValuePair)
				if !ok {
					return ok
				}

				return exp.Value().(uint16) == *got.Value().(*uint16)
			},
		},
		{
			descriptor: FieldDescriptor{
				name: []byte("Map_ArrayUint16_Uint16"),
				vp:   newMapPropertiesPanic(NewArrayProperties(NewScalarProperties(Uint16Type), 3), NewScalarProperties(Uint16Type)),
			},
			expValue: NewKeyValuePair([]uint16{1, 2, 3}, uint16(222)),
			readTo: func() interface{} {
				v := uint16(0)
				return NewKeyValuePair([]uint16{1, 2, 3}, &v)
			},
			isEqual: func(v interface{}, ptr interface{}) bool {
				exp, ok := v.(*KeyValuePair)
				if !ok {
					return ok
				}

				got, ok := ptr.(*KeyValuePair)
				if !ok {
					return ok
				}

				return exp.Value().(uint16) == *got.Value().(*uint16)
			},
		},
		{
			descriptor: FieldDescriptor{
				name: []byte("Map_Uint16_ArrayUint16"),
				vp:   newMapPropertiesPanic(NewScalarProperties(Uint16Type), NewArrayProperties(NewScalarProperties(Uint16Type), 3)),
			},
			expValue: NewKeyValuePair(uint16(222), []uint16{1, 2, 3}),
			readTo: func() interface{} {
				var v []uint16
				return NewKeyValuePair(uint16(222), &v)
			},
			isEqual: func(v interface{}, ptr interface{}) bool {
				exp, ok := v.(*KeyValuePair)
				if !ok {
					return ok
				}

				got, ok := ptr.(*KeyValuePair)
				if !ok {
					return ok
				}

				expVal := exp.Value().([]uint16)
				gotVal := *got.Value().(*[]uint16)
				if len(expVal) != len(gotVal) {
					return false
				}

				for i, u := range expVal {
					if u != gotVal[i] {
						return false
					}
				}

				return true
			},
		},
		{
			descriptor: FieldDescriptor{
				name: []byte("Map_ArrayUint256_ArrayUint256"),
				vp: newMapPropertiesPanic(
					NewArrayProperties(NewScalarProperties(Uint256Type), 3),
					NewArrayProperties(NewScalarProperties(Uint256Type), 3),
				),
			},
			expValue: NewKeyValuePair(
				[]*uint256.Int{
					new(uint256.Int).SetBytes([]byte{0x01, 0x02, 0x03, 0x04}),
					new(uint256.Int).SetBytes([]byte{0x05, 0x06, 0x07, 0x08}),
					new(uint256.Int).SetBytes([]byte{0x09, 0x0a, 0x0b, 0x0c}),
				},
				[]*uint256.Int{
					new(uint256.Int).SetBytes([]byte{0x0d, 0x0e, 0x0f, 0x00}),
					new(uint256.Int).SetBytes([]byte{0x11, 0x12, 0x13, 0x14}),
					new(uint256.Int).SetBytes([]byte{0x15, 0x16, 0x17, 0x18}),
				},
			),
			readTo: func() interface{} {
				var v []*uint256.Int
				return NewKeyValuePair([]*uint256.Int{
					new(uint256.Int).SetBytes([]byte{0x01, 0x02, 0x03, 0x04}),
					new(uint256.Int).SetBytes([]byte{0x05, 0x06, 0x07, 0x08}),
					new(uint256.Int).SetBytes([]byte{0x09, 0x0a, 0x0b, 0x0c}),
				}, &v)
			},
			isEqual: func(v interface{}, ptr interface{}) bool {
				exp, ok := v.(*KeyValuePair)
				if !ok {
					return ok
				}

				got, ok := ptr.(*KeyValuePair)
				if !ok {
					return ok
				}

				expVal := exp.Value().([]*uint256.Int)
				gotVal := *got.Value().(*[]*uint256.Int)
				if len(expVal) != len(gotVal) {
					return false
				}

				for i, u := range expVal {
					if !u.Eq(gotVal[i]) {
						return false
					}
				}

				return true
			},
		},
	}

	descriptors := make([]FieldDescriptor, len(tests))
	for i, test := range tests {
		descriptors[i] = test.descriptor
	}

	storage := newStorage(t, descriptors)

	for _, test := range tests {
		name := test.descriptor.Name()
		t.Run(name, func(t *testing.T) {
			err := storage.WriteField(name, test.expValue)
			assert.NoError(t, err, err)

			to := test.readTo()
			err = storage.ReadField(name, to)
			assert.NoError(t, err, err)
			assert.True(t, test.isEqual(test.expValue, to))
			assert.ObjectsAreEqualValues(test.expValue, to)
		})
	}
}

func newStorage(t *testing.T, descriptors []FieldDescriptor) Storage {
	t.Helper()

	db, err := state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
	require.NoError(t, err, err)

	addr := common.BytesToAddress(testutils.RandomData(20))
	stream := NewStorageStream(addr, db)

	storage, err := NewStorage(stream, descriptors)
	require.NoError(t, err, err)
	require.NotNil(t, storage)

	return storage
}
