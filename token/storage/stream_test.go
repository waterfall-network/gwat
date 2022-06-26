package storage

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/internal/token/testutils"
	"testing"
)

var (
	stateDb, _ = state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
	Address    = common.HexToAddress("d049bfd667cb46aa3ef5df0da3e57db3be39e511")
	buf        = []byte{
		243, 39, 248, 136, 130, 2, 209, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 130, 48, 57, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128,
	}
)

func TestWriteStream(t *testing.T) {
	type testData struct {
		scr []byte
		dst []byte
		off int
	}

	cases := []testutils.TestCase{
		{
			CaseName: "Test full result without offset",
			TestData: testData{
				scr: buf,
				dst: make([]byte, len(buf), len(buf)),
				off: 0,
			},
			Errs: []error{nil},
			Fn: func(c *testutils.TestCase, a *common.Address) {
				v := c.TestData.(testData)
				stream := NewStorageStream(*a, stateDb)
				write(t, stream, v.scr, v.off, c.Errs)
				read(t, stream, v.dst, v.off, c.Errs)

				testutils.CompareBytes(t, v.dst, v.scr)
			},
		},
		{
			CaseName: "Test full result with offset 25",
			TestData: testData{
				scr: buf,
				dst: make([]byte, len(buf), len(buf)),
				off: 25,
			},
			Errs: []error{nil},
			Fn: func(c *testutils.TestCase, a *common.Address) {
				v := c.TestData.(testData)
				stream := NewStorageStream(*a, stateDb)
				write(t, stream, v.scr, v.off, c.Errs)
				read(t, stream, v.dst, v.off, c.Errs)

				testutils.CompareBytes(t, v.dst, v.scr)
			},
		},
		{
			CaseName: "Test reading with offset 30",
			TestData: testData{
				scr: buf,
				dst: make([]byte, 110, len(buf)),
				off: 30,
			},
			Errs: []error{},
			Fn: func(c *testutils.TestCase, a *common.Address) {
				v := c.TestData.(testData)
				stream := NewStorageStream(*a, stateDb)
				write(t, stream, v.scr, 0, c.Errs)
				read(t, stream, v.dst, v.off, c.Errs)

				testutils.CompareBytes(t, v.dst, v.scr[v.off:])
			},
		},
		{
			CaseName: "Test with empty slice",
			TestData: testData{
				scr: []byte{},
				dst: make([]byte, 0, 10),
				off: 0,
			},
			Errs: []error{nil},
			Fn: func(c *testutils.TestCase, a *common.Address) {
				v := c.TestData.(testData)
				stream := NewStorageStream(*a, stateDb)
				write(t, stream, v.scr, v.off, c.Errs)
				read(t, stream, v.dst, v.off, c.Errs)

				testutils.CompareBytes(t, v.dst, v.scr)
			},
		},
		{
			CaseName: "Test with flush",
			TestData: testData{
				scr: buf,
				dst: make([]byte, len(buf), len(buf)),
				off: 10,
			},
			Errs: []error{nil},
			Fn: func(c *testutils.TestCase, a *common.Address) {
				v := c.TestData.(testData)
				stream := NewStorageStream(*a, stateDb)
				write(t, stream, v.scr, v.off, c.Errs)
				stream.Flush()
				read(t, stream, v.dst, v.off, c.Errs)

				testutils.CompareBytes(t, v.dst, v.scr)
			},
		},
		{
			CaseName: "Test with negative offset",
			TestData: testData{
				scr: buf,
				dst: make([]byte, len(buf), len(buf)),
				off: -50,
			},
			Errs: []error{ErrInvalidOff},
			Fn: func(c *testutils.TestCase, a *common.Address) {
				v := c.TestData.(testData)
				stream := NewStorageStream(*a, stateDb)
				write(t, stream, v.scr, v.off, c.Errs)
				stream.Flush()
				read(t, stream, v.dst, v.off, c.Errs)

				testutils.CompareBytes(t, v.dst, make([]byte, len(buf), len(buf)))
			},
		},
	}

	for _, c := range cases {
		t.Run(c.CaseName, func(t *testing.T) {
			c.Fn(&c, &Address)
		})
	}

}

func read(t *testing.T, s *StorageStream, b []byte, off int, errs []error) {
	_, err := s.ReadAt(b, off)
	if err != nil {
		if !testutils.CheckError(err, errs) {
			t.Fatal(err)
		}
	}
}

func write(t *testing.T, s *StorageStream, b []byte, off int, errs []error) {
	_, err := s.WriteAt(b, off)
	if err != nil {
		if !testutils.CheckError(err, errs) {
			t.Fatal(err)
		}
	}
}
