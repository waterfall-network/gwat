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
