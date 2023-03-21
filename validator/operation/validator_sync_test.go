package operation

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/status-im/keycard-go/hexutils"
	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/types"
)

func TestValidatorSync(t *testing.T) {
	type decodedOp struct {
		opType            types.ValidatorSyncOp
		procEpoch         uint64
		index             uint64
		creator           common.Address
		amount            *big.Int
		withdrawalAddress *common.Address
	}

	var (
		procEpoch         = uint64(0xaa)
		index             = uint64(0xbb)
		creator           = common.HexToAddress("0xa7e558cc6efa1c41270ef4aa227b3dd6b4a3951e")
		amount, _         = new(big.Int).SetString("32000000000000000000000", 10)
		withdrawalAddress = &common.Address{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	)

	cases := []operationTestCase{
		{
			caseName: "activation OK",
			decoded: decodedOp{
				opType:    types.Activate,
				procEpoch: procEpoch,
				index:     index,
				creator:   creator,
			},
			encoded: hexutils.HexToBytes("f4" +
				"02" +
				"0000000000000000" +
				"00000000000000aa" +
				"00000000000000bb" +
				"a7e558cc6efa1c41270ef4aa227b3dd6b4a3951e"),
			errs: []error{},
		},
		{
			caseName: "activation: creator_address is required",
			decoded: decodedOp{
				opType:    types.Activate,
				procEpoch: procEpoch,
				index:     index,
			},
			errs: []error{ErrNoCreatorAddress},
		},

		{
			caseName: "exit OK",
			decoded: decodedOp{
				opType:    types.Deactivate,
				procEpoch: procEpoch,
				index:     index,
				creator:   creator,
			},
			encoded: hexutils.HexToBytes("f4" +
				"04" +
				"0000000000000001" +
				"00000000000000aa" +
				"00000000000000bb" +
				"a7e558cc6efa1c41270ef4aa227b3dd6b4a3951e"),
			errs: []error{},
		},

		{
			caseName: "withdrawal OK",
			decoded: decodedOp{
				opType:            types.UpdateBalance,
				procEpoch:         procEpoch,
				index:             index,
				creator:           creator,
				withdrawalAddress: withdrawalAddress,
				amount:            amount,
			},
			encoded: hexutils.HexToBytes("f4" +
				"05" +
				"0000000000000002" +
				"00000000000000aa" +
				"00000000000000bb" +
				"a7e558cc6efa1c41270ef4aa227b3dd6b4a3951e" +
				"ffffffffffffffffffffffffffffffffffffffff" +
				"06c6b935b8bbd4000000"),
			errs: []error{},
		},
	}

	operationEncode := func(b []byte, i interface{}) error {
		o := i.(decodedOp)
		createOp, err := NewValidatorSyncOperation(
			o.opType,
			o.procEpoch,
			o.index,
			o.creator,
			o.amount,
			o.withdrawalAddress,
		)
		if err != nil {
			return err
		}
		return equalOpBytes(createOp, b)
	}

	operationDecode := func(b []byte, i interface{}) error {
		op, err := DecodeBytes(b)
		if err != nil {
			return err
		}

		o := i.(decodedOp)
		opDecoded, ok := op.(ValidatorSync)
		if !ok {
			return errors.New("invalid operation type")
		}
		err = checkOpCode(b, opDecoded)
		if err != nil {
			return err
		}
		if opDecoded.OpType() != o.opType {
			return fmt.Errorf("opType do not match:\nwant: %#x\nhave: %#x", o.opType, opDecoded.OpType())
		}
		if opDecoded.ProcEpoch() != o.procEpoch {
			return fmt.Errorf("procEpoch do not match:\nwant: %#x\nhave: %#x", o.procEpoch, opDecoded.ProcEpoch())
		}
		if opDecoded.Index() != o.index {
			return fmt.Errorf("index do not match:\nwant: %#x\nhave: %#x", o.index, opDecoded.Index())
		}
		if !bytes.Equal(opDecoded.Creator().Bytes(), o.creator.Bytes()) {
			return fmt.Errorf("creator do not match:\nwant: %#x\nhave: %#x", o.creator, opDecoded.Creator())
		}
		if o.opType == types.UpdateBalance {
			if !bytes.Equal(opDecoded.WithdrawalAddress().Bytes(), o.withdrawalAddress.Bytes()) {
				return fmt.Errorf("withdrawalAddress do not match:\nwant: %#x\nhave: %#x", o.withdrawalAddress, opDecoded.WithdrawalAddress())
			}
			if opDecoded.Amount().Cmp(o.amount) != 0 {
				return fmt.Errorf("withdrawalAddress do not match:\nwant: %#x\nhave: %#x", o.amount.String(), opDecoded.Amount().String())
			}
		}
		return nil
	}

	startSubTests(t, cases, operationEncode, operationDecode)
}
