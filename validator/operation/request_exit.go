package operation

import (
	"encoding/binary"
	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
)

const (
	minExitRequestLen = common.BlsPubKeyLength + common.AddressLength
)

type exitRequestOperation struct {
	pubKey         common.BlsPubKey
	creatorAddress common.Address
	exitAfterEpoch *uint64
}

func (op *exitRequestOperation) init(
	pubKey common.BlsPubKey,
	creatorAddress common.Address,
	exitAfterEpoch *uint64,
) error {
	if pubKey == (common.BlsPubKey{}) {
		return ErrNoPubKey
	}

	if creatorAddress == (common.Address{}) {
		return ErrNoCreatorAddress
	}

	op.pubKey = pubKey
	op.creatorAddress = creatorAddress
	op.exitAfterEpoch = exitAfterEpoch

	return nil
}

func NewExitRequestOperation(
	pubKey common.BlsPubKey,
	creatorAddress common.Address,
	exitAfterEpoch *uint64,
) (ExitRequest, error) {
	op := &exitRequestOperation{}
	if err := op.init(pubKey, creatorAddress, exitAfterEpoch); err != nil {
		return nil, err
	}

	return op, nil

}

func (op *exitRequestOperation) MarshalBinary() ([]byte, error) {
	var offset int
	dataLen := minExitRequestLen
	if op.exitAfterEpoch != nil {
		dataLen += 8
	}

	data := make([]byte, dataLen)

	copy(data[:common.BlsPubKeyLength], op.pubKey.Bytes())
	offset += common.BlsPubKeyLength

	copy(data[offset:offset+common.AddressLength], op.creatorAddress.Bytes())
	offset += common.AddressLength

	if op.exitAfterEpoch != nil {
		binary.BigEndian.PutUint64(data[offset:], *op.exitAfterEpoch)
	}

	return data, nil
}

func (op *exitRequestOperation) UnmarshalBinary(data []byte) error {
	if len(data) < minExitRequestLen {
		return ErrBadDataLen
	}

	var offset int
	pubKey := common.BytesToBlsPubKey(data[:common.BlsPubKeyLength])
	offset += common.BlsPubKeyLength

	creatorAddress := common.BytesToAddress(data[offset : offset+common.AddressLength])
	offset += common.AddressLength

	if len(data) > minExitRequestLen {
		exitAfterEpoch := binary.BigEndian.Uint64(data[offset:])
		return op.init(pubKey, creatorAddress, &exitAfterEpoch)
	} else {
		return op.init(pubKey, creatorAddress, nil)
	}
}

func (op *exitRequestOperation) OpCode() Code {
	return ExitCode
}

func (op *exitRequestOperation) PubKey() common.BlsPubKey {
	return op.pubKey
}

func (op *exitRequestOperation) CreatorAddress() common.Address {
	return op.creatorAddress
}

func (op *exitRequestOperation) ExitAfterEpoch() *uint64 {
	return op.exitAfterEpoch
}
