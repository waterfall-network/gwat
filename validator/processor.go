package validator

import (
	"errors"
	"fmt"
	"math/big"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/types"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/vm"
	"gitlab.waterfall.network/waterfall/protocol/gwat/crypto"
	"gitlab.waterfall.network/waterfall/protocol/gwat/log"
	"gitlab.waterfall.network/waterfall/protocol/gwat/validator/operation"
	validatorStorage "gitlab.waterfall.network/waterfall/protocol/gwat/validator/storage"
)

var (
	// errors
	ErrTooLowDepositValue = errors.New("deposit value is too low")
	// ErrInsufficientFundsForTransfer is returned if the transaction sender doesn't
	// have enough funds for transfer(topmost call only).
	ErrInsufficientFundsForTransfer = errors.New("insufficient funds for transfer")
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
	depositEventSignature = crypto.Keccak256Hash([]byte("DepositEvent(bytes,bytes,bytes,bytes,bytes)"))
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
}

// NewProcessor creates new validator processor
func NewProcessor(blockCtx vm.BlockContext, stateDb vm.StateDB) *Processor {
	return &Processor{
		ctx:          blockCtx,
		state:        stateDb,
		eventEmmiter: NewEventEmmiter(stateDb),
	}
}

//todo get the address from genesis config
// IsValidatorOp returns true if tx is validator operation
func (p *Processor) GetValidatorsStateAddress() common.Address {
	return common.HexToAddress("0x1111111111111111111111111111111111111111")
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
//  * validator: Deposit
//  * coordinating node: Activation
//  * validator: RequestExit
//  * coordinating node: Exit
// It returns byte representation of the return value of an operation.
func (p *Processor) Call(caller Ref, toAddrs common.Address, value *big.Int, op operation.Operation) (ret []byte, err error) {
	nonce := p.state.GetNonce(caller.Address())
	p.state.SetNonce(caller.Address(), nonce+1)

	snapshot := p.state.Snapshot()

	ret = nil
	switch v := op.(type) {
	case operation.Deposit:
		ret, err = p.validatorDeposit(caller, toAddrs, value, v)

		//	todo implement
		//case operation.Activation:
		//	ret, err = p.activation(caller, toAddr, v)
		//case operation.RequestExit:
		//	ret, err = p.requestExitFrom(caller, toAddr, v)
		//case operation.Exit:
		//	ret, err = p.exitFrom(caller, toAddr, v)
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

	// burn value from sender balance
	p.state.SubBalance(from, value)

	//todo add creator to list of creators
	// predefined account of common validators info
	validatorsStorage, err := p.newStorage(toAddr)
	if err != nil {
		return nil, err
	}

	//todo update state of creator
	// creators account
	creatorAddr := op.CreatorAddress()
	creatorStorage, err := p.newStorage(creatorAddr)
	if err != nil {
		return nil, err
	}

	//op.DepositDataRoot()

	//todo
	deposit_count := uint64(222)

	logData := PackDepositLogData(op.PubKey(), op.CreatorAddress(), op.WithdrawalAddress(), value, op.Signature(), deposit_count)

	defer p.eventEmmiter.Deposit(toAddr, logData)

	log.Info("Deposit", "address", toAddr.Hex(), "from", from.Hex(), "value", value.String(), "pabkey", op.PubKey().Hex(), "creator", op.CreatorAddress().Hex())
	validatorsStorage.Flush()
	creatorStorage.Flush()

	return value.FillBytes(make([]byte, 32)), nil

	//todo
	return toAddr.Bytes(), nil
}

func (p *Processor) newStorage(addr common.Address) (validatorStorage.Storage, error) {
	storage, err := validatorStorage.ReadStorage(validatorStorage.NewStorageStream(addr, p.state))
	if err != nil {
		return nil, err
	}
	return storage, nil
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
