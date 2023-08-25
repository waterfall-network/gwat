package validator

import (
	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/tests/testutils"
	"math/big"
	"testing"
)

var (
	pubkey         = common.HexToBlsPubKey("0x9728bc733c8fcedde0c3a33dac12da3ebbaa0eb74d813a34b600520e7976a260d85f057687e8c923d52c78715515348d")
	creatorAddress = common.HexToAddress("0xa7e558cc6efa1c41270ef4aa227b3dd6b4a3951e")
	withdrawal     = common.HexToAddress("0xa7e558cc6efa1c41270ef4aa227b3dd6b4a3951e")
	amount         = big.NewInt(1000000000) // 1 ETH
	sign           = common.HexToBlsSig("0xb9221f2308c1e1655a8e1977f32241384fa77efedbb3079bcc9a95930152ee87" +
		"f341134a4e59c3e312ee5c2197732ea30d9aac2993cc4aad75335009815d07a8735f96c6dde443ba3a10f5523c4d00f6b3a7b48af" +
		"5a42795183ab5aa2f1b2dd1")
	index                   = uint64(123)
	exitEpoch               = uint64(100)
	depositLogData          = []byte{151, 40, 188, 115, 60, 143, 206, 221, 224, 195, 163, 61, 172, 18, 218, 62, 187, 170, 14, 183, 77, 129, 58, 52, 182, 0, 82, 14, 121, 118, 162, 96, 216, 95, 5, 118, 135, 232, 201, 35, 213, 44, 120, 113, 85, 21, 52, 141, 167, 229, 88, 204, 110, 250, 28, 65, 39, 14, 244, 170, 34, 123, 61, 214, 180, 163, 149, 30, 167, 229, 88, 204, 110, 250, 28, 65, 39, 14, 244, 170, 34, 123, 61, 214, 180, 163, 149, 30, 1, 0, 0, 0, 0, 0, 0, 0, 185, 34, 31, 35, 8, 193, 225, 101, 90, 142, 25, 119, 243, 34, 65, 56, 79, 167, 126, 254, 219, 179, 7, 155, 204, 154, 149, 147, 1, 82, 238, 135, 243, 65, 19, 74, 78, 89, 195, 227, 18, 238, 92, 33, 151, 115, 46, 163, 13, 154, 172, 41, 147, 204, 74, 173, 117, 51, 80, 9, 129, 93, 7, 168, 115, 95, 150, 198, 221, 228, 67, 186, 58, 16, 245, 82, 60, 77, 0, 246, 179, 167, 180, 138, 245, 164, 39, 149, 24, 58, 181, 170, 47, 27, 45, 209, 123, 0, 0, 0, 0, 0, 0, 0}
	exitLogWithExitEpoch    = []byte{151, 40, 188, 115, 60, 143, 206, 221, 224, 195, 163, 61, 172, 18, 218, 62, 187, 170, 14, 183, 77, 129, 58, 52, 182, 0, 82, 14, 121, 118, 162, 96, 216, 95, 5, 118, 135, 232, 201, 35, 213, 44, 120, 113, 85, 21, 52, 141, 167, 229, 88, 204, 110, 250, 28, 65, 39, 14, 244, 170, 34, 123, 61, 214, 180, 163, 149, 30, 123, 0, 0, 0, 0, 0, 0, 0, 100, 0, 0, 0, 0, 0, 0, 0}
	exitLogWithOutExitEpoch = []byte{151, 40, 188, 115, 60, 143, 206, 221, 224, 195, 163, 61, 172, 18, 218, 62, 187, 170, 14, 183, 77, 129, 58, 52, 182, 0, 82, 14, 121, 118, 162, 96, 216, 95, 5, 118, 135, 232, 201, 35, 213, 44, 120, 113, 85, 21, 52, 141, 167, 229, 88, 204, 110, 250, 28, 65, 39, 14, 244, 170, 34, 123, 61, 214, 180, 163, 149, 30, 123, 0, 0, 0, 0, 0, 0, 0}
)

func TestPackDepositLogData(t *testing.T) {
	data := PackDepositLogData(pubkey, creatorAddress, withdrawal, amount, sign, index)
	testutils.AssertEqual(t, depositLogData, data)
}

func TestUnpackDepositLogData(t *testing.T) {
	pkey, creator, withdrowalAddr, depositAmount, signature, depositIndex, err := UnpackDepositLogData(depositLogData)
	testutils.AssertEqual(t, pubkey, pkey)
	testutils.AssertEqual(t, creatorAddress, creator)
	testutils.AssertEqual(t, withdrawal, withdrowalAddr)
	testutils.BigIntEquals(amount, big.NewInt(int64(depositAmount)))
	testutils.AssertEqual(t, sign, signature)
	testutils.AssertEqual(t, index, depositIndex)
	testutils.AssertNoError(t, err)
}

func TestPackExitRequestLogData(t *testing.T) {
	// Have exit epoch
	data := PackExitRequestLogData(pubkey, creatorAddress, index, &exitEpoch)
	testutils.AssertEqual(t, exitLogWithExitEpoch, data)
	// Exit epoch is nil
	data = PackExitRequestLogData(pubkey, creatorAddress, index, nil)
	testutils.AssertEqual(t, exitLogWithOutExitEpoch, data)
}

func TestUnpackExitRequestLogData(t *testing.T) {
	// Have exit epoch
	pKey, creator, valIndex, exitAfter, err := UnpackExitRequestLogData(exitLogWithExitEpoch)
	testutils.AssertEqual(t, pubkey, pKey)
	testutils.AssertEqual(t, creatorAddress, creator)
	testutils.AssertEqual(t, index, valIndex)
	testutils.AssertEqual(t, exitEpoch, *exitAfter)
	testutils.AssertNoError(t, err)

	//	Exit epoch is nil
	pKey, creator, valIndex, exitAfter, err = UnpackExitRequestLogData(exitLogWithOutExitEpoch)
	testutils.AssertEqual(t, pubkey, pKey)
	testutils.AssertEqual(t, creatorAddress, creator)
	testutils.AssertEqual(t, index, valIndex)
	testutils.AssertNil(t, exitAfter)
	testutils.AssertNoError(t, err)
}

func TestPackWithdrawalLogData(t *testing.T) {
	withdrawalLogData := []byte{151, 40, 188, 115, 60, 143, 206, 221, 224, 195, 163, 61, 172, 18, 218, 62, 187, 170, 14, 183, 77, 129, 58, 52, 182, 0, 82, 14, 121, 118, 162, 96, 216, 95, 5, 118, 135, 232, 201, 35, 213, 44, 120, 113, 85, 21, 52, 141, 167, 229, 88, 204, 110, 250, 28, 65, 39, 14, 244, 170, 34, 123, 61, 214, 180, 163, 149, 30, 123, 0, 0, 0, 0, 0, 0, 0, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	withdLogDataAmt_0 := []byte{151, 40, 188, 115, 60, 143, 206, 221, 224, 195, 163, 61, 172, 18, 218, 62, 187, 170, 14, 183, 77, 129, 58, 52, 182, 0, 82, 14, 121, 118, 162, 96, 216, 95, 5, 118, 135, 232, 201, 35, 213, 44, 120, 113, 85, 21, 52, 141, 167, 229, 88, 204, 110, 250, 28, 65, 39, 14, 244, 170, 34, 123, 61, 214, 180, 163, 149, 30, 123, 0, 0, 0, 0, 0, 0, 0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}

	data := PackWithdrawalLogData(pubkey, creatorAddress, index, 0xffffffffffffffff)
	testutils.AssertEqual(t, withdrawalLogData, data)
	// Exit amt == 0
	data = PackWithdrawalLogData(pubkey, creatorAddress, index, 0x00)
	testutils.AssertEqual(t, withdLogDataAmt_0, data)
}

func TestUnpackWithdrawalLogData(t *testing.T) {
	withdrawalLogData := []byte{151, 40, 188, 115, 60, 143, 206, 221, 224, 195, 163, 61, 172, 18, 218, 62, 187, 170, 14, 183, 77, 129, 58, 52, 182, 0, 82, 14, 121, 118, 162, 96, 216, 95, 5, 118, 135, 232, 201, 35, 213, 44, 120, 113, 85, 21, 52, 141, 167, 229, 88, 204, 110, 250, 28, 65, 39, 14, 244, 170, 34, 123, 61, 214, 180, 163, 149, 30, 123, 0, 0, 0, 0, 0, 0, 0, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	withdLogDataAmt_0 := []byte{151, 40, 188, 115, 60, 143, 206, 221, 224, 195, 163, 61, 172, 18, 218, 62, 187, 170, 14, 183, 77, 129, 58, 52, 182, 0, 82, 14, 121, 118, 162, 96, 216, 95, 5, 118, 135, 232, 201, 35, 213, 44, 120, 113, 85, 21, 52, 141, 167, 229, 88, 204, 110, 250, 28, 65, 39, 14, 244, 170, 34, 123, 61, 214, 180, 163, 149, 30, 123, 0, 0, 0, 0, 0, 0, 0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}

	// Have exit epoch
	pKey, creator, valIndex, gwAmt, err := UnpackWithdrawalLogData(withdrawalLogData)
	testutils.AssertEqual(t, pubkey, pKey)
	testutils.AssertEqual(t, creatorAddress, creator)
	testutils.AssertEqual(t, index, valIndex)
	testutils.AssertEqual(t, uint64(0xffffffffffffffff), gwAmt)
	testutils.AssertNoError(t, err)

	//	Exit epoch is nil
	pKey, creator, valIndex, gwAmt, err = UnpackWithdrawalLogData(withdLogDataAmt_0)
	testutils.AssertEqual(t, pubkey, pKey)
	testutils.AssertEqual(t, creatorAddress, creator)
	testutils.AssertEqual(t, index, valIndex)
	testutils.AssertEqual(t, uint64(0x00), gwAmt)
	testutils.AssertNoError(t, err)
}
