package storage

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/internal/token/testutils"
	"github.com/ethereum/go-ethereum/token/operation"
	"math/big"
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
	total   int
}

type fieldParams struct {
	offset *big.Int
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
			offset: big.NewInt(10),
			length: len(name),
		},
			fieldParams{
				offset: big.NewInt(int64(10 + len(name))),
				length: len(symbol),
			},
			fieldParams{},
			fieldParams{
				offset: big.NewInt(int64(10 + len(name) + len(symbol))),
				length: decimalsSize,
			},
			fieldParams{
				offset: big.NewInt(int64(10 + len(name) + len(symbol) + decimalsSize)),
				length: totalSupplySize,
			},
		},
		std:     operation.StdWRC20,
		version: 1,
		total:   10 + len(name) + len(symbol) + decimalsSize + totalSupplySize + mapSize + mapSize,
	}

	wrc721InputData = testCase{
		fields: struct {
			name        fieldParams
			symbol      fieldParams
			baseUri     fieldParams
			decimals    fieldParams
			totalSupply fieldParams
		}{fieldParams{
			offset: big.NewInt(11),
			length: len(name),
		},
			fieldParams{
				offset: big.NewInt(int64(11 + len(name))),
				length: len(symbol),
			},
			fieldParams{
				offset: big.NewInt(int64(11 + len(name) + len(symbol))),
				length: len(baseUri),
			},
			fieldParams{},
			fieldParams{},
		},
		std:     operation.StdWRC721,
		version: 1,
		total:   11 + len(name) + len(symbol) + len(baseUri) + mapSize + mapSize + mapSize + mapSize,
	}
}

func TestWRC20CreateSignature(t *testing.T) {
	sign := newWrc20SignatureV1(len(name), len(symbol))

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

	compareWrc20Values(t, readingSign, wrc20InputData)
}

func TestWRC721CreateSignature(t *testing.T) {
	sign := newWrc721SignatureV1(len(name), len(symbol), len(baseUri))

	compareWrc721Values(t, sign, wrc721InputData)
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

	compareWrc721Values(t, readingSign, wrc721InputData)
}

func compareValues(t *testing.T, a, b int) {
	if a != b {
		t.Fatal()
	}
}

func compareWrc20Values(t *testing.T, sign wrc20Signature, inputData testCase) {
	if sign.Version() != inputData.version {
		t.Fatal()
	}

	compareValues(t, sign.TotalLength(), inputData.total)

	fieldName := sign.Name()
	testutils.CompareBigInt(t, fieldName.offset, inputData.fields.name.offset)
	compareValues(t, fieldName.length, inputData.fields.name.length)

	fieldSymbol := sign.Symbol()
	testutils.CompareBigInt(t, fieldSymbol.offset, inputData.fields.symbol.offset)
	compareValues(t, fieldSymbol.length, inputData.fields.symbol.length)

	fieldDecimals := sign.Decimals()
	testutils.CompareBigInt(t, fieldDecimals.offset, inputData.fields.decimals.offset)
	compareValues(t, fieldDecimals.length, inputData.fields.decimals.length)

	fieldTotalSupply := sign.TotalSupply()
	testutils.CompareBigInt(t, fieldTotalSupply.offset, inputData.fields.totalSupply.offset)
	compareValues(t, fieldTotalSupply.length, inputData.fields.totalSupply.length)
}

func compareWrc721Values(t *testing.T, sign wrc721Signature, inputData testCase) {
	if sign.Version() != inputData.version {
		t.Fatal()
	}

	compareValues(t, sign.TotalLength(), inputData.total)

	fieldName := sign.Name()
	testutils.CompareBigInt(t, fieldName.offset, inputData.fields.name.offset)
	compareValues(t, fieldName.length, inputData.fields.name.length)

	fieldSymbol := sign.Symbol()
	testutils.CompareBigInt(t, fieldSymbol.offset, inputData.fields.symbol.offset)
	compareValues(t, fieldSymbol.length, inputData.fields.symbol.length)

	fieldBaseUri := sign.BaseUri()
	testutils.CompareBigInt(t, fieldBaseUri.offset, inputData.fields.baseUri.offset)
	compareValues(t, fieldBaseUri.length, inputData.fields.baseUri.length)
}
