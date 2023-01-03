// Code generated by github.com/fjl/gencodec. DO NOT EDIT.

package types

import (
	"encoding/json"
	"errors"
	"math/big"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/common/hexutil"
)

var _ = (*headerMarshaling)(nil)

// MarshalJSON marshals as JSON.
func (h Header) MarshalJSON() ([]byte, error) {
	type Header struct {
		ParentHashes  []common.Hash    `json:"parentHashes"     gencodec:"required"`
		Slot          hexutil.Uint64   `json:"slot"             gencodec:"required"`
		Height        hexutil.Uint64   `json:"height"           gencodec:"required"`
		Coinbase      common.Address   `json:"miner"            gencodec:"required"`
		TxHash        common.Hash      `json:"transactionsRoot" gencodec:"required"`
		GasLimit      hexutil.Uint64   `json:"gasLimit"         gencodec:"required"`
		Time          hexutil.Uint64   `json:"timestamp"        gencodec:"required"`
		Extra         hexutil.Bytes    `json:"extraData"        gencodec:"required"`
		LFHash        common.Hash      `json:"lfHash"           gencodec:"required"`
		LFNumber      uint64           `json:"lfNumber"         gencodec:"required"`
		LFBaseFee     *hexutil.Big     `json:"lfBaseFeePerGas"  gencodec:"required"`
		LFRoot        common.Hash      `json:"lfStateRoot"      gencodec:"required"`
		LFReceiptHash common.Hash      `json:"lfReceiptsRoot"   gencodec:"required"`
		LFGasUsed     hexutil.Uint64   `json:"lfGasUsed"        gencodec:"required"`
		LFBloom       Bloom            `json:"lfLogsBloom"      gencodec:"required"`
		BaseFee       *hexutil.Big     `json:"baseFeePerGas" rlp:"optional"`
		Number        *hexutil.Uint64   `json:"number"        rlp:"optional"`
		Root          common.Hash      `json:"stateRoot"     rlp:"optional"`
		ReceiptHash   common.Hash      `json:"receiptsRoot"  rlp:"optional"`
		GasUsed       hexutil.Uint64   `json:"gasUsed"       rlp:"optional"`
		Bloom         Bloom            `json:"logsBloom"     rlp:"optional"`
		Hash          common.Hash      `json:"hash"`
	}
	var enc Header
	enc.ParentHashes = h.ParentHashes
	enc.Slot = hexutil.Uint64(h.Slot)
	enc.Height = hexutil.Uint64(h.Height)
	enc.Coinbase = h.Coinbase
	enc.TxHash = h.TxHash
	enc.GasLimit = hexutil.Uint64(h.GasLimit)
	enc.Time = hexutil.Uint64(h.Time)
	enc.Extra = h.Extra
	enc.LFHash = h.LFHash
	enc.LFNumber = h.LFNumber
	enc.LFBaseFee = (*hexutil.Big)(h.LFBaseFee)
	enc.LFRoot = h.LFRoot
	enc.LFReceiptHash = h.LFReceiptHash
	enc.LFGasUsed = hexutil.Uint64(h.LFGasUsed)
	enc.LFBloom = h.LFBloom
	enc.BaseFee = (*hexutil.Big)(h.BaseFee)
	if h.Number != nil {
		nr := hexutil.Uint64(*h.Number)
		enc.Number = &nr
	}
	enc.Root = h.Root
	enc.ReceiptHash = h.ReceiptHash
	enc.GasUsed = hexutil.Uint64(h.GasUsed)
	enc.Bloom = h.Bloom
	enc.Hash = h.Hash()
	return json.Marshal(&enc)
}

// UnmarshalJSON unmarshals from JSON.
func (h *Header) UnmarshalJSON(input []byte) error {
	type Header struct {
		ParentHashes  *[]common.Hash     `json:"parentHashes"     gencodec:"required"`
		Slot          *hexutil.Uint64    `json:"slot"             gencodec:"required"`
		Height        *hexutil.Uint64    `json:"height"           gencodec:"required"`
		Coinbase      *common.Address   `json:"miner"            gencodec:"required"`
		TxHash        *common.Hash      `json:"transactionsRoot" gencodec:"required"`
		GasLimit      *hexutil.Uint64   `json:"gasLimit"         gencodec:"required"`
		Time          *hexutil.Uint64   `json:"timestamp"        gencodec:"required"`
		Extra         *hexutil.Bytes    `json:"extraData"        gencodec:"required"`
		LFHash        *common.Hash      `json:"lfHash"           gencodec:"required"`
		LFNumber      *hexutil.Uint64   `json:"lfNumber"         gencodec:"required"`
		LFBaseFee     *hexutil.Big      `json:"lfBaseFeePerGas"  gencodec:"required"`
		LFRoot        *common.Hash      `json:"lfStateRoot"      gencodec:"required"`
		LFReceiptHash *common.Hash      `json:"lfReceiptsRoot"   gencodec:"required"`
		LFGasUsed     *hexutil.Uint64   `json:"lfGasUsed"        gencodec:"required"`
		LFBloom       *Bloom            `json:"lfLogsBloom"      gencodec:"required"`
		BaseFee       *hexutil.Big      `json:"baseFeePerGas" rlp:"optional"`
		Number        *hexutil.Uint64   `json:"number"        rlp:"optional"`
		Root          *common.Hash      `json:"stateRoot"     rlp:"optional"`
		ReceiptHash   *common.Hash      `json:"receiptsRoot"  rlp:"optional"`
		GasUsed       *hexutil.Uint64   `json:"gasUsed"       rlp:"optional"`
		Bloom         *Bloom            `json:"logsBloom"     rlp:"optional"`
	}
	var dec Header
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	if dec.ParentHashes == nil {
		return errors.New("missing required field 'parentHashes' for Header")
	}
	h.ParentHashes = *dec.ParentHashes
	if dec.Slot == nil {
		return errors.New("missing required field 'slot' for Header")
	}
	h.Slot = uint64(*dec.Slot)
	if dec.Height == nil {
		return errors.New("missing required field 'height' for Header")
	}
	h.Height = uint64(*dec.Height)
	if dec.LFHash == nil {
		return errors.New("missing required field 'LFHash' for Header")
	}
	if dec.Coinbase == nil {
		return errors.New("missing required field 'miner' for Header")
	}
	h.Coinbase = *dec.Coinbase
	if dec.TxHash == nil {
		return errors.New("missing required field 'transactionsRoot' for Header")
	}
	h.TxHash = *dec.TxHash
	if dec.GasLimit == nil {
		return errors.New("missing required field 'gasLimit' for Header")
	}
	h.GasLimit = uint64(*dec.GasLimit)
	if dec.Time == nil {
		return errors.New("missing required field 'timestamp' for Header")
	}
	h.Time = uint64(*dec.Time)
	if dec.Extra == nil {
		return errors.New("missing required field 'extraData' for Header")
	}
	h.Extra = *dec.Extra
	if dec.LFHash == nil {
		return errors.New("missing required field 'lfHash' for Header")
	}
	h.LFHash = *dec.LFHash
	if dec.LFNumber == nil {
		return errors.New("missing required field 'lfNumber' for Header")
	}
	h.LFNumber = uint64(*dec.LFNumber)
	if dec.LFBaseFee == nil {
		return errors.New("missing required field 'lfBaseFeePerGas' for Header")
	}
	h.LFBaseFee = (*big.Int)(dec.LFBaseFee)
	if dec.LFRoot == nil {
		return errors.New("missing required field 'lfStateRoot' for Header")
	}
	h.LFRoot = *dec.LFRoot
	if dec.LFReceiptHash == nil {
		return errors.New("missing required field 'lfReceiptsRoot' for Header")
	}
	h.LFReceiptHash = *dec.LFReceiptHash
	if dec.LFGasUsed == nil {
		return errors.New("missing required field 'lfGasUsed' for Header")
	}
	h.LFGasUsed = uint64(*dec.LFGasUsed)
	if dec.LFBloom == nil {
		return errors.New("missing required field 'lfLogsBloom' for Header")
	}
	h.LFBloom = *dec.LFBloom
	if dec.BaseFee != nil {
		h.BaseFee = (*big.Int)(dec.BaseFee)
	}
	if dec.Number != nil {
		nr := uint64(*dec.Number)
		h.Number = &nr
	}
	if dec.Root != nil {
		h.Root = *dec.Root
	}
	if dec.ReceiptHash != nil {
		h.ReceiptHash = *dec.ReceiptHash
	}
	if dec.GasUsed != nil {
		h.GasUsed = uint64(*dec.GasUsed)
	}
	if dec.Bloom != nil {
		h.Bloom = *dec.Bloom
	}
	return nil
}
