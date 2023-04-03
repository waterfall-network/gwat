package operation

import (
	"errors"
	"testing"

	"github.com/status-im/keycard-go/hexutils"
	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/tests/testutils"
)

func TestDepositData(t *testing.T) {
	var (
		pubkey             = common.HexToBlsPubKey("0x9728bc733c8fcedde0c3a33dac12da3ebbaa0eb74d813a34b600520e7976a260d85f057687e8c923d52c78715515348d")
		creator_address    = common.HexToAddress("0xa7e558cc6efa1c41270ef4aa227b3dd6b4a3951e")
		withdrawal_address = common.HexToAddress("0xa7e558cc6efa1c41270ef4aa227b3dd6b4a3951e")
		//depositAmount      = 32000000000000
		signature = common.HexToBlsSig("0xb9221f2308c1e1655a8e1977f32241384fa77efedbb3079bcc9a95930152ee87" +
			"f341134a4e59c3e312ee5c2197732ea30d9aac2993cc4aad75335009815d07a8735f96c6dde443ba3a10f5523c4d00f6b3a7b48af" +
			"5a42795183ab5aa2f1b2dd1")
		depositData = "f4019728bc733c8fcedde0c3a33dac12da3ebbaa0eb74d813a34b600520e7976a260d85f057687e8c923d52c787155153" +
			"48da7e558cc6efa1c41270ef4aa227b3dd6b4a3951ea7e558cc6efa1c41270ef4aa227b3dd6b4a3951eb9221f2308c1e1655a8e1977" +
			"f32241384fa77efedbb3079bcc9a95930152ee87f341134a4e59c3e312ee5c2197732ea30d9aac2993cc4aad75335009815d07a8735" +
			"f96c6dde443ba3a10f5523c4d00f6b3a7b48af5a42795183ab5aa2f1b2dd1"
	)

	type decodedOp struct {
		pubkey             common.BlsPubKey // validator public key
		creator_address    common.Address   // attached creator account
		withdrawal_address common.Address   // attached withdrawal credentials
		signature          common.BlsSignature
	}

	cases := []operationTestCase{
		{
			caseName: "OK",
			decoded: decodedOp{
				pubkey:             pubkey,
				creator_address:    creator_address,
				withdrawal_address: withdrawal_address,
				signature:          signature,
			},
			encoded: hexutils.HexToBytes(depositData),
			errs:    []error{},
		},
		{
			caseName: "ErrNoPubKey",
			decoded: decodedOp{
				creator_address:    creator_address,
				withdrawal_address: withdrawal_address,
				signature:          signature,
			},
			encoded: hexutils.HexToBytes(""),
			errs:    []error{ErrNoPubKey},
		},
		{
			caseName: "ErrNoCreatorAddress",
			decoded: decodedOp{
				pubkey:             pubkey,
				withdrawal_address: withdrawal_address,
				signature:          signature,
			},
			encoded: hexutils.HexToBytes(""),
			errs:    []error{ErrNoCreatorAddress},
		},
		{
			caseName: "ErrNoWithdrawalAddress",
			decoded: decodedOp{
				pubkey:          pubkey,
				creator_address: creator_address,
				signature:       signature,
			},
			encoded: hexutils.HexToBytes(""),
			errs:    []error{ErrNoWithdrawalAddress},
		},
		{
			caseName: "ErrNoSignature",
			decoded: decodedOp{
				pubkey:             pubkey,
				creator_address:    creator_address,
				withdrawal_address: withdrawal_address,
			},
			encoded: hexutils.HexToBytes(""),
			errs:    []error{ErrNoSignature},
		},
	}

	operationEncode := func(b []byte, i interface{}) error {
		o := i.(decodedOp)
		createOp, err := NewDepositOperation(
			o.pubkey,
			o.creator_address,
			o.withdrawal_address,
			o.signature,
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
		opDecoded, ok := op.(Deposit)
		if !ok {
			return errors.New("invalid operation type")
		}
		err = checkOpCode(b, opDecoded)
		testutils.AssertNoError(t, err)
		testutils.AssertEqual(t, opDecoded.PubKey().Bytes(), o.pubkey.Bytes())
		testutils.AssertEqual(t, opDecoded.CreatorAddress().Bytes(), o.creator_address.Bytes())
		testutils.AssertEqual(t, opDecoded.WithdrawalAddress().Bytes(), o.withdrawal_address.Bytes())
		testutils.AssertEqual(t, opDecoded.Signature().Bytes(), o.signature.Bytes())

		return nil
	}

	startSubTests(t, cases, operationEncode, operationDecode)
}
