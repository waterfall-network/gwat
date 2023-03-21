package testmodels

import (
	"math/big"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/rawdb"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/vm"
	"gitlab.waterfall.network/waterfall/protocol/gwat/ethdb"
	"gitlab.waterfall.network/waterfall/protocol/gwat/params"
	"gitlab.waterfall.network/waterfall/protocol/gwat/tests/testutils"
)

var (
	TestDb                 ethdb.Database
	TestChainConfig        *params.ChainConfig
	validatorsStateAddress common.Address

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
		ForkSlotSubNet1:        9999999,
		ValidatorsStateAddress: &validatorsStateAddress,
		ValidatorsPerSlot:      6,
	}
}

type TestCase struct {
	CaseName string
	TestData interface{}
	Errs     []error
	Fn       func(c *TestCase, a *common.Address)
}

type TestData struct {
	Caller vm.AccountRef
	AddrTo common.Address
}
