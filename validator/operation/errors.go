package operation

import (
	"errors"
)

var (
	ErrBadDataLen = errors.New("bad data length")

	ErrNoPubKey            = errors.New("pubkey is required")
	ErrNoInitTxHash        = errors.New("initTxHash is required")
	ErrNoCreatorAddress    = errors.New("creator_address is required")
	ErrNoWithdrawalAddress = errors.New("withdrawal_address is required")
	ErrNoSignature         = errors.New("signature is required")
	ErrNoAmount            = errors.New("amount is required")
	ErrNoBalance           = errors.New("balance is required")
	ErrNoRules             = errors.New("rules is required")
	ErrRawDataShort        = errors.New("binary data for validator operation is short")

	ErrPrefixNotValid = errors.New("not valid value for prefix")
	ErrOpNotValid     = errors.New("op code is not valid")
	ErrOpBadVersion   = errors.New("op version is not valid")

	ErrNoExitRoles         = errors.New("no exit roles")
	ErrNoWithdrawalRoles   = errors.New("no withdrawal roles")
	ErrBadProfitShare      = errors.New("profit share totally must be 100%")
	ErrBadStakeShare       = errors.New("stake share totally must be 100%")
	ErrDelegateForkRequire = errors.New("can not process transaction before fork of delegating stake")
)
