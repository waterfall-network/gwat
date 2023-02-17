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
	"github.com/stretchr/testify/assert"
	"gitlab.waterfall.network/waterfall/protocol/gwat/internal/token/testutils"
	"gitlab.waterfall.network/waterfall/protocol/gwat/token"
	"gitlab.waterfall.network/waterfall/protocol/gwat/token/operation"
	"math/big"
	"testing"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/rawdb"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/state"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/types"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/vm"
	"gitlab.waterfall.network/waterfall/protocol/gwat/params"
)

var (
	from, to                               common.Address
	value, gasPrice, gasFreeCap, gasTipCap *big.Int
	gas                                    uint64
	data                                   []byte
	name                                   []byte
	symbol                                 []byte
	baseURI                                []byte
	totalSupply                            *big.Int
	decimals                               uint8
	percentFee                             uint8
)

func init() {
	from = common.BytesToAddress(testutils.RandomData(20))
	to = common.BytesToAddress(testutils.RandomData(20))
	value = big.NewInt(int64(testutils.RandomInt(10, 30)))
	totalSupply = big.NewInt(int64(testutils.RandomInt(100, 1000)))
	decimals = uint8(testutils.RandomInt(0, 255))
	name = testutils.RandomStringInBytes(testutils.RandomInt(10, 20))
	symbol = testutils.RandomStringInBytes(testutils.RandomInt(5, 8))
	baseURI = testutils.RandomStringInBytes(testutils.RandomInt(10, 20))
	gasPrice = big.NewInt(100)
	gasFreeCap = big.NewInt(0)
	gasTipCap = big.NewInt(0)
	gas = 25200
	percentFee = 5
}

func TestTransitionDb(t *testing.T) {
	// set up a test scenario
	stateDB, err := state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
	stateDB.CreateAccount(from)
	stateDB.SetBalance(from, big.NewInt(10000000000000000))
	if err != nil {
		panic("cannot create state DB")
	}

	op, err := operation.NewWrc20CreateOperation(name, symbol, &decimals, totalSupply)
	if err != nil {
		panic("cannot create wrc20 token")
	}
	binaryData, err := op.MarshalBinary()
	if err != nil {
		panic("cannot marshal create op")
	}
	data := append([]byte{operation.Prefix, operation.CreateCode}, binaryData...)

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
		tp  = token.NewProcessor(vm.BlockContext{}, stateDB)
		msg = NewMockMessage(from, nil, value, gas, data)
		st  = NewStateTransition(evm, tp, msg, new(GasPool).AddGas(50000000))
	)

	st.value = value
	result, err := st.TransitionDb()
	assert.NoError(t, err, err)
	assert.NotNil(t, result)
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

func NewMockMessage(from common.Address, to *common.Address, value *big.Int, gas uint64, data []byte) Message {
	return &MockMessage{
		from, to, value, gas, data,
	}
}
