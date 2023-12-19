package operation

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"sort"

	"github.com/prysmaticlabs/go-bitfield"
	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/rlp"
)

type delegateStakeOperation struct {
	pubKey         common.BlsPubKey    // validator public key
	creatorAddress common.Address      // attached creator account
	signature      common.BlsSignature // sig
	rules          DelegateStakeRules  // rules after trial period
	trialPeriod    uint64              // period while trial_rules are active (in slots, starts from activation slot)
	trialRules     DelegateStakeRules  // rules for trial period
}

func (op *delegateStakeOperation) init(
	pubKey common.BlsPubKey,
	creatorAddress common.Address,
	signature common.BlsSignature,
	rules *DelegateStakeRules,
	trialPeriod uint64,
	trialRules *DelegateStakeRules,
) error {
	if pubKey == (common.BlsPubKey{}) {
		return ErrNoPubKey
	}
	if creatorAddress == (common.Address{}) {
		return ErrNoCreatorAddress
	}
	if signature == (common.BlsSignature{}) {
		return ErrNoSignature
	}

	if rules == nil {
		rules = &DelegateStakeRules{}
	}
	if err := rules.Validate(); err != nil {
		return fmt.Errorf("delegate rules err: %w", err)
	}

	if trialRules == nil {
		trialRules = &DelegateStakeRules{}
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

	op.pubKey = pubKey
	op.creatorAddress = creatorAddress
	op.signature = signature
	op.rules = *rules
	op.trialPeriod = trialPeriod
	op.trialRules = *trialRules
	return nil
}

// NewDelegateStakeOperation creates an operation for creating validator delegate stake
func NewDelegateStakeOperation(
	pubkey common.BlsPubKey,
	creator_address common.Address,
	signature common.BlsSignature,
	rules *DelegateStakeRules,
	trialPeriod uint64,
	trialRules *DelegateStakeRules,
) (DelegateStake, error) {
	op := delegateStakeOperation{}
	if err := op.init(pubkey, creator_address, signature, rules, trialPeriod, trialRules); err != nil {
		return nil, err
	}
	return &op, nil
}

type rlpDelegateStakeOperation struct {
	PK common.BlsPubKey
	CA common.Address
	S  common.BlsSignature
	R  []byte
	TP uint64
	TR []byte
}

// MarshalBinary marshals a create operation to byte encoding
func (op *delegateStakeOperation) MarshalBinary() ([]byte, error) {
	bRules, err := op.Rules().MarshalBinary()
	if err != nil {
		return nil, err
	}
	bTRules, err := op.TrialRules().MarshalBinary()
	if err != nil {
		return nil, err
	}
	rd := &rlpDelegateStakeOperation{
		PK: op.pubKey,
		CA: op.creatorAddress,
		S:  op.signature,
		R:  bRules,
		TP: op.trialPeriod,
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
func (op *delegateStakeOperation) UnmarshalBinary(b []byte) error {
	if len(b) < common.Uint32Size {
		return ErrBadDataLen
	}
	dataLen := int(binary.BigEndian.Uint32(b[0:common.Uint32Size]))
	if len(b) < common.Uint32Size+dataLen {
		return ErrBadDataLen
	}

	rop := &rlpDelegateStakeOperation{}
	if err := rlp.DecodeBytes(b[common.Uint32Size:common.Uint32Size+dataLen], rop); err != nil {
		return err
	}

	op.pubKey = rop.PK
	op.creatorAddress = rop.CA
	op.signature = rop.S
	op.trialPeriod = rop.TP

	op.rules = DelegateStakeRules{}
	if err := op.rules.UnmarshalBinary(rop.R); err != nil {
		return err
	}
	op.trialRules = DelegateStakeRules{}
	if err := op.trialRules.UnmarshalBinary(rop.TR); err != nil {
		return err
	}
	return nil
}

// Code returns op code of a delegate stake operation
func (op *delegateStakeOperation) OpCode() Code {
	return DelegateStakeCode
}

// Code always returns an empty address
// It's just a stub for the Operation interface.
func (op *delegateStakeOperation) Address() common.Address {
	return common.Address{}
}

func (op *delegateStakeOperation) PubKey() common.BlsPubKey {
	return common.BytesToBlsPubKey(makeCopy(op.pubKey[:]))
}

func (op *delegateStakeOperation) CreatorAddress() common.Address {
	return common.BytesToAddress(makeCopy(op.creatorAddress[:]))
}

func (op *delegateStakeOperation) Signature() common.BlsSignature {
	return common.BytesToBlsSig(makeCopy(op.signature[:]))
}

func (op *delegateStakeOperation) Rules() *DelegateStakeRules {
	return op.rules.Copy()
}

func (op *delegateStakeOperation) TrialPeriod() uint64 {
	return op.trialPeriod
}

func (op *delegateStakeOperation) TrialRules() *DelegateStakeRules {
	return op.trialRules.Copy()
}

/** DelegateStakeRules
 */
type DelegateStakeRules struct {
	addrs       []common.Address
	profitShare []uint8
	stakeShare  []uint8
	exit        bitfield.Bitlist
	withdrawal  bitfield.Bitlist
}

func NewDelegateStakeRules(
	profitShare map[common.Address]uint8,
	stakeShare map[common.Address]uint8,
	exit []common.Address,
	withdrawal []common.Address,
) (*DelegateStakeRules, error) {
	dsr := DelegateStakeRules{}
	if err := dsr.init(profitShare, stakeShare, exit, withdrawal); err != nil {
		return nil, err
	}
	return &dsr, nil
}

func (dr *DelegateStakeRules) init(
	profitShare map[common.Address]uint8,
	stakeShare map[common.Address]uint8,
	exit []common.Address,
	withdrawal []common.Address,
) error {
	type adrInfo struct {
		Profit   uint8
		Stake    uint8
		Exit     bool
		Withdraw bool
	}
	var (
		addrMap = make(map[common.Address]*adrInfo)
		addrs   []common.Address
	)
	// collect data
	for a, v := range profitShare {
		if v == 0 {
			continue
		}
		if _, ok := addrMap[a]; !ok {
			addrMap[a] = new(adrInfo)
		}
		addrMap[a].Profit = v
	}
	for a, v := range stakeShare {
		if v == 0 {
			continue
		}
		if _, ok := addrMap[a]; !ok {
			addrMap[a] = new(adrInfo)
		}
		addrMap[a].Stake = v
	}
	for _, a := range exit {
		if _, ok := addrMap[a]; !ok {
			addrMap[a] = new(adrInfo)
		}
		addrMap[a].Exit = true
	}
	for _, a := range withdrawal {
		if _, ok := addrMap[a]; !ok {
			addrMap[a] = new(adrInfo)
		}
		addrMap[a].Withdraw = true
	}
	// set data
	addrs = make([]common.Address, len(addrMap))
	addrCount := 0
	for adr := range addrMap {
		addrs[addrCount] = adr
		addrCount++
	}
	sort.Slice(addrs, func(i, j int) bool { return bytes.Compare(addrs[i][:], addrs[j][:]) < 0 })
	dr.addrs = addrs

	dr.profitShare = make([]uint8, len(addrs))
	dr.stakeShare = make([]uint8, len(addrs))
	dr.exit = bitfield.NewBitlist(uint64(len(addrs)))
	dr.withdrawal = bitfield.NewBitlist(uint64(len(addrs)))
	for i, adr := range addrs {
		info := addrMap[adr]
		if info.Profit > 0 {
			dr.profitShare[i] = info.Profit
		}
		if info.Stake > 0 {
			dr.stakeShare[i] = info.Stake
		}
		if info.Exit {
			dr.exit.SetBitAt(uint64(i), true)
		}
		if info.Withdraw {
			dr.withdrawal.SetBitAt(uint64(i), true)
		}
	}
	return nil
}

func (dr *DelegateStakeRules) Validate() error {
	if err := dr.ValidateProfitShare(); err != nil {
		return err
	}
	if err := dr.ValidateStakeShare(); err != nil {
		return err
	}
	if err := dr.validateExit(); err != nil {
		return err
	}
	if err := dr.validateWithdrawal(); err != nil {
		return err
	}
	return nil
}

func (dr *DelegateStakeRules) ValidateProfitShare() error {
	//check the percentage values are correct
	var totalPrc uint
	for _, v := range dr.profitShare {
		totalPrc += uint(v)
	}
	if totalPrc != 100 {
		return ErrBadProfitShare
	}
	return nil
}

func (dr *DelegateStakeRules) ValidateStakeShare() error {
	var totalPrc uint
	for _, v := range dr.stakeShare {
		totalPrc += uint(v)
	}
	if totalPrc != 100 {
		return ErrBadStakeShare
	}
	return nil
}

func (dr *DelegateStakeRules) validateExit() error {
	if len(dr.Exit()) == 0 {
		return ErrNoExitRoles
	}
	return nil
}

func (dr *DelegateStakeRules) validateWithdrawal() error {
	if len(dr.Withdrawal()) == 0 {
		return ErrNoWithdrawalRoles
	}
	return nil
}

func (dr *DelegateStakeRules) Copy() *DelegateStakeRules {
	if dr == nil {
		return nil
	}
	var (
		adrCount    = len(dr.addrs)
		addrs       = make([]common.Address, adrCount)
		profitShare = make([]uint8, adrCount)
		stakeShare  = make([]uint8, adrCount)
		exit        = bitfield.NewBitlist(uint64(adrCount))
		withdrawal  = bitfield.NewBitlist(uint64(adrCount))
	)

	for i, adr := range dr.addrs {
		copy(addrs[i][:], adr[:])
		profitShare[i] = dr.profitShare[i]
		stakeShare[i] = dr.stakeShare[i]
	}
	copy(exit[:], dr.exit[:])
	copy(withdrawal[:], dr.withdrawal[:])

	return &DelegateStakeRules{
		addrs:       addrs,
		profitShare: profitShare,
		stakeShare:  stakeShare,
		exit:        exit,
		withdrawal:  withdrawal,
	}
}

// ProfitShare returns map of participants profit share in %
func (dr *DelegateStakeRules) ProfitShare() map[common.Address]uint8 {
	data := map[common.Address]uint8{}
	for i, a := range dr.addrs {
		if v := dr.profitShare[i]; v > 0 {
			data[a] = v
		}
	}
	return data
}

// StakeShare returns map of participants stake share in % (after exit)
func (dr *DelegateStakeRules) StakeShare() map[common.Address]uint8 {
	data := map[common.Address]uint8{}
	for i, a := range dr.addrs {
		if v := dr.stakeShare[i]; v > 0 {
			data[a] = v
		}
	}
	return data
}

// Exit returns the addresses authorized to init exit procedure.
func (dr *DelegateStakeRules) Exit() []common.Address {
	ixs := dr.exit.BitIndices()
	data := make([]common.Address, len(ixs))
	for i, ix := range ixs {
		data[i] = dr.addrs[ix]
	}
	return data
}

// Withdrawal returns the addresses authorized to init exit procedure.
func (dr *DelegateStakeRules) Withdrawal() []common.Address {
	ixs := dr.withdrawal.BitIndices()
	data := make([]common.Address, len(ixs))
	for i, ix := range ixs {
		data[i] = dr.addrs[ix]
	}
	return data
}

type rlpDelegateStakeRules struct {
	A []common.Address
	P []uint8
	S []uint8
	E []byte
	W []byte
}

func (dr *DelegateStakeRules) MarshalBinary() ([]byte, error) {
	rd := &rlpDelegateStakeRules{
		A: dr.addrs,
		P: dr.profitShare,
		S: dr.stakeShare,
		E: dr.exit,
		W: dr.withdrawal,
	}
	return rlp.EncodeToBytes(rd)
}

func (dr *DelegateStakeRules) UnmarshalBinary(b []byte) error {
	rd := &rlpDelegateStakeRules{}
	if err := rlp.DecodeBytes(b, rd); err != nil {
		return err
	}
	dr.addrs = rd.A
	dr.profitShare = rd.P
	dr.stakeShare = rd.S
	dr.exit = rd.E
	dr.withdrawal = rd.W
	return nil
}
