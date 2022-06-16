package token

import (
	"bytes"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"math/big"
	"math/rand"
	"testing"
	"time"
)

var (
	stateDb        *state.StateDB
	p              *Processor
	wrc20Address   common.Address
	wrc721Address  common.Address
	caller         Ref
	operator       common.Address
	address        common.Address
	spender        common.Address
	owner          common.Address
	to             common.Address
	approveAddress common.Address
	value          *big.Int
	id             *big.Int
	id2            *big.Int
	id3            *big.Int
	id4            *big.Int
	id5            *big.Int
	id6            *big.Int
	id7            *big.Int
	totalSupply    *big.Int
	decimals       uint8
	name           []byte
	symbol         []byte
	baseURI        []byte
	data           []byte
)

func init() {
	stateDb, _ = state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
	ctx := vm.BlockContext{
		CanTransfer: nil,
		Transfer:    nil,
		Coinbase:    common.Address{},
		BlockNumber: new(big.Int).SetUint64(8000000),
		Time:        new(big.Int).SetUint64(5),
		Difficulty:  big.NewInt(0x30000),
		GasLimit:    uint64(6000000),
	}

	p = NewProcessor(ctx, stateDb)

	operator = common.BytesToAddress(randomData(20))
	address = common.BytesToAddress(randomData(20))
	spender = common.BytesToAddress(randomData(20))
	owner = common.BytesToAddress(randomData(20))
	to = common.BytesToAddress(randomData(20))
	approveAddress = common.BytesToAddress(randomData(20))

	caller = vm.AccountRef(owner)

	value = big.NewInt(int64(randomInt(10, 30)))
	id = big.NewInt(int64(randomInt(1000, 99999999)))
	id2 = big.NewInt(int64(randomInt(1000, 99999999)))
	id3 = big.NewInt(int64(randomInt(1000, 99999999)))
	id4 = big.NewInt(int64(randomInt(1000, 99999999)))
	id5 = big.NewInt(int64(randomInt(1000, 99999999)))
	id6 = big.NewInt(int64(randomInt(1000, 99999999)))
	id7 = big.NewInt(int64(randomInt(1000, 99999999)))
	totalSupply = big.NewInt(int64(randomInt(100, 1000)))

	decimals = uint8(randomInt(0, 255))

	name = randomStringInBytes(randomInt(10, 20))
	symbol = randomStringInBytes(randomInt(5, 8))
	baseURI = randomStringInBytes(randomInt(20, 40))

	data = randomData(randomInt(20, 50))
}

type test struct {
	caseName string
	testData interface{}
	errs     []error
	fn       func(c *test, a *common.Address)
}

type testData struct {
	caller       vm.AccountRef
	tokenAddress common.Address
}

func TestProcessorCreateOperationWRC20Call(t *testing.T) {
	createOpWrc20, err := NewWrc20CreateOperation(name, symbol, &decimals, totalSupply)
	if err != nil {
		t.Fatal(err)
	}

	cases := []test{
		{
			caseName: "Correct test WRC20",
			testData: testData{
				caller:       vm.AccountRef(owner),
				tokenAddress: common.Address{},
			},
			errs: []error{nil},
			fn: func(c *test, a *common.Address) {
				v := c.testData.(testData)
				adr := call(t, v.caller, v.tokenAddress, createOpWrc20, c.errs)
				*a = common.BytesToAddress(adr)

				balance := checkBalance(t, wrc20Address, owner)
				if balance.Cmp(totalSupply) != 0 {
					t.Fatal()
				}
			},
		},
		{
			caseName: "WRC20 non empty token address",
			testData: testData{
				caller:       vm.AccountRef(owner),
				tokenAddress: address,
			},
			errs: []error{ErrNotNilTo},
			fn: func(c *test, a *common.Address) {
				v := c.testData.(testData)
				call(t, v.caller, v.tokenAddress, createOpWrc20, c.errs)
			},
		},
	}

	for _, c := range cases {
		t.Run(c.caseName, func(t *testing.T) {
			c.fn(&c, &wrc20Address)
		})
	}
}

func TestProcessorCreateOperationWRC721Call(t *testing.T) {
	createOpWrc721, err := NewWrc721CreateOperation(name, symbol, baseURI)
	if err != nil {
		t.Fatal(err)
	}
	cases := []test{
		{
			caseName: "Correct test WRC721",
			testData: testData{
				caller:       vm.AccountRef(owner),
				tokenAddress: common.Address{},
			},
			errs: []error{nil},
			fn: func(c *test, a *common.Address) {
				v := c.testData.(testData)
				adr := call(t, v.caller, v.tokenAddress, createOpWrc721, c.errs)
				*a = common.BytesToAddress(adr)
			},
		},
		{
			caseName: "WRC721 non empty token address",
			testData: testData{
				caller:       vm.AccountRef(owner),
				tokenAddress: address,
			},
			errs: []error{ErrNotNilTo},
			fn: func(c *test, a *common.Address) {
				v := c.testData.(testData)
				call(t, v.caller, v.tokenAddress, createOpWrc721, c.errs)
			},
		},
	}

	for _, c := range cases {
		t.Run(c.caseName, func(t *testing.T) {
			c.fn(&c, &wrc721Address)
		})
	}
}

func TestProcessorTransferFromOperationCall(t *testing.T) {
	opWrc20, err := NewTransferFromOperation(StdWRC20, owner, to, value)
	if err != nil {
		t.Fatal(err)
	}
	opWrc721, err := NewTransferFromOperation(StdWRC721, owner, to, id)
	if err != nil {
		t.Fatal(err)
	}

	cases := []test{
		{
			caseName: "Correct test WRC20",
			testData: testData{
				caller:       vm.AccountRef(owner),
				tokenAddress: wrc20Address,
			},
			errs: []error{nil},
			fn: func(c *test, a *common.Address) {
				v := c.testData.(testData)

				callApprove(t, StdWRC20, spender, v.tokenAddress, v.caller, value, c.errs)

				call(t, vm.AccountRef(spender), v.tokenAddress, opWrc20, c.errs)

				balance := checkBalance(t, wrc20Address, owner)

				var z, res big.Int
				if res.Sub(balance, z.Sub(totalSupply, value)).Cmp(big.NewInt(0)) != 0 {
					t.Fatal()
				}
			},
		},
		{
			caseName: "Correct test WRC721",
			testData: testData{
				caller:       vm.AccountRef(owner),
				tokenAddress: wrc721Address,
			},
			errs: []error{nil},
			fn: func(c *test, a *common.Address) {
				v := c.testData.(testData)

				mintNewToken(t, owner, wrc721Address, id, data, caller, c.errs)

				balance := checkBalance(t, wrc721Address, owner)

				callApprove(t, StdWRC721, spender, v.tokenAddress, v.caller, id, c.errs)

				call(t, vm.AccountRef(spender), v.tokenAddress, opWrc721, c.errs)

				balanceAfter := checkBalance(t, wrc721Address, owner)

				var res big.Int
				if res.Sub(balance, big.NewInt(1)).Cmp(balanceAfter) != 0 {
					t.Fatal()
				}

			},
		},
		{
			caseName: "Wrong caller",
			testData: testData{
				caller:       vm.AccountRef(owner),
				tokenAddress: wrc721Address,
			},
			errs: []error{ErrWrongCaller},
			fn: func(c *test, a *common.Address) {
				v := c.testData.(testData)

				callApprove(t, StdWRC721, spender, v.tokenAddress, v.caller, id, c.errs)

				call(t, vm.AccountRef(spender), v.tokenAddress, opWrc721, c.errs)
			},
		},
		{
			caseName: "Not minted token",
			testData: testData{
				caller:       vm.AccountRef(owner),
				tokenAddress: wrc721Address,
			},
			errs: []error{ErrNotMinted},
			fn: func(c *test, a *common.Address) {
				v := c.testData.(testData)

				op, err := NewTransferFromOperation(StdWRC721, owner, to, id7)
				if err != nil {
					t.Fatal(err)
				}

				call(t, v.caller, v.tokenAddress, op, c.errs)
			},
		},
	}

	for _, c := range cases {
		t.Run(c.caseName, func(t *testing.T) {
			c.fn(&c, &common.Address{})
		})
	}
}

func TestProcessorMintOperationCall(t *testing.T) {
	mintOp, err := NewMintOperation(owner, id2, data)
	if err != nil {
		t.Fatal(err)
	}

	cases := []test{
		{
			caseName: "Correct test",
			testData: testData{
				caller:       vm.AccountRef(owner),
				tokenAddress: wrc721Address,
			},
			errs: []error{nil},
			fn: func(c *test, a *common.Address) {
				v := c.testData.(testData)
				call(t, v.caller, v.tokenAddress, mintOp, c.errs)

				balance := checkBalance(t, wrc721Address, owner)

				if balance.Cmp(big.NewInt(1)) != 0 {
					t.Fatal()
				}
			},
		}, {
			caseName: "Unknown minter",
			testData: testData{
				caller:       vm.AccountRef(to),
				tokenAddress: wrc721Address,
			},
			errs: []error{ErrWrongMinter},
			fn: func(c *test, a *common.Address) {
				v := c.testData.(testData)
				call(t, v.caller, v.tokenAddress, mintOp, c.errs)
			},
		},
	}

	for _, c := range cases {
		t.Run(c.caseName, func(t *testing.T) {
			c.fn(&c, &common.Address{})
		})
	}
}

func TestProcessorTransferOperationCall(t *testing.T) {
	transferOp, err := NewTransferOperation(to, value)
	if err != nil {
		t.Fatal(err)
	}

	cases := []test{
		{
			caseName: "Correct transfer test",
			testData: testData{
				caller:       vm.AccountRef(owner),
				tokenAddress: wrc20Address,
			},
			errs: []error{nil},
			fn: func(c *test, a *common.Address) {
				v := c.testData.(testData)
				balance := checkBalance(t, wrc20Address, owner)

				call(t, v.caller, v.tokenAddress, transferOp, c.errs)

				balanceAfter := checkBalance(t, wrc20Address, owner)

				z := new(big.Int)
				if balanceAfter.Cmp(z.Sub(balance, value)) != 0 {
					t.Fatal()
				}
			},
		}, {
			caseName: "No empty address",
			testData: testData{
				caller:       vm.AccountRef(owner),
				tokenAddress: common.Address{},
			},
			errs: []error{ErrNoAddress},
			fn: func(c *test, a *common.Address) {
				v := c.testData.(testData)
				call(t, v.caller, v.tokenAddress, transferOp, c.errs)
			},
		},
		{
			caseName: "Unknown caller",
			testData: testData{
				caller:       vm.AccountRef(address),
				tokenAddress: wrc20Address,
			},
			errs: []error{ErrNotEnoughBalance},
			fn: func(c *test, a *common.Address) {
				v := c.testData.(testData)
				call(t, v.caller, v.tokenAddress, transferOp, c.errs)
			},
		},
	}

	for _, c := range cases {
		t.Run(c.caseName, func(t *testing.T) {
			c.fn(&c, &common.Address{})
		})
	}

}

func TestProcessorBurnOperationCall(t *testing.T) {
	burnOp, err := NewBurnOperation(id3)
	if err != nil {
		t.Fatal(err)
	}

	cases := []test{
		{
			caseName: "Correct test",
			testData: testData{
				caller:       vm.AccountRef(owner),
				tokenAddress: wrc721Address,
			},
			errs: []error{nil},
			fn: func(c *test, a *common.Address) {
				v := c.testData.(testData)

				mintNewToken(t, owner, wrc721Address, id3, data, caller, c.errs)

				balance := checkBalance(t, wrc721Address, owner)

				call(t, v.caller, v.tokenAddress, burnOp, c.errs)

				balanceAfter := checkBalance(t, wrc721Address, owner)

				z := new(big.Int)
				if balanceAfter.Cmp(z.Sub(balance, big.NewInt(1))) != 0 {
					t.Fatal()
				}
			},
		},
		{
			caseName: "Unknown minter",
			testData: testData{
				caller:       vm.AccountRef(to),
				tokenAddress: wrc721Address,
			},
			errs: []error{ErrWrongMinter},
			fn: func(c *test, a *common.Address) {
				v := c.testData.(testData)
				call(t, v.caller, v.tokenAddress, burnOp, c.errs)
			},
		},
	}

	for _, c := range cases {
		t.Run(c.caseName, func(t *testing.T) {
			c.fn(&c, &common.Address{})
		})
	}
}

func TestProcessorApprovalForAllCall(t *testing.T) {
	op, err := NewSetApprovalForAllOperation(spender, true)
	if err != nil {
		t.Fatal()
	}

	unapproveOp, err := NewSetApprovalForAllOperation(spender, false)
	if err != nil {
		t.Fatal()
	}

	cases := []test{
		{
			caseName: "Use approvalForAll",
			testData: testData{
				caller:       vm.AccountRef(owner),
				tokenAddress: wrc721Address,
			},
			errs: []error{nil},
			fn: func(c *test, a *common.Address) {
				v := c.testData.(testData)

				call(t, v.caller, v.tokenAddress, op, c.errs)

				mintNewToken(t, owner, v.tokenAddress, id4, data, caller, c.errs)

				callTransferFrom(t, StdWRC721, owner, to, v.tokenAddress, id4, vm.AccountRef(spender), c.errs)
			},
		},
		{
			caseName: "Cancel approvalForAll",
			testData: testData{
				caller:       vm.AccountRef(owner),
				tokenAddress: wrc721Address,
			},
			errs: []error{nil, ErrWrongCaller},
			fn: func(c *test, a *common.Address) {
				v := c.testData.(testData)

				call(t, v.caller, v.tokenAddress, op, c.errs)

				mintNewToken(t, owner, v.tokenAddress, id5, data, caller, c.errs)
				mintNewToken(t, owner, v.tokenAddress, id6, data, caller, c.errs)

				callTransferFrom(t, StdWRC721, owner, to, v.tokenAddress, id5, vm.AccountRef(spender), c.errs)

				call(t, v.caller, v.tokenAddress, unapproveOp, c.errs)

				callTransferFrom(t, StdWRC721, owner, to, v.tokenAddress, id6, vm.AccountRef(spender), c.errs)
			},
		},
	}

	for _, c := range cases {
		t.Run(c.caseName, func(t *testing.T) {
			c.fn(&c, &common.Address{})
		})
	}
}

func TestProcessorIsApprovedForAll(t *testing.T) {
	cases := []test{
		{
			caseName: "IsApprovalForAll",
			testData: testData{
				caller:       vm.AccountRef(owner),
				tokenAddress: wrc721Address,
			},
			errs: []error{nil},
			fn: func(c *test, a *common.Address) {
				v := c.testData.(testData)
				approvalOp, err := NewSetApprovalForAllOperation(operator, true)
				if err != nil {
					t.Fatal(err)
				}

				call(t, v.caller, v.tokenAddress, approvalOp, c.errs)

				ok := checkApprove(t, wrc721Address, owner, operator)

				if !ok {
					t.Fatal()
				}
			},
		},
		{
			caseName: "IsNotApprovalForAll",
			testData: testData{
				caller:       vm.AccountRef(owner),
				tokenAddress: wrc721Address,
			},
			errs: []error{nil},
			fn: func(c *test, a *common.Address) {
				ok := checkApprove(t, wrc721Address, owner, spender)

				if ok {
					t.Fatal()
				}
			},
		},
	}
	for _, c := range cases {
		t.Run(c.caseName, func(t *testing.T) {
			c.fn(&c, &common.Address{})
		})
	}
}

func TestProcessorPropertiesWRC20(t *testing.T) {
	wrc20Op, err := NewPropertiesOperation(wrc20Address, nil)
	if err != nil {
		t.Fatal(err)
	}

	i, err := p.Properties(wrc20Op)
	if err != nil {
		t.Fatal(err)
	}

	prop := i.(*WRC20PropertiesResult)
	compareBytes(t, prop.Name, name)

	compareBytes(t, prop.Symbol, symbol)

	compareBigInt(t, prop.TotalSupply, totalSupply)

	if prop.Decimals != decimals {
		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", decimals, prop.Decimals)
	}
}

func TestProcessorPropertiesWRC721(t *testing.T) {
	mintNewToken(t, owner, wrc721Address, id7, data, caller, []error{nil})
	approveOp, err := NewApproveOperation(StdWRC721, spender, id7)
	call(t, vm.AccountRef(owner), wrc721Address, approveOp, []error{nil})

	wrc721Op, err := NewPropertiesOperation(wrc721Address, id7)
	if err != nil {
		t.Fatal(err)
	}

	i, err := p.Properties(wrc721Op)
	if err != nil {
		t.Fatal(err)
	}

	prop := i.(*WRC721PropertiesResult)
	compareBytes(t, prop.Name, name)

	compareBytes(t, prop.Symbol, symbol)

	compareBytes(t, prop.BaseURI, baseURI)

	compareBytes(t, prop.Metadata, data)

	compareBytes(t, prop.TokenURI, concatTokenURI(baseURI, id7))

	if prop.OwnerOf != owner {
		t.Fatal()
	}

	if prop.GetApproved != spender {
		t.Fatal()
	}
}

func TestProcessorApproveCall(t *testing.T) {
	approveOp, err := NewApproveOperation(StdWRC20, approveAddress, value)
	if err != nil {
		t.Fatal()
	}

	cases := []test{
		{
			caseName: "Use approve",
			testData: testData{
				caller:       vm.AccountRef(owner),
				tokenAddress: wrc20Address,
			},
			errs: []error{nil},
			fn: func(c *test, a *common.Address) {
				v := c.testData.(testData)

				call(t, v.caller, v.tokenAddress, approveOp, c.errs)

				allowanceOp, err := NewAllowanceOperation(wrc20Address, owner, approveAddress)

				total, err := p.Allowance(allowanceOp)
				if err != nil {
					t.Fatal(err)
				}

				if total.Cmp(value) != 0 {
					t.Fatal()
				}
			},
		},
		{
			caseName: "Non approved address",
			testData: testData{
				caller:       vm.AccountRef(owner),
				tokenAddress: wrc20Address,
			},
			errs: []error{nil, ErrWrongCaller},
			fn: func(c *test, a *common.Address) {
				allowanceOp, err := NewAllowanceOperation(wrc20Address, owner, to)
				if err != nil {
					t.Fatal(err)
				}

				total, err := p.Allowance(allowanceOp)
				if err != nil {
					t.Fatal(err)
				}

				if total.Cmp(big.NewInt(0)) != 0 {
					t.Fatal()
				}
			},
		},
	}

	for _, c := range cases {
		t.Run(c.caseName, func(t *testing.T) {
			c.fn(&c, &common.Address{})
		})
	}
}

func checkError(e error, arr []error) bool {
	for _, err := range arr {
		if e == err {
			return true
		}
	}

	return false
}

func checkBalance(t *testing.T, tokenAddress, owner common.Address) *big.Int {
	balanceOp, err := NewBalanceOfOperation(tokenAddress, owner)
	if err != nil {
		t.Fatal(err)
	}

	balance, err := p.BalanceOf(balanceOp)
	if err != nil {
		t.Fatal(err)
	}

	return balance
}

func mintNewToken(t *testing.T, owner, tokenAddress common.Address, id *big.Int, data []byte, caller Ref, errs []error) {
	mintOp, err := NewMintOperation(owner, id, data)
	if err != nil {
		t.Fatal(err)
	}

	call(t, caller, tokenAddress, mintOp, errs)
}

func call(t *testing.T, caller Ref, tokenAddress common.Address, op Operation, errs []error) []byte {
	res, err := p.Call(caller, tokenAddress, op)
	if !checkError(err, errs) {
		t.Fatalf("Case failed\nwant errors: %s\nhave errors: %s", errs, err)
	}

	return res
}

func compareBigInt(t *testing.T, a, b *big.Int) {
	haveValue := a
	wantValue := b

	if wantValue == nil {
		wantValue = big.NewInt(0)
	}

	if haveValue.Cmp(wantValue) != 0 {
		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", wantValue, haveValue)
	}
}

func compareBytes(t *testing.T, a, b []byte) {
	if !bytes.Equal(b, a) {
		t.Fatalf("values do not match:\n want: %+v\nhave: %+v", b, a)
	}
}

func callTransferFrom(
	t *testing.T,
	std Std,
	owner, to, tokenAddress common.Address,
	id *big.Int,
	caller Ref,
	errs []error,
) {
	transferOp, err := NewTransferFromOperation(std, owner, to, id)
	if err != nil {
		t.Fatal(err)
	}

	call(t, caller, tokenAddress, transferOp, errs)
}

func checkApprove(t *testing.T, tokenAddress, owner, operator common.Address) bool {
	op, err := NewIsApprovedForAllOperation(tokenAddress, owner, operator)
	if err != nil {
		t.Fatal(err)
	}

	ok, err := p.IsApprovedForAll(op)
	if err != nil {
		t.Fatal(err)
	}

	return ok
}

func callApprove(t *testing.T, std Std, spender, tokenAddress common.Address, caller Ref, value *big.Int, errs []error) {
	approveOp, err := NewApproveOperation(std, spender, value)
	if err != nil {
		t.Fatal(err)
	}

	call(t, caller, tokenAddress, approveOp, errs)
}

func randomInt(min, max int) int {
	rand.Seed(time.Now().UTC().UnixNano())
	a := rand.Intn(max-min+1) + min

	return a
}

func randomStringInBytes(l int) []byte {
	letters := []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")

	b := make([]byte, l)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}

	return b
}

func randomData(n int) []byte {
	b := make([]byte, n)
	rand.Read(b)

	return b
}
