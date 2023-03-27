package validator

import (
	"errors"
	"fmt"
	"math"
	"math/big"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/rawdb"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/types"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/vm"
	"gitlab.waterfall.network/waterfall/protocol/gwat/crypto"
	"gitlab.waterfall.network/waterfall/protocol/gwat/log"
	"gitlab.waterfall.network/waterfall/protocol/gwat/validator/era"
	"gitlab.waterfall.network/waterfall/protocol/gwat/validator/operation"
	valStore "gitlab.waterfall.network/waterfall/protocol/gwat/validator/storage"
)

var (
	// errors
	ErrTooLowDepositValue = errors.New("deposit value is too low")
	// ErrInsufficientFundsForTransfer is returned if the transaction sender doesn't
	// have enough funds for transfer(topmost call only).
	ErrInsufficientFundsForTransfer = errors.New("insufficient funds for transfer")
	ErrInvalidFromAddresses         = errors.New("withdrawal and sender addresses are mismatch")
	ErrUnknownValidator             = errors.New("unknown validator")
	ErrNotActivatedValidator        = errors.New("validator not activated yet")
	ErrValidatorIsOut               = errors.New("validator is out")
	ErrInvalidToAddress             = errors.New("address to must be validators state address")
	ErrNoExitRequest                = errors.New("exit request is require before withdrawal operation")
	ErrCtxEraNotFound               = errors.New("context block era not found")
	ErrTargetEraNotFound            = errors.New("target era not found")
)

const (
	// 1024 bytes
	MetadataMaxSize = 1 << 10

	// Common fields
	pubkeyLogType    = "pubkey"
	signatureLogType = "signature"
	addressLogType   = "address"
	uint256LogType   = "uint256"
	boolLogType      = "bool"
)

var (
	// minimal value - 1000 wat
	MinDepositVal, _ = new(big.Int).SetString("1000000000000000000000", 10)

	//Events signatures.
	EvtDepositLogSignature = crypto.Keccak256Hash([]byte("DepositLog"))
	EvtExitReqLogSignature = crypto.Keccak256Hash([]byte("ExitRequestLog"))
)

// Ref represents caller of the validator processor
type Ref interface {
	Address() common.Address
}

// Processor is a processor of all validator related operations.
// All transaction related operations that mutates state of the validator are called using Call method.
// Methods of the operation name are used for getting state of the validator.
type Processor struct {
	state        vm.StateDB
	ctx          vm.BlockContext
	eventEmmiter *EventEmmiter
	storage      valStore.Storage
	blockchain   era.Blockchain
}

// NewProcessor creates new validator processor
func NewProcessor(blockCtx vm.BlockContext, stateDb vm.StateDB, bc era.Blockchain) *Processor {
	return &Processor{
		ctx:          blockCtx,
		state:        stateDb,
		eventEmmiter: NewEventEmmiter(stateDb),
		storage:      valStore.NewStorage(bc.GetConfig()),
		blockchain:   bc,
	}
}

func (p *Processor) getDepositCount() uint64 {
	return p.Storage().GetDepositCount(p.state)
}

func (p *Processor) incrDepositCount() {
	p.Storage().IncrementDepositCount(p.state)
}

func (p *Processor) GetValidatorsStateAddress() common.Address {
	valAddress := p.Storage().GetValidatorsStateAddress()
	if valAddress == nil {
		return common.Address{}
	}

	return *valAddress
}

func (p *Processor) Storage() valStore.Storage {
	return p.storage
}

// IsValidatorOp returns true if tx is validator operation
func (p *Processor) IsValidatorOp(addrTo *common.Address) bool {
	if addrTo == nil {
		return false
	}

	return *addrTo == p.GetValidatorsStateAddress()
}

// Call performs all transaction related operations that mutates state of the validator and validators state
//
// The only following operations can be performed using the method:
//   - validator: Deposit
//   - coordinating node: Activate
//   - validator: RequestExit
//   - coordinating node: Deactivate
//
// It returns byte representation of the return value of an operation.
func (p *Processor) Call(caller Ref, toAddr common.Address, value *big.Int, op operation.Operation) (ret []byte, err error) {
	nonce := p.state.GetNonce(caller.Address())
	p.state.SetNonce(caller.Address(), nonce+1)

	snapshot := p.state.Snapshot()

	ret = nil
	switch v := op.(type) {
	case operation.Deposit:
		ret, err = p.validatorDeposit(caller, toAddr, value, v)
	case operation.ValidatorSync:
		ret, err = p.syncOpProcessing(v)
	case operation.Exit:
		ret, err = p.validatorExit(caller, toAddr, v)
	case operation.Withdrawal:
		ret, err = p.validatorWithdrawal(caller, toAddr, v)
	}

	if err != nil {
		p.state.RevertToSnapshot(snapshot)
	}

	return ret, err
}

func (p *Processor) validatorDeposit(caller Ref, toAddr common.Address, value *big.Int, op operation.Deposit) (_ []byte, err error) {
	if !p.IsValidatorOp(&toAddr) {
		return nil, ErrInvalidToAddress
	}

	from := caller.Address()

	if value == nil || value.Cmp(MinDepositVal) < 0 {
		return nil, ErrTooLowDepositValue
	}
	balanceFrom := p.state.GetBalance(from)
	if balanceFrom.Cmp(value) < 0 {
		return nil, fmt.Errorf("%w: address %v", ErrInsufficientFundsForTransfer, from.Hex())
	}

	withdrawalAddress := op.WithdrawalAddress()

	validator := valStore.NewValidator(op.CreatorAddress(), &withdrawalAddress)

	valInfo, err := validator.MarshalBinary()
	if err != nil {
		return nil, err
	}

	p.Storage().SetValidatorInfo(p.state, valInfo)

	logData := PackDepositLogData(op.PubKey(), op.CreatorAddress(), op.WithdrawalAddress(), value, op.Signature(), p.getDepositCount())
	p.eventEmmiter.Deposit(toAddr, logData)
	p.incrDepositCount()
	// burn value from sender balance
	p.state.SubBalance(from, value)

	log.Info("Deposit", "address", toAddr.Hex(), "from", from.Hex(), "value", value.String(), "pabkey", op.PubKey().Hex(), "creator", op.CreatorAddress().Hex())
	return value.FillBytes(make([]byte, 32)), nil
}

func (p *Processor) validatorExit(caller Ref, toAddr common.Address, op operation.Exit) ([]byte, error) {
	if !p.IsValidatorOp(&toAddr) {
		return nil, ErrInvalidToAddress
	}

	from := caller.Address()
	validator := p.Storage().GetValidatorInfo(p.state, op.CreatorAddress())
	if validator == nil {
		return nil, ErrUnknownValidator
	}

	if op.ExitAfterEpoch() == nil {
		currentEpoch := p.blockchain.GetSlotInfo().SlotToEpoch(p.blockchain.GetSlotInfo().CurrentSlot())
		op.SetExitAfterEpoch(&currentEpoch)
	}

	if validator.GetActivationEpoch() > *op.ExitAfterEpoch() {
		return nil, ErrNotActivatedValidator
	}

	if validator.GetExitEpoch() != math.MaxUint64 {
		return nil, ErrValidatorIsOut
	}

	if from != validator.GetWithdrawalAddress() {
		return nil, ErrInvalidFromAddresses
	}

	logData := PackExitRequestLogData(op.PubKey(), op.CreatorAddress(), validator.GetValidatorIndex(), op.ExitAfterEpoch())
	p.eventEmmiter.ExitRequest(toAddr, logData)

	return op.CreatorAddress().Bytes(), nil
}

func (p *Processor) validatorWithdrawal(caller Ref, toAddr common.Address, op operation.Withdrawal) ([]byte, error) {
	if !p.IsValidatorOp(&toAddr) {
		return nil, ErrInvalidToAddress
	}

	from := caller.Address()
	valInfo := p.Storage().GetValidatorInfo(p.state, op.CreatorAddress())

	if from != valInfo.GetWithdrawalAddress() {
		return nil, ErrInvalidFromAddresses
	}

	if valInfo.GetActivationEpoch() == math.MaxUint64 {
		return nil, ErrNotActivatedValidator
	}

	currentBalance := valInfo.GetValidatorBalance()

	if currentBalance.Cmp(op.Amount()) == -1 {
		return nil, ErrInsufficientFundsForTransfer
	}

	valInfo.SetValidatorBalance(new(big.Int).Sub(currentBalance, op.Amount()))
	p.Storage().SetValidatorInfo(p.state, valInfo)

	p.state.AddBalance(from, op.Amount())

	return from.Bytes(), nil
}

func (p *Processor) syncOpProcessing(op operation.ValidatorSync) (ret []byte, err error) {
	switch op.OpCode() {
	case operation.ActivateCode:
		ret, err = p.validatorActivate(op)
	case operation.DeactivateCode:
		ret, err = p.validatorDeactivate(op)
	case operation.UpdateBalanceCode:
		ret, err = p.validatorUpdateBalance(op)
	}

	return ret, err
}

func (p *Processor) validatorActivate(op operation.ValidatorSync) ([]byte, error) {
	valInfo := p.Storage().GetValidatorInfo(p.state, op.Creator())
	if valInfo == nil {
		log.Error("Validator sync: err", "era", p.ctx.Era, "opCode", op.OpCode(), "creator", op.Creator().Hex(), "procEpoch", op.ProcEpoch(), "err", ErrUnknownValidator)
		return nil, ErrUnknownValidator
	}

	//retrieve block eraInfo
	eraInfo := p.blockchain.GetEraInfo()
	var blkEraInfo *era.EraInfo
	if eraInfo.Number() == p.ctx.Era {
		blkEraInfo = eraInfo
	} else {
		blkEra := rawdb.ReadEra(p.blockchain.Database(), p.ctx.Era)
		if blkEra == nil {
			log.Error("Validator sync: block era not found", "era", p.ctx.Era, "opCode", op.OpCode(), "creator", op.Creator().Hex(), "procEpoch", op.ProcEpoch())
			return nil, ErrCtxEraNotFound
		}
		bei := era.NewEraInfo(*blkEra)
		blkEraInfo = &bei
	}

	// calculate activation epoch
	var targetEpoch uint64
	blkEpoch := p.blockchain.GetSlotInfo().SlotToEpoch(p.ctx.Slot)
	if blkEraInfo.IsTransitionPeriodEpoch(p.blockchain, blkEpoch) {
		var nextEraInfo *era.EraInfo
		if eraInfo.Number() == p.ctx.Era+1 {
			nextEraInfo = eraInfo
		} else {
			blkEra := rawdb.ReadEra(p.blockchain.Database(), p.ctx.Era+1)
			if blkEra == nil {
				log.Error("Validator sync: target era not found", "era", p.ctx.Era, "opCode", op.OpCode(), "creator", op.Creator().Hex(), "procEpoch", op.ProcEpoch())
				return nil, ErrTargetEraNotFound
			}
			bei := era.NewEraInfo(*blkEra)
			nextEraInfo = &bei
		}
		targetEpoch = nextEraInfo.NextEraFirstEpoch()
	} else {
		targetEpoch = blkEraInfo.NextEraFirstEpoch()
	}

	valInfo.SetActivationEpoch(targetEpoch)
	valInfo.SetValidatorIndex(op.Index())
	p.Storage().SetValidatorInfo(p.state, valInfo)

	p.Storage().AddValidatorToList(p.state, op.Index(), op.Creator())

	log.Info("Validator sync: activate", "opCode", op.OpCode(), "creator", op.Creator().Hex(), "procEpoch", op.ProcEpoch(), "targetEpoch", targetEpoch)

	return op.Creator().Bytes(), nil
}

func (p *Processor) validatorDeactivate(op operation.ValidatorSync) ([]byte, error) {
	valInfo := p.Storage().GetValidatorInfo(p.state, op.Creator())
	if valInfo == nil {
		log.Error("Validator sync: err", "era", p.ctx.Era, "opCode", op.OpCode(), "creator", op.Creator().Hex(), "procEpoch", op.ProcEpoch(), "err", ErrUnknownValidator)
		return nil, ErrUnknownValidator
	}

	if valInfo.GetActivationEpoch() > op.ProcEpoch() {
		log.Error("Validator sync: err", "era", p.ctx.Era, "opCode", op.OpCode(), "creator", op.Creator().Hex(), "procEpoch", op.ProcEpoch(), "err", ErrNotActivatedValidator)
		return nil, ErrNotActivatedValidator
	}
	if valInfo.GetExitEpoch() != math.MaxUint64 {
		log.Error("Validator sync: err", "era", p.ctx.Era, "opCode", op.OpCode(), "creator", op.Creator().Hex(), "procEpoch", op.ProcEpoch(), "err", ErrValidatorIsOut)
		return nil, ErrValidatorIsOut
	}

	//retrieve block eraInfo
	eraInfo := p.blockchain.GetEraInfo()
	var blkEraInfo *era.EraInfo
	if eraInfo.Number() == p.ctx.Era {
		blkEraInfo = eraInfo
	} else {
		blkEra := rawdb.ReadEra(p.blockchain.Database(), p.ctx.Era)
		if blkEra == nil {
			log.Error("Validator sync: block era not found", "era", p.ctx.Era, "opCode", op.OpCode(), "creator", op.Creator().Hex(), "procEpoch", op.ProcEpoch())
			return nil, ErrCtxEraNotFound
		}
		bei := era.NewEraInfo(*blkEra)
		blkEraInfo = &bei
	}

	// calculate deactivation epoch
	var targetEpoch uint64
	blkEpoch := p.blockchain.GetSlotInfo().SlotToEpoch(p.ctx.Slot)
	if blkEraInfo.IsTransitionPeriodEpoch(p.blockchain, blkEpoch) {
		var nextEraInfo *era.EraInfo
		if eraInfo.Number() == p.ctx.Era+1 {
			nextEraInfo = eraInfo
		} else {
			blkEra := rawdb.ReadEra(p.blockchain.Database(), p.ctx.Era+1)
			if blkEra == nil {
				log.Error("Validator sync: target era not found", "era", p.ctx.Era, "opCode", op.OpCode(), "creator", op.Creator().Hex(), "procEpoch", op.ProcEpoch())
				return nil, ErrTargetEraNotFound
			}
			bei := era.NewEraInfo(*blkEra)
			nextEraInfo = &bei
		}
		targetEpoch = nextEraInfo.NextEraFirstEpoch()
	} else {
		targetEpoch = blkEraInfo.NextEraFirstEpoch()
	}

	valInfo.SetExitEpoch(targetEpoch)
	p.Storage().SetValidatorInfo(p.state, valInfo)

	log.Info("Validator sync: deactivate", "opCode", op.OpCode(), "creator", op.Creator().Hex(), "procEpoch", op.ProcEpoch(), "targetEpoch", targetEpoch)

	return op.Creator().Bytes(), nil
}

func (p *Processor) validatorUpdateBalance(op operation.ValidatorSync) ([]byte, error) {
	valAddress := op.Creator()

	valInfo := p.Storage().GetValidatorInfo(p.state, valAddress)
	if valInfo == nil {
		log.Error("Validator sync: err", "era", p.ctx.Era, "opCode", op.OpCode(), "creator", op.Creator().Hex(), "procEpoch", op.ProcEpoch(), "err", ErrUnknownValidator)
		return nil, ErrUnknownValidator
	}

	if valInfo.GetActivationEpoch() > op.ProcEpoch() {
		log.Error("Validator sync: err", "era", p.ctx.Era, "opCode", op.OpCode(), "creator", op.Creator().Hex(), "procEpoch", op.ProcEpoch(), "err", ErrNotActivatedValidator)
		return nil, ErrNotActivatedValidator
	}

	if valInfo.GetExitEpoch() == math.MaxUint64 {
		log.Error("Validator sync: err", "era", p.ctx.Era, "opCode", op.OpCode(), "creator", op.Creator().Hex(), "procEpoch", op.ProcEpoch(), "err", ErrNoExitRequest)
		return nil, ErrNoExitRequest
	}

	valInfo.SetValidatorBalance(op.Amount())

	p.Storage().SetValidatorInfo(p.state, valInfo)

	log.Info("Validator sync: update balance", "opCode", op.OpCode(), "creator", op.Creator().Hex(), "procEpoch", op.ProcEpoch(), "amount", op.Amount())
	return op.Creator().Bytes(), nil
}

type logEntry struct {
	name      string
	entryType string
	indexed   bool
	data      []byte
}

func newDataLogEntry(data []byte) logEntry {
	return logEntry{
		indexed: false,
		data:    data,
	}
}

func newIndexedPubkeyLogEntry(name string, data []byte) logEntry {
	return logEntry{
		name:      name,
		entryType: pubkeyLogType,
		indexed:   true,
		data:      data,
	}
}
func newIndexedSignatureLogEntry(name string, data []byte) logEntry {
	return logEntry{
		name:      name,
		entryType: signatureLogType,
		indexed:   true,
		data:      data,
	}
}

func newIndexedAddressLogEntry(name string, data []byte) logEntry {
	return logEntry{
		name:      name,
		entryType: addressLogType,
		indexed:   true,
		data:      data,
	}
}

func newUint256LogEntry(name string, data []byte) logEntry {
	return logEntry{
		name:      name,
		entryType: uint256LogType,
		indexed:   false,
		data:      data,
	}
}

func newBoolLogEntry(name string, data []byte) logEntry {
	return logEntry{
		name:      name,
		entryType: boolLogType,
		indexed:   false,
		data:      data,
	}
}

type EventEmmiter struct {
	state vm.StateDB
}

func NewEventEmmiter(state vm.StateDB) *EventEmmiter {
	return &EventEmmiter{state: state}
}

func (e *EventEmmiter) Deposit(evtAddr common.Address, data []byte) {
	e.addLog(
		evtAddr,
		EvtDepositLogSignature,
		data,
	)
}

func (e *EventEmmiter) ExitRequest(evtAddr common.Address, data []byte) {
	e.addLog(evtAddr, EvtExitReqLogSignature, data)
}

func (e *EventEmmiter) addLog(targetAddr common.Address, signature common.Hash, data []byte, logsEntries ...logEntry) {
	//var data []byte
	topics := []common.Hash{signature}

	for _, entry := range logsEntries {
		if entry.indexed {
			topics = append(topics, common.BytesToHash(entry.data))
		} else {
			data = append(data, entry.data...)
		}
	}

	e.state.AddLog(&types.Log{
		Address: targetAddr,
		Topics:  topics,
		Data:    data,
	})
}
