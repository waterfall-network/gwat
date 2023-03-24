// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package core

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/rawdb"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/state"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/types"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/vm"
	"gitlab.waterfall.network/waterfall/protocol/gwat/params"
	"gitlab.waterfall.network/waterfall/protocol/gwat/tests/testutils"
	"gitlab.waterfall.network/waterfall/protocol/gwat/token"
	"gitlab.waterfall.network/waterfall/protocol/gwat/token/operation"
)

var (
	stateTransition                        *StateTransition
	tokenProcessor                         *token.Processor
	stateDB                                vm.StateDB
	owner, spender, to                     common.Address
	WRC20Address, WRC721Address            common.Address
	caller                                 token.Ref
	value, gasPrice, gasFreeCap, gasTipCap *big.Int
	gas                                    uint64
	data                                   []byte
	name                                   []byte
	symbol                                 []byte
	baseURI                                []byte
	metadata                               []byte
	totalSupply                            *big.Int
	decimals                               uint8
	percentFee                             uint8
	ID1, ID2, ID3                          *big.Int
)

func init() {
	owner = common.BytesToAddress(testutils.RandomData(20))
	spender = common.BytesToAddress(testutils.RandomData(20))
	caller = vm.AccountRef(owner)
	to = common.BytesToAddress(testutils.RandomData(20))
	value = big.NewInt(int64(testutils.RandomInt(10, 30)))
	totalSupply = big.NewInt(int64(testutils.RandomInt(100, 1000)))
	decimals = uint8(testutils.RandomInt(0, 255))
	name = testutils.RandomStringInBytes(testutils.RandomInt(10, 20))
	symbol = testutils.RandomStringInBytes(testutils.RandomInt(5, 8))
	baseURI = testutils.RandomStringInBytes(testutils.RandomInt(10, 20))
	metadata = testutils.RandomData(testutils.RandomInt(20, 50))
	ID1 = big.NewInt(int64(testutils.RandomInt(1000, 99999999)))
	ID2 = big.NewInt(int64(testutils.RandomInt(1000, 99999999)))
	ID3 = big.NewInt(int64(testutils.RandomInt(1000, 99999999)))
	gasPrice = big.NewInt(100)
	gasFreeCap = big.NewInt(0)
	gasTipCap = big.NewInt(0)
	gas = 25200
	percentFee = 5

	db, err := state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
	if err != nil {
		panic("cannot create state DB")
	}
	db.CreateAccount(owner)
	db.SetBalance(owner, big.NewInt(10000000000000000))
	db.CreateAccount(spender)
	db.SetBalance(spender, big.NewInt(10000000000000000))
	db.CreateAccount(to)
	db.SetBalance(to, big.NewInt(10000000000000000))
	stateDB = db
	tokenProcessor = token.NewProcessor(vm.BlockContext{}, stateDB)
}

//func TestTransitionDb(t *testing.T) {
//	tests := []testutils.TestCase{
//		{
//			CaseName: "Create WRC20 token",
//			TestData: testutils.TestData{
//				Caller: vm.AccountRef(owner),
//			},
//			Errs: []error{nil},
//			Fn: func(c *testutils.TestCase, a *common.Address) {
//				st := initStateTransition(func(token common.Address) Message {
//					return NewMockMessage(owner, nil, value, gas, newTokenOpData(operation.CreateCode, func() []byte {
//						op, err := operation.NewWrc20CreateOperation(name, symbol, &decimals, totalSupply)
//						assert.NoError(t, err)
//						binaryData, err := op.MarshalBinary()
//						assert.NoError(t, err)
//						return binaryData
//					}))
//				}, nil)
//
//				result, _ := st.TransitionDb()
//				WRC20Address.SetBytes(result.ReturnData)
//				balance := checkBalance(t, st.tp, WRC20Address, owner)
//
//				assert.NoError(t, result.Err)
//				assert.NotNil(t, result.ReturnData)
//				testutils.BigIntEquals(balance, totalSupply)
//			},
//		},
//		{
//			CaseName: "Create WRC721 token",
//			TestData: testutils.TestData{
//				Caller:       vm.AccountRef(owner),
//				TokenAddress: common.Address{},
//			},
//			Errs: []error{nil},
//			Fn: func(c *testutils.TestCase, a *common.Address) {
//				st := initStateTransition(func(token common.Address) Message {
//					return NewMockMessage(owner, nil, value, gas, newTokenOpData(operation.CreateCode, func() []byte {
//						op, err := operation.NewWrc721CreateOperation(name, symbol, baseURI, &percentFee)
//						assert.NoError(t, err)
//						binaryData, err := op.MarshalBinary()
//						assert.NoError(t, err)
//						return binaryData
//					}))
//				}, nil)
//
//				result, _ := st.TransitionDb()
//				WRC721Address.SetBytes(result.ReturnData)
//				balance := checkBalance(t, st.tp, WRC721Address, owner)
//
//				assert.NoError(t, result.Err)
//				assert.NotNil(t, result.ReturnData)
//				testutils.BigIntEquals(balance, big.NewInt(0))
//			},
//		},
//		{
//			CaseName: "Properties of the WRC20 token",
//			TestData: testutils.TestData{
//				Caller: vm.AccountRef(owner),
//			},
//			Errs: []error{nil},
//			Fn: func(c *testutils.TestCase, a *common.Address) {
//				st := initStateTransition(func(token common.Address) Message {
//					return NewMockMessage(owner, &token, value, gas, newTokenOpData(operation.PropertiesCode, func() []byte {
//						op, err := operation.NewPropertiesOperation(WRC20Address, nil)
//						assert.NoError(t, err)
//						binaryData, err := op.MarshalBinary()
//						assert.NoError(t, err)
//						return binaryData
//					}))
//				}, nil)
//
//				result, _ := st.TransitionDb()
//
//				assert.NoError(t, result.Err)
//				assert.Nil(t, result.ReturnData)
//			},
//		},
//		{
//			CaseName: "Properties of the WRC721 token",
//			TestData: testutils.TestData{
//				Caller:       vm.AccountRef(owner),
//				TokenAddress: common.Address{},
//			},
//			Errs: []error{nil},
//			Fn: func(c *testutils.TestCase, a *common.Address) {
//				st := initStateTransition(func(token common.Address) Message {
//					return NewMockMessage(owner, &token, value, gas, newTokenOpData(operation.PropertiesCode, func() []byte {
//						op, err := operation.NewPropertiesOperation(WRC721Address, nil)
//						assert.NoError(t, err)
//						binaryData, err := op.MarshalBinary()
//						assert.NoError(t, err)
//						return binaryData
//					}))
//				}, nil)
//
//				result, _ := st.TransitionDb()
//
//				assert.NoError(t, result.Err)
//				assert.Nil(t, result.ReturnData)
//			},
//		},
//		{
//			CaseName: "Balance of the WRC20 token",
//			TestData: testutils.TestData{
//				Caller:       vm.AccountRef(owner),
//				TokenAddress: common.Address{},
//			},
//			Errs: []error{nil},
//			Fn: func(c *testutils.TestCase, a *common.Address) {
//				st := initStateTransition(func(token common.Address) Message {
//					return NewMockMessage(owner, &token, value, gas, newTokenOpData(operation.BalanceOfCode, func() []byte {
//						op, err := operation.NewBalanceOfOperation(WRC20Address, owner)
//						assert.NoError(t, err)
//						binaryData, err := op.MarshalBinary()
//						assert.NoError(t, err)
//						return binaryData
//					}))
//				}, nil)
//
//				result, _ := st.TransitionDb()
//
//				assert.NoError(t, result.Err)
//				assert.Nil(t, result.ReturnData)
//			},
//		},
//		{
//			CaseName: "Balance of the WRC721 token",
//			TestData: testutils.TestData{
//				Caller:       vm.AccountRef(owner),
//				TokenAddress: common.Address{},
//			},
//			Errs: []error{nil},
//			Fn: func(c *testutils.TestCase, a *common.Address) {
//				st := initStateTransition(func(token common.Address) Message {
//					return NewMockMessage(owner, &token, value, gas, newTokenOpData(operation.BalanceOfCode, func() []byte {
//						op, err := operation.NewBalanceOfOperation(WRC721Address, owner)
//						assert.NoError(t, err)
//						binaryData, err := op.MarshalBinary()
//						assert.NoError(t, err)
//						return binaryData
//					}))
//				}, nil)
//
//				result, _ := st.TransitionDb()
//
//				assert.NoError(t, result.Err)
//				assert.Nil(t, result.ReturnData)
//			},
//		},
//		{
//			CaseName: "Transfer from of the WRC20 token",
//			TestData: testutils.TestData{
//				Caller:       vm.AccountRef(owner),
//				TokenAddress: common.Address{},
//			},
//			Errs: []error{nil},
//			Fn: func(c *testutils.TestCase, a *common.Address) {
//				transferAmount := big.NewInt(10)
//				st := initStateTransition(func(token common.Address) Message {
//					return NewMockMessage(spender, &token, value, gas, newTokenOpData(operation.TransferFromCode, func() []byte {
//						op, err := operation.NewTransferFromOperation(operation.StdWRC20, owner, to, transferAmount)
//						assert.NoError(t, err)
//						binaryData, err := op.MarshalBinary()
//						assert.NoError(t, err)
//						return binaryData
//					}))
//				}, func(tp *token.Processor) common.Address {
//					approve, _ := operation.NewApproveOperation(operation.StdWRC20, spender, transferAmount)
//					tp.Call(caller, WRC20Address, nil, approve)
//
//					return WRC20Address
//				})
//
//				result, _ := st.TransitionDb()
//				transferred := new(big.Int).SetBytes(result.ReturnData)
//
//				assert.NoError(t, result.Err)
//				assert.NotNil(t, result.ReturnData)
//				testutils.BigIntEquals(transferred, transferAmount)
//			},
//		},
//		{
//			CaseName: "Transfer from of the WRC721 token",
//			TestData: testutils.TestData{
//				Caller:       vm.AccountRef(owner),
//				TokenAddress: WRC721Address,
//			},
//			Errs: []error{nil},
//			Fn: func(c *testutils.TestCase, a *common.Address) {
//				v := c.TestData.(testutils.TestData)
//				st := initStateTransition(func(token common.Address) Message {
//					return NewMockMessage(spender, &token, value, gas, newTokenOpData(operation.TransferFromCode, func() []byte {
//						op, err := operation.NewTransferFromOperation(operation.StdWRC721, owner, to, ID1)
//						assert.NoError(t, err)
//						binaryData, err := op.MarshalBinary()
//						assert.NoError(t, err)
//						return binaryData
//					}))
//				}, func(tp *token.Processor) common.Address {
//					mintNewToken(t, tp, owner, WRC721Address, ID1, metadata, v.Caller, c.Errs)
//					callApprove(t, tp, operation.StdWRC721, spender, WRC721Address, v.Caller, ID1, c.Errs)
//					return WRC721Address
//				})
//
//				result, _ := st.TransitionDb()
//				transferred := new(big.Int).SetBytes(result.ReturnData)
//
//				assert.NoError(t, result.Err)
//				assert.NotNil(t, result.ReturnData)
//				testutils.BigIntEquals(transferred, ID1)
//			},
//		},
//		{
//			CaseName: "Transfer of the WRC20 token",
//			TestData: testutils.TestData{
//				Caller: vm.AccountRef(owner),
//			},
//			Errs: []error{nil},
//			Fn: func(c *testutils.TestCase, a *common.Address) {
//				transferAmount := big.NewInt(10)
//				st := initStateTransition(func(token common.Address) Message {
//					return NewMockMessage(owner, &WRC20Address, value, gas, newTokenOpData(operation.TransferCode, func() []byte {
//						op, err := operation.NewTransferOperation(to, transferAmount)
//						assert.NoError(t, err)
//						binaryData, err := op.MarshalBinary()
//						assert.NoError(t, err)
//						return binaryData
//					}))
//				}, nil)
//
//				result, _ := st.TransitionDb()
//				transferred := new(big.Int).SetBytes(result.ReturnData)
//
//				assert.NoError(t, result.Err)
//				assert.NotNil(t, result.ReturnData)
//				testutils.BigIntEquals(transferred, transferAmount)
//			},
//		},
//		{
//			CaseName: "Mint of the WRC721 token",
//			TestData: testutils.TestData{
//				Caller: vm.AccountRef(owner),
//			},
//			Errs: []error{nil},
//			Fn: func(c *testutils.TestCase, a *common.Address) {
//				st := initStateTransition(func(token common.Address) Message {
//					return NewMockMessage(owner, &WRC721Address, value, gas, newTokenOpData(operation.MintCode, func() []byte {
//						op, err := operation.NewMintOperation(owner, ID2, metadata)
//						assert.NoError(t, err)
//						binaryData, err := op.MarshalBinary()
//						assert.NoError(t, err)
//						return binaryData
//					}))
//				}, nil)
//
//				result, _ := st.TransitionDb()
//				minted := new(big.Int).SetBytes(result.ReturnData)
//
//				assert.NoError(t, result.Err)
//				assert.NotNil(t, result.ReturnData)
//				testutils.BigIntEquals(minted, ID2)
//			},
//		},
//		{
//			CaseName: "Burn of the WRC721 token",
//			TestData: testutils.TestData{
//				Caller:       vm.AccountRef(owner),
//				TokenAddress: common.Address{},
//			},
//			Errs: []error{nil},
//			Fn: func(c *testutils.TestCase, a *common.Address) {
//				st := initStateTransition(func(token common.Address) Message {
//					return NewMockMessage(owner, &token, value, gas, newTokenOpData(operation.BurnCode, func() []byte {
//						op, err := operation.NewBurnOperation(ID3)
//						assert.NoError(t, err)
//						binaryData, err := op.MarshalBinary()
//						assert.NoError(t, err)
//						return binaryData
//					}))
//				}, func(tp *token.Processor) common.Address {
//					mintNewToken(t, tp, owner, WRC721Address, ID3, metadata, caller, c.Errs)
//					return WRC721Address
//				})
//
//				result, _ := st.TransitionDb()
//				burned := new(big.Int).SetBytes(result.ReturnData)
//
//				assert.NoError(t, result.Err)
//				assert.NotNil(t, result.ReturnData)
//				testutils.BigIntEquals(burned, ID3)
//			},
//		},
//		{
//			CaseName: "Buy of the WRC20 token",
//			TestData: testutils.TestData{
//				Caller: vm.AccountRef(owner),
//			},
//			Errs: []error{nil},
//			Fn: func(c *testutils.TestCase, a *common.Address) {
//				v := c.TestData.(testutils.TestData)
//				pricePerToken := big.NewInt(25)
//				expTokenCount := big.NewInt(10)
//				expTotalPrice := big.NewInt(0).Mul(pricePerToken, expTokenCount)
//				value.Set(expTotalPrice).Add(value, big.NewInt(1))
//
//				st := initStateTransition(func(token common.Address) Message {
//					return NewMockMessage(spender, &token, expTotalPrice, gas, newTokenOpData(operation.BuyCode, func() []byte {
//						op, err := operation.NewBuyOperation(nil, nil)
//						assert.NoError(t, err)
//						binaryData, err := op.MarshalBinary()
//						assert.NoError(t, err)
//						return binaryData
//					}))
//				}, func(tp *token.Processor) common.Address {
//					setPrice(t, tp, v.Caller, WRC20Address, nil, pricePerToken)
//					return WRC20Address
//				})
//
//				result, _ := st.TransitionDb()
//				spenderBalance := checkBalance(t, st.tp, WRC20Address, spender)
//
//				assert.NoError(t, result.Err)
//				assert.NotNil(t, result.ReturnData)
//				testutils.BigIntEquals(spenderBalance, expTokenCount)
//			},
//		},
//		{
//			CaseName: "Buy of the WRC721 NFT",
//			TestData: testutils.TestData{
//				Caller: vm.AccountRef(owner),
//			},
//			Errs: []error{nil},
//			Fn: func(c *testutils.TestCase, a *common.Address) {
//				var tokenID *big.Int
//				tokenPrice := big.NewInt(30)
//				value.Set(tokenPrice).Add(value, big.NewInt(1))
//				st := initStateTransition(func(token common.Address) Message {
//					return NewMockMessage(spender, &token, value, gas, newTokenOpData(operation.BuyCode, func() []byte {
//						op, err := operation.NewBuyOperation(tokenID, nil)
//						assert.NoError(t, err)
//						binaryData, err := op.MarshalBinary()
//						assert.NoError(t, err)
//						return binaryData
//					}))
//				}, func(tp *token.Processor) common.Address {
//					tokenID = big.NewInt(int64(testutils.RandomInt(1000, 99999999)))
//					mintNewToken(t, tp, spender, WRC721Address, tokenID, data, caller, nil)
//
//					sellCaller := vm.AccountRef(spender)
//					setPrice(t, tp, sellCaller, WRC721Address, tokenID, tokenPrice)
//
//					return WRC721Address
//				})
//
//				result, _ := st.TransitionDb()
//				spenderBalance := checkBalance(t, st.tp, WRC721Address, spender)
//
//				assert.NoError(t, result.Err)
//				assert.NotNil(t, result.ReturnData)
//				testutils.BigIntEquals(spenderBalance, big.NewInt(1))
//			},
//		},
//		{
//			CaseName: "Set price of the WRC20 token",
//			TestData: testutils.TestData{
//				Caller: vm.AccountRef(owner),
//			},
//			Errs: []error{nil},
//			Fn: func(c *testutils.TestCase, a *common.Address) {
//				newPrice := big.NewInt(15)
//				st := initStateTransition(func(token common.Address) Message {
//					return NewMockMessage(owner, &WRC20Address, value, gas, newTokenOpData(operation.SetPriceCode, func() []byte {
//						op, err := operation.NewSetPriceOperation(nil, newPrice)
//						assert.NoError(t, err)
//						binaryData, err := op.MarshalBinary()
//						assert.NoError(t, err)
//						return binaryData
//					}))
//				}, nil)
//
//				result, _ := st.TransitionDb()
//				currentPrice, _ := checkCost(st.tp, WRC20Address, nil)
//
//				assert.NoError(t, result.Err)
//				assert.NotNil(t, result.ReturnData)
//				testutils.BigIntEquals(currentPrice, newPrice)
//			},
//		},
//		{
//			CaseName: "Set price of the WRC721 token",
//			TestData: testutils.TestData{
//				Caller:       vm.AccountRef(owner),
//				TokenAddress: common.Address{},
//			},
//			Errs: []error{nil},
//			Fn: func(c *testutils.TestCase, a *common.Address) {
//				newPrice := big.NewInt(15)
//				var tokenID *big.Int
//				st := initStateTransition(func(token common.Address) Message {
//					return NewMockMessage(owner, &WRC721Address, value, gas, newTokenOpData(operation.SetPriceCode, func() []byte {
//						op, err := operation.NewSetPriceOperation(tokenID, newPrice)
//						assert.NoError(t, err)
//						binaryData, err := op.MarshalBinary()
//						assert.NoError(t, err)
//						return binaryData
//					}))
//				}, func(tp *token.Processor) common.Address {
//					tokenID = big.NewInt(int64(testutils.RandomInt(1000, 99999999)))
//					mintNewToken(t, tp, owner, WRC721Address, tokenID, data, caller, c.Errs)
//
//					return WRC721Address
//				})
//
//				result, _ := st.TransitionDb()
//				currentPrice, _ := checkCost(st.tp, WRC721Address, tokenID)
//
//				assert.NoError(t, result.Err)
//				assert.NotNil(t, result.ReturnData)
//				testutils.BigIntEquals(currentPrice, newPrice)
//			},
//		},
//		{
//			CaseName: "Approve spending of the WRC20 token",
//			TestData: testutils.TestData{
//				Caller: vm.AccountRef(owner),
//			},
//			Errs: []error{nil},
//			Fn: func(c *testutils.TestCase, a *common.Address) {
//				approveAmount := big.NewInt(20)
//				st := initStateTransition(func(token common.Address) Message {
//					return NewMockMessage(owner, &WRC20Address, value, gas, newTokenOpData(operation.ApproveCode, func() []byte {
//						op, err := operation.NewApproveOperation(operation.StdWRC20, spender, approveAmount)
//						assert.NoError(t, err)
//						binaryData, err := op.MarshalBinary()
//						assert.NoError(t, err)
//						return binaryData
//					}))
//				}, nil)
//
//				result, _ := st.TransitionDb()
//				approvedAmount := new(big.Int).SetBytes(result.ReturnData)
//
//				assert.NoError(t, result.Err)
//				assert.NotNil(t, result.ReturnData)
//				testutils.BigIntEquals(approvedAmount, approveAmount)
//			},
//		},
//		{
//			CaseName: "Approve spending of the WRC721 token",
//			TestData: testutils.TestData{
//				Caller: vm.AccountRef(owner),
//			},
//			Errs: []error{nil},
//			Fn: func(c *testutils.TestCase, a *common.Address) {
//				var tokenID *big.Int
//				st := initStateTransition(func(token common.Address) Message {
//					return NewMockMessage(owner, &WRC721Address, value, gas, newTokenOpData(operation.ApproveCode, func() []byte {
//						op, err := operation.NewApproveOperation(operation.StdWRC721, spender, tokenID)
//						assert.NoError(t, err)
//						binaryData, err := op.MarshalBinary()
//						assert.NoError(t, err)
//						return binaryData
//					}))
//				}, func(tp *token.Processor) common.Address {
//					tokenID = big.NewInt(int64(testutils.RandomInt(1000, 99999999)))
//					mintNewToken(t, tp, owner, WRC721Address, tokenID, data, caller, c.Errs)
//
//					return WRC721Address
//				})
//
//				result, _ := st.TransitionDb()
//				approvedID := new(big.Int).SetBytes(result.ReturnData)
//
//				assert.NoError(t, result.Err)
//				assert.NotNil(t, result.ReturnData)
//				testutils.BigIntEquals(approvedID, tokenID)
//			},
//		},
//		{
//			CaseName: "Set approval for all WRC721 tokens",
//			TestData: testutils.TestData{
//				Caller: vm.AccountRef(owner),
//			},
//			Errs: []error{nil},
//			Fn: func(c *testutils.TestCase, a *common.Address) {
//				st := initStateTransition(func(token common.Address) Message {
//					return NewMockMessage(owner, &WRC721Address, value, gas, newTokenOpData(operation.SetApprovalForAllCode, func() []byte {
//						op, err := operation.NewSetApprovalForAllOperation(spender, true)
//						assert.NoError(t, err)
//						binaryData, err := op.MarshalBinary()
//						assert.NoError(t, err)
//						return binaryData
//					}))
//				}, nil)
//
//				result, _ := st.TransitionDb()
//				approvedSpender := common.BytesToAddress(result.ReturnData)
//
//				assert.NoError(t, result.Err)
//				assert.NotNil(t, result.ReturnData)
//				assert.Equal(t, spender, approvedSpender)
//			},
//		},
//	}
//
//	for _, c := range tests {
//		t.Run(c.CaseName, func(t *testing.T) {
//			c.Fn(&c, &common.Address{})
//		})
//	}
//}

func initStateTransition(m func(token common.Address) Message, init func(tp *token.Processor) common.Address) *StateTransition {
	t := func(init func(tp *token.Processor) common.Address) common.Address {
		if init != nil {
			return init(tokenProcessor)
		}

		return common.Address{}
	}(init)

	if stateTransition != nil {
		stateTransition.msg = m(t)
		return stateTransition
	}

	var (
		tracer = vm.NewStructLogger(&vm.LogConfig{})
		evm    = vm.NewEVM(vm.BlockContext{
			CanTransfer: func(db vm.StateDB, addr common.Address, amount *big.Int) bool {
				return db.GetBalance(addr).Cmp(amount) >= 0
			},
			Transfer: func(db vm.StateDB, sender, recipient common.Address, amount *big.Int) {
				db.SubBalance(sender, amount)
				db.AddBalance(recipient, amount)
			},
			BaseFee: big.NewInt(10),
		}, vm.TxContext{}, stateDB, params.TestChainConfig, vm.Config{NoBaseFee: true, Debug: true, Tracer: tracer})
		st = NewStateTransition(evm, tokenProcessor, nil, m(t), new(GasPool).AddGas(50000000))
	)

	stateTransition = st

	return stateTransition
}

func newTokenOpData(opCode byte, f func() []byte) []byte {
	return append([]byte{operation.Prefix, opCode}, f()...)
}

func callApprove(t *testing.T, tp *token.Processor, std operation.Std, spender, TokenAddress common.Address, Caller token.Ref, value *big.Int, Errs []error) {
	approveOp, err := operation.NewApproveOperation(std, spender, value)
	assert.NoError(t, err)

	call(t, tp, Caller, TokenAddress, nil, approveOp, Errs)
}

func call(t *testing.T, tp *token.Processor, Caller token.Ref, TokenAddress common.Address, value *big.Int, op operation.Operation, Errs []error) []byte {
	res, err := tp.Call(Caller, TokenAddress, value, op)
	if !testutils.CheckError(err, Errs) {
		t.Fatalf("Case failed\nwant errors: %s\nhave errors: %s", Errs, err)
	}

	return res
}

func mintNewToken(t *testing.T, tp *token.Processor, owner, TokenAddress common.Address, id *big.Int, data []byte, Caller token.Ref, Errs []error) {
	mintOp, err := operation.NewMintOperation(owner, id, data)
	assert.NoError(t, err)

	call(t, tp, Caller, TokenAddress, nil, mintOp, Errs)
}

func checkCost(tp *token.Processor, tokenAddress common.Address, tokenId *big.Int) (*big.Int, error) {
	costOp, err := operation.NewCostOperation(tokenAddress, tokenId)
	if err != nil {
		return nil, err
	}

	return tp.Cost(costOp)
}

func checkBalance(t *testing.T, tp *token.Processor, TokenAddress, owner common.Address) *big.Int {
	balanceOp, err := operation.NewBalanceOfOperation(TokenAddress, owner)
	assert.NoError(t, err)

	balance, err := tp.BalanceOf(balanceOp)
	assert.NoError(t, err)

	return balance
}

func setPrice(t *testing.T, tp *token.Processor, caller token.Ref, tokenAddress common.Address, tokenId, value *big.Int) {
	setPriceOp, err := operation.NewSetPriceOperation(tokenId, value)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	call(t, tp, caller, tokenAddress, nil, setPriceOp, nil)
}

type MockMessage struct {
	from  common.Address
	to    *common.Address
	value *big.Int
	gas   uint64
	data  []byte
}

func (m MockMessage) From() common.Address {
	return m.from
}

func (m MockMessage) To() *common.Address {
	return m.to
}

func (m MockMessage) GasPrice() *big.Int {
	return gasPrice
}

func (m MockMessage) GasFeeCap() *big.Int {
	return gasFreeCap
}

func (m MockMessage) GasTipCap() *big.Int {
	return gasTipCap
}

func (m MockMessage) Gas() uint64 {
	return m.gas
}

func (m MockMessage) Value() *big.Int {
	return m.value
}

func (m MockMessage) Nonce() uint64 {
	panic("Nonce: implement me")
}

func (m MockMessage) IsFake() bool {
	return true
}

func (m MockMessage) Data() []byte {
	return m.data
}

func (m MockMessage) AccessList() types.AccessList {
	return []types.AccessTuple{}
}

func (m MockMessage) setTo(addr *common.Address) {
	m.to = addr
}

func NewMockMessage(from common.Address, to *common.Address, value *big.Int, gas uint64, data []byte) *MockMessage {
	return &MockMessage{
		from, to, value, gas, data,
	}
}
