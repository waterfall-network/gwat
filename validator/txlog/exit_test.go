package txlog

import (
	"testing"

	"gitlab.waterfall.network/waterfall/protocol/gwat/tests/testutils"
)

var (
	exitEpoch               = uint64(100)
	exitLogWithExitEpoch    = []byte{151, 40, 188, 115, 60, 143, 206, 221, 224, 195, 163, 61, 172, 18, 218, 62, 187, 170, 14, 183, 77, 129, 58, 52, 182, 0, 82, 14, 121, 118, 162, 96, 216, 95, 5, 118, 135, 232, 201, 35, 213, 44, 120, 113, 85, 21, 52, 141, 167, 229, 88, 204, 110, 250, 28, 65, 39, 14, 244, 170, 34, 123, 61, 214, 180, 163, 149, 30, 123, 0, 0, 0, 0, 0, 0, 0, 100, 0, 0, 0, 0, 0, 0, 0}
	exitLogWithOutExitEpoch = []byte{151, 40, 188, 115, 60, 143, 206, 221, 224, 195, 163, 61, 172, 18, 218, 62, 187, 170, 14, 183, 77, 129, 58, 52, 182, 0, 82, 14, 121, 118, 162, 96, 216, 95, 5, 118, 135, 232, 201, 35, 213, 44, 120, 113, 85, 21, 52, 141, 167, 229, 88, 204, 110, 250, 28, 65, 39, 14, 244, 170, 34, 123, 61, 214, 180, 163, 149, 30, 123, 0, 0, 0, 0, 0, 0, 0}
)

func TestPackExitRequestLogData(t *testing.T) {
	// Have exit epoch
	data := PackExitRequestLogData(testPubkey, testCreatorAddress, testIndex, &exitEpoch)
	testutils.AssertEqual(t, exitLogWithExitEpoch, data)
	// Exit epoch is nil
	data = PackExitRequestLogData(testPubkey, testCreatorAddress, testIndex, nil)
	testutils.AssertEqual(t, exitLogWithOutExitEpoch, data)
}

func TestUnpackExitRequestLogData(t *testing.T) {
	// Have exit epoch
	pKey, creator, valIndex, exitAfter, err := UnpackExitRequestLogData(exitLogWithExitEpoch)
	testutils.AssertEqual(t, testPubkey, pKey)
	testutils.AssertEqual(t, testCreatorAddress, creator)
	testutils.AssertEqual(t, testIndex, valIndex)
	testutils.AssertEqual(t, exitEpoch, *exitAfter)
	testutils.AssertNoError(t, err)

	//	Exit epoch is nil
	pKey, creator, valIndex, exitAfter, err = UnpackExitRequestLogData(exitLogWithOutExitEpoch)
	testutils.AssertEqual(t, testPubkey, pKey)
	testutils.AssertEqual(t, testCreatorAddress, creator)
	testutils.AssertEqual(t, testIndex, valIndex)
	testutils.AssertNil(t, exitAfter)
	testutils.AssertNoError(t, err)
}
