package validator

import (
	"math/big"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/validator/operation"
)

const Uint64Length = 8

const (
	DepositLogDataLength        = common.BlsPubKeyLength + common.AddressLength + common.AddressLength + Uint64Length + common.BlsSigLength + Uint64Length
	MinExitRequestLogDataLength = common.BlsPubKeyLength + common.AddressLength + Uint64Length
	WithdrawalLogDataLength     = common.BlsPubKeyLength + common.AddressLength + Uint64Length + Uint64Length
)

// PackDepositLogData packs the deposit log.
func PackDepositLogData(
	pubkey common.BlsPubKey,
	creatorAddress common.Address,
	withdrawalAddress common.Address,
	amount *big.Int,
	signature common.BlsSignature,
	depositIndex uint64,
) []byte {
	data := make([]byte, 0, DepositLogDataLength)
	data = append(data, pubkey.Bytes()...)
	data = append(data, creatorAddress.Bytes()...)
	data = append(data, withdrawalAddress.Bytes()...)

	if amount == nil || amount.Sign() < 0 {
		amount = big.NewInt(1000000000)
	}
	amntGwei := new(big.Int).Div(amount, common.BigGwei).Uint64()
	data = append(data, common.Uint64ToBytes(amntGwei)...)

	data = append(data, signature.Bytes()...)
	data = append(data, common.Uint64ToBytes(depositIndex)...)
	return data
}

// UnpackDepositLogData unpacks the data from a deposit log using the ABI decoder.
func UnpackDepositLogData(data []byte) (
	pubkey common.BlsPubKey,
	creatorAddress common.Address,
	withdrawalAddress common.Address,
	amount uint64,
	signature common.BlsSignature,
	depositIndex uint64,
	err error,
) {
	if len(data) != DepositLogDataLength {
		err = operation.ErrBadDataLen
		return
	}
	startOffset := 0
	endOffset := startOffset + common.BlsPubKeyLength
	pubkey = common.BytesToBlsPubKey(data[startOffset:endOffset])

	startOffset = endOffset
	endOffset = startOffset + common.AddressLength
	creatorAddress = common.BytesToAddress(data[startOffset:endOffset])

	startOffset = endOffset
	endOffset = startOffset + common.AddressLength
	withdrawalAddress = common.BytesToAddress(data[startOffset:endOffset])

	startOffset = endOffset
	endOffset = startOffset + 8
	amount = common.BytesToUint64(data[startOffset:endOffset])

	startOffset = endOffset
	endOffset = startOffset + common.BlsSigLength
	signature = common.BytesToBlsSig(data[startOffset:endOffset])

	startOffset = endOffset
	endOffset = startOffset + 8
	depositIndex = common.BytesToUint64(data[startOffset:endOffset])

	return
}

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
	if len(data) != MinExitRequestLogDataLength && len(data) != MinExitRequestLogDataLength+Uint64Length {
		err = operation.ErrBadDataLen
		return
	}

	pubkey = common.BytesToBlsPubKey(data[:common.BlsPubKeyLength])
	offset := common.BlsPubKeyLength

	creatorAddress = common.BytesToAddress(data[offset : offset+common.AddressLength])
	offset += common.AddressLength

	valIndex = common.BytesToUint64(data[offset : offset+Uint64Length])
	offset += Uint64Length

	rawExitEpoch := data[offset:]
	if len(rawExitEpoch) == Uint64Length {
		exitEpoch := common.BytesToUint64(rawExitEpoch)
		exitAfterEpoch = &exitEpoch
	}
	return
}

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

	valIndex = common.BytesToUint64(data[offset : offset+Uint64Length])
	offset += Uint64Length

	amtGwei = common.BytesToUint64(data[offset : offset+Uint64Length])

	return
}
