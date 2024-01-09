package txlog

import (
	"math/big"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/validator/operation"
)

const (
	DepositLogDataLength = common.BlsPubKeyLength + common.AddressLength + common.AddressLength + common.Uint64Size + common.BlsSigLength + common.Uint64Size
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
