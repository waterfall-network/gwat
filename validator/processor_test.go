package validator

import (
	"math"
	"math/big"
	"testing"

	"github.com/golang/mock/gomock"
	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/rawdb"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/state"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/types"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/vm"
	"gitlab.waterfall.network/waterfall/protocol/gwat/tests/testutils"
	"gitlab.waterfall.network/waterfall/protocol/gwat/validator/era"
	"gitlab.waterfall.network/waterfall/protocol/gwat/validator/operation"
	"gitlab.waterfall.network/waterfall/protocol/gwat/validator/storage"
	"gitlab.waterfall.network/waterfall/protocol/gwat/validator/testmodels"
)

var (
	stateDb           *state.StateDB
	from              common.Address
	pubKey            common.BlsPubKey // validator public key
	creatorAddress    common.Address   // attached creator account
	creatorAddress2   common.Address   // attached creator account
	creatorAddress3   common.Address   // attached creator account
	creatorAddress4   common.Address   // attached creator account
	creatorAddress5   common.Address   // attached creator account
	creatorAddress6   common.Address   // attached creator account
	withdrawalAddress common.Address   // attached withdrawal credentials
	signature         common.BlsSignature
	ctrl              *gomock.Controller
	eraInfo           era.EraInfo

	value     = MinDepositVal
	procEpoch = uint64(100)
	ctx       = vm.BlockContext{
		CanTransfer: nil,
		Transfer:    nil,
		Coinbase:    common.Address{},
		BlockNumber: new(big.Int).SetUint64(8000000),
		Time:        new(big.Int).SetUint64(5),
		Difficulty:  big.NewInt(0x30000),
		GasLimit:    uint64(6000000),
	}
)

func init() {
	stateDb, _ = state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)

	from = common.BytesToAddress(testutils.RandomData(20))
	pubKey = common.BytesToBlsPubKey(testutils.RandomData(48))
	creatorAddress = common.BytesToAddress(testutils.RandomData(20))
	creatorAddress2 = common.BytesToAddress(testutils.RandomData(20))
	creatorAddress3 = common.BytesToAddress(testutils.RandomData(20))
	creatorAddress4 = common.BytesToAddress(testutils.RandomData(20))
	creatorAddress5 = common.BytesToAddress(testutils.RandomData(20))
	creatorAddress6 = common.BytesToAddress(testutils.RandomData(20))
	withdrawalAddress = common.BytesToAddress(testutils.RandomData(20))
	signature = common.BytesToBlsSig(testutils.RandomData(96))

	eraInfo = era.NewEraInfo(testmodels.TestEra)
}

func TestProcessorDeposit(t *testing.T) {
	ctrl = gomock.NewController(t)
	defer ctrl.Finish()

	bc := NewMockBlockchain(ctrl)
	processor := NewProcessor(ctx, stateDb, testmodels.TestChainConfig, bc)
	to := processor.GetValidatorsStateAddress()

	depositOperation, err := operation.NewDepositOperation(pubKey, creatorAddress, withdrawalAddress, signature)
	testutils.AssertNoError(t, err)
	cases := []*testmodels.TestCase{
		{
			CaseName: "Deposit: OK",
			TestData: testmodels.TestData{
				Caller: vm.AccountRef(from),
				AddrTo: to,
			},
			Errs: []error{nil},
			Fn: func(c *testmodels.TestCase) {
				v := c.TestData.(testmodels.TestData)

				bal, _ := new(big.Int).SetString("32000000000000000000000", 10)
				processor.state.AddBalance(from, bal)

				balanceFromBfr := processor.state.GetBalance(from)

				call(t, processor, v.Caller, v.AddrTo, value, depositOperation, c.Errs)

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
				AddrTo: to,
			},
			Errs: []error{ErrTooLowDepositValue},
			Fn: func(c *testmodels.TestCase) {
				v := c.TestData.(testmodels.TestData)
				call(t, processor, v.Caller, v.AddrTo, nil, depositOperation, c.Errs)
			},
		},
		{
			CaseName: "Deposit: ErrTooLowDepositValue (val = 1 wat)",
			TestData: testmodels.TestData{
				Caller: vm.AccountRef(from),
				AddrTo: to,
			},
			Errs: []error{ErrTooLowDepositValue},
			Fn: func(c *testmodels.TestCase) {
				v := c.TestData.(testmodels.TestData)
				val1, _ := new(big.Int).SetString("1000000000000000000", 10)
				call(t, processor, v.Caller, v.AddrTo, val1, depositOperation, c.Errs)
			},
		},
		{
			CaseName: "Deposit: invalid address to",
			TestData: testmodels.TestData{
				Caller: vm.AccountRef(from),
				AddrTo: from,
			},
			Errs: []error{ErrInvalidToAddress},
			Fn: func(c *testmodels.TestCase) {
				v := c.TestData.(testmodels.TestData)
				call(t, processor, v.Caller, v.AddrTo, nil, depositOperation, c.Errs)
			},
		},
	}

	for _, c := range cases {
		t.Run(c.CaseName, func(t *testing.T) {
			c.Fn(c)
		})
	}
}

func TestProcessorActivate(t *testing.T) {
	ctrl = gomock.NewController(t)
	defer ctrl.Finish()

	bc := NewMockBlockchain(ctrl)
	processor := NewProcessor(ctx, stateDb, testmodels.TestChainConfig, bc)
	to := processor.GetValidatorsStateAddress()
	activateOperation, err := operation.NewValidatorSyncOperation(types.Activate, procEpoch, 0, creatorAddress2, nil, &withdrawalAddress)
	testutils.AssertNoError(t, err)
	cases := []*testmodels.TestCase{
		{
			CaseName: "Activate: Unknown validator",
			TestData: testmodels.TestData{
				Caller: vm.AccountRef(from),
				AddrTo: to,
			},
			Errs: []error{ErrUnknownValidator},
			Fn: func(c *testmodels.TestCase) {
				v := c.TestData.(testmodels.TestData)
				call(t, processor, v.Caller, v.AddrTo, nil, activateOperation, c.Errs)
			},
		},
		{
			CaseName: "Activate: OK",
			TestData: testmodels.TestData{
				Caller: vm.AccountRef(from),
				AddrTo: to,
			},
			Errs: []error{nil},
			Fn: func(c *testmodels.TestCase) {
				bc.EXPECT().GetEraInfo().Return(&eraInfo)
				bc.EXPECT().GetConfig().Return(testmodels.TestChainConfig)

				v := c.TestData.(testmodels.TestData)

				validator := storage.NewValidator(creatorAddress2, &withdrawalAddress)

				buf, err := validator.MarshalBinary()
				testutils.AssertNoError(t, err)
				processor.Storage().SetValidatorInfo(processor.state, buf)

				valInfo := processor.storage.GetValidatorInfo(processor.state, creatorAddress2)
				if valInfo.GetValidatorIndex() != math.MaxUint64 {
					t.Fatal()
				}
				if valInfo.GetActivationEpoch() != math.MaxUint64 {
					t.Fatal()
				}

				call(t, processor, v.Caller, v.AddrTo, value, activateOperation, c.Errs)

				valInfo = processor.storage.GetValidatorInfo(processor.state, creatorAddress2)

				testutils.AssertEqual(t, valInfo.GetActivationEpoch(), testmodels.TestEra.To+1)
				testutils.AssertEqual(t, valInfo.GetValidatorIndex(), activateOperation.Index())

				valList := processor.Storage().GetValidatorsList(processor.state)
				if len(valList) > 1 {
					t.Fatal()
				}

				testutils.AssertEqual(t, validator.Address, valList[0])
			},
		},
	}

	for _, c := range cases {
		t.Run(c.CaseName, func(t *testing.T) {
			c.Fn(c)
		})
	}
}

func TestProcessorExit(t *testing.T) {
	ctrl = gomock.NewController(t)
	defer ctrl.Finish()

	bc := NewMockBlockchain(ctrl)
	processor := NewProcessor(ctx, stateDb, testmodels.TestChainConfig, bc)
	to := processor.GetValidatorsStateAddress()

	exitOperation, err := operation.NewExitOperation(pubKey, creatorAddress3, &procEpoch)
	testutils.AssertNoError(t, err)
	cases := []*testmodels.TestCase{
		{
			CaseName: "Exit: Unknown validator",
			TestData: testmodels.TestData{
				Caller: vm.AccountRef(from),
				AddrTo: processor.GetValidatorsStateAddress(),
			},
			Errs: []error{ErrUnknownValidator},
			Fn: func(c *testmodels.TestCase) {
				v := c.TestData.(testmodels.TestData)
				call(t, processor, v.Caller, v.AddrTo, nil, exitOperation, c.Errs)
			},
		},
		{
			CaseName: "Exit: invalid to address",
			TestData: testmodels.TestData{
				Caller: vm.AccountRef(from),
				AddrTo: creatorAddress,
			},
			Errs: []error{ErrInvalidToAddress},
			Fn: func(c *testmodels.TestCase) {
				v := c.TestData.(testmodels.TestData)
				call(t, processor, v.Caller, v.AddrTo, nil, exitOperation, c.Errs)
			},
		},
		{
			CaseName: "Exit: not activated validator",
			TestData: testmodels.TestData{
				Caller: vm.AccountRef(from),
				AddrTo: to,
			},
			Errs: []error{ErrNotActivatedValidator},
			Fn: func(c *testmodels.TestCase) {
				v := c.TestData.(testmodels.TestData)

				validator := storage.NewValidator(creatorAddress3, &withdrawalAddress)

				buf, err := validator.MarshalBinary()
				testutils.AssertNoError(t, err)
				processor.Storage().SetValidatorInfo(processor.state, buf)

				call(t, processor, v.Caller, v.AddrTo, value, exitOperation, c.Errs)
			},
		},
		{
			CaseName: "Exit: validator is out",
			TestData: testmodels.TestData{
				Caller: vm.AccountRef(from),
				AddrTo: to,
			},
			Errs: []error{ErrValidatorIsOut},
			Fn: func(c *testmodels.TestCase) {
				v := c.TestData.(testmodels.TestData)

				validator := storage.NewValidator(creatorAddress3, &withdrawalAddress)
				validator.ActivationEpoch = 0
				validator.ExitEpoch = procEpoch

				buf, err := validator.MarshalBinary()
				testutils.AssertNoError(t, err)
				processor.Storage().SetValidatorInfo(processor.state, buf)

				call(t, processor, v.Caller, v.AddrTo, value, exitOperation, c.Errs)
			},
		},
		{
			CaseName: "Exit: invalid from address",
			TestData: testmodels.TestData{
				Caller: vm.AccountRef(from),
				AddrTo: to,
			},
			Errs: []error{ErrInvalidFromAddresses},
			Fn: func(c *testmodels.TestCase) {
				v := c.TestData.(testmodels.TestData)

				validator := storage.NewValidator(creatorAddress3, &withdrawalAddress)
				validator.ActivationEpoch = 0

				buf, err := validator.MarshalBinary()
				testutils.AssertNoError(t, err)
				processor.Storage().SetValidatorInfo(processor.state, buf)

				call(t, processor, v.Caller, v.AddrTo, value, exitOperation, c.Errs)
			},
		},
		{
			CaseName: "Exit: OK",
			TestData: testmodels.TestData{
				Caller: vm.AccountRef(withdrawalAddress),
				AddrTo: to,
			},
			Errs: []error{nil},
			Fn: func(c *testmodels.TestCase) {
				v := c.TestData.(testmodels.TestData)

				validator := storage.NewValidator(creatorAddress3, &withdrawalAddress)
				validator.ActivationEpoch = 0

				buf, err := validator.MarshalBinary()
				testutils.AssertNoError(t, err)
				processor.Storage().SetValidatorInfo(processor.state, buf)

				call(t, processor, v.Caller, v.AddrTo, value, exitOperation, c.Errs)
			},
		},
	}

	for _, c := range cases {
		t.Run(c.CaseName, func(t *testing.T) {
			c.Fn(c)
		})
	}
}

func TestProcessorDeactivate(t *testing.T) {
	ctrl = gomock.NewController(t)
	defer ctrl.Finish()

	bc := NewMockBlockchain(ctrl)
	processor := NewProcessor(ctx, stateDb, testmodels.TestChainConfig, bc)
	to := processor.GetValidatorsStateAddress()

	exitOperation, err := operation.NewValidatorSyncOperation(types.Deactivate, procEpoch, 0, creatorAddress4, nil, &withdrawalAddress)
	testutils.AssertNoError(t, err)
	cases := []*testmodels.TestCase{
		{
			CaseName: "Deactivate: Unknown validator",
			TestData: testmodels.TestData{
				Caller: vm.AccountRef(from),
				AddrTo: processor.GetValidatorsStateAddress(),
			},
			Errs: []error{ErrUnknownValidator},
			Fn: func(c *testmodels.TestCase) {
				v := c.TestData.(testmodels.TestData)
				call(t, processor, v.Caller, v.AddrTo, nil, exitOperation, c.Errs)
			},
		},
		{
			CaseName: "Deactivate: not activated validator",
			TestData: testmodels.TestData{
				Caller: vm.AccountRef(from),
				AddrTo: to,
			},
			Errs: []error{ErrNotActivatedValidator},
			Fn: func(c *testmodels.TestCase) {
				v := c.TestData.(testmodels.TestData)

				validator := storage.NewValidator(creatorAddress4, &withdrawalAddress)

				buf, err := validator.MarshalBinary()
				testutils.AssertNoError(t, err)
				processor.Storage().SetValidatorInfo(processor.state, buf)

				call(t, processor, v.Caller, v.AddrTo, value, exitOperation, c.Errs)
			},
		},
		{
			CaseName: "Deactivate: validator is out",
			TestData: testmodels.TestData{
				Caller: vm.AccountRef(from),
				AddrTo: to,
			},
			Errs: []error{ErrValidatorIsOut},
			Fn: func(c *testmodels.TestCase) {
				v := c.TestData.(testmodels.TestData)

				validator := storage.NewValidator(creatorAddress4, &withdrawalAddress)
				validator.ActivationEpoch = 0
				validator.ExitEpoch = procEpoch

				buf, err := validator.MarshalBinary()
				testutils.AssertNoError(t, err)
				processor.Storage().SetValidatorInfo(processor.state, buf)

				call(t, processor, v.Caller, v.AddrTo, value, exitOperation, c.Errs)
			},
		},
		{
			CaseName: "Deactivate: OK",
			TestData: testmodels.TestData{
				Caller: vm.AccountRef(withdrawalAddress),
				AddrTo: to,
			},
			Errs: []error{nil},
			Fn: func(c *testmodels.TestCase) {
				bc.EXPECT().GetEraInfo().Return(&eraInfo)
				bc.EXPECT().GetConfig().Return(testmodels.TestChainConfig)

				v := c.TestData.(testmodels.TestData)

				validator := storage.NewValidator(creatorAddress4, &withdrawalAddress)
				validator.ActivationEpoch = 0

				buf, err := validator.MarshalBinary()
				testutils.AssertNoError(t, err)
				processor.Storage().SetValidatorInfo(processor.state, buf)

				call(t, processor, v.Caller, v.AddrTo, value, exitOperation, c.Errs)

				valInfo := processor.storage.GetValidatorInfo(processor.state, creatorAddress4)

				testutils.AssertEqual(t, valInfo.GetExitEpoch(), testmodels.TestEra.To+1)
			},
		},
	}

	for _, c := range cases {
		t.Run(c.CaseName, func(t *testing.T) {
			c.Fn(c)
		})
	}
}

func TestProcessorWithdrawal(t *testing.T) {
	ctrl = gomock.NewController(t)
	defer ctrl.Finish()

	bc := NewMockBlockchain(ctrl)
	processor := NewProcessor(ctx, stateDb, testmodels.TestChainConfig, bc)
	to := processor.GetValidatorsStateAddress()

	withdrawalOperation, err := operation.NewWithdrawalOperation(creatorAddress5, value)
	testutils.AssertNoError(t, err)
	cases := []*testmodels.TestCase{
		{
			CaseName: "Withdrawal: invalid address to",
			TestData: testmodels.TestData{
				Caller: vm.AccountRef(from),
				AddrTo: creatorAddress,
			},
			Errs: []error{ErrInvalidToAddress},
			Fn: func(c *testmodels.TestCase) {
				v := c.TestData.(testmodels.TestData)
				call(t, processor, v.Caller, v.AddrTo, nil, withdrawalOperation, c.Errs)
			},
		},
		{
			CaseName: "Withdrawal: invalid address from",
			TestData: testmodels.TestData{
				Caller: vm.AccountRef(from),
				AddrTo: to,
			},
			Errs: []error{ErrInvalidFromAddresses},
			Fn: func(c *testmodels.TestCase) {
				v := c.TestData.(testmodels.TestData)

				validator := storage.NewValidator(creatorAddress5, &withdrawalAddress)
				valInfo, err := validator.MarshalBinary()
				testutils.AssertNoError(t, err)

				processor.Storage().SetValidatorInfo(processor.state, valInfo)

				call(t, processor, v.Caller, v.AddrTo, nil, withdrawalOperation, c.Errs)
			},
		},
		{
			CaseName: "Withdrawal: not activated validator",
			TestData: testmodels.TestData{
				Caller: vm.AccountRef(withdrawalAddress),
				AddrTo: to,
			},
			Errs: []error{ErrNotActivatedValidator},
			Fn: func(c *testmodels.TestCase) {
				v := c.TestData.(testmodels.TestData)

				validator := storage.NewValidator(creatorAddress5, &withdrawalAddress)
				buf, err := validator.MarshalBinary()
				testutils.AssertNoError(t, err)

				processor.Storage().SetValidatorInfo(processor.state, buf)

				call(t, processor, v.Caller, v.AddrTo, value, withdrawalOperation, c.Errs)
			},
		},
		{
			CaseName: "Withdrawal: insufficient funds",
			TestData: testmodels.TestData{
				Caller: vm.AccountRef(withdrawalAddress),
				AddrTo: to,
			},
			Errs: []error{ErrInsufficientFundsForTransfer},
			Fn: func(c *testmodels.TestCase) {
				v := c.TestData.(testmodels.TestData)

				validator := storage.NewValidator(creatorAddress5, &withdrawalAddress)
				validator.ActivationEpoch = 0
				buf, err := validator.MarshalBinary()
				testutils.AssertNoError(t, err)

				processor.Storage().SetValidatorInfo(processor.state, buf)

				call(t, processor, v.Caller, v.AddrTo, value, withdrawalOperation, c.Errs)
			},
		},
		{
			CaseName: "Withdrawal: OK",
			TestData: testmodels.TestData{
				Caller: vm.AccountRef(withdrawalAddress),
				AddrTo: to,
			},
			Errs: []error{nil},
			Fn: func(c *testmodels.TestCase) {
				valBalance := new(big.Int).Add(value, value)

				v := c.TestData.(testmodels.TestData)

				validator := storage.NewValidator(creatorAddress5, &withdrawalAddress)
				validator.Balance = valBalance
				validator.ActivationEpoch = 0
				buf, err := validator.MarshalBinary()
				testutils.AssertNoError(t, err)
				processor.Storage().SetValidatorInfo(processor.state, buf)

				call(t, processor, v.Caller, v.AddrTo, value, withdrawalOperation, c.Errs)

				valInfo := processor.Storage().GetValidatorInfo(processor.state, creatorAddress5)
				balanceDif := new(big.Int).Sub(valBalance, valInfo.GetValidatorBalance())

				if !testutils.BigIntEquals(value, balanceDif) {
					t.Fatalf("mismatch balance value, have %+v, want %+v", balanceDif, value)
				}
			},
		},
	}

	for _, c := range cases {
		t.Run(c.CaseName, func(t *testing.T) {
			c.Fn(c)
		})
	}
}

func TestProcessorUpdateBalance(t *testing.T) {
	ctrl = gomock.NewController(t)
	defer ctrl.Finish()

	bc := NewMockBlockchain(ctrl)
	processor := NewProcessor(ctx, stateDb, testmodels.TestChainConfig, bc)
	to := processor.GetValidatorsStateAddress()

	withdrawalOperation, err := operation.NewValidatorSyncOperation(types.UpdateBalance, procEpoch, 0, creatorAddress6, value, &withdrawalAddress)
	testutils.AssertNoError(t, err)
	cases := []*testmodels.TestCase{
		{
			CaseName: "UpdateBalance: unknown validator",
			TestData: testmodels.TestData{
				Caller: vm.AccountRef(from),
				AddrTo: to,
			},
			Errs: []error{ErrUnknownValidator},
			Fn: func(c *testmodels.TestCase) {
				v := c.TestData.(testmodels.TestData)
				call(t, processor, v.Caller, v.AddrTo, nil, withdrawalOperation, c.Errs)
			},
		},
		{
			CaseName: "UpdateBalance: not activated validator",
			TestData: testmodels.TestData{
				Caller: vm.AccountRef(withdrawalAddress),
				AddrTo: to,
			},
			Errs: []error{ErrNotActivatedValidator},
			Fn: func(c *testmodels.TestCase) {
				v := c.TestData.(testmodels.TestData)

				validator := storage.NewValidator(creatorAddress6, &withdrawalAddress)
				buf, err := validator.MarshalBinary()
				testutils.AssertNoError(t, err)

				processor.Storage().SetValidatorInfo(processor.state, buf)

				call(t, processor, v.Caller, v.AddrTo, value, withdrawalOperation, c.Errs)
			},
		},
		{
			CaseName: "UpdateBalance: no exit request",
			TestData: testmodels.TestData{
				Caller: vm.AccountRef(withdrawalAddress),
				AddrTo: to,
			},
			Errs: []error{ErrNoExitRequest},
			Fn: func(c *testmodels.TestCase) {
				v := c.TestData.(testmodels.TestData)

				validator := storage.NewValidator(creatorAddress6, &withdrawalAddress)
				validator.ActivationEpoch = 0
				buf, err := validator.MarshalBinary()
				testutils.AssertNoError(t, err)

				processor.Storage().SetValidatorInfo(processor.state, buf)

				call(t, processor, v.Caller, v.AddrTo, value, withdrawalOperation, c.Errs)
			},
		},
		{
			CaseName: "UpdateBalance: OK",
			TestData: testmodels.TestData{
				Caller: vm.AccountRef(withdrawalAddress),
				AddrTo: to,
			},
			Errs: []error{nil},
			Fn: func(c *testmodels.TestCase) {
				valBalance := new(big.Int).Add(value, value)

				v := c.TestData.(testmodels.TestData)

				validator := storage.NewValidator(creatorAddress6, &withdrawalAddress)
				validator.Balance = valBalance
				validator.ActivationEpoch = 0
				validator.ExitEpoch = procEpoch
				buf, err := validator.MarshalBinary()
				testutils.AssertNoError(t, err)
				processor.Storage().SetValidatorInfo(processor.state, buf)

				call(t, processor, v.Caller, v.AddrTo, value, withdrawalOperation, c.Errs)

				valInfo := processor.Storage().GetValidatorInfo(processor.state, creatorAddress6)

				if !testutils.BigIntEquals(value, valInfo.GetValidatorBalance()) {
					t.Fatalf("mismatch balance value, have %+v, want %+v", valInfo.GetValidatorBalance(), value)
				}
			},
		},
	}

	for _, c := range cases {
		t.Run(c.CaseName, func(t *testing.T) {
			c.Fn(c)
		})
	}
}

func call(t *testing.T, processor *Processor, Caller Ref, addrTo common.Address, value *big.Int, op operation.Operation, Errs []error) []byte {
	res, err := processor.Call(Caller, addrTo, value, op)
	if !testutils.CheckError(err, Errs) {
		t.Fatalf("Case failed\nwant errors: %s\nhave errors: %s", Errs, err)
	}

	return res
}
