package testutils

import (
	"bytes"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"math/big"
	"math/rand"
	"testing"
	"time"
)

type TestCase struct {
	CaseName string
	TestData interface{}
	Errs     []error
	Fn       func(c *TestCase, a *common.Address)
}

type TestData struct {
	Caller       vm.AccountRef
	TokenAddress common.Address
}

func CompareBytes(t *testing.T, a, b []byte) {
	if !bytes.Equal(b, a) {
		t.Fatalf("values do not match:\n want: %+v\nhave: %+v", b, a)
	}
}

func RandomInt(min, max int) int {
	rand.Seed(time.Now().UTC().UnixNano())
	a := rand.Intn(max-min+1) + min

	return a
}

func RandomStringInBytes(l int) []byte {
	letters := []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")

	b := make([]byte, l)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}

	return b
}

func RandomData(n int) []byte {
	b := make([]byte, n)
	rand.Read(b)

	return b
}

func CheckError(e error, arr []error) bool {
	if e == nil && len(arr) == 0 {
		return true
	}

	for _, err := range arr {
		if e == err {
			return true
		}
	}

	return false
}

func BigIntEquals(a, b *big.Int) bool {
	haveValue := a
	wantValue := b

	if wantValue == nil {
		wantValue = big.NewInt(0)
	}

	return haveValue.Cmp(wantValue) == 0
}
