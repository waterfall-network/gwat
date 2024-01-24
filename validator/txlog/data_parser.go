package txlog

import (
	"fmt"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/types"
)

var topicsNameMap = map[common.Hash]string{
	EvtDepositLogSignature:       "deposit",
	EvtExitReqLogSignature:       "exit",
	EvtWithdrawalLogSignature:    "withdrawal",
	EvtActivateLogSignature:      "activate",
	EvtDeactivateLogSignature:    "deactivate",
	EvtUpdateBalanceLogSignature: "update-balance",
	EvtDelegatingStakeSignature:  "delegating-stake",
	types.EvtErrorLogSignature:   "error",
}

type parsedDataFailed struct {
	Error string `json:"error"`
}

type parsedDeposit struct {
	PublicKey      string `json:"publicKey"`
	CreatorAddr    string `json:"creatorAddr"`
	WithdrawalAddr string `json:"withdrawalAddr"`
	DepositAmount  uint64 `json:"depositAmount"`
	Signature      string `json:"signature"`
	DepositIndex   uint64 `json:"depositIndex"`
}

type parsedExit struct {
	PublicKey      string  `json:"publicKey"`
	CreatorAddr    string  `json:"creatorAddr"`
	ValidatorIndex uint64  `json:"validatorIndex"`
	ExitAfterEpoch *uint64 `json:"exitAfterEpoch"`
}

type parsedWithdrawal struct {
	PublicKey      string `json:"publicKey"`
	CreatorAddr    string `json:"creatorAddr"`
	ValidatorIndex uint64 `json:"validatorIndex"`
	Amount         uint64 `json:"amount"`
}

type parsedDeActivate struct {
	InitTxHash     string `json:"publicKey"`
	CreatorAddr    string `json:"creatorAddr"`
	ProcEpoch      uint64 `json:"procEpoch"`
	ValidatorIndex uint64 `json:"validatorIndex"`
}

type parsedUpdateBalance struct {
	InitTxHash     string `json:"publicKey"`
	CreatorAddr    string `json:"creatorAddr"`
	ProcEpoch      uint64 `json:"procEpoch"`
	ValidatorIndex uint64 `json:"validatorIndex"`
	Amount         string `json:"amount"`
}

type parsedSharing struct {
	Address  string `json:"address"`
	RuleType string `json:"ruleType"`
	IsTrial  bool   `json:"isTrial"`
	Amount   string `json:"amount"`
}

func getTopicName(topic common.Hash) string {
	if _, ok := topicsNameMap[topic]; ok {
		return topicsNameMap[topic]
	}
	return topic.Hex()
}

func LogToParsedLog(log *types.Log) *types.ParsedLog {
	parsed := log.ToParsedLog()

	// set parsed topics
	var isErrorLog bool
	parsed.ParsedTopics = make([]string, len(log.Topics))
	for i, topic := range log.Topics {
		if topic == types.EvtErrorLogSignature {
			isErrorLog = true
		}
		parsed.ParsedTopics[i] = getTopicName(topic)
	}
	if isErrorLog {
		parsed.ParsedData = string(parsed.Data)
	}
	//set parsed data
	topicOp := log.Topics[0]
	switch topicOp {
	case EvtDepositLogSignature:
		pkey, creator, withdrawalAddr, depositAmount, signature, depositIndex, err := UnpackDepositLogData(log.Data)
		if err != nil {
			parsed.ParsedData = parsedDataFailed{
				Error: fmt.Sprintf("log data parcing error='%s' topic=%s", err.Error(), getTopicName(topicOp)),
			}
		}
		parsed.ParsedData = parsedDeposit{
			PublicKey:      pkey.Hex(),
			CreatorAddr:    creator.Hex(),
			WithdrawalAddr: withdrawalAddr.Hex(),
			DepositAmount:  depositAmount,
			Signature:      signature.Hex(),
			DepositIndex:   depositIndex,
		}
	case EvtExitReqLogSignature:
		pKey, creator, valIndex, exitAfter, err := UnpackExitRequestLogData(log.Data)
		if err != nil {
			parsed.ParsedData = parsedDataFailed{
				Error: fmt.Sprintf("log data parcing error='%s' topic=%s", err.Error(), getTopicName(topicOp)),
			}
		}

		parsed.ParsedData = parsedExit{
			PublicKey:      pKey.Hex(),
			CreatorAddr:    creator.Hex(),
			ValidatorIndex: valIndex,
			ExitAfterEpoch: exitAfter,
		}
	case EvtWithdrawalLogSignature:
		pKey, creator, valIndex, gwAmt, err := UnpackWithdrawalLogData(log.Data)
		if err != nil {
			parsed.ParsedData = parsedDataFailed{
				Error: fmt.Sprintf("log data parcing error='%s' topic=%s", err.Error(), getTopicName(topicOp)),
			}
		}
		parsed.ParsedData = parsedWithdrawal{
			PublicKey:      pKey.Hex(),
			CreatorAddr:    creator.Hex(),
			ValidatorIndex: valIndex,
			Amount:         gwAmt,
		}
	case EvtActivateLogSignature, EvtDeactivateLogSignature:
		initTx, creator, proc, vix, err := UnpackActivateLogData(log.Data)
		if err != nil {
			parsed.ParsedData = parsedDataFailed{
				Error: fmt.Sprintf("log data parcing error='%s' topic=%s", err.Error(), getTopicName(topicOp)),
			}
		}
		parsed.ParsedData = parsedDeActivate{
			InitTxHash:     initTx.Hex(),
			CreatorAddr:    creator.Hex(),
			ProcEpoch:      proc,
			ValidatorIndex: vix,
		}
	case EvtUpdateBalanceLogSignature:
		initTx, creator, proc, amt, err := UnpackUpdateBalanceLogData(log.Data)
		if err != nil {
			parsed.ParsedData = parsedDataFailed{
				Error: fmt.Sprintf("log data parcing error='%s' topic=%s", err.Error(), getTopicName(topicOp)),
			}
		}
		parsed.ParsedData = parsedUpdateBalance{
			InitTxHash:  initTx.Hex(),
			CreatorAddr: creator.Hex(),
			ProcEpoch:   proc,
			Amount:      amt.String(),
		}
	case EvtDelegatingStakeSignature:
		amtSharing, err := UnpackDelegatingStakeLogData(log.Data)
		if err != nil {
			parsed.ParsedData = parsedDataFailed{
				Error: fmt.Sprintf("log data parcing error='%s' topic=%s", err.Error(), getTopicName(topicOp)),
			}
		}
		if amtSharing == nil {
			return nil
		}
		delegatingData := make([]parsedSharing, len(*amtSharing))
		for i, v := range *amtSharing {
			var shareType string
			switch v.RuleType {
			case StakeShare:
				shareType = "stake"
			case ProfitShare:
				shareType = "profit"
			default:
				shareType = fmt.Sprintf("unknown type (%d)", v.RuleType)
			}
			delegatingData[i] = parsedSharing{
				Address:  v.Address.Hex(),
				RuleType: shareType,
				IsTrial:  v.IsTrial,
				Amount:   v.Amount.String(),
			}
		}
		parsed.ParsedData = delegatingData
	default:
		parsed.ParsedData = parsedDataFailed{
			Error: fmt.Sprintf("log data parcing error='unknown operation of topick=%#x'", log.Topics[0]),
		}
	}
	return parsed
}
