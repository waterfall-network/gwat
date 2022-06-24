package storage

import (
	"bytes"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"testing"
)

var (
	stateDb, _ = state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
	opAddress  = common.HexToAddress("d049bfd667cb46aa3ef5df0da3e57db3be39e511")
	b          = []byte{
		243, 39, 248, 136, 130, 2, 209, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 130, 48, 57, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128,
	}
	buf = make([]byte, 139, 200)
)

func TestWriteStream(t *testing.T) {
	stream := NewStorageStream(opAddress, stateDb)
	_, err := stream.WriteAt(b, 25)
	if err != nil {
		t.Fatal(err)
	}

	_, err = stream.ReadAt(buf, 25)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(b, buf) {
		t.Fatal()
	}
}
