package txlog

import (
	"fmt"
	"math/big"
	"testing"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/tests/testutils"
)

var (
	testPubkey         = common.HexToBlsPubKey("0x9728bc733c8fcedde0c3a33dac12da3ebbaa0eb74d813a34b600520e7976a260d85f057687e8c923d52c78715515348d")
	testCreatorAddress = common.HexToAddress("0xa7e558cc6efa1c41270ef4aa227b3dd6b4a3951e")
	testWithdrawal     = common.HexToAddress("0xa7e558cc6efa1c41270ef4aa227b3dd6b4a3951e")
	testAmount         = big.NewInt(1000000000) // 1 ETH
	testSign           = common.HexToBlsSig("0xb9221f2308c1e1655a8e1977f32241384fa77efedbb3079bcc9a95930152ee87" +
		"f341134a4e59c3e312ee5c2197732ea30d9aac2993cc4aad75335009815d07a8735f96c6dde443ba3a10f5523c4d00f6b3a7b48af" +
		"5a42795183ab5aa2f1b2dd1")
	testIndex          = uint64(123)
	testDepositLogData = []byte{151, 40, 188, 115, 60, 143, 206, 221, 224, 195, 163, 61, 172, 18, 218, 62, 187, 170, 14, 183, 77, 129, 58, 52, 182, 0, 82, 14, 121, 118, 162, 96, 216, 95, 5, 118, 135, 232, 201, 35, 213, 44, 120, 113, 85, 21, 52, 141, 167, 229, 88, 204, 110, 250, 28, 65, 39, 14, 244, 170, 34, 123, 61, 214, 180, 163, 149, 30, 167, 229, 88, 204, 110, 250, 28, 65, 39, 14, 244, 170, 34, 123, 61, 214, 180, 163, 149, 30, 1, 0, 0, 0, 0, 0, 0, 0, 185, 34, 31, 35, 8, 193, 225, 101, 90, 142, 25, 119, 243, 34, 65, 56, 79, 167, 126, 254, 219, 179, 7, 155, 204, 154, 149, 147, 1, 82, 238, 135, 243, 65, 19, 74, 78, 89, 195, 227, 18, 238, 92, 33, 151, 115, 46, 163, 13, 154, 172, 41, 147, 204, 74, 173, 117, 51, 80, 9, 129, 93, 7, 168, 115, 95, 150, 198, 221, 228, 67, 186, 58, 16, 245, 82, 60, 77, 0, 246, 179, 167, 180, 138, 245, 164, 39, 149, 24, 58, 181, 170, 47, 27, 45, 209, 123, 0, 0, 0, 0, 0, 0, 0}
)

func TestPackDepositLogData(t *testing.T) {
	var (
		pubkey         = common.HexToBlsPubKey("0x9728bc733c8fcedde0c3a33dac12da3ebbaa0eb74d813a34b600520e7976a260d85f057687e8c923d52c78715515348d")
		creatorAddress = common.HexToAddress("0xa7e558cc6efa1c41270ef4aa227b3dd6b4a3951e")
		withdrawal     = common.HexToAddress("0xa7e558cc6efa1c41270ef4aa227b3dd6b4a3951e")
		amount         = big.NewInt(1000000000) // 1 ETH
		sign           = common.HexToBlsSig("0xb9221f2308c1e1655a8e1977f32241384fa77efedbb3079bcc9a95930152ee87" +
			"f341134a4e59c3e312ee5c2197732ea30d9aac2993cc4aad75335009815d07a8735f96c6dde443ba3a10f5523c4d00f6b3a7b48af" +
			"5a42795183ab5aa2f1b2dd1")
		index          = uint64(123)
		depositLogData = []byte{151, 40, 188, 115, 60, 143, 206, 221, 224, 195, 163, 61, 172, 18, 218, 62, 187, 170, 14, 183, 77, 129, 58, 52, 182, 0, 82, 14, 121, 118, 162, 96, 216, 95, 5, 118, 135, 232, 201, 35, 213, 44, 120, 113, 85, 21, 52, 141, 167, 229, 88, 204, 110, 250, 28, 65, 39, 14, 244, 170, 34, 123, 61, 214, 180, 163, 149, 30, 167, 229, 88, 204, 110, 250, 28, 65, 39, 14, 244, 170, 34, 123, 61, 214, 180, 163, 149, 30, 1, 0, 0, 0, 0, 0, 0, 0, 185, 34, 31, 35, 8, 193, 225, 101, 90, 142, 25, 119, 243, 34, 65, 56, 79, 167, 126, 254, 219, 179, 7, 155, 204, 154, 149, 147, 1, 82, 238, 135, 243, 65, 19, 74, 78, 89, 195, 227, 18, 238, 92, 33, 151, 115, 46, 163, 13, 154, 172, 41, 147, 204, 74, 173, 117, 51, 80, 9, 129, 93, 7, 168, 115, 95, 150, 198, 221, 228, 67, 186, 58, 16, 245, 82, 60, 77, 0, 246, 179, 167, 180, 138, 245, 164, 39, 149, 24, 58, 181, 170, 47, 27, 45, 209, 123, 0, 0, 0, 0, 0, 0, 0}
	)
	data := PackDepositLogData(pubkey, creatorAddress, withdrawal, amount, sign, index)
	testutils.AssertEqual(t, fmt.Sprintf("%#x", depositLogData), fmt.Sprintf("%#x", data))
}

func TestUnpackDepositLogData(t *testing.T) {
	pkey, creator, withdrawalAddr, depositAmount, signature, depositIndex, err := UnpackDepositLogData(testDepositLogData)
	testutils.AssertEqual(t, testPubkey, pkey)
	testutils.AssertEqual(t, testCreatorAddress, creator)
	testutils.AssertEqual(t, testWithdrawal, withdrawalAddr)
	testutils.BigIntEquals(testAmount, big.NewInt(int64(depositAmount)))
	testutils.AssertEqual(t, testSign, signature)
	testutils.AssertEqual(t, testIndex, depositIndex)
	testutils.AssertNoError(t, err)
}
