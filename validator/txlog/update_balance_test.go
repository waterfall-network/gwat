package txlog

import (
	"testing"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/tests/testutils"
)

func TestUpdateBalanceLogData_Copy(t *testing.T) {
	upBalData := UpdateBalanceLogData{
		InitTxHash:     common.Hash{0x11},
		CreatorAddress: common.Address{0x22},
		ProcEpoch:      6,
		Amount:         common.BigGwei,
	}

	cmp := *upBalData.Copy()
	testutils.AssertEqual(t, upBalData, cmp)

	upBalDataEmpty := UpdateBalanceLogData{}
	cmpEmpty := *upBalDataEmpty.Copy()
	testutils.AssertEqual(t, upBalDataEmpty, cmpEmpty)
}

func TestUpdateBalanceLogData_Marshaling(t *testing.T) {
	upBalData := &UpdateBalanceLogData{
		InitTxHash:     common.Hash{0x11},
		CreatorAddress: common.Address{0x22},
		ProcEpoch:      6,
		Amount:         common.BigGwei,
	}

	bin, err := upBalData.MarshalBinary()
	testutils.AssertNoError(t, err)

	unmarshaled := &UpdateBalanceLogData{}
	err = unmarshaled.UnmarshalBinary(bin)
	testutils.AssertNoError(t, err)
	testutils.AssertEqual(t, upBalData, unmarshaled)
}

func TestPackUpdateBalanceLogData(t *testing.T) {
	var (
		initTxHash     = common.Hash{0x11}
		creatorAddress = common.Address{0x22}
		procEpoch      = uint64(753)
		amount         = common.BigGwei
		binLogData     = common.Hex2Bytes("f83ea01100000000000000000000000000000000000000000000000000000000000000" +
			"9422000000000000000000000000000000000000008202f1843b9aca00")
	)
	data, err := PackUpdateBalanceLogData(initTxHash, creatorAddress, procEpoch, amount)
	testutils.AssertNoError(t, err)
	testutils.AssertEqual(t, binLogData, data)
}

func TestUnpackUpdateBalanceLogData(t *testing.T) {
	var (
		initTxHash     = common.Hash{0x11}
		creatorAddress = common.Address{0x22}
		procEpoch      = uint64(753)
		amount         = common.BigGwei
		binLogData     = common.Hex2Bytes("f83ea01100000000000000000000000000000000000000000000000000000000000000" +
			"9422000000000000000000000000000000000000008202f1843b9aca00")
	)
	initTx, creator, proc, amt, err := UnpackUpdateBalanceLogData(binLogData)
	testutils.AssertNoError(t, err)
	testutils.AssertEqual(t, initTxHash, initTx)
	testutils.AssertEqual(t, creatorAddress, creator)
	testutils.AssertEqual(t, procEpoch, proc)
	testutils.AssertEqual(t, amount, amt)
}
