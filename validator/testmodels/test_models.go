// Copyright 2024   Blue Wave Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package testmodels

import (
	"math/big"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/rawdb"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/vm"
	"gitlab.waterfall.network/waterfall/protocol/gwat/ethdb"
	"gitlab.waterfall.network/waterfall/protocol/gwat/params"
	"gitlab.waterfall.network/waterfall/protocol/gwat/tests/testutils"
	"gitlab.waterfall.network/waterfall/protocol/gwat/validator/era"
)

var (
	TestDb                 ethdb.Database
	TestChainConfig        *params.ChainConfig
	validatorsStateAddress common.Address
	TestEra                era.Era

	Addr1  = common.BytesToAddress(testutils.RandomStringInBytes(50))
	Addr2  = common.BytesToAddress(testutils.RandomStringInBytes(30))
	Addr3  = common.BytesToAddress(testutils.RandomStringInBytes(20))
	Addr4  = common.BytesToAddress(testutils.RandomStringInBytes(40))
	Addr5  = common.BytesToAddress(testutils.RandomStringInBytes(50))
	Addr6  = common.BytesToAddress(testutils.RandomStringInBytes(70))
	Addr7  = common.BytesToAddress(testutils.RandomStringInBytes(70))
	Addr8  = common.BytesToAddress(testutils.RandomStringInBytes(70))
	Addr9  = common.BytesToAddress(testutils.RandomStringInBytes(70))
	Addr10 = common.BytesToAddress(testutils.RandomStringInBytes(70))
	Addr11 = common.BytesToAddress(testutils.RandomStringInBytes(70))
	Addr12 = common.BytesToAddress(testutils.RandomStringInBytes(70))

	InputValidators = []common.Address{Addr1, Addr2, Addr3, Addr4, Addr5, Addr6, Addr7, Addr8, Addr9, Addr10, Addr11, Addr12}
)

func init() {
	validatorsStateAddress = common.BytesToAddress(testutils.RandomStringInBytes(70))

	TestDb = rawdb.NewMemoryDatabase()
	TestChainConfig = &params.ChainConfig{
		ChainID:                big.NewInt(111111),
		SecondsPerSlot:         4,
		SlotsPerEpoch:          32,
		ValidatorsStateAddress: &validatorsStateAddress,
		ValidatorsPerSlot:      2,
		EpochsPerEra:           22,
		TransitionPeriod:       2,
		ValidatorOpExpireSlots: 14400,
		ForkSlotSubNet1:        9999999,
		ForkSlotDelegate:       10,
		ForkSlotPrefixFin:      10,
		ForkSlotShanghai:       0,
		ForkSlotValOpTracking:  100,
		ForkSlotValSyncProc:    100,
		StartEpochsPerEra:      0,
		EffectiveBalance:       big.NewInt(3200),
	}

	TestEra = era.Era{
		Number: 7,
		From:   89,
		To:     110,
		Root:   common.BytesToHash(testutils.RandomData(32)),
	}
}

type TestCase struct {
	CaseName string
	TestData interface{}
	Errs     []error
	Fn       func(c *TestCase)
}

type TestData struct {
	Caller vm.AccountRef
	AddrTo common.Address
}
