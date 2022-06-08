package token

import (
	"bytes"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"log"
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

func TestMintOperation(t *testing.T) {
	type decodedOp struct {
		op   Std
		to   common.Address
		id   *big.Int
		data []byte
	}

	type op struct {
		caseName string
		decoded  decodedOp
		encoded  []byte
	}

	operations := []op{
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
		},
	}

	operationEncode := func(o decodedOp, b []byte) error {
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

	operationDecode := func(b []byte, o decodedOp) error {
		op, err := DecodeBytes(b)
		if err != nil {
			return err
		}

		opDecoded, ok := op.(MintOperation)
		if !ok {
			return fmt.Errorf("invalid operation type")
		}

		if opDecoded.Standard() != StdWRC721 {
			return fmt.Errorf("values do not match:\nwant: %+v\nhave: %+v", o.op, opDecoded.Standard())
		}

		err = checkOpCode(b, opDecoded)
		if err != nil {
			return err
		}

		if opDecoded.To() != o.to {
			return fmt.Errorf("values do not match:\nwant: %+v\nhave: %+v", opDecoded.To(), o.to)
		}

		haveData, ok := opDecoded.Metadata()
		if !ok {
			log.Printf("have data is empty")
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

	for _, op := range operations {
		t.Run("encoding", func(t *testing.T) {
			err := operationEncode(op.decoded, op.encoded)
			if err != nil {
				switch err {
				case ErrNoTo, ErrNoTokenId:
					t.Log(err)
				default:
					t.Logf("operationEncode: invalid test case %s", op.caseName)
					t.Fatal(err)
				}
			}
		})

		t.Run("decoding", func(t *testing.T) {
			err := operationDecode(op.encoded, op.decoded)
			if err != nil {
				switch err {
				case ErrNoTo, ErrNoTokenId, ErrOpNotValid:
					t.Log(err)
				default:
					t.Logf("operationDecode: invalid test case %s", op.caseName)
					t.Fatal(err)
				}
			}
		})
	}
}

func TestSetApprovalForAllOperation(t *testing.T) {
	type decodedOp struct {
		op       Std
		operator common.Address
		approve  bool
	}

	type op struct {
		caseName string
		decoded  decodedOp
		encoded  []byte
	}

	operations := []op{
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
		},
		{
			caseName: "setApprovalForAllOperation 2",
			decoded: decodedOp{
				op:       StdWRC20,
				operator: common.Address{},
				approve:  true,
			},
			encoded: []byte{
				243, 37, 248, 134, 130, 2, 209, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 1, 128},
		},
		{
			caseName: "setApprovalForAllOperation 3",
			decoded: decodedOp{

				op:       StdWRC721,
				operator: operator,
				approve:  false,
			},
			encoded: []byte{
				243, 37, 248, 134, 130, 2, 209, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 19, 228, 172, 239, 230, 166, 112, 6, 4, 146, 153, 70, 231, 14, 100, 67, 228, 231, 52, 71, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128},
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
		},
	}

	operationEncode := func(o decodedOp, b []byte) error {
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

	operationDecode := func(b []byte, o decodedOp) error {
		op, err := DecodeBytes(b)
		if err != nil {
			return err
		}

		opDecoded, ok := op.(SetApprovalForAllOperation)
		if !ok {
			return fmt.Errorf("invalid operation type")
		}

		if opDecoded.Standard() != StdWRC721 {
			return fmt.Errorf("values do not match:\nwant: %+v\nhave: %+v", o.op, opDecoded.Standard())
		}

		err = checkOpCode(b, opDecoded)
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

	for _, op := range operations {
		t.Run("encoding", func(t *testing.T) {
			err := operationEncode(op.decoded, op.encoded)
			if err != nil {
				switch err {
				case ErrNoOperator:
					t.Log(err)
				default:
					t.Logf("operationEncode: invalid test case %s", op.caseName)
					t.Fatal(err)
				}
			}
		})

		t.Run("decoding", func(t *testing.T) {
			err := operationDecode(op.encoded, op.decoded)
			if err != nil {
				switch err {
				case ErrNoOperator, ErrOpNotValid:
					t.Log(err)
				default:
					t.Logf("operationDecode: invalid test case %s", op.caseName)
					t.Fatal(err)
				}
			}
		})
	}
}

func TestIsApprovedForAllOperation(t *testing.T) {
	type decodedOp struct {
		op       Std
		address  common.Address
		owner    common.Address
		operator common.Address
	}

	type op struct {
		caseName string
		decoded  decodedOp
		encoded  []byte
	}

	operations := []op{
		{
			caseName: "isApprovedForAllOperation 1",
			decoded: decodedOp{
				op:       StdWRC721,
				address:  address,
				owner:    owner,
				operator: operator,
			},
			encoded: []byte{
				243, 36, 248, 134, 130, 2, 209, 148, 208, 73, 191, 214, 103, 203, 70, 170, 62, 245, 223, 13, 163, 229, 125, 179, 190, 57, 229, 17, 128, 148, 25, 119, 194, 72, 225, 1, 76, 193, 3, 146, 157, 215, 241, 84, 25, 156, 145, 110, 57, 236, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 19, 228, 172, 239, 230, 166, 112, 6, 4, 146, 153, 70, 231, 14, 100, 67, 228, 231, 52, 71, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128},
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
				243, 36, 248, 132, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 25, 119, 194, 72, 225, 1, 76, 193, 3, 146, 157, 215, 241, 84, 25, 156, 145, 110, 57, 236, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 19, 228, 172, 239, 230, 166, 112, 6, 4, 146, 153, 70, 231, 14, 100, 67, 228, 231, 52, 71, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128},
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
		},
	}

	operationEncode := func(o decodedOp, b []byte) error {
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

	operationDecode := func(b []byte, o decodedOp) error {
		op, err := DecodeBytes(b)
		if err != nil {
			return err
		}

		opDecoded, ok := op.(IsApprovedForAllOperation)
		if !ok {
			return fmt.Errorf("invalid operation type")
		}

		if opDecoded.Standard() != StdWRC721 {
			return fmt.Errorf("values do not match:\nwant: %+v\nhave: %+v", StdWRC721, opDecoded.Standard())
		}

		err = checkOpCode(b, opDecoded)
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

	for _, op := range operations {
		t.Run("encoding", func(t *testing.T) {
			err := operationEncode(op.decoded, op.encoded)
			if err != nil {
				switch err {
				case ErrNoOperator, ErrNoOwner, ErrNoAddress:
					t.Log(err)
				default:
					t.Logf("operationEncode: invalid test case %s", op.caseName)
					t.Fatal(err)
				}
			}
		})

		t.Run("decoding", func(t *testing.T) {
			err := operationDecode(op.encoded, op.decoded)
			if err != nil {
				switch err {
				case ErrNoOperator, ErrNoOwner, ErrNoAddress, ErrOpNotValid:
					t.Log(err)
				default:
					t.Logf("operationDecode: invalid test case %s", op.caseName)
					t.Fatal(err)
				}
			}
		})
	}
}

func TestAllowanceOperation(t *testing.T) {
	type decodedOp struct {
		op      Std
		address common.Address
		owner   common.Address
		spender common.Address
	}

	type op struct {
		caseName string
		decoded  decodedOp
		encoded  []byte
	}

	operations := []op{
		{
			caseName: "AllowanceOperation 1",
			decoded: decodedOp{
				op:      StdWRC721,
				address: address,
				owner:   owner,
				spender: spender,
			},
			encoded: []byte{
				243, 35, 248, 132, 20, 148, 208, 73, 191, 214, 103, 203, 70, 170, 62, 245, 223, 13, 163, 229, 125, 179, 190, 57, 229, 17, 128, 148, 25, 119, 194, 72, 225, 1, 76, 193, 3, 146, 157, 215, 241, 84, 25, 156, 145, 110, 57, 236, 148, 44, 204, 245, 224, 83, 132, 147, 194, 53, 209, 197, 239, 101, 128, 247, 125, 153, 233, 19, 150, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128},
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
				243, 35, 248, 132, 128, 148, 208, 73, 191, 214, 103, 203, 70, 170, 62, 245, 223, 13, 163, 229, 125, 179, 190, 57, 229, 17, 128, 148, 25, 119, 194, 72, 225, 1, 76, 193, 3, 146, 157, 215, 241, 84, 25, 156, 145, 110, 57, 236, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128},
		},
	}

	operationEncode := func(o decodedOp, b []byte) error {
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

	operationDecode := func(b []byte, o decodedOp) error {
		op, err := DecodeBytes(b)
		if err != nil {
			return err
		}

		opDecoded, ok := op.(AllowanceOperation)
		if !ok {
			return fmt.Errorf("invalid operation type")
		}

		if opDecoded.Standard() != StdWRC20 {
			return fmt.Errorf("values do not match:\nwant: %+v\nhave: %+v", StdWRC20, opDecoded.Standard())
		}

		err = checkOpCode(b, opDecoded)
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

	for _, op := range operations {
		t.Run("encoding", func(t *testing.T) {
			err := operationEncode(op.decoded, op.encoded)
			if err != nil {
				switch err {
				case ErrNoSpender, ErrNoOwner, ErrNoAddress, ErrOpNotValid:
					t.Log(err)
				default:
					t.Logf("operationEncode: invalid test case %s", op.caseName)
					t.Fatal(err)
				}
			}
		})

		t.Run("decoding", func(t *testing.T) {
			err := operationDecode(op.encoded, op.decoded)
			if err != nil {
				switch err {
				case ErrNoSpender, ErrNoOwner, ErrNoAddress:
					t.Log(err)
				default:
					t.Logf("operationDecode: invalid test case %s", op.caseName)
					t.Fatal(err)
				}
			}
		})
	}
}

func TestApproveOperation(t *testing.T) {
	type decodedOp struct {
		op      Std
		spender common.Address
		value   *big.Int
	}

	type op struct {
		caseName string
		decoded  decodedOp
		encoded  []byte
	}

	operations := []op{
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
		},
	}

	operationEncode := func(o decodedOp, b []byte) error {
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

	operationDecode := func(b []byte, o decodedOp) error {
		op, err := DecodeBytes(b)
		if err != nil {
			return err
		}

		opDecoded, ok := op.(ApproveOperation)
		if !ok {
			return fmt.Errorf("invalid operation type")
		}

		if opDecoded.Standard() != o.op {
			return fmt.Errorf("values do not match:\nwant: %+v\nhave: %+v", o.op, opDecoded.Standard())
		}

		err = checkOpCode(b, opDecoded)
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

	for _, op := range operations {
		t.Run("encoding", func(t *testing.T) {
			err := operationEncode(op.decoded, op.encoded)
			if err != nil {
				switch err {
				case ErrNoSpender, ErrNoValue:
					t.Log(err)
				default:
					t.Logf("operationEncode: invalid test case %s", op.caseName)
					t.Fatal(err)
				}
			}
		})

		t.Run("decoding", func(t *testing.T) {
			err := operationDecode(op.encoded, op.decoded)
			if err != nil {
				switch err {
				case ErrNoSpender, ErrNoValue, ErrOpNotValid:
					t.Log(err)
				default:
					t.Logf("operationDecode: invalid test case %s", op.caseName)
					t.Fatal(err)
				}
			}
		})
	}
}

func TestSafeTransferFromOperation(t *testing.T) {
	type decodedOp struct {
		transferFromOp transferFromOperation
		data           []byte
	}

	type op struct {
		caseName string
		decoded  decodedOp
		encoded  []byte
	}

	operations := []op{
		{
			caseName: "SafeTransferFromOperation 1",
			decoded: decodedOp{
				transferFromOp: transferFromOperation{
					transferOperation: transferOperation{
						operation:      operation{StdWRC721},
						valueOperation: valueOperation{value},
						toOperation:    toOperation{to},
					},
					FromAddress: from,
				},
				data: data,
			},
			encoded: []byte{
				243, 41, 248, 150, 130, 2, 209, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 121, 134, 186, 216, 31, 76, 189, 147, 23, 245, 164, 104, 97, 67, 125, 174, 88, 214, 145, 19, 148, 125, 201, 201, 115, 6, 137, 255, 11, 15, 213, 6, 198, 125, 184, 21, 241, 45, 144, 164, 72, 131, 1, 182, 105, 128, 128, 141, 243, 12, 202, 20, 133, 116, 111, 107, 101, 110, 116, 100, 5,
			},
		},
		{
			caseName: "SafeTransferFromOperation 2",
			decoded: decodedOp{
				transferFromOp: transferFromOperation{
					transferOperation: transferOperation{
						operation:      operation{StdWRC721},
						valueOperation: valueOperation{},
						toOperation:    toOperation{to},
					},
					FromAddress: from,
				},
				data: nil,
			},
			encoded: []byte{
				243, 41, 248, 134, 130, 2, 209, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 121, 134, 186, 216, 31, 76, 189, 147, 23, 245, 164, 104, 97, 67, 125, 174, 88, 214, 145, 19, 148, 125, 201, 201, 115, 6, 137, 255, 11, 15, 213, 6, 198, 125, 184, 21, 241, 45, 144, 164, 72, 128, 128, 128, 128,
			},
		},
		{
			caseName: "SafeTransferFromOperation 3",
			decoded: decodedOp{
				transferFromOp: transferFromOperation{
					transferOperation: transferOperation{
						operation:      operation{},
						valueOperation: valueOperation{value},
						toOperation:    toOperation{to},
					},
					FromAddress: common.Address{},
				},
				data: nil,
			},
			encoded: []byte{
				243, 41, 248, 135, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 125, 201, 201, 115, 6, 137, 255, 11, 15, 213, 6, 198, 125, 184, 21, 241, 45, 144, 164, 72, 131, 1, 182, 105, 128, 128, 128,
			},
		},
		{
			caseName: "SafeTransferFromOperation 4",
			decoded: decodedOp{
				transferFromOp: transferFromOperation{
					transferOperation: transferOperation{
						operation:      operation{StdWRC721},
						valueOperation: valueOperation{},
						toOperation:    toOperation{},
					},
					FromAddress: from,
				},
				data: nil,
			},
			encoded: []byte{
				243, 41, 248, 134, 130, 2, 209, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 121, 134, 186, 216, 31, 76, 189, 147, 23, 245, 164, 104, 97, 67, 125, 174, 88, 214, 145, 19, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128,
			},
		},
	}

	operationEncode := func(o decodedOp, b []byte) error {
		op, err := NewSafeTransferFromOperation(
			o.transferFromOp.FromAddress,
			o.transferFromOp.toOperation.ToAddress,
			o.transferFromOp.valueOperation.TokenValue,
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

	operationDecode := func(b []byte, o decodedOp) error {
		op, err := DecodeBytes(b)
		if err != nil {
			return err
		}

		opDecoded, ok := op.(SafeTransferFromOperation)
		if !ok {
			return fmt.Errorf("invalid operation type")
		}

		if opDecoded.Standard() != StdWRC721 {
			return fmt.Errorf("values do not match:\nwant: %+v\nhave: %+v", StdWRC721, opDecoded.Standard())
		}

		err = checkOpCode(b, opDecoded)
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

		err = checkBigInt(opDecoded.Value(), o.transferFromOp.valueOperation.TokenValue)
		if err != nil {
			return err
		}

		return nil
	}

	for _, op := range operations {
		t.Run("encoding", func(t *testing.T) {
			err := operationEncode(op.decoded, op.encoded)
			if err != nil {
				switch err {
				case ErrNoFrom, ErrNoTo, ErrNoValue:
					t.Log(err)
				default:
					t.Logf("operationEncode: invalid test case %s", op.caseName)
					t.Fatal(err)
				}
			}
		})

		t.Run("decoding", func(t *testing.T) {
			err := operationDecode(op.encoded, op.decoded)
			if err != nil {
				switch err {
				case ErrNoFrom, ErrNoTo, ErrNoValue, ErrOpNotValid:
					t.Log(err)
				default:
					t.Logf("operationDecode: invalid test case %s", op.caseName)
					t.Fatal(err)
				}
			}
		})
	}
}

func TestTransferFromOperation(t *testing.T) {
	type decodedOp struct {
		transferOp transferOperation
		from       common.Address
	}

	type op struct {
		caseName string
		decoded  decodedOp
		encoded  []byte
	}

	operations := []op{
		{
			caseName: "TransferFromOperation 1",
			decoded: decodedOp{
				transferOp: transferOperation{
					operation:      operation{StdWRC20},
					valueOperation: valueOperation{value},
					toOperation:    toOperation{to},
				},
				from: from,
			},
			encoded: []byte{
				243, 31, 248, 135, 20, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 121, 134, 186, 216, 31, 76, 189, 147, 23, 245, 164, 104, 97, 67, 125, 174, 88, 214, 145, 19, 148, 125, 201, 201, 115, 6, 137, 255, 11, 15, 213, 6, 198, 125, 184, 21, 241, 45, 144, 164, 72, 131, 1, 182, 105, 128, 128, 128,
			},
		},
		{
			caseName: "TransferFromOperation 2",
			decoded: decodedOp{
				transferOp: transferOperation{
					operation:      operation{StdWRC721},
					valueOperation: valueOperation{id},
					toOperation:    toOperation{to},
				},
				from: from,
			},
			encoded: []byte{
				243, 31, 248, 136, 130, 2, 209, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 121, 134, 186, 216, 31, 76, 189, 147, 23, 245, 164, 104, 97, 67, 125, 174, 88, 214, 145, 19, 148, 125, 201, 201, 115, 6, 137, 255, 11, 15, 213, 6, 198, 125, 184, 21, 241, 45, 144, 164, 72, 130, 48, 57, 128, 128, 128,
			},
		},
		{
			caseName: "TransferFromOperation 3",
			decoded: decodedOp{
				transferOp: transferOperation{
					operation:      operation{StdWRC721},
					valueOperation: valueOperation{id},
					toOperation:    toOperation{to},
				},
				from: common.Address{},
			},
			encoded: []byte{
				243, 31, 248, 136, 130, 2, 209, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 125, 201, 201, 115, 6, 137, 255, 11, 15, 213, 6, 198, 125, 184, 21, 241, 45, 144, 164, 72, 130, 48, 57, 128, 128, 128,
			},
		},
		{
			caseName: "TransferFromOperation 4",
			decoded: decodedOp{
				transferOp: transferOperation{
					operation:      operation{StdWRC20},
					valueOperation: valueOperation{value},
					toOperation:    toOperation{},
				},
				from: from,
			},
			encoded: []byte{
				243, 31, 248, 135, 20, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 121, 134, 186, 216, 31, 76, 189, 147, 23, 245, 164, 104, 97, 67, 125, 174, 88, 214, 145, 19, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 131, 1, 182, 105, 128, 128, 128,
			},
		},
	}

	operationEncode := func(o decodedOp, b []byte) error {
		op, err := NewTransferFromOperation(
			o.transferOp.operation.Std,
			o.from,
			o.transferOp.toOperation.ToAddress,
			o.transferOp.valueOperation.TokenValue,
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

	operationDecode := func(b []byte, o decodedOp) error {
		op, err := DecodeBytes(b)
		if err != nil {
			return err
		}

		opDecoded, ok := op.(TransferFromOperation)
		if !ok {
			return fmt.Errorf("invalid operation type")
		}

		if opDecoded.Standard() != o.transferOp.operation.Std {
			return fmt.Errorf("values do not match:\nwant: %+v\nhave: %+v", o.transferOp.operation.Std, opDecoded.Standard())
		}

		err = checkOpCode(b, opDecoded)
		if err != nil {
			return err
		}

		err = checkBigInt(opDecoded.Value(), o.transferOp.valueOperation.TokenValue)
		if err != nil {
			return err
		}

		if o.from != opDecoded.From() {
			t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", opDecoded.From(), from)
		}

		if o.transferOp.toOperation.ToAddress != opDecoded.To() {
			t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", opDecoded.To(), o.transferOp.toOperation.ToAddress)
		}

		return nil
	}

	for _, op := range operations {
		t.Run("encoding", func(t *testing.T) {
			err := operationEncode(op.decoded, op.encoded)
			if err != nil {
				switch err {
				case ErrNoFrom, ErrNoTo, ErrNoValue:
					t.Log(err)
				default:
					t.Logf("operationEncode: invalid test case %s", op.caseName)
					t.Fatal(err)
				}
			}
		})

		t.Run("decoding", func(t *testing.T) {
			err := operationDecode(op.encoded, op.decoded)
			if err != nil {
				switch err {
				case ErrNoFrom, ErrNoTo, ErrNoValue, ErrOpNotValid:
					t.Log(err)
				default:
					t.Logf("operationDecode: invalid test case %s", op.caseName)
					t.Fatal(err)
				}
			}
		})
	}
}

func TestTransferOperation(t *testing.T) {
	type decodedOp struct {
		op    Std
		value *big.Int
		to    common.Address
	}

	type op struct {
		caseName string
		decoded  decodedOp
		encoded  []byte
	}

	operations := []op{
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
		},
	}

	for _, o := range operations {
		m := transferOperation{
			operation:      operation{o.decoded.op},
			valueOperation: valueOperation{o.decoded.value},
			toOperation:    toOperation{o.decoded.to},
		}

		t, _ := EncodeToBytes(&m)
		log.Printf("Case %s\nBytes %+v", o.caseName, t)
	}

	operationEncode := func(o decodedOp, b []byte) error {
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

	operationDecode := func(b []byte, o decodedOp) error {
		op, err := DecodeBytes(b)
		if err != nil {
			return err
		}

		opDecoded, ok := op.(TransferOperation)
		if !ok {
			return fmt.Errorf("invalid operation type")
		}

		if opDecoded.Standard() != o.op {
			return fmt.Errorf("values do not match:\nwant: %+v\nhave: %+v", o.op, opDecoded.Standard())
		}

		err = checkOpCode(b, opDecoded)
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

	for _, op := range operations {
		t.Run("encoding", func(t *testing.T) {
			err := operationEncode(op.decoded, op.encoded)
			if err != nil {
				switch err {
				case ErrNoTo, ErrNoValue:
					t.Log(err)
				default:
					t.Logf("operationEncode: invalid test case %s", op.caseName)
					t.Fatal(err)
				}
			}
		})

		t.Run("decoding", func(t *testing.T) {
			err := operationDecode(op.encoded, op.decoded)
			if err != nil {
				switch err {
				case ErrNoTo, ErrNoValue, ErrOpNotValid:
					t.Log(err)
				default:
					t.Logf("operationDecode: invalid test case %s", op.caseName)
					t.Fatal(err)
				}
			}
		})
	}
}

func TestBalanceOfOperation(t *testing.T) {
	type decodedOp struct {
		op      Std
		address common.Address
		owner   common.Address
	}

	type op struct {
		caseName string
		decoded  decodedOp
		encoded  []byte
	}

	operations := []op{
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
		},
	}

	operationEncode := func(o decodedOp, b []byte) error {
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

	operationDecode := func(b []byte, o decodedOp) error {
		op, err := DecodeBytes(b)
		if err != nil {
			return err
		}

		opDecoded, ok := op.(BalanceOfOperation)
		if !ok {
			return fmt.Errorf("invalid operation type")
		}

		if opDecoded.Standard() != 0 {
			return fmt.Errorf("values do not match:\nwant: %+v\nhave: %+v", 0, opDecoded.Standard())
		}

		err = checkOpCode(b, opDecoded)
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

	for _, op := range operations {
		t.Run("encoding", func(t *testing.T) {
			err := operationEncode(op.decoded, op.encoded)
			if err != nil {
				switch err {
				case ErrNoAddress, ErrNoOwner:
					t.Log(err)
				default:
					t.Logf("operationEncode: invalid test case %s", op.caseName)
					t.Fatal(err)
				}
			}
		})

		t.Run("decoding", func(t *testing.T) {
			err := operationDecode(op.encoded, op.decoded)
			if err != nil {
				switch err {
				case ErrNoAddress, ErrNoOwner, ErrOpNotValid:
					t.Log(err)
				default:
					t.Logf("operationDecode: invalid test case %s", op.caseName)
					t.Fatal(err)
				}
			}
		})
	}
}

func TestBurnOperation(t *testing.T) {
	type decodedOp struct {
		op Std
		id *big.Int
	}

	type op struct {
		caseName string
		decoded  decodedOp
		encoded  []byte
	}

	operations := []op{
		{
			caseName: "BurnOperation 1",
			decoded: decodedOp{
				op: StdWRC20,
				id: id,
			},
			encoded: []byte{
				243, 39, 248, 136, 130, 2, 209, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 130, 48, 57, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128,
			},
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
		},
		{
			caseName: "BurnOperation 4",
			decoded: decodedOp{
				op: 0,
				id: id,
			},
			encoded: []byte{
				243, 39, 248, 136, 130, 2, 209, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 130, 48, 57, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128},
		},
	}

	operationEncode := func(o decodedOp, b []byte) error {
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

	operationDecode := func(b []byte, o decodedOp) error {
		op, err := DecodeBytes(b)
		if err != nil {
			return err
		}

		opDecoded, ok := op.(BurnOperation)
		if !ok {
			return fmt.Errorf("invalid operation type")
		}

		if opDecoded.Standard() != StdWRC721 {
			return fmt.Errorf("values do not match:\nwant: %+v\nhave: %+v", StdWRC721, opDecoded.Standard())
		}

		err = checkOpCode(b, opDecoded)
		if err != nil {
			return err
		}

		err = checkBigInt(opDecoded.TokenId(), o.id)
		if err != nil {
			return err
		}

		return nil
	}

	for _, op := range operations {
		t.Run("encoding", func(t *testing.T) {
			err := operationEncode(op.decoded, op.encoded)
			if err != nil {
				switch err {
				case ErrNoTokenId:
					t.Log(err)
				default:
					t.Logf("operationEncode: invalid test case %s", op.caseName)
					t.Fatal(err)
				}
			}
		})

		t.Run("decoding", func(t *testing.T) {
			err := operationDecode(op.encoded, op.decoded)
			if err != nil {
				switch err {
				case ErrNoTokenId, ErrOpNotValid:
					t.Log(err)
				default:
					t.Logf("operationDecode: invalid test case %s", op.caseName)
					t.Fatal(err)
				}
			}
		})
	}
}

func TestTokenOfOwnerByIndexOperation(t *testing.T) {
	type decodedOp struct {
		op      Std
		address common.Address
		owner   common.Address
		index   *big.Int
	}

	type op struct {
		caseName string
		decoded  decodedOp
		encoded  []byte
	}

	operations := []op{
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
		},
	}

	operationEncode := func(o decodedOp, b []byte) error {
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

	operationDecode := func(b []byte, o decodedOp) error {
		op, err := DecodeBytes(b)
		if err != nil {
			return err
		}

		opDecoded, ok := op.(TokenOfOwnerByIndexOperation)
		if !ok {
			return fmt.Errorf("invalid operation type")
		}

		if opDecoded.Standard() != StdWRC721 {
			return fmt.Errorf("values do not match:\nwant: %+v\nhave: %+v", StdWRC721, opDecoded.Standard())
		}

		err = checkOpCode(b, opDecoded)
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

	for _, op := range operations {
		t.Run("encoding", func(t *testing.T) {
			err := operationEncode(op.decoded, op.encoded)
			if err != nil {
				switch err {
				case ErrNoIndex, ErrNoOwner, ErrNoAddress:
					t.Log(err)
				default:
					t.Logf("operationEncode: invalid test case %s", op.caseName)
					t.Fatal(err)
				}
			}
		})

		t.Run("decoding", func(t *testing.T) {
			err := operationDecode(op.encoded, op.decoded)
			if err != nil {
				switch err {
				case ErrNoIndex, ErrNoOwner, ErrNoAddress, ErrOpNotValid:
					t.Log(err)
				default:
					t.Logf("operationDecode: invalid test case %s", op.caseName)
					t.Fatal(err)
				}
			}
		})
	}
}

func TestPropertiesOperation(t *testing.T) {
	type decodedOp struct {
		address common.Address
		id      *big.Int
	}

	type op struct {
		caseName string
		decoded  decodedOp
		encoded  []byte
	}

	operations := []op{
		{
			caseName: "PropertiesOperation 1",
			decoded: decodedOp{
				address: address,
				id:      id,
			},
			encoded: []byte{
				243, 33, 248, 134, 128, 148, 208, 73, 191, 214, 103, 203, 70, 170, 62, 245, 223, 13, 163, 229, 125, 179, 190, 57, 229, 17, 130, 48, 57, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128,
			},
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
		},
	}

	for _, o := range operations {
		m := propertiesOperation{
			addressOperation: addressOperation{o.decoded.address},
			Id:               o.decoded.id,
		}

		t, _ := EncodeToBytes(&m)
		log.Printf("Case %s\nBytes %+v", o.caseName, t)
	}

	operationEncode := func(o decodedOp, b []byte) error {
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

	operationDecode := func(b []byte, o decodedOp) error {
		op, err := DecodeBytes(b)
		if err != nil {
			return err
		}

		opDecoded, ok := op.(PropertiesOperation)
		if !ok {
			return fmt.Errorf("invalid operation type")
		}

		if opDecoded.Standard() != 0 {
			return fmt.Errorf("values do not match:\nwant: %+v\nhave: %+v", 0, opDecoded.Standard())
		}

		err = checkOpCode(b, opDecoded)
		if err != nil {
			return err
		}

		tokenId, ok := opDecoded.TokenId()
		if !ok {
			return fmt.Errorf("invalid tokenId")
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

	for _, op := range operations {
		t.Run("encoding", func(t *testing.T) {
			err := operationEncode(op.decoded, op.encoded)
			if err != nil {
				switch err {
				case ErrNoAddress:
					t.Log(err)
				default:
					t.Logf("operationEncode: invalid test case %s", op.caseName)
					t.Fatal(err)
				}
			}
		})

		t.Run("decoding", func(t *testing.T) {
			err := operationDecode(op.encoded, op.decoded)
			if err != nil {
				switch err {
				case ErrNoAddress, ErrOpNotValid:
					t.Log(err)
				default:
					t.Logf("operationDecode: invalid test case %s", op.caseName)
					t.Fatal(err)
				}
			}
		})
	}
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

	type op struct {
		caseName string
		decoded  decodedOp
		encoded  []byte
	}

	operations := []op{
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
		},
	}

	for _, o := range operations {
		m := createOperation{
			operation:   operation{o.decoded.op},
			name:        o.decoded.name,
			symbol:      o.decoded.symbol,
			decimals:    o.decoded.decimals,
			totalSupply: o.decoded.totalSupply,
			baseURI:     o.decoded.baseURI,
		}

		t, _ := EncodeToBytes(&m)
		log.Printf("Case %s\nBytes %+v", o.caseName, t)
	}

	operationEncode := func(o decodedOp, b []byte) error {
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

	operationDecode := func(b []byte, o decodedOp) error {
		op, err := DecodeBytes(b)
		if err != nil {
			return err
		}

		opDecoded, ok := op.(CreateOperation)
		if !ok {
			return fmt.Errorf("invalid operation type")
		}

		if opDecoded.Standard() != o.op {
			return fmt.Errorf("values do not match:\nwant: %+v\nhave: %+v", o.op, opDecoded.Standard())
		}

		err = checkOpCode(b, opDecoded)
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

	for _, op := range operations {
		t.Run("encoding", func(t *testing.T) {
			err := operationEncode(op.decoded, op.encoded)
			if err != nil {
				switch err {
				case ErrNoName, ErrNoSymbol, ErrNoTokenSupply, ErrNoBaseURI:
					t.Log(err)
				default:
					t.Logf("operationEncode: invalid test case %s", op.caseName)
					t.Fatal(err)
				}
			}
		})

		t.Run("decoding", func(t *testing.T) {
			err := operationDecode(op.encoded, op.decoded)
			if err != nil {
				switch err {
				case ErrNoName, ErrNoSymbol, ErrNoTokenSupply, ErrNoBaseURI, ErrOpNotValid:
					t.Log(err)
				default:
					t.Logf("operationDecode: invalid test case %s", op.caseName)
					t.Fatal(err)
				}
			}
		})
	}
}

func checkOpCode(b []byte, op Operation) error {
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
