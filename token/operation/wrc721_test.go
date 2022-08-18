package operation

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/internal/token/testutils"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestBurnOperation(t *testing.T) {
	type decodedOp struct {
		op Std
		id *big.Int
	}

	cases := []operationTestCase{
		{
			caseName: "Correct test",
			decoded: decodedOp{
				op: StdWRC721,
				id: opId,
			},
			encoded: []byte{
				243, 39, 248, 136, 130, 2, 209, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 130, 48, 57, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128,
			},
			errs: []error{nil},
		},
		{
			caseName: "No empty token id",
			decoded: decodedOp{
				op: StdWRC721,
				id: nil,
			},
			encoded: []byte{
				243, 39, 248, 134, 130, 2, 209, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128,
			},
			errs: []error{ErrNoTokenId},
		},
	}

	operationEncode := func(b []byte, i interface{}) error {
		o := i.(decodedOp)

		op, err := NewBurnOperation(
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
		opDecoded, ok := op.(Burn)
		if !ok {
			return errors.New("invalid operation type")
		}

		err = checkOpCodeAndStandard(b, opDecoded, StdWRC721)
		if err != nil {
			return err
		}

		if !testutils.BigIntEquals(opDecoded.TokenId(), o.id) {
			return fmt.Errorf("values do not match:\nwant: %+v\nhave: %+v", opDecoded.TokenId(), o.id)
		}

		tokenId := opDecoded.TokenId()
		if !testutils.BigIntEquals(tokenId, o.id) {
			return fmt.Errorf("id do not match:\nwant: %+v\nhave: %+v", tokenId, o.id)
		}

		if len(tokenId.Bytes()) == 0 {
			// just stub for encoding tests
			return ErrNoTokenId
		}

		return nil
	}

	startSubTests(t, cases, operationEncode, operationDecode)
}

func TestMintOperation(t *testing.T) {
	type decodedOp struct {
		op   Std
		to   common.Address
		id   *big.Int
		data []byte
	}

	cases := []operationTestCase{
		{
			caseName: "Correct test",
			decoded: decodedOp{
				op:   StdWRC721,
				to:   opTo,
				id:   opId,
				data: oData,
			},
			encoded: []byte{
				243, 38, 248, 149, 130, 2, 209, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 130, 48, 57, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 125, 201, 201, 115, 6, 137, 255, 11, 15, 213, 6, 198, 125, 184, 21, 241, 45, 144, 164, 72, 128, 128, 128, 141, 243, 12, 202, 20, 133, 116, 111, 107, 101, 110, 116, 100, 5,
			},
			errs: []error{nil},
		},
		{
			caseName: "No empty tokenID",
			decoded: decodedOp{
				op:   StdWRC20,
				to:   opTo,
				id:   nil,
				data: nil,
			},
			encoded: []byte{
				243, 38, 248, 134, 130, 2, 209, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 125, 201, 201, 115, 6, 137, 255, 11, 15, 213, 6, 198, 125, 184, 21, 241, 45, 144, 164, 72, 128, 128, 128, 128,
			},
			errs: []error{ErrNoTokenId},
		},
		{
			caseName: "No empty to",
			decoded: decodedOp{
				op:   0,
				to:   common.Address{},
				id:   nil,
				data: nil,
			},
			encoded: []byte{
				243, 38, 248, 134, 130, 2, 209, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128,
			},
			errs: []error{ErrNoTo, ErrNoTokenId},
		},
		{
			caseName: "Correct test without op and data",
			decoded: decodedOp{
				op:   0,
				to:   opTo,
				id:   opId,
				data: nil,
			},
			encoded: []byte{
				243, 38, 248, 136, 130, 2, 209, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 130, 48, 57, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 125, 201, 201, 115, 6, 137, 255, 11, 15, 213, 6, 198, 125, 184, 21, 241, 45, 144, 164, 72, 128, 128, 128, 128,
			},
			errs: []error{nil},
		},
	}

	operationEncode := func(b []byte, i interface{}) error {
		o := i.(decodedOp)

		op, err := NewMintOperation(
			o.to,
			o.id,
			o.data,
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
		opDecoded, ok := op.(Mint)
		if !ok {
			return errors.New("invalid operation type")
		}

		err = checkOpCodeAndStandard(b, opDecoded, StdWRC721)
		if err != nil {
			return err
		}

		if opDecoded.To() != o.to {
			return fmt.Errorf("values do not match:\nwant: %+v\nhave: %+v", opDecoded.To(), o.to)
		}

		haveData, ok := opDecoded.Metadata()
		if !ok {
			t.Log("have data is empty")
		}

		if !bytes.Equal(haveData, o.data) {
			return fmt.Errorf("values do not match:\nwant: %+v\nhave: %+v", o.data, haveData)
		}

		tokenId := opDecoded.TokenId()
		if !testutils.BigIntEquals(tokenId, o.id) {
			return fmt.Errorf("id do not match:\nwant: %+v\nhave: %+v", tokenId, o.id)
		}

		if len(tokenId.Bytes()) == 0 {
			// just stub for encoding tests
			return ErrNoTokenId
		}

		return nil
	}

	startSubTests(t, cases, operationEncode, operationDecode)
}

func TestSetApprovalForAllOperation(t *testing.T) {
	type decodedOp struct {
		op       Std
		operator common.Address
		approve  bool
	}

	cases := []operationTestCase{
		{
			caseName: "Correct test approve",
			decoded: decodedOp{
				op:       StdWRC721,
				operator: opOperator,
				approve:  true,
			},
			encoded: []byte{
				243, 37, 248, 134, 130, 2, 209, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 19, 228, 172, 239, 230, 166, 112, 6, 4, 146, 153, 70, 231, 14, 100, 67, 228, 231, 52, 71, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 1, 128,
			},
			errs: []error{nil},
		},
		{
			caseName: "Approve no empty operator",
			decoded: decodedOp{
				op:       StdWRC20,
				operator: common.Address{},
				approve:  true,
			},
			encoded: []byte{
				243, 37, 248, 134, 130, 2, 209, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 1, 128,
			},
			errs: []error{ErrNoOperator},
		},
		{
			caseName: "Correct test unapprove",
			decoded: decodedOp{

				op:       StdWRC721,
				operator: opOperator,
				approve:  false,
			},
			encoded: []byte{
				243, 37, 248, 134, 130, 2, 209, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 19, 228, 172, 239, 230, 166, 112, 6, 4, 146, 153, 70, 231, 14, 100, 67, 228, 231, 52, 71, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128,
			},
			errs: []error{nil},
		},
		{
			caseName: "Unapprove no empty operator",
			decoded: decodedOp{
				op:       0,
				operator: common.Address{},
				approve:  false,
			},
			encoded: []byte{
				243, 37, 248, 132, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128,
			},
			errs: []error{ErrNoOperator},
		},
	}

	operationEncode := func(b []byte, i interface{}) error {
		o := i.(decodedOp)

		op, err := NewSetApprovalForAllOperation(
			o.operator,
			o.approve,
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
		opDecoded, ok := op.(SetApprovalForAll)
		if !ok {
			return errors.New("invalid operation type")
		}

		err = checkOpCodeAndStandard(b, opDecoded, StdWRC721)
		if err != nil {
			return err
		}

		operator := opDecoded.Operator()
		if operator != o.operator {
			return fmt.Errorf("values do not match:\nwant: %+v\nhave: %+v", o.operator, operator)
		}

		isApproved := opDecoded.IsApproved()
		if isApproved != o.approve {
			return fmt.Errorf("values do not match:\nwant: %+v\nhave: %+v", o.approve, isApproved)
		}

		return nil
	}

	startSubTests(t, cases, operationEncode, operationDecode)
}

func TestIsApprovedForAllOperation(t *testing.T) {
	type decodedOp struct {
		op       Std
		address  common.Address
		owner    common.Address
		operator common.Address
	}

	cases := []operationTestCase{
		{
			caseName: "Correct test",
			decoded: decodedOp{
				op:       StdWRC721,
				address:  opAddress,
				owner:    opOwner,
				operator: opOperator,
			},
			encoded: []byte{
				243, 36, 248, 134, 130, 2, 209, 148, 208, 73, 191, 214, 103, 203, 70, 170, 62, 245, 223, 13, 163, 229, 125, 179, 190, 57, 229, 17, 128, 148, 25, 119, 194, 72, 225, 1, 76, 193, 3, 146, 157, 215, 241, 84, 25, 156, 145, 110, 57, 236, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 19, 228, 172, 239, 230, 166, 112, 6, 4, 146, 153, 70, 231, 14, 100, 67, 228, 231, 52, 71, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128,
			},
			errs: []error{nil},
		},
		{
			caseName: "No empty owner",
			decoded: decodedOp{
				op:       StdWRC20,
				address:  opAddress,
				owner:    common.Address{},
				operator: opOperator,
			},
			encoded: []byte{
				243, 36, 248, 132, 20, 148, 208, 73, 191, 214, 103, 203, 70, 170, 62, 245, 223, 13, 163, 229, 125, 179, 190, 57, 229, 17, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 19, 228, 172, 239, 230, 166, 112, 6, 4, 146, 153, 70, 231, 14, 100, 67, 228, 231, 52, 71, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128,
			},
			errs: []error{ErrNoOwner},
		},
		{
			caseName: "No empty address",
			decoded: decodedOp{
				op:       0,
				address:  common.Address{},
				owner:    opOwner,
				operator: opOperator,
			},
			encoded: []byte{
				243, 36, 248, 132, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 25, 119, 194, 72, 225, 1, 76, 193, 3, 146, 157, 215, 241, 84, 25, 156, 145, 110, 57, 236, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 19, 228, 172, 239, 230, 166, 112, 6, 4, 146, 153, 70, 231, 14, 100, 67, 228, 231, 52, 71, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128,
			},
			errs: []error{ErrNoAddress},
		},
		{
			caseName: "No empty operator",
			decoded: decodedOp{
				op:       0,
				address:  opAddress,
				owner:    opOwner,
				operator: common.Address{},
			},
			encoded: []byte{
				243, 36, 248, 132, 128, 148, 208, 73, 191, 214, 103, 203, 70, 170, 62, 245, 223, 13, 163, 229, 125, 179, 190, 57, 229, 17, 128, 148, 25, 119, 194, 72, 225, 1, 76, 193, 3, 146, 157, 215, 241, 84, 25, 156, 145, 110, 57, 236, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128,
			},
			errs: []error{ErrNoOperator},
		},
	}

	operationEncode := func(b []byte, i interface{}) error {
		o := i.(decodedOp)

		op, err := NewIsApprovedForAllOperation(
			o.address,
			o.owner,
			o.operator,
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
		opDecoded, ok := op.(IsApprovedForAll)
		if !ok {
			return errors.New("invalid operation type")
		}

		err = checkOpCodeAndStandard(b, opDecoded, StdWRC721)
		if err != nil {
			return err
		}

		operator := opDecoded.Operator()
		if operator != o.operator {
			return fmt.Errorf("values do not match:\nwant: %+v\nhave: %+v", o.operator, operator)
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

func TestSafeTransferFromOperation(t *testing.T) {
	type decodedOp struct {
		op    Std
		to    common.Address
		value *big.Int
		from  common.Address
		data  []byte
	}

	cases := []operationTestCase{
		{
			caseName: "Correct test",
			decoded: decodedOp{
				op:    StdWRC721,
				to:    opTo,
				value: opValue,
				from:  opFrom,
				data:  oData,
			},
			encoded: []byte{
				243, 41, 248, 150, 130, 2, 209, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 121, 134, 186, 216, 31, 76, 189, 147, 23, 245, 164, 104, 97, 67, 125, 174, 88, 214, 145, 19, 148, 125, 201, 201, 115, 6, 137, 255, 11, 15, 213, 6, 198, 125, 184, 21, 241, 45, 144, 164, 72, 131, 1, 182, 105, 128, 128, 141, 243, 12, 202, 20, 133, 116, 111, 107, 101, 110, 116, 100, 5,
			},
			errs: []error{nil},
		},
		{
			caseName: "No empty value",
			decoded: decodedOp{
				op:    StdWRC721,
				to:    opTo,
				value: nil,
				from:  opFrom,
				data:  nil,
			},
			encoded: []byte{
				243, 41, 248, 134, 130, 2, 209, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 121, 134, 186, 216, 31, 76, 189, 147, 23, 245, 164, 104, 97, 67, 125, 174, 88, 214, 145, 19, 148, 125, 201, 201, 115, 6, 137, 255, 11, 15, 213, 6, 198, 125, 184, 21, 241, 45, 144, 164, 72, 128, 128, 128, 128,
			},
			errs: []error{ErrNoValue},
		},
		{
			caseName: "No empty from",
			decoded: decodedOp{
				op:    0,
				to:    opTo,
				value: opValue,
				from:  common.Address{},
				data:  nil,
			},
			encoded: []byte{
				243, 41, 248, 135, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 125, 201, 201, 115, 6, 137, 255, 11, 15, 213, 6, 198, 125, 184, 21, 241, 45, 144, 164, 72, 131, 1, 182, 105, 128, 128, 128,
			},
			errs: []error{ErrNoFrom},
		},
		{
			caseName: "No empty to",
			decoded: decodedOp{
				op:    StdWRC721,
				to:    common.Address{},
				value: opValue,
				from:  opFrom,
				data:  nil,
			},
			encoded: []byte{
				243, 41, 248, 134, 130, 2, 209, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 121, 134, 186, 216, 31, 76, 189, 147, 23, 245, 164, 104, 97, 67, 125, 174, 88, 214, 145, 19, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128,
			},
			errs: []error{ErrNoTo},
		},
	}

	operationEncode := func(b []byte, i interface{}) error {
		o := i.(decodedOp)

		op, err := NewSafeTransferFromOperation(
			o.from,
			o.to,
			o.value,
			o.data,
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
		opDecoded, ok := op.(SafeTransferFrom)
		if !ok {
			return errors.New("invalid operation type")
		}

		err = checkOpCodeAndStandard(b, opDecoded, StdWRC721)
		if err != nil {
			return err
		}

		data, ok := opDecoded.Data()
		if !ok {
			t.Logf("data is empty")
		}

		if !bytes.Equal(data, o.data) {
			t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", o.data, data)
		}

		value := opDecoded.Value()
		if !testutils.BigIntEquals(value, o.value) {
			return fmt.Errorf("values do not match:\nwant: %+v\nhave: %+v", value, o.value)
		}

		if len(value.Bytes()) == 0 {
			// just stub for encoding tests
			return ErrNoValue
		}

		return nil
	}

	startSubTests(t, cases, operationEncode, operationDecode)
}

func TestTokenOfOwnerByIndexOperation(t *testing.T) {
	type decodedOp struct {
		op      Std
		address common.Address
		owner   common.Address
		index   *big.Int
	}

	cases := []operationTestCase{
		{
			caseName: "Correct test",
			decoded: decodedOp{
				op:      StdWRC721,
				address: opAddress,
				owner:   opOwner,
				index:   opIndex,
			},
			encoded: []byte{
				243, 40, 248, 136, 130, 2, 209, 148, 208, 73, 191, 214, 103, 203, 70, 170, 62, 245, 223, 13, 163, 229, 125, 179, 190, 57, 229, 17, 128, 148, 25, 119, 194, 72, 225, 1, 76, 193, 3, 146, 157, 215, 241, 84, 25, 156, 145, 110, 57, 236, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 130, 217, 3, 128, 128,
			},
			errs: []error{nil},
		},
		{
			caseName: "No empty token address",
			decoded: decodedOp{
				op:      StdWRC721,
				address: common.Address{},
				owner:   opOwner,
				index:   opIndex,
			},
			encoded: []byte{
				243, 40, 248, 136, 130, 2, 209, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 25, 119, 194, 72, 225, 1, 76, 193, 3, 146, 157, 215, 241, 84, 25, 156, 145, 110, 57, 236, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 130, 217, 3, 128, 128,
			},
			errs: []error{ErrNoAddress},
		},
		{
			caseName: "No empty owner",
			decoded: decodedOp{
				op:      StdWRC721,
				address: opAddress,
				owner:   common.Address{},
				index:   opIndex,
			},
			encoded: []byte{
				243, 40, 248, 136, 130, 2, 209, 148, 208, 73, 191, 214, 103, 203, 70, 170, 62, 245, 223, 13, 163, 229, 125, 179, 190, 57, 229, 17, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 130, 217, 3, 128, 128,
			},
			errs: []error{ErrNoOwner},
		},
		{
			caseName: "No empty index",
			decoded: decodedOp{
				op:      StdWRC721,
				address: opAddress,
				owner:   opOwner,
				index:   nil,
			},
			encoded: []byte{
				243, 40, 248, 134, 130, 2, 209, 148, 208, 73, 191, 214, 103, 203, 70, 170, 62, 245, 223, 13, 163, 229, 125, 179, 190, 57, 229, 17, 128, 148, 25, 119, 194, 72, 225, 1, 76, 193, 3, 146, 157, 215, 241, 84, 25, 156, 145, 110, 57, 236, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128,
			},
			errs: []error{ErrNoIndex},
		},
	}

	operationEncode := func(b []byte, i interface{}) error {
		o := i.(decodedOp)

		op, err := NewTokenOfOwnerByIndexOperation(
			o.address,
			o.owner,
			o.index,
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
		opDecoded, ok := op.(TokenOfOwnerByIndex)
		if !ok {
			return errors.New("invalid operation type")
		}

		err = checkOpCodeAndStandard(b, opDecoded, StdWRC721)
		if err != nil {
			return err
		}

		if !testutils.BigIntEquals(opDecoded.Index(), o.index) {
			return fmt.Errorf("values do not match:\nwant: %+v\nhave: %+v", opDecoded.Index(), o.index)
		}

		if o.owner != opDecoded.Owner() {
			t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", o.owner, opDecoded.Owner())
		}

		if o.address != opDecoded.Address() {
			t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", o.address, opDecoded.Address())
		}

		gotIndex := opDecoded.Index()
		if !testutils.BigIntEquals(gotIndex, o.index) {
			return fmt.Errorf("indexes do not match:\nwant: %+v\nhave: %+v", gotIndex, o.index)
		}

		if len(gotIndex.Bytes()) == 0 {
			// just stub for encoding tests
			return ErrNoIndex
		}

		return nil
	}

	startSubTests(t, cases, operationEncode, operationDecode)
}
