package txlog

import (
	"testing"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/tests/testutils"
)

func TestDeactivateLogData_Copy(t *testing.T) {
	upBalData := DeactivateLogData{
		InitTxHash:     common.Hash{0x11},
		CreatorAddress: common.Address{0x22},
		ProcEpoch:      6,
		ValidatorIndex: 456,
	}

	cmp := *upBalData.Copy()
	testutils.AssertEqual(t, upBalData, cmp)

	upBalDataEmpty := DeactivateLogData{}
	cmpEmpty := *upBalDataEmpty.Copy()
	testutils.AssertEqual(t, upBalDataEmpty, cmpEmpty)
}

func TestDeactivateLogData_Marshaling(t *testing.T) {
	upBalData := &DeactivateLogData{
		InitTxHash:     common.Hash{0x11},
		CreatorAddress: common.Address{0x22},
		ProcEpoch:      6,
		ValidatorIndex: 456,
	}

	bin, err := upBalData.MarshalBinary()
	testutils.AssertNoError(t, err)

	unmarshaled := &DeactivateLogData{}
	err = unmarshaled.UnmarshalBinary(bin)
	testutils.AssertNoError(t, err)
	testutils.AssertEqual(t, upBalData, unmarshaled)
}

func TestPackDeactivateLogData(t *testing.T) {
	var (
		initTxHash     = common.Hash{0x11}
		creatorAddress = common.Address{0x22}
		procEpoch      = uint64(753)
		validatorIndex = uint64(456)
		binLogData     = common.Hex2Bytes("f83ca01100000000000000000000000000000000000000000000000000000000000000" +
			"9422000000000000000000000000000000000000008202f18201c8")
	)
	data, err := PackDeactivateLogData(initTxHash, creatorAddress, procEpoch, validatorIndex)
	testutils.AssertNoError(t, err)
	testutils.AssertEqual(t, binLogData, data)
}

func TestUnpackDeactivateLogData(t *testing.T) {
	var (
		initTxHash     = common.Hash{0x11}
		creatorAddress = common.Address{0x22}
		procEpoch      = uint64(753)
		validatorIndex = uint64(456)
		binLogData     = common.Hex2Bytes("f83ca01100000000000000000000000000000000000000000000000000000000000000" +
			"9422000000000000000000000000000000000000008202f18201c8")
	)
	initTx, creator, proc, vix, err := UnpackDeactivateLogData(binLogData)
	testutils.AssertNoError(t, err)
	testutils.AssertEqual(t, initTxHash, initTx)
	testutils.AssertEqual(t, creatorAddress, creator)
	testutils.AssertEqual(t, procEpoch, proc)
	testutils.AssertEqual(t, validatorIndex, vix)
}
