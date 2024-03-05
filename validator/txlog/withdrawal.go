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
