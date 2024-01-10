package txlog

import (
	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/types"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/vm"
	"gitlab.waterfall.network/waterfall/protocol/gwat/crypto"
)

var (
	//Events signatures.
	EvtDepositLogSignature    = crypto.Keccak256Hash([]byte("DepositLog"))
	EvtExitReqLogSignature    = crypto.Keccak256Hash([]byte("ExitRequestLog"))
	EvtWithdrawalLogSignature = crypto.Keccak256Hash([]byte("WithdrawaRequestLog"))
	//validator sync op
	EvtActivateLogSignature      = crypto.Keccak256Hash([]byte("ActivateLog"))
	EvtUpdateBalanceLogSignature = crypto.Keccak256Hash([]byte("UpdateBalanceLog"))
)

type logEntry struct {
	// name string
	//entryType string
	indexed bool
	data    []byte
}

type EventEmmiter struct {
	state vm.StateDB
}

func NewEventEmmiter(state vm.StateDB) *EventEmmiter {
	return &EventEmmiter{state: state}
}

func (e *EventEmmiter) Deposit(evtAddr common.Address, data []byte) {
	e.addLog(
		evtAddr,
		EvtDepositLogSignature,
		data,
	)
}

func (e *EventEmmiter) ExitRequest(evtAddr common.Address, data []byte) {
	e.addLog(evtAddr, EvtExitReqLogSignature, data)
}

func (e *EventEmmiter) WithdrawalRequest(evtAddr common.Address, data []byte) {
	e.addLog(evtAddr, EvtWithdrawalLogSignature, data)
}

func (e *EventEmmiter) addLog(targetAddr common.Address, signature common.Hash, data []byte, logsEntries ...logEntry) {
	//var data []byte
	topics := []common.Hash{signature}

	for _, entry := range logsEntries {
		if entry.indexed {
			topics = append(topics, common.BytesToHash(entry.data))
		} else {
			data = append(data, entry.data...)
		}
	}

	e.state.AddLog(&types.Log{
		Address: targetAddr,
		Topics:  topics,
		Data:    data,
	})
}
