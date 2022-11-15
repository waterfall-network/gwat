// Code generated by github.com/fjl/gencodec. DO NOT EDIT.

package types

import (
	"encoding/json"
	"errors"
	"math/big"

	"github.com/waterfall-foundation/gwat/common"
	"github.com/waterfall-foundation/gwat/common/hexutil"
)

var _ = (*headerMarshaling)(nil)

// MarshalJSON marshals as JSON.
func (h Header) MarshalJSON() ([]byte, error) {
	type Header struct {
		ParentHashes []common.Hash   `json:"parentHashes"     gencodec:"required"`
		Slot         hexutil.Uint64  `json:"slot"             gencodec:"required"`
		Height       hexutil.Uint64  `json:"height"           gencodec:"required"`
		LFNumber     uint64          `json:"lfNumber"         gencodec:"required"`
		LFHash       common.Hash     `json:"lfHash"           gencodec:"required"`
		Coinbase     common.Address  `json:"miner"            gencodec:"required"`
		TxHash       common.Hash     `json:"transactionsRoot" gencodec:"required"`
		GasLimit     hexutil.Uint64  `json:"gasLimit"         gencodec:"required"`
		Time         hexutil.Uint64  `json:"timestamp"        gencodec:"required"`
		Extra        hexutil.Bytes   `json:"extraData"        gencodec:"required"`
		BaseFee      *hexutil.Big    `json:"baseFeePerGas"    rlp:"optional"`
		Number       *hexutil.Uint64 `json:"number"           rlp:"optional"`
		Root         common.Hash     `json:"stateRoot"        rlp:"optional"`
		ReceiptHash  common.Hash     `json:"receiptsRoot"     rlp:"optional"`
		Bloom        Bloom           `json:"logsBloom"        rlp:"optional"`
		GasUsed      hexutil.Uint64  `json:"gasUsed"          rlp:"optional"`
		Hash         common.Hash     `json:"hash"`
	}
	var enc Header
	enc.ParentHashes = h.ParentHashes
	enc.Slot = hexutil.Uint64(h.Slot)
	enc.Height = hexutil.Uint64(h.Height)
	enc.LFNumber = h.LFNumber
	enc.LFHash = h.LFHash
	enc.Coinbase = h.Coinbase
	enc.Root = h.Root
	enc.TxHash = h.TxHash
	enc.ReceiptHash = h.ReceiptHash
	enc.Bloom = h.Bloom
	enc.GasLimit = hexutil.Uint64(h.GasLimit)
	enc.GasUsed = hexutil.Uint64(h.GasUsed)
	enc.Time = hexutil.Uint64(h.Time)
	enc.Extra = h.Extra
	enc.BaseFee = (*hexutil.Big)(h.BaseFee)
	if h.Number != nil {
		nr := hexutil.Uint64(*h.Number)
		enc.Number = &nr
	}
	enc.Hash = h.Hash()
	return json.Marshal(&enc)
}

// UnmarshalJSON unmarshals from JSON.
func (h *Header) UnmarshalJSON(input []byte) error {
	type Header struct {
		ParentHashes *[]common.Hash  `json:"parentHashes"     gencodec:"required"`
		Slot         *hexutil.Uint64 `json:"slot"             gencodec:"required"`
		Height       *hexutil.Uint64 `json:"height"           gencodec:"required"`
		LFHash       *common.Hash    `json:"lfHash"           gencodec:"required"`
		LFNumber     *hexutil.Uint64 `json:"lfNumber"         gencodec:"required"`
		Coinbase     *common.Address `json:"miner"            gencodec:"required"`
		TxHash       *common.Hash    `json:"transactionsRoot" gencodec:"required"`
		GasLimit     *hexutil.Uint64 `json:"gasLimit"         gencodec:"required"`
		Time         *hexutil.Uint64 `json:"timestamp"        gencodec:"required"`
		Extra        *hexutil.Bytes  `json:"extraData"        gencodec:"required"`
		BaseFee      *hexutil.Big    `json:"baseFeePerGas"    rlp:"optional"`
		Number       *hexutil.Uint64 `json:"number"           rlp:"optional"`
		Root         *common.Hash    `json:"stateRoot"        rlp:"optional"`
		ReceiptHash  *common.Hash    `json:"receiptsRoot"     rlp:"optional"`
		Bloom        *Bloom          `json:"logsBloom"        rlp:"optional"`
		GasUsed      *hexutil.Uint64 `json:"gasUsed"          rlp:"optional"`
	}
	var dec Header
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	if dec.ParentHashes == nil {
		return errors.New("missing required field 'ParentHashes' for Header")
	}
	h.ParentHashes = *dec.ParentHashes
	if dec.Slot == nil {
		return errors.New("missing required field 'Slot' for Header")
	}
	h.Slot = uint64(*dec.Slot)
	if dec.Height == nil {
		return errors.New("missing required field 'Height' for Header")
	}
	h.Height = uint64(*dec.Height)
	if dec.LFHash == nil {
		return errors.New("missing required field 'LFHash' for Header")
	}
	h.LFHash = *dec.LFHash
	if dec.LFNumber == nil {
		return errors.New("missing required field 'LFNumber' for Header")
	}
	h.LFNumber = uint64(*dec.LFNumber)
	if dec.Coinbase == nil {
		return errors.New("missing required field 'miner' for Header")
	}
	h.Coinbase = *dec.Coinbase
	if dec.Root == nil {
		return errors.New("missing required field 'stateRoot' for Header")
	}
	h.Root = *dec.Root
	if dec.TxHash == nil {
		return errors.New("missing required field 'transactionsRoot' for Header")
	}
	h.TxHash = *dec.TxHash
	if dec.ReceiptHash == nil {
		return errors.New("missing required field 'receiptsRoot' for Header")
	}
	h.ReceiptHash = *dec.ReceiptHash
	if dec.Bloom == nil {
		return errors.New("missing required field 'logsBloom' for Header")
	}
	h.Bloom = *dec.Bloom
	if dec.GasLimit == nil {
		return errors.New("missing required field 'gasLimit' for Header")
	}
	h.GasLimit = uint64(*dec.GasLimit)
	if dec.GasUsed == nil {
		return errors.New("missing required field 'gasUsed' for Header")
	}
	h.GasUsed = uint64(*dec.GasUsed)
	if dec.Time == nil {
		return errors.New("missing required field 'timestamp' for Header")
	}
	h.Time = uint64(*dec.Time)
	if dec.Extra == nil {
		return errors.New("missing required field 'extraData' for Header")
	}
	h.Extra = *dec.Extra
	if dec.BaseFee != nil {
		h.BaseFee = (*big.Int)(dec.BaseFee)
	}
	if dec.Number != nil {
		nr := uint64(*dec.Number)
		h.Number = &nr
	}
	return nil
}
