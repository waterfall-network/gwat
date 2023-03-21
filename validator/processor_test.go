package validator

import (
	"math/big"
	"testing"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/rawdb"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/state"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/vm"
	testUtils "gitlab.waterfall.network/waterfall/protocol/gwat/tests/testutils"
	"gitlab.waterfall.network/waterfall/protocol/gwat/validator/operation"
	"gitlab.waterfall.network/waterfall/protocol/gwat/validator/testmodels"
)

var (
	stateDb   *state.StateDB
	processor *Processor

	from               common.Address
	to                 common.Address
	value              *big.Int
	pubkey             common.BlsPubKey // validator public key
	creator_address    common.Address   // attached creator account
	withdrawal_address common.Address   // attached withdrawal credentials
	signature          common.BlsSignature

	depositAddress common.Address
	address        common.Address
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
	processor = NewProcessor(ctx, stateDb, testmodels.TestChainConfig)

	to = processor.GetValidatorsStateAddress()
	from = common.BytesToAddress(testUtils.RandomData(20))
	value = MinDepositVal
	pubkey = common.HexToBlsPubKey("0x9728bc733c8fcedde0c3a33dac12da3ebbaa0eb74d813a34b600520e7976a260d85f057687e8c923d52c78715515348d")
	creator_address = common.HexToAddress("0xa7e558cc6efa1c41270ef4aa227b3dd6b4a3951e")
	withdrawal_address = common.HexToAddress("0xa7e558cc6efa1c41270ef4aa227b3dd6b4a3951e")
	signature = common.HexToBlsSig("0xb9221f2308c1e1655a8e1977f32241384fa77efedbb3079bcc9a95930152ee87" +
		"f341134a4e59c3e312ee5c2197732ea30d9aac2993cc4aad75335009815d07a8735f96c6dde443ba3a10f5523c4d00f6b3a7b48af" +
		"5a42795183ab5aa2f1b2dd1")
}

func TestProcessorDeposit(t *testing.T) {
	depositOperation, err := operation.NewDepositOperation(pubkey, creator_address, withdrawal_address, signature)
	testUtils.AssertNoError(t, err)
	cases := []testmodels.TestCase{
		{
			CaseName: "Deposit: OK",
			TestData: testmodels.TestData{
				Caller: vm.AccountRef(from),
				AddrTo: common.Address{},
			},
			Errs: []error{nil},
			Fn: func(c *testmodels.TestCase, a *common.Address) {
				v := c.TestData.(testmodels.TestData)

				bal, _ := new(big.Int).SetString("32000000000000000000000", 10)
				processor.state.AddBalance(from, bal)

				balanceFromBfr := processor.state.GetBalance(from)

				adr := call(t, v.Caller, v.AddrTo, value, depositOperation, c.Errs)
				*a = common.BytesToAddress(adr)

				balanceFromAft := processor.state.GetBalance(from)
				balDif := new(big.Int).Sub(balanceFromBfr, balanceFromAft)
				if balDif.Cmp(value) != 0 {
					t.Errorf("Expected balance From ios bad : %d\nactual: %s", 1, balanceFromAft)
				}

			},
		},
		{
			CaseName: "Deposit: ErrTooLowDepositValue (val = nil)",
			TestData: testmodels.TestData{
				Caller: vm.AccountRef(from),
				AddrTo: address,
			},
			Errs: []error{ErrTooLowDepositValue},
			Fn: func(c *testmodels.TestCase, a *common.Address) {
				v := c.TestData.(testmodels.TestData)
				call(t, v.Caller, v.AddrTo, nil, depositOperation, c.Errs)
			},
		},
		{
			CaseName: "Deposit: ErrTooLowDepositValue (val = 1 wat)",
			TestData: testmodels.TestData{
				Caller: vm.AccountRef(from),
				AddrTo: address,
			},
			Errs: []error{ErrTooLowDepositValue},
			Fn: func(c *testmodels.TestCase, a *common.Address) {
				v := c.TestData.(testmodels.TestData)
				val1, _ := new(big.Int).SetString("1000000000000000000", 10)
				call(t, v.Caller, v.AddrTo, val1, depositOperation, c.Errs)
			},
		},
	}

	for _, c := range cases {
		t.Run(c.CaseName, func(t *testing.T) {
			c.Fn(&c, &depositAddress)
		})
	}
}

func call(t *testing.T, Caller Ref, addrTo common.Address, value *big.Int, op operation.Operation, Errs []error) []byte {
	res, err := processor.Call(Caller, addrTo, value, op)
	if !testUtils.CheckError(err, Errs) {
		t.Fatalf("Case failed\nwant errors: %s\nhave errors: %s", Errs, err)
	}

	return res
}
