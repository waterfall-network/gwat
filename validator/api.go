package validator

import (
	"context"
	"time"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/common/hexutil"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/state"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/types"
	"gitlab.waterfall.network/waterfall/protocol/gwat/log"
	"gitlab.waterfall.network/waterfall/protocol/gwat/rpc"
	"gitlab.waterfall.network/waterfall/protocol/gwat/validator/operation"
)

type Backend interface {
	StateAndHeaderByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*state.StateDB, *types.Header, error)
	RPCEVMTimeout() time.Duration // global timeout for eth_call over rpc: DoS protection
	GetVP(ctx context.Context, state *state.StateDB, header *types.Header) (*Processor, func() error, error)
}

// PublicValidatorAPI provides an API to access validator functions.
type PublicValidatorAPI struct {
	b Backend
}

// NewPublicValidatorAPI creates a new validator API.
func NewPublicValidatorAPI(b Backend) *PublicValidatorAPI {
	return &PublicValidatorAPI{b}
}

type DepositArgs struct {
	PubKey            *common.BlsPubKey    `json:"pubkey"`             // validator public key
	CreatorAddress    *common.Address      `json:"creator_address"`    // attached creator account
	WithdrawalAddress *common.Address      `json:"withdrawal_address"` // attached withdrawal credentials
	Signature         *common.BlsSignature `json:"signature"`
	DepositDataRoot   *common.Hash         `json:"deposit_data_root"`
}

// GetAPIs provides api access
func GetAPIs(apiBackend Backend) []rpc.API {
	return []rpc.API{
		{
			Namespace: "validator",
			Version:   "1.0",
			Service:   NewPublicValidatorAPI(apiBackend),
			Public:    true,
		},
	}
}

// DepositData creates a validators deposit data for deposit tx.
func (s *PublicValidatorAPI) DepositData(_ context.Context, args DepositArgs) (hexutil.Bytes, error) {
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
	if args.DepositDataRoot == nil {
		return nil, operation.ErrNoDepositDataRoot
	}

	var (
		op  operation.Operation
		err error
	)

	if op, err = operation.NewDepositOperation(*args.PubKey, *args.CreatorAddress, *args.WithdrawalAddress, *args.Signature, *args.DepositDataRoot); err != nil {
		return nil, err
	}

	b, err := operation.EncodeToBytes(op)
	if err != nil {
		log.Warn("Failed to encode validator deposit operation", "err", err)
		return nil, err
	}
	return b, nil
}

func (s *PublicValidatorAPI) newTokenProcessor(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (tp *Processor, cancel context.CancelFunc, tpError func() error, err error) {
	// Setup context so it may be cancelled the call has completed
	// or, in case of unmetered gas, setup a context with a timeout.
	timeout := s.b.RPCEVMTimeout()
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, timeout)
	} else {
		ctx, cancel = context.WithCancel(ctx)
	}

	state, header, err := s.b.StateAndHeaderByNumberOrHash(ctx, blockNrOrHash)
	if state == nil || err != nil {
		return nil, cancel, nil, err
	}

	tp, tpError, err = s.b.GetVP(ctx, state, header)
	if err != nil {
		return nil, cancel, nil, err
	}

	return
}
