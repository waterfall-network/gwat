package operation

import (
	"errors"
	"testing"

	"github.com/status-im/keycard-go/hexutils"
	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/tests/testutils"
)

func TestRequestExitData(t *testing.T) {
	var (
		pubKey         = common.HexToBlsPubKey("0x9728bc733c8fcedde0c3a33dac12da3ebbaa0eb74d813a34b600520e7976a260d85f057687e8c923d52c78715515348d")
		creatorAddress = common.HexToAddress("0xa7e558cc6efa1c41270ef4aa227b3dd6b4a3951e")

		exitAfterEpoch = uint64(999)

		depositData = "f4039728bc733c8fcedde0c3a33dac12da3ebbaa0eb74d813a34b600520e7976a260d85f057687e8c923d52c78715515348da7e558cc6efa1c41270ef4aa227b3dd6b4a3951e00000000000003e7"
	)

	type decodedOp struct {
		pubkey          common.BlsPubKey // validator public key
		creator_address common.Address   // attached creator account
		exitAfterEpoch  *uint64
	}

	cases := []operationTestCase{
		{
			caseName: "OK",
			decoded: decodedOp{
				pubkey:          pubKey,
				creator_address: creatorAddress,
				exitAfterEpoch:  &exitAfterEpoch,
			},
			encoded: hexutils.HexToBytes(depositData),
			errs:    []error{},
		},
		{
			caseName: "ErrNoPubKey",
			decoded: decodedOp{
				creator_address: creatorAddress,
				exitAfterEpoch:  &exitAfterEpoch,
			},
			encoded: hexutils.HexToBytes(""),
			errs:    []error{ErrNoPubKey},
		},
		{
			caseName: "ErrNoCreatorAddress",
			decoded: decodedOp{
				pubkey:         pubKey,
				exitAfterEpoch: &exitAfterEpoch,
			},
			encoded: hexutils.HexToBytes(""),
			errs:    []error{ErrNoCreatorAddress},
		},
	}

	operationEncode := func(b []byte, i interface{}) error {
		o := i.(decodedOp)
		createOp, err := NewExitRequestOperation(
			o.pubkey,
			o.creator_address,
			o.exitAfterEpoch,
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
		opDecoded, ok := op.(ExitRequest)
		if !ok {
			return errors.New("invalid operation type")
		}
		err = checkOpCode(b, opDecoded)
		testutils.AssertNoError(t, err)
		testutils.AssertEqual(t, opDecoded.PubKey().Bytes(), o.pubkey.Bytes())
		testutils.AssertEqual(t, opDecoded.CreatorAddress().Bytes(), o.creator_address.Bytes())
		testutils.AssertEqual(t, opDecoded.ExitAfterEpoch(), o.exitAfterEpoch)

		return nil
	}

	startSubTests(t, cases, operationEncode, operationDecode)
}
