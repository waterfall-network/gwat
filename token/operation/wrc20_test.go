package operation

import (
	"errors"
	"fmt"
	"math/big"
	"testing"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/tests/testutils"
)

func TestTransferOperation(t *testing.T) {
	type decodedOp struct {
		op    Std
		value *big.Int
		to    common.Address
	}

	cases := []operationTestCase{
		{
			caseName: "Correct test",
			decoded: decodedOp{
				op:    StdWRC20,
				value: opValue,
				to:    opTo,
			},
			encoded: []byte{
				243, 30, 248, 136, 20, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 125, 201, 201, 115, 6, 137, 255, 11, 15, 213, 6, 198, 125, 184, 21, 241, 45, 144, 164, 72, 128, 131, 1, 182, 105, 128, 128, 128,
			},
			errs: []error{nil},
		},
		{
			caseName: "No empty to",
			decoded: decodedOp{
				op:    StdWRC20,
				value: opValue,
				to:    common.Address{},
			},
			encoded: []byte{
				243, 30, 248, 136, 20, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 131, 1, 182, 105, 128, 128, 128,
			},
			errs: []error{ErrNoTo},
		},
		{
			caseName: "No empty value",
			decoded: decodedOp{
				op:    StdWRC20,
				value: nil,
				to:    opTo,
			},
			encoded: []byte{
				243, 30, 248, 133, 20, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 125, 201, 201, 115, 6, 137, 255, 11, 15, 213, 6, 198, 125, 184, 21, 241, 45, 144, 164, 72, 128, 128, 128, 128, 128,
			},
			errs: []error{ErrNoValue},
		},
	}

	operationEncode := func(b []byte, i interface{}) error {
		o := i.(decodedOp)

		op, err := NewTransferOperation(o.to, o.value)
		if err != nil {
			return err
		}

		return equalOpBytes(op, b)
	}

	operationDecode := func(b []byte, i interface{}) error {
		op, err := DecodeBytes(b)
		if err != nil {
			return err
		}

		o := i.(decodedOp)
		opDecoded, ok := op.(Transfer)
		if !ok {
			return errors.New("invalid operation type")
		}

		err = checkOpCodeAndStandard(b, opDecoded, o.op)
		if err != nil {
			return err
		}

		value := opDecoded.Value()
		if !testutils.BigIntEquals(value, o.value) {
			return fmt.Errorf("values do not match:\nwant: %+v\nhave: %+v", value, o.value)
		}

		if len(value.Bytes()) == 0 {
			// just stub for encoding tests
			return ErrNoValue
		}

		if o.to != opDecoded.To() {
			t.Fatalf("to do not match:\nwant: %+v\nhave: %+v", opDecoded.To(), o.to)
		}

		return nil
	}

	startSubTests(t, cases, operationEncode, operationDecode)
}

func TestAllowanceOperation(t *testing.T) {
	type decodedOp struct {
		op      Std
		address common.Address
		owner   common.Address
		spender common.Address
	}

	cases := []operationTestCase{
		{
			caseName: "Correct test",
			decoded: decodedOp{
				op:      StdWRC721,
				address: opAddress,
				owner:   opOwner,
				spender: opSpender,
			},
			encoded: []byte{
				243, 35, 248, 133, 20, 148, 208, 73, 191, 214, 103, 203, 70, 170, 62, 245, 223, 13, 163, 229, 125, 179, 190, 57, 229, 17, 128, 148, 25, 119, 194, 72, 225, 1, 76, 193, 3, 146, 157, 215, 241, 84, 25, 156, 145, 110, 57, 236, 148, 44, 204, 245, 224, 83, 132, 147, 194, 53, 209, 197, 239, 101, 128, 247, 125, 153, 233, 19, 150, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128, 128,
			},
			errs: []error{},
		},
		{
			caseName: "No empty owner",
			decoded: decodedOp{
				op:      StdWRC20,
				address: opAddress,
				owner:   common.Address{},
				spender: opSpender,
			},
			encoded: []byte{
				243, 35, 248, 133, 20, 148, 208, 73, 191, 214, 103, 203, 70, 170, 62, 245, 223, 13, 163, 229, 125, 179, 190, 57, 229, 17, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 44, 204, 245, 224, 83, 132, 147, 194, 53, 209, 197, 239, 101, 128, 247, 125, 153, 233, 19, 150, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128, 128,
			},
			errs: []error{ErrNoOwner},
		},
		{
			caseName: "No empty token address",
			decoded: decodedOp{
				op:      0,
				address: common.Address{},
				owner:   opOwner,
				spender: opSpender,
			},
			encoded: []byte{
				243, 35, 248, 133, 20, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 25, 119, 194, 72, 225, 1, 76, 193, 3, 146, 157, 215, 241, 84, 25, 156, 145, 110, 57, 236, 148, 44, 204, 245, 224, 83, 132, 147, 194, 53, 209, 197, 239, 101, 128, 247, 125, 153, 233, 19, 150, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128, 128,
			},
			errs: []error{ErrNoAddress},
		},
		{
			caseName: "No empty spender",
			decoded: decodedOp{
				op:      0,
				address: opAddress,
				owner:   opOwner,
				spender: common.Address{},
			},
			encoded: []byte{
				243, 35, 248, 133, 20, 148, 208, 73, 191, 214, 103, 203, 70, 170, 62, 245, 223, 13, 163, 229, 125, 179, 190, 57, 229, 17, 128, 148, 25, 119, 194, 72, 225, 1, 76, 193, 3, 146, 157, 215, 241, 84, 25, 156, 145, 110, 57, 236, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128, 128,
			},
			errs: []error{ErrNoSpender},
		},
	}

	operationEncode := func(b []byte, i interface{}) error {
		o := i.(decodedOp)

		op, err := NewAllowanceOperation(o.address, o.owner, o.spender)
		if err != nil {
			return err
		}

		return equalOpBytes(op, b)
	}

	operationDecode := func(b []byte, i interface{}) error {
		op, err := DecodeBytes(b)
		if err != nil {
			return err
		}

		o := i.(decodedOp)
		opDecoded, ok := op.(Allowance)
		if !ok {
			return errors.New("invalid operation type")
		}

		err = checkOpCodeAndStandard(b, opDecoded, StdWRC20)
		if err != nil {
			return err
		}

		operator := opDecoded.Spender()
		if operator != o.spender {
			return fmt.Errorf("values do not match:\nwant: %+v\nhave: %+v", o.spender, operator)
		}

		owner := opDecoded.Owner()
		if owner != o.owner {
			return fmt.Errorf("values do not match:\nwant: %+v\nhave: %+v", o.owner, owner)
		}

		address := opDecoded.Address()
		if address != o.address {
			return fmt.Errorf("values do not match:\nwant: %+v\nhave: %+v", o.address, address)
		}

		return nil
	}

	startSubTests(t, cases, operationEncode, operationDecode)
}
