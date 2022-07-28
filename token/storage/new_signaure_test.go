package storage

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/internal/token/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNewValueProperties(t *testing.T) {
	tests := []struct {
		name         string
		newVp        func() ValueProperties
		expectedType Type

		expectLength bool
		length       int64

		expectValueProperties bool
		expectKeyProperties   bool
	}{
		{
			"NewScalar",
			func() ValueProperties {
				return NewScalarProperties(uint8Type)
			},
			uint8Type,
			false,
			0,
			false,
			false,
		},
		{
			"NewArray",
			func() ValueProperties {
				return NewArrayProperties(NewScalarProperties(uint16Type), 10)
			},
			arrayType,
			true,
			10,
			true,
			false,
		},
		{
			"NewMap",
			func() ValueProperties {
				m, err := NewMapProperties(NewScalarProperties(uint32Type), NewScalarProperties(uint64Type))
				require.NoError(t, err, err)

				return m
			},
			mapType,
			false,
			0,
			true,
			true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			vp := test.newVp()
			assert.Equal(t, test.expectedType, vp.Type())

			l, err := vp.Length()
			if test.expectLength {
				assert.Equal(t, test.length, l)
				assert.NoError(t, err, err)
			} else {
				assert.EqualValues(t, 0, l)
				assert.Error(t, err)
			}

			v, err := vp.ValueProperties()
			if test.expectValueProperties {
				assert.NotNil(t, v)
				assert.NoError(t, err, err)
			} else {
				assert.Nil(t, v)
				assert.Error(t, err)
			}

			k, err := vp.ValueProperties()
			if test.expectValueProperties {
				assert.NotNil(t, k)
				assert.NoError(t, err, err)
			} else {
				assert.Nil(t, k)
				assert.Error(t, err)
			}
		})
	}
}

func TestSignatureV1_Stream(t *testing.T) {
	db, _ := state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
	addr := common.BytesToAddress(testutils.RandomData(20))

	stream := NewStorageStream(addr, db)

	tests := []struct {
		name  string
		newFd []FieldDescriptor
	}{
		{
			"ScalarField",
			[]FieldDescriptor{
				{
					name: []byte("scalar"),
					vp:   NewScalarProperties(uint8Type),
				},
			},
		},
		{
			"ArrayField",
			[]FieldDescriptor{
				{
					name: []byte("array"),
					vp:   NewArrayProperties(NewScalarProperties(uint16Type), 10),
				},
			},
		},
		{
			"MapScalarScalar",
			[]FieldDescriptor{
				{
					name: []byte("MapScalarScalar"),
					vp:   newMapPropertiesPanic(NewScalarProperties(uint16Type), NewScalarProperties(uint16Type)),
				},
			},
		},
		{
			"MapScalarArray",
			[]FieldDescriptor{
				{
					name: []byte("MapScalarArray"),
					vp:   newMapPropertiesPanic(NewScalarProperties(uint16Type), NewArrayProperties(NewScalarProperties(uint16Type), 10)),
				},
			},
		},
		{
			"MapArrayScalarField",
			[]FieldDescriptor{
				{
					name: []byte("MapArrayScalar"),
					vp:   newMapPropertiesPanic(NewArrayProperties(NewScalarProperties(uint16Type), 10), NewScalarProperties(uint16Type)),
				},
			},
		},
		{
			"MapArrayArrayField",
			[]FieldDescriptor{
				{
					name: []byte("MapArrayArray"),
					vp: newMapPropertiesPanic(
						NewArrayProperties(NewScalarProperties(uint16Type), 10),
						NewArrayProperties(NewScalarProperties(uint16Type), 10),
					),
				},
			},
		},
		{
			"MapAllFields",
			[]FieldDescriptor{
				{
					name: []byte("scalar"),
					vp:   NewScalarProperties(uint8Type),
				},
				{
					name: []byte("array"),
					vp:   NewArrayProperties(NewScalarProperties(uint16Type), 10),
				},
				{
					name: []byte("MapScalarScalar"),
					vp:   newMapPropertiesPanic(NewScalarProperties(uint16Type), NewScalarProperties(uint16Type)),
				},
				{
					name: []byte("MapScalarArray"),
					vp:   newMapPropertiesPanic(NewScalarProperties(uint16Type), NewArrayProperties(NewScalarProperties(uint16Type), 10)),
				},
				{
					name: []byte("MapArrayScalar"),
					vp:   newMapPropertiesPanic(NewArrayProperties(NewScalarProperties(uint16Type), 10), NewScalarProperties(uint16Type)),
				},
				{
					name: []byte("MapArrayArray"),
					vp: newMapPropertiesPanic(
						NewArrayProperties(NewScalarProperties(uint16Type), 10),
						NewArrayProperties(NewScalarProperties(uint16Type), 10),
					),
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sign, err := NewSignatureV1(test.newFd)
			require.NoError(t, err, err)
			assert.NotNil(t, sign, sign)

			_, err = sign.WriteToStream(stream)
			assert.NoError(t, err, err)

			newSign := new(SignatureV1)
			_, err = newSign.ReadFromStream(stream)
			assert.NoError(t, err, err)

			assert.EqualValues(t, sign.Version(), sign.Version())
			assert.EqualValues(t, sign.Fields(), sign.Fields())
		})
	}
}

func TestValueProperties_Marshaling(t *testing.T) {
	tests := []struct {
		name          string
		vp            ValueProperties
		expectedBytes []byte
	}{
		{
			"Scalar",
			NewScalarProperties(uint16Type),
			[]byte{uint16Type.InnerType(), uint16Type.Size()},
		},
		{
			"Array",
			NewArrayProperties(NewScalarProperties(uint32Type), 10),
			[]byte{
				arrayType.InnerType(),                         // array type
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xa, // array size
				uint32Type.InnerType(), uint32Type.Size(), // array element type and size
			},
		},
		{
			"MapScalarScalar",
			newMapPropertiesPanic(NewScalarProperties(uint32Type), NewScalarProperties(uint64Type)),
			[]byte{
				mapType.InnerType(),
				uint32Type.InnerType(), uint32Type.Size(), // key type and size
				uint64Type.InnerType(), uint64Type.Size(), // value type and size
			},
		},
		{
			"MapScalarArray",
			newMapPropertiesPanic(NewScalarProperties(uint32Type), NewArrayProperties(NewScalarProperties(uint32Type), 10)),
			[]byte{
				mapType.InnerType(),
				uint32Type.InnerType(), uint32Type.Size(), // key type and size
				arrayType.InnerType(),                         // value array type
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xa, // value array size
				uint32Type.InnerType(), uint32Type.Size(), // value array element type and size
			},
		},
		{
			"MapArrayScalar",
			newMapPropertiesPanic(NewArrayProperties(NewScalarProperties(uint32Type), 10), NewScalarProperties(uint32Type)),
			[]byte{
				mapType.InnerType(),
				arrayType.InnerType(),                         // key array type
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xa, // key array size
				uint32Type.InnerType(), uint32Type.Size(), // key array element type and size
				uint32Type.InnerType(), uint32Type.Size(), // value type and size
			},
		},
		{
			"MapArrayArray",
			newMapPropertiesPanic(
				NewArrayProperties(NewScalarProperties(uint32Type), 10),
				NewArrayProperties(NewScalarProperties(uint64Type), 10),
			),
			[]byte{
				mapType.InnerType(),
				arrayType.InnerType(),                         // key array type
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xa, // key array size
				uint32Type.InnerType(), uint32Type.Size(), // key array element type and size
				arrayType.InnerType(),                         // value array type
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xa, // value array size
				uint64Type.InnerType(), uint64Type.Size(), // value array element type and size
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			b, err := test.vp.MarshalBinary()
			require.NoError(t, err, err)
			assert.NotNil(t, b)
			assert.EqualValues(t, test.expectedBytes, b)
		})
	}
}

func TestValueProperties_Unmarshaling(t *testing.T) {
	tests := []struct {
		name       string
		bytes      []byte
		expectedVp ValueProperties
	}{
		{
			"Scalar",
			[]byte{uint16Type.InnerType(), uint16Type.Size()},
			NewScalarProperties(uint16Type),
		},
		{
			"Array",
			[]byte{
				arrayType.InnerType(),                         // array type
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xa, // array size
				uint32Type.InnerType(), uint32Type.Size(), // array element type and size
			},
			NewArrayProperties(NewScalarProperties(uint32Type), 10),
		},
		{
			"MapScalarScalar",
			[]byte{
				mapType.InnerType(),
				uint32Type.InnerType(), uint32Type.Size(), // key type and size
				uint64Type.InnerType(), uint64Type.Size(), // value type and size
			},
			newMapPropertiesPanic(NewScalarProperties(uint32Type), NewScalarProperties(uint64Type)),
		},
		{
			"MapScalarArray",
			[]byte{
				mapType.InnerType(),
				uint32Type.InnerType(), uint32Type.Size(), // key type and size
				arrayType.InnerType(),                         // value array type
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xa, // value array size
				uint32Type.InnerType(), uint32Type.Size(), // value array element type and size
			},
			newMapPropertiesPanic(NewScalarProperties(uint32Type), NewArrayProperties(NewScalarProperties(uint32Type), 10)),
		},
		{
			"MapArrayScalar",
			[]byte{
				mapType.InnerType(),
				arrayType.InnerType(),                         // key array type
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xa, // key array size
				uint32Type.InnerType(), uint32Type.Size(), // key array element type and size
				uint32Type.InnerType(), uint32Type.Size(), // value type and size
			},
			newMapPropertiesPanic(NewArrayProperties(NewScalarProperties(uint32Type), 10), NewScalarProperties(uint32Type)),
		},
		{
			"MapArrayArray",
			[]byte{
				mapType.InnerType(),
				arrayType.InnerType(),                         // key array type
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xa, // key array size
				uint32Type.InnerType(), uint32Type.Size(), // key array element type and size
				arrayType.InnerType(),                         // value array type
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xb, // value array size
				uint64Type.InnerType(), uint64Type.Size(), // value array element type and size
			},
			newMapPropertiesPanic(
				NewArrayProperties(NewScalarProperties(uint32Type), 10),
				NewArrayProperties(NewScalarProperties(uint64Type), 11),
			),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			vp := newProperties(test.bytes[0])
			err := vp.UnmarshalBinary(test.bytes)
			require.NoError(t, err, err)
			assert.ObjectsAreEqual(test.expectedVp, vp)
			assert.Equal(t, test.expectedVp, vp)
		})
	}
}

func newMapPropertiesPanic(key, value ValueProperties) ValueProperties {
	vp, err := NewMapProperties(key, value)
	if err != nil {
		panic(err)
	}

	return vp
}
