package operation

import (
	"errors"
	"fmt"
	"github.com/waterfall-foundation/gwat/internal/token/testutils"
	"math/big"
	"testing"

	"github.com/waterfall-foundation/gwat/common"
)

func TestApproveOperation(t *testing.T) {
	type decodedOp struct {
		op      Std
		spender common.Address
		value   *big.Int
	}

	cases := []operationTestCase{
		{
			caseName: "Correct WRC721 test",
			decoded: decodedOp{
				op:      StdWRC721,
				value:   opId,
				spender: opSpender,
			},
			encoded: []byte{
				243, 13, 248, 136, 130, 2, 209, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 44, 204, 245, 224, 83, 132, 147, 194, 53, 209, 197, 239, 101, 128, 247, 125, 153, 233, 19, 150, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 130, 48, 57, 128, 128, 128,
			},
			errs: []error{},
		},
		{
			caseName: "Correct WRC20 test",
			decoded: decodedOp{
				op:      StdWRC20,
				value:   opValue,
				spender: opSpender,
			},
			encoded: []byte{
				243, 13, 248, 135, 20, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 44, 204, 245, 224, 83, 132, 147, 194, 53, 209, 197, 239, 101, 128, 247, 125, 153, 233, 19, 150, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 131, 1, 182, 105, 128, 128, 128,
			},
			errs: []error{nil},
		},
		{
			caseName: "No empty value",
			decoded: decodedOp{
				op:      0,
				value:   nil,
				spender: opSpender,
			},
			encoded: []byte{
				243, 13, 248, 132, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 44, 204, 245, 224, 83, 132, 147, 194, 53, 209, 197, 239, 101, 128, 247, 125, 153, 233, 19, 150, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128,
			},
			errs: []error{ErrNoValue},
		},
		{
			caseName: "No empty spender",
			decoded: decodedOp{
				op:      StdWRC721,
				value:   opId,
				spender: common.Address{},
			},
			encoded: []byte{
				243, 13, 248, 132, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128,
			},
			errs: []error{ErrNoSpender},
		},
	}

	operationEncode := func(b []byte, i interface{}) error {
		o := i.(decodedOp)

		op, err := NewApproveOperation(
			o.op,
			o.spender,
			o.value,
		)
		if err != nil {
			return err
		}

		equalOpBytes(t, op, b)

		return nil
	}

	operationDecode := func(b []byte, i interface{}) error {
		op, err := DecodeBytes(b)
		if err != nil {
			return err
		}

		o := i.(decodedOp)
		opDecoded, ok := op.(Approve)
		if !ok {
			return errors.New("invalid operation type")
		}

		err = checkOpCodeAndStandart(b, opDecoded, o.op)
		if err != nil {
			return err
		}

		operator := opDecoded.Spender()
		if operator != o.spender {
			return fmt.Errorf("values do not match:\nwant: %+v\nhave: %+v", o.spender, operator)
		}

		testutils.CompareBigInt(t, opDecoded.Value(), o.value)

		return nil
	}

	startSubTests(t, cases, operationEncode, operationDecode)
}

func TestBalanceOfOperation(t *testing.T) {
	type decodedOp struct {
		op      Std
		address common.Address
		owner   common.Address
	}

	cases := []operationTestCase{
		{
			caseName: "Correct WRC20 test",
			decoded: decodedOp{
				op:      StdWRC20,
				address: opAddress,
				owner:   opTo,
			},
			encoded: []byte{
				243, 34, 248, 132, 128, 148, 208, 73, 191, 214, 103, 203, 70, 170, 62, 245, 223, 13, 163, 229, 125, 179, 190, 57, 229, 17, 128, 148, 125, 201, 201, 115, 6, 137, 255, 11, 15, 213, 6, 198, 125, 184, 21, 241, 45, 144, 164, 72, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128,
			},
			errs: []error{nil},
		},
		{
			caseName: "Correct WRC721 test",
			decoded: decodedOp{
				op:      StdWRC721,
				address: opAddress,
				owner:   opTo,
			},
			encoded: []byte{
				243, 34, 248, 132, 128, 148, 208, 73, 191, 214, 103, 203, 70, 170, 62, 245, 223, 13, 163, 229, 125, 179, 190, 57, 229, 17, 128, 148, 125, 201, 201, 115, 6, 137, 255, 11, 15, 213, 6, 198, 125, 184, 21, 241, 45, 144, 164, 72, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128,
			},
			errs: []error{nil},
		},
		{
			caseName: "No empty owner",
			decoded: decodedOp{
				op:      StdWRC20,
				address: opAddress,
				owner:   common.Address{},
			},
			encoded: []byte{
				243, 34, 248, 132, 128, 148, 208, 73, 191, 214, 103, 203, 70, 170, 62, 245, 223, 13, 163, 229, 125, 179, 190, 57, 229, 17, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128,
			},
			errs: []error{ErrNoOwner},
		},
		{
			caseName: "No empty token address",
			decoded: decodedOp{
				op:      0,
				address: common.Address{},
				owner:   opTo,
			},
			encoded: []byte{
				243, 34, 248, 132, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 125, 201, 201, 115, 6, 137, 255, 11, 15, 213, 6, 198, 125, 184, 21, 241, 45, 144, 164, 72, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128,
			},
			errs: []error{ErrNoAddress},
		},
	}

	operationEncode := func(b []byte, i interface{}) error {
		o := i.(decodedOp)

		op, err := NewBalanceOfOperation(
			o.address,
			o.owner,
		)
		if err != nil {
			return err
		}

		equalOpBytes(t, op, b)

		return nil
	}

	operationDecode := func(b []byte, i interface{}) error {
		op, err := DecodeBytes(b)
		if err != nil {
			return err
		}

		o := i.(decodedOp)
		opDecoded, ok := op.(BalanceOf)
		if !ok {
			return errors.New("invalid operation type")
		}

		err = checkOpCodeAndStandart(b, opDecoded, 0)
		if err != nil {
			return err
		}

		if o.address != opDecoded.Address() {
			t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", opDecoded.Address(), o.address)
		}

		if o.owner != opDecoded.Owner() {
			t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", opDecoded.Owner(), o.owner)
		}

		return nil
	}

	startSubTests(t, cases, operationEncode, operationDecode)
}

func TestPropertiesOperation(t *testing.T) {
	type decodedOp struct {
		address common.Address
		id      *big.Int
	}

	cases := []operationTestCase{
		{
			caseName: "Correct test",
			decoded: decodedOp{
				address: opAddress,
				id:      opId,
			},
			encoded: []byte{
				243, 33, 248, 134, 128, 148, 208, 73, 191, 214, 103, 203, 70, 170, 62, 245, 223, 13, 163, 229, 125, 179, 190, 57, 229, 17, 130, 48, 57, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128,
			},
			errs: []error{nil},
		},
		{
			caseName: "No empty token id",
			decoded: decodedOp{
				address: opAddress,
				id:      nil,
			},
			encoded: []byte{
				243, 33, 248, 132, 128, 148, 208, 73, 191, 214, 103, 203, 70, 170, 62, 245, 223, 13, 163, 229, 125, 179, 190, 57, 229, 17, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128,
			},
			errs: []error{ErrNoTokenId},
		},
		{
			caseName: "No empty token address",
			decoded: decodedOp{
				address: common.Address{},
				id:      opId,
			},
			encoded: []byte{
				243, 33, 248, 134, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 130, 48, 57, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128,
			},
			errs: []error{ErrNoAddress},
		},
	}

	operationEncode := func(b []byte, i interface{}) error {
		o := i.(decodedOp)

		op, err := NewPropertiesOperation(
			o.address,
			o.id,
		)
		if err != nil {
			return err
		}

		equalOpBytes(t, op, b)

		return nil
	}

	operationDecode := func(b []byte, i interface{}) error {
		op, err := DecodeBytes(b)
		if err != nil {
			return err
		}

		o := i.(decodedOp)
		opDecoded, ok := op.(Properties)
		if !ok {
			return errors.New("invalid operation type")
		}

		err = checkOpCodeAndStandart(b, opDecoded, 0)
		if err != nil {
			return err
		}

		tokenId, ok := opDecoded.TokenId()
		if !ok {
			return errors.New("invalid tokenId")
		}

		testutils.CompareBigInt(t, tokenId, o.id)

		if o.address != opDecoded.Address() {
			t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", o.address, opDecoded.Address())
		}

		return nil
	}

	startSubTests(t, cases, operationEncode, operationDecode)
}

func TestTransferFromOperation(t *testing.T) {
	type decodedOp struct {
		op    Std
		to    common.Address
		value *big.Int
		from  common.Address
	}

	cases := []operationTestCase{
		{
			caseName: "Correct WRC20 test",
			decoded: decodedOp{
				op:    StdWRC20,
				to:    opTo,
				value: opValue,
				from:  opFrom,
			},
			encoded: []byte{
				243, 31, 248, 135, 20, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 121, 134, 186, 216, 31, 76, 189, 147, 23, 245, 164, 104, 97, 67, 125, 174, 88, 214, 145, 19, 148, 125, 201, 201, 115, 6, 137, 255, 11, 15, 213, 6, 198, 125, 184, 21, 241, 45, 144, 164, 72, 131, 1, 182, 105, 128, 128, 128,
			},
			errs: []error{nil},
		},
		{
			caseName: "Correct WRC721 test",
			decoded: decodedOp{
				op:    StdWRC721,
				to:    opTo,
				value: opId,
				from:  opFrom,
			},
			encoded: []byte{
				243, 31, 248, 136, 130, 2, 209, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 121, 134, 186, 216, 31, 76, 189, 147, 23, 245, 164, 104, 97, 67, 125, 174, 88, 214, 145, 19, 148, 125, 201, 201, 115, 6, 137, 255, 11, 15, 213, 6, 198, 125, 184, 21, 241, 45, 144, 164, 72, 130, 48, 57, 128, 128, 128,
			},
			errs: []error{nil},
		},
		{
			caseName: "No empty from",
			decoded: decodedOp{
				op:    StdWRC721,
				to:    opTo,
				value: opValue,
				from:  common.Address{},
			},
			encoded: []byte{
				243, 31, 248, 136, 130, 2, 209, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 125, 201, 201, 115, 6, 137, 255, 11, 15, 213, 6, 198, 125, 184, 21, 241, 45, 144, 164, 72, 130, 48, 57, 128, 128, 128,
			},
			errs: []error{ErrNoFrom},
		},
		{
			caseName: "No empty to",
			decoded: decodedOp{
				op:    StdWRC20,
				to:    common.Address{},
				value: opValue,
				from:  opFrom,
			},
			encoded: []byte{
				243, 31, 248, 135, 20, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 121, 134, 186, 216, 31, 76, 189, 147, 23, 245, 164, 104, 97, 67, 125, 174, 88, 214, 145, 19, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 131, 1, 182, 105, 128, 128, 128,
			},
			errs: []error{ErrNoTo},
		},
	}

	operationEncode := func(b []byte, i interface{}) error {
		o := i.(decodedOp)

		op, err := NewTransferFromOperation(
			o.op,
			o.from,
			o.to,
			o.value,
		)
		if err != nil {
			return err
		}

		equalOpBytes(t, op, b)

		return nil
	}

	operationDecode := func(b []byte, i interface{}) error {
		op, err := DecodeBytes(b)
		if err != nil {
			return err
		}

		o := i.(decodedOp)
		opDecoded, ok := op.(TransferFrom)
		if !ok {
			return errors.New("invalid operation type")
		}

		err = checkOpCodeAndStandart(b, opDecoded, o.op)
		if err != nil {
			return err
		}

		testutils.CompareBigInt(t, opDecoded.Value(), o.value)

		if o.from != opDecoded.From() {
			t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", opDecoded.From(), o.from)
		}

		if o.to != opDecoded.To() {
			t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", opDecoded.To(), o.to)
		}

		return nil
	}

	startSubTests(t, cases, operationEncode, operationDecode)
}
