package token

import (
	"bytes"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"math/big"
	"math/rand"
	"testing"
	"time"
)

type testCase struct {
	caseName string
	testData interface{}
	errs     []error
	fn       func(c *testCase, a *common.Address)
}

type testData struct {
	caller       vm.AccountRef
	tokenAddress common.Address
}

func checkError(e error, arr []error) bool {
	for _, err := range arr {
		if e == err {
			return true
		}
	}

	return false
}

func compareBigInt(t *testing.T, a, b *big.Int) {
	haveValue := a
	wantValue := b

	if wantValue == nil {
		wantValue = big.NewInt(0)
	}

	if haveValue.Cmp(wantValue) != 0 {
		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", wantValue, haveValue)
	}
}

func compareBytes(t *testing.T, a, b []byte) {
	if !bytes.Equal(b, a) {
		t.Fatalf("values do not match:\n want: %+v\nhave: %+v", b, a)
	}
}

func randomInt(min, max int) int {
	rand.Seed(time.Now().UTC().UnixNano())
	a := rand.Intn(max-min+1) + min

	return a
}

func randomStringInBytes(l int) []byte {
	letters := []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")

	b := make([]byte, l)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}

	return b
}

func randomData(n int) []byte {
	b := make([]byte, n)
	rand.Read(b)

	return b
}
