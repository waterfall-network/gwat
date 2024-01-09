package txlog

import (
	"math/big"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/validator/operation"
)

type ApplyData struct {
	OpCode        operation.Code `json:"opCode"`
	InitTxHash    common.Hash    `json:"initTxHash"`
	ActivateEpoch *uint64
	ExitEpoch     *uint64
	Amount        *big.Int
	OpAccount     *common.Address
	ShareData     []ShareRuleAppling
}

type UpdateBalanceRuleType uint8

const (
	NoRule UpdateBalanceRuleType = iota
	ProfitShare
	StakeShare
)

type ShareRuleAppling struct {
	Address  common.Address
	RuleType UpdateBalanceRuleType
	IsTrial  bool
	Amount   *big.Int
}
