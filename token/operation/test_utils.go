package operation

import (
	"bytes"
	"fmt"
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
