package txlog

import (
	"testing"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/tests/testutils"
)

func TestDelegatingStakeLogData_Copy(t *testing.T) {
	logData := DelegatingStakeLogData{
		{
			Address:  common.Address{0x11},
			RuleType: StakeShare,
			IsTrial:  true,
			Amount:   common.Big256,
		},
		{
			Address:  common.Address{0x22},
			RuleType: ProfitShare,
			IsTrial:  true,
			Amount:   common.Big257,
		},
	}

	cmp := *logData.Copy()
	testutils.AssertEqual(t, logData, cmp)

	upBalDataEmpty := DelegatingStakeLogData{}
	cmpEmpty := *upBalDataEmpty.Copy()
	testutils.AssertEqual(t, upBalDataEmpty, cmpEmpty)
}

func TestDelegatingStakeLogData_Marshaling(t *testing.T) {
	logData := &DelegatingStakeLogData{
		{
			Address:  common.Address{0x11},
			RuleType: StakeShare,
			IsTrial:  true,
			Amount:   common.Big256,
		},
		{
			Address:  common.Address{0x22},
			RuleType: ProfitShare,
			IsTrial:  true,
			Amount:   common.Big257,
		},
	}

	bin, err := logData.MarshalBinary()
	testutils.AssertNoError(t, err)

	unmarshaled := &DelegatingStakeLogData{}
	err = unmarshaled.UnmarshalBinary(bin)
	testutils.AssertNoError(t, err)
	testutils.AssertEqual(t, logData, unmarshaled)
}

func TestPackDelegatingStakeLogData(t *testing.T) {
	var (
		logData = &DelegatingStakeLogData{
			{
				Address:  common.Address{0x11},
				RuleType: StakeShare,
				IsTrial:  true,
				Amount:   common.Big256,
			},
			{
				Address:  common.Address{0x22},
				RuleType: ProfitShare,
				IsTrial:  true,
				Amount:   common.Big257,
			},
		}
		binLogData = common.Hex2Bytes("f6da9411000000000000000000000000000000000000000201820100da" +
			"9422000000000000000000000000000000000000000101820101")
	)
	data, err := PackDelegatingStakeLogData(logData)
	testutils.AssertNoError(t, err)
	testutils.AssertEqual(t, binLogData, data)
}

func TestUnpackDelegatingStakeLogData(t *testing.T) {
	var (
		logData = &DelegatingStakeLogData{
			{
				Address:  common.Address{0x11},
				RuleType: StakeShare,
				IsTrial:  true,
				Amount:   common.Big256,
			},
			{
				Address:  common.Address{0x22},
				RuleType: ProfitShare,
				IsTrial:  true,
				Amount:   common.Big257,
			},
		}
		binLogData = common.Hex2Bytes("f6da9411000000000000000000000000000000000000000201820100da" +
			"9422000000000000000000000000000000000000000101820101")
	)
	gotData, err := UnpackDelegatingStakeLogData(binLogData)
	testutils.AssertNoError(t, err)
	testutils.AssertEqual(t, logData, gotData)
}
