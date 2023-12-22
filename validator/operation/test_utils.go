package operation

import (
	"bytes"
	"fmt"
	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
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

func TestParamsDelegateStakeRules() (
	profitShare, stakeShare map[common.Address]uint8,
	exit, withdrawal []common.Address,
) {
	profitShare = map[common.Address]uint8{
		common.HexToAddress("0x1111111111111111111111111111111111111111"): 10,
		common.HexToAddress("0x2222222222222222222222222222222222222222"): 30,
		common.HexToAddress("0x3333333333333333333333333333333333333333"): 60,
	}
	stakeShare = map[common.Address]uint8{
		common.HexToAddress("0x4444444444444444444444444444444444444444"): 70,
		common.HexToAddress("0x5555555555555555555555555555555555555555"): 30,
	}
	exit = []common.Address{common.HexToAddress("0x6666666666666666666666666666666666666666")}
	withdrawal = []common.Address{common.HexToAddress("0x7777777777777777777777777777777777777777")}
	return
}
