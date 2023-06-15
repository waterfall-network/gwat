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
)

type Backend interface {
	StateAndHeaderByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*state.StateDB, *types.Header, error)
	RPCEVMTimeout() time.Duration // global timeout for eth_call over rpc: DoS protection
	GetVP(ctx context.Context, state *state.StateDB, header *types.Header) (*Processor, func() error, error)
	GetLastFinalizedBlock() *types.Block
	ChainConfig() *params.ChainConfig
}
type Blockchain interface {
	ValidatorStorage() valStore.Storage
	StateAt(root common.Hash) (*state.StateDB, error)
	GetBlock(hash common.Hash) *types.Block
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

type DepositArgs struct {
	PubKey            *common.BlsPubKey    `json:"pubkey"`             // validator public key
	CreatorAddress    *common.Address      `json:"creator_address"`    // attached creator account
	WithdrawalAddress *common.Address      `json:"withdrawal_address"` // attached withdrawal credentials
	Signature         *common.BlsSignature `json:"signature"`
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

// DepositData creates a validators deposit data for deposit tx.
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
		op  operation.Operation
		err error
	)

	if op, err = operation.NewDepositOperation(*args.PubKey, *args.CreatorAddress, *args.WithdrawalAddress, *args.Signature); err != nil {
		return nil, err
	}

	b, err := operation.EncodeToBytes(op)
	if err != nil {
		log.Warn("Failed to encode validator deposit operation", "err", err)
		return nil, err
	}
	return b, nil
}

// DepositCount returns a validators deposit count.
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
func (s *PublicValidatorAPI) Validator_GetInfo(ctx context.Context, address common.Address) (*valStore.Validator, error) {
	slotInfo := s.chain.GetSlotInfo()
	if slotInfo == nil {
		return nil, errors.New("no slot info")
	}

	stateDb, _ := s.chain.StateAt(s.chain.GetEraInfo().GetEra().Root)

	return s.chain.ValidatorStorage().GetValidator(stateDb, address)
}
