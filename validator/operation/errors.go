package operation

import (
	"errors"
)

var (
	ErrBadDataLen = errors.New("bad data length")

	ErrNoPubKey            = errors.New("pubkey is required")
	ErrNoCreatorAddress    = errors.New("creator_address is required")
	ErrNoWithdrawalAddress = errors.New("withdrawal_address is required")
	ErrNoSignature         = errors.New("signature is required")
	ErrNoAmount            = errors.New("amount is required")
	ErrNoValue             = errors.New("value is required")
	ErrNegativeCost        = errors.New("cost is negative")
	ErrNoTo                = errors.New("to address is required")
	ErrNoFrom              = errors.New("from address is required")
	ErrRawDataShort        = errors.New("binary data for validator operation is short")

	ErrNoOperator     = errors.New("operator address is required")
	ErrPrefixNotValid = errors.New("not valid value for prefix")
	ErrOpNotValid     = errors.New("op code is not valid")
)
