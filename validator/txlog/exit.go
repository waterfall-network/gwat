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

package txlog

import (
	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/validator/operation"
)

const (
	MinExitRequestLogDataLength = common.BlsPubKeyLength + common.AddressLength + common.Uint64Size
)

func PackExitRequestLogData(
	pubkey common.BlsPubKey,
	creatorAddress common.Address,
	valIndex uint64,
	exitAfterEpoch *uint64,
) []byte {
	data := make([]byte, 0, MinExitRequestLogDataLength)
	data = append(data, pubkey.Bytes()...)
	data = append(data, creatorAddress.Bytes()...)
	data = append(data, common.Uint64ToBytes(valIndex)...)

	if exitAfterEpoch != nil {
		data = append(data, common.Uint64ToBytes(*exitAfterEpoch)...)
	}

	return data
}

func UnpackExitRequestLogData(data []byte) (
	pubkey common.BlsPubKey,
	creatorAddress common.Address,
	valIndex uint64,
	exitAfterEpoch *uint64,
	err error,
) {
	if len(data) != MinExitRequestLogDataLength && len(data) != MinExitRequestLogDataLength+common.Uint64Size {
		err = operation.ErrBadDataLen
		return
	}

	pubkey = common.BytesToBlsPubKey(data[:common.BlsPubKeyLength])
	offset := common.BlsPubKeyLength

	creatorAddress = common.BytesToAddress(data[offset : offset+common.AddressLength])
	offset += common.AddressLength

	valIndex = common.BytesToUint64(data[offset : offset+common.Uint64Size])
	offset += common.Uint64Size

	rawExitEpoch := data[offset:]
	if len(rawExitEpoch) == common.Uint64Size {
		exitEpoch := common.BytesToUint64(rawExitEpoch)
		exitAfterEpoch = &exitEpoch
	}
	return
}
