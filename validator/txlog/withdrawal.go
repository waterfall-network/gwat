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
	WithdrawalLogDataLength = common.BlsPubKeyLength + common.AddressLength + common.Uint64Size + common.Uint64Size
)

func PackWithdrawalLogData(
	pubkey common.BlsPubKey,
	creatorAddress common.Address,
	valIndex uint64,
	amtGwei uint64,
) []byte {
	data := make([]byte, 0, WithdrawalLogDataLength)
	data = append(data, pubkey.Bytes()...)
	data = append(data, creatorAddress.Bytes()...)
	data = append(data, common.Uint64ToBytes(valIndex)...)
	data = append(data, common.Uint64ToBytes(amtGwei)...)

	return data
}

func UnpackWithdrawalLogData(data []byte) (
	pubkey common.BlsPubKey,
	creatorAddress common.Address,
	valIndex uint64,
	amtGwei uint64,
	err error,
) {
	if len(data) != WithdrawalLogDataLength {
		err = operation.ErrBadDataLen
		return
	}

	pubkey = common.BytesToBlsPubKey(data[:common.BlsPubKeyLength])
	offset := common.BlsPubKeyLength

	creatorAddress = common.BytesToAddress(data[offset : offset+common.AddressLength])
	offset += common.AddressLength

	valIndex = common.BytesToUint64(data[offset : offset+common.Uint64Size])
	offset += common.Uint64Size

	amtGwei = common.BytesToUint64(data[offset : offset+common.Uint64Size])

	return
}
