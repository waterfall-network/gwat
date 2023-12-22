package operation

import (
	"fmt"
	"testing"
	"time"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/tests/testutils"
)

func TestDelegateStakeData_NewDelegateStakeData(t *testing.T) {
	profitShare, stakeShare, exit, withdrawal := TestParamsDelegateStakeRules()
	trialPeriod := uint64(321)

	rules, err := NewDelegateStakeRules(profitShare, stakeShare, exit, withdrawal)
	testutils.AssertNoError(t, err)
	trialRules, err := NewDelegateStakeRules(profitShare, stakeShare, exit, withdrawal)
	testutils.AssertNoError(t, err)

	dsr, err := NewDelegateStakeData(rules, trialPeriod, trialRules)
	testutils.AssertNoError(t, err)
	testutils.AssertEqual(t, profitShare, dsr.Rules.ProfitShare())
	testutils.AssertEqual(t, stakeShare, dsr.Rules.StakeShare())
	testutils.AssertEqual(t, exit, dsr.Rules.Exit())
	testutils.AssertEqual(t, withdrawal, dsr.Rules.Withdrawal())
	testutils.AssertEqual(t, trialPeriod, dsr.TrialPeriod)
	testutils.AssertEqual(t, profitShare, dsr.TrialRules.ProfitShare())
	testutils.AssertEqual(t, stakeShare, dsr.TrialRules.StakeShare())
	testutils.AssertEqual(t, exit, dsr.TrialRules.Exit())
	testutils.AssertEqual(t, withdrawal, dsr.TrialRules.Withdrawal())

	// empty instance
	dsrEmpty, err := NewDelegateStakeData(nil, 0, nil)
	testutils.AssertNoError(t, err)
	testutils.AssertEqual(t, map[common.Address]uint8{}, dsrEmpty.Rules.ProfitShare())
	testutils.AssertEqual(t, map[common.Address]uint8{}, dsrEmpty.Rules.StakeShare())
	testutils.AssertEqual(t, []common.Address{}, dsrEmpty.Rules.Exit())
	testutils.AssertEqual(t, []common.Address{}, dsrEmpty.Rules.Withdrawal())
	testutils.AssertEqual(t, uint64(0), dsrEmpty.TrialPeriod)
	testutils.AssertEqual(t, map[common.Address]uint8{}, dsrEmpty.TrialRules.ProfitShare())
	testutils.AssertEqual(t, map[common.Address]uint8{}, dsrEmpty.TrialRules.StakeShare())
	testutils.AssertEqual(t, []common.Address{}, dsrEmpty.TrialRules.Exit())
	testutils.AssertEqual(t, []common.Address{}, dsrEmpty.TrialRules.Withdrawal())
}

func TestDelegateStakeData_Marshaling(t *testing.T) {
	defer func(tStart time.Time) {
		fmt.Println("TOTAL TIME",
			"elapsed", common.PrettyDuration(time.Since(tStart)),
		)
	}(time.Now())

	profitShare, stakeShare, exit, withdrawal := TestParamsDelegateStakeRules()
	trialPeriod := uint64(321)

	rules, err := NewDelegateStakeRules(profitShare, stakeShare, exit, withdrawal)
	testutils.AssertNoError(t, err)
	trialRules, err := NewDelegateStakeRules(profitShare, stakeShare, exit, withdrawal)
	testutils.AssertNoError(t, err)

	dsr, err := NewDelegateStakeData(rules, trialPeriod, trialRules)
	testutils.AssertNoError(t, err)

	bin, err := dsr.MarshalBinary()
	testutils.AssertNoError(t, err)

	unmarshaled := &DelegatedStakeData{}
	err = unmarshaled.UnmarshalBinary(bin)
	testutils.AssertNoError(t, err)

	testutils.AssertEqual(t, dsr.Rules, unmarshaled.Rules)
	testutils.AssertEqual(t, dsr.TrialPeriod, unmarshaled.TrialPeriod)
	testutils.AssertEqual(t, dsr.TrialRules, unmarshaled.TrialRules)

	//// empty instance
	dsrEmpty, err := NewDelegateStakeData(nil, 0, nil)
	testutils.AssertNoError(t, err)

	binEmpty, err := dsrEmpty.MarshalBinary()
	testutils.AssertNoError(t, err)

	unmarshaledEmpty := &DelegatedStakeData{}
	err = unmarshaled.UnmarshalBinary(binEmpty)
	testutils.AssertNoError(t, err)

	testutils.AssertEqual(t, dsrEmpty.Rules, unmarshaledEmpty.Rules)
	testutils.AssertEqual(t, dsrEmpty.TrialPeriod, unmarshaledEmpty.TrialPeriod)
	testutils.AssertEqual(t, dsrEmpty.TrialRules, unmarshaledEmpty.TrialRules)

	// nil instance
	var dsrNil *DelegatedStakeData
	binNil, err := dsrNil.MarshalBinary()
	testutils.AssertNoError(t, err)

	unmarshaledNil := &DelegatedStakeData{}
	err = unmarshaledNil.UnmarshalBinary(binNil)
	testutils.AssertError(t, err, errDelegateStakeNilValBin)
}

func TestDelegateStakeData_NewDelegateStakeDataFromBinary(t *testing.T) {
	profitShare, stakeShare, exit, withdrawal := TestParamsDelegateStakeRules()
	trialPeriod := uint64(321)

	rules, err := NewDelegateStakeRules(profitShare, stakeShare, exit, withdrawal)
	testutils.AssertNoError(t, err)
	trialRules, err := NewDelegateStakeRules(profitShare, stakeShare, exit, withdrawal)
	testutils.AssertNoError(t, err)

	dsr, err := NewDelegateStakeData(rules, trialPeriod, trialRules)
	testutils.AssertNoError(t, err)

	bin, err := dsr.MarshalBinary()
	testutils.AssertNoError(t, err)

	dsrBin, err := NewDelegateStakeDataFromBinary(bin)
	testutils.AssertNoError(t, err)

	testutils.AssertEqual(t, dsr.Rules, dsrBin.Rules)
	testutils.AssertEqual(t, dsr.TrialPeriod, dsrBin.TrialPeriod)
	testutils.AssertEqual(t, dsr.TrialRules, dsrBin.TrialRules)

	//// empty instance
	dsrEmpty, err := NewDelegateStakeData(nil, 0, nil)
	testutils.AssertNoError(t, err)

	binEmpty, err := dsrEmpty.MarshalBinary()
	testutils.AssertNoError(t, err)

	dsrBinEmpty, err := NewDelegateStakeDataFromBinary(binEmpty)
	testutils.AssertNoError(t, err)

	testutils.AssertEqual(t, fmt.Sprintf("%#X", dsrEmpty.Rules), fmt.Sprintf("%#X", dsrBinEmpty.Rules))
	testutils.AssertEqual(t, dsrEmpty.TrialPeriod, dsrBinEmpty.TrialPeriod)
	testutils.AssertEqual(t, fmt.Sprintf("%#X", dsrEmpty.TrialRules), fmt.Sprintf("%#X", dsrBinEmpty.TrialRules))

	// nil instance
	var dsrNil *DelegatedStakeData
	binNil, err := dsrNil.MarshalBinary()
	testutils.AssertNoError(t, err)

	dsrBinNil, err := NewDelegateStakeDataFromBinary(binNil)
	testutils.AssertNoError(t, err)
	testutils.AssertEqual(t, dsrNil, dsrBinNil)
}
