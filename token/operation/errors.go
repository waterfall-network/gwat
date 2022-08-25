package operation

import (
	"errors"
)

var (
	ErrNoTokenSupply    = errors.New("token supply is required")
	ErrNoName           = errors.New("token name is required")
	ErrNoSymbol         = errors.New("token symbol is required")
	ErrNoBaseURI        = errors.New("token baseURI is required")
	ErrNoAddress        = errors.New("token address is required")
	ErrNoOwner          = errors.New("token owner address is required")
	ErrNoValue          = errors.New("value is required")
	ErrNegativeCost     = errors.New("cost is negative")
	ErrNoTo             = errors.New("to address is required")
	ErrNoFrom           = errors.New("from address is required")
	ErrNoSpender        = errors.New("spender address is required")
	ErrNoOperator       = errors.New("operator address is required")
	ErrNoTokenId        = errors.New("token id is required")
	ErrNoIndex          = errors.New("token index is required")
	ErrStandardNotValid = errors.New("not valid value for token standard")
	ErrPrefixNotValid   = errors.New("not valid value for prefix")
	ErrRawDataShort     = errors.New("binary data for token operation is short")
	ErrOpNotValid       = errors.New("not valid op code for token operation")
)
