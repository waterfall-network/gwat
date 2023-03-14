package validator

import (
	"bytes"
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
	ErrToLowExitEpoch               = errors.New("exit epoch is less than activation epoch")
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
	depositEventSignature     = crypto.Keccak256Hash([]byte("DepositEvent(bytes,bytes,bytes,bytes,bytes)"))
	exitRequestEventSignature = crypto.Keccak256Hash([]byte("ExitRequestEvent(bytes,bytes,bytes,bytes,bytes)"))
	withdrawalEventSignature  = crypto.Keccak256Hash([]byte("WithdrawalEvent(bytes,bytes,bytes,bytes,bytes)"))
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
		p.syncOpProcessing(caller, toAddr, v)
	case operation.ExitRequest:
		err = p.validatorExitRequest(caller, toAddr, v)
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

	validator := valStore.NewValidator(op.CreatorAddress(), &withdrawalAddress, 0, math.MaxUint64, math.MaxUint64, value)

	var valInfo valStore.ValidatorInfo
	valIndex, ok := p.Storage().AddValidatorToList(validator, p.state)
	if ok {
		log.Info("validator already exist")

		valInfo = p.Storage().GetValidatorInfo(p.state, validator.Address)
		currentBalance := valInfo.GetValidatorBalance()
		validator.Balance = currentBalance.Add(currentBalance, value)
	}

	validator.Index = valIndex

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

func (p *Processor) validatorExitRequest(caller Ref, toAddr common.Address, op operation.ExitRequest) error {
	from := caller.Address()
	validator := p.Storage().GetValidatorInfo(p.state, op.ValidatorAddress())
	if validator == nil {
		return ErrUnknownValidator
	}

	if !bytes.Equal(from.Bytes(), validator.GetWithdrawalAddress().Bytes()) {
		return ErrMismatchAddresses
	}

	if op.ExitEpoch() < validator.GetActivationEpoch() {
		return ErrToLowExitEpoch
	}

	validator.SetExitEpoch(op.ExitEpoch())
	p.Storage().SetValidatorInfo(p.state, validator)

	logData := PackExitRequestLogData(op.PubKey(), op.ValidatorAddress(), op.ExitEpoch())
	p.eventEmmiter.ExitRequest(toAddr, logData)

	return nil
}

func (p *Processor) validatorWithdrawalRequest(caller Ref, toAddr common.Address, op operation.WithdrawalRequest) error {
	from := caller.Address()
	valInfo := p.Storage().GetValidatorInfo(p.state, op.ValidatorAddress())

	if !bytes.Equal(from.Bytes(), valInfo.GetWithdrawalAddress().Bytes()) {
		return ErrMismatchAddresses
	}

	valInfo.SetValidatorBalance(new(big.Int).Div(valInfo.GetValidatorBalance(), op.Amount()))
	p.Storage().SetValidatorInfo(p.state, valInfo)

	logData := PackWithdrawalRequestLogData(op.ValidatorAddress(), op.Amount())
	p.eventEmmiter.WithdrawalRequest(toAddr, logData)

	return nil
}

func (p *Processor) syncOpProcessing(caller Ref, toAddr common.Address, op operation.ValidatorSync) {
	switch op.OpCode() {
	case operation.ActivationCode:
		p.validatorActivate(caller, toAddr, op)
	case operation.ExitCode:
		p.validatorExit(caller, toAddr, op)
	}
}

func (p *Processor) validatorActivate(caller Ref, toAddr common.Address, op operation.ValidatorSync) {
	from := caller.Address()

	valInfo := p.Storage().GetValidatorInfo(p.state, op.Creator())
	valInfo.SetActivationEpoch(op.ProcEpoch())
	p.Storage().SetValidatorInfo(p.state, valInfo)

	log.Info("Activate", "address", toAddr.Hex(), "from", from.Hex(), "creator", op.Creator().Hex(), "activationEpoch", op.ProcEpoch())
}

func (p *Processor) validatorExit(caller Ref, toAddr common.Address, op operation.ValidatorSync) {
	from := caller.Address()

	valList := p.Storage().GetValidatorsList(p.state)
	for i, address := range valList {
		if bytes.Equal(address.Bytes(), op.Creator().Bytes()) {
			valList = append(valList[:i], valList[i+1:]...)
		}
	}

	p.Storage().SetValidatorsList(p.state, valList)

	log.Info("Exit", "address", toAddr.Hex(), "from", from.Hex(), "creator", op.Creator().Hex(), "ExitEpoch", op.ProcEpoch())
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
