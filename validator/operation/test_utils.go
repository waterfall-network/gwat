package operation

import (
	"bytes"
	"fmt"
	"testing"

	"gitlab.waterfall.network/waterfall/protocol/gwat/tests/testutils"
)

type operationTestCase struct {
	caseName string
	decoded  interface{}
	encoded  []byte
	errs     []error
}

func checkOpCode(b []byte, op Operation) error {
	haveOpCode, err := GetOpCode(b)
	if err != nil {
		return err
	}
	if haveOpCode != op.OpCode() {
		return fmt.Errorf("values do not match:\nwant: %+v\nhave: %+v", op.OpCode(), haveOpCode)
	}
	return nil
}

func equalOpBytes(op Operation, b []byte) error {
	have, err := EncodeToBytes(op)
	if err != nil {
		return fmt.Errorf("can`t encode operation %+v\nerror: %+v", op, err)
	}

	if !bytes.Equal(b, have) {
		return fmt.Errorf("values do not match:\n want: %#x\nhave: %#x", b, have)
	}

	return nil
}

func startSubTests(t *testing.T, cases []operationTestCase, operationEncode, operationDecode func([]byte, interface{}) error) {
	t.Helper()

	for _, c := range cases {
		var err error
		t.Run("encoding"+" "+c.caseName, func(t *testing.T) {
			err = operationEncode(c.encoded, c.decoded)
			if !testutils.CheckError(err, c.errs) {
				t.Errorf("operationEncode: invalid test case %s\nwant errors: %s\nhave errors: %s", c.caseName, c.errs, err)
			}
		})
		if err != nil {
			continue
		}
		t.Run("decoding"+" "+c.caseName, func(t *testing.T) {
			err = operationDecode(c.encoded, c.decoded)
			if !testutils.CheckError(err, c.errs) {
				t.Errorf("operationDecode: invalid test case %s\nwant errors: %s\nhave errors: %s", c.caseName, c.errs, err)
			}
		})
	}
}
