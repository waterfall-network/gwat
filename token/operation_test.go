package token

import (
	"bytes"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"testing"
)

var testOpData = &opData{
	Std:        StdWRC721,
	Address:    common.Address{},
	TokenId:    big.NewInt(22222),
	Owner:      common.Address{},
	Spender:    common.Address{},
	Operator:   common.Address{},
	From:       common.Address{},
	To:         common.Address{},
	Value:      nil,
	Index:      nil,
	IsApproved: false,
	Data:       nil,
}

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

func TestMintOperation(t *testing.T) {
	t.Parallel()

	mintOp, err := NewMintOperation(
		testingOperation.to,
		testingOperation.tokenId.Id,
		testingOperation.data,
	)
	if err != nil {
		t.Fatal(err)
	}

	have, err := EncodeToBytes(mintOp)
	if err != nil {
		t.Fatalf("can`t encode operation %+v\nerror: %+v", mintOp, err)
	}

	if !bytes.Equal(wantMintOperation, have) {
		t.Fatalf("values do not match:\n want: %+v\nhave: %+v", wantMintOperation, have)
	}
}

func TestSetApprovalForAllOperation(t *testing.T) {
	t.Parallel()

	setApprovalForAllOp, err := NewSetApprovalForAllOperation(
		testingOperation.operator,
		testingOperation.isApproved,
	)
	if err != nil {
		t.Fatal(err)
	}

	have, err := EncodeToBytes(setApprovalForAllOp)
	if err != nil {
		t.Fatalf("can`t encode operation %+v\nerror: %+v", setApprovalForAllOp, err)
	}

	if !bytes.Equal(wantSetApprovalForAllOperation, have) {
		t.Fatalf("values do not match:\n want: %+v\nhave: %+v", wantSetApprovalForAllOperation, have)
	}
}

func TestIsApprovedForAllOperation(t *testing.T) {
	t.Parallel()

	isApprovedForAllOp, err := NewIsApprovedForAllOperation(
		testingOperation.tokenAddress,
		testingOperation.owner,
		testingOperation.operator,
	)
	if err != nil {
		t.Fatal(err)
	}

	have, err := EncodeToBytes(isApprovedForAllOp)
	if err != nil {
		t.Fatalf("can`t encode operation %+v\nerror: %+v", isApprovedForAllOp, err)
	}

	if !bytes.Equal(wantIsApprovedForAllOperation, have) {
		t.Fatalf("values do not match:\n want: %+v\nhave: %+v", wantIsApprovedForAllOperation, have)
	}
}

func TestAllowanceOperation(t *testing.T) {
	t.Parallel()

	allowanceOp, err := NewAllowanceOperation(
		testingOperation.tokenAddress,
		testingOperation.owner,
		testingOperation.spender,
	)
	if err != nil {
		t.Fatal(err)
	}

	have, err := EncodeToBytes(allowanceOp)
	if err != nil {
		t.Fatalf("can`t encode operation %+v\nerror: %+v", allowanceOp, err)
	}

	if !bytes.Equal(wantAllowanceOperation, have) {
		t.Fatalf("values do not match:\n want: %+v\nhave: %+v", wantAllowanceOperation, have)
	}
}

func TestApproveOperation(t *testing.T) {
	t.Parallel()

	approveOp, err := NewApproveOperation(
		testingOperation.op.Std,
		testingOperation.spender,
		testingOperation.value,
	)
	if err != nil {
		t.Fatal(err)
	}

	have, err := EncodeToBytes(approveOp)
	if err != nil {
		t.Fatalf("can`t encode operation %+v\nerror: %+v", approveOp, err)
	}

	if !bytes.Equal(wantApproveOperation, have) {
		t.Fatalf("values do not match:\n want: %+v\nhave: %+v", wantApproveOperation, have)
	}
}

func TestSafeTransferFromOperation(t *testing.T) {
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

func TestTransferFromOperation(t *testing.T) {
	t.Parallel()

	transferFromOp, err := NewTransferFromOperation(
		testingOperation.op.Std,
		testingOperation.from,
		testingOperation.to,
		testingOperation.value,
	)
	if err != nil {
		t.Fatal(err)
	}

	have, err := EncodeToBytes(transferFromOp)
	if err != nil {
		t.Fatalf("can`t encode operation %+v\nerror: %+v", transferFromOp, err)
	}

	if !bytes.Equal(wantTransferFromOperation, have) {
		t.Fatalf("values do not match:\n want: %+v\nhave: %+v", wantTransferFromOperation, have)
	}
}

func TestTransferOperation(t *testing.T) {
	t.Parallel()

	transferOp, err := NewTransferOperation(testingOperation.to, testingOperation.value)
	if err != nil {
		t.Fatal(err)
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
		t.Fatal(err)
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
		t.Fatal(err)
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

func TestPropertiesOperation(t *testing.T) {
	t.Parallel()

	propertiesOp, err := NewPropertiesOperation(
		testingOperation.tokenAddress,
		testingOperation.tokenId.Id,
	)
	if err != nil {
		t.Fatal(err)
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

	have, err := EncodeToBytes(tokenOfOwnerByIndexOp)
	if err != nil {
		t.Fatalf("can`t encode operation %+v\nerror: %+v", tokenOfOwnerByIndexOp, err)
	}

	if !bytes.Equal(have, wantTokenOfOwnerByIndexOperation) {
		t.Fatalf("values do not match: want: %+v\nhave: %+v", wantTokenOfOwnerByIndexOperation, have)
	}
}

//func TestCreateOperationDecode(t *testing.T) {
//	op, _ := DecodeBytes(wantBurnOperation)
//	s := op.Standard()
//	code := op.OpCode()
//
//}

//func TestTokenOfOwnerByIndexOperationDecode(t *testing.T) {
//	var have *tokenOfOwnerByIndexOperation
//	ok := testRlpDecode(wantTokenOfOwnerByIndexOperation, &have)
//	if !ok {
//		t.Errorf("can`t decode struct %+v", wantTokenOfOwnerByIndexOperation)
//	}
//
//	if have != testTokenOfOwnerByIndexOperation {
//		t.Errorf("mismutches types: have: %+v\nwant: %+v", have, testTokenOfOwnerByIndexOperation)
//	}
//}
