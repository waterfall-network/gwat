package txlog

import (
	"testing"

	"gitlab.waterfall.network/waterfall/protocol/gwat/tests/testutils"
)

func TestPackWithdrawalLogData(t *testing.T) {
	withdrawalLogData := []byte{151, 40, 188, 115, 60, 143, 206, 221, 224, 195, 163, 61, 172, 18, 218, 62, 187, 170, 14, 183, 77, 129, 58, 52, 182, 0, 82, 14, 121, 118, 162, 96, 216, 95, 5, 118, 135, 232, 201, 35, 213, 44, 120, 113, 85, 21, 52, 141, 167, 229, 88, 204, 110, 250, 28, 65, 39, 14, 244, 170, 34, 123, 61, 214, 180, 163, 149, 30, 123, 0, 0, 0, 0, 0, 0, 0, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	withdLogDataAmt_0 := []byte{151, 40, 188, 115, 60, 143, 206, 221, 224, 195, 163, 61, 172, 18, 218, 62, 187, 170, 14, 183, 77, 129, 58, 52, 182, 0, 82, 14, 121, 118, 162, 96, 216, 95, 5, 118, 135, 232, 201, 35, 213, 44, 120, 113, 85, 21, 52, 141, 167, 229, 88, 204, 110, 250, 28, 65, 39, 14, 244, 170, 34, 123, 61, 214, 180, 163, 149, 30, 123, 0, 0, 0, 0, 0, 0, 0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}

	data := PackWithdrawalLogData(testPubkey, testCreatorAddress, testIndex, 0xffffffffffffffff)
	testutils.AssertEqual(t, withdrawalLogData, data)
	// Exit amt == 0
	data = PackWithdrawalLogData(testPubkey, testCreatorAddress, testIndex, 0x00)
	testutils.AssertEqual(t, withdLogDataAmt_0, data)
}

func TestUnpackWithdrawalLogData(t *testing.T) {
	withdrawalLogData := []byte{151, 40, 188, 115, 60, 143, 206, 221, 224, 195, 163, 61, 172, 18, 218, 62, 187, 170, 14, 183, 77, 129, 58, 52, 182, 0, 82, 14, 121, 118, 162, 96, 216, 95, 5, 118, 135, 232, 201, 35, 213, 44, 120, 113, 85, 21, 52, 141, 167, 229, 88, 204, 110, 250, 28, 65, 39, 14, 244, 170, 34, 123, 61, 214, 180, 163, 149, 30, 123, 0, 0, 0, 0, 0, 0, 0, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	withdLogDataAmt_0 := []byte{151, 40, 188, 115, 60, 143, 206, 221, 224, 195, 163, 61, 172, 18, 218, 62, 187, 170, 14, 183, 77, 129, 58, 52, 182, 0, 82, 14, 121, 118, 162, 96, 216, 95, 5, 118, 135, 232, 201, 35, 213, 44, 120, 113, 85, 21, 52, 141, 167, 229, 88, 204, 110, 250, 28, 65, 39, 14, 244, 170, 34, 123, 61, 214, 180, 163, 149, 30, 123, 0, 0, 0, 0, 0, 0, 0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}

	// Have exit epoch
	pKey, creator, valIndex, gwAmt, err := UnpackWithdrawalLogData(withdrawalLogData)
	testutils.AssertEqual(t, testPubkey, pKey)
	testutils.AssertEqual(t, testCreatorAddress, creator)
	testutils.AssertEqual(t, testIndex, valIndex)
	testutils.AssertEqual(t, uint64(0xffffffffffffffff), gwAmt)
	testutils.AssertNoError(t, err)

	//	Exit epoch is nil
	pKey, creator, valIndex, gwAmt, err = UnpackWithdrawalLogData(withdLogDataAmt_0)
	testutils.AssertEqual(t, testPubkey, pKey)
	testutils.AssertEqual(t, testCreatorAddress, creator)
	testutils.AssertEqual(t, testIndex, valIndex)
	testutils.AssertEqual(t, uint64(0x00), gwAmt)
	testutils.AssertNoError(t, err)
}
