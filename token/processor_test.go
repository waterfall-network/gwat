package token

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"math/big"
	"testing"
)

var (
	stateDb       *state.StateDB
	p             *Processor
	wrc20Address  = common.Address{}
	wrc721Address = common.Address{}
	caller        = vm.AccountRef(owner)
	operator      = common.HexToAddress("13e4acefe6a6700604929946e70e6443e4e73447")
	address       = common.HexToAddress("d049bfd667cb46aa3ef5df0da3e57db3be39e511")
	spender       = common.HexToAddress("2cccf5e0538493c235d1c5ef6580f77d99e91396")
	owner         = common.HexToAddress("2e068e8bd9e38e801a592fc61118e66d29d1124c")
	to            = common.HexToAddress("7dc9c9730689ff0b0fd506c67db815f12d90a448")
	from          = common.HexToAddress("7986bad81f4cbd9317f5a46861437dae58d69113")
	value         = big.NewInt(10)
	id            = big.NewInt(2221)
	id2           = big.NewInt(2222)
	id3           = big.NewInt(2223)
	id4           = big.NewInt(2224)
	totalSupply   = big.NewInt(100)
	index         = big.NewInt(20)
	decimals      = uint8(5)
	name          = []byte("Test Tokken")
	symbol        = []byte("TT")
	baseURI       = []byte("test.token.com")
	data          = []byte{243, 12, 202, 20, 133, 116, 111, 107, 101, 110, 116, 100, 5}
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
				adr, err := p.Call(v.caller, v.tokenAddress, createOpWrc20)
				if !checkError(err, c.errs) {
					t.Fatalf("Case failed\nwant errors: %s\nhave errors: %s", c.errs, err)
				}
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
				_, err := p.Call(v.caller, v.tokenAddress, createOpWrc20)
				if !checkError(err, c.errs) {
					t.Fatalf("Case failed\nwant errors: %s\nhave errors: %s", c.errs, err)
				}
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
				adr, err := p.Call(v.caller, v.tokenAddress, createOpWrc721)
				if !checkError(err, c.errs) {
					t.Fatalf("Case failed\nwant errors: %s\nhave errors: %s", c.errs, err)
				}
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
				_, err := p.Call(v.caller, v.tokenAddress, createOpWrc721)
				if !checkError(err, c.errs) {
					t.Fatalf("Case failed\nwant errors: %s\nhave errors: %s", c.errs, err)
				}
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
				approveOp, err := NewApproveOperation(StdWRC20, spender, value)
				if err != nil {
					t.Fatal(err)
				}
				_, err = p.Call(v.caller, v.tokenAddress, approveOp)
				if !checkError(err, c.errs) {
					t.Fatalf("Case failed\nwant errors: %s\nhave errors: %s", c.errs, err)
				}

				_, err = p.Call(vm.AccountRef(spender), v.tokenAddress, opWrc20)
				if !checkError(err, c.errs) {
					t.Fatalf("Case failed\nwant errors: %s\nhave errors: %s", c.errs, err)
				}

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

				approveOp, err := NewApproveOperation(StdWRC721, spender, id)
				if err != nil {
					t.Fatal(err)
				}
				_, err = p.Call(v.caller, v.tokenAddress, approveOp)
				if !checkError(err, c.errs) {
					t.Fatalf("Case failed\nwant errors: %s\nhave errors: %s", c.errs, err)
				}

				_, err = p.Call(vm.AccountRef(spender), v.tokenAddress, opWrc721)
				if !checkError(err, c.errs) {
					t.Fatalf("Case failed\nwant errors: %s\nhave errors: %s", c.errs, err)
				}

				balanceAfter := checkBalance(t, wrc721Address, owner)

				var res big.Int
				if res.Sub(balance, big.NewInt(1)).Cmp(balanceAfter) != 0 {
					t.Fatal()
				}

			},
		},
		{
			caseName: "Unknown caller",
			testData: testData{
				caller:       vm.AccountRef(owner),
				tokenAddress: wrc721Address,
			},
			errs: []error{ErrWrongCaller},
			fn: func(c *test, a *common.Address) {
				v := c.testData.(testData)

				approveOp, err := NewApproveOperation(StdWRC721, spender, id)
				if err != nil {
					t.Fatal(err)
				}
				_, err = p.Call(v.caller, v.tokenAddress, approveOp)
				if !checkError(err, c.errs) {
					t.Fatalf("Case failed\nwant errors: %s\nhave errors: %s", c.errs, err)
				}

				_, err = p.Call(vm.AccountRef(spender), v.tokenAddress, opWrc721)
				if !checkError(err, c.errs) {
					t.Fatalf("Case failed\nwant errors: %s\nhave errors: %s", c.errs, err)
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
				_, err = p.Call(v.caller, v.tokenAddress, mintOp)
				if !checkError(err, c.errs) {
					t.Fatalf("Case failed\nwant errors: %s\nhave errors: %s", c.errs, err)
				}

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
				_, err = p.Call(v.caller, v.tokenAddress, mintOp)
				if !checkError(err, c.errs) {
					t.Fatalf("Case failed\nwant errors: %s\nhave errors: %s", c.errs, err)
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

				_, err = p.Call(v.caller, v.tokenAddress, transferOp)
				if !checkError(err, c.errs) {
					t.Fatalf("Case failed\nwant errors: %s\nhave errors: %s", c.errs, err)
				}

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
				_, err = p.Call(v.caller, v.tokenAddress, transferOp)
				if !checkError(err, c.errs) {
					t.Fatalf("Case failed\nwant errors: %s\nhave errors: %s", c.errs, err)
				}
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
				_, err = p.Call(v.caller, v.tokenAddress, transferOp)
				if !checkError(err, c.errs) {
					t.Fatalf("Case failed\nwant errors: %s\nhave errors: %s", c.errs, err)
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

func TestProcessorBurnOperationCall(t *testing.T) {
	burnOp, err := NewBurnOperation(id4)
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

				mintNewToken(t, owner, wrc721Address, id4, data, caller, c.errs)

				balance := checkBalance(t, wrc721Address, owner)

				_, err = p.Call(v.caller, v.tokenAddress, burnOp)
				if !checkError(err, c.errs) {
					t.Fatalf("Case failed\nwant errors: %s\nhave errors: %s", c.errs, err)
				}

				balanceAfter := checkBalance(t, wrc721Address, owner)

				z := new(big.Int)
				if balanceAfter.Cmp(z.Sub(balance, big.NewInt(1))) != 0 {
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
				_, err = p.Call(v.caller, v.tokenAddress, burnOp)
				if !checkError(err, c.errs) {
					t.Fatalf("Case failed\nwant errors: %s\nhave errors: %s", c.errs, err)
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

//TODO complete this test
//func TestProcessorApprovalForAllOperation(t *testing.T) {
//	op, err := NewSetApprovalForAllOperation(spender, true)
//	if err != nil {
//		t.Fatal()
//	}
//
//	cases := []test{
//		{
//			caseName: "Correct test",
//			testData: testData{
//				caller:       vm.AccountRef(owner),
//				tokenAddress: wrc721Address,
//			},
//			errs: []error{nil},
//			fn: func(c *test, a *common.Address) {
//				v := c.testData.(testData)
//
//				mintNewToken(t, owner, v.tokenAddress, id3, data, caller, c.errs)
//
//				_, err = p.Call(v.caller, v.tokenAddress, op)
//				if !checkError(err, c.errs) {
//					t.Fatalf("Case failed\nwant errors: %s\nhave errors: %s", c.errs, err)
//				}
//
//				transferOp, err := NewTransferFromOperation(StdWRC721, owner, to, id3)
//				if err != nil {
//					t.Fatal(err)
//				}
//
//				_, err = p.Call(vm.AccountRef(spender), v.tokenAddress, transferOp)
//				if !checkError(err, c.errs) {
//					t.Fatalf("Case failed\nwant errors: %s\nhave errors: %s", c.errs, err)
//				}
//
//				transferOp2, err := NewTransferFromOperation(StdWRC721, owner, from, id3)
//				if err != nil {
//					t.Fatal(err)
//				}
//
//				//
//				//burnOp, err := NewBurnOperation(id3)
//
//				_, err = p.Call(vm.AccountRef(spender), v.tokenAddress, transferOp2)
//				if !checkError(err, c.errs) {
//					t.Fatalf("Case failed\nwant errors: %s\nhave errors: %s", c.errs, err)
//				}
//
//			},
//		},
//		//{
//		//	caseName: "Unknown minter",
//		//	testData: testData{
//		//		caller:       vm.AccountRef(to),
//		//		tokenAddress: wrc721Address,
//		//	},
//		//	errs: []error{ErrWrongMinter},
//		//	fn: func(c *test, a *common.Address) {
//		//		v := c.testData.(testData)
//		//		_, err = p.Call(v.caller, v.tokenAddress, burnOp)
//		//		if !checkError(err, c.errs) {
//		//			t.Fatalf("Case failed\nwant errors: %s\nhave errors: %s", c.errs, err)
//		//		}
//		//	},
//		//},
//	}
//
//	for _, c := range cases {
//		t.Run(c.caseName, func(t *testing.T) {
//			c.fn(&c, &common.Address{})
//		})
//	}
//}

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

	_, err = p.Call(caller, tokenAddress, mintOp)
	if !checkError(err, errs) {
		t.Fatalf("Case failed\nwant errors: %s\nhave errors: %s", errs, err)
	}
}
