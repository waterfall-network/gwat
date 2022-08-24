package storage

import (
	"math/big"
	"testing"

	"github.com/waterfall-foundation/gwat/common"
	"github.com/waterfall-foundation/gwat/core/rawdb"
	"github.com/waterfall-foundation/gwat/core/state"
	"github.com/waterfall-foundation/gwat/internal/token/testutils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewValueProperties(t *testing.T) {
	expectedLength := uint64(10)
	tests := []struct {
		name         string
		newVp        func() ValueProperties
		expectedType Type

		length *uint64

		expectValueProperties bool
		expectKeyProperties   bool
	}{
		{
			"NewScalar",
			func() ValueProperties {
				return newScalarPanic(t, Uint8Type)
			},
			Uint8Type,
			nil,
			false,
			false,
		},
		{
			"NewArray",
			func() ValueProperties {
				return NewArrayProperties(newScalarPanic(t, Uint16Type), 10)
			},
			ArrayType,
			&expectedLength,
			true,
			false,
		},
		{
			"NewMap",
			func() ValueProperties {
				m, err := NewMapProperties(newScalarPanic(t, Uint32Type), newScalarPanic(t, Uint64Type))
				require.NoError(t, err, err)

				return m
			},
			MapType,
			nil,
			true,
			true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			vp := test.newVp()
			assert.Equal(t, test.expectedType, vp.Type())

			l, err := vp.Length()
			if test.length != nil {
				assert.EqualValues(t, *test.length, l)
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

func TestSignatureV1(t *testing.T) {
	db, _ := state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
	addr := common.BytesToAddress(testutils.RandomData(20))

	stream := NewStorageStream(addr, db)

	tests := []struct {
		name         string
		descriptors  []FieldDescriptor
		expectedSign Signature
	}{
		{
			"ScalarField",
			[]FieldDescriptor{
				newPanicFieldDescriptor(t, []byte("scalar"), newScalarPanic(t, Uint8Type)),
			},
			&SignatureV1{
				version: signatureV1,
				fields: []FieldInfo{
					{
						offset: big.NewInt(12),
						length: uint64(Uint8Type.Size()),
					},
				},
			},
		},
		{
			"ArrayField",
			[]FieldDescriptor{
				newPanicFieldDescriptor(t, []byte("array"), NewArrayProperties(newScalarPanic(t, Uint16Type), 10)),
			},
			&SignatureV1{
				version: signatureV1,
				fields: []FieldInfo{
					{
						offset: big.NewInt(20),
						length: uint64(Uint16Type.Size()) * 10,
					},
				},
			},
		},
		{
			"MapScalarScalar",
			[]FieldDescriptor{
				newPanicFieldDescriptor(
					t,
					[]byte("MapScalarScalar"),
					newMapPropertiesPanic(t, newScalarPanic(t, Uint16Type), newScalarPanic(t, Uint16Type)),
				),
			},
			&SignatureV1{
				version: signatureV1,
				fields: []FieldInfo{
					{
						offset: big.NewInt(24),
						length: 0,
					},
				},
			},
		},
		{
			"MapScalarArray",
			[]FieldDescriptor{
				newPanicFieldDescriptor(
					t,
					[]byte("MapScalarArray"),
					newMapPropertiesPanic(
						t,
						newScalarPanic(t, Uint16Type),
						NewArrayProperties(newScalarPanic(t, Uint16Type), 10),
					),
				),
			},
			&SignatureV1{
				version: signatureV1,
				fields: []FieldInfo{
					{
						offset: big.NewInt(32),
						length: 0,
					},
				},
			},
		},
		{
			"MapArrayScalarField",
			[]FieldDescriptor{
				newPanicFieldDescriptor(
					t,
					[]byte("MapArrayScalar"),
					newMapPropertiesPanic(
						t,
						NewArrayProperties(newScalarPanic(t, Uint16Type), 10),
						newScalarPanic(t, Uint16Type),
					),
				),
			},
			&SignatureV1{
				version: signatureV1,
				fields: []FieldInfo{
					{
						offset: big.NewInt(32),
						length: 0,
					},
				},
			},
		},
		{
			"MapArrayArrayField",
			[]FieldDescriptor{
				newPanicFieldDescriptor(
					t,
					[]byte("MapArrayArray"),
					newMapPropertiesPanic(
						t,
						NewArrayProperties(newScalarPanic(t, Uint16Type), 10),
						NewArrayProperties(newScalarPanic(t, Uint16Type), 10),
					),
				),
			},
			&SignatureV1{
				version: signatureV1,
				fields: []FieldInfo{
					{
						offset: big.NewInt(40),
						length: 0,
					},
				},
			},
		},
		{
			"AllFields",
			[]FieldDescriptor{
				newPanicFieldDescriptor(t, []byte("scalar"), newScalarPanic(t, Uint8Type)),
				newPanicFieldDescriptor(t, []byte("array"), NewArrayProperties(newScalarPanic(t, Uint16Type), 10)),
				newPanicFieldDescriptor(
					t,
					[]byte("MapScalarScalar"),
					newMapPropertiesPanic(t, newScalarPanic(t, Uint16Type), newScalarPanic(t, Uint16Type)),
				),
				newPanicFieldDescriptor(
					t,
					[]byte("MapScalarArray"),
					newMapPropertiesPanic(t, newScalarPanic(t, Uint16Type), NewArrayProperties(newScalarPanic(t, Uint16Type), 10)),
				),
				newPanicFieldDescriptor(
					t,
					[]byte("MapArrayScalar"),
					newMapPropertiesPanic(t, NewArrayProperties(newScalarPanic(t, Uint16Type), 10), newScalarPanic(t, Uint16Type)),
				),
				newPanicFieldDescriptor(
					t,
					[]byte("MapArrayArray"),
					newMapPropertiesPanic(t,
						NewArrayProperties(newScalarPanic(t, Uint16Type), 10),
						NewArrayProperties(newScalarPanic(t, Uint16Type), 10),
					),
				),
			},
			&SignatureV1{
				version: signatureV1,
				fields: []FieldInfo{
					{
						offset: big.NewInt(145),
						length: 1,
					},
					{
						offset: big.NewInt(145 + 1),
						length: uint64(Uint16Type.Size()) * 10,
					},
					{
						offset: big.NewInt(145 + 1 + int64(Uint16Type.Size())*10),
						length: 0,
					},
					{
						offset: big.NewInt(145 + 1 + int64(Uint16Type.Size())*10),
						length: 0,
					},
					{
						offset: big.NewInt(145 + 1 + int64(Uint16Type.Size())*10),
						length: 0,
					},
					{
						offset: big.NewInt(145 + 1 + int64(Uint16Type.Size())*10),
						length: 0,
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name+"_Create", func(t *testing.T) {
			sign, err := NewSignatureV1(test.descriptors)
			require.NoError(t, err, err)
			assert.NotNil(t, sign)
			assert.Equal(t, test.expectedSign.Version(), sign.Version())
			expectedFields := test.expectedSign.Fields()
			for i, field := range sign.Fields() {
				assert.EqualValues(t, expectedFields[i].Length(), field.Length())
				assert.EqualValues(t, expectedFields[i].Offset(), field.Offset())
			}
		})

		t.Run(test.name+"_Stream", func(t *testing.T) {
			sign, err := NewSignatureV1(test.descriptors)
			require.NoError(t, err, err)
			assert.NotNil(t, sign)

			_, err = sign.WriteToStream(stream)
			assert.NoError(t, err, err)

			newSign := new(SignatureV1)
			_, err = newSign.ReadFromStream(stream)
			assert.NoError(t, err, err)

			assert.EqualValues(t, sign.Version(), newSign.Version())
			signFields := sign.Fields()
			for i, field := range newSign.Fields() {
				assert.EqualValues(t, signFields[i].Length(), field.Length())
				assert.EqualValues(t, signFields[i].Offset(), field.Offset())
			}
		})
	}
}

func TestValueProperties_MarshalingAndUnmarshaling(t *testing.T) {
	tests := []struct {
		name          string
		expectedVp    ValueProperties
		expectedBytes []byte
	}{
		{
			"Scalar",
			newScalarPanic(t, Uint16Type),
			[]byte{Uint16Type.Id(), Uint16Type.Size()},
		},
		{
			"Array",
			NewArrayProperties(newScalarPanic(t, Uint32Type), 10),
			[]byte{
				ArrayType.Id(),                                // array type
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xa, // array size
				Uint32Type.Id(), Uint32Type.Size(), // array element type and size
			},
		},
		{
			"MapScalarScalar",
			newMapPropertiesPanic(t, newScalarPanic(t, Uint32Type), newScalarPanic(t, Uint64Type)),
			[]byte{
				MapType.Id(),
				Uint32Type.Id(), Uint32Type.Size(), // key type and size
				Uint64Type.Id(), Uint64Type.Size(), // value type and size
			},
		},
		{
			"MapScalarArray",
			newMapPropertiesPanic(t, newScalarPanic(t, Uint32Type), NewArrayProperties(newScalarPanic(t, Uint32Type), 10)),
			[]byte{
				MapType.Id(),
				Uint32Type.Id(), Uint32Type.Size(), // key type and size
				ArrayType.Id(),                                // value array type
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xa, // value array size
				Uint32Type.Id(), Uint32Type.Size(), // value array element type and size
			},
		},
		{
			"MapArrayScalar",
			newMapPropertiesPanic(t, NewArrayProperties(newScalarPanic(t, Uint32Type), 10), newScalarPanic(t, Uint32Type)),
			[]byte{
				MapType.Id(),
				ArrayType.Id(),                                // key array type
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xa, // key array size
				Uint32Type.Id(), Uint32Type.Size(), // key array element type and size
				Uint32Type.Id(), Uint32Type.Size(), // value type and size
			},
		},
		{
			"MapArrayArray",
			newMapPropertiesPanic(t,
				NewArrayProperties(newScalarPanic(t, Uint32Type), 10),
				NewArrayProperties(newScalarPanic(t, Uint64Type), 10),
			),
			[]byte{
				MapType.Id(),
				ArrayType.Id(),                                // key array type
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xa, // key array size
				Uint32Type.Id(), Uint32Type.Size(), // key array element type and size
				ArrayType.Id(),                                // value array type
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xa, // value array size
				Uint64Type.Id(), Uint64Type.Size(), // value array element type and size
			},
		},
		{
			"MapScalarSlice",
			newMapPropertiesPanic(t, newScalarPanic(t, Uint32Type), NewSliceProperties(newScalarPanic(t, Uint32Type))),
			[]byte{
				MapType.Id(),
				Uint32Type.Id(), Uint32Type.Size(), // key type and size
				SliceType.Id(),                     // value slice type
				Uint32Type.Id(), Uint32Type.Size(), // value slice element type and size
			},
		},
		{
			"MapArraySlice",
			newMapPropertiesPanic(t,
				NewArrayProperties(newScalarPanic(t, Uint32Type), 10),
				NewSliceProperties(newScalarPanic(t, Uint64Type)),
			),
			[]byte{
				MapType.Id(),
				ArrayType.Id(),                                // key array type
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xa, // key array size
				Uint32Type.Id(), Uint32Type.Size(), // key array element type and size
				SliceType.Id(),                     // value slice type
				Uint64Type.Id(), Uint64Type.Size(), // value slice element type and size
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name+"_Marshal", func(t *testing.T) {
			b, err := test.expectedVp.MarshalBinary()
			require.NoError(t, err, err)
			assert.NotNil(t, b)
			assert.EqualValues(t, test.expectedBytes, b)
		})

		t.Run(test.name+"_Unmarshal", func(t *testing.T) {
			vp := newProperties(test.expectedBytes[0])
			err := vp.UnmarshalBinary(test.expectedBytes)
			require.NoError(t, err, err)
			assert.ObjectsAreEqual(test.expectedVp, vp)
			assert.Equal(t, test.expectedVp, vp)
		})
	}
}

func newMapPropertiesPanic(t *testing.T, key, value ValueProperties) *MapProperties {
	t.Helper()

	vp, err := NewMapProperties(key, value)
	require.NoError(t, err)

	return vp
}

func newScalarPanic(t *testing.T, tp Type) *ScalarProperties {
	t.Helper()

	vp, err := NewScalarProperties(tp)
	require.NoError(t, err)

	return vp
}

func newPanicFieldDescriptor(t *testing.T, name []byte, properties ValueProperties) FieldDescriptor {
	t.Helper()

	fd, err := NewFieldDescriptor(name, properties)
	require.NoError(t, err)

	return fd
}
