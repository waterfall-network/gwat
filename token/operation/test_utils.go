package operation

import (
	"bytes"
	"fmt"
	"math/big"
	"testing"
)

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
