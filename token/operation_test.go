package token

import (
	"bytes"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"testing"
)

var (
	wantCreateOperation = []byte{
		243, 12, 202, 20, 133, 116, 111, 107, 101, 110, 116, 100, 5,
	}
	wantTokenOfOwnerByIndexOperation = []byte{
		243, 40, 248, 136, 130, 2, 209, 148, 208, 73, 191, 214, 103, 203, 70, 170, 62, 245, 223, 13, 163, 229, 125, 179, 190, 57, 229, 17, 128, 148, 208, 73, 191, 214, 103, 203, 70, 170, 62, 245, 223, 13, 163, 229, 125, 179, 190, 57, 229, 17, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 130, 43, 103, 128, 128}
	wantBurnOperation = []byte{
		243, 39, 248, 136, 130, 2, 209, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 130, 86, 206, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128,
	}
	wantPropertiesOperation = []byte{
		243, 33, 248, 134, 128, 148, 208, 73, 191, 214, 103, 203, 70, 170, 62, 245, 223, 13, 163, 229, 125, 179, 190, 57, 229, 17, 130, 86, 206, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128,
	}
	wantBalanceOfOperation = []byte{
		243, 34, 248, 132, 128, 148, 208, 73, 191, 214, 103, 203, 70, 170, 62, 245, 223, 13, 163, 229, 125, 179, 190, 57, 229, 17, 128, 148, 208, 73, 191, 214, 103, 203, 70, 170, 62, 245, 223, 13, 163, 229, 125, 179, 190, 57, 229, 17, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128,
	}
	wantTransferOperation = []byte{
		243, 30, 248, 134, 20, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 208, 73, 191, 214, 103, 203, 70, 170, 62, 245, 223, 13, 163, 229, 125, 179, 190, 57, 229, 17, 130, 195, 80, 128, 128, 128,
	}
	wantTransferFromOperation = []byte{
		243, 31, 248, 134, 20, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 208, 73, 191, 214, 103, 203, 70, 170, 62, 245, 223, 13, 163, 229, 125, 179, 190, 57, 229, 17, 148, 208, 73, 191, 214, 103, 203, 70, 170, 62, 245, 223, 13, 163, 229, 125, 179, 190, 57, 229, 17, 130, 195, 80, 128, 128, 128,
	}
	wantSafeTransferFromOperation = []byte{
		243, 41, 248, 136, 130, 2, 209, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 208, 73, 191, 214, 103, 203, 70, 170, 62, 245, 223, 13, 163, 229, 125, 179, 190, 57, 229, 17, 148, 208, 73, 191, 214, 103, 203, 70, 170, 62, 245, 223, 13, 163, 229, 125, 179, 190, 57, 229, 17, 130, 195, 80, 128, 128, 128,
	}
	wantApproveOperation = []byte{
		243, 13, 248, 134, 20, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 208, 73, 191, 214, 103, 203, 70, 170, 62, 245, 223, 13, 163, 229, 125, 179, 190, 57, 229, 17, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 130, 195, 80, 128, 128, 128,
	}
	wantAllowanceOperation = []byte{
		243, 35, 248, 132, 20, 148, 208, 73, 191, 214, 103, 203, 70, 170, 62, 245, 223, 13, 163, 229, 125, 179, 190, 57, 229, 17, 128, 148, 208, 73, 191, 214, 103, 203, 70, 170, 62, 245, 223, 13, 163, 229, 125, 179, 190, 57, 229, 17, 148, 208, 73, 191, 214, 103, 203, 70, 170, 62, 245, 223, 13, 163, 229, 125, 179, 190, 57, 229, 17, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128,
	}
	wantIsApprovedForAllOperation = []byte{
		243, 36, 248, 134, 130, 2, 209, 148, 208, 73, 191, 214, 103, 203, 70, 170, 62, 245, 223, 13, 163, 229, 125, 179, 190, 57, 229, 17, 128, 148, 208, 73, 191, 214, 103, 203, 70, 170, 62, 245, 223, 13, 163, 229, 125, 179, 190, 57, 229, 17, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 208, 73, 191, 214, 103, 203, 70, 170, 62, 245, 223, 13, 163, 229, 125, 179, 190, 57, 229, 17, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 128, 128,
	}
	wantSetApprovalForAllOperation = []byte{
		243, 37, 248, 134, 130, 2, 209, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 208, 73, 191, 214, 103, 203, 70, 170, 62, 245, 223, 13, 163, 229, 125, 179, 190, 57, 229, 17, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 128, 1, 128,
	}
	wantMintOperation = []byte{
		243, 38, 248, 136, 130, 2, 209, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 130, 86, 206, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 148, 208, 73, 191, 214, 103, 203, 70, 170, 62, 245, 223, 13, 163, 229, 125, 179, 190, 57, 229, 17, 128, 128, 128, 128,
	}
)

var testingOperation = struct {
	op           operation
	tokenId      tokenIdOperation
	tokenAddress common.Address
	name         []byte
	symbol       []byte
	decimals     uint8
	totalSupply  *big.Int
	baseURI      []byte
	owner        common.Address
	spender      common.Address
	operator     common.Address
	from         common.Address
	to           common.Address
	value        *big.Int
	index        *big.Int
	isApproved   bool
	data         []byte
}{
	operation{Std(StdWRC20)},
	tokenIdOperation{big.NewInt(22222)},
	common.HexToAddress("d049bfd667cb46aa3ef5df0da3e57db3be39e511"),
	[]byte("token"),
	[]byte("t"),
	5,
	big.NewInt(100),
	nil,
	common.HexToAddress("d049bfd667cb46aa3ef5df0da3e57db3be39e511"),
	common.HexToAddress("d049bfd667cb46aa3ef5df0da3e57db3be39e511"),
	common.HexToAddress("d049bfd667cb46aa3ef5df0da3e57db3be39e511"),
	common.HexToAddress("d049bfd667cb46aa3ef5df0da3e57db3be39e511"),
	common.HexToAddress("d049bfd667cb46aa3ef5df0da3e57db3be39e511"),
	big.NewInt(50000),
	big.NewInt(11111),
	true,
	nil,
}

func TestMintOperationEncode(t *testing.T) {
	t.Parallel()

	mintOp, err := NewMintOperation(
		testingOperation.to,
		testingOperation.tokenId.Id,
		testingOperation.data,
	)
	if err != nil {
		switch err {
		case ErrNoTo:
			t.Logf("error: %+v", err)
		case ErrNoTokenId:
			t.Logf("error: %+v", err)
		default:
			t.Fatal(err)
		}
	}

	have, err := EncodeToBytes(mintOp)
	if err != nil {
		t.Fatalf("can`t encode operation %+v\nerror: %+v", mintOp, err)
	}

	if !bytes.Equal(wantMintOperation, have) {
		t.Fatalf("values do not match:\n want: %+v\nhave: %+v", wantMintOperation, have)
	}
}

func TestSetApprovalForAllOperationEncode(t *testing.T) {
	t.Parallel()

	setApprovalForAllOp, err := NewSetApprovalForAllOperation(
		testingOperation.operator,
		testingOperation.isApproved,
	)

	if err != nil {
		switch err {
		case ErrNoOperator:
			t.Logf("error: %+v", err)
		default:
			t.Fatal(err)
		}
	}

	have, err := EncodeToBytes(setApprovalForAllOp)
	if err != nil {
		t.Fatalf("can`t encode operation %+v\nerror: %+v", setApprovalForAllOp, err)
	}

	if !bytes.Equal(wantSetApprovalForAllOperation, have) {
		t.Fatalf("values do not match:\n want: %+v\nhave: %+v", wantSetApprovalForAllOperation, have)
	}
}

func TestIsApprovedForAllOperationEncode(t *testing.T) {
	t.Parallel()

	isApprovedForAllOp, err := NewIsApprovedForAllOperation(
		testingOperation.tokenAddress,
		testingOperation.owner,
		testingOperation.operator,
	)
	if err != nil {
		switch err {
		case ErrNoOperator:
			t.Logf("error: %+v", err)
		case ErrNoOwner:
			t.Logf("error: %+v", err)
		case ErrNoAddress:
			t.Logf("error: %+v", err)
		default:
			t.Fatal(err)
		}
	}

	have, err := EncodeToBytes(isApprovedForAllOp)
	if err != nil {
		t.Fatalf("can`t encode operation %+v\nerror: %+v", isApprovedForAllOp, err)
	}

	if !bytes.Equal(wantIsApprovedForAllOperation, have) {
		t.Fatalf("values do not match:\n want: %+v\nhave: %+v", wantIsApprovedForAllOperation, have)
	}
}

func TestAllowanceOperationEncode(t *testing.T) {
	t.Parallel()

	allowanceOp, err := NewAllowanceOperation(
		testingOperation.tokenAddress,
		testingOperation.owner,
		testingOperation.spender,
	)
	if err != nil {
		switch err {
		case ErrNoSpender:
			t.Logf("error: %+v", err)
		case ErrNoOwner:
			t.Logf("error: %+v", err)
		case ErrNoAddress:
			t.Logf("error: %+v", err)
		default:
			t.Fatal(err)
		}
	}

	have, err := EncodeToBytes(allowanceOp)
	if err != nil {
		t.Fatalf("can`t encode operation %+v\nerror: %+v", allowanceOp, err)
	}

	if !bytes.Equal(wantAllowanceOperation, have) {
		t.Fatalf("values do not match:\n want: %+v\nhave: %+v", wantAllowanceOperation, have)
	}
}

func TestApproveOperationEncode(t *testing.T) {
	t.Parallel()

	approveOp, err := NewApproveOperation(
		testingOperation.op.Std,
		testingOperation.spender,
		testingOperation.value,
	)
	if err != nil {
		switch err {
		case ErrNoSpender:
			t.Logf("error: %+v", err)
		case ErrNoValue:
			t.Logf("error: %+v", err)
		default:
			t.Fatal(err)
		}
	}

	have, err := EncodeToBytes(approveOp)
	if err != nil {
		t.Fatalf("can`t encode operation %+v\nerror: %+v", approveOp, err)
	}

	if !bytes.Equal(wantApproveOperation, have) {
		t.Fatalf("values do not match:\n want: %+v\nhave: %+v", wantApproveOperation, have)
	}
}

func TestSafeTransferFromOperationEncode(t *testing.T) {
	t.Parallel()

	safeTransferFromOp, err := NewSafeTransferFromOperation(
		testingOperation.from,
		testingOperation.to,
		testingOperation.value,
		testingOperation.data,
	)
	if err != nil {
		t.Fatal(err)
	}

	have, err := EncodeToBytes(safeTransferFromOp)
	if err != nil {
		t.Fatalf("can`t encode operation %+v\nerror: %+v", safeTransferFromOp, err)
	}

	if !bytes.Equal(wantSafeTransferFromOperation, have) {
		t.Fatalf("values do not match:\n want: %+v\nhave: %+v", wantSafeTransferFromOperation, have)
	}
}

func TestTransferFromOperationEncode(t *testing.T) {
	t.Parallel()

	transferFromOp, err := NewTransferFromOperation(
		testingOperation.op.Std,
		testingOperation.from,
		testingOperation.to,
		testingOperation.value,
	)
	if err != nil {
		switch err {
		case ErrNoFrom:
			t.Logf("error: %+v", err)
		default:
			t.Fatal(err)
		}
	}

	have, err := EncodeToBytes(transferFromOp)
	if err != nil {
		t.Fatalf("can`t encode operation %+v\nerror: %+v", transferFromOp, err)
	}

	if !bytes.Equal(wantTransferFromOperation, have) {
		t.Fatalf("values do not match:\n want: %+v\nhave: %+v", wantTransferFromOperation, have)
	}
}

func TestTransferOperationEncode(t *testing.T) {
	t.Parallel()

	transferOp, err := NewTransferOperation(testingOperation.to, testingOperation.value)
	if err != nil {
		switch err {
		case ErrNoTo:
			t.Logf("error: %+v", err)
		case ErrNoValue:
			t.Logf("error: %+v", err)
		default:
			t.Fatal(err)
		}
	}

	have, _ := EncodeToBytes(transferOp)
	if err != nil {
		t.Fatalf("can`t encode operation %+v\nerror: %+v", transferOp, err)
	}

	if !bytes.Equal(wantTransferOperation, have) {
		t.Fatalf("values do not match:\n want: %+v\nhave: %+v", wantTransferOperation, have)
	}
}

func TestBalanceOfOperationEncode(t *testing.T) {
	t.Parallel()

	balanceOp, err := NewBalanceOfOperation(testingOperation.tokenAddress, testingOperation.owner)
	if err != nil {
		switch err {
		case ErrNoOwner:
			t.Logf("error: %+v", err)
		case ErrNoAddress:
			t.Logf("error: %+v", err)
		default:
			t.Fatal(err)
		}
	}

	have, err := EncodeToBytes(balanceOp)
	if err != nil {
		t.Fatalf("can`t encode operation %+v\nerror: %+v", balanceOp, err)
	}

	if !bytes.Equal(wantBalanceOfOperation, have) {
		t.Fatalf("values do not match:\n want: %+v\nhave: %+v", wantBalanceOfOperation, have)
	}
}

func TestBurnOperationEncode(t *testing.T) {
	t.Parallel()

	burnOp, err := NewBurnOperation(big.NewInt(testingOperation.tokenId.Id.Int64()))
	if err != nil {
		switch err {
		case ErrNoTokenId:
			t.Logf("error: %+v", err)
		default:
			t.Fatal(err)
		}
	}

	have, err := EncodeToBytes(burnOp)
	if err != nil {
		t.Fatalf("can`t encode operation %+v\nerror: %+v", burnOp, err)
	}

	if !bytes.Equal(wantBurnOperation, have) {
		t.Fatalf("values do not match: want: %+v\nhave: %+v", wantBurnOperation, have)
	}
}

func TestCreateOperationEncode(t *testing.T) {
	t.Parallel()

	createOp, err := NewWrc20CreateOperation(
		testingOperation.name,
		testingOperation.symbol,
		&testingOperation.decimals,
		testingOperation.totalSupply)
	if err != nil {
		t.Fatal(err)
	}

	have, err := EncodeToBytes(createOp)
	if err != nil {
		t.Fatalf("can`t encode operation %+v\nerror: %+v", createOp, err)
	}

	if !bytes.Equal(wantCreateOperation, have) {
		t.Fatalf("values do not match: want: %+v\nhave: %+v", wantCreateOperation, have)
	}
}

func TestPropertiesOperationEncode(t *testing.T) {
	t.Parallel()

	propertiesOp, err := NewPropertiesOperation(
		testingOperation.tokenAddress,
		testingOperation.tokenId.Id,
	)
	if err != nil {
		switch err {
		case ErrNoAddress:
			t.Logf("error: %+v", err)
		default:
			t.Fatal(err)
		}
	}

	have, err := EncodeToBytes(propertiesOp)
	if err != nil {
		t.Fatalf("can`t encode operation %+v\nerror: %+v", propertiesOp, err)
	}

	if !bytes.Equal(wantPropertiesOperation, have) {
		t.Fatalf("values do not match: want: %+v\nhave: %+v", wantPropertiesOperation, have)
	}
}

func TestTokenOfOwnerByIndexOperationEncode(t *testing.T) {
	t.Parallel()

	tokenOfOwnerByIndexOp, err := NewTokenOfOwnerByIndexOperation(
		testingOperation.tokenAddress,
		testingOperation.owner,
		testingOperation.index)
	if err != nil {
		switch err {
		case ErrNoIndex:
			t.Logf("error: %+v", err)
		case ErrNoOwner:
			t.Logf("error: %+v", err)
		case ErrNoAddress:
			t.Logf("error: %+v", err)
		default:
			t.Fatal(err)
		}
	}

	have, err := EncodeToBytes(tokenOfOwnerByIndexOp)
	if err != nil {
		t.Fatalf("can`t encode operation %+v\nerror: %+v", tokenOfOwnerByIndexOp, err)
	}

	if !bytes.Equal(have, wantTokenOfOwnerByIndexOperation) {
		t.Fatalf("values do not match: want: %+v\nhave: %+v", wantTokenOfOwnerByIndexOperation, have)
	}
}

func TestCreateOperationDecode(t *testing.T) {
	t.Parallel()

	op, err := DecodeBytes(wantCreateOperation)
	if err != nil {
		t.Fatal(err)
	}

	o, ok := op.(CreateOperation)
	if !ok {
		t.Fatalf("invalid operation type")
	}

	std := o.Standard()
	if std != testingOperation.op.Std {
		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", testingOperation.op.Std, std)
	}

	haveOpCode := o.OpCode()
	wantOpCode, err := GetOpCode(wantCreateOperation)
	if err != nil {
		t.Fatal(err)
	}

	if haveOpCode != wantOpCode {
		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", wantOpCode, haveOpCode)
	}

	name := o.Name()
	if !bytes.Equal(name, testingOperation.name) {
		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", string(testingOperation.name), string(name))
	}

	symbol := o.Symbol()
	if !bytes.Equal(symbol, testingOperation.symbol) {
		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", string(testingOperation.symbol), string(symbol))
	}

	decimals := o.Decimals()
	if decimals != testingOperation.decimals {
		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", testingOperation.decimals, decimals)
	}

	totalSupply, ok := o.TotalSupply()
	if !ok {
		t.Fatalf("totalSupply cannot be nil")
	}

	if totalSupply.Int64() != testingOperation.totalSupply.Int64() {
		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", testingOperation.totalSupply.Int64(), totalSupply.Int64())
	}
}

func TestTokenOfOwnerByIndexOperationDecode(t *testing.T) {
	t.Parallel()

	op, err := DecodeBytes(wantTokenOfOwnerByIndexOperation)
	if err != nil {
		t.Fatal(err)
	}

	o, ok := op.(TokenOfOwnerByIndexOperation)
	if !ok {
		t.Fatalf("invalid operation type")
	}

	std := o.Standard()
	if std != StdWRC721 {
		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", StdWRC721, std)
	}

	haveOpCode := o.OpCode()
	wantOpCode, err := GetOpCode(wantTokenOfOwnerByIndexOperation)
	if err != nil {
		t.Fatal(err)
	}

	if haveOpCode != wantOpCode {
		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", wantOpCode, haveOpCode)
	}

	haveIndex := o.Index()
	wantIndex := testingOperation.index
	if haveIndex.Int64() != wantIndex.Int64() {
		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", wantIndex, haveIndex)
	}

	owner := o.Owner()
	if owner != testingOperation.owner {
		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", testingOperation.owner, owner)
	}

	address := o.Address()
	if address != testingOperation.tokenAddress {
		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", testingOperation.tokenAddress, address)
	}
}

func TestBurnOperationDecode(t *testing.T) {
	t.Parallel()

	op, err := DecodeBytes(wantBurnOperation)
	if err != nil {
		t.Fatal(err)
	}

	o, ok := op.(BurnOperation)
	if !ok {
		t.Fatalf("invalid operation type")
	}

	std := o.Standard()
	if std != StdWRC721 {
		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", StdWRC721, std)
	}

	haveOpCode := o.OpCode()
	wantOpCode, err := GetOpCode(wantBurnOperation)
	if err != nil {
		t.Fatal(err)
	}

	if haveOpCode != wantOpCode {
		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", wantOpCode, haveOpCode)
	}

	id := o.TokenId()
	if id.Int64() != testingOperation.tokenId.Id.Int64() {
		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", testingOperation.tokenId.Id.Int64(), id)
	}
}

func TestMintOperationDecode(t *testing.T) {
	t.Parallel()

	op, err := DecodeBytes(wantMintOperation)
	if err != nil {
		t.Fatal(err)
	}

	o, ok := op.(MintOperation)
	if !ok {
		t.Fatalf("invalid operation type")
	}

	std := o.Standard()
	if std != StdWRC721 {
		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", StdWRC721, std)
	}

	haveOpCode := o.OpCode()
	wantOpCode, err := GetOpCode(wantMintOperation)
	if err != nil {
		t.Fatal(err)
	}

	if haveOpCode != wantOpCode {
		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", wantOpCode, haveOpCode)
	}

	address := o.To()
	if address != testingOperation.to {
		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", testingOperation.to, address)
	}

	data, ok := o.Metadata()
	if !ok {
		t.Logf("data is empty")
	}

	if !bytes.Equal(data, testingOperation.data) {
		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", testingOperation.data, data)
	}
}

func TestSetApprovalForAllOperationDecode(t *testing.T) {
	t.Parallel()

	op, err := DecodeBytes(wantSetApprovalForAllOperation)
	if err != nil {
		t.Fatal(err)
	}

	o, ok := op.(SetApprovalForAllOperation)
	if !ok {
		t.Fatalf("invalid operation type")
	}

	std := o.Standard()
	if std != StdWRC721 {
		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", StdWRC721, std)
	}

	haveOpCode := o.OpCode()
	wantOpCode, err := GetOpCode(wantSetApprovalForAllOperation)
	if err != nil {
		t.Fatal(err)
	}

	if haveOpCode != wantOpCode {
		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", wantOpCode, haveOpCode)
	}

	operator := o.Operator()
	if operator != testingOperation.to {
		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", testingOperation.operator, operator)
	}

	isApproved := o.IsApproved()
	if isApproved != testingOperation.isApproved {
		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", testingOperation.isApproved, isApproved)
	}
}

func TestIsApprovedForAllOperationDecode(t *testing.T) {
	t.Parallel()

	op, err := DecodeBytes(wantIsApprovedForAllOperation)
	if err != nil {
		t.Fatal(err)
	}

	o, ok := op.(IsApprovedForAllOperation)
	if !ok {
		t.Fatalf("invalid operation type")
	}

	std := o.Standard()
	if std != StdWRC721 {
		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", StdWRC721, std)
	}

	haveOpCode := o.OpCode()
	wantOpCode, err := GetOpCode(wantIsApprovedForAllOperation)
	if err != nil {
		t.Fatal(err)
	}

	if haveOpCode != wantOpCode {
		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", wantOpCode, haveOpCode)
	}

	operator := o.Operator()
	if operator != testingOperation.to {
		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", testingOperation.operator, operator)
	}

	owner := o.Owner()
	if owner != testingOperation.owner {
		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", testingOperation.owner, owner)
	}

	address := o.Address()
	if address != testingOperation.tokenAddress {
		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", testingOperation.tokenAddress, address)
	}
}

func TestAllowanceOperationDecode(t *testing.T) {
	t.Parallel()

	op, err := DecodeBytes(wantAllowanceOperation)
	if err != nil {
		t.Fatal(err)
	}

	o, ok := op.(AllowanceOperation)
	if !ok {
		t.Fatalf("invalid operation type")
	}

	std := o.Standard()
	if std != StdWRC20 {
		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", StdWRC20, std)
	}

	haveOpCode := o.OpCode()
	wantOpCode, err := GetOpCode(wantAllowanceOperation)
	if err != nil {
		t.Fatal(err)
	}

	if haveOpCode != wantOpCode {
		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", wantOpCode, haveOpCode)
	}

	spender := o.Spender()
	if spender != testingOperation.to {
		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", testingOperation.spender, spender)
	}

	owner := o.Owner()
	if owner != testingOperation.owner {
		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", testingOperation.owner, owner)
	}

	address := o.Address()
	if address != testingOperation.tokenAddress {
		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", testingOperation.tokenAddress, address)
	}
}

func TestApproveOperationDecode(t *testing.T) {
	t.Parallel()

	op, err := DecodeBytes(wantApproveOperation)
	if err != nil {
		t.Fatal(err)
	}

	o, ok := op.(ApproveOperation)
	if !ok {
		t.Fatalf("invalid operation type")
	}

	std := o.Standard()
	if std != testingOperation.op.Std {
		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", testingOperation.op.Std, std)
	}

	haveOpCode := o.OpCode()
	wantOpCode, err := GetOpCode(wantApproveOperation)
	if err != nil {
		t.Fatal(err)
	}

	if haveOpCode != wantOpCode {
		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", wantOpCode, haveOpCode)
	}

	spender := o.Spender()
	if spender != testingOperation.to {
		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", testingOperation.spender, spender)
	}

	value := o.Value().Int64()
	if value != testingOperation.value.Int64() {
		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", testingOperation.value.Int64(), value)
	}
}

func TestSafeTransferFromOperationDecode(t *testing.T) {
	t.Parallel()

	op, err := DecodeBytes(wantSafeTransferFromOperation)
	if err != nil {
		t.Fatal(err)
	}

	o, ok := op.(SafeTransferFromOperation)
	if !ok {
		t.Fatalf("invalid operation type")
	}

	std := o.Standard()
	if std != StdWRC721 {
		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", StdWRC721, std)
	}

	haveOpCode, err := GetOpCode(wantSafeTransferFromOperation)
	if err != nil {
		switch err {
		case ErrRawDataShort, ErrPrefixNotValid:
			t.Logf("error: %+v", err)
		default:
			t.Fatal(err)
		}
	}
	if haveOpCode != OpSafeTransferFrom {
		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", OpSafeTransferFrom, haveOpCode)
	}

	from := o.From()
	if from != testingOperation.from {
		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", testingOperation.from, from)
	}

	to := o.To()
	if to != testingOperation.to {
		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", testingOperation.to, to)
	}

	value := o.Value().Int64()
	if value != testingOperation.value.Int64() {
		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", testingOperation.value.Int64(), value)
	}

	data, ok := o.Data()
	if !ok {
		t.Logf("data is empty")
	}

	if !bytes.Equal(data, testingOperation.data) {
		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", testingOperation.data, data)
	}
}

func TestTransferFromOperationDecode(t *testing.T) {
	t.Parallel()

	op, err := DecodeBytes(wantTransferFromOperation)
	if err != nil {
		t.Fatal(err)
	}

	o, ok := op.(TransferFromOperation)
	if !ok {
		t.Fatalf("invalid operation type")
	}

	std := o.Standard()
	if std != testingOperation.op.Std {
		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", testingOperation.op.Std, std)
	}

	haveOpCode, err := GetOpCode(wantTransferFromOperation)
	if err != nil {
		switch err {
		case ErrRawDataShort, ErrPrefixNotValid:
			t.Logf("error: %+v", err)
		default:
			t.Fatal(err)
		}
	}

	if haveOpCode != OpTransferFrom {
		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", OpTransferFrom, haveOpCode)
	}

	from := o.From()
	if from != testingOperation.from {
		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", testingOperation.from, from)
	}

	to := o.To()
	if to != testingOperation.to {
		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", testingOperation.to, to)
	}

	value := o.Value().Int64()
	if value != testingOperation.value.Int64() {
		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", testingOperation.value.Int64(), value)
	}
}

func TestTransferOperationDecode(t *testing.T) {
	t.Parallel()

	op, err := DecodeBytes(wantTransferOperation)
	if err != nil {
		t.Fatal(err)
	}

	o, ok := op.(TransferOperation)
	if !ok {
		t.Fatalf("invalid operation type")
	}

	std := o.Standard()
	if std != StdWRC20 {
		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", StdWRC20, std)
	}

	haveOpCode := o.OpCode()
	wantOpCode, err := GetOpCode(wantTransferOperation)
	if err != nil {
		switch err {
		case ErrRawDataShort, ErrPrefixNotValid:
			t.Logf("error: %+v", err)
		default:
			t.Fatal(err)
		}
	}

	if haveOpCode != wantOpCode {
		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", wantOpCode, haveOpCode)
	}

	to := o.To()
	if to != testingOperation.to {
		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", testingOperation.to, to)
	}

	value := o.Value().Int64()
	if value != testingOperation.value.Int64() {
		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", testingOperation.value.Int64(), value)
	}
}

func TestBalanceOfOperationDecode(t *testing.T) {
	t.Parallel()

	op, err := DecodeBytes(wantBalanceOfOperation)
	if err != nil {
		t.Fatal(err)
	}

	o, ok := op.(BalanceOfOperation)
	if !ok {
		t.Fatalf("invalid operation type")
	}

	std := o.Standard()
	if std != 0 {
		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", 0, std)
	}

	haveOpCode := o.OpCode()
	wantOpCode, err := GetOpCode(wantBalanceOfOperation)
	if err != nil {
		switch err {
		case ErrRawDataShort, ErrPrefixNotValid:
			t.Logf("error: %+v", err)
		default:
			t.Fatal(err)
		}
	}

	if haveOpCode != wantOpCode {
		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", wantOpCode, haveOpCode)
	}

	address := o.Address()
	if address != testingOperation.tokenAddress {
		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", testingOperation.tokenAddress, address)
	}

	owner := o.Owner()
	if owner != testingOperation.owner {
		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", testingOperation.owner, owner)
	}
}

func TestPropertiesOperationDecode(t *testing.T) {
	t.Parallel()

	op, err := DecodeBytes(wantPropertiesOperation)
	if err != nil {
		t.Fatal(err)
	}

	o, ok := op.(PropertiesOperation)
	if !ok {
		t.Fatalf("invalid operation type")
	}

	std := o.Standard()
	if std != 0 {
		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", testingOperation.op.Std, std)
	}

	haveOpCode := o.OpCode()
	wantOpCode, err := GetOpCode(wantPropertiesOperation)
	if err != nil {
		switch err {
		case ErrRawDataShort, ErrPrefixNotValid:
			t.Logf("error: %+v", err)
		default:
			t.Fatal(err)
		}
	}

	if haveOpCode != wantOpCode {
		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", wantOpCode, haveOpCode)
	}

	address := o.Address()
	if address != testingOperation.tokenAddress {
		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", testingOperation.tokenAddress, address)
	}

	id, ok := o.TokenId()
	if !ok {
		t.Logf("tokenId is empty")
	}
	if id.Int64() != testingOperation.tokenId.Id.Int64() {
		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", testingOperation.tokenId.Id.Int64(), id.Int64())
	}
}
