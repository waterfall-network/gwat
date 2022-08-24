package token

import (
	"math/big"
	"testing"

	"github.com/waterfall-foundation/gwat/common"
	"github.com/waterfall-foundation/gwat/core/rawdb"
	"github.com/waterfall-foundation/gwat/core/state"
	"github.com/waterfall-foundation/gwat/core/vm"
	"github.com/waterfall-foundation/gwat/internal/token/testutils"
	"github.com/waterfall-foundation/gwat/token/operation"
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

	operator = common.BytesToAddress(testutils.RandomData(20))
	address = common.BytesToAddress(testutils.RandomData(20))
	spender = common.BytesToAddress(testutils.RandomData(20))
	owner = common.BytesToAddress(testutils.RandomData(20))
	to = common.BytesToAddress(testutils.RandomData(20))
	approveAddress = common.BytesToAddress(testutils.RandomData(20))

	caller = vm.AccountRef(owner)

	value = big.NewInt(int64(testutils.RandomInt(10, 30)))
	id = big.NewInt(int64(testutils.RandomInt(1000, 99999999)))
	id2 = big.NewInt(int64(testutils.RandomInt(1000, 99999999)))
	id3 = big.NewInt(int64(testutils.RandomInt(1000, 99999999)))
	id4 = big.NewInt(int64(testutils.RandomInt(1000, 99999999)))
	id5 = big.NewInt(int64(testutils.RandomInt(1000, 99999999)))
	id6 = big.NewInt(int64(testutils.RandomInt(1000, 99999999)))
	id7 = big.NewInt(int64(testutils.RandomInt(1000, 99999999)))
	totalSupply = big.NewInt(int64(testutils.RandomInt(100, 1000)))

	decimals = uint8(testutils.RandomInt(0, 255))

	name = testutils.RandomStringInBytes(testutils.RandomInt(10, 20))
	symbol = testutils.RandomStringInBytes(testutils.RandomInt(5, 8))
	baseURI = testutils.RandomStringInBytes(testutils.RandomInt(20, 40))

	data = testutils.RandomData(testutils.RandomInt(20, 50))
}

func TestProcessorCreateOperationWRC20Call(t *testing.T) {
	createOpWrc20, err := operation.NewWrc20CreateOperation(name, symbol, &decimals, totalSupply)
	if err != nil {
		t.Fatal(err)
	}

	cases := []testutils.TestCase{
		{
			CaseName: "Correct test WRC20",
			TestData: testutils.TestData{
				Caller:       vm.AccountRef(owner),
				TokenAddress: common.Address{},
			},
			Errs: []error{nil},
			Fn: func(c *testutils.TestCase, a *common.Address) {
				v := c.TestData.(testutils.TestData)
				adr := call(t, v.Caller, v.TokenAddress, createOpWrc20, c.Errs)
				*a = common.BytesToAddress(adr)

				balance := checkBalance(t, wrc20Address, owner)
				if balance.Cmp(totalSupply) != 0 {
					t.Fatal()
				}
			},
		},
		{
			CaseName: "WRC20 non empty token address",
			TestData: testutils.TestData{
				Caller:       vm.AccountRef(owner),
				TokenAddress: address,
			},
			Errs: []error{ErrNotNilTo},
			Fn: func(c *testutils.TestCase, a *common.Address) {
				v := c.TestData.(testutils.TestData)
				call(t, v.Caller, v.TokenAddress, createOpWrc20, c.Errs)
			},
		},
	}

	for _, c := range cases {
		t.Run(c.CaseName, func(t *testing.T) {
			c.Fn(&c, &wrc20Address)
		})
	}
}

func TestProcessorCreateOperationWRC721Call(t *testing.T) {
	createOpWrc721, err := operation.NewWrc721CreateOperation(name, symbol, baseURI)
	if err != nil {
		t.Fatal(err)
	}
	cases := []testutils.TestCase{
		{
			CaseName: "Correct test WRC721",
			TestData: testutils.TestData{
				Caller:       vm.AccountRef(owner),
				TokenAddress: common.Address{},
			},
			Errs: []error{nil},
			Fn: func(c *testutils.TestCase, a *common.Address) {
				v := c.TestData.(testutils.TestData)
				adr := call(t, v.Caller, v.TokenAddress, createOpWrc721, c.Errs)
				*a = common.BytesToAddress(adr)
			},
		},
		{
			CaseName: "WRC721 non empty token address",
			TestData: testutils.TestData{
				Caller:       vm.AccountRef(owner),
				TokenAddress: address,
			},
			Errs: []error{ErrNotNilTo},
			Fn: func(c *testutils.TestCase, a *common.Address) {
				v := c.TestData.(testutils.TestData)
				call(t, v.Caller, v.TokenAddress, createOpWrc721, c.Errs)
			},
		},
	}

	for _, c := range cases {
		t.Run(c.CaseName, func(t *testing.T) {
			c.Fn(&c, &wrc721Address)
		})
	}
}

func TestProcessorTransferFromOperationCall(t *testing.T) {
	opWrc20, err := operation.NewTransferFromOperation(operation.StdWRC20, owner, to, value)
	if err != nil {
		t.Fatal(err)
	}
	opWrc721, err := operation.NewTransferFromOperation(operation.StdWRC721, owner, to, id)
	if err != nil {
		t.Fatal(err)
	}

	cases := []testutils.TestCase{
		{
			CaseName: "Correct test WRC20",
			TestData: testutils.TestData{
				Caller:       vm.AccountRef(owner),
				TokenAddress: wrc20Address,
			},
			Errs: []error{nil},
			Fn: func(c *testutils.TestCase, a *common.Address) {
				v := c.TestData.(testutils.TestData)

				callApprove(t, operation.StdWRC20, spender, v.TokenAddress, v.Caller, value, c.Errs)

				call(t, vm.AccountRef(spender), v.TokenAddress, opWrc20, c.Errs)

				balance := checkBalance(t, wrc20Address, owner)

				var z, res big.Int
				if res.Sub(balance, z.Sub(totalSupply, value)).Cmp(big.NewInt(0)) != 0 {
					t.Fatal()
				}
			},
		},
		{
			CaseName: "Correct test WRC721",
			TestData: testutils.TestData{
				Caller:       vm.AccountRef(owner),
				TokenAddress: wrc721Address,
			},
			Errs: []error{nil},
			Fn: func(c *testutils.TestCase, a *common.Address) {
				v := c.TestData.(testutils.TestData)

				mintNewToken(t, owner, wrc721Address, id, data, caller, c.Errs)

				balance := checkBalance(t, wrc721Address, owner)

				callApprove(t, operation.StdWRC721, spender, v.TokenAddress, v.Caller, id, c.Errs)

				call(t, vm.AccountRef(spender), v.TokenAddress, opWrc721, c.Errs)

				balanceAfter := checkBalance(t, wrc721Address, owner)

				var res big.Int
				if res.Sub(balance, big.NewInt(1)).Cmp(balanceAfter) != 0 {
					t.Fatal()
				}

			},
		},
		{
			CaseName: "Wrong Caller",
			TestData: testutils.TestData{
				Caller:       vm.AccountRef(owner),
				TokenAddress: wrc721Address,
			},
			Errs: []error{ErrWrongCaller},
			Fn: func(c *testutils.TestCase, a *common.Address) {
				v := c.TestData.(testutils.TestData)

				callApprove(t, operation.StdWRC721, spender, v.TokenAddress, v.Caller, id, c.Errs)

				call(t, vm.AccountRef(spender), v.TokenAddress, opWrc721, c.Errs)
			},
		},
		{
			CaseName: "Not minted token",
			TestData: testutils.TestData{
				Caller:       vm.AccountRef(owner),
				TokenAddress: wrc721Address,
			},
			Errs: []error{ErrNotMinted},
			Fn: func(c *testutils.TestCase, a *common.Address) {
				v := c.TestData.(testutils.TestData)

				op, err := operation.NewTransferFromOperation(operation.StdWRC721, owner, to, id7)
				if err != nil {
					t.Fatal(err)
				}

				call(t, v.Caller, v.TokenAddress, op, c.Errs)
			},
		},
	}

	for _, c := range cases {
		t.Run(c.CaseName, func(t *testing.T) {
			c.Fn(&c, &common.Address{})
		})
	}
}

func TestProcessorMintOperationCall(t *testing.T) {
	mintOp, err := operation.NewMintOperation(owner, id2, data)
	if err != nil {
		t.Fatal(err)
	}

	cases := []testutils.TestCase{
		{
			CaseName: "Correct test",
			TestData: testutils.TestData{
				Caller:       vm.AccountRef(owner),
				TokenAddress: wrc721Address,
			},
			Errs: []error{nil},
			Fn: func(c *testutils.TestCase, a *common.Address) {
				v := c.TestData.(testutils.TestData)
				call(t, v.Caller, v.TokenAddress, mintOp, c.Errs)

				balance := checkBalance(t, wrc721Address, owner)

				if balance.Cmp(big.NewInt(1)) != 0 {
					t.Fatal()
				}
			},
		}, {
			CaseName: "Unknown minter",
			TestData: testutils.TestData{
				Caller:       vm.AccountRef(to),
				TokenAddress: wrc721Address,
			},
			Errs: []error{ErrWrongMinter},
			Fn: func(c *testutils.TestCase, a *common.Address) {
				v := c.TestData.(testutils.TestData)
				call(t, v.Caller, v.TokenAddress, mintOp, c.Errs)
			},
		},
	}

	for _, c := range cases {
		t.Run(c.CaseName, func(t *testing.T) {
			c.Fn(&c, &common.Address{})
		})
	}
}

func TestProcessorTransferOperationCall(t *testing.T) {
	transferOp, err := operation.NewTransferOperation(to, value)
	if err != nil {
		t.Fatal(err)
	}

	cases := []testutils.TestCase{
		{
			CaseName: "Correct transfer test",
			TestData: testutils.TestData{
				Caller:       vm.AccountRef(owner),
				TokenAddress: wrc20Address,
			},
			Errs: []error{nil},
			Fn: func(c *testutils.TestCase, a *common.Address) {
				v := c.TestData.(testutils.TestData)
				balance := checkBalance(t, wrc20Address, owner)

				call(t, v.Caller, v.TokenAddress, transferOp, c.Errs)

				balanceAfter := checkBalance(t, wrc20Address, owner)

				z := new(big.Int)
				if balanceAfter.Cmp(z.Sub(balance, value)) != 0 {
					t.Fatal()
				}
			},
		}, {
			CaseName: "No empty address",
			TestData: testutils.TestData{
				Caller:       vm.AccountRef(owner),
				TokenAddress: common.Address{},
			},
			Errs: []error{operation.ErrNoAddress},
			Fn: func(c *testutils.TestCase, a *common.Address) {
				v := c.TestData.(testutils.TestData)
				call(t, v.Caller, v.TokenAddress, transferOp, c.Errs)
			},
		},
		{
			CaseName: "Unknown Caller",
			TestData: testutils.TestData{
				Caller:       vm.AccountRef(address),
				TokenAddress: wrc20Address,
			},
			Errs: []error{ErrNotEnoughBalance},
			Fn: func(c *testutils.TestCase, a *common.Address) {
				v := c.TestData.(testutils.TestData)
				call(t, v.Caller, v.TokenAddress, transferOp, c.Errs)
			},
		},
	}

	for _, c := range cases {
		t.Run(c.CaseName, func(t *testing.T) {
			c.Fn(&c, &common.Address{})
		})
	}

}

func TestProcessorBurnOperationCall(t *testing.T) {
	burnOp, err := operation.NewBurnOperation(id3)
	if err != nil {
		t.Fatal(err)
	}

	cases := []testutils.TestCase{
		{
			CaseName: "Correct test",
			TestData: testutils.TestData{
				Caller:       vm.AccountRef(owner),
				TokenAddress: wrc721Address,
			},
			Errs: []error{nil},
			Fn: func(c *testutils.TestCase, a *common.Address) {
				v := c.TestData.(testutils.TestData)

				mintNewToken(t, owner, wrc721Address, id3, data, caller, c.Errs)

				balance := checkBalance(t, wrc721Address, owner)

				call(t, v.Caller, v.TokenAddress, burnOp, c.Errs)

				balanceAfter := checkBalance(t, wrc721Address, owner)

				z := new(big.Int)
				if balanceAfter.Cmp(z.Sub(balance, big.NewInt(1))) != 0 {
					t.Fatal()
				}
			},
		},
		{
			CaseName: "Unknown minter",
			TestData: testutils.TestData{
				Caller:       vm.AccountRef(to),
				TokenAddress: wrc721Address,
			},
			Errs: []error{ErrWrongMinter},
			Fn: func(c *testutils.TestCase, a *common.Address) {
				v := c.TestData.(testutils.TestData)
				call(t, v.Caller, v.TokenAddress, burnOp, c.Errs)
			},
		},
	}

	for _, c := range cases {
		t.Run(c.CaseName, func(t *testing.T) {
			c.Fn(&c, &common.Address{})
		})
	}
}

func TestProcessorApprovalForAllCall(t *testing.T) {
	op, err := operation.NewSetApprovalForAllOperation(spender, true)
	if err != nil {
		t.Fatal()
	}

	unapproveOp, err := operation.NewSetApprovalForAllOperation(spender, false)
	if err != nil {
		t.Fatal()
	}

	cases := []testutils.TestCase{
		{
			CaseName: "Use approvalForAll",
			TestData: testutils.TestData{
				Caller:       vm.AccountRef(owner),
				TokenAddress: wrc721Address,
			},
			Errs: []error{nil},
			Fn: func(c *testutils.TestCase, a *common.Address) {
				v := c.TestData.(testutils.TestData)

				call(t, v.Caller, v.TokenAddress, op, c.Errs)

				mintNewToken(t, owner, v.TokenAddress, id4, data, caller, c.Errs)

				callTransferFrom(t, operation.StdWRC721, owner, to, v.TokenAddress, id4, vm.AccountRef(spender), c.Errs)
			},
		},
		{
			CaseName: "Cancel approvalForAll",
			TestData: testutils.TestData{
				Caller:       vm.AccountRef(owner),
				TokenAddress: wrc721Address,
			},
			Errs: []error{nil, ErrWrongCaller},
			Fn: func(c *testutils.TestCase, a *common.Address) {
				v := c.TestData.(testutils.TestData)

				call(t, v.Caller, v.TokenAddress, op, c.Errs)

				mintNewToken(t, owner, v.TokenAddress, id5, data, caller, c.Errs)
				mintNewToken(t, owner, v.TokenAddress, id6, data, caller, c.Errs)

				callTransferFrom(t, operation.StdWRC721, owner, to, v.TokenAddress, id5, vm.AccountRef(spender), c.Errs)

				call(t, v.Caller, v.TokenAddress, unapproveOp, c.Errs)

				callTransferFrom(t, operation.StdWRC721, owner, to, v.TokenAddress, id6, vm.AccountRef(spender), c.Errs)
			},
		},
	}

	for _, c := range cases {
		t.Run(c.CaseName, func(t *testing.T) {
			c.Fn(&c, &common.Address{})
		})
	}
}

func TestProcessorIsApprovedForAll(t *testing.T) {
	cases := []testutils.TestCase{
		{
			CaseName: "IsApprovalForAll",
			TestData: testutils.TestData{
				Caller:       vm.AccountRef(owner),
				TokenAddress: wrc721Address,
			},
			Errs: []error{nil},
			Fn: func(c *testutils.TestCase, a *common.Address) {
				v := c.TestData.(testutils.TestData)
				approvalOp, err := operation.NewSetApprovalForAllOperation(operator, true)
				if err != nil {
					t.Fatal(err)
				}

				call(t, v.Caller, v.TokenAddress, approvalOp, c.Errs)

				ok := checkApprove(t, wrc721Address, owner, operator)

				if !ok {
					t.Fatal()
				}
			},
		},
		{
			CaseName: "IsNotApprovalForAll",
			TestData: testutils.TestData{
				Caller:       vm.AccountRef(owner),
				TokenAddress: wrc721Address,
			},
			Errs: []error{nil},
			Fn: func(c *testutils.TestCase, a *common.Address) {
				ok := checkApprove(t, wrc721Address, owner, spender)

				if ok {
					t.Fatal()
				}
			},
		},
	}
	for _, c := range cases {
		t.Run(c.CaseName, func(t *testing.T) {
			c.Fn(&c, &common.Address{})
		})
	}
}

func TestProcessorPropertiesWRC20(t *testing.T) {
	wrc20Op, err := operation.NewPropertiesOperation(wrc20Address, nil)
	if err != nil {
		t.Fatal(err)
	}

	i, err := p.Properties(wrc20Op)
	if err != nil {
		t.Fatal(err)
	}

	prop := i.(*WRC20PropertiesResult)
	testutils.CompareBytes(t, prop.Name, name)

	testutils.CompareBytes(t, prop.Symbol, symbol)

	testutils.CompareBigInt(t, prop.TotalSupply, totalSupply)

	if prop.Decimals != decimals {
		t.Fatalf("values do not match:\nwant: %+v\nhave: %+v", decimals, prop.Decimals)
	}
}

func TestProcessorPropertiesWRC721(t *testing.T) {
	mintNewToken(t, owner, wrc721Address, id7, data, caller, []error{nil})
	approveOp, err := operation.NewApproveOperation(operation.StdWRC721, spender, id7)
	call(t, vm.AccountRef(owner), wrc721Address, approveOp, []error{nil})

	wrc721Op, err := operation.NewPropertiesOperation(wrc721Address, id7)
	if err != nil {
		t.Fatal(err)
	}

	i, err := p.Properties(wrc721Op)
	if err != nil {
		t.Fatal(err)
	}

	prop := i.(*WRC721PropertiesResult)
	testutils.CompareBytes(t, prop.Name, name)

	testutils.CompareBytes(t, prop.Symbol, symbol)

	testutils.CompareBytes(t, prop.BaseURI, baseURI)

	testutils.CompareBytes(t, prop.Metadata, data)

	testutils.CompareBytes(t, prop.TokenURI, concatTokenURI(baseURI, id7))

	if prop.OwnerOf != owner {
		t.Fatal()
	}

	if prop.GetApproved != spender {
		t.Fatal()
	}
}

func TestProcessorApproveCall(t *testing.T) {
	approveOp, err := operation.NewApproveOperation(operation.StdWRC20, approveAddress, value)
	if err != nil {
		t.Fatal()
	}

	cases := []testutils.TestCase{
		{
			CaseName: "Use approve",
			TestData: testutils.TestData{
				Caller:       vm.AccountRef(owner),
				TokenAddress: wrc20Address,
			},
			Errs: []error{nil},
			Fn: func(c *testutils.TestCase, a *common.Address) {
				v := c.TestData.(testutils.TestData)

				call(t, v.Caller, v.TokenAddress, approveOp, c.Errs)

				allowanceOp, err := operation.NewAllowanceOperation(wrc20Address, owner, approveAddress)

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
			CaseName: "Non approved address",
			TestData: testutils.TestData{
				Caller:       vm.AccountRef(owner),
				TokenAddress: wrc20Address,
			},
			Errs: []error{nil, ErrWrongCaller},
			Fn: func(c *testutils.TestCase, a *common.Address) {
				allowanceOp, err := operation.NewAllowanceOperation(wrc20Address, owner, to)
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
		t.Run(c.CaseName, func(t *testing.T) {
			c.Fn(&c, &common.Address{})
		})
	}
}

func checkBalance(t *testing.T, TokenAddress, owner common.Address) *big.Int {
	balanceOp, err := operation.NewBalanceOfOperation(TokenAddress, owner)
	if err != nil {
		t.Fatal(err)
	}

	balance, err := p.BalanceOf(balanceOp)
	if err != nil {
		t.Fatal(err)
	}

	return balance
}

func mintNewToken(t *testing.T, owner, TokenAddress common.Address, id *big.Int, data []byte, Caller Ref, Errs []error) {
	mintOp, err := operation.NewMintOperation(owner, id, data)
	if err != nil {
		t.Fatal(err)
	}

	call(t, Caller, TokenAddress, mintOp, Errs)
}

func call(t *testing.T, Caller Ref, TokenAddress common.Address, op operation.Operation, Errs []error) []byte {
	res, err := p.Call(Caller, TokenAddress, op)
	if !testutils.CheckError(err, Errs) {
		t.Fatalf("Case failed\nwant errors: %s\nhave errors: %s", Errs, err)
	}

	return res
}

func checkApprove(t *testing.T, TokenAddress, owner, operator common.Address) bool {
	op, err := operation.NewIsApprovedForAllOperation(TokenAddress, owner, operator)
	if err != nil {
		t.Fatal(err)
	}

	ok, err := p.IsApprovedForAll(op)
	if err != nil {
		t.Fatal(err)
	}

	return ok
}

func callApprove(t *testing.T, std operation.Std, spender, TokenAddress common.Address, Caller Ref, value *big.Int, Errs []error) {
	approveOp, err := operation.NewApproveOperation(std, spender, value)
	if err != nil {
		t.Fatal(err)
	}

	call(t, Caller, TokenAddress, approveOp, Errs)
}

func callTransferFrom(
	t *testing.T,
	std operation.Std,
	owner, to, TokenAddress common.Address,
	id *big.Int,
	Caller Ref,
	Errs []error,
) {
	transferOp, err := operation.NewTransferFromOperation(std, owner, to, id)
	if err != nil {
		t.Fatal(err)
	}

	call(t, Caller, TokenAddress, transferOp, Errs)
}
