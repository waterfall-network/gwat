package validator

import (
	"math"
	"math/big"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/rawdb"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/state"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/types"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/vm"
	"gitlab.waterfall.network/waterfall/protocol/gwat/crypto"
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
	withdrawalAddress common.Address   // attached withdrawal credentials
	signature         common.BlsSignature
	ctrl              *gomock.Controller
	eraInfo           era.EraInfo

	initTxHash = common.HexToHash("0303030303030303030303030303030303030303030303030303030303030303")
	value      = MinDepositVal
	procEpoch  = uint64(100)
	ctx        = vm.BlockContext{
		CanTransfer: nil,
		Transfer:    nil,
		Coinbase:    common.Address{},
		BlockNumber: new(big.Int).SetUint64(8000000),
		Time:        new(big.Int).SetUint64(1658844221),
		Difficulty:  big.NewInt(0x30000),
		GasLimit:    uint64(6000000),
		Era:         5,
		Slot:        0,
	}
)

func init() {
	stateDb, _ = state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)

	from = common.BytesToAddress(testutils.RandomData(20))
	pubKey = common.BytesToBlsPubKey(testutils.RandomData(48))
	withdrawalAddress = common.BytesToAddress(testutils.RandomData(20))
	signature = common.BytesToBlsSig(testutils.RandomData(96))

	eraInfo = era.NewEraInfo(testmodels.TestEra)
}

func TestProcessorDeposit(t *testing.T) {
	ctrl = gomock.NewController(t)
	defer ctrl.Finish()
	msg := NewMockmessage(ctrl)

	bc := NewMockblockchain(ctrl)
	bc.EXPECT().Config().Return(testmodels.TestChainConfig)

	processor := NewProcessor(ctx, stateDb, bc)
	to := processor.GetValidatorsStateAddress()

	depositOperation, err := operation.NewDepositOperation(pubKey, testmodels.Addr1, withdrawalAddress, signature, nil)
	testutils.AssertNoError(t, err)

	opData, err := operation.EncodeToBytes(depositOperation)
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

				msg.EXPECT().Data().AnyTimes().Return(opData)
				msg.EXPECT().TxHash().AnyTimes().Return(common.Hash{})

				call(t, processor, v.Caller, v.AddrTo, value, msg, c.Errs)

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
				call(t, processor, v.Caller, v.AddrTo, nil, msg, c.Errs)
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
				call(t, processor, v.Caller, v.AddrTo, val1, msg, c.Errs)
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
				call(t, processor, v.Caller, v.AddrTo, nil, msg, c.Errs)
			},
		},
	}

	for _, c := range cases {
		t.Run(c.CaseName, func(t *testing.T) {
			c.Fn(c)
		})
	}
}

func TestTestProcessorDeposit_DelegatingStake(t *testing.T) {
	ctrl = gomock.NewController(t)
	defer ctrl.Finish()

	dsProfitShare, dsStakeShare, dsExit, dsWithdrawal := operation.TestParamsDelegatingStakeRules()

	bc := NewMockblockchain(ctrl)
	bc.EXPECT().Config().Return(testmodels.TestChainConfig).AnyTimes()
	processor := NewProcessor(ctx, stateDb, bc)
	to := processor.GetValidatorsStateAddress()

	rules, _ := operation.NewDelegatingStakeRules(dsProfitShare, dsStakeShare, dsExit, dsWithdrawal)
	trialRules, _ := operation.NewDelegatingStakeRules(dsProfitShare, dsStakeShare, dsExit, dsWithdrawal)
	cases := []*testmodels.TestCase{
		{
			CaseName: "Deposit_DelegatingStake_OK",
			TestData: testmodels.TestData{
				Caller: vm.AccountRef(from),
				AddrTo: to,
			},
			Errs: []error{nil},
			Fn: func(c *testmodels.TestCase) {
				stateDb, _ = state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
				bc := NewMockblockchain(ctrl)
				bc.EXPECT().Config().Return(testmodels.TestChainConfig).AnyTimes()
				processor := NewProcessor(ctx, stateDb, bc)

				delegateData, err := operation.NewDelegatingStakeData(
					rules,
					321,
					trialRules,
				)
				testutils.AssertNoError(t, err)

				wdrAddr := withdrawalAddress
				pk := pubKey
				depositOperation, err := operation.NewDepositOperation(pk, testmodels.Addr1, wdrAddr, signature, delegateData.Copy())
				testutils.AssertNoError(t, err)

				//set empty rules
				depositOperation.DelegatingStake().Rules = operation.DelegatingStakeRules{}

				opData, err := operation.EncodeToBytes(depositOperation)
				testutils.AssertNoError(t, err)

				v := c.TestData.(testmodels.TestData)

				bal, _ := new(big.Int).SetString("32000000000000000000000", 10)
				processor.state.AddBalance(from, bal)
				balanceFromBfr := processor.state.GetBalance(from)

				msg := NewMockmessage(ctrl)
				msg.EXPECT().Data().AnyTimes().Return(opData)
				msg.EXPECT().TxHash().AnyTimes().Return(common.Hash{})

				call(t, processor, v.Caller, v.AddrTo, value, msg, c.Errs)

				balanceFromAft := processor.state.GetBalance(from)
				balDif := new(big.Int).Sub(balanceFromBfr, balanceFromAft)
				if balDif.Cmp(value) != 0 {
					t.Errorf("Expected balance From ios bad : %d\nactual: %s", 1, balanceFromAft)
				}

				val, err := processor.Storage().GetValidator(processor.state, testmodels.Addr1)
				testutils.AssertNoError(t, err)
				testutils.AssertEqual(t, testmodels.Addr1.Bytes(), val.Address.Bytes())
				testutils.AssertEqual(t, pk.Bytes(), val.PubKey.Bytes())
				testutils.AssertEqual(t, wdrAddr.Bytes(), val.WithdrawalAddress.Bytes())

				//check delegate data
				testutils.AssertEqual(t, dsProfitShare, val.DelegatingStake.Rules.ProfitShare())
				testutils.AssertEqual(t, dsStakeShare, val.DelegatingStake.Rules.StakeShare())
				testutils.AssertEqual(t, dsExit, val.DelegatingStake.Rules.Exit())
				testutils.AssertEqual(t, dsWithdrawal, val.DelegatingStake.Rules.Withdrawal())

				testutils.AssertEqual(t, uint64(321), val.DelegatingStake.TrialPeriod)
				testutils.AssertEqual(t, dsProfitShare, val.DelegatingStake.TrialRules.ProfitShare())
				testutils.AssertEqual(t, dsStakeShare, val.DelegatingStake.TrialRules.StakeShare())
				testutils.AssertEqual(t, dsExit, val.DelegatingStake.TrialRules.Exit())
				testutils.AssertEqual(t, dsWithdrawal, val.DelegatingStake.TrialRules.Withdrawal())
			},
		},
		{
			CaseName: "Deposit_DelegatingStake_no_trial_OK",
			TestData: testmodels.TestData{
				Caller: vm.AccountRef(from),
				AddrTo: to,
			},
			Errs: []error{nil},
			Fn: func(c *testmodels.TestCase) {
				stateDb, _ = state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
				bc := NewMockblockchain(ctrl)
				bc.EXPECT().Config().Return(testmodels.TestChainConfig).AnyTimes()
				processor := NewProcessor(ctx, stateDb, bc)

				delegateData, err := operation.NewDelegatingStakeData(
					rules,
					321,
					nil,
				)
				testutils.AssertNoError(t, err)

				wdrAddr := withdrawalAddress
				pk := pubKey
				depositOperation, err := operation.NewDepositOperation(pk, testmodels.Addr1, wdrAddr, signature, delegateData)
				testutils.AssertNoError(t, err)

				opData, err := operation.EncodeToBytes(depositOperation)
				testutils.AssertNoError(t, err)

				v := c.TestData.(testmodels.TestData)

				bal, _ := new(big.Int).SetString("32000000000000000000000", 10)
				processor.state.AddBalance(from, bal)
				balanceFromBfr := processor.state.GetBalance(from)

				msg := NewMockmessage(ctrl)
				msg.EXPECT().Data().AnyTimes().Return(opData)
				msg.EXPECT().TxHash().AnyTimes().Return(common.Hash{})

				call(t, processor, v.Caller, v.AddrTo, value, msg, c.Errs)

				balanceFromAft := processor.state.GetBalance(from)
				balDif := new(big.Int).Sub(balanceFromBfr, balanceFromAft)
				if balDif.Cmp(value) != 0 {
					t.Errorf("Expected balance From ios bad : %d\nactual: %s", 1, balanceFromAft)
				}

				val, err := processor.Storage().GetValidator(processor.state, testmodels.Addr1)
				testutils.AssertNoError(t, err)
				testutils.AssertEqual(t, testmodels.Addr1.Bytes(), val.Address.Bytes())
				testutils.AssertEqual(t, pk.Bytes(), val.PubKey.Bytes())
				testutils.AssertEqual(t, wdrAddr.Bytes(), val.WithdrawalAddress.Bytes())

				//check delegate data
				testutils.AssertEqual(t, dsProfitShare, val.DelegatingStake.Rules.ProfitShare())
				testutils.AssertEqual(t, dsStakeShare, val.DelegatingStake.Rules.StakeShare())
				testutils.AssertEqual(t, dsExit, val.DelegatingStake.Rules.Exit())
				testutils.AssertEqual(t, dsWithdrawal, val.DelegatingStake.Rules.Withdrawal())

				testutils.AssertEqual(t, uint64(321), val.DelegatingStake.TrialPeriod)
				testutils.AssertEqual(t, make(map[common.Address]uint8), val.DelegatingStake.TrialRules.ProfitShare())
				testutils.AssertEqual(t, make(map[common.Address]uint8), val.DelegatingStake.TrialRules.StakeShare())
				testutils.AssertEqual(t, make([]common.Address, 0), val.DelegatingStake.TrialRules.Exit())
				testutils.AssertEqual(t, make([]common.Address, 0), val.DelegatingStake.TrialRules.Withdrawal())
			},
		},
		{
			CaseName: "Deposit_DelegatingStake_validator_exists_OK",
			TestData: testmodels.TestData{
				Caller: vm.AccountRef(from),
				AddrTo: to,
			},
			Errs: []error{nil},
			Fn: func(c *testmodels.TestCase) {
				stateDb, _ = state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
				bc := NewMockblockchain(ctrl)
				bc.EXPECT().Config().Return(testmodels.TestChainConfig).AnyTimes()
				processor := NewProcessor(ctx, stateDb, bc)

				delegateData, err := operation.NewDelegatingStakeData(
					rules,
					321,
					trialRules,
				)
				testutils.AssertNoError(t, err)

				wdrAddr := withdrawalAddress
				pk := pubKey
				depositOperation, err := operation.NewDepositOperation(pk, testmodels.Addr1, wdrAddr, signature, delegateData)
				testutils.AssertNoError(t, err)

				//create existed validator
				validator := storage.NewValidator(depositOperation.PubKey(), depositOperation.CreatorAddress(), &wdrAddr)
				validator.DelegatingStake = depositOperation.DelegatingStake()
				err = processor.Storage().SetValidator(processor.state, validator)
				testutils.AssertNoError(t, err)

				opData, err := operation.EncodeToBytes(depositOperation)
				testutils.AssertNoError(t, err)

				v := c.TestData.(testmodels.TestData)

				bal, _ := new(big.Int).SetString("32000000000000000000000", 10)
				processor.state.AddBalance(from, bal)
				balanceFromBfr := processor.state.GetBalance(from)

				msg := NewMockmessage(ctrl)
				msg.EXPECT().Data().AnyTimes().Return(opData)
				msg.EXPECT().TxHash().AnyTimes().Return(common.Hash{})

				call(t, processor, v.Caller, v.AddrTo, value, msg, c.Errs)

				balanceFromAft := processor.state.GetBalance(from)
				balDif := new(big.Int).Sub(balanceFromBfr, balanceFromAft)
				if balDif.Cmp(value) != 0 {
					t.Errorf("Expected balance From ios bad : %d\nactual: %s", 1, balanceFromAft)
				}

				val, err := processor.Storage().GetValidator(processor.state, testmodels.Addr1)
				testutils.AssertNoError(t, err)
				testutils.AssertEqual(t, testmodels.Addr1.Bytes(), val.Address.Bytes())
				testutils.AssertEqual(t, pk.Bytes(), val.PubKey.Bytes())
				testutils.AssertEqual(t, wdrAddr.Bytes(), val.WithdrawalAddress.Bytes())

				//check delegate data
				testutils.AssertEqual(t, dsProfitShare, val.DelegatingStake.Rules.ProfitShare())
				testutils.AssertEqual(t, dsStakeShare, val.DelegatingStake.Rules.StakeShare())
				testutils.AssertEqual(t, dsExit, val.DelegatingStake.Rules.Exit())
				testutils.AssertEqual(t, dsWithdrawal, val.DelegatingStake.Rules.Withdrawal())

				testutils.AssertEqual(t, uint64(321), val.DelegatingStake.TrialPeriod)
				testutils.AssertEqual(t, dsProfitShare, val.DelegatingStake.TrialRules.ProfitShare())
				testutils.AssertEqual(t, dsStakeShare, val.DelegatingStake.TrialRules.StakeShare())
				testutils.AssertEqual(t, dsExit, val.DelegatingStake.TrialRules.Exit())
				testutils.AssertEqual(t, dsWithdrawal, val.DelegatingStake.TrialRules.Withdrawal())
			},
		},

		{
			CaseName: "Deposit_DelegatingStake_validator_exists_&_no_trial_OK",
			TestData: testmodels.TestData{
				Caller: vm.AccountRef(from),
				AddrTo: to,
			},
			Errs: []error{nil},
			Fn: func(c *testmodels.TestCase) {
				stateDb, _ = state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
				bc := NewMockblockchain(ctrl)
				bc.EXPECT().Config().Return(testmodels.TestChainConfig).AnyTimes()
				processor := NewProcessor(ctx, stateDb, bc)

				delegateData, err := operation.NewDelegatingStakeData(
					rules,
					321,
					nil,
				)
				testutils.AssertNoError(t, err)

				wdrAddr := withdrawalAddress
				pk := pubKey
				depositOperation, err := operation.NewDepositOperation(pk, testmodels.Addr1, wdrAddr, signature, delegateData)
				testutils.AssertNoError(t, err)
				opData, err := operation.EncodeToBytes(depositOperation)
				testutils.AssertNoError(t, err)

				depositOperation.UnmarshalBinary(opData)

				//create existed validator
				validator := storage.NewValidator(depositOperation.PubKey(), depositOperation.CreatorAddress(), &wdrAddr)
				validator.DelegatingStake = depositOperation.DelegatingStake()
				err = processor.Storage().SetValidator(processor.state, validator)
				testutils.AssertNoError(t, err)

				v := c.TestData.(testmodels.TestData)

				bal, _ := new(big.Int).SetString("32000000000000000000000", 10)
				processor.state.AddBalance(from, bal)
				balanceFromBfr := processor.state.GetBalance(from)

				msg := NewMockmessage(ctrl)
				msg.EXPECT().Data().AnyTimes().Return(opData)
				msg.EXPECT().TxHash().AnyTimes().Return(common.Hash{})

				call(t, processor, v.Caller, v.AddrTo, value, msg, c.Errs)

				balanceFromAft := processor.state.GetBalance(from)
				balDif := new(big.Int).Sub(balanceFromBfr, balanceFromAft)
				if balDif.Cmp(value) != 0 {
					t.Errorf("Expected balance From ios bad : %d\nactual: %s", 1, balanceFromAft)
				}

				val, err := processor.Storage().GetValidator(processor.state, testmodels.Addr1)
				testutils.AssertNoError(t, err)
				testutils.AssertEqual(t, testmodels.Addr1.Bytes(), val.Address.Bytes())
				testutils.AssertEqual(t, pk.Bytes(), val.PubKey.Bytes())
				testutils.AssertEqual(t, wdrAddr.Bytes(), val.WithdrawalAddress.Bytes())

				//check delegate data
				testutils.AssertEqual(t, dsProfitShare, val.DelegatingStake.Rules.ProfitShare())
				testutils.AssertEqual(t, dsStakeShare, val.DelegatingStake.Rules.StakeShare())
				testutils.AssertEqual(t, dsExit, val.DelegatingStake.Rules.Exit())
				testutils.AssertEqual(t, dsWithdrawal, val.DelegatingStake.Rules.Withdrawal())

				testutils.AssertEqual(t, uint64(321), val.DelegatingStake.TrialPeriod)
				testutils.AssertEqual(t, make(map[common.Address]uint8), val.DelegatingStake.TrialRules.ProfitShare())
				testutils.AssertEqual(t, make(map[common.Address]uint8), val.DelegatingStake.TrialRules.StakeShare())
				testutils.AssertEqual(t, make([]common.Address, 0), val.DelegatingStake.TrialRules.Exit())
				testutils.AssertEqual(t, make([]common.Address, 0), val.DelegatingStake.TrialRules.Withdrawal())
			},
		},

		{
			CaseName: "Deposit_DelegatingStake_validator_exists_ErrMismatchDelegateData",
			TestData: testmodels.TestData{
				Caller: vm.AccountRef(from),
				AddrTo: to,
			},
			Errs: []error{ErrMismatchDelegateData},
			Fn: func(c *testmodels.TestCase) {
				stateDb, _ = state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
				bc := NewMockblockchain(ctrl)
				bc.EXPECT().Config().Return(testmodels.TestChainConfig).AnyTimes()
				processor := NewProcessor(ctx, stateDb, bc)

				changedRules, err := operation.NewDelegatingStakeRules(
					map[common.Address]uint8{common.Address{0x16}: 100},
					map[common.Address]uint8{},
					nil,
					[]common.Address{common.Address{0x16}},
				)
				testutils.AssertNoError(t, err)

				delegateData, err := operation.NewDelegatingStakeData(
					rules,
					321,
					changedRules,
				)
				testutils.AssertNoError(t, err)

				wdrAddr := withdrawalAddress
				pk := pubKey
				depositOperation, err := operation.NewDepositOperation(pk, testmodels.Addr1, wdrAddr, signature, delegateData)
				testutils.AssertNoError(t, err)
				opData, err := operation.EncodeToBytes(depositOperation)
				testutils.AssertNoError(t, err)

				depositOperation.UnmarshalBinary(opData)

				//create existed validator
				valDelegateData, err := operation.NewDelegatingStakeData(
					rules,
					321,
					nil,
				)
				testutils.AssertNoError(t, err)
				validator := storage.NewValidator(depositOperation.PubKey(), depositOperation.CreatorAddress(), &wdrAddr)
				validator.DelegatingStake = valDelegateData.Copy()
				err = processor.Storage().SetValidator(processor.state, validator)
				testutils.AssertNoError(t, err)

				v := c.TestData.(testmodels.TestData)

				bal, _ := new(big.Int).SetString("32000000000000000000000", 10)
				processor.state.AddBalance(from, bal)
				//balanceFromBfr := processor.state.GetBalance(from)

				msg := NewMockmessage(ctrl)
				msg.EXPECT().Data().AnyTimes().Return(opData)
				msg.EXPECT().TxHash().AnyTimes().Return(common.Hash{})

				call(t, processor, v.Caller, v.AddrTo, value, msg, c.Errs)

				val, err := processor.Storage().GetValidator(processor.state, testmodels.Addr1)
				testutils.AssertNoError(t, err)
				//check delegate data
				testutils.AssertEqual(t, valDelegateData.Rules.ProfitShare(), val.DelegatingStake.Rules.ProfitShare())
				testutils.AssertEqual(t, valDelegateData.Rules.StakeShare(), val.DelegatingStake.Rules.StakeShare())
				testutils.AssertEqual(t, valDelegateData.Rules.Exit(), val.DelegatingStake.Rules.Exit())
				testutils.AssertEqual(t, valDelegateData.Rules.Withdrawal(), val.DelegatingStake.Rules.Withdrawal())

				testutils.AssertEqual(t, uint64(321), val.DelegatingStake.TrialPeriod)
				testutils.AssertEqual(t, valDelegateData.TrialRules.ProfitShare(), val.DelegatingStake.TrialRules.ProfitShare())
				testutils.AssertEqual(t, valDelegateData.TrialRules.StakeShare(), val.DelegatingStake.TrialRules.StakeShare())
				testutils.AssertEqual(t, valDelegateData.TrialRules.Exit(), val.DelegatingStake.TrialRules.Exit())
				testutils.AssertEqual(t, valDelegateData.TrialRules.Withdrawal(), val.DelegatingStake.TrialRules.Withdrawal())
			},
		},

		{
			CaseName: "Deposit_DelegatingStake_ErrDelegateForkRequire",
			TestData: testmodels.TestData{
				Caller: vm.AccountRef(from),
				AddrTo: to,
			},
			Errs: []error{operation.ErrDelegateForkRequire},
			Fn: func(c *testmodels.TestCase) {
				testChainConfig := testmodels.TestChainConfig
				testChainConfig.ForkSlotDelegate = 1000
				bc.EXPECT().Config().Return(testChainConfig).AnyTimes()

				delegateData, err := operation.NewDelegatingStakeData(
					rules,
					321,
					trialRules,
				)
				testutils.AssertNoError(t, err)

				wdrAddr := withdrawalAddress
				pk := pubKey
				depositOperation, err := operation.NewDepositOperation(pk, testmodels.Addr1, wdrAddr, signature, delegateData.Copy())
				testutils.AssertNoError(t, err)

				//set empty rules
				depositOperation.DelegatingStake().Rules = operation.DelegatingStakeRules{}

				opData, err := operation.EncodeToBytes(depositOperation)
				testutils.AssertNoError(t, err)

				v := c.TestData.(testmodels.TestData)

				msg := NewMockmessage(ctrl)
				msg.EXPECT().Data().AnyTimes().Return(opData)
				msg.EXPECT().TxHash().AnyTimes().Return(common.Hash{})
				call(t, processor, v.Caller, v.AddrTo, value, msg, c.Errs)
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
	activateOperation, err := operation.NewValidatorSyncOperation(0, types.Activate, initTxHash, procEpoch, 0, testmodels.Addr2, nil, &withdrawalAddress, nil)
	testutils.AssertNoError(t, err)

	ctrl = gomock.NewController(t)
	defer ctrl.Finish()

	msg := NewMockmessage(ctrl)

	db := rawdb.NewMemoryDatabase()
	rawdb.WriteEra(db, eraInfo.Number(), *eraInfo.GetEra())
	bc := NewMockblockchain(ctrl)
	bc.EXPECT().Config().AnyTimes().Return(testmodels.TestChainConfig)
	bc.EXPECT().GetSlotInfo().AnyTimes().Return(&types.SlotInfo{
		GenesisTime:    uint64(time.Now().Unix()),
		SecondsPerSlot: testmodels.TestChainConfig.SecondsPerSlot,
		SlotsPerEpoch:  testmodels.TestChainConfig.SlotsPerEpoch,
	})
	bc.EXPECT().GetEraInfo().AnyTimes().Return(&eraInfo)
	bc.EXPECT().Database().AnyTimes().Return(db)
	bc.EXPECT().GetValidatorSyncData(
		gomock.AssignableToTypeOf(common.Hash{})).
		AnyTimes().Return(&types.ValidatorSync{
		OpType:     activateOperation.OpType(),
		ProcEpoch:  activateOperation.ProcEpoch(),
		Index:      activateOperation.Index(),
		Creator:    activateOperation.Creator(),
		Amount:     activateOperation.Amount(),
		InitTxHash: initTxHash,
	})
	bc.EXPECT().EpochToEra(gomock.AssignableToTypeOf(uint64(0))).AnyTimes().Return(&era.Era{Number: 6})

	processor := NewProcessor(ctx, stateDb, bc)
	to := processor.GetValidatorsStateAddress()

	opData, err := operation.EncodeToBytes(activateOperation)
	testutils.AssertNoError(t, err)
	msg.EXPECT().Data().AnyTimes().Return(opData)
	msg.EXPECT().TxHash().AnyTimes().Return(common.Hash{})

	//add init tx
	depositOperation, err := operation.NewDepositOperation(pubKey, testmodels.Addr2, withdrawalAddress, signature, nil)
	testutils.AssertNoError(t, err)
	initTxData, err := operation.EncodeToBytes(depositOperation)
	testutils.AssertNoError(t, err)
	initTx := types.NewTx(&types.AccessListTx{Data: initTxData})
	bc.EXPECT().GetTransaction(
		common.HexToHash("0x0303030303030303030303030303030303030303030303030303030303030303"),
	).AnyTimes().Return(initTx, common.Hash{}, uint64(0))
	initTxRcp := &types.Receipt{Status: types.ReceiptStatusSuccessful}
	bc.EXPECT().GetTransactionReceipt(
		common.HexToHash("0x0303030303030303030303030303030303030303030303030303030303030303"),
	).AnyTimes().Return(initTxRcp, common.Hash{}, uint64(0))

	cases := []*testmodels.TestCase{
		{
			CaseName: "Activate: Unknown validator",
			TestData: testmodels.TestData{
				Caller: vm.AccountRef(from),
				AddrTo: to,
			},
			Errs: []error{storage.ErrNoStateValidatorInfo},
			Fn: func(c *testmodels.TestCase) {
				v := c.TestData.(testmodels.TestData)

				call(t, processor, v.Caller, v.AddrTo, nil, msg, c.Errs)
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
				v := c.TestData.(testmodels.TestData)

				validator := storage.NewValidator(pubKey, testmodels.Addr2, &withdrawalAddress)

				err = processor.Storage().SetValidator(processor.state, validator)
				testutils.AssertNoError(t, err)

				val, err := processor.storage.GetValidator(processor.state, testmodels.Addr2)
				testutils.AssertNoError(t, err)

				if val.GetIndex() != math.MaxUint64 {
					t.Fatal()
				}
				if val.GetActivationEra() != math.MaxUint64 {
					t.Fatal()
				}

				call(t, processor, v.Caller, v.AddrTo, value, msg, c.Errs)

				val, err = processor.storage.GetValidator(processor.state, testmodels.Addr2)
				testutils.AssertNoError(t, err)

				testutils.AssertEqual(t, val.GetActivationEra(), testmodels.TestEra.Number+1)
				testutils.AssertEqual(t, val.GetIndex(), activateOperation.Index())

				valList := processor.Storage().GetValidatorsList(processor.state)
				if len(valList) > 1 {
					t.Fatal()
				}

				testutils.AssertEqual(t, validator.Address, valList[0])
			},
		},
		{
			CaseName: "Activate: read era from db",
			TestData: testmodels.TestData{
				Caller: vm.AccountRef(from),
				AddrTo: to,
			},
			Errs: []error{},
			Fn: func(c *testmodels.TestCase) {
				v := c.TestData.(testmodels.TestData)

				validator := storage.NewValidator(pubKey, testmodels.Addr2, &withdrawalAddress)

				err = processor.Storage().SetValidator(processor.state, validator)
				testutils.AssertNoError(t, err)

				val, err := processor.storage.GetValidator(processor.state, testmodels.Addr2)
				testutils.AssertNoError(t, err)

				if val.GetIndex() != math.MaxUint64 {
					t.Fatal()
				}
				if val.GetActivationEra() != math.MaxUint64 {
					t.Fatal()
				}

				processor.ctx.Era = 3
				rawdb.WriteEra(db, 3, era.Era{
					Number: 3,
					From:   50,
					To:     80,
					Root:   common.BytesToHash(testutils.RandomStringInBytes(32)),
				})

				call(t, processor, v.Caller, v.AddrTo, value, msg, c.Errs)

				val, err = processor.storage.GetValidator(processor.state, testmodels.Addr2)
				testutils.AssertNoError(t, err)

				testutils.AssertEqual(t, val.GetActivationEra(), uint64(8))
				testutils.AssertEqual(t, val.GetIndex(), activateOperation.Index())

				valList := processor.Storage().GetValidatorsList(processor.state)
				if len(valList) > 1 {
					t.Fatal()
				}

				testutils.AssertEqual(t, validator.Address, valList[0])
			},
		},
		{
			CaseName: "Activate: epoch in transition period",
			TestData: testmodels.TestData{
				Caller: vm.AccountRef(from),
				AddrTo: to,
			},
			Errs: []error{},
			Fn: func(c *testmodels.TestCase) {
				v := c.TestData.(testmodels.TestData)

				validator := storage.NewValidator(pubKey, testmodels.Addr2, &withdrawalAddress)

				err = processor.Storage().SetValidator(processor.state, validator)
				testutils.AssertNoError(t, err)

				val, err := processor.storage.GetValidator(processor.state, testmodels.Addr2)
				testutils.AssertNoError(t, err)

				if val.GetIndex() != math.MaxUint64 {
					t.Fatal()
				}
				if val.GetActivationEra() != math.MaxUint64 {
					t.Fatal()
				}

				processor.ctx.Slot = 2790
				processor.ctx.Era = 4
				rawdb.WriteEra(db, 3, era.Era{
					Number: 3,
					From:   44,
					To:     66,
					Root:   common.BytesToHash(testutils.RandomStringInBytes(32)),
				})
				rawdb.WriteEra(db, 4, era.Era{
					Number: 4,
					From:   66,
					To:     88,
					Root:   common.BytesToHash(testutils.RandomStringInBytes(32)),
				})

				call(t, processor, v.Caller, v.AddrTo, value, msg, c.Errs)

				val, err = processor.storage.GetValidator(processor.state, testmodels.Addr2)
				testutils.AssertNoError(t, err)

				testutils.AssertEqual(t, val.GetActivationEra(), testmodels.TestEra.Number+1)
				testutils.AssertEqual(t, val.GetIndex(), activateOperation.Index())

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

	msg := NewMockmessage(ctrl)

	eraInfo := era.NewEraInfo(testmodels.TestEra)
	bc := NewMockblockchain(ctrl)
	bc.EXPECT().Config().Return(testmodels.TestChainConfig)
	bc.EXPECT().GetEraInfo().AnyTimes().Return(&eraInfo)

	processor := NewProcessor(ctx, stateDb, bc)
	to := processor.GetValidatorsStateAddress()

	exitOperation, err := operation.NewExitOperation(pubKey, testmodels.Addr3, &procEpoch)
	testutils.AssertNoError(t, err)

	opData, err := operation.EncodeToBytes(exitOperation)
	testutils.AssertNoError(t, err)
	msg.EXPECT().Data().AnyTimes().Return(opData)
	msg.EXPECT().TxHash().AnyTimes().Return(common.Hash{})

	cases := []*testmodels.TestCase{
		{
			CaseName: "Exit: Unknown validator",
			TestData: testmodels.TestData{
				Caller: vm.AccountRef(from),
				AddrTo: processor.GetValidatorsStateAddress(),
			},
			Errs: []error{storage.ErrNoStateValidatorInfo},
			Fn: func(c *testmodels.TestCase) {
				v := c.TestData.(testmodels.TestData)
				call(t, processor, v.Caller, v.AddrTo, nil, msg, c.Errs)
			},
		},
		{
			CaseName: "Exit: invalid to address",
			TestData: testmodels.TestData{
				Caller: vm.AccountRef(from),
				AddrTo: testmodels.Addr1,
			},
			Errs: []error{ErrInvalidToAddress},
			Fn: func(c *testmodels.TestCase) {
				v := c.TestData.(testmodels.TestData)
				call(t, processor, v.Caller, v.AddrTo, nil, msg, c.Errs)
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

				validator := storage.NewValidator(pubKey, testmodels.Addr3, &withdrawalAddress)

				err = processor.Storage().SetValidator(processor.state, validator)
				testutils.AssertNoError(t, err)

				call(t, processor, v.Caller, v.AddrTo, value, msg, c.Errs)
			},
		},
		{
			CaseName: "Exit: validator is exited",
			TestData: testmodels.TestData{
				Caller: vm.AccountRef(from),
				AddrTo: to,
			},
			Errs: []error{ErrValidatorIsOut},
			Fn: func(c *testmodels.TestCase) {
				v := c.TestData.(testmodels.TestData)

				validator := storage.NewValidator(pubKey, testmodels.Addr3, &withdrawalAddress)
				validator.ActivationEra = 0
				validator.ExitEra = procEpoch

				err = processor.Storage().SetValidator(processor.state, validator)
				testutils.AssertNoError(t, err)

				call(t, processor, v.Caller, v.AddrTo, value, msg, c.Errs)
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

				validator := storage.NewValidator(pubKey, testmodels.Addr3, &withdrawalAddress)
				validator.ActivationEra = 0

				err = processor.Storage().SetValidator(processor.state, validator)
				testutils.AssertNoError(t, err)

				call(t, processor, v.Caller, v.AddrTo, value, msg, c.Errs)
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

				validator := storage.NewValidator(pubKey, testmodels.Addr3, &withdrawalAddress)
				validator.ActivationEra = 0

				err = processor.Storage().SetValidator(processor.state, validator)
				testutils.AssertNoError(t, err)

				call(t, processor, v.Caller, v.AddrTo, value, msg, c.Errs)
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
	deactivateOp, err := operation.NewValidatorSyncOperation(0, types.Deactivate, initTxHash, procEpoch, 0, testmodels.Addr4, nil, &withdrawalAddress, nil)
	testutils.AssertNoError(t, err)

	ctrl = gomock.NewController(t)
	defer ctrl.Finish()

	msg := NewMockmessage(ctrl)

	db := rawdb.NewMemoryDatabase()
	rawdb.WriteEra(db, eraInfo.Number(), *eraInfo.GetEra())
	bc := NewMockblockchain(ctrl)
	bc.EXPECT().Config().AnyTimes().Return(testmodels.TestChainConfig)
	bc.EXPECT().GetSlotInfo().AnyTimes().Return(&types.SlotInfo{
		GenesisTime:    uint64(time.Now().Unix()),
		SecondsPerSlot: testmodels.TestChainConfig.SecondsPerSlot,
		SlotsPerEpoch:  testmodels.TestChainConfig.SlotsPerEpoch,
	})
	bc.EXPECT().GetEraInfo().AnyTimes().Return(&eraInfo)
	bc.EXPECT().Database().AnyTimes().Return(db)
	bc.EXPECT().GetValidatorSyncData(gomock.AssignableToTypeOf(common.Hash{})).
		AnyTimes().Return(&types.ValidatorSync{
		OpType:     deactivateOp.OpType(),
		ProcEpoch:  deactivateOp.ProcEpoch(),
		Index:      deactivateOp.Index(),
		Creator:    deactivateOp.Creator(),
		Amount:     deactivateOp.Amount(),
		InitTxHash: initTxHash,
	})
	bc.EXPECT().EpochToEra(uint64(100)).AnyTimes().Return(&testmodels.TestEra)

	processor := NewProcessor(ctx, stateDb, bc)
	to := processor.GetValidatorsStateAddress()

	opData, err := operation.EncodeToBytes(deactivateOp)
	testutils.AssertNoError(t, err)
	msg.EXPECT().Data().AnyTimes().Return(opData)
	msg.EXPECT().TxHash().AnyTimes().Return(common.Hash{1, 2, 3})

	//add init tx
	depositOperation, err := operation.NewDepositOperation(pubKey, testmodels.Addr4, withdrawalAddress, signature, nil)
	testutils.AssertNoError(t, err)
	initTxData, err := operation.EncodeToBytes(depositOperation)
	testutils.AssertNoError(t, err)
	initTx := types.NewTx(&types.AccessListTx{Data: initTxData})
	bc.EXPECT().GetTransaction(
		common.HexToHash("0x0303030303030303030303030303030303030303030303030303030303030303"),
	).AnyTimes().Return(initTx, common.Hash{}, uint64(0))
	initTxRcp := &types.Receipt{Status: types.ReceiptStatusSuccessful}
	bc.EXPECT().GetTransactionReceipt(
		common.HexToHash("0x0303030303030303030303030303030303030303030303030303030303030303"),
	).AnyTimes().Return(initTxRcp, common.Hash{}, uint64(0))

	cases := []*testmodels.TestCase{
		{
			CaseName: "Deactivate: Unknown validator",
			TestData: testmodels.TestData{
				Caller: vm.AccountRef(from),
				AddrTo: processor.GetValidatorsStateAddress(),
			},
			Errs: []error{storage.ErrNoStateValidatorInfo},
			Fn: func(c *testmodels.TestCase) {
				v := c.TestData.(testmodels.TestData)
				call(t, processor, v.Caller, v.AddrTo, nil, msg, c.Errs)
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

				validator := storage.NewValidator(pubKey, testmodels.Addr4, &withdrawalAddress)

				err = processor.Storage().SetValidator(processor.state, validator)
				testutils.AssertNoError(t, err)

				call(t, processor, v.Caller, v.AddrTo, value, msg, c.Errs)
			},
		},
		{
			CaseName: "Deactivate: validator is exited",
			TestData: testmodels.TestData{
				Caller: vm.AccountRef(from),
				AddrTo: to,
			},
			Errs: []error{ErrValidatorIsOut},
			Fn: func(c *testmodels.TestCase) {
				v := c.TestData.(testmodels.TestData)

				validator := storage.NewValidator(pubKey, testmodels.Addr4, &withdrawalAddress)
				validator.ActivationEra = 0
				validator.ExitEra = procEpoch

				err = processor.Storage().SetValidator(processor.state, validator)
				testutils.AssertNoError(t, err)

				call(t, processor, v.Caller, v.AddrTo, value, msg, c.Errs)
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
				v := c.TestData.(testmodels.TestData)

				validator := storage.NewValidator(pubKey, testmodels.Addr4, &withdrawalAddress)
				validator.ActivationEra = 0

				err = processor.Storage().SetValidator(processor.state, validator)
				testutils.AssertNoError(t, err)

				call(t, processor, v.Caller, v.AddrTo, value, msg, c.Errs)

				val, err := processor.storage.GetValidator(processor.state, testmodels.Addr4)
				testutils.AssertNoError(t, err)

				testutils.AssertEqual(t, uint64(9), val.GetExitEra())
			},
		},
		{
			CaseName: "Deactivate: read era from db",
			TestData: testmodels.TestData{
				Caller: vm.AccountRef(from),
				AddrTo: to,
			},
			Errs: []error{},
			Fn: func(c *testmodels.TestCase) {
				v := c.TestData.(testmodels.TestData)

				validator := storage.NewValidator(pubKey, testmodels.Addr4, &withdrawalAddress)
				validator.ActivationEra = 0
				err = processor.Storage().SetValidator(processor.state, validator)
				testutils.AssertNoError(t, err)

				val, err := processor.storage.GetValidator(processor.state, testmodels.Addr4)
				testutils.AssertNoError(t, err)

				processor.ctx.Era = 3
				rawdb.WriteEra(db, 3, era.Era{
					Number: 3,
					From:   50,
					To:     80,
					Root:   common.BytesToHash(testutils.RandomStringInBytes(32)),
				})

				call(t, processor, v.Caller, v.AddrTo, value, msg, c.Errs)

				val, err = processor.storage.GetValidator(processor.state, testmodels.Addr4)
				testutils.AssertNoError(t, err)

				testutils.AssertEqual(t, uint64(9), val.GetExitEra())
			},
		},
		{
			CaseName: "Deactivate: epoch in transition period",
			TestData: testmodels.TestData{
				Caller: vm.AccountRef(from),
				AddrTo: to,
			},
			Errs: []error{},
			Fn: func(c *testmodels.TestCase) {
				v := c.TestData.(testmodels.TestData)

				validator := storage.NewValidator(pubKey, testmodels.Addr4, &withdrawalAddress)
				validator.ActivationEra = 0
				validator.ExitEra = math.MaxUint64
				err = processor.Storage().SetValidator(processor.state, validator)
				testutils.AssertNoError(t, err)

				val, err := processor.storage.GetValidator(processor.state, testmodels.Addr4)
				testutils.AssertNoError(t, err)

				processor.ctx.Slot = 2790
				processor.ctx.Era = 3
				rawdb.WriteEra(db, 3, era.Era{
					Number: 3,
					From:   44,
					To:     66,
					Root:   common.BytesToHash(testutils.RandomStringInBytes(32)),
				})
				rawdb.WriteEra(db, 4, era.Era{
					Number: 4,
					From:   66,
					To:     88,
					Root:   common.BytesToHash(testutils.RandomStringInBytes(32)),
				})

				call(t, processor, v.Caller, v.AddrTo, value, msg, c.Errs)

				val, err = processor.storage.GetValidator(processor.state, testmodels.Addr4)
				testutils.AssertNoError(t, err)

				testutils.AssertEqual(t, uint64(9), val.GetExitEra())
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
	msg := NewMockmessage(ctrl)

	bc := NewMockblockchain(ctrl)
	bc.EXPECT().Config().AnyTimes().Return(testmodels.TestChainConfig)

	processor := NewProcessor(ctx, stateDb, bc)
	to := processor.GetValidatorsStateAddress()

	withdrawalOperation, err := operation.NewWithdrawalOperation(testmodels.Addr5, value)
	testutils.AssertNoError(t, err)

	opData, err := operation.EncodeToBytes(withdrawalOperation)
	testutils.AssertNoError(t, err)
	msg.EXPECT().Data().AnyTimes().Return(opData)
	msg.EXPECT().TxHash().AnyTimes().Return(common.Hash{0x11})

	cases := []*testmodels.TestCase{
		{
			CaseName: "Withdrawal: invalid address to",
			TestData: testmodels.TestData{
				Caller: vm.AccountRef(from),
				AddrTo: testmodels.Addr1,
			},
			Errs: []error{ErrInvalidToAddress},
			Fn: func(c *testmodels.TestCase) {
				v := c.TestData.(testmodels.TestData)
				call(t, processor, v.Caller, v.AddrTo, nil, msg, c.Errs)
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
				validator := storage.NewValidator(pubKey, testmodels.Addr5, &withdrawalAddress)
				err = processor.Storage().SetValidator(processor.state, validator)
				testutils.AssertNoError(t, err)
				call(t, processor, v.Caller, v.AddrTo, nil, msg, c.Errs)
			},
		},
		{
			CaseName: "Withdrawal: not activated validator",
			TestData: testmodels.TestData{
				Caller: vm.AccountRef(withdrawalAddress),
				AddrTo: to,
			},
			Errs: []error{ErrInvalidFromAddresses},
			Fn: func(c *testmodels.TestCase) {
				v := c.TestData.(testmodels.TestData)
				validator := storage.NewValidator(pubKey, testmodels.Addr5, &withdrawalAddress)
				err = processor.Storage().SetValidator(processor.state, validator)
				testutils.AssertNoError(t, err)
				call(t, processor, v.Caller, v.AddrTo, value, msg, c.Errs)
			},
		},
		{
			CaseName: "Withdrawal: insufficient funds",
			TestData: testmodels.TestData{
				Caller: vm.AccountRef(withdrawalAddress),
				AddrTo: to,
			},
			Errs: []error{ErrInsufficientFundsForOp},
			Fn: func(c *testmodels.TestCase) {
				v := c.TestData.(testmodels.TestData)
				validator := storage.NewValidator(pubKey, testmodels.Addr5, &withdrawalAddress)
				validator.AddStake(withdrawalAddress, big.NewInt(3200/4*2))
				err = processor.Storage().SetValidator(processor.state, validator)
				testutils.AssertNoError(t, err)
				txVal := big.NewInt(3200 / 4 * 3)
				call(t, processor, v.Caller, v.AddrTo, txVal, msg, c.Errs)
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
				v := c.TestData.(testmodels.TestData)
				validator := storage.NewValidator(pubKey, testmodels.Addr5, &withdrawalAddress)
				validator.ActivationEra = 0
				err = processor.Storage().SetValidator(processor.state, validator)
				testutils.AssertNoError(t, err)
				call(t, processor, v.Caller, v.AddrTo, value, msg, c.Errs)
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
	updateBalanceOperation, err := operation.NewValidatorSyncOperation(0, types.UpdateBalance, initTxHash, procEpoch, 0, testmodels.Addr6, value, &withdrawalAddress, nil)
	testutils.AssertNoError(t, err)

	ctrl = gomock.NewController(t)
	defer ctrl.Finish()

	msg := NewMockmessage(ctrl)

	bc := NewMockblockchain(ctrl)
	bc.EXPECT().Config().Return(testmodels.TestChainConfig).AnyTimes()

	bc.EXPECT().GetSlotInfo().AnyTimes().Return(&types.SlotInfo{
		GenesisTime:    uint64(time.Now().Unix()),
		SecondsPerSlot: testmodels.TestChainConfig.SecondsPerSlot,
		SlotsPerEpoch:  testmodels.TestChainConfig.SlotsPerEpoch,
	})
	bc.EXPECT().GetValidatorSyncData(
		gomock.AssignableToTypeOf(common.Hash{})).
		AnyTimes().Return(&types.ValidatorSync{
		OpType:     updateBalanceOperation.OpType(),
		ProcEpoch:  updateBalanceOperation.ProcEpoch(),
		Index:      updateBalanceOperation.Index(),
		Creator:    updateBalanceOperation.Creator(),
		Amount:     updateBalanceOperation.Amount(),
		InitTxHash: initTxHash,
	})
	bc.EXPECT().EpochToEra(uint64(100)).AnyTimes().Return(&testmodels.TestEra)

	processor := NewProcessor(ctx, stateDb, bc)
	to := processor.GetValidatorsStateAddress()

	opData, err := operation.EncodeToBytes(updateBalanceOperation)
	testutils.AssertNoError(t, err)
	msg.EXPECT().Data().AnyTimes().Return(opData)
	msg.EXPECT().TxHash().AnyTimes().Return(common.Hash{})

	//add init tx
	depositOperation, err := operation.NewWithdrawalOperation(testmodels.Addr6, new(big.Int))
	testutils.AssertNoError(t, err)
	initTxData, err := operation.EncodeToBytes(depositOperation)
	testutils.AssertNoError(t, err)
	initTx := types.NewTx(&types.AccessListTx{Data: initTxData})
	bc.EXPECT().GetTransaction(
		common.HexToHash("0x0303030303030303030303030303030303030303030303030303030303030303"),
	).AnyTimes().Return(initTx, common.Hash{}, uint64(0))
	initTxRcp := &types.Receipt{Status: types.ReceiptStatusSuccessful}
	bc.EXPECT().GetTransactionReceipt(
		common.HexToHash("0x0303030303030303030303030303030303030303030303030303030303030303"),
	).AnyTimes().Return(initTxRcp, common.Hash{}, uint64(0))

	cases := []*testmodels.TestCase{
		{
			CaseName: "UpdateBalance: unknown validator",
			TestData: testmodels.TestData{
				Caller: vm.AccountRef(from),
				AddrTo: to,
			},
			Errs: []error{storage.ErrNoStateValidatorInfo},
			Fn: func(c *testmodels.TestCase) {
				v := c.TestData.(testmodels.TestData)
				call(t, processor, v.Caller, v.AddrTo, nil, msg, c.Errs)
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
				v := c.TestData.(testmodels.TestData)
				validator := storage.NewValidator(pubKey, testmodels.Addr6, &withdrawalAddress)
				validator.ActivationEra = 0
				validator.ExitEra = procEpoch
				err = processor.Storage().SetValidator(processor.state, validator)
				testutils.AssertNoError(t, err)
				call(t, processor, v.Caller, v.AddrTo, value, msg, c.Errs)
			},
		},
		{
			CaseName: "UpdateBalance: not activated validator OK",
			TestData: testmodels.TestData{
				Caller: vm.AccountRef(withdrawalAddress),
				AddrTo: to,
			},
			Errs: []error{},
			Fn: func(c *testmodels.TestCase) {
				v := c.TestData.(testmodels.TestData)
				validator := storage.NewValidator(pubKey, testmodels.Addr6, &withdrawalAddress)
				err = processor.Storage().SetValidator(processor.state, validator)
				testutils.AssertNoError(t, err)
				call(t, processor, v.Caller, v.AddrTo, value, msg, c.Errs)
			},
		},
	}

	for _, c := range cases {
		t.Run(c.CaseName, func(t *testing.T) {
			c.Fn(c)
		})
	}
}

func TestProcessorValidatorSyncProcessing(t *testing.T) {
	ctrl = gomock.NewController(t)
	defer ctrl.Finish()

	valSyncData := types.ValidatorSync{
		OpType:     0,
		ProcEpoch:  procEpoch,
		Index:      index,
		Creator:    creatorAddress,
		Amount:     amount,
		TxHash:     nil,
		InitTxHash: common.Hash{1, 2, 3},
	}

	msg := NewMockmessage(ctrl)

	bc := NewMockblockchain(ctrl)
	bc.EXPECT().Config().AnyTimes().Return(testmodels.TestChainConfig)
	bc.EXPECT().GetSlotInfo().AnyTimes().Return(&types.SlotInfo{
		GenesisTime:    168752223,
		SecondsPerSlot: 4,
		SlotsPerEpoch:  32,
	})

	processor := NewProcessor(ctx, stateDb, bc)

	activateOperation, err := operation.NewValidatorSyncOperation(0, types.Activate, initTxHash, procEpoch, index, creatorAddress, big.NewInt(123), &withdrawalAddress, nil)
	testutils.AssertNoError(t, err)

	opData, err := operation.EncodeToBytes(activateOperation)
	testutils.AssertNoError(t, err)
	msg.EXPECT().Data().AnyTimes().Return(opData)
	txHash := crypto.Keccak256Hash(opData)
	msg.EXPECT().TxHash().AnyTimes().Return(txHash)
	cases := []*testmodels.TestCase{
		{
			CaseName: "ErrInvalidOpEpoch",
			TestData: testmodels.TestData{
				Caller: vm.AccountRef(from),
				AddrTo: processor.GetValidatorsStateAddress(),
			},
			Errs: []error{ErrInvalidOpEpoch},
			Fn: func(c *testmodels.TestCase) {
				v := c.TestData.(testmodels.TestData)
				valSyncData.Amount = nil
				bc.EXPECT().GetValidatorSyncData(initTxHash).Return(&valSyncData)
				processor.ctx.Slot = math.MaxUint64
				call(t, processor, v.Caller, v.AddrTo, value, msg, c.Errs)
				processor.ctx.Slot = 0
			},
		},
		{
			CaseName: "ErrNoSavedValSyncOp",
			TestData: testmodels.TestData{
				Caller: vm.AccountRef(from),
				AddrTo: processor.GetValidatorsStateAddress(),
			},
			Errs: []error{ErrNoSavedValSyncOp},
			Fn: func(c *testmodels.TestCase) {
				v := c.TestData.(testmodels.TestData)
				bc.EXPECT().GetValidatorSyncData(initTxHash).Return(nil)
				call(t, processor, v.Caller, v.AddrTo, value, msg, c.Errs)
			},
		},
		{
			CaseName: "ErrMismatchTxHashes",
			TestData: testmodels.TestData{
				Caller: vm.AccountRef(from),
				AddrTo: processor.GetValidatorsStateAddress(),
			},
			Errs: []error{ErrMismatchTxHashes},
			Fn: func(c *testmodels.TestCase) {
				v := c.TestData.(testmodels.TestData)
				hash := common.BytesToHash(testutils.RandomStringInBytes(32))
				valSyncData.TxHash = &hash
				valSyncData.InitTxHash = common.HexToHash("0x0303030303030303030303030303030303030303030303030303030303030303")
				bc.EXPECT().GetValidatorSyncData(initTxHash).Return(&valSyncData)
				call(t, processor, v.Caller, v.AddrTo, value, msg, c.Errs)
				valSyncData.TxHash = &txHash
			},
		},
		{
			CaseName: "ErrMismatchValSyncOp: different op type",
			TestData: testmodels.TestData{
				Caller: vm.AccountRef(from),
				AddrTo: processor.GetValidatorsStateAddress(),
			},
			Errs: []error{ErrMismatchValSyncOp},
			Fn: func(c *testmodels.TestCase) {
				v := c.TestData.(testmodels.TestData)
				valSyncData.OpType = types.Deactivate
				bc.EXPECT().GetValidatorSyncData(initTxHash).Return(&valSyncData)
				call(t, processor, v.Caller, v.AddrTo, value, msg, c.Errs)
				valSyncData.OpType = types.Activate
			},
		},
		{
			CaseName: "ErrMismatchValSyncOp: different creator",
			TestData: testmodels.TestData{
				Caller: vm.AccountRef(from),
				AddrTo: processor.GetValidatorsStateAddress(),
			},
			Errs: []error{ErrMismatchValSyncOp},
			Fn: func(c *testmodels.TestCase) {
				v := c.TestData.(testmodels.TestData)
				valSyncData.Creator = withdrawalAddress
				bc.EXPECT().GetValidatorSyncData(initTxHash).Return(&valSyncData)
				call(t, processor, v.Caller, v.AddrTo, value, msg, c.Errs)
				valSyncData.Creator = creatorAddress
			},
		},
		{
			CaseName: "ErrMismatchValSyncOp: different index",
			TestData: testmodels.TestData{
				Caller: vm.AccountRef(from),
				AddrTo: processor.GetValidatorsStateAddress(),
			},
			Errs: []error{ErrMismatchValSyncOp},
			Fn: func(c *testmodels.TestCase) {
				v := c.TestData.(testmodels.TestData)
				valSyncData.Index = index + 1
				bc.EXPECT().GetValidatorSyncData(initTxHash).Return(&valSyncData)
				call(t, processor, v.Caller, v.AddrTo, value, msg, c.Errs)
				valSyncData.Index = index
			},
		},
		{
			CaseName: "ErrMismatchValSyncOp: different process epoch",
			TestData: testmodels.TestData{
				Caller: vm.AccountRef(from),
				AddrTo: processor.GetValidatorsStateAddress(),
			},
			Errs: []error{ErrMismatchValSyncOp},
			Fn: func(c *testmodels.TestCase) {
				v := c.TestData.(testmodels.TestData)
				valSyncData.ProcEpoch = procEpoch + 1
				bc.EXPECT().GetValidatorSyncData(initTxHash).Return(&valSyncData)
				call(t, processor, v.Caller, v.AddrTo, value, msg, c.Errs)
				valSyncData.ProcEpoch = procEpoch
			},
		},
		{
			CaseName: "ErrMismatchValSyncOp: different amount",
			TestData: testmodels.TestData{
				Caller: vm.AccountRef(from),
				AddrTo: processor.GetValidatorsStateAddress(),
			},
			Errs: []error{ErrMismatchValSyncOp},
			Fn: func(c *testmodels.TestCase) {
				v := c.TestData.(testmodels.TestData)
				valSyncData.Amount = amount.Add(amount, amount)
				bc.EXPECT().GetValidatorSyncData(initTxHash).Return(&valSyncData)
				call(t, processor, v.Caller, v.AddrTo, value, msg, c.Errs)
			},
		},
	}

	for _, c := range cases {
		t.Run(c.CaseName, func(t *testing.T) {
			c.Fn(c)
		})
	}
}

func call(t *testing.T, processor *Processor, Caller Ref, addrTo common.Address, value *big.Int, msg message, Errs []error) []byte {
	res, err := processor.Call(Caller, addrTo, value, msg)
	if !testutils.CheckError(err, Errs) {
		t.Fatalf("Case failed\nwant errors: %s\nhave errors: %s", Errs, err)
	}

	return res
}
