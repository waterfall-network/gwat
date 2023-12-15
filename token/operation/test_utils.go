package operation

import (
	"bytes"
	"fmt"
	"math/big"
	"testing"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/tests/testutils"
)

var (
	opOperator    = common.HexToAddress("13e4acefe6a6700604929946e70e6443e4e73447")
	opAddress     = common.HexToAddress("d049bfd667cb46aa3ef5df0da3e57db3be39e511")
	opSpender     = common.HexToAddress("2cccf5e0538493c235d1c5ef6580f77d99e91396")
	opOwner       = common.HexToAddress("1977c248e1014cc103929dd7f154199c916e39ec")
	opTo          = common.HexToAddress("7dc9c9730689ff0b0fd506c67db815f12d90a448")
	opFrom        = common.HexToAddress("7986bad81f4cbd9317f5a46861437dae58d69113")
	opValue       = big.NewInt(112233)
	opId          = big.NewInt(12345)
	opTotalSupply = big.NewInt(9999)
	opIndex       = big.NewInt(55555)
	opDecimals    = uint8(100)
	opName        = []byte("Test Tokken")
	opSymbol      = []byte("TT")
	opBaseURI     = []byte("test.token.com")
	oData         = []byte{243, 12, 202, 20, 133, 116, 111, 107, 101, 110, 116, 100, 5}
	opPercentFee  = uint8(10)
)

type operationTestCase struct {
	caseName string
	decoded  interface{}
	encoded  []byte
	errs     []error
}

func checkOpCodeAndStandard(b []byte, op Operation, std Std) error {
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
		t.Run("encoding"+" "+c.caseName, func(t *testing.T) {
			err := operationEncode(c.encoded, c.decoded)
			if !testutils.CheckError(err, c.errs) {
				t.Errorf("operationEncode: invalid test case %s\nwant errors: %s\nhave errors: %s", c.caseName, c.errs, err)
			}
		})

		t.Run("decoding"+" "+c.caseName, func(t *testing.T) {
			err := operationDecode(c.encoded, c.decoded)
			if !testutils.CheckError(err, c.errs) {
				t.Errorf("operationDecode: invalid test case %s\nwant errors: %s\nhave errors: %s", c.caseName, c.errs, err)
			}
		})
	}
}
