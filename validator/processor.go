package validator

import (
	"errors"
	"fmt"
	"math"
	"math/big"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/types"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/vm"
	"gitlab.waterfall.network/waterfall/protocol/gwat/crypto"
	"gitlab.waterfall.network/waterfall/protocol/gwat/log"
	"gitlab.waterfall.network/waterfall/protocol/gwat/params"
	"gitlab.waterfall.network/waterfall/protocol/gwat/validator/operation"
	valStore "gitlab.waterfall.network/waterfall/protocol/gwat/validator/storage"
)

var (
	// errors
	ErrTooLowDepositValue = errors.New("deposit value is too low")
	// ErrInsufficientFundsForTransfer is returned if the transaction sender doesn't
	// have enough funds for transfer(topmost call only).
	ErrInsufficientFundsForTransfer = errors.New("insufficient funds for transfer")
	ErrMismatchAddresses            = errors.New("withdrawal and sender addresses are mismatch")
	ErrUnknownValidator             = errors.New("unknown validator")
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
	// todo add creator_address
	depositEventSignature     = crypto.Keccak256Hash([]byte("DepositEvent"))
	exitRequestEventSignature = crypto.Keccak256Hash([]byte("ExitRequestEvent"))
	withdrawalEventSignature  = crypto.Keccak256Hash([]byte("WithdrawalEvent"))
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
}

// NewProcessor creates new validator processor
func NewProcessor(blockCtx vm.BlockContext, stateDb vm.StateDB, config *params.ChainConfig) *Processor {
	return &Processor{
		ctx:          blockCtx,
		state:        stateDb,
		eventEmmiter: NewEventEmmiter(stateDb),
		storage:      valStore.NewStorage(config),
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
//   - coordinating node: Activation
//   - validator: RequestExit
//   - coordinating node: Exit
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
		ret, err = p.syncOpProcessing(caller, v)
	case operation.ExitRequest:
		ret, err = p.validatorExitRequest(caller, toAddr, v)
	case operation.WithdrawalRequest:
		ret, err = p.validatorWithdrawalRequest(caller, toAddr, v)
	}

	if err != nil {
		p.state.RevertToSnapshot(snapshot)
	}

	return ret, err
}

func (p *Processor) validatorDeposit(caller Ref, toAddr common.Address, value *big.Int, op operation.Deposit) (_ []byte, err error) {
	from := caller.Address()

	if value == nil || value.Cmp(MinDepositVal) < 0 {
		return nil, ErrTooLowDepositValue
	}
	balanceFrom := p.state.GetBalance(from)
	if balanceFrom.Cmp(value) < 0 {
		return nil, fmt.Errorf("%w: address %v", ErrInsufficientFundsForTransfer, from.Hex())
	}

	withdrawalAddress := op.WithdrawalAddress()

	validator := valStore.NewValidator(op.CreatorAddress(), &withdrawalAddress, math.MaxUint64, math.MaxUint64, math.MaxUint64, value)

	var valInfo valStore.ValidatorInfo
	p.Storage().AddValidatorToList(p.state, validator.Address)
	valInfo = p.Storage().GetValidatorInfo(p.state, validator.Address)
	if valInfo != nil {
		currentBalance := valInfo.GetValidatorBalance()
		validator.Balance = currentBalance.Add(currentBalance, value)
	}

	valInfo, err = validator.MarshalBinary()
	if err != nil {
		return nil, err
	}

	p.Storage().SetValidatorInfo(p.state, valInfo)
	p.incrDepositCount()

	logData := PackDepositLogData(op.PubKey(), op.CreatorAddress(), op.WithdrawalAddress(), value, op.Signature(), p.getDepositCount())
	p.eventEmmiter.Deposit(toAddr, logData)

	// burn value from sender balance
	p.state.SubBalance(from, value)

	log.Info("Deposit", "address", toAddr.Hex(), "from", from.Hex(), "value", value.String(), "pabkey", op.PubKey().Hex(), "creator", op.CreatorAddress().Hex())
	return value.FillBytes(make([]byte, 32)), nil
}

func (p *Processor) validatorExitRequest(caller Ref, toAddr common.Address, op operation.ExitRequest) ([]byte, error) {
	from := caller.Address()
	validator := p.Storage().GetValidatorInfo(p.state, op.CreatorAddress())
	if validator == nil {
		return nil, ErrUnknownValidator
	}

	if from != validator.GetWithdrawalAddress() {
		return nil, ErrMismatchAddresses
	}

	logData := PackExitRequestLogData(op.PubKey(), op.CreatorAddress(), validator.GetValidatorIndex(), op.ExitAfterEpoch())
	p.eventEmmiter.ExitRequest(toAddr, logData)

	return op.CreatorAddress().Bytes(), nil
}

func (p *Processor) validatorWithdrawalRequest(caller Ref, toAddr common.Address, op operation.WithdrawalRequest) ([]byte, error) {
	from := caller.Address()
	valInfo := p.Storage().GetValidatorInfo(p.state, op.CreatorAddress())

	if from != valInfo.GetWithdrawalAddress() {
		return nil, ErrMismatchAddresses
	}

	currentBalance := valInfo.GetValidatorBalance()

	if currentBalance.Cmp(op.Amount()) == -1 {
		return nil, ErrInsufficientFundsForTransfer
	}

	valInfo.SetValidatorBalance(new(big.Int).Div(currentBalance, op.Amount()))
	p.Storage().SetValidatorInfo(p.state, valInfo)

	logData := PackWithdrawalRequestLogData(op.CreatorAddress(), op.Amount())
	p.eventEmmiter.WithdrawalRequest(toAddr, logData)

	return from.Bytes(), nil
}

func (p *Processor) syncOpProcessing(caller Ref, op operation.ValidatorSync) (ret []byte, err error) {
	switch op.OpCode() {
	case operation.ActivationCode:
		ret, err = p.validatorActivate(caller, op)
	case operation.ExitCode:
		ret, err = p.validatorExit(caller, op)
	case operation.WithdrawalCode:
		ret, err = p.validatorWithdrawal(caller, op)
	}

	return ret, err
}

func (p *Processor) validatorActivate(caller Ref, op operation.ValidatorSync) ([]byte, error) {
	from := caller.Address()

	valInfo := p.Storage().GetValidatorInfo(p.state, op.Creator())

	if valInfo == nil {
		validator := valStore.NewValidator(op.Creator(), op.WithdrawalAddress(), op.Index(), math.MaxUint64, math.MaxUint64, op.Amount())
		valInfo, err := validator.MarshalBinary()
		if err != nil {
			return nil, err
		}
		// TODO: calculate activation era and set to valInfo
		//valInfo.SetActivationEpoch(era)
		p.Storage().SetValidatorInfo(p.state, valInfo)
	} else {
		// TODO: calculate activation era and set to valInfo
		//valInfo.SetActivationEpoch(era)
		valInfo.SetValidatorIndex(op.Index())
		p.Storage().SetValidatorInfo(p.state, valInfo)
	}

	p.Storage().AddValidatorToList(p.state, op.Creator())

	log.Info("Activate", "from", from.Hex(), "creator", op.Creator().Hex(), "activationEpoch", op.ProcEpoch())

	return op.Creator().Bytes(), nil
}

func (p *Processor) validatorExit(caller Ref, op operation.ValidatorSync) ([]byte, error) {
	from := caller.Address()

	valInfo := p.Storage().GetValidatorInfo(p.state, op.Creator())
	// TODO calculate exit era and set to valInfo
	//valInfo.SetExitEpoch(era)
	p.Storage().SetValidatorInfo(p.state, valInfo)

	log.Info("Exit", "from", from.Hex(), "creator", op.Creator().Hex(), "ExitEpoch", op.ProcEpoch())

	return op.Creator().Bytes(), nil
}

func (p *Processor) validatorWithdrawal(caller Ref, op operation.ValidatorSync) ([]byte, error) {
	from := caller.Address()

	p.state.AddBalance(from, op.Amount())

	log.Info("Withdrawal", "from", from.Hex(), "creator", op.Creator().Hex(), "amount", op.Amount())
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
		depositEventSignature,
		data,
	)
}

func (e *EventEmmiter) ExitRequest(evtAddr common.Address, data []byte) {
	e.addLog(evtAddr, exitRequestEventSignature, data)
}

func (e *EventEmmiter) WithdrawalRequest(evtAddr common.Address, data []byte) {
	e.addLog(evtAddr, withdrawalEventSignature, data)
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
