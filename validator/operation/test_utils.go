package operation

import (
	"bytes"
	"fmt"
	"math/big"
	"testing"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
)

var (
	pubkey             = common.HexToBlsPubKey("0x9728bc733c8fcedde0c3a33dac12da3ebbaa0eb74d813a34b600520e7976a260d85f057687e8c923d52c78715515348d")
	creator_address    = common.HexToAddress("0xa7e558cc6efa1c41270ef4aa227b3dd6b4a3951e")
	withdrawal_address = common.HexToAddress("0xa7e558cc6efa1c41270ef4aa227b3dd6b4a3951e")
	amount             = 32000000000000
	signature          = common.HexToBlsSig("0xb9221f2308c1e1655a8e1977f32241384fa77efedbb3079bcc9a95930152ee87" +
		"f341134a4e59c3e312ee5c2197732ea30d9aac2993cc4aad75335009815d07a8735f96c6dde443ba3a10f5523c4d00f6b3a7b48af" +
		"5a42795183ab5aa2f1b2dd1")
	deposit_data_root = common.HexToHash("0xb4cb40679413e0a38f670a4d19b21871f830f955ce41dace0926f19aad0d434b")
	depositData       = "f4019728bc733c8fcedde0c3a33dac12da3ebbaa0eb74d813a34b600520e7976a260d85f057687e8c923d52c787155153" +
		"48da7e558cc6efa1c41270ef4aa227b3dd6b4a3951ea7e558cc6efa1c41270ef4aa227b3dd6b4a3951eb9221f2308c1e1655a8e19" +
		"77f32241384fa77efedbb3079bcc9a95930152ee87f341134a4e59c3e312ee5c2197732ea30d9aac2993cc4aad75335009815d07a" +
		"8735f96c6dde443ba3a10f5523c4d00f6b3a7b48af5a42795183ab5aa2f1b2dd1b4cb40679413e0a38f670a4d19b21871f830f955" +
		"ce41dace0926f19aad0d434b"
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
		return fmt.Errorf("values do not match:\n want: %+v\nhave: %+v", b, have)
	}

	return nil
}

func startSubTests(t *testing.T, cases []operationTestCase, operationEncode, operationDecode func([]byte, interface{}) error) {
	t.Helper()

	for _, c := range cases {
		var err error
		t.Run("encoding"+" "+c.caseName, func(t *testing.T) {
			err = operationEncode(c.encoded, c.decoded)
			if !CheckError(err, c.errs) {
				t.Errorf("operationEncode: invalid test case %s\nwant errors: %s\nhave errors: %s", c.caseName, c.errs, err)
			}
		})
		if err != nil {
			continue
		}
		t.Run("decoding"+" "+c.caseName, func(t *testing.T) {
			err = operationDecode(c.encoded, c.decoded)
			if !CheckError(err, c.errs) {
				t.Errorf("operationDecode: invalid test case %s\nwant errors: %s\nhave errors: %s", c.caseName, c.errs, err)
			}
		})
	}
}

func BigIntEquals(haveValue, wantValue *big.Int) bool {
	if haveValue == nil && wantValue == nil {
		return true
	}

	if wantValue == nil {
		wantValue = big.NewInt(0)
	}

	return haveValue.Cmp(wantValue) == 0
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
