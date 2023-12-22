package validator

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math"
	"math/big"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/state"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/types"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/vm"
	"gitlab.waterfall.network/waterfall/protocol/gwat/crypto"
	"gitlab.waterfall.network/waterfall/protocol/gwat/ethdb"
	"gitlab.waterfall.network/waterfall/protocol/gwat/log"
	"gitlab.waterfall.network/waterfall/protocol/gwat/params"
	"gitlab.waterfall.network/waterfall/protocol/gwat/validator/era"
	"gitlab.waterfall.network/waterfall/protocol/gwat/validator/operation"
	valStore "gitlab.waterfall.network/waterfall/protocol/gwat/validator/storage"
)

var (
	// errors
	ErrTooLowDepositValue = errors.New("deposit value is too low")
	// ErrInsufficientFundsForOp is returned if the transaction sender doesn't
	// have enough funds for transfer(topmost call only).
	ErrInsufficientFundsForOp = errors.New("insufficient funds for op")
	ErrInvalidFromAddresses   = errors.New("withdrawal and sender addresses are mismatch")
	ErrUnknownValidator       = errors.New("unknown validator")
	ErrNoWithdrawalCred       = errors.New("no withdrawal credentials")
	ErrMismatchPulicKey       = errors.New("validators public key mismatch")
	ErrNotActivatedValidator  = errors.New("validator not activated yet")
	ErrValidatorIsOut         = errors.New("validator is exited")
	ErrInvalidToAddress       = errors.New("address to must be validators state address")
	ErrNoExitRequest          = errors.New("exit request is require before withdrawal operation")
	ErrTargetEraNotFound      = errors.New("target era not found")
	ErrNoSavedValSyncOp       = errors.New("no coordinated confirmation of validator sync data")
	ErrMismatchTxHashes       = errors.New("validator sync tx already exists")
	ErrMismatchValSyncOp      = errors.New("validator sync tx data is not conforms to coordinated confirmation data")
	ErrInvalidOpEpoch         = errors.New("epoch to apply tx is not acceptable")
	ErrTxNF                   = errors.New("tx not found")
	ErrReceiptNF              = errors.New("receipt not found")
	ErrInvalidReceiptStatus   = errors.New("receipt status is failed")
	ErrInvalidOpCode          = errors.New("invalid operation code")
	ErrInvalidCreator         = errors.New("invalid creator address")
	ErrInvalidAmount          = errors.New("invalid amount")
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
	postpone         = 2
)

var (
	// minimal value - 10 wat
	MinDepositVal, _ = new(big.Int).SetString("10000000000000000000", 10)

	//Events signatures.
	EvtDepositLogSignature    = crypto.Keccak256Hash([]byte("DepositLog"))
	EvtExitReqLogSignature    = crypto.Keccak256Hash([]byte("ExitRequestLog"))
	EvtWithdrawalLogSignature = crypto.Keccak256Hash([]byte("WithdrawaRequestLog"))
)

// Ref represents caller of the validator processor
type Ref interface {
	Address() common.Address
}

type blockchain interface {
	era.Blockchain
	EpochToEra(epoch uint64) *era.Era
	GetSlotInfo() *types.SlotInfo
	GetEraInfo() *era.EraInfo
	Config() *params.ChainConfig
	Database() ethdb.Database
	GetValidatorSyncData(InitTxHash common.Hash) *types.ValidatorSync
	GetTransaction(txHash common.Hash) (tx *types.Transaction, blHash common.Hash, index uint64)
	GetTransactionReceipt(txHash common.Hash) (rc *types.Receipt, blHash common.Hash, index uint64)
	GetLastCoordinatedCheckpoint() *types.Checkpoint
	ValidatorStorage() valStore.Storage
	StateAt(root common.Hash) (*state.StateDB, error)
	GetBlock(ctx context.Context, hash common.Hash) *types.Block
	GetEpoch(epoch uint64) common.Hash
}

type message interface {
	TxHash() common.Hash
	Data() []byte
}

// Processor is a processor of all validator related operations.
// All transaction related operations that mutates state of the validator are called using Call method.
// Methods of the operation name are used for getting state of the validator.
type Processor struct {
	state        vm.StateDB
	ctx          vm.BlockContext
	eventEmmiter *EventEmmiter
	storage      valStore.Storage
	blockchain   blockchain
}

// NewProcessor creates new validator processor
func NewProcessor(blockCtx vm.BlockContext, stateDb vm.StateDB, bc blockchain) *Processor {
	return &Processor{
		ctx:          blockCtx,
		state:        stateDb,
		eventEmmiter: NewEventEmmiter(stateDb),
		storage:      valStore.NewStorage(bc.Config()),
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
func (p *Processor) Call(caller Ref, toAddr common.Address, value *big.Int, msg message) (ret []byte, err error) {
	op, err := operation.DecodeBytes(msg.Data())
	if err != nil {
		return nil, err
	}

	nonce := p.state.GetNonce(caller.Address())
	p.state.SetNonce(caller.Address(), nonce+1)

	snapshot := p.state.Snapshot()

	ret = nil
	switch v := op.(type) {
	case operation.Deposit:
		ret, err = p.validatorDeposit(caller, toAddr, value, v)
		if err != nil {
			log.Error("Validator deposit: err",
				"opCode", op.OpCode(),
				"tx", msg.TxHash().Hex(),
				"amount", value.String(),
				"from", caller.Address(),
				"creator", v.CreatorAddress().Hex(),
				"withdrawalAddress", v.WithdrawalAddress().Hex(),
				"pubKey", v.PubKey().Hex(),
				"delegatedStake", v.DelegatedStake() != nil,
				"err", err,
			)
		} else {
			log.Info("Validator deposit: success",
				"opCode", op.OpCode(),
				"tx", msg.TxHash().Hex(),
				"amount", value.String(),
				"from", caller.Address(),
				"creator", v.CreatorAddress().Hex(),
				"withdrawalAddress", v.WithdrawalAddress().Hex(),
				"pubKey", v.PubKey().Hex(),
				"delegatedStake", v.DelegatedStake() != nil,
			)
		}
	case operation.ValidatorSync:
		ret, err = p.syncOpProcessing(v, msg)
		if err != nil {
			log.Error("Validator sync: err",
				"opCode", op.OpCode(),
				"initTx", v.InitTxHash().Hex(),
				"creator", v.Creator().Hex(),
				"procEpoch", v.ProcEpoch(),
				"err", err,
			)
		} else {
			log.Info("Validator sync: success",
				"opCode", op.OpCode(),
				"initTx", v.InitTxHash().Hex(),
				"creator", v.Creator().Hex(),
				"procEpoch", v.ProcEpoch(),
			)
		}
	case operation.Exit:
		ret, err = p.validatorExit(caller, toAddr, v)
		if err != nil {
			log.Error("Validator exit: err",
				"opCode", op.OpCode(),
				"tx", msg.TxHash().Hex(),
				"creator", v.CreatorAddress().Hex(),
				"exitAfterEpoch", fmt.Sprintf("%d", v.ExitAfterEpoch()),
				"err", err,
			)
		} else {
			log.Info("Validator exit: success",
				"opCode", op.OpCode(),
				"tx", msg.TxHash().Hex(),
				"creator", v.CreatorAddress().Hex(),
				"exitAfterEpoch", fmt.Sprintf("%d", v.ExitAfterEpoch()),
			)
		}
	case operation.Withdrawal:
		ret, err = p.validatorWithdrawal(caller, toAddr, v)
		if err != nil {
			log.Error("Validator withdrawal: err",
				"opCode", op.OpCode(),
				"tx", msg.TxHash().Hex(),
				"amount", v.Amount().String(),
				"creator", v.CreatorAddress().Hex(),
				"err", err,
			)
		} else {
			log.Info("Validator withdrawal: success",
				"opCode", op.OpCode(),
				"tx", msg.TxHash().Hex(),
				"amount", v.Amount().String(),
				"creator", v.CreatorAddress().Hex(),
			)
		}
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

	if value == nil || value.Cmp(MinDepositVal) < 0 {
		return nil, ErrTooLowDepositValue
	}

	// check amount can add to log
	if !common.BnCanCastToUint64(new(big.Int).Div(value, common.BigGwei)) {
		return nil, ErrInvalidAmount
	}

	from := caller.Address()

	balanceFrom := p.state.GetBalance(from)
	if balanceFrom.Cmp(value) < 0 {
		return nil, fmt.Errorf("%w: address %v", ErrInsufficientFundsForOp, from.Hex())
	}

	withdrawalAddress := op.WithdrawalAddress()

	validator := valStore.NewValidator(op.PubKey(), op.CreatorAddress(), &withdrawalAddress)

	// if validator already exist
	currValidator, _ := p.Storage().GetValidator(p.state, op.CreatorAddress())

	if currValidator != nil {
		if currValidator.ActivationEra < math.MaxUint64 {
			return nil, errors.New("validator deposit failed (validator already activated)")
		}

		if currValidator.PubKey != op.PubKey() {
			return nil, errors.New("validator deposit failed (mismatch public key)")
		}
		validator = currValidator
	}

	validator.AddStake(from, value)

	err = p.Storage().SetValidator(p.state, validator)
	if err != nil {
		return nil, err
	}

	logData := PackDepositLogData(op.PubKey(), op.CreatorAddress(), op.WithdrawalAddress(), value, op.Signature(), p.getDepositCount())
	p.eventEmmiter.Deposit(toAddr, logData)
	p.incrDepositCount()
	// burn value from sender balance
	p.state.SubBalance(from, value)

	return value.FillBytes(make([]byte, 32)), nil
}

func (p *Processor) validatorExit(caller Ref, toAddr common.Address, op operation.Exit) ([]byte, error) {
	if !p.IsValidatorOp(&toAddr) {
		return nil, ErrInvalidToAddress
	}

	from := caller.Address()
	validator, err := p.Storage().GetValidator(p.state, op.CreatorAddress())
	if err != nil {
		return nil, err
	}

	if validator == nil {
		return nil, ErrUnknownValidator
	}

	if validator.GetPubKey() != op.PubKey() {
		return nil, ErrMismatchPulicKey
	}

	if op.ExitAfterEpoch() == nil {
		exitAftEpoch := p.blockchain.GetSlotInfo().SlotToEpoch(p.blockchain.GetSlotInfo().CurrentSlot()) + 1
		op.SetExitAfterEpoch(&exitAftEpoch)
	}

	if validator.GetActivationEra() > p.blockchain.GetEraInfo().Number() {
		return nil, ErrNotActivatedValidator
	}

	if validator.GetExitEra() != math.MaxUint64 {
		return nil, ErrValidatorIsOut
	}

	if from != *validator.GetWithdrawalAddress() {
		return nil, ErrInvalidFromAddresses
	}

	logData := PackExitRequestLogData(op.PubKey(), op.CreatorAddress(), validator.GetIndex(), op.ExitAfterEpoch())
	p.eventEmmiter.ExitRequest(toAddr, logData)

	return op.CreatorAddress().Bytes(), nil
}

func (p *Processor) validatorWithdrawal(caller Ref, toAddr common.Address, op operation.Withdrawal) ([]byte, error) {
	if !p.IsValidatorOp(&toAddr) {
		return nil, ErrInvalidToAddress
	}

	// check amount can add to log
	opAmount := new(big.Int).Set(op.Amount())
	if !common.BnCanCastToUint64(new(big.Int).Div(opAmount, common.BigGwei)) {
		return nil, ErrInvalidAmount
	}

	from := caller.Address()
	validator, err := p.Storage().GetValidator(p.state, op.CreatorAddress())
	if err != nil {
		return nil, err
	}

	if validator == nil {
		return nil, ErrUnknownValidator
	}

	// if total deposited amount is less than the effective balance
	// - deposit is insufficient to activate validator.
	effectiveBalanceWei := new(big.Int).Mul(p.blockchain.Config().EffectiveBalance, common.BigWat)
	if stake := validator.TotalStake(); validator.GetActivationEra() == math.MaxUint64 &&
		stake != nil && stake.Cmp(effectiveBalanceWei) < 0 {
		// withdrawal of insufficient deposit to activate validator
		log.Info("Validator withdrawal: refunds of insufficient deposit",
			"opCode", op.OpCode(),
			"amount", opAmount.String(),
			"creator", op.CreatorAddress().Hex(),
		)
		stakeByAddr := validator.StakeByAddress(from)
		//withdrawal address must be one of the depositors
		if stakeByAddr == nil || stakeByAddr.Cmp(new(big.Int)) == 0 {
			return nil, ErrInvalidFromAddresses
		}
		// check amount
		if stakeByAddr.Cmp(opAmount) < 0 {
			return nil, ErrInsufficientFundsForOp
		}
		// if opAmount == 0 - refunds full deposit amount
		if opAmount.Cmp(common.Big0) == 0 {
			opAmount = new(big.Int).Set(stakeByAddr)
		}
		// withdrawal amount from deposit balance
		newStake, err := validator.SubtractStake(from, opAmount)
		if err != nil {
			log.Error("Validator withdrawal: refunds of insufficient deposit failed",
				"opCode", op.OpCode(),
				"amount", opAmount.String(),
				"creator", op.CreatorAddress().Hex(),
				"error", err,
			)
			return nil, err
		}
		// rm validator stake if empty
		if newStake != nil && newStake.Cmp(common.Big0) == 0 {
			validator.RmStakeByAddress(from)
		}
		// update validator
		err = p.Storage().SetValidator(p.state, validator)
		if err != nil {
			return nil, err
		}
	} else {
		// check withdrawal credentials
		if from != *validator.GetWithdrawalAddress() {
			return nil, ErrInvalidFromAddresses
		}
	}

	// create tx log
	amtGwei := new(big.Int).Div(opAmount, common.BigGwei).Uint64()

	logData := PackWithdrawalLogData(validator.GetPubKey(), op.CreatorAddress(), validator.GetIndex(), amtGwei)

	p.eventEmmiter.WithdrawalRequest(toAddr, logData)

	return op.CreatorAddress().Bytes(), nil
}

func (p *Processor) validatorDelegateStake(caller Ref, toAddr common.Address, value *big.Int, op operation.DelegateStake) (_ []byte, err error) {
	if !p.IsValidatorOp(&toAddr) {
		return nil, ErrInvalidToAddress
	}

	if value == nil || value.Cmp(MinDepositVal) < 0 {
		return nil, ErrTooLowDepositValue
	}

	// check amount can add to log
	if !common.BnCanCastToUint64(new(big.Int).Div(value, common.BigGwei)) {
		return nil, ErrInvalidAmount
	}

	from := caller.Address()

	balanceFrom := p.state.GetBalance(from)
	if balanceFrom.Cmp(value) < 0 {
		return nil, fmt.Errorf("%w: address %v", ErrInsufficientFundsForOp, from.Hex())
	}

	//todo
	withdrawalAddress := common.Address{}

	validator := valStore.NewValidator(op.PubKey(), op.CreatorAddress(), &withdrawalAddress)

	validator.DelegateStake, err = valStore.NewDelegateStakeData(op.Rules(), op.TrialPeriod(), op.TrialRules())
	if err != nil {
		return nil, err
	}

	// if validator already exist
	currValidator, _ := p.Storage().GetValidator(p.state, op.CreatorAddress())

	if currValidator != nil {
		if currValidator.ActivationEra < math.MaxUint64 {
			return nil, errors.New("validator deposit failed (validator already activated)")
		}

		if currValidator.PubKey != op.PubKey() {
			return nil, errors.New("validator deposit failed (mismatch public key)")
		}
		validator = currValidator
	}

	validator.AddStake(from, value)

	err = p.Storage().SetValidator(p.state, validator)
	if err != nil {
		return nil, err
	}

	logData := PackDepositLogData(op.PubKey(), op.CreatorAddress(), withdrawalAddress, value, op.Signature(), p.getDepositCount())
	p.eventEmmiter.Deposit(toAddr, logData)
	p.incrDepositCount()
	// burn value from sender balance
	p.state.SubBalance(from, value)

	return value.FillBytes(make([]byte, 32)), nil
}

func (p *Processor) syncOpProcessing(op operation.ValidatorSync, msg message) (ret []byte, err error) {
	if err = ValidateValidatorSyncOp(p.blockchain, op, p.ctx.Slot, msg.TxHash()); err != nil {
		log.Error("Invalid validator sync op",
			"error", err,
			"OpType", op.OpType(),
			"OpCode", op.OpCode(),
			"Creator", op.Creator().Hex(),
			"InitTxHash", op.InitTxHash().Hex(),
		)
		return nil, err
	}

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
	validator, err := p.Storage().GetValidator(p.state, op.Creator())
	if err != nil {
		return nil, err
	}
	if validator == nil {
		return nil, ErrUnknownValidator
	}

	opEra := p.blockchain.EpochToEra(op.ProcEpoch())

	validator.SetActivationEra(opEra.Number + postpone)
	validator.SetIndex(op.Index())
	validator.UnsetStake()
	err = p.Storage().SetValidator(p.state, validator)
	if err != nil {
		return nil, err
	}

	p.Storage().AddValidatorToList(p.state, op.Index(), op.Creator())

	return op.Creator().Bytes(), nil
}

func (p *Processor) validatorDeactivate(op operation.ValidatorSync) ([]byte, error) {
	validator, err := p.Storage().GetValidator(p.state, op.Creator())
	if err != nil {
		return nil, err
	}
	if validator == nil {
		return nil, ErrUnknownValidator
	}
	if validator.GetExitEra() != math.MaxUint64 {
		return nil, ErrValidatorIsOut
	}

	opEra := p.blockchain.EpochToEra(op.ProcEpoch())
	if validator.GetActivationEra() >= opEra.Number {
		return nil, ErrNotActivatedValidator
	}

	exitEra := opEra.Number + postpone
	validator.SetExitEra(exitEra)
	err = p.Storage().SetValidator(p.state, validator)
	if err != nil {
		return nil, err
	}
	return op.Creator().Bytes(), nil
}

func (p *Processor) validatorUpdateBalance(op operation.ValidatorSync) ([]byte, error) {
	valAddress := op.Creator()
	validator, err := p.Storage().GetValidator(p.state, valAddress)
	if err != nil {
		return nil, err
	}
	if validator == nil {
		return nil, ErrUnknownValidator
	}
	var withdrawalTo *common.Address
	// if total deposited amount is less than the effective balance
	// - deposit is insufficient to activate validator.
	effectiveBalanceWei := new(big.Int).Mul(p.blockchain.Config().EffectiveBalance, common.BigWat)
	if stake := validator.TotalStake(); validator.GetActivationEra() == math.MaxUint64 &&
		stake != nil && stake.Cmp(effectiveBalanceWei) < 0 {
		// withdrawal of insufficient deposit to activate validator
		log.Info("Validator update balance: refunds of insufficient deposit",
			"opCode", op.OpCode(),
			"InitTxHash", op.InitTxHash().Hex(),
			"amount", op.Amount().String(),
			"procEpoch", op.ProcEpoch(),
			"vIndex", op.Index(),
			"creator", op.Creator().Hex(),
		)
		// check initial tx
		initTx, _, _ := p.blockchain.GetTransaction(op.InitTxHash())
		if initTx == nil {
			return nil, ErrTxNF
		}
		// check init tx data
		iop, err := operation.DecodeBytes(initTx.Data())
		if err != nil {
			log.Error("can`t unmarshal validator sync operation from tx data", "err", err)
			return nil, err
		}
		if iop.OpCode() != operation.WithdrawalCode {
			return nil, ErrInvalidOpCode
		}
		// withdrawal to sender of initial tx
		signer := types.LatestSigner(p.blockchain.Config())
		iTxFrom, _ := types.Sender(signer, initTx)
		withdrawalTo = &iTxFrom
	} else {
		// set withdrawal credentials
		withdrawalTo = validator.GetWithdrawalAddress()
		if withdrawalTo == nil {
			return nil, ErrNoWithdrawalCred
		}
	}
	// transfer amount to withdrawal address
	p.state.AddBalance(*withdrawalTo, op.Amount())

	return op.Creator().Bytes(), nil
}

type logEntry struct {
	// name string
	//entryType string
	indexed bool
	data    []byte
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

func (e *EventEmmiter) WithdrawalRequest(evtAddr common.Address, data []byte) {
	e.addLog(evtAddr, EvtWithdrawalLogSignature, data)
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

// ValidateValidatorSyncOp validate validator sync op data with context of apply.
func ValidateValidatorSyncOp(bc blockchain, valSyncOp operation.ValidatorSync, applySlot uint64, txHash common.Hash) error {
	savedValSync := bc.GetValidatorSyncData(valSyncOp.InitTxHash())
	if savedValSync == nil {
		return ErrNoSavedValSyncOp
	}

	blockEpoch := bc.GetSlotInfo().SlotToEpoch(applySlot)
	if blockEpoch > valSyncOp.ProcEpoch() {
		return ErrInvalidOpEpoch
	}
	if !CompareValSync(savedValSync, valSyncOp) {
		return ErrMismatchValSyncOp
	}
	if savedValSync.TxHash != nil && *savedValSync.TxHash != txHash {
		return ErrMismatchTxHashes
	}

	// check is op initialised by slashing
	if bytes.Equal(valSyncOp.InitTxHash().Bytes()[:common.AddressLength], valSyncOp.Creator().Bytes()) {
		return nil
	}

	// check initial tx
	initTx, _, _ := bc.GetTransaction(valSyncOp.InitTxHash())
	if initTx == nil {
		return ErrTxNF
	}
	// check init tx data
	iop, err := operation.DecodeBytes(initTx.Data())
	if err != nil {
		log.Error("can`t unmarshal validator sync operation from tx data", "err", err)
		return err
	}
	switch initTxData := iop.(type) {
	case operation.Deposit:
		if valSyncOp.OpCode() == operation.DepositCode {
			return ErrInvalidOpCode
		}
		if initTxData.CreatorAddress() != valSyncOp.Creator() {
			return ErrInvalidCreator
		}
	case operation.Exit:
		if valSyncOp.OpCode() == operation.ExitCode {
			return ErrInvalidOpCode
		}
		if initTxData.CreatorAddress() != valSyncOp.Creator() {
			return ErrInvalidCreator
		}
		if initTxData.ExitAfterEpoch() != nil && *initTxData.ExitAfterEpoch() < valSyncOp.ProcEpoch() {
			return ErrInvalidOpEpoch
		}
	case operation.Withdrawal:
		if valSyncOp.OpCode() == operation.WithdrawalCode {
			return ErrInvalidOpCode
		}
		if initTxData.CreatorAddress() != valSyncOp.Creator() {
			return ErrInvalidCreator
		}
	default:
		return ErrInvalidOpCode
	}

	// check initial tx status
	rc, _, _ := bc.GetTransactionReceipt(valSyncOp.InitTxHash())
	if rc == nil {
		return ErrReceiptNF
	}
	if rc.Status != types.ReceiptStatusSuccessful {
		return ErrInvalidReceiptStatus
	}

	return nil
}

func CompareValSync(saved *types.ValidatorSync, input operation.ValidatorSync) bool {
	if saved.InitTxHash != input.InitTxHash() {
		log.Warn("check validator sync failed: InitTxHash", "s.InitTxHash", saved.InitTxHash.Hex(), "i.InitTxHash", input.InitTxHash().Hex())
		return false
	}

	if saved.OpType != input.OpType() {
		log.Warn("check validator sync failed: OpType", "s.OpType", saved.OpType, "i.OpType", input.OpType())
		return false
	}

	if saved.Creator != input.Creator() {
		log.Warn("check validator sync failed: Creator",
			"s.Creator", fmt.Sprintf("%#x", saved.Creator),
			"i.Creator", fmt.Sprintf("%#x", input.Creator()))
		return false
	}

	if saved.Index != input.Index() {
		log.Warn("check validator sync failed: Index", "s.Index", saved.Index, "i.Index", input.Index())
		return false
	}

	if saved.ProcEpoch != input.ProcEpoch() {
		log.Warn("check validator sync failed: ProcEpoch", "s.ProcEpoch", saved.ProcEpoch, "i.ProcEpoch", input.ProcEpoch())
		return false
	}

	if saved.Amount != nil && input.Amount() != nil && saved.Amount.Cmp(input.Amount()) != 0 {
		log.Warn("check validator sync failed: Amount", "s.Amount", saved.Amount.String(), "i.Amount", input.Amount().String())
		return false
	}

	if saved.Amount != nil && input.Amount() == nil || saved.Amount == nil && input.Amount() != nil {
		log.Warn("check validator sync failed: Amount nil", "s.Amount", saved.Amount, "i.Amount", input.Amount())
		return false
	}

	return true
}
