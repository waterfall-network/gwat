package operation

import (
	"math/big"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/types"
)

// Deposit contains all attributes for validator deposit.
type Deposit interface {
	Operation
	PubKey() common.BlsPubKey
	CreatorAddress() common.Address
	WithdrawalAddress() common.Address
	Signature() common.BlsSignature
	DelegatingStake() *DelegatingStakeData
}

// ValidatorSync contains all attributes for validator sync op.
type ValidatorSync interface {
	Operation
	OpType() types.ValidatorSyncOp
	ProcEpoch() uint64
	Index() uint64
	Creator() common.Address
	InitTxHash() common.Hash
	Amount() *big.Int
	WithdrawalAddress() *common.Address
}

type Exit interface {
	Operation
	PubKey() common.BlsPubKey
	CreatorAddress() common.Address
	ExitAfterEpoch() *uint64
	SetExitAfterEpoch(*uint64)
}

type Withdrawal interface {
	Operation
	CreatorAddress() common.Address
	Amount() *big.Int
}
