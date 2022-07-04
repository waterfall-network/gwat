package storage

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/internal/token/testutils"
	"github.com/ethereum/go-ethereum/token/operation"
	"testing"
)

var (
	name            []byte
	symbol          []byte
	baseUri         []byte
	db              vm.StateDB
	addr            common.Address
	wrc20InputData  testCase
	wrc721InputData testCase
)

type testCase struct {
	fields struct {
		name        fieldParams
		symbol      fieldParams
		baseUri     fieldParams
		decimals    fieldParams
		totalSupply fieldParams
	}
	std     operation.Std
	version uint16
}

type fieldParams struct {
	offset int
	length int
}

func init() {
	db, _ = state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
	addr = common.BytesToAddress(testutils.RandomData(20))
	name = testutils.RandomStringInBytes(testutils.RandomInt(0, 255))
	symbol = testutils.RandomStringInBytes(testutils.RandomInt(0, 10))
	baseUri = testutils.RandomStringInBytes(testutils.RandomInt(0, 255))

	wrc20InputData = testCase{
		fields: struct {
			name        fieldParams
			symbol      fieldParams
			baseUri     fieldParams
			decimals    fieldParams
			totalSupply fieldParams
		}{fieldParams{
			offset: 8,
			length: len(name),
		},
			fieldParams{
				offset: 8 + len(name),
				length: len(symbol),
			},
			fieldParams{},
			fieldParams{
				offset: 8 + len(name) + len(symbol),
				length: decimalsSize,
			},
			fieldParams{
				offset: 8 + len(name) + len(symbol) + decimalsSize,
				length: totalSupplySize,
			},
		},
		std:     operation.StdWRC20,
		version: 1,
	}

	wrc721InputData = testCase{
		fields: struct {
			name        fieldParams
			symbol      fieldParams
			baseUri     fieldParams
			decimals    fieldParams
			totalSupply fieldParams
		}{fieldParams{
			offset: 7,
			length: len(name),
		},
			fieldParams{
				offset: 7 + len(name),
				length: len(symbol),
			},
			fieldParams{
				offset: 7 + len(name) + len(symbol),
				length: len(baseUri),
			},
			fieldParams{},
			fieldParams{},
		},
		std:     operation.StdWRC721,
		version: 1,
	}
}

func TestWRC20CreateSignature(t *testing.T) {
	sign := newWrc20SignatureV1(len(name), len(symbol))

	if sign.Version() != wrc20InputData.version {
		t.Fatal()
	}

	compareWrc20Values(t, sign, wrc20InputData)
}

func TestWRC20ReadWriteSignature(t *testing.T) {
	sign := newWrc20SignatureV1(len(name), len(symbol))

	stream := NewStorageStream(addr, db)
	err := sign.WriteToStream(stream)
	if err != nil {
		t.Fatal()
	}

	readingSign, err := readWrc20Signature(stream)
	if err != nil {
		t.Fatal(err)
	}

	if readingSign.Version() != wrc20InputData.version {
		t.Fatal()
	}

	compareWrc20Values(t, readingSign, wrc20InputData)
}

func TestWRC721CreateSignature(t *testing.T) {
	sign := newWrc721SignatureV1(len(name), len(symbol), len(baseUri))

	if sign.Version() != wrc721InputData.version {
		t.Fatal()
	}

	offLength, fieldLength := sign.Name()
	compareValues(t, offLength, wrc721InputData.fields.name.offset)
	compareValues(t, fieldLength, wrc721InputData.fields.name.length)

	offLength, fieldLength = sign.Symbol()
	compareValues(t, offLength, wrc721InputData.fields.symbol.offset)
	compareValues(t, fieldLength, wrc721InputData.fields.symbol.length)

	offLength, fieldLength = sign.BaseUri()
	compareValues(t, offLength, wrc721InputData.fields.baseUri.offset)
	compareValues(t, fieldLength, wrc721InputData.fields.baseUri.length)
}

func TestWRC721ReadWriteSignature(t *testing.T) {
	sign := newWrc721SignatureV1(len(name), len(symbol), len(baseUri))

	stream := NewStorageStream(addr, db)
	err := sign.WriteToStream(stream)
	if err != nil {
		t.Fatal()
	}

	readingSign, err := readWrc721Signature(stream)
	if err != nil {
		t.Fatal(err)
	}

	if readingSign.Version() != wrc721InputData.version {
		t.Fatal()
	}

	compareWrc721Values(t, readingSign, wrc721InputData)
}

func compareValues(t *testing.T, a, b int) {
	if a != b {
		t.Fatal()
	}
}

func compareWrc20Values(t *testing.T, sign wrc20Signature, inputData testCase) {
	offLength, fieldLength := sign.Name()
	compareValues(t, offLength, inputData.fields.name.offset)
	compareValues(t, fieldLength, inputData.fields.name.length)

	offLength, fieldLength = sign.Symbol()
	compareValues(t, offLength, inputData.fields.symbol.offset)
	compareValues(t, fieldLength, inputData.fields.symbol.length)

	offLength, fieldLength = sign.Decimals()
	compareValues(t, offLength, inputData.fields.decimals.offset)
	compareValues(t, fieldLength, inputData.fields.decimals.length)

	offLength, fieldLength = sign.TotalSupply()
	compareValues(t, offLength, inputData.fields.totalSupply.offset)
	compareValues(t, fieldLength, inputData.fields.totalSupply.length)
}

func compareWrc721Values(t *testing.T, sign wrc721Signature, inputData testCase) {
	offLength, fieldLength := sign.Name()
	compareValues(t, offLength, inputData.fields.name.offset)
	compareValues(t, fieldLength, inputData.fields.name.length)

	offLength, fieldLength = sign.Symbol()
	compareValues(t, offLength, inputData.fields.symbol.offset)
	compareValues(t, fieldLength, inputData.fields.symbol.length)

	offLength, fieldLength = sign.BaseUri()
	compareValues(t, offLength, inputData.fields.baseUri.offset)
	compareValues(t, fieldLength, inputData.fields.baseUri.length)
}
