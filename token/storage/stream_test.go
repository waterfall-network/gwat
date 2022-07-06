package storage

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/internal/token/testutils"
	"math/big"
	"testing"
)

var (
	stateDb     *state.StateDB
	address     common.Address
	buf         []byte
	off         *big.Int
	negativeOff = big.NewInt(0)
)

func init() {
	stateDb, _ = state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
	address = common.BytesToAddress(testutils.RandomData(20))
	lenBuf := testutils.RandomInt(0, 200)
	buf = testutils.RandomData(lenBuf)
	// -2 because RandomInt is inclusive [min,max].
	// It is needed for offset to be smaller than buffer`s length
	// because of test case "Test reading with offset".
	off = big.NewInt(int64(testutils.RandomInt(0, len(buf)-2)))
}

type testData struct {
	scr []byte
	dst []byte
	off *big.Int
}

func TestWriteStream(t *testing.T) {
	cases := []testutils.TestCase{
		{
			CaseName: "Test full result without offset",
			TestData: testData{
				scr: buf,
				dst: make([]byte, len(buf)),
				off: big.NewInt(0),
			},
			Errs: []error{nil},
			Fn: func(c *testutils.TestCase, a *common.Address) {
				runWithoutFlush(t, a, c)
			},
		},
		{
			CaseName: "Test full result with offset",
			TestData: testData{
				scr: buf,
				dst: make([]byte, len(buf)),
				off: off,
			},
			Errs: []error{nil},
			Fn: func(c *testutils.TestCase, a *common.Address) {
				runWithoutFlush(t, a, c)
			},
		},
		{
			CaseName: "Test reading with offset",
			TestData: testData{
				scr: buf,
				dst: make([]byte, uint64(len(buf))-off.Uint64()),
				off: off,
			},
			Errs: []error{},
			Fn: func(c *testutils.TestCase, a *common.Address) {
				v := c.TestData.(testData)
				stream := NewStorageStream(*a, stateDb)
				write(t, stream, v.scr, big.NewInt(0), c.Errs)
				read(t, stream, v.dst, off, c.Errs)

				testutils.CompareBytes(t, v.dst, v.scr[off.Uint64():])
			},
		},
		{
			CaseName: "Test with empty slice",
			TestData: testData{
				scr: []byte{},
				dst: make([]byte, 0),
				off: big.NewInt(0),
			},
			Errs: []error{nil},
			Fn: func(c *testutils.TestCase, a *common.Address) {
				runWithoutFlush(t, a, c)
			},
		},
		{
			CaseName: "Test with flush",
			TestData: testData{
				scr: buf,
				dst: make([]byte, len(buf)),
				off: off,
			},
			Errs: []error{nil},
			Fn: func(c *testutils.TestCase, a *common.Address) {
				runWithFlush(t, a, c)
			},
		},
		{
			CaseName: "Test with negative offset",
			TestData: testData{
				scr: buf,
				dst: make([]byte, len(buf)),
				off: negativeOff.Neg(off),
			},
			Errs: []error{ErrInvalidOff},
			Fn: func(c *testutils.TestCase, a *common.Address) {
				runWithFlush(t, a, c)
			},
		},
	}
	for _, c := range cases {
		t.Run(c.CaseName, func(t *testing.T) {
			c.Fn(&c, &address)
		})
	}

}

func read(t *testing.T, s *StorageStream, b []byte, off *big.Int, errs []error) {
	_, err := s.ReadAt(b, off)
	if err != nil {
		if !testutils.CheckError(err, errs) {
			t.Fatal(err)
		}
	}
}

func write(t *testing.T, s *StorageStream, b []byte, off *big.Int, errs []error) {
	_, err := s.WriteAt(b, off)
	if err != nil {
		if !testutils.CheckError(err, errs) {
			t.Fatal(err)
		}
	}
}

func runWithoutFlush(t *testing.T, a *common.Address, c *testutils.TestCase) {
	v := c.TestData.(testData)
	stream := NewStorageStream(*a, stateDb)
	write(t, stream, v.scr, v.off, c.Errs)
	read(t, stream, v.dst, v.off, c.Errs)

	if off.Cmp(big.NewInt(0)) < 0 {
		return
	}
	testutils.CompareBytes(t, v.dst, v.scr)
}

func runWithFlush(t *testing.T, a *common.Address, c *testutils.TestCase) {
	v := c.TestData.(testData)
	stream := NewStorageStream(*a, stateDb)
	write(t, stream, v.scr, v.off, c.Errs)
	stream.Flush()
	stream = NewStorageStream(*a, stateDb)
	read(t, stream, v.dst, v.off, c.Errs)

	if off.Cmp(big.NewInt(0)) < 0 {
		return
	}
	testutils.CompareBytes(t, v.dst, v.scr)
}
