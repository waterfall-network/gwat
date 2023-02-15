package operation

import (
	"unsafe"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
)

type depositOperation struct {
	pubkey             common.BlsPubKey // validator public key
	creator_address    common.Address   // attached creator account
	withdrawal_address common.Address   // attached withdrawal credentials
	signature          common.BlsSignature
	deposit_data_root  common.Hash
}

func (op *depositOperation) init(
	pubkey common.BlsPubKey,
	creator_address common.Address,
	withdrawal_address common.Address,
	signature common.BlsSignature,
	deposit_data_root common.Hash,
) error {
	if pubkey == (common.BlsPubKey{}) {
		return ErrNoPubKey
	}
	if creator_address == (common.Address{}) {
		return ErrNoCreatorAddress
	}
	if withdrawal_address == (common.Address{}) {
		return ErrNoWithdrawalAddress
	}
	if signature == (common.BlsSignature{}) {
		return ErrNoSignature
	}
	if deposit_data_root == (common.Hash{}) {
		return ErrNoDepositDataRoot
	}
	op.pubkey = pubkey
	op.creator_address = creator_address
	op.withdrawal_address = withdrawal_address
	op.signature = signature
	op.deposit_data_root = deposit_data_root
	return nil
}

// NewDepositOperation creates an operation for creating validator deposit
func NewDepositOperation(
	pubkey common.BlsPubKey,
	creator_address common.Address,
	withdrawal_address common.Address,
	signature common.BlsSignature,
	deposit_data_root common.Hash,
) (Deposit, error) {
	op := depositOperation{}
	if err := op.init(pubkey, creator_address, withdrawal_address, signature, deposit_data_root); err != nil {
		return nil, err
	}
	return &op, nil
}

// UnmarshalBinary unmarshals a create operation from byte encoding
func (op *depositOperation) UnmarshalBinary(b []byte) error {
	dataLen := int(unsafe.Sizeof(depositOperation{}))
	if len(b) != dataLen {
		return ErrBadDataLen
	}
	startOffset := 0
	endOffset := startOffset + common.BlsPubKeyLength
	pubKey := common.BytesToBlsPubKey(b[startOffset:endOffset])

	startOffset = endOffset
	endOffset = startOffset + common.AddressLength
	creatorAddress := common.BytesToAddress(b[startOffset:endOffset])

	startOffset = endOffset
	endOffset = startOffset + common.AddressLength
	withdrawalAddress := common.BytesToAddress(b[startOffset:endOffset])

	startOffset = endOffset
	endOffset = startOffset + common.BlsSigLength
	signature := common.BytesToBlsSig(b[startOffset:endOffset])

	startOffset = endOffset
	endOffset = startOffset + common.HashLength
	depositDataRoot := common.BytesToHash(b[startOffset:endOffset])

	return op.init(pubKey, creatorAddress, withdrawalAddress, signature, depositDataRoot)
}

// MarshalBinary marshals a create operation to byte encoding
func (op *depositOperation) MarshalBinary() ([]byte, error) {
	dataLen := int(unsafe.Sizeof(depositOperation{}))
	bin := make([]byte, 0, dataLen)
	bin = append(bin, op.pubkey.Bytes()...)
	bin = append(bin, op.creator_address.Bytes()...)
	bin = append(bin, op.withdrawal_address.Bytes()...)
	bin = append(bin, op.signature.Bytes()...)
	bin = append(bin, op.deposit_data_root.Bytes()...)
	return bin, nil
}

// Code returns op code of a deposit operation
func (op *depositOperation) OpCode() Code {
	return DepositCode
}

// Code always returns an empty address
// It's just a stub for the Operation interface.
func (op *depositOperation) Address() common.Address {
	return common.Address{}
}

func (op *depositOperation) PubKey() common.BlsPubKey {
	return common.BytesToBlsPubKey(makeCopy(op.pubkey[:]))
}

func (op *depositOperation) CreatorAddress() common.Address {
	return common.BytesToAddress(makeCopy(op.creator_address[:]))
}

func (op *depositOperation) WithdrawalAddress() common.Address {
	return common.BytesToAddress(makeCopy(op.withdrawal_address[:]))
}

func (op *depositOperation) Signature() common.BlsSignature {
	return common.BytesToBlsSig(makeCopy(op.signature[:]))
}

func (op *depositOperation) DepositDataRoot() common.Hash {
	return common.BytesToHash(makeCopy(op.deposit_data_root[:]))
}
