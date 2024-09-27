// Copyright 2024   Blue Wave Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package operation

import (
	"testing"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/tests/testutils"
)

func TestVerifyDepositSig_OK(t *testing.T) {
	var (
		pubkey             = common.HexToBlsPubKey("931f74533c800ebb6d4b4330a9f7ad609314303c01ca7cd235635fe30fcaa33cdcc2c09e9a07d22d7126e0a078657cbe")
		creator_address    = common.HexToAddress("0x6e9e76fa278190cfb2404e5923d3ccd7e8f6c777")
		withdrawal_address = common.HexToAddress("0xa7e558cc6efa1c41270ef4aa227b3dd6b4a3951e")
		signature          = common.HexToBlsSig("0xa4798654cec11445dcb58eac0fc21a5f668ad7709c0bfdd0265793710f781d7bdab9469936cc528f77eab5ee78eb9b1807ec450a146eceeedb0687deea17d56972800abc1f4c65b6026d23a264443b71efc1f040495e6a7499cac2f944a7cf28")
	)
	err := VerifyDepositSig(signature, pubkey, creator_address, withdrawal_address)
	testutils.AssertNoError(t, err)
}
