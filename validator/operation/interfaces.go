package operation

import (
	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
)

// Deposit contains all attributes for validator deposit
// Methods for getting optional attributes also return boolean values which indicates if the attribute was set
type Deposit interface {
	Operation
	PubKey() common.BlsPubKey
	CreatorAddress() common.Address
	WithdrawalAddress() common.Address
	Signature() common.BlsSignature
	DepositDataRoot() common.Hash
}
