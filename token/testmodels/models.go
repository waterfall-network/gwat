package testmodels

import (
	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/vm"
)

type TestCase struct {
	CaseName string
	TestData interface{}
	Errs     []error
	Fn       func(c *TestCase, a *common.Address)
}

type TestData struct {
	Caller       vm.AccountRef
	TokenAddress common.Address
}
