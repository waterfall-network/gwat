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
		ParentHashes []common.Hash   `json:"parentHashes"     gencodec:"required"`
		Slot         hexutil.Uint64  `json:"slot"             gencodec:"required"`
		Era          hexutil.Uint64  `json:"era"              gencodec:"required"`
		Height       hexutil.Uint64  `json:"height"           gencodec:"required"`
		CpNumber     hexutil.Uint64  `json:"cpNumber"         gencodec:"required"`
		CpHash       common.Hash     `json:"cpHash"           gencodec:"required"`
		CpBaseFee     *hexutil.Big     `json:"cpBaseFeePerGas"  gencodec:"required"`
		CpRoot        common.Hash      `json:"cpStateRoot"      gencodec:"required"`
		CpReceiptHash common.Hash      `json:"cpReceiptsRoot"   gencodec:"required"`
		CpGasUsed     hexutil.Uint64   `json:"cpGasUsed"        gencodec:"required"`
		CpBloom       Bloom            `json:"cpLogsBloom"      gencodec:"required"`
		Coinbase     common.Address  `json:"miner"            gencodec:"required"`
		TxHash       common.Hash     `json:"transactionsRoot" gencodec:"required"`
		BodyHash     common.Hash     `json:"bodyRoot" 		  gencodec:"required"`
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
	enc.Era = hexutil.Uint64(h.Era)
	enc.Height = hexutil.Uint64(h.Height)
	enc.Coinbase = h.Coinbase
	enc.TxHash = h.TxHash
	enc.BodyHash = h.BodyHash
	enc.ReceiptHash = h.ReceiptHash
	enc.Bloom = h.Bloom
	enc.GasLimit = hexutil.Uint64(h.GasLimit)
	enc.Time = hexutil.Uint64(h.Time)
	enc.Extra = h.Extra
	enc.CpHash = h.CpHash
	enc.CpNumber = hexutil.Uint64(h.CpNumber)
	enc.CpBaseFee = (*hexutil.Big)(h.CpBaseFee)
	enc.CpRoot = h.CpRoot
	enc.CpReceiptHash = h.CpReceiptHash
	enc.CpGasUsed = hexutil.Uint64(h.CpGasUsed)
	enc.CpBloom = h.CpBloom
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
		ParentHashes *[]common.Hash  `json:"parentHashes"     gencodec:"required"`
		Slot         *hexutil.Uint64 `json:"slot"             gencodec:"required"`
		Era          *hexutil.Uint64 `json:"era"              gencodec:"required"`
		Height       *hexutil.Uint64 `json:"height"           gencodec:"required"`
		CpHash       *common.Hash    `json:"cpHash"           gencodec:"required"`
		CpNumber     *hexutil.Uint64 `json:"cpNumber"         gencodec:"required"`
		CpBaseFee     *hexutil.Big      `json:"cpBaseFeePerGas"  gencodec:"required"`
		CpRoot        *common.Hash      `json:"cpStateRoot"      gencodec:"required"`
		CpReceiptHash *common.Hash      `json:"cpReceiptsRoot"   gencodec:"required"`
		CpGasUsed     *hexutil.Uint64   `json:"cpGasUsed"        gencodec:"required"`
		CpBloom       *Bloom            `json:"cpLogsBloom"      gencodec:"required"`
		Coinbase     *common.Address `json:"miner"            gencodec:"required"`
		TxHash       *common.Hash    `json:"transactionsRoot" gencodec:"required"`
		BodyHash     *common.Hash    `json:"bodyRoot"         gencodec:"required"`
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
		return errors.New("missing required field 'parentHashes' for Header")
	}
	h.ParentHashes = *dec.ParentHashes
	if dec.Slot == nil {
		return errors.New("missing required field 'slot' for Header")
	}
	h.Slot = uint64(*dec.Slot)
	if dec.Era == nil {
		return errors.New("missing required field 'era' for Header")
	}
	h.Era = uint64(*dec.Era)
	if dec.Height == nil {
		return errors.New("missing required field 'height' for Header")
	}
	h.Height = uint64(*dec.Height)
	if dec.CpHash == nil {
		return errors.New("missing required field 'CpHash' for Header")
	}
	if dec.Coinbase == nil {
		return errors.New("missing required field 'miner' for Header")
	}
	h.Coinbase = *dec.Coinbase
	if dec.TxHash == nil {
		return errors.New("missing required field 'transactionsRoot' for Header")
	}
	h.TxHash = *dec.TxHash
	if dec.BodyHash == nil {
		return errors.New("missing required field 'bodyRoot' for Header")
	}
	h.BodyHash = *dec.BodyHash
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
	if dec.Time == nil {
		return errors.New("missing required field 'timestamp' for Header")
	}
	h.Time = uint64(*dec.Time)
	if dec.Extra == nil {
		return errors.New("missing required field 'extraData' for Header")
	}
	h.Extra = *dec.Extra
	if dec.CpHash == nil {
		return errors.New("missing required field 'cpHash' for Header")
	}
	h.CpHash = *dec.CpHash
	if dec.CpNumber == nil {
		return errors.New("missing required field 'cpNumber' for Header")
	}
	h.CpNumber = uint64(*dec.CpNumber)
	if dec.CpBaseFee == nil {
		return errors.New("missing required field 'lfBaseFeePerGas' for Header")
	}
	h.CpBaseFee = (*big.Int)(dec.CpBaseFee)
	if dec.CpRoot == nil {
		return errors.New("missing required field 'cpStateRoot' for Header")
	}
	h.CpRoot = *dec.CpRoot
	if dec.CpReceiptHash == nil {
		return errors.New("missing required field 'cpReceiptsRoot' for Header")
	}
	h.CpReceiptHash = *dec.CpReceiptHash
	if dec.CpGasUsed == nil {
		return errors.New("missing required field 'cpGasUsed' for Header")
	}
	h.CpGasUsed = uint64(*dec.CpGasUsed)
	if dec.CpBloom == nil {
		return errors.New("missing required field 'cpLogsBloom' for Header")
	}
	h.CpBloom = *dec.CpBloom
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
