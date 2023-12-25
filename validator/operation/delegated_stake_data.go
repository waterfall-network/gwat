package operation

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/log"
	"gitlab.waterfall.network/waterfall/protocol/gwat/rlp"
)

var (
	errDelegatingStakeNilValBin = errors.New("delegate stake binary data conforms to nil instance")
)

var (
	minDsdLen = minDelegatingStakeDataLen() //20
)

type DelegatingStakeData struct {
	Rules       DelegatingStakeRules `json:"rules"`        // rules after trial period
	TrialPeriod uint64               `json:"trial_period"` // period while trial_rules are active (in slots, starts from activation slot)
	TrialRules  DelegatingStakeRules `json:"trial_rules"`  // rules for trial period
}

func (dsd *DelegatingStakeData) init(
	rules *DelegatingStakeRules,
	trialPeriod uint64,
	trialRules *DelegatingStakeRules,
) error {
	if rules == nil {
		rules = &DelegatingStakeRules{}
	} else if err := rules.Validate(); err != nil {
		return fmt.Errorf("delegate rules err: %w", err)
	}

	if trialRules == nil {
		trialRules = &DelegatingStakeRules{}
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

// NewDelegatingStakeOperation creates an operation for creating validator delegate stake
func NewDelegatingStakeData(
	rules *DelegatingStakeRules,
	trialPeriod uint64,
	trialRules *DelegatingStakeRules,
) (*DelegatingStakeData, error) {
	dsd := DelegatingStakeData{}
	if err := dsd.init(rules, trialPeriod, trialRules); err != nil {
		return nil, err
	}
	return &dsd, nil
}

// NewDelegatingStakeDataFromBinary create new instance from binary data.
// Support to init nil values.
func NewDelegatingStakeDataFromBinary(bin []byte) (*DelegatingStakeData, error) {
	dsd := &DelegatingStakeData{}
	err := dsd.UnmarshalBinary(bin)
	// if binary data conforms to nil instance
	if errors.Is(err, errDelegatingStakeNilValBin) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return dsd, nil
}

type rlpDelegatingStakeOperation struct {
	R  []byte
	TP uint64
	TR []byte
}

// MarshalBinary marshals a create operation to byte encoding
func (dsd *DelegatingStakeData) MarshalBinary() ([]byte, error) {
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
	rd := &rlpDelegatingStakeOperation{
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
func (dsd *DelegatingStakeData) UnmarshalBinary(b []byte) error {
	if len(b) < minDsdLen {
		// if binary data conforms to nil instance
		if bytes.Equal(b, make([]byte, common.Uint32Size)) {
			return errDelegatingStakeNilValBin
		}
		return ErrBadDataLen
	}
	dataLen := int(binary.BigEndian.Uint32(b[0:common.Uint32Size]))
	if len(b) < common.Uint32Size+dataLen {
		return ErrBadDataLen
	}

	rop := &rlpDelegatingStakeOperation{}
	if err := rlp.DecodeBytes(b[common.Uint32Size:common.Uint32Size+dataLen], rop); err != nil {
		return err
	}

	dsd.TrialPeriod = rop.TP

	dsd.Rules = DelegatingStakeRules{}
	if err := dsd.Rules.UnmarshalBinary(rop.R); err != nil {
		return err
	}
	dsd.TrialRules = DelegatingStakeRules{}
	if err := dsd.TrialRules.UnmarshalBinary(rop.TR); err != nil {
		return err
	}
	return nil
}

func (dsd *DelegatingStakeData) Copy() *DelegatingStakeData {
	if dsd == nil {
		return nil
	}
	rules := dsd.Rules.Copy()
	tRules := dsd.TrialRules.Copy()
	return &DelegatingStakeData{
		Rules:       *rules,
		TrialPeriod: dsd.TrialPeriod,
		TrialRules:  *tRules,
	}
}

func minDelegatingStakeDataLen() int {
	emptyBin, err := (&DelegatingStakeData{}).MarshalBinary()
	if err != nil {
		log.Crit("Validator: calc min delegate stake binary data length failed")
	}
	return len(emptyBin)
}
