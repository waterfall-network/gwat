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
	name    []byte
	symbol  []byte
	baseUri []byte
	db      vm.StateDB
	addr    common.Address
)

type testCase struct {
	fields  []field
	totalL  int
	std     operation.Std
	version uint16
}

func init() {
	db, _ = state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
	addr = common.BytesToAddress(testutils.RandomData(20))
	name = testutils.RandomStringInBytes(testutils.RandomInt(0, 255))
	symbol = testutils.RandomStringInBytes(testutils.RandomInt(0, 10))
	baseUri = testutils.RandomStringInBytes(testutils.RandomInt(0, 255))
}

func TestWRC20Signature(t *testing.T) {
	cases := testCase{
		fields: []field{
			{
				8, len(name),
			},
			{
				8 + len(name), len(symbol),
			},
			{
				8 + len(name) + len(symbol), decimalsSize,
			},
			{
				8 + len(name) + len(symbol) + decimalsSize, totalSupplySize,
			},
		},
		totalL:  8 + len(name) + len(symbol) + decimalsSize + totalSupplySize,
		std:     operation.StdWRC20,
		version: 1,
	}

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

	if readingSign.Version() != cases.version {
		t.Fatal()
	}

	offLength, fieldLength := readingSign.Name()
	compareValues(t, offLength, cases.fields[nameField].offset)
	compareValues(t, fieldLength, cases.fields[nameField].length)

	offLength, fieldLength = readingSign.Symbol()
	compareValues(t, offLength, cases.fields[symbolField].offset)
	compareValues(t, fieldLength, cases.fields[symbolField].length)

	offLength, fieldLength = readingSign.Decimals()
	compareValues(t, offLength, cases.fields[decimalsField].offset)
	compareValues(t, fieldLength, cases.fields[decimalsField].length)

	offLength, fieldLength = readingSign.TotalSupply()
	compareValues(t, offLength, cases.fields[totalSupplyField].offset)
	compareValues(t, fieldLength, cases.fields[totalSupplyField].length)
}

func TestWRC721Signature(t *testing.T) {
	cases := testCase{
		fields: []field{
			{
				7, len(name),
			},
			{
				7 + len(name), len(symbol),
			},
			{
				7 + len(name) + len(symbol), len(baseUri),
			},
		},
		totalL:  7 + len(name) + len(symbol) + len(baseUri),
		std:     operation.StdWRC721,
		version: 1,
	}

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

	if readingSign.Version() != cases.version {
		t.Fatal()
	}

	offLength, fieldLength := readingSign.Name()
	compareValues(t, offLength, cases.fields[nameField].offset)
	compareValues(t, fieldLength, cases.fields[nameField].length)

	offLength, fieldLength = readingSign.Symbol()
	compareValues(t, offLength, cases.fields[symbolField].offset)
	compareValues(t, fieldLength, cases.fields[symbolField].length)

	offLength, fieldLength = readingSign.BaseUri()
	compareValues(t, offLength, cases.fields[baseUriField].offset)
	compareValues(t, fieldLength, cases.fields[baseUriField].length)
}

func compareValues(t *testing.T, a, b int) {
	if a != b {
		t.Fatal()
	}
}
