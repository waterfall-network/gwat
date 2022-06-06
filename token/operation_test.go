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
	operator = common.HexToAddress("13e4acefe6a6700604929946e70e6443e4e73447")
	address  = common.HexToAddress("d049bfd667cb46aa3ef5df0da3e57db3be39e511")
	owner    = common.HexToAddress("1977c248e1014cc103929dd7f154199c916e39ec")
	to       = common.HexToAddress("7dc9c9730689ff0b0fd506c67db815f12d90a448")
	id       = big.NewInt(12345)
	data     = []byte{243, 12, 202, 20, 133, 116, 111, 107, 101, 110, 116, 100, 5}
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

	operations := struct {
		operations []op
	}{[]op{
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
	},
	}
	mintOperationEncode := func(o decodedOp, b []byte) error {
		op, err := NewMintOperation(
			o.to,
			o.id,
			o.data,
		)
		if err != nil {
			return err
		}

		have, err := EncodeToBytes(op)
		if err != nil {
			return fmt.Errorf("can`t encode operation %+v\nerror: %+v", op, err)
		}

		log.Printf("Have %+v", have)

		if !bytes.Equal(b, have) {
			return fmt.Errorf("values do not match:\n want: %+v\nhave: %+v", b, have)
		}

		return nil
	}

	mintOperationDecode := func(b []byte, m decodedOp) error {
		op, err := DecodeBytes(b)
		if err != nil {
			return err
		}

		o, ok := op.(MintOperation)
		if !ok {
			return fmt.Errorf("invalid operation type")
		}

		if o.Standard() != StdWRC721 {
			return fmt.Errorf("values do not match:\nwant: %+v\nhave: %+v", m.op, o.Standard())
		}

		if o.To() != m.to {
			return fmt.Errorf("values do not match:\nwant: %+v\nhave: %+v", o.To(), m.to)
		}

		haveData, ok := o.Metadata()
		if !ok {
			log.Printf("have data is empty")
		}

		if !bytes.Equal(haveData, m.data) {
			return fmt.Errorf("values do not match:\nwant: %+v\nhave: %+v", m.data, haveData)
		}

		haveId := o.TokenId().Int64()
		wantId := m.id

		if wantId == nil {
			wantId = big.NewInt(0)
		}

		if haveId != wantId.Int64() {
			return fmt.Errorf("values do not match:\nwant: %+v\nhave: %+v", haveId, wantId.Int64())
		}

		return nil
	}

	for _, op := range operations.operations {
		t.Run("encoding", func(t *testing.T) {
			err := mintOperationEncode(op.decoded, op.encoded)
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
			err := mintOperationDecode(op.encoded, op.decoded)
			if err != nil {
				switch err {
				case ErrNoTo, ErrNoTokenId:
					t.Log(err)
				default:
					t.Logf("operationDecode: invalid test case %s", op.caseName)
					t.Fatal(err)
				}
			}
		})
	}
}

//func TestSetApprovalForAllOperation(t *testing.T) {
//	type st struct {
//		caseName string
//		op       Std
//		operator common.Address
//		approve  bool
//	}
//
//	var operations = struct {
//		decoded []st
//		encoded [][]byte
//	}{
//		decoded: []st{
//			{
//				caseName: "setApprovalForAllOperation 1",
//				op:       StdWRC721,
//				operator: operator,
//				approve:  true,
//			},
//			{
//				caseName: "setApprovalForAllOperation 2",
//				op:       StdWRC20,
//				operator: common.Address{},
//				approve:  true,
//			},
//			{
//				caseName: "setApprovalForAllOperation 3",
//				op:       StdWRC721,
//				operator: operator,
//				approve:  false,
//			},
//			{
//				caseName: "setApprovalForAllOperation 4",
//				op:       0,
//				operator: common.Address{},
//				approve:  false,
//			},
//		},
//		encoded: [][]byte{
//			{
//				243, 37, 248, 134, 130, 2, 209, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 19, 228, 172, 239, 230, 166, 112, 6, 4, 146, 153, 70, 231, 14, 100, 67, 228, 231, 52, 71, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 1, 128,
//			},
//			{},
//			{
//				243, 37, 248, 134, 130, 2, 209, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 19, 228, 172, 239, 230, 166, 112, 6, 4, 146, 153, 70, 231, 14, 100, 67, 228, 231, 52, 71, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128,
//			},
//			{},
//		},
//	}
//
//	for i := 0; i < len(operations.decoded); i++ {
//		op, err := NewSetApprovalForAllOperation(
//			operations.decoded[i].operator,
//			operations.decoded[i].approve,
//		)
//		if err != nil {
//			switch err {
//			case ErrNoOperator:
//				t.Log(err)
//				continue
//			default:
//				t.Fatal(err)
//			}
//		}
//
//		t.Run("encoding", func(t *testing.T) {
//			err := operationEncode(op, operations.encoded[i])
//			if err != nil {
//				t.Logf("operationEncode: invalid test case %s", operations.decoded[i].caseName)
//				t.Fatal(err)
//			}
//		})
//
//		t.Run("decoding", func(t *testing.T) {
//			err := setApprovalForAllOperationDecode(operations.encoded[i], op)
//			if err != nil {
//				t.Logf("operationDecode: invalid test case %s", operations.decoded[i].caseName)
//				t.Fatal(err)
//			}
//		})
//	}
//}
//
//func setApprovalForAllOperationDecode(b []byte, s SetApprovalForAllOperation) error {
//	op, err := DecodeBytes(b)
//	if err != nil {
//		return err
//	}
//
//	o, ok := op.(SetApprovalForAllOperation)
//	if !ok {
//		return fmt.Errorf("invalid operation type")
//	}
//
//	err = comparisonStandartAndCodeOp(o, s)
//	if err != nil {
//		return err
//	}
//
//	operator := o.Operator()
//	if operator != s.Operator() {
//		return fmt.Errorf("values do not match:\nwant: %+v\nhave: %+v", s.Operator(), operator)
//	}
//
//	isApproved := o.IsApproved()
//	if isApproved != s.IsApproved() {
//		return fmt.Errorf("values do not match:\nwant: %+v\nhave: %+v", s.IsApproved(), isApproved)
//	}
//
//	return nil
//}
//
//func TestIsApprovedForAllOperation(t *testing.T) {
//	type st struct {
//		caseName string
//		op       Std
//		address  common.Address
//		owner    common.Address
//		operator common.Address
//	}
//
//	var operations = struct {
//		decoded []st
//		encoded [][]byte
//	}{
//		decoded: []st{
//			{
//				caseName: "isApprovedForAllOperation 1",
//				op:       StdWRC721,
//				address:  address,
//				owner:    owner,
//				operator: operator,
//			},
//			{
//				caseName: "setApprovalForAllOperation 2",
//				op:       StdWRC20,
//				address:  address,
//				owner:    common.Address{},
//				operator: operator,
//			},
//			{
//				caseName: "setApprovalForAllOperation 3",
//				op:       0,
//				address:  common.Address{},
//				owner:    owner,
//				operator: operator,
//			},
//			{
//				caseName: "setApprovalForAllOperation 4",
//				op:       0,
//				address:  address,
//				owner:    owner,
//				operator: common.Address{},
//			},
//		},
//		encoded: [][]byte{
//			{
//				243, 36, 248, 134, 130, 2, 209, 148, 208, 73, 191, 214, 103, 203, 70, 170, 62, 245, 223, 13, 163, 229, 125, 179, 190, 57, 229, 17, 128, 148, 25, 119, 194, 72, 225, 1, 76, 193, 3, 146, 157, 215, 241, 84, 25, 156, 145, 110, 57, 236, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 19, 228, 172, 239, 230, 166, 112, 6, 4, 146, 153, 70, 231, 14, 100, 67, 228, 231, 52, 71, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128,
//			},
//			{},
//			{},
//			{},
//		},
//	}
//
//	for i := 0; i < len(operations.decoded); i++ {
//		op, err := NewIsApprovedForAllOperation(
//			operations.decoded[i].address,
//			operations.decoded[i].owner,
//			operations.decoded[i].operator,
//		)
//		if err != nil {
//			switch err {
//			case ErrNoOperator, ErrNoOwner, ErrNoAddress:
//				t.Logf("error: %+v", err)
//				continue
//			default:
//				t.Fatal(err)
//			}
//		}
//
//		t.Run("encoding", func(t *testing.T) {
//			err := operationEncode(op, operations.encoded[i])
//			if err != nil {
//				t.Logf("operationEncode: invalid test case %s", operations.decoded[i].caseName)
//				t.Fatal(err)
//			}
//		})
//
//		t.Run("decoding", func(t *testing.T) {
//			err := isApprovedForAllOperationDecode(operations.encoded[i], op)
//			if err != nil {
//				t.Logf("operationDecode: invalid test case %s", operations.decoded[i].caseName)
//				t.Fatal(err)
//			}
//		})
//	}
//}
//
//func isApprovedForAllOperationDecode(b []byte, s IsApprovedForAllOperation) error {
//	op, err := DecodeBytes(b)
//	if err != nil {
//		return err
//	}
//
//	o, ok := op.(IsApprovedForAllOperation)
//	if !ok {
//		return fmt.Errorf("invalid operation type")
//	}
//
//	err = comparisonStandartAndCodeOp(o, s)
//	if err != nil {
//		return err
//	}
//
//	operator := o.Operator()
//	if operator != s.Operator() {
//		return fmt.Errorf("values do not match:\nwant: %+v\nhave: %+v", s.Owner(), operator)
//	}
//
//	owner := o.Owner()
//	if owner != s.Owner() {
//		return fmt.Errorf("values do not match:\nwant: %+v\nhave: %+v", s.Owner(), owner)
//	}
//
//	address := o.Address()
//	if address != s.Address() {
//		return fmt.Errorf("values do not match:\nwant: %+v\nhave: %+v", s.Address(), address)
//	}
//
//	return nil
//}

//func TestAllowanceOperationEncode(t *testing.T) {
//	t.Parallel()
//
//	allowanceOp, err := NewAllowanceOperation(
//		testingOperation.tokenAddress,
//		testingOperation.owner,
//		testingOperation.spender,
//	)
//	if err != nil {
//		switch err {
//		case ErrNoSpender, ErrNoOwner, ErrNoAddress:
//			t.Logf("error: %+v", err)
//		default:
//			t.Fatal(err)
//		}
//	}
//
//	have, err := EncodeToBytes(allowanceOp)
//	if err != nil {
//		t.Fatalf("can`t encode operation %+v\nerror: %+v", allowanceOp, err)
//	}
//
//	if !bytes.Equal(wantAllowanceOperation, have) {
//		t.Fatalf("values do not match:\n want: %+v\nhave: %+v", wantAllowanceOperation, have)
//	}
//}
//
//func TestApproveOperationEncode(t *testing.T) {
//	t.Parallel()
//
//	approveOp, err := NewApproveOperation(
//		testingOperation.op.Std,
//		testingOperation.spender,
//		testingOperation.value,
//	)
//	if err != nil {
//		switch err {
//		case ErrNoSpender:
//			t.Logf("error: %+v", err)
//		case ErrNoValue:
//			t.Logf("error: %+v", err)
//		default:
//			t.Fatal(err)
//		}
//	}
//
//	have, err := EncodeToBytes(approveOp)
//	if err != nil {
//		t.Fatalf("can`t encode operation %+v\nerror: %+v", approveOp, err)
//	}
//
//	if !bytes.Equal(wantApproveOperation, have) {
//		t.Fatalf("values do not match:\n want: %+v\nhave: %+v", wantApproveOperation, have)
//	}
//}
//
//func TestSafeTransferFromOperationEncode(t *testing.T) {
//	t.Parallel()
//
//	safeTransferFromOp, err := NewSafeTransferFromOperation(
//		testingOperation.from,
//		testingOperation.to,
//		testingOperation.value,
//		testingOperation.data,
//	)
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	have, err := EncodeToBytes(safeTransferFromOp)
//	if err != nil {
//		t.Fatalf("can`t encode operation %+v\nerror: %+v", safeTransferFromOp, err)
//	}
//
//	if !bytes.Equal(wantSafeTransferFromOperation, have) {
//		t.Fatalf("values do not match:\n want: %+v\nhave: %+v", wantSafeTransferFromOperation, have)
//	}
//}
//
//func TestTransferFromOperationEncode(t *testing.T) {
//	t.Parallel()
//
//	transferFromOp, err := NewTransferFromOperation(
//		testingOperation.op.Std,
//		testingOperation.from,
//		testingOperation.to,
//		testingOperation.value,
//	)
//	if err != nil {
//		switch err {
//		case ErrNoFrom:
//			t.Logf("error: %+v", err)
//		default:
//			t.Fatal(err)
//		}
//	}
//
//	have, err := EncodeToBytes(transferFromOp)
//	if err != nil {
//		t.Fatalf("can`t encode operation %+v\nerror: %+v", transferFromOp, err)
//	}
//
//	if !bytes.Equal(wantTransferFromOperation, have) {
//		t.Fatalf("values do not match:\n want: %+v\nhave: %+v", wantTransferFromOperation, have)
//	}
//}
//
//func TestTransferOperationEncode(t *testing.T) {
//	t.Parallel()
//
//	transferOp, err := NewTransferOperation(testingOperation.to, testingOperation.value)
//	if err != nil {
//		switch err {
//		case ErrNoTo:
//			t.Logf("error: %+v", err)
//		case ErrNoValue:
//			t.Logf("error: %+v", err)
//		default:
//			t.Fatal(err)
//		}
//	}
//
//	have, _ := EncodeToBytes(transferOp)
//	if err != nil {
//		t.Fatalf("can`t encode operation %+v\nerror: %+v", transferOp, err)
//	}
//
//	if !bytes.Equal(wantTransferOperation, have) {
//		t.Fatalf("values do not match:\n want: %+v\nhave: %+v", wantTransferOperation, have)
//	}
//}
//
//func TestBalanceOfOperationEncode(t *testing.T) {
//	t.Parallel()
//
//	balanceOp, err := NewBalanceOfOperation(testingOperation.tokenAddress, testingOperation.owner)
//	if err != nil {
//		switch err {
//		case ErrNoOwner:
//			t.Logf("error: %+v", err)
//		case ErrNoAddress:
//			t.Logf("error: %+v", err)
//		default:
//			t.Fatal(err)
//		}
//	}
//
//	have, err := EncodeToBytes(balanceOp)
//	if err != nil {
//		t.Fatalf("can`t encode operation %+v\nerror: %+v", balanceOp, err)
//	}
//
//	if !bytes.Equal(wantBalanceOfOperation, have) {
//		t.Fatalf("values do not match:\n want: %+v\nhave: %+v", wantBalanceOfOperation, have)
//	}
//}
//
//func TestBurnOperationEncode(t *testing.T) {
//	t.Parallel()
//
//	burnOp, err := NewBurnOperation(big.NewInt(testingOperation.tokenId.Id.Int64()))
//	if err != nil {
//		switch err {
//		case ErrNoTokenId:
//			t.Logf("error: %+v", err)
//		default:
//			t.Fatal(err)
//		}
//	}
//
//	have, err := EncodeToBytes(burnOp)
//	if err != nil {
//		t.Fatalf("can`t encode operation %+v\nerror: %+v", burnOp, err)
//	}
//
//	if !bytes.Equal(wantBurnOperation, have) {
//		t.Fatalf("values do not match: want: %+v\nhave: %+v", wantBurnOperation, have)
//	}
//}
//
//func TestCreateOperationEncode(t *testing.T) {
//	t.Parallel()
//
//	createOp, err := NewWrc20CreateOperation(
//		testingOperation.name,
//		testingOperation.symbol,
//		&testingOperation.decimals,
//		testingOperation.totalSupply)
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	have, err := EncodeToBytes(createOp)
//	if err != nil {
//		t.Fatalf("can`t encode operation %+v\nerror: %+v", createOp, err)
//	}
//
//	if !bytes.Equal(wantCreateOperation, have) {
//		t.Fatalf("values do not match: want: %+v\nhave: %+v", wantCreateOperation, have)
//	}
//}
//
//func TestPropertiesOperationEncode(t *testing.T) {
//	t.Parallel()
//
//	propertiesOp, err := NewPropertiesOperation(
//		testingOperation.tokenAddress,
//		testingOperation.tokenId.Id,
//	)
//	if err != nil {
//		switch err {
//		case ErrNoAddress:
//			t.Logf("error: %+v", err)
//		default:
//			t.Fatal(err)
//		}
//	}
//
//	have, err := EncodeToBytes(propertiesOp)
//	if err != nil {
//		t.Fatalf("can`t encode operation %+v\nerror: %+v", propertiesOp, err)
//	}
//
//	if !bytes.Equal(wantPropertiesOperation, have) {
//		t.Fatalf("values do not match: want: %+v\nhave: %+v", wantPropertiesOperation, have)
//	}
//}
//
//func TestTokenOfOwnerByIndexOperationEncode(t *testing.T) {
//	t.Parallel()
//
//	tokenOfOwnerByIndexOp, err := NewTokenOfOwnerByIndexOperation(
//		testingOperation.tokenAddress,
//		testingOperation.owner,
//		testingOperation.index)
//	if err != nil {
//		switch err {
//		case ErrNoIndex:
//			t.Logf("error: %+v", err)
//		case ErrNoOwner:
//			t.Logf("error: %+v", err)
//		case ErrNoAddress:
//			t.Logf("error: %+v", err)
//		default:
//			t.Fatal(err)
//		}
//	}
//
//	have, err := EncodeToBytes(tokenOfOwnerByIndexOp)
//	if err != nil {
//		t.Fatalf("can`t encode operation %+v\nerror: %+v", tokenOfOwnerByIndexOp, err)
//	}
//
//	if !bytes.Equal(have, wantTokenOfOwnerByIndexOperation) {
//		t.Fatalf("values do not match: want: %+v\nhave: %+v", wantTokenOfOwnerByIndexOperation, have)
//	}
//}
//
//func TestCreateOperationDecode(t *testing.T) {
//	t.Parallel()
//
//	op, err := DecodeBytes(wantCreateOperation)
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	o, ok := op.(CreateOperation)
//	if !ok {
//		t.Fatalf("invalid operation type")
//	}
//
//	std := o.Standard()
//	if std != testingOperation.op.Std {
//		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", testingOperation.op.Std, std)
//	}
//
//	haveOpCode := o.OpCode()
//	wantOpCode, err := GetOpCode(wantCreateOperation)
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	if haveOpCode != wantOpCode {
//		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", wantOpCode, haveOpCode)
//	}
//
//	name := o.Name()
//	if !bytes.Equal(name, testingOperation.name) {
//		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", string(testingOperation.name), string(name))
//	}
//
//	symbol := o.Symbol()
//	if !bytes.Equal(symbol, testingOperation.symbol) {
//		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", string(testingOperation.symbol), string(symbol))
//	}
//
//	decimals := o.Decimals()
//	if decimals != testingOperation.decimals {
//		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", testingOperation.decimals, decimals)
//	}
//
//	totalSupply, ok := o.TotalSupply()
//	if !ok {
//		t.Fatalf("totalSupply cannot be nil")
//	}
//
//	if totalSupply.Int64() != testingOperation.totalSupply.Int64() {
//		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", testingOperation.totalSupply.Int64(), totalSupply.Int64())
//	}
//}
//
//func TestTokenOfOwnerByIndexOperationDecode(t *testing.T) {
//	t.Parallel()
//
//	op, err := DecodeBytes(wantTokenOfOwnerByIndexOperation)
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	o, ok := op.(TokenOfOwnerByIndexOperation)
//	if !ok {
//		t.Fatalf("invalid operation type")
//	}
//
//	std := o.Standard()
//	if std != StdWRC721 {
//		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", StdWRC721, std)
//	}
//
//	haveOpCode := o.OpCode()
//	wantOpCode, err := GetOpCode(wantTokenOfOwnerByIndexOperation)
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	if haveOpCode != wantOpCode {
//		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", wantOpCode, haveOpCode)
//	}
//
//	haveIndex := o.Index()
//	wantIndex := testingOperation.index
//	if haveIndex.Int64() != wantIndex.Int64() {
//		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", wantIndex, haveIndex)
//	}
//
//	owner := o.Owner()
//	if owner != testingOperation.owner {
//		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", testingOperation.owner, owner)
//	}
//
//	address := o.Address()
//	if address != testingOperation.tokenAddress {
//		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", testingOperation.tokenAddress, address)
//	}
//}
//
//func TestBurnOperationDecode(t *testing.T) {
//	t.Parallel()
//
//	op, err := DecodeBytes(wantBurnOperation)
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	o, ok := op.(BurnOperation)
//	if !ok {
//		t.Fatalf("invalid operation type")
//	}
//
//	std := o.Standard()
//	if std != StdWRC721 {
//		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", StdWRC721, std)
//	}
//
//	haveOpCode := o.OpCode()
//	wantOpCode, err := GetOpCode(wantBurnOperation)
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	if haveOpCode != wantOpCode {
//		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", wantOpCode, haveOpCode)
//	}
//
//	id := o.TokenId()
//	if id.Int64() != testingOperation.tokenId.Id.Int64() {
//		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", testingOperation.tokenId.Id.Int64(), id)
//	}
//}
//
//func TestAllowanceOperationDecode(t *testing.T) {
//	t.Parallel()
//
//	op, err := DecodeBytes(wantAllowanceOperation)
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	o, ok := op.(AllowanceOperation)
//	if !ok {
//		t.Fatalf("invalid operation type")
//	}
//
//	std := o.Standard()
//	if std != StdWRC20 {
//		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", StdWRC20, std)
//	}
//
//	haveOpCode := o.OpCode()
//	wantOpCode, err := GetOpCode(wantAllowanceOperation)
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	if haveOpCode != wantOpCode {
//		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", wantOpCode, haveOpCode)
//	}
//
//	spender := o.Spender()
//	if spender != testingOperation.to {
//		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", testingOperation.spender, spender)
//	}
//
//	owner := o.Owner()
//	if owner != testingOperation.owner {
//		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", testingOperation.owner, owner)
//	}
//
//	address := o.Address()
//	if address != testingOperation.tokenAddress {
//		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", testingOperation.tokenAddress, address)
//	}
//}
//
//func TestApproveOperationDecode(t *testing.T) {
//	t.Parallel()
//
//	op, err := DecodeBytes(wantApproveOperation)
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	o, ok := op.(ApproveOperation)
//	if !ok {
//		t.Fatalf("invalid operation type")
//	}
//
//	std := o.Standard()
//	if std != testingOperation.op.Std {
//		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", testingOperation.op.Std, std)
//	}
//
//	haveOpCode := o.OpCode()
//	wantOpCode, err := GetOpCode(wantApproveOperation)
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	if haveOpCode != wantOpCode {
//		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", wantOpCode, haveOpCode)
//	}
//
//	spender := o.Spender()
//	if spender != testingOperation.to {
//		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", testingOperation.spender, spender)
//	}
//
//	value := o.Value().Int64()
//	if value != testingOperation.value.Int64() {
//		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", testingOperation.value.Int64(), value)
//	}
//}
//
//func TestSafeTransferFromOperationDecode(t *testing.T) {
//	t.Parallel()
//
//	op, err := DecodeBytes(wantSafeTransferFromOperation)
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	o, ok := op.(SafeTransferFromOperation)
//	if !ok {
//		t.Fatalf("invalid operation type")
//	}
//
//	std := o.Standard()
//	if std != StdWRC721 {
//		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", StdWRC721, std)
//	}
//
//	haveOpCode, err := GetOpCode(wantSafeTransferFromOperation)
//	if err != nil {
//		switch err {
//		case ErrRawDataShort, ErrPrefixNotValid:
//			t.Logf("error: %+v", err)
//		default:
//			t.Fatal(err)
//		}
//	}
//	if haveOpCode != OpSafeTransferFrom {
//		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", OpSafeTransferFrom, haveOpCode)
//	}
//
//	from := o.From()
//	if from != testingOperation.from {
//		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", testingOperation.from, from)
//	}
//
//	to := o.To()
//	if to != testingOperation.to {
//		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", testingOperation.to, to)
//	}
//
//	value := o.Value().Int64()
//	if value != testingOperation.value.Int64() {
//		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", testingOperation.value.Int64(), value)
//	}
//
//	data, ok := o.Data()
//	if !ok {
//		t.Logf("data is empty")
//	}
//
//	if !bytes.Equal(data, testingOperation.data) {
//		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", testingOperation.data, data)
//	}
//}
//
//func TestTransferFromOperationDecode(t *testing.T) {
//	t.Parallel()
//
//	op, err := DecodeBytes(wantTransferFromOperation)
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	o, ok := op.(TransferFromOperation)
//	if !ok {
//		t.Fatalf("invalid operation type")
//	}
//
//	std := o.Standard()
//	if std != testingOperation.op.Std {
//		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", testingOperation.op.Std, std)
//	}
//
//	haveOpCode, err := GetOpCode(wantTransferFromOperation)
//	if err != nil {
//		switch err {
//		case ErrRawDataShort, ErrPrefixNotValid:
//			t.Logf("error: %+v", err)
//		default:
//			t.Fatal(err)
//		}
//	}
//
//	if haveOpCode != OpTransferFrom {
//		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", OpTransferFrom, haveOpCode)
//	}
//
//	from := o.From()
//	if from != testingOperation.from {
//		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", testingOperation.from, from)
//	}
//
//	to := o.To()
//	if to != testingOperation.to {
//		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", testingOperation.to, to)
//	}
//
//	value := o.Value().Int64()
//	if value != testingOperation.value.Int64() {
//		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", testingOperation.value.Int64(), value)
//	}
//}
//
//func TestTransferOperationDecode(t *testing.T) {
//	t.Parallel()
//
//	op, err := DecodeBytes(wantTransferOperation)
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	o, ok := op.(TransferOperation)
//	if !ok {
//		t.Fatalf("invalid operation type")
//	}
//
//	std := o.Standard()
//	if std != StdWRC20 {
//		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", StdWRC20, std)
//	}
//
//	haveOpCode := o.OpCode()
//	wantOpCode, err := GetOpCode(wantTransferOperation)
//	if err != nil {
//		switch err {
//		case ErrRawDataShort, ErrPrefixNotValid:
//			t.Logf("error: %+v", err)
//		default:
//			t.Fatal(err)
//		}
//	}
//
//	if haveOpCode != wantOpCode {
//		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", wantOpCode, haveOpCode)
//	}
//
//	to := o.To()
//	if to != testingOperation.to {
//		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", testingOperation.to, to)
//	}
//
//	value := o.Value().Int64()
//	if value != testingOperation.value.Int64() {
//		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", testingOperation.value.Int64(), value)
//	}
//}
//
//func TestBalanceOfOperationDecode(t *testing.T) {
//	t.Parallel()
//
//	op, err := DecodeBytes(wantBalanceOfOperation)
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	o, ok := op.(BalanceOfOperation)
//	if !ok {
//		t.Fatalf("invalid operation type")
//	}
//
//	std := o.Standard()
//	if std != 0 {
//		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", 0, std)
//	}
//
//	haveOpCode := o.OpCode()
//	wantOpCode, err := GetOpCode(wantBalanceOfOperation)
//	if err != nil {
//		switch err {
//		case ErrRawDataShort, ErrPrefixNotValid:
//			t.Logf("error: %+v", err)
//		default:
//			t.Fatal(err)
//		}
//	}
//
//	if haveOpCode != wantOpCode {
//		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", wantOpCode, haveOpCode)
//	}
//
//	address := o.Address()
//	if address != testingOperation.tokenAddress {
//		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", testingOperation.tokenAddress, address)
//	}
//
//	owner := o.Owner()
//	if owner != testingOperation.owner {
//		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", testingOperation.owner, owner)
//	}
//}
//
//func TestPropertiesOperationDecode(t *testing.T) {
//	t.Parallel()
//
//	op, err := DecodeBytes(wantPropertiesOperation)
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	o, ok := op.(PropertiesOperation)
//	if !ok {
//		t.Fatalf("invalid operation type")
//	}
//
//	std := o.Standard()
//	if std != 0 {
//		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", testingOperation.op.Std, std)
//	}
//
//	haveOpCode := o.OpCode()
//	wantOpCode, err := GetOpCode(wantPropertiesOperation)
//	if err != nil {
//		switch err {
//		case ErrRawDataShort, ErrPrefixNotValid:
//			t.Logf("error: %+v", err)
//		default:
//			t.Fatal(err)
//		}
//	}
//
//	if haveOpCode != wantOpCode {
//		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", wantOpCode, haveOpCode)
//	}
//
//	address := o.Address()
//	if address != testingOperation.tokenAddress {
//		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", testingOperation.tokenAddress, address)
//	}
//
//	id, ok := o.TokenId()
//	if !ok {
//		t.Logf("tokenId is empty")
//	}
//	if id.Int64() != testingOperation.tokenId.Id.Int64() {
//		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", testingOperation.tokenId.Id.Int64(), id.Int64())
//	}
//}
