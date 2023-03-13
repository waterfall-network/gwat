package operation

import (
	"encoding/binary"
	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
)

const (
	exitRequestLen = common.BlsPubKeyLength + common.AddressLength + 8
)

type exitRequestOperation struct {
	pubKey           common.BlsPubKey
	validatorAddress common.Address
	exitEpoch        uint64
}

func (op *exitRequestOperation) init(
	pubKey common.BlsPubKey,
	validatorAddress common.Address,
	exitEpoch uint64,
) error {
	if pubKey == (common.BlsPubKey{}) {
		return ErrNoPubKey
	}

	if validatorAddress == (common.Address{}) {
		return ErrNoCreatorAddress
	}

	op.pubKey = pubKey
	op.validatorAddress = validatorAddress
	op.exitEpoch = exitEpoch

	return nil
}

func NewExitRequestOperation(
	pubKey common.BlsPubKey,
	validatorAddress common.Address,
	exitEpoch uint64,
) (ExitRequest, error) {
	op := &exitRequestOperation{}
	if err := op.init(pubKey, validatorAddress, exitEpoch); err != nil {
		return nil, err
	}

	return op, nil

}

func (op *exitRequestOperation) MarshalBinary() ([]byte, error) {
	var offset int
	data := make([]byte, exitRequestLen)

	copy(data[:common.BlsPubKeyLength], op.pubKey.Bytes())
	offset += common.BlsPubKeyLength

	copy(data[offset:offset+common.AddressLength], op.validatorAddress.Bytes())
	offset += common.AddressLength

	binary.BigEndian.PutUint64(data[offset:], op.exitEpoch)

	return data, nil
}

func (op *exitRequestOperation) UnmarshalBinary(data []byte) error {
	if len(data) != exitRequestLen {
		return ErrBadDataLen
	}

	var offset int
	pubKey := common.BytesToBlsPubKey(data[:common.BlsPubKeyLength])
	offset += common.BlsPubKeyLength

	validatorAddress := common.BytesToAddress(data[offset : offset+common.AddressLength])
	offset += common.AddressLength

	exitEpoch := binary.BigEndian.Uint64(data[offset:])

	return op.init(pubKey, validatorAddress, exitEpoch)
}

func (op *exitRequestOperation) OpCode() Code {
	return ActivationCode
}

func (op *exitRequestOperation) PubKey() common.BlsPubKey {
	return op.pubKey
}

func (op *exitRequestOperation) ValidatorAddress() common.Address {
	return op.validatorAddress
}

func (op *exitRequestOperation) ExitEpoch() uint64 {
	return op.exitEpoch
}
