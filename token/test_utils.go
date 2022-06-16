package token

import (
	"bytes"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"math/big"
	"math/rand"
	"testing"
	"time"
)

type testCase struct {
	caseName string
	decoded  interface{}
	encoded  []byte
	errs     []error
}

type test struct {
	caseName string
	testData interface{}
	errs     []error
	fn       func(c *test, a *common.Address)
}

type testData struct {
	caller       vm.AccountRef
	tokenAddress common.Address
}

func startSubTests(t *testing.T, cases []testCase, operationEncode, operationDecode func([]byte, interface{}) error) {
	for _, c := range cases {
		t.Run("encoding"+" "+c.caseName, func(t *testing.T) {
			err := operationEncode(c.encoded, c.decoded)
			if err != nil {
				if !checkError(err, c.errs) {
					t.Fatalf("operationEncode: invalid test case %s\nwant errors: %s\nhave errors: %s", c.caseName, c.errs, err)
				}
			}
		})

		t.Run("decoding"+" "+c.caseName, func(t *testing.T) {
			err := operationDecode(c.encoded, c.decoded)
			if err != nil {
				if !checkError(err, c.errs) {
					t.Fatalf("operationDecode: invalid test case %s\nwant errors: %s\nhave errors: %s", c.caseName, c.errs, err)
				}
			}
		})
	}
}

func checkBalance(t *testing.T, p *Processor, tokenAddress, owner common.Address) *big.Int {
	balanceOp, err := NewBalanceOfOperation(tokenAddress, owner)
	if err != nil {
		t.Fatal(err)
	}

	balance, err := p.BalanceOf(balanceOp)
	if err != nil {
		t.Fatal(err)
	}

	return balance
}

func mintNewToken(t *testing.T, p *Processor, owner, tokenAddress common.Address, id *big.Int, data []byte, caller Ref, errs []error) {
	mintOp, err := NewMintOperation(owner, id, data)
	if err != nil {
		t.Fatal(err)
	}

	call(t, p, caller, tokenAddress, mintOp, errs)
}

func call(t *testing.T, p *Processor, caller Ref, tokenAddress common.Address, op Operation, errs []error) []byte {
	res, err := p.Call(caller, tokenAddress, op)
	if !checkError(err, errs) {
		t.Fatalf("Case failed\nwant errors: %s\nhave errors: %s", errs, err)
	}

	return res
}

func checkOpCodeAndStandart(b []byte, op Operation, std Std) error {
	if op.Standard() != std {
		return fmt.Errorf("values do not match:\nwant: %+v\nhave: %+v", std, op.Standard())
	}

	haveOpCode, err := GetOpCode(b)
	if err != nil {
		return err
	}

	if haveOpCode != op.OpCode() {
		return fmt.Errorf("values do not match:\nwant: %+v\nhave: %+v", op.OpCode(), haveOpCode)
	}

	return nil
}

func equalOpBytes(t *testing.T, op Operation, b []byte) {
	have, err := EncodeToBytes(op)
	if err != nil {
		t.Fatalf("can`t encode operation %+v\nerror: %+v", op, err)
	}

	if !bytes.Equal(b, have) {
		t.Fatalf("values do not match:\n want: %+v\nhave: %+v", b, have)
	}
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

func callTransferFrom(
	t *testing.T,
	p *Processor,
	std Std,
	owner, to, tokenAddress common.Address,
	id *big.Int,
	caller Ref,
	errs []error,
) {
	transferOp, err := NewTransferFromOperation(std, owner, to, id)
	if err != nil {
		t.Fatal(err)
	}

	call(t, p, caller, tokenAddress, transferOp, errs)
}

func checkApprove(t *testing.T, p *Processor, tokenAddress, owner, operator common.Address) bool {
	op, err := NewIsApprovedForAllOperation(tokenAddress, owner, operator)
	if err != nil {
		t.Fatal(err)
	}

	ok, err := p.IsApprovedForAll(op)
	if err != nil {
		t.Fatal(err)
	}

	return ok
}

func callApprove(t *testing.T, p *Processor, std Std, spender, tokenAddress common.Address, caller Ref, value *big.Int, errs []error) {
	approveOp, err := NewApproveOperation(std, spender, value)
	if err != nil {
		t.Fatal(err)
	}

	call(t, p, caller, tokenAddress, approveOp, errs)
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
