// Copyright 2024   Blue Wave Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package txlog

import (
	"testing"

	"github.com/aws/smithy-go/ptr"
	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/types"
	"gitlab.waterfall.network/waterfall/protocol/gwat/tests/testutils"
)

func TestLogToParsedLog_Error(t *testing.T) {
	txLog := &types.Log{
		Address:     common.Address{0x11},
		Topics:      []common.Hash{EvtDepositLogSignature, common.Hash{0x44}},
		Data:        common.Hex2Bytes("931f74533c800ebb6"),
		BlockNumber: 100,
		TxHash:      common.Hash{0x22},
		TxIndex:     55,
		BlockHash:   common.Hash{0x33},
		Index:       261,
		Removed:     false,
	}
	expParsed := txLog.ToParsedLog()

	// deposit failed
	expParsed.ParsedTopics = []string{"deposit", common.Hash{0x44}.Hex()}
	expParsed.ParsedData = parsedDataFailed{
		Error: "log data parcing error='bad data length' topic=deposit",
	}
	gotParsed := LogToParsedLog(txLog)
	testutils.AssertEqual(t, expParsed, gotParsed)

	// withdrawal failed
	txLog.Topics[0] = EvtWithdrawalLogSignature
	expParsed.ParsedTopics = []string{"withdrawal", common.Hash{0x44}.Hex()}
	expParsed.ParsedData = parsedDataFailed{
		Error: "log data parcing error='bad data length' topic=withdrawal",
	}
	gotParsed = LogToParsedLog(txLog)
	testutils.AssertEqual(t, expParsed, gotParsed)

	// exit failed
	txLog.Topics[0] = EvtExitReqLogSignature
	expParsed.ParsedTopics = []string{"exit", common.Hash{0x44}.Hex()}
	expParsed.ParsedData = parsedDataFailed{
		Error: "log data parcing error='bad data length' topic=exit",
	}
	gotParsed = LogToParsedLog(txLog)
	testutils.AssertEqual(t, expParsed, gotParsed)

	// activate failed
	txLog.Topics[0] = EvtActivateLogSignature
	expParsed.ParsedTopics = []string{"activate", common.Hash{0x44}.Hex()}
	expParsed.ParsedData = parsedDataFailed{
		Error: "log data parcing error='rlp: value size exceeds available input length' topic=activate",
	}
	gotParsed = LogToParsedLog(txLog)
	testutils.AssertEqual(t, expParsed, gotParsed)

	// deactivate failed
	txLog.Topics[0] = EvtDeactivateLogSignature
	expParsed.ParsedTopics = []string{"deactivate", common.Hash{0x44}.Hex()}
	expParsed.ParsedData = parsedDataFailed{
		Error: "log data parcing error='rlp: value size exceeds available input length' topic=deactivate",
	}
	gotParsed = LogToParsedLog(txLog)
	testutils.AssertEqual(t, expParsed, gotParsed)

	// update balance failed
	txLog.Topics[0] = EvtUpdateBalanceLogSignature
	expParsed.ParsedTopics = []string{"update-balance", common.Hash{0x44}.Hex()}
	expParsed.ParsedData = parsedDataFailed{
		Error: "log data parcing error='rlp: value size exceeds available input length' topic=update-balance",
	}
	gotParsed = LogToParsedLog(txLog)
	testutils.AssertEqual(t, expParsed, gotParsed)

	// update balance failed
	txLog.Topics[0] = EvtDelegatingStakeSignature
	expParsed.ParsedTopics = []string{"delegating-stake", common.Hash{0x44}.Hex()}
	expParsed.ParsedData = parsedDataFailed{
		Error: "log data parcing error='rlp: value size exceeds available input length' topic=delegating-stake",
	}
	gotParsed = LogToParsedLog(txLog)
	testutils.AssertEqual(t, expParsed, gotParsed)
}

func TestLogToParsedLog_Deposit(t *testing.T) {
	txLog := &types.Log{
		Address: common.Address{0x11},
		Topics:  []common.Hash{EvtDepositLogSignature, common.Hash{0x44}},
		Data: common.Hex2Bytes("931f74533c800ebb6d4b4330a9f7ad609314303c01ca7cd235635fe30fcaa33cdcc2c09e9" +
			"a07d22d7126e0a078657cbe6e9e76fa278190cfb2404e5923d3ccd7e8f6c777a7e558cc6efa1c41270ef4aa227b3dd6b4a3951e00" +
			"1082e3d1030000a4798654cec11445dcb58eac0fc21a5f668ad7709c0bfdd0265793710f781d7bdab9469936cc528f77eab5ee78e" +
			"b9b1807ec450a146eceeedb0687deea17d56972800abc1f4c65b6026d23a264443b71efc1f040495e6a7499cac2f944a7cf280000000000000000"),
		BlockNumber: 100,
		TxHash:      common.Hash{0x22},
		TxIndex:     55,
		BlockHash:   common.Hash{0x33},
		Index:       261,
		Removed:     false,
	}

	expParsed := txLog.ToParsedLog()
	expParsed.ParsedTopics = []string{"deposit", common.Hash{0x44}.Hex()}
	expParsed.ParsedData = parsedDeposit{
		CreatorAddr:   "0x6E9E76FA278190cFb2404E5923D3cCd7E8F6c777",
		DepositAmount: 4200000000000,
		DepositIndex:  0,
		PublicKey:     "0x931f74533c800ebb6d4b4330a9f7ad609314303c01ca7cd235635fe30fcaa33cdcc2c09e9a07d22d7126e0a078657cbe",
		Signature: "0xa4798654cec11445dcb58eac0fc21a5f668ad7709c0bfdd0265793710f781d7bdab9469936cc528f77eab5ee78eb9b18" +
			"07ec450a146eceeedb0687deea17d56972800abc1f4c65b6026d23a264443b71efc1f040495e6a7499cac2f944a7cf28",
		WithdrawalAddr: "0xa7e558Cc6efA1c41270eF4Aa227b3dd6B4a3951E",
	}

	gotParsed := LogToParsedLog(txLog)
	testutils.AssertEqual(t, expParsed, gotParsed)
}

func TestLogToParsedLog_Withdrawal(t *testing.T) {
	txLog := &types.Log{
		Address: common.Address{0x11},
		Topics:  []common.Hash{EvtWithdrawalLogSignature, common.Hash{0x44}},
		Data: common.Hex2Bytes("931f74533c800ebb6d4b4330a9f7ad609314303c01ca7cd235635fe30fcaa33cdcc2c09e9a07d22d712" +
			"6e0a078657cbe6e9e76fa278190cfb2404e5923d3ccd7e8f6c777050100000000000000e1f50500000000"),
		BlockNumber: 100,
		TxHash:      common.Hash{0x22},
		TxIndex:     55,
		BlockHash:   common.Hash{0x33},
		Index:       261,
		Removed:     false,
	}

	expParsed := txLog.ToParsedLog()
	expParsed.ParsedTopics = []string{"withdrawal", common.Hash{0x44}.Hex()}
	expParsed.ParsedData = parsedWithdrawal{
		PublicKey:      "0x931f74533c800ebb6d4b4330a9f7ad609314303c01ca7cd235635fe30fcaa33cdcc2c09e9a07d22d7126e0a078657cbe",
		CreatorAddr:    "0x6E9E76FA278190cFb2404E5923D3cCd7E8F6c777",
		ValidatorIndex: 261,
		Amount:         100000000,
	}

	gotParsed := LogToParsedLog(txLog)
	testutils.AssertEqual(t, expParsed, gotParsed)
}

func TestLogToParsedLog_Exit(t *testing.T) {
	txLog := &types.Log{
		Address: common.Address{0x11},
		Topics:  []common.Hash{EvtExitReqLogSignature, common.Hash{0x44}},
		Data: common.Hex2Bytes("931f74533c800ebb6d4b4330a9f7ad609314303c01ca7cd235635fe30fcaa33cdcc2c09e9a0" +
			"7d22d7126e0a078657cbe6e9e76fa278190cfb2404e5923d3ccd7e8f6c77705010000000000009400000000000000"),
		BlockNumber: 100,
		TxHash:      common.Hash{0x22},
		TxIndex:     55,
		BlockHash:   common.Hash{0x33},
		Index:       261,
		Removed:     false,
	}

	expParsed := txLog.ToParsedLog()
	expParsed.ParsedTopics = []string{"exit", common.Hash{0x44}.Hex()}
	expParsed.ParsedData = parsedExit{
		CreatorAddr:    "0x6E9E76FA278190cFb2404E5923D3cCd7E8F6c777",
		ExitAfterEpoch: ptr.Uint64(148),
		PublicKey:      "0x931f74533c800ebb6d4b4330a9f7ad609314303c01ca7cd235635fe30fcaa33cdcc2c09e9a07d22d7126e0a078657cbe",
		ValidatorIndex: 261,
	}

	gotParsed := LogToParsedLog(txLog)
	testutils.AssertEqual(t, expParsed, gotParsed)
}

func TestLogToParsedLog_Activate(t *testing.T) {
	txLog := &types.Log{
		Address: common.Address{0x11},
		Topics:  []common.Hash{EvtActivateLogSignature, common.Hash{0x44}},
		Data: common.Hex2Bytes("f83aa0c4508a97bfd88b9cf48aa4485ee27f2398ce7d6f03630ad7c19bac678cb77" +
			"ecd946e9e76fa278190cfb2404e5923d3ccd7e8f6c77727820105"),
		BlockNumber: 100,
		TxHash:      common.Hash{0x22},
		TxIndex:     55,
		BlockHash:   common.Hash{0x33},
		Index:       261,
		Removed:     false,
	}

	expParsed := txLog.ToParsedLog()
	expParsed.ParsedTopics = []string{"activate", common.Hash{0x44}.Hex()}
	expParsed.ParsedData = parsedDeActivate{
		CreatorAddr:    "0x6E9E76FA278190cFb2404E5923D3cCd7E8F6c777",
		ProcEpoch:      39,
		InitTxHash:     "0xc4508a97bfd88b9cf48aa4485ee27f2398ce7d6f03630ad7c19bac678cb77ecd",
		ValidatorIndex: 261,
	}

	gotParsed := LogToParsedLog(txLog)
	testutils.AssertEqual(t, expParsed, gotParsed)
}

func TestLogToParsedLog_Deactivate(t *testing.T) {
	txLog := &types.Log{
		Address: common.Address{0x11},
		Topics:  []common.Hash{EvtDeactivateLogSignature, common.Hash{0x44}},
		Data: common.Hex2Bytes("f83aa0c4508a97bfd88b9cf48aa4485ee27f2398ce7d6f03630ad7c19bac678cb77" +
			"ecd946e9e76fa278190cfb2404e5923d3ccd7e8f6c77727820105"),
		BlockNumber: 100,
		TxHash:      common.Hash{0x22},
		TxIndex:     55,
		BlockHash:   common.Hash{0x33},
		Index:       261,
		Removed:     false,
	}

	expParsed := txLog.ToParsedLog()
	expParsed.ParsedTopics = []string{"deactivate", common.Hash{0x44}.Hex()}
	expParsed.ParsedData = parsedDeActivate{
		CreatorAddr:    "0x6E9E76FA278190cFb2404E5923D3cCd7E8F6c777",
		ProcEpoch:      39,
		InitTxHash:     "0xc4508a97bfd88b9cf48aa4485ee27f2398ce7d6f03630ad7c19bac678cb77ecd",
		ValidatorIndex: 261,
	}

	gotParsed := LogToParsedLog(txLog)
	testutils.AssertEqual(t, expParsed, gotParsed)
}

func TestLogToParsedLog_UpdateBalance(t *testing.T) {
	txLog := &types.Log{
		Address: common.Address{0x11},
		Topics:  []common.Hash{EvtUpdateBalanceLogSignature, common.Hash{0x44}},
		Data: common.Hex2Bytes("f842a0ff69fdb3f260dd66f5e2795719fe75341f3f9cb468f2d3832936789178cd984f946" +
			"e9e76fa278190cfb2404e5923d3ccd7e8f6c77781ae89e3ace33bf2ba5e3000"),
		BlockNumber: 100,
		TxHash:      common.Hash{0x22},
		TxIndex:     55,
		BlockHash:   common.Hash{0x33},
		Index:       261,
		Removed:     false,
	}

	expParsed := txLog.ToParsedLog()
	expParsed.ParsedTopics = []string{"update-balance", common.Hash{0x44}.Hex()}
	expParsed.ParsedData = parsedUpdateBalance{
		Amount:      "4199868771640000000000",
		CreatorAddr: "0x6E9E76FA278190cFb2404E5923D3cCd7E8F6c777",
		ProcEpoch:   174,
		InitTxHash:  "0xff69fdb3f260dd66f5e2795719fe75341f3f9cb468f2d3832936789178cd984f",
	}

	gotParsed := LogToParsedLog(txLog)
	testutils.AssertEqual(t, expParsed, gotParsed)
}

func TestLogToParsedLog_DelegatingStake(t *testing.T) {
	txLog := &types.Log{
		Address: common.Address{0x11},
		Topics:  []common.Hash{EvtDelegatingStakeSignature, common.Hash{0x44}},
		Data: common.Hex2Bytes("f8aae1941111111111111111111111111111111111111111018089056b98bf0708d63800e1" +
			"9422222222222222222222222222222222222222220180892085947a2a35055000e1943333333333333333333333333333333333" +
			"3333330180891042ca3d151a82a800e194444444444444444444444444444444444444444402808968155a43676e000000e19477" +
			"777777777777777777777777777777777777770280894563918244f4000000"),
		BlockNumber: 100,
		TxHash:      common.Hash{0x22},
		TxIndex:     55,
		BlockHash:   common.Hash{0x33},
		Index:       261,
		Removed:     false,
	}

	expParsed := txLog.ToParsedLog()
	expParsed.ParsedTopics = []string{"delegating-stake", common.Hash{0x44}.Hex()}
	expParsed.ParsedData = []parsedDelegatingItm{
		{
			Address:  "0x1111111111111111111111111111111111111111",
			Amount:   "99986877164000000000",
			IsTrial:  false,
			RuleType: "profit",
		}, {
			Address:  "0x2222222222222222222222222222222222222222",
			Amount:   "599921262984000000000",
			IsTrial:  false,
			RuleType: "profit",
		}, {
			Address:  "0x3333333333333333333333333333333333333333",
			Amount:   "299960631492000000000",
			IsTrial:  false,
			RuleType: "profit",
		}, {
			Address:  "0x4444444444444444444444444444444444444444",
			Amount:   "1920000000000000000000",
			IsTrial:  false,
			RuleType: "stake",
		}, {
			Address:  "0x7777777777777777777777777777777777777777",
			Amount:   "1280000000000000000000",
			IsTrial:  false,
			RuleType: "stake",
		},
	}

	gotParsed := LogToParsedLog(txLog)
	testutils.AssertEqual(t, expParsed, gotParsed)
}
