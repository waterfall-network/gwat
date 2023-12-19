package storage

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/log"
	"gitlab.waterfall.network/waterfall/protocol/gwat/rlp"
	"gitlab.waterfall.network/waterfall/protocol/gwat/validator/operation"
)

var (
	errDelegateStakeNilValBin = errors.New("delegate stake binary data conforms to nil instance")
)

var (
	minDsdLen = minDelegateStakeDataLen() //20
)

type DelegateStakeData struct {
	Rules       operation.DelegateStakeRules // rules after trial period
	TrialPeriod uint64                       // period while trial_rules are active (in slots, starts from activation slot)
	TrialRules  operation.DelegateStakeRules // rules for trial period
}

func (dsd *DelegateStakeData) init(
	rules *operation.DelegateStakeRules,
	trialPeriod uint64,
	trialRules *operation.DelegateStakeRules,
) error {
	if rules == nil {
		rules = &operation.DelegateStakeRules{}
	} else if err := rules.Validate(); err != nil {
		return fmt.Errorf("delegate rules err: %w", err)
	}
	if trialRules == nil {
		trialRules = &operation.DelegateStakeRules{}
	}
	// while trial
	if trialPeriod > 0 && len(trialRules.ProfitShare()) > 0 {
		if err := trialRules.ValidateProfitShare(); err != nil {
			return fmt.Errorf("delegate trial rules err: %w", err)
		}
	}
	if trialPeriod > 0 && len(trialRules.StakeShare()) > 0 {
		if err := trialRules.ValidateStakeShare(); err != nil {
			return fmt.Errorf("delegate trial rules err: %w", err)
		}
	}

	dsd.Rules = *rules
	dsd.TrialPeriod = trialPeriod
	dsd.TrialRules = *trialRules
	return nil
}

// NewDelegateStakeOperation creates an operation for creating validator delegate stake
func NewDelegateStakeData(
	rules *operation.DelegateStakeRules,
	trialPeriod uint64,
	trialRules *operation.DelegateStakeRules,
) (*DelegateStakeData, error) {
	dsd := DelegateStakeData{}
	if err := dsd.init(rules, trialPeriod, trialRules); err != nil {
		return nil, err
	}
	return &dsd, nil
}

// NewDelegateStakeDataFromBinary create new instance from binary data.
// Support to init nil values.
func NewDelegateStakeDataFromBinary(bin []byte) (*DelegateStakeData, error) {
	dsd := &DelegateStakeData{}
	err := dsd.UnmarshalBinary(bin)
	// if binary data conforms to nil instance
	if errors.Is(err, errDelegateStakeNilValBin) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return dsd, nil
}

type rlpDelegateStakeOperation struct {
	R  []byte
	TP uint64
	TR []byte
}

// MarshalBinary marshals a create operation to byte encoding
func (dsd *DelegateStakeData) MarshalBinary() ([]byte, error) {
	if dsd == nil {
		return make([]byte, common.Uint32Size), nil
	}

	bRules, err := dsd.Rules.MarshalBinary()
	if err != nil {
		return nil, err
	}
	bTRules, err := dsd.TrialRules.MarshalBinary()
	if err != nil {
		return nil, err
	}
	rd := &rlpDelegateStakeOperation{
		R:  bRules,
		TP: dsd.TrialPeriod,
		TR: bTRules,
	}
	enc, err := rlp.EncodeToBytes(rd)
	if err != nil {
		return nil, err
	}
	binData := make([]byte, common.Uint32Size+len(enc))
	// set len of encoded data
	binary.BigEndian.PutUint32(binData[:common.Uint32Size], uint32(len(enc)))
	// set encoded data
	copy(binData[common.Uint32Size:], enc)
	return binData, nil
}

// UnmarshalBinary unmarshals a create operation from byte encoding
func (dsd *DelegateStakeData) UnmarshalBinary(b []byte) error {
	if len(b) < minDsdLen {
		// if binary data conforms to nil instance
		if bytes.Equal(b, make([]byte, common.Uint32Size)) {
			return errDelegateStakeNilValBin
		}
		return operation.ErrBadDataLen
	}
	dataLen := int(binary.BigEndian.Uint32(b[0:common.Uint32Size]))
	if len(b) < common.Uint32Size+dataLen {
		return operation.ErrBadDataLen
	}

	rop := &rlpDelegateStakeOperation{}
	if err := rlp.DecodeBytes(b[common.Uint32Size:common.Uint32Size+dataLen], rop); err != nil {
		return err
	}

	dsd.TrialPeriod = rop.TP

	dsd.Rules = operation.DelegateStakeRules{}
	if err := dsd.Rules.UnmarshalBinary(rop.R); err != nil {
		return err
	}
	dsd.TrialRules = operation.DelegateStakeRules{}
	if err := dsd.TrialRules.UnmarshalBinary(rop.TR); err != nil {
		return err
	}
	return nil
}

func minDelegateStakeDataLen() int {
	emptyBin, err := (&DelegateStakeData{}).MarshalBinary()
	if err != nil {
		log.Crit("Validator: calc min delegate stake binary data length failed")
	}
	return len(emptyBin)
}
