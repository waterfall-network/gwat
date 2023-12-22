package operation

import (
	"encoding/binary"
	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
)

type depositOperation struct {
	pubkey             common.BlsPubKey // validator public key
	creator_address    common.Address   // attached creator account
	withdrawal_address common.Address   // attached withdrawal credentials
	signature          common.BlsSignature
	delegate           *DelegatedStakeData
}

func (op *depositOperation) init(
	pubkey common.BlsPubKey,
	creator_address common.Address,
	withdrawal_address common.Address,
	signature common.BlsSignature,
	delegate *DelegatedStakeData,
) error {
	if pubkey == (common.BlsPubKey{}) {
		return ErrNoPubKey
	}
	if creator_address == (common.Address{}) {
		return ErrNoCreatorAddress
	}
	if withdrawal_address == (common.Address{}) && delegate == nil {
		return ErrNoWithdrawalAddress
	}
	if signature == (common.BlsSignature{}) {
		return ErrNoSignature
	}
	op.pubkey = pubkey
	op.creator_address = creator_address
	op.withdrawal_address = withdrawal_address
	op.signature = signature

	// validate delegate stake data
	if delegate != nil {
		if err := delegate.Rules.Validate(); err != nil {
			return err
		}
		// while trial
		if delegate.TrialPeriod > 0 && len(delegate.Rules.ProfitShare()) > 0 {
			if err := delegate.Rules.ValidateProfitShare(); err != nil {
				return err
			}
		}
		if delegate.TrialPeriod > 0 && len(delegate.Rules.StakeShare()) > 0 {
			if err := delegate.Rules.ValidateStakeShare(); err != nil {
				return err
			}
		}
	}
	op.delegate = delegate
	return nil
}

// NewDepositOperation creates an operation for creating validator deposit
func NewDepositOperation(
	pubkey common.BlsPubKey,
	creator_address common.Address,
	withdrawal_address common.Address,
	signature common.BlsSignature,
	delegate *DelegatedStakeData,
) (Deposit, error) {
	op := depositOperation{}
	if err := op.init(pubkey, creator_address, withdrawal_address, signature, delegate); err != nil {
		return nil, err
	}
	return &op, nil
}

// UnmarshalBinary unmarshals a create operation from byte encoding
func (op *depositOperation) UnmarshalBinary(b []byte) error {
	var err error
	baseDataLen := op.minBinaryLen()
	if len(b) < baseDataLen {
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

	// retrieve extended data
	var delegateStake *DelegatedStakeData
	extendedData := b[endOffset:]
	if len(extendedData) > 0 {
		// delegate stake data
		// retrieve data len
		startOffset = 0
		endOfset := startOffset + common.Uint32Size
		delegateDataLen := int(binary.BigEndian.Uint32(extendedData[startOffset:endOfset]))
		if delegateDataLen > len(extendedData[endOfset:]) {
			return ErrBadDataLen
		}
		// get delegate data
		startOffset = endOfset
		endOfset = startOffset + delegateDataLen
		delegateStake, err = NewDelegateStakeDataFromBinary(extendedData[startOffset:endOfset])
		if err != nil {
			return err
		}
	}

	return op.init(pubKey, creatorAddress, withdrawalAddress, signature, delegateStake)
}

// MarshalBinary marshals a create operation to byte encoding
func (op *depositOperation) MarshalBinary() ([]byte, error) {
	// marshal binary extended data
	// delegate stake data
	delegateBin, err := op.delegate.MarshalBinary()
	if err != nil {
		return nil, err
	}

	dataLen := op.minBinaryLen() + len(delegateBin)
	bin := make([]byte, 0, dataLen)
	bin = append(bin, op.pubkey.Bytes()...)
	bin = append(bin, op.creator_address.Bytes()...)
	bin = append(bin, op.withdrawal_address.Bytes()...)
	bin = append(bin, op.signature.Bytes()...)

	if op.delegate == nil {
		return bin, nil
	}

	// set extended data
	//set len of delegate stake data
	dlgBinLen := make([]byte, common.Uint32Size)
	binary.BigEndian.PutUint32(dlgBinLen, uint32(len(delegateBin)))
	bin = append(bin, dlgBinLen...)
	// delegate stake data
	bin = append(bin, delegateBin...)

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

func (op *depositOperation) DelegatedStake() *DelegatedStakeData {
	return op.delegate.Copy()
}

func (op *depositOperation) minBinaryLen() int {
	return common.BlsPubKeyLength + common.AddressLength + common.AddressLength + common.BlsSigLength
}
