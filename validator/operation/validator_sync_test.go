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
		version           VersionValSyncOp
		initTxHash        common.Hash
		opType            types.ValidatorSyncOp
		procEpoch         uint64
		index             uint64
		creator           common.Address
		amount            *big.Int
		withdrawalAddress *common.Address
		balance           *big.Int
	}

	var (
		initTxHash        = common.HexToHash("0303030303030303030303030303030303030303030303030303030303030303")
		procEpoch         = uint64(0xaa)
		index             = uint64(0xbb)
		creator           = common.HexToAddress("0xa7e558cc6efa1c41270ef4aa227b3dd6b4a3951e")
		amount, _         = new(big.Int).SetString("32000000000000000000000", 10)
		balance, _        = new(big.Int).SetString("64000000000000000000000", 10)
		withdrawalAddress = &common.Address{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	)

	cases := []operationTestCase{
		{
			caseName: "activation OK",
			decoded: decodedOp{
				initTxHash: initTxHash,
				opType:     types.Activate,
				procEpoch:  procEpoch,
				index:      index,
				creator:    creator,
			},
			encoded: hexutils.HexToBytes("f4" +
				"02" +
				"0000000000000000" +
				"0303030303030303030303030303030303030303030303030303030303030303" +
				"00000000000000aa" +
				"00000000000000bb" +
				"a7e558cc6efa1c41270ef4aa227b3dd6b4a3951e"),
			errs: []error{},
		},
		{
			caseName: "activation: creator_address is required",
			decoded: decodedOp{
				initTxHash: initTxHash,
				opType:     types.Activate,
				procEpoch:  procEpoch,
				index:      index,
			},
			errs: []error{ErrNoCreatorAddress},
		},

		{
			caseName: "deactivation OK",
			decoded: decodedOp{
				initTxHash: initTxHash,
				opType:     types.Deactivate,
				procEpoch:  procEpoch,
				index:      index,
				creator:    creator,
			},
			encoded: hexutils.HexToBytes("f4" +
				"04" +
				"0000000000000001" +
				"0303030303030303030303030303030303030303030303030303030303030303" +
				"00000000000000aa" +
				"00000000000000bb" +
				"a7e558cc6efa1c41270ef4aa227b3dd6b4a3951e"),
			errs: []error{},
		},

		{
			caseName: "withdrawal OK",
			decoded: decodedOp{
				initTxHash:        initTxHash,
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
				"0303030303030303030303030303030303030303030303030303030303030303" +
				"00000000000000aa" +
				"00000000000000bb" +
				"a7e558cc6efa1c41270ef4aa227b3dd6b4a3951e" +
				"ffffffffffffffffffffffffffffffffffffffff" +
				"06c6b935b8bbd4000000"),
			errs: []error{},
		},

		//Ver1
		{
			caseName: "activation Ver1 OK",
			decoded: decodedOp{
				version:    Ver1,
				initTxHash: initTxHash,
				opType:     types.Activate,
				procEpoch:  procEpoch,
				index:      index,
				creator:    creator,
			},
			encoded: hexutils.HexToBytes("f4" +
				"02" +
				"f85701b854f85280a00303030303030303030303030303030303030303030303030303030303030303" +
				"81aa81bb94a7e558cc6efa1c41270ef4aa227b3dd6b4a3951e9400000000000000000000000000000000000000008080"),
			errs: []error{},
		},
		{
			caseName: "deactivation Ver1 OK",
			decoded: decodedOp{
				version:    Ver1,
				initTxHash: initTxHash,
				opType:     types.Deactivate,
				procEpoch:  procEpoch,
				index:      index,
				creator:    creator,
			},
			encoded: hexutils.HexToBytes("f4" +
				"04" +
				"f85701b854f85201a00303030303030303030303030303030303030303030303030303030303030303" +
				"81aa81bb94a7e558cc6efa1c41270ef4aa227b3dd6b4a3951e9400000000000000000000000000000000000000008080"),
			errs: []error{},
		},
		{
			caseName: "withdrawal Ver1 OK",
			decoded: decodedOp{
				version:           1,
				initTxHash:        initTxHash,
				opType:            types.UpdateBalance,
				procEpoch:         procEpoch,
				index:             index,
				creator:           creator,
				withdrawalAddress: withdrawalAddress,
				amount:            amount,
				balance:           balance,
			},
			encoded: hexutils.HexToBytes("f4" +
				"05" +
				"f86b01b868f86602a00303030303030303030303030303030303030303030303030303030303030303" +
				"81aa81bb94a7e558cc6efa1c41270ef4aa227b3dd6b4a3951e94ffffffffffffffffffffffffffffffffffffffff" +
				"8a06c6b935b8bbd40000008a0d8d726b7177a8000000"),
			errs: []error{},
		},
	}

	operationEncode := func(b []byte, i interface{}) error {
		o := i.(decodedOp)
		createOp, err := NewValidatorSyncOperation(
			o.version,
			o.opType,
			o.initTxHash,
			o.procEpoch,
			o.index,
			o.creator,
			o.amount,
			o.withdrawalAddress,
			o.balance,
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
			if opDecoded.Balance().Cmp(o.balance) != 0 {
				return fmt.Errorf("balance do not match:\nwant: %#x\nhave: %#x", o.amount.String(), opDecoded.Amount().String())
			}
		}
		return nil
	}

	startSubTests(t, cases, operationEncode, operationDecode)
}
