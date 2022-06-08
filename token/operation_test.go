package token

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"testing"
)

var (
	operator    = common.HexToAddress("13e4acefe6a6700604929946e70e6443e4e73447")
	address     = common.HexToAddress("d049bfd667cb46aa3ef5df0da3e57db3be39e511")
	spender     = common.HexToAddress("2cccf5e0538493c235d1c5ef6580f77d99e91396")
	owner       = common.HexToAddress("1977c248e1014cc103929dd7f154199c916e39ec")
	to          = common.HexToAddress("7dc9c9730689ff0b0fd506c67db815f12d90a448")
	from        = common.HexToAddress("7986bad81f4cbd9317f5a46861437dae58d69113")
	value       = big.NewInt(112233)
	id          = big.NewInt(12345)
	totalSupply = big.NewInt(9999)
	index       = big.NewInt(55555)
	decimals    = uint8(100)
	name        = []byte("Test Tokken")
	symbol      = []byte("TT")
	baseURI     = []byte("test.token.com")
	data        = []byte{243, 12, 202, 20, 133, 116, 111, 107, 101, 110, 116, 100, 5}
)

type testCase struct {
	caseName string
	decoded  interface{}
	encoded  []byte
	errs     []error
}

func TestMintOperation(t *testing.T) {
	type decodedOp struct {
		op   Std
		to   common.Address
		id   *big.Int
		data []byte
	}

	cases := []testCase{
		{
			caseName: "mintOperation 1",
			decoded: decodedOp{
				op:   StdWRC721,
				to:   to,
				id:   id,
				data: data,
			},
			encoded: []byte{
				243, 38, 248, 149, 130, 2, 209, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 130, 48, 57, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 125, 201, 201, 115, 6, 137, 255, 11, 15, 213, 6, 198, 125, 184, 21, 241, 45, 144, 164, 72, 128, 128, 128, 141, 243, 12, 202, 20, 133, 116, 111, 107, 101, 110, 116, 100, 5,
			},
			errs: []error{},
		},
		{
			caseName: "mintOperation 2",
			decoded: decodedOp{
				op:   StdWRC20,
				to:   to,
				id:   nil,
				data: nil,
			},
			encoded: []byte{
				243, 38, 248, 134, 130, 2, 209, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 125, 201, 201, 115, 6, 137, 255, 11, 15, 213, 6, 198, 125, 184, 21, 241, 45, 144, 164, 72, 128, 128, 128, 128,
			},
			errs: []error{ErrNoTokenId},
		},
		{
			caseName: "mintOperation 3",
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
			caseName: "mintOperation 4",
			decoded: decodedOp{
				op:   0,
				to:   to,
				id:   id,
				data: nil,
			},
			encoded: []byte{
				243, 38, 248, 136, 130, 2, 209, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 130, 48, 57, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 125, 201, 201, 115, 6, 137, 255, 11, 15, 213, 6, 198, 125, 184, 21, 241, 45, 144, 164, 72, 128, 128, 128, 128,
			},
			errs: []error{},
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

		err = equalOpBytes(op, b)
		if err != nil {
			return err
		}

		return nil
	}

	operationDecode := func(b []byte, i interface{}) error {
		op, err := DecodeBytes(b)
		if err != nil {
			return err
		}

		o := i.(decodedOp)
		opDecoded, ok := op.(MintOperation)
		if !ok {
			return errors.New("invalid operation type")
		}

		err = checkOpCodeAndStandart(b, opDecoded, StdWRC721)
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

		err = checkBigInt(opDecoded.TokenId(), o.id)
		if err != nil {
			return err
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

	cases := []testCase{
		{
			caseName: "setApprovalForAllOperation 1",
			decoded: decodedOp{
				op:       StdWRC721,
				operator: operator,
				approve:  true,
			},
			encoded: []byte{
				243, 37, 248, 134, 130, 2, 209, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 19, 228, 172, 239, 230, 166, 112, 6, 4, 146, 153, 70, 231, 14, 100, 67, 228, 231, 52, 71, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 1, 128,
			},
			errs: []error{},
		},
		{
			caseName: "setApprovalForAllOperation 2",
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
			caseName: "setApprovalForAllOperation 3",
			decoded: decodedOp{

				op:       StdWRC721,
				operator: operator,
				approve:  false,
			},
			encoded: []byte{
				243, 37, 248, 134, 130, 2, 209, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 19, 228, 172, 239, 230, 166, 112, 6, 4, 146, 153, 70, 231, 14, 100, 67, 228, 231, 52, 71, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128,
			},
			errs: []error{},
		},
		{
			caseName: "setApprovalForAllOperation 4",
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

		err = equalOpBytes(op, b)
		if err != nil {
			return err
		}

		return nil
	}

	operationDecode := func(b []byte, i interface{}) error {
		op, err := DecodeBytes(b)
		if err != nil {
			return err
		}

		o := i.(decodedOp)
		opDecoded, ok := op.(SetApprovalForAllOperation)
		if !ok {
			return errors.New("invalid operation type")
		}

		err = checkOpCodeAndStandart(b, opDecoded, StdWRC721)
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

	cases := []testCase{
		{
			caseName: "isApprovedForAllOperation 1",
			decoded: decodedOp{
				op:       StdWRC721,
				address:  address,
				owner:    owner,
				operator: operator,
			},
			encoded: []byte{
				243, 36, 248, 134, 130, 2, 209, 148, 208, 73, 191, 214, 103, 203, 70, 170, 62, 245, 223, 13, 163, 229, 125, 179, 190, 57, 229, 17, 128, 148, 25, 119, 194, 72, 225, 1, 76, 193, 3, 146, 157, 215, 241, 84, 25, 156, 145, 110, 57, 236, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 19, 228, 172, 239, 230, 166, 112, 6, 4, 146, 153, 70, 231, 14, 100, 67, 228, 231, 52, 71, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128,
			},
			errs: []error{},
		},
		{
			caseName: "setApprovalForAllOperation 2",
			decoded: decodedOp{
				op:       StdWRC20,
				address:  address,
				owner:    common.Address{},
				operator: operator,
			},
			encoded: []byte{
				243, 36, 248, 132, 20, 148, 208, 73, 191, 214, 103, 203, 70, 170, 62, 245, 223, 13, 163, 229, 125, 179, 190, 57, 229, 17, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 19, 228, 172, 239, 230, 166, 112, 6, 4, 146, 153, 70, 231, 14, 100, 67, 228, 231, 52, 71, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128,
			},
			errs: []error{ErrNoOwner},
		},
		{
			caseName: "setApprovalForAllOperation 3",
			decoded: decodedOp{
				op:       0,
				address:  common.Address{},
				owner:    owner,
				operator: operator,
			},
			encoded: []byte{
				243, 36, 248, 132, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 25, 119, 194, 72, 225, 1, 76, 193, 3, 146, 157, 215, 241, 84, 25, 156, 145, 110, 57, 236, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 19, 228, 172, 239, 230, 166, 112, 6, 4, 146, 153, 70, 231, 14, 100, 67, 228, 231, 52, 71, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128,
			},
			errs: []error{ErrNoAddress},
		},
		{
			caseName: "setApprovalForAllOperation 4",
			decoded: decodedOp{
				op:       0,
				address:  address,
				owner:    owner,
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

		err = equalOpBytes(op, b)
		if err != nil {
			return err
		}

		return nil
	}

	operationDecode := func(b []byte, i interface{}) error {
		op, err := DecodeBytes(b)
		if err != nil {
			return err
		}

		o := i.(decodedOp)
		opDecoded, ok := op.(IsApprovedForAllOperation)
		if !ok {
			return errors.New("invalid operation type")
		}

		err = checkOpCodeAndStandart(b, opDecoded, StdWRC721)
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

func TestAllowanceOperation(t *testing.T) {
	type decodedOp struct {
		op      Std
		address common.Address
		owner   common.Address
		spender common.Address
	}

	cases := []testCase{
		{
			caseName: "AllowanceOperation 1",
			decoded: decodedOp{
				op:      StdWRC721,
				address: address,
				owner:   owner,
				spender: spender,
			},
			encoded: []byte{
				243, 35, 248, 132, 20, 148, 208, 73, 191, 214, 103, 203, 70, 170, 62, 245, 223, 13, 163, 229, 125, 179, 190, 57, 229, 17, 128, 148, 25, 119, 194, 72, 225, 1, 76, 193, 3, 146, 157, 215, 241, 84, 25, 156, 145, 110, 57, 236, 148, 44, 204, 245, 224, 83, 132, 147, 194, 53, 209, 197, 239, 101, 128, 247, 125, 153, 233, 19, 150, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128,
			},
			errs: []error{},
		},
		{
			caseName: "AllowanceOperation 2",
			decoded: decodedOp{
				op:      StdWRC20,
				address: address,
				owner:   common.Address{},
				spender: spender,
			},
			encoded: []byte{
				243, 35, 248, 132, 20, 148, 208, 73, 191, 214, 103, 203, 70, 170, 62, 245, 223, 13, 163, 229, 125, 179, 190, 57, 229, 17, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 44, 204, 245, 224, 83, 132, 147, 194, 53, 209, 197, 239, 101, 128, 247, 125, 153, 233, 19, 150, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128,
			},
			errs: []error{ErrNoOwner},
		},
		{
			caseName: "AllowanceOperation 3",
			decoded: decodedOp{
				op:      0,
				address: common.Address{},
				owner:   owner,
				spender: spender,
			},
			encoded: []byte{
				243, 35, 248, 132, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 25, 119, 194, 72, 225, 1, 76, 193, 3, 146, 157, 215, 241, 84, 25, 156, 145, 110, 57, 236, 148, 44, 204, 245, 224, 83, 132, 147, 194, 53, 209, 197, 239, 101, 128, 247, 125, 153, 233, 19, 150, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128,
			},
			errs: []error{ErrNoAddress},
		},
		{
			caseName: "AllowanceOperation 4",
			decoded: decodedOp{
				op:      0,
				address: address,
				owner:   owner,
				spender: common.Address{},
			},
			encoded: []byte{
				243, 35, 248, 132, 128, 148, 208, 73, 191, 214, 103, 203, 70, 170, 62, 245, 223, 13, 163, 229, 125, 179, 190, 57, 229, 17, 128, 148, 25, 119, 194, 72, 225, 1, 76, 193, 3, 146, 157, 215, 241, 84, 25, 156, 145, 110, 57, 236, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128,
			},
			errs: []error{ErrNoSpender},
		},
	}

	operationEncode := func(b []byte, i interface{}) error {
		o := i.(decodedOp)

		op, err := NewAllowanceOperation(
			o.address,
			o.owner,
			o.spender,
		)
		if err != nil {
			return err
		}

		err = equalOpBytes(op, b)
		if err != nil {
			return err
		}

		return nil
	}

	operationDecode := func(b []byte, i interface{}) error {
		op, err := DecodeBytes(b)
		if err != nil {
			return err
		}

		o := i.(decodedOp)
		opDecoded, ok := op.(AllowanceOperation)
		if !ok {
			return errors.New("invalid operation type")
		}

		err = checkOpCodeAndStandart(b, opDecoded, StdWRC20)
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

func TestApproveOperation(t *testing.T) {
	type decodedOp struct {
		op      Std
		spender common.Address
		value   *big.Int
	}

	cases := []testCase{
		{
			caseName: "ApproveOperation 1",
			decoded: decodedOp{
				op:      StdWRC721,
				value:   value,
				spender: spender,
			},
			encoded: []byte{
				243, 13, 248, 137, 130, 2, 209, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 44, 204, 245, 224, 83, 132, 147, 194, 53, 209, 197, 239, 101, 128, 247, 125, 153, 233, 19, 150, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 131, 1, 182, 105, 128, 128, 128,
			},
			errs: []error{},
		},
		{
			caseName: "ApproveOperation 2",
			decoded: decodedOp{
				op:      StdWRC20,
				value:   value,
				spender: spender,
			},
			encoded: []byte{
				243, 13, 248, 135, 20, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 44, 204, 245, 224, 83, 132, 147, 194, 53, 209, 197, 239, 101, 128, 247, 125, 153, 233, 19, 150, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 131, 1, 182, 105, 128, 128, 128,
			},
			errs: []error{},
		},
		{
			caseName: "ApproveOperation 3",
			decoded: decodedOp{
				op:      0,
				value:   nil,
				spender: spender,
			},
			encoded: []byte{
				243, 13, 248, 132, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 44, 204, 245, 224, 83, 132, 147, 194, 53, 209, 197, 239, 101, 128, 247, 125, 153, 233, 19, 150, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128,
			},
			errs: []error{ErrNoValue},
		},
		{
			caseName: "ApproveOperation 4",
			decoded: decodedOp{
				op:      0,
				value:   nil,
				spender: common.Address{},
			},
			encoded: []byte{
				243, 13, 248, 132, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128,
			},
			errs: []error{ErrNoSpender, ErrNoValue},
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

		err = equalOpBytes(op, b)
		if err != nil {
			return err
		}

		return nil
	}

	operationDecode := func(b []byte, i interface{}) error {
		op, err := DecodeBytes(b)
		if err != nil {
			return err
		}

		o := i.(decodedOp)
		opDecoded, ok := op.(ApproveOperation)
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

		err = checkBigInt(opDecoded.Value(), o.value)
		if err != nil {
			return err
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

	cases := []testCase{
		{
			caseName: "SafeTransferFromOperation 1",
			decoded: decodedOp{
				op:    StdWRC721,
				to:    to,
				value: value,
				from:  from,
				data:  data,
			},
			encoded: []byte{
				243, 41, 248, 150, 130, 2, 209, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 121, 134, 186, 216, 31, 76, 189, 147, 23, 245, 164, 104, 97, 67, 125, 174, 88, 214, 145, 19, 148, 125, 201, 201, 115, 6, 137, 255, 11, 15, 213, 6, 198, 125, 184, 21, 241, 45, 144, 164, 72, 131, 1, 182, 105, 128, 128, 141, 243, 12, 202, 20, 133, 116, 111, 107, 101, 110, 116, 100, 5,
			},
			errs: []error{},
		},
		{
			caseName: "SafeTransferFromOperation 2",
			decoded: decodedOp{
				op:    StdWRC721,
				to:    to,
				value: nil,
				from:  from,
				data:  nil,
			},
			encoded: []byte{
				243, 41, 248, 134, 130, 2, 209, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 121, 134, 186, 216, 31, 76, 189, 147, 23, 245, 164, 104, 97, 67, 125, 174, 88, 214, 145, 19, 148, 125, 201, 201, 115, 6, 137, 255, 11, 15, 213, 6, 198, 125, 184, 21, 241, 45, 144, 164, 72, 128, 128, 128, 128,
			},
			errs: []error{ErrNoValue},
		},
		{
			caseName: "SafeTransferFromOperation 3",
			decoded: decodedOp{
				op:    0,
				to:    to,
				value: value,
				from:  common.Address{},
				data:  nil,
			},
			encoded: []byte{
				243, 41, 248, 135, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 125, 201, 201, 115, 6, 137, 255, 11, 15, 213, 6, 198, 125, 184, 21, 241, 45, 144, 164, 72, 131, 1, 182, 105, 128, 128, 128,
			},
			errs: []error{ErrNoFrom},
		},
		{
			caseName: "SafeTransferFromOperation 4",
			decoded: decodedOp{
				op:    StdWRC721,
				to:    common.Address{},
				value: nil,
				from:  from,
				data:  nil,
			},
			encoded: []byte{
				243, 41, 248, 134, 130, 2, 209, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 121, 134, 186, 216, 31, 76, 189, 147, 23, 245, 164, 104, 97, 67, 125, 174, 88, 214, 145, 19, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128,
			},
			errs: []error{ErrNoTo, ErrNoValue},
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

		err = equalOpBytes(op, b)
		if err != nil {
			return err
		}

		return nil
	}

	operationDecode := func(b []byte, i interface{}) error {
		op, err := DecodeBytes(b)
		if err != nil {
			return err
		}

		o := i.(decodedOp)
		opDecoded, ok := op.(SafeTransferFromOperation)
		if !ok {
			return errors.New("invalid operation type")
		}

		err = checkOpCodeAndStandart(b, opDecoded, StdWRC721)
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

		err = checkBigInt(opDecoded.Value(), o.value)
		if err != nil {
			return err
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

	cases := []testCase{
		{
			caseName: "TransferFromOperation 1",
			decoded: decodedOp{
				op:    StdWRC20,
				to:    to,
				value: value,
				from:  from,
			},
			encoded: []byte{
				243, 31, 248, 135, 20, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 121, 134, 186, 216, 31, 76, 189, 147, 23, 245, 164, 104, 97, 67, 125, 174, 88, 214, 145, 19, 148, 125, 201, 201, 115, 6, 137, 255, 11, 15, 213, 6, 198, 125, 184, 21, 241, 45, 144, 164, 72, 131, 1, 182, 105, 128, 128, 128,
			},
			errs: []error{},
		},
		{
			caseName: "TransferFromOperation 2",
			decoded: decodedOp{
				op:    StdWRC721,
				to:    to,
				value: id,
				from:  from,
			},
			encoded: []byte{
				243, 31, 248, 136, 130, 2, 209, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 121, 134, 186, 216, 31, 76, 189, 147, 23, 245, 164, 104, 97, 67, 125, 174, 88, 214, 145, 19, 148, 125, 201, 201, 115, 6, 137, 255, 11, 15, 213, 6, 198, 125, 184, 21, 241, 45, 144, 164, 72, 130, 48, 57, 128, 128, 128,
			},
			errs: []error{},
		},
		{
			caseName: "TransferFromOperation 3",
			decoded: decodedOp{
				op:    StdWRC721,
				to:    to,
				value: value,
				from:  common.Address{},
			},
			encoded: []byte{
				243, 31, 248, 136, 130, 2, 209, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 125, 201, 201, 115, 6, 137, 255, 11, 15, 213, 6, 198, 125, 184, 21, 241, 45, 144, 164, 72, 130, 48, 57, 128, 128, 128,
			},
			errs: []error{ErrNoFrom},
		},
		{
			caseName: "TransferFromOperation 4",
			decoded: decodedOp{
				op:    StdWRC20,
				to:    common.Address{},
				value: value,
				from:  from,
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

		err = equalOpBytes(op, b)
		if err != nil {
			return err
		}

		return nil
	}

	operationDecode := func(b []byte, i interface{}) error {
		op, err := DecodeBytes(b)
		if err != nil {
			return err
		}

		o := i.(decodedOp)
		opDecoded, ok := op.(TransferFromOperation)
		if !ok {
			return errors.New("invalid operation type")
		}

		err = checkOpCodeAndStandart(b, opDecoded, o.op)
		if err != nil {
			return err
		}

		err = checkBigInt(opDecoded.Value(), o.value)
		if err != nil {
			return err
		}

		if o.from != opDecoded.From() {
			t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", opDecoded.From(), from)
		}

		if o.to != opDecoded.To() {
			t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", opDecoded.To(), o.to)
		}

		return nil
	}

	startSubTests(t, cases, operationEncode, operationDecode)
}

func TestTransferOperation(t *testing.T) {
	type decodedOp struct {
		op    Std
		value *big.Int
		to    common.Address
	}

	cases := []testCase{
		{
			caseName: "TransferOperation 1",
			decoded: decodedOp{
				op:    StdWRC20,
				value: value,
				to:    to,
			},
			encoded: []byte{
				243, 30, 248, 135, 20, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 125, 201, 201, 115, 6, 137, 255, 11, 15, 213, 6, 198, 125, 184, 21, 241, 45, 144, 164, 72, 131, 1, 182, 105, 128, 128, 128,
			},
			errs: []error{},
		},
		{
			caseName: "TransferOperation 2",
			decoded: decodedOp{
				op:    0,
				value: nil,
				to:    common.Address{},
			},
			encoded: []byte{
				243, 30, 248, 132, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128,
			},
			errs: []error{ErrNoTo, ErrNoValue},
		},
		{
			caseName: "TransferOperation 3",
			decoded: decodedOp{
				op:    StdWRC20,
				value: value,
				to:    common.Address{},
			},
			encoded: []byte{
				243, 30, 248, 135, 20, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 131, 1, 182, 105, 128, 128, 128,
			},
			errs: []error{ErrNoTo},
		},
		{
			caseName: "TransferOperation 4",
			decoded: decodedOp{
				op:    StdWRC20,
				value: nil,
				to:    to,
			},
			encoded: []byte{
				243, 30, 248, 132, 20, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 125, 201, 201, 115, 6, 137, 255, 11, 15, 213, 6, 198, 125, 184, 21, 241, 45, 144, 164, 72, 128, 128, 128, 128,
			},
			errs: []error{ErrNoValue},
		},
	}

	operationEncode := func(b []byte, i interface{}) error {
		o := i.(decodedOp)

		op, err := NewTransferOperation(
			o.to,
			o.value,
		)
		if err != nil {
			return err
		}

		err = equalOpBytes(op, b)
		if err != nil {
			return err
		}

		return nil
	}

	operationDecode := func(b []byte, i interface{}) error {
		op, err := DecodeBytes(b)
		if err != nil {
			return err
		}

		o := i.(decodedOp)
		opDecoded, ok := op.(TransferOperation)
		if !ok {
			return errors.New("invalid operation type")
		}

		err = checkOpCodeAndStandart(b, opDecoded, o.op)
		if err != nil {
			return err
		}

		err = checkBigInt(opDecoded.Value(), o.value)
		if err != nil {
			return err
		}

		if o.to != opDecoded.To() {
			t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", opDecoded.To(), o.to)
		}

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

	cases := []testCase{
		{
			caseName: "BalanceOfOperation 1",
			decoded: decodedOp{
				op:      StdWRC20,
				address: address,
				owner:   to,
			},
			encoded: []byte{
				243, 34, 248, 132, 128, 148, 208, 73, 191, 214, 103, 203, 70, 170, 62, 245, 223, 13, 163, 229, 125, 179, 190, 57, 229, 17, 128, 148, 125, 201, 201, 115, 6, 137, 255, 11, 15, 213, 6, 198, 125, 184, 21, 241, 45, 144, 164, 72, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128,
			},
			errs: []error{},
		},
		{
			caseName: "BalanceOfOperation 2",
			decoded: decodedOp{
				op:      StdWRC721,
				address: address,
				owner:   to,
			},
			encoded: []byte{
				243, 34, 248, 132, 128, 148, 208, 73, 191, 214, 103, 203, 70, 170, 62, 245, 223, 13, 163, 229, 125, 179, 190, 57, 229, 17, 128, 148, 125, 201, 201, 115, 6, 137, 255, 11, 15, 213, 6, 198, 125, 184, 21, 241, 45, 144, 164, 72, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128,
			},
			errs: []error{},
		},
		{
			caseName: "TransferOperation 3",
			decoded: decodedOp{
				op:      StdWRC20,
				address: address,
				owner:   common.Address{},
			},
			encoded: []byte{
				243, 34, 248, 132, 128, 148, 208, 73, 191, 214, 103, 203, 70, 170, 62, 245, 223, 13, 163, 229, 125, 179, 190, 57, 229, 17, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128,
			},
			errs: []error{ErrNoOwner},
		},
		{
			caseName: "TransferOperation 4",
			decoded: decodedOp{
				op:      0,
				address: common.Address{},
				owner:   to,
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

		err = equalOpBytes(op, b)
		if err != nil {
			return err
		}

		return nil
	}

	operationDecode := func(b []byte, i interface{}) error {
		op, err := DecodeBytes(b)
		if err != nil {
			return err
		}

		o := i.(decodedOp)
		opDecoded, ok := op.(BalanceOfOperation)
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

func TestBurnOperation(t *testing.T) {
	type decodedOp struct {
		op Std
		id *big.Int
	}

	cases := []testCase{
		{
			caseName: "BurnOperation 1",
			decoded: decodedOp{
				op: StdWRC20,
				id: id,
			},
			encoded: []byte{
				243, 39, 248, 136, 130, 2, 209, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 130, 48, 57, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128,
			},
			errs: []error{},
		},
		{
			caseName: "BurnOperation 2",
			decoded: decodedOp{
				op: StdWRC721,
				id: id,
			},
			encoded: []byte{
				243, 39, 248, 136, 130, 2, 209, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 130, 48, 57, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128,
			},
			errs: []error{},
		},
		{
			caseName: "BurnOperation 3",
			decoded: decodedOp{
				op: StdWRC20,
				id: nil,
			},
			encoded: []byte{
				243, 39, 248, 134, 130, 2, 209, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128,
			},
			errs: []error{ErrNoTokenId},
		},
		{
			caseName: "BurnOperation 4",
			decoded: decodedOp{
				op: 0,
				id: id,
			},
			encoded: []byte{
				243, 39, 248, 136, 130, 2, 209, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 130, 48, 57, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128,
			},
			errs: []error{},
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

		err = equalOpBytes(op, b)
		if err != nil {
			return err
		}

		return nil
	}

	operationDecode := func(b []byte, i interface{}) error {
		op, err := DecodeBytes(b)
		if err != nil {
			return err
		}

		o := i.(decodedOp)
		opDecoded, ok := op.(BurnOperation)
		if !ok {
			return errors.New("invalid operation type")
		}

		err = checkOpCodeAndStandart(b, opDecoded, StdWRC721)
		if err != nil {
			return err
		}

		err = checkBigInt(opDecoded.TokenId(), o.id)
		if err != nil {
			return err
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

	cases := []testCase{
		{
			caseName: "TokenOfOwnerByIndexOperation 1",
			decoded: decodedOp{
				op:      StdWRC721,
				address: address,
				owner:   owner,
				index:   index,
			},
			encoded: []byte{
				243, 40, 248, 136, 130, 2, 209, 148, 208, 73, 191, 214, 103, 203, 70, 170, 62, 245, 223, 13, 163, 229, 125, 179, 190, 57, 229, 17, 128, 148, 25, 119, 194, 72, 225, 1, 76, 193, 3, 146, 157, 215, 241, 84, 25, 156, 145, 110, 57, 236, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 130, 217, 3, 128, 128,
			},
			errs: []error{},
		},
		{
			caseName: "TokenOfOwnerByIndexOperation 2",
			decoded: decodedOp{
				op:      StdWRC721,
				address: common.Address{},
				owner:   owner,
				index:   index,
			},
			encoded: []byte{
				243, 40, 248, 136, 130, 2, 209, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 25, 119, 194, 72, 225, 1, 76, 193, 3, 146, 157, 215, 241, 84, 25, 156, 145, 110, 57, 236, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 130, 217, 3, 128, 128,
			},
			errs: []error{ErrNoAddress},
		},
		{
			caseName: "TokenOfOwnerByIndexOperation 3",
			decoded: decodedOp{
				op:      StdWRC721,
				address: address,
				owner:   common.Address{},
				index:   index,
			},
			encoded: []byte{
				243, 40, 248, 136, 130, 2, 209, 148, 208, 73, 191, 214, 103, 203, 70, 170, 62, 245, 223, 13, 163, 229, 125, 179, 190, 57, 229, 17, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 130, 217, 3, 128, 128,
			},
			errs: []error{ErrNoOwner},
		},
		{
			caseName: "TokenOfOwnerByIndexOperation 4",
			decoded: decodedOp{
				op:      StdWRC721,
				address: address,
				owner:   owner,
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

		err = equalOpBytes(op, b)
		if err != nil {
			return err
		}

		return nil
	}

	operationDecode := func(b []byte, i interface{}) error {
		op, err := DecodeBytes(b)
		if err != nil {
			return err
		}

		o := i.(decodedOp)
		opDecoded, ok := op.(TokenOfOwnerByIndexOperation)
		if !ok {
			return errors.New("invalid operation type")
		}

		err = checkOpCodeAndStandart(b, opDecoded, StdWRC721)
		if err != nil {
			return err
		}

		err = checkBigInt(opDecoded.Index(), o.index)
		if err != nil {
			return err
		}

		if o.owner != opDecoded.Owner() {
			t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", o.owner, opDecoded.Owner())
		}

		if o.address != opDecoded.Address() {
			t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", o.address, opDecoded.Address())
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

	cases := []testCase{
		{
			caseName: "PropertiesOperation 1",
			decoded: decodedOp{
				address: address,
				id:      id,
			},
			encoded: []byte{
				243, 33, 248, 134, 128, 148, 208, 73, 191, 214, 103, 203, 70, 170, 62, 245, 223, 13, 163, 229, 125, 179, 190, 57, 229, 17, 130, 48, 57, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128,
			},
			errs: []error{},
		},
		{
			caseName: "PropertiesOperation 2",
			decoded: decodedOp{
				address: address,
				id:      nil,
			},
			encoded: []byte{
				243, 33, 248, 132, 128, 148, 208, 73, 191, 214, 103, 203, 70, 170, 62, 245, 223, 13, 163, 229, 125, 179, 190, 57, 229, 17, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128,
			},
			errs: []error{ErrNoTokenId},
		},
		{
			caseName: "PropertiesOperation 3",
			decoded: decodedOp{
				address: common.Address{},
				id:      id,
			},
			encoded: []byte{
				243, 33, 248, 134, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 130, 48, 57, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128,
			},
			errs: []error{ErrNoAddress},
		},
		{
			caseName: "PropertiesOperation 4",
			decoded: decodedOp{
				address: common.Address{},
				id:      nil,
			},
			encoded: []byte{
				243, 33, 248, 132, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128,
			},
			errs: []error{ErrNoAddress, ErrNoTokenId},
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

		err = equalOpBytes(op, b)
		if err != nil {
			return err
		}

		return nil
	}

	operationDecode := func(b []byte, i interface{}) error {
		op, err := DecodeBytes(b)
		if err != nil {
			return err
		}

		o := i.(decodedOp)
		opDecoded, ok := op.(PropertiesOperation)
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

		err = checkBigInt(tokenId, o.id)
		if err != nil {
			return err
		}

		if o.address != opDecoded.Address() {
			t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", o.address, opDecoded.Address())
		}

		return nil
	}

	startSubTests(t, cases, operationEncode, operationDecode)
}

func TestCreateOperationOperation(t *testing.T) {
	type decodedOp struct {
		op          Std
		name        []byte
		symbol      []byte
		decimals    uint8
		totalSupply *big.Int
		baseURI     []byte
	}

	cases := []testCase{
		{
			caseName: "CreateOperation 1",
			decoded: decodedOp{
				op:          StdWRC20,
				name:        name,
				symbol:      symbol,
				decimals:    decimals,
				totalSupply: totalSupply,
				baseURI:     nil,
			},
			encoded: []byte{
				243, 12, 212, 20, 139, 84, 101, 115, 116, 32, 84, 111, 107, 107, 101, 110, 130, 84, 84, 130, 39, 15, 100,
			},
			errs: []error{ErrNoBaseURI},
		},
		{
			caseName: "CreateOperation 2",
			decoded: decodedOp{
				op:          StdWRC721,
				name:        name,
				symbol:      symbol,
				decimals:    decimals,
				totalSupply: totalSupply,
				baseURI:     baseURI,
			},
			encoded: []byte{
				243, 12, 227, 130, 2, 209, 139, 84, 101, 115, 116, 32, 84, 111, 107, 107, 101, 110, 130, 84, 84, 128, 128, 142, 116, 101, 115, 116, 46, 116, 111, 107, 101, 110, 46, 99, 111, 109,
			},
			errs: []error{ErrNoTokenSupply},
		},
		{
			caseName: "CreateOperation 3",
			decoded: decodedOp{
				op:          0,
				name:        name,
				symbol:      nil,
				decimals:    decimals,
				totalSupply: nil,
				baseURI:     baseURI,
			},
			encoded: []byte{
				243, 12, 223, 128, 139, 84, 101, 115, 116, 32, 84, 111, 107, 107, 101, 110, 128, 128, 100, 142, 116, 101, 115, 116, 46, 116, 111, 107, 101, 110, 46, 99, 111, 109,
			},
			errs: []error{ErrNoTokenSupply, ErrNoSymbol},
		},
		{
			caseName: "CreateOperation 4",
			decoded: decodedOp{
				op:          0,
				name:        name,
				symbol:      symbol,
				decimals:    0,
				totalSupply: totalSupply,
				baseURI:     nil,
			},
			encoded: []byte{
				243, 12, 214, 130, 2, 209, 139, 84, 101, 115, 116, 32, 84, 111, 107, 107, 101, 110, 130, 84, 84, 130, 39, 15, 128,
			},
			errs: []error{ErrNoTokenSupply},
		},
	}

	operationEncode := func(b []byte, i interface{}) error {
		o := i.(decodedOp)
		switch o.op {
		case StdWRC20:
			createOp, err := NewWrc20CreateOperation(
				o.name,
				o.symbol,
				&o.decimals,
				o.totalSupply,
			)
			if err != nil {
				return err
			}

			err = equalOpBytes(createOp, b)
			if err != nil {
				return err
			}
		case StdWRC721:
			createOp, err := NewWrc721CreateOperation(
				o.name,
				o.symbol,
				o.baseURI,
			)
			if err != nil {
				return err
			}

			err = equalOpBytes(createOp, b)
			if err != nil {
				return err
			}
		}

		return nil
	}

	operationDecode := func(b []byte, i interface{}) error {
		op, err := DecodeBytes(b)
		if err != nil {
			return err
		}

		o := i.(decodedOp)
		opDecoded, ok := op.(CreateOperation)
		if !ok {
			return errors.New("invalid operation type")
		}

		err = checkOpCodeAndStandart(b, opDecoded, o.op)
		if err != nil {
			return err
		}

		tS, ok := opDecoded.TotalSupply()
		if !ok {
			return ErrNoTokenSupply
		}

		err = checkBigInt(tS, o.totalSupply)
		if err != nil {
			return err
		}

		if opDecoded.Decimals() != o.decimals {
			return fmt.Errorf("values do not match:\nwant: %+v\nhave: %+v", o.decimals, opDecoded.Decimals())
		}

		if !bytes.Equal(opDecoded.Name(), o.name) {
			return fmt.Errorf("values do not match:\nwant: %+v\nhave: %+v", o.name, opDecoded.Name())
		}

		if !bytes.Equal(opDecoded.Symbol(), o.symbol) {
			return fmt.Errorf("values do not match:\nwant: %+v\nhave: %+v", o.symbol, opDecoded.Symbol())
		}

		uri, ok := opDecoded.BaseURI()
		if !ok {
			return ErrNoBaseURI
		}
		if !bytes.Equal(uri, o.baseURI) {
			return fmt.Errorf("values do not match:\nwant: %+v\nhave: %+v", o.baseURI, uri)
		}

		return nil
	}

	startSubTests(t, cases, operationEncode, operationDecode)
}

func startSubTests(t *testing.T, cases []testCase, operationEncode, operationDecode func([]byte, interface{}) error) {
	for _, c := range cases {
		t.Run("encoding", func(t *testing.T) {
			err := operationEncode(c.encoded, c.decoded)
			if err != nil {
				if err = findNotCorrectError(err, c.errs); err != nil {
					t.Fatalf("operationEncode: invalid test case %s\nwant errors: %s\nhave errors: %s", c.caseName, c.errs, err)
				}
			}
		})

		t.Run("decoding", func(t *testing.T) {
			err := operationDecode(c.encoded, c.decoded)
			if err != nil {
				if err = findNotCorrectError(err, c.errs); err != nil {
					t.Fatalf("operationDecode: invalid test case %s\nwant errors: %s\nhave errors: %s", c.caseName, c.errs, err)
				}
			}
		})
	}
}

func checkOpCodeAndStandart(b []byte, op Operation, std Std) error {
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

func checkBigInt(a, b *big.Int) error {
	haveValue := a
	wantValue := b

	if wantValue == nil {
		wantValue = big.NewInt(0)
	}

	if haveValue.Cmp(wantValue) != 0 {
		return fmt.Errorf("values do not match:\nwant: %+v\nhave: %+v", wantValue, haveValue)
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

func findNotCorrectError(e error, arr []error) error {
	for _, err := range arr {
		if e == err {
			return nil
		}
	}

	return e
}
