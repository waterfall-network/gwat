package operation

import (
	"errors"
	"math/big"
	"testing"

	"github.com/status-im/keycard-go/hexutils"
	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/tests/testutils"
)

func TestWithdrawalData(t *testing.T) {
	var (
		creatorAddress = common.HexToAddress("0xa7e558cc6efa1c41270ef4aa227b3dd6b4a3951e")
		amount         = big.NewInt(50000)

		depositData = "f406a7e558cc6efa1c41270ef4aa227b3dd6b4a3951ec350"
	)

	type decodedOp struct {
		creatorAddress common.Address // attached creator account
		amount         *big.Int
	}

	cases := []operationTestCase{
		{
			caseName: "OK",
			decoded: decodedOp{
				creatorAddress: creatorAddress,
				amount:         amount,
			},
			encoded: hexutils.HexToBytes(depositData),
			errs:    []error{},
		},
		{
			caseName: "ErrNoCreatorAddress",
			decoded: decodedOp{
				amount: amount,
			},
			encoded: hexutils.HexToBytes(""),
			errs:    []error{ErrNoCreatorAddress},
		},
		{
			caseName: "ErrNoAmount",
			decoded: decodedOp{
				creatorAddress: creatorAddress,
			},
			encoded: hexutils.HexToBytes(""),
			errs:    []error{ErrNoAmount},
		},
	}

	operationEncode := func(b []byte, i interface{}) error {
		o := i.(decodedOp)
		createOp, err := NewWithdrawalOperation(
			o.creatorAddress,
			o.amount,
		)
		if err != nil {
			return err
		}

		return equalOpBytes(createOp, b)
	}

	operationDecode := func(b []byte, i interface{}) error {
		op, err := DecodeBytes(b)
		testutils.AssertNoError(t, err)

		o := i.(decodedOp)
		opDecoded, ok := op.(WithdrawalRequest)
		if !ok {
			return errors.New("invalid operation type")
		}
		err = checkOpCode(b, opDecoded)
		testutils.AssertNoError(t, err)
		testutils.AssertEqual(t, opDecoded.CreatorAddress(), o.creatorAddress)
		if !testutils.BigIntEquals(opDecoded.Amount(), o.amount) {
			t.Fatalf("\n\tExpect:\t%v\n\tGot:\t%v", opDecoded.Amount(), o.amount)
		}

		return nil
	}

	startSubTests(t, cases, operationEncode, operationDecode)
}
