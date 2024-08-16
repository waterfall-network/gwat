package validator

import (
	"context"
	"errors"
	"math/big"
	"time"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/common/hexutil"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/rawdb"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/state"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/types"
	"gitlab.waterfall.network/waterfall/protocol/gwat/ethdb"
	"gitlab.waterfall.network/waterfall/protocol/gwat/log"
	"gitlab.waterfall.network/waterfall/protocol/gwat/params"
	"gitlab.waterfall.network/waterfall/protocol/gwat/rpc"
	"gitlab.waterfall.network/waterfall/protocol/gwat/validator/era"
	"gitlab.waterfall.network/waterfall/protocol/gwat/validator/operation"
	valStore "gitlab.waterfall.network/waterfall/protocol/gwat/validator/storage"
	"gitlab.waterfall.network/waterfall/protocol/gwat/validator/txlog"
)

type Backend interface {
	StateAndHeaderByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*state.StateDB, *types.Header, error)
	RPCEVMTimeout() time.Duration // global timeout for eth_call over rpc: DoS protection
	GetVP(ctx context.Context, state *state.StateDB, header *types.Header) (*Processor, func() error, error)
	GetLastFinalizedBlock() *types.Block
	ChainConfig() *params.ChainConfig
	GetBlockFinalizedNumber(hash common.Hash) *uint64
	HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error)
	GetTransaction(context.Context, common.Hash) (*types.Transaction, common.Hash, uint64, error)
	GetReceipts(ctx context.Context, hash common.Hash) (types.Receipts, error)
}
type Blockchain interface {
	ValidatorStorage() valStore.Storage
	StateAt(root common.Hash) (*state.StateDB, error)
	GetBlock(ctx context.Context, hash common.Hash) *types.Block
	GetSlotInfo() *types.SlotInfo
	GetLastCoordinatedCheckpoint() *types.Checkpoint
	Database() ethdb.Database
	GetEpoch(epoch uint64) common.Hash
	EpochToEra(uint64) *era.Era
	GetEraInfo() *era.EraInfo
}

// PublicValidatorAPI provides an API to access validator functions.
type PublicValidatorAPI struct {
	b     Backend
	chain Blockchain
}

// NewPublicValidatorAPI creates a new validator API.
func NewPublicValidatorAPI(b Backend, chain Blockchain) *PublicValidatorAPI {
	return &PublicValidatorAPI{b, chain}
}

// GetAPIs provides api access
func GetAPIs(apiBackend Backend, chain Blockchain) []rpc.API {
	return []rpc.API{
		{
			Namespace: "wat",
			Version:   "1.0",
			Service:   NewPublicValidatorAPI(apiBackend, chain),
			Public:    true,
		},
	}
}

type DepositArgs struct {
	PubKey            *common.BlsPubKey    `json:"pubkey"`             // validator public key
	CreatorAddress    *common.Address      `json:"creator_address"`    // attached creator account
	WithdrawalAddress *common.Address      `json:"withdrawal_address"` // attached withdrawal credentials
	Signature         *common.BlsSignature `json:"signature"`
	DelegatingStake   *DelegatingStakeArgs `json:"delegating_stake"`
}

type DelegatingStakeArgs struct {
	Rules       *DelegatingRulesArgs `json:"rules"`        // rules after trial period
	TrialPeriod *uint64              `json:"trial_period"` // period while trial_rules are active (in slots, starts from activation slot)
	TrialRules  *DelegatingRulesArgs `json:"trial_rules"`  // rules for trial period
}

type DelegatingRulesArgs struct {
	ProfitShare *map[common.Address]uint8 `json:"profit_share"` // map of participants profit share in %
	StakeShare  *map[common.Address]uint8 `json:"stake_share"`  // map of participants stake share in % (after exit)
	Exit        *[]common.Address         `json:"exit"`         // addresses of role  to init exit
	Withdrawal  *[]common.Address         `json:"withdrawal"`   // addresses of role  to init exit
}

// Validator_DepositData creates a validators deposit data for deposit tx.
func (s *PublicValidatorAPI) Validator_DepositData(_ context.Context, args DepositArgs) (hexutil.Bytes, error) {
	if args.PubKey == nil {
		return nil, operation.ErrNoPubKey
	}
	if args.CreatorAddress == nil {
		return nil, operation.ErrNoCreatorAddress
	}
	if args.WithdrawalAddress == nil {
		return nil, operation.ErrNoWithdrawalAddress
	}
	if args.Signature == nil {
		return nil, operation.ErrNoSignature
	}

	var (
		op                operation.Operation
		err               error
		delegatingStake   *operation.DelegatingStakeData
		rules, trialRules *operation.DelegatingStakeRules
	)
	if args.DelegatingStake != nil {
		dlgStakeArg := args.DelegatingStake
		if dlgStakeArg.Rules == nil {
			return nil, operation.ErrNoRules
		}
		if dlgStakeArg.TrialPeriod == nil {
			def := uint64(0)
			dlgStakeArg.TrialPeriod = &def
		}
		if dlgStakeArg.TrialRules == nil {
			dlgStakeArg.TrialRules = &DelegatingRulesArgs{}
		}
		ar := dlgStakeArg.Rules
		rules, err = operation.NewDelegatingStakeRules(*ar.ProfitShare, *ar.StakeShare, *ar.Exit, *ar.Withdrawal)
		if err != nil {
			return nil, err
		}
		atr := dlgStakeArg.TrialRules
		trialRules, err = operation.NewDelegatingStakeRules(*atr.ProfitShare, *atr.StakeShare, *atr.Exit, *atr.Withdrawal)
		if err != nil {
			return nil, err
		}

		if delegatingStake, err = operation.NewDelegatingStakeData(rules, *dlgStakeArg.TrialPeriod, trialRules); err != nil {
			return nil, err
		}
	}

	if op, err = operation.NewDepositOperation(*args.PubKey, *args.CreatorAddress, *args.WithdrawalAddress, *args.Signature, delegatingStake); err != nil {
		return nil, err
	}

	b, err := operation.EncodeToBytes(op)
	if err != nil {
		log.Warn("Failed to encode validator deposit operation", "err", err)
		return nil, err
	}
	return b, nil
}

// Validator_DepositCount returns a validators deposit count.
func (s *PublicValidatorAPI) Validator_DepositCount(ctx context.Context, blockNrOrHash *rpc.BlockNumberOrHash) (hexutil.Uint64, error) {
	bNrOrHash := rpc.BlockNumberOrHashWithHash(s.b.GetLastFinalizedBlock().Hash(), false)
	if blockNrOrHash != nil {
		bNrOrHash = *blockNrOrHash
	}
	stateDb, header, err := s.b.StateAndHeaderByNumberOrHash(ctx, bNrOrHash)
	if stateDb == nil || err != nil {
		return 0, err
	}

	validatorProcessor, _, err := s.b.GetVP(ctx, stateDb, header)
	if err != nil {
		return 0, err
	}

	count := validatorProcessor.getDepositCount()
	return hexutil.Uint64(count), stateDb.Error()
}

type ExitRequestArgs struct {
	PubKey         *common.BlsPubKey `json:"pubkey"`
	CreatorAddress *common.Address   `json:"creator_address"`
	ExitEpoch      *uint64           `json:"exit_epoch"`
}

func (s *PublicValidatorAPI) Validator_ExitData(_ context.Context, args ExitRequestArgs) (hexutil.Bytes, error) {
	if args.PubKey == nil {
		return nil, operation.ErrNoPubKey
	}
	if args.CreatorAddress == nil {
		return nil, operation.ErrNoCreatorAddress
	}

	var (
		op  operation.Operation
		err error
	)

	if op, err = operation.NewExitOperation(
		*args.PubKey,
		*args.CreatorAddress,
		args.ExitEpoch,
	); err != nil {
		return nil, err
	}

	b, err := operation.EncodeToBytes(op)
	if err != nil {
		log.Warn("Failed to encode validator exit operation", "err", err)
		return nil, err
	}

	return b, nil
}

type WithdrawalArgs struct {
	CreatorAddress *common.Address `json:"creator_address"`
	Amount         *hexutil.Big    `json:"amount"`
}

func (s *PublicValidatorAPI) Validator_WithdrawalData(args WithdrawalArgs) (hexutil.Bytes, error) {
	if args.CreatorAddress == nil {
		return nil, operation.ErrNoCreatorAddress
	}

	if args.Amount == nil {
		return nil, operation.ErrNoAmount
	}

	var (
		op  operation.Operation
		err error
	)

	if op, err = operation.NewWithdrawalOperation(
		*args.CreatorAddress,
		(*big.Int)(args.Amount),
	); err != nil {
		return nil, err
	}

	b, err := operation.EncodeToBytes(op)
	if err != nil {
		log.Warn("Failed to encode validator withdrawal operation", "err", err)
		return nil, err
	}

	return b, nil
}

func (s *PublicValidatorAPI) Validator_DepositAddress() hexutil.Bytes {
	return s.b.ChainConfig().ValidatorsStateAddress[:]
}

// GetValidatorsBySlot retrieves validators by provided slot.
func (s *PublicValidatorAPI) GetValidatorsBySlot(ctx context.Context, slot uint64) ([]common.Address, error) {
	if s.chain.GetSlotInfo() == nil {
		return nil, errors.New("no slot info")
	}

	creatorsPerSlot, err := s.chain.ValidatorStorage().GetCreatorsBySlot(s.chain, slot)
	if err != nil {
		return nil, err
	}
	return creatorsPerSlot, nil
}

// GetValidators retrieves creators by provided era.
func (s *PublicValidatorAPI) GetValidators(ctx context.Context, era *uint64) ([]common.Address, error) {
	slotInfo := s.chain.GetSlotInfo()
	if slotInfo == nil {
		return nil, errors.New("no slot info")
	}

	var startEpoch uint64
	if era == nil {
		startEpoch = s.chain.GetEraInfo().FromEpoch()
	} else {
		dbEra := rawdb.ReadEra(s.chain.Database(), *era)
		if dbEra == nil {
			return nil, errors.New("era not found")
		}
		startEpoch = dbEra.From
	}

	slot, err := slotInfo.SlotOfEpochStart(startEpoch)
	if err != nil {
		return nil, err
	}

	_, addresses := s.chain.ValidatorStorage().GetValidators(s.chain, slot, true, true, "GetValidators")
	return addresses, nil
}

// Validator_GetInfo retrieves validator info by provided address.
func (s *PublicValidatorAPI) Validator_GetInfo(ctx context.Context, address common.Address, blockNrOrHash *rpc.BlockNumberOrHash) (*valStore.Validator, error) {
	slotInfo := s.chain.GetSlotInfo()
	if slotInfo == nil {
		return nil, errors.New("no slot info")
	}

	if blockNrOrHash == nil {
		cpBlNr := rpc.CheckpointBlockNumber
		blockNrOrHash = &rpc.BlockNumberOrHash{
			BlockNumber: &cpBlNr,
		}
	}

	stateDb, _, err := s.b.StateAndHeaderByNumberOrHash(ctx, *blockNrOrHash)
	if stateDb == nil || err != nil {
		return nil, err
	}
	return s.chain.ValidatorStorage().GetValidator(stateDb, address)
}

// Validator_GetTransactionReceipt returns the transaction receipt of the validator op with parsed data.
func (s *PublicValidatorAPI) Validator_GetTransactionReceipt(ctx context.Context, hash common.Hash) (map[string]interface{}, error) {
	tx, blockHash, index, err := s.b.GetTransaction(ctx, hash)
	if err != nil {
		return nil, nil
	}
	blockNumber := s.b.GetBlockFinalizedNumber(blockHash)
	if blockNumber == nil || *blockNumber == 0 {
		return nil, nil
	}
	receipts, err := s.b.GetReceipts(ctx, blockHash)
	if err != nil {
		return nil, err
	}
	if len(receipts) <= int(index) {
		return nil, nil
	}
	receipt := receipts[index]

	// Derive the sender.
	signer := types.MakeSigner(s.b.ChainConfig())
	from, _ := types.Sender(signer, tx)

	fields := map[string]interface{}{
		"blockHash":         blockHash,
		"blockNumber":       hexutil.Uint64(*blockNumber),
		"transactionHash":   hash,
		"transactionIndex":  hexutil.Uint64(index),
		"from":              from,
		"to":                tx.To(),
		"gasUsed":           hexutil.Uint64(receipt.GasUsed),
		"cumulativeGasUsed": hexutil.Uint64(receipt.CumulativeGasUsed),
		"contractAddress":   nil,
		"logsBloom":         receipt.Bloom,
		"type":              hexutil.Uint(tx.Type()),
	}
	header, err := s.b.HeaderByHash(ctx, blockHash)
	if err != nil {
		return nil, err
	}
	gasPrice := new(big.Int).Add(header.BaseFee, tx.EffectiveGasTipValue(header.BaseFee))
	fields["effectiveGasPrice"] = hexutil.Uint64(gasPrice.Uint64())

	// Assign receipt status or post state.
	if len(receipt.PostState) > 0 {
		fields["root"] = hexutil.Bytes(receipt.PostState)
	} else {
		fields["status"] = hexutil.Uint(receipt.Status)
	}
	// If the ContractAddress is 20 0x0 bytes, assume it is not a contract creation
	if receipt.ContractAddress != (common.Address{}) {
		fields["contractAddress"] = receipt.ContractAddress
	}
	// add parsed logs
	if receipt.Logs == nil {
		fields["logs"] = [][]*types.Log{}
	} else {
		parsedLogs := make([]*types.ParsedLog, len(receipt.Logs))
		for i, log := range receipt.Logs {
			parsedLogs[i] = txlog.LogToParsedLog(log)
		}
		fields["logs"] = parsedLogs
	}
	return fields, nil
}
