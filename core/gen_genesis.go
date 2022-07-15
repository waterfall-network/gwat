// Code generated by github.com/fjl/gencodec. DO NOT EDIT.

package core

import (
	"encoding/json"
	"errors"
	"math/big"

	"github.com/waterfall-foundation/gwat/common"
	"github.com/waterfall-foundation/gwat/common/hexutil"
	"github.com/waterfall-foundation/gwat/common/math"
	"github.com/waterfall-foundation/gwat/params"
)

var _ = (*genesisSpecMarshaling)(nil)

// MarshalJSON marshals as JSON.
func (g Genesis) MarshalJSON() ([]byte, error) {
	type Genesis struct {
		Config    *params.ChainConfig                         `json:"config"`
		Nonce     math.HexOrDecimal64                         `json:"nonce"`
		Timestamp math.HexOrDecimal64                         `json:"timestamp"`
		ExtraData hexutil.Bytes                               `json:"extraData"`
		GasLimit  math.HexOrDecimal64                         `json:"gasLimit"   gencodec:"required"`
		Mixhash   common.Hash                                 `json:"mixHash"`
		Coinbase  common.Address                              `json:"coinbase"`
		Alloc     map[common.UnprefixedAddress]GenesisAccount `json:"alloc"      gencodec:"required"`
		Deposit   common.Address                              `json:"deposit"    gencodec:"required"`
		//Number       math.HexOrDecimal64                         `json:"number"`
		GasUsed      math.HexOrDecimal64   `json:"gasUsed"`
		ParentHashes []common.Hash         `json:"ParentHashes"`
		Epoch        math.HexOrDecimal64   `json:"epoch"`
		Slot         math.HexOrDecimal64   `json:"slot"`
		Height       math.HexOrDecimal64   `json:"height"`
		BaseFee      *math.HexOrDecimal256 `json:"baseFeePerGas"`
	}
	var enc Genesis
	enc.Config = g.Config
	enc.Nonce = math.HexOrDecimal64(g.Nonce)
	enc.Timestamp = math.HexOrDecimal64(g.Timestamp)
	enc.ExtraData = g.ExtraData
	enc.GasLimit = math.HexOrDecimal64(g.GasLimit)
	enc.Mixhash = g.Mixhash
	enc.Coinbase = g.Coinbase
	if g.Alloc != nil {
		enc.Alloc = make(map[common.UnprefixedAddress]GenesisAccount, len(g.Alloc))
		for k, v := range g.Alloc {
			enc.Alloc[common.UnprefixedAddress(k)] = v
		}
	}
	enc.Deposit = g.Deposit
	enc.GasUsed = math.HexOrDecimal64(g.GasUsed)
	enc.ParentHashes = g.ParentHashes
	enc.Epoch = math.HexOrDecimal64(g.Epoch)
	enc.Slot = math.HexOrDecimal64(g.Slot)
	enc.Height = math.HexOrDecimal64(g.Height)
	enc.BaseFee = (*math.HexOrDecimal256)(g.BaseFee)
	return json.Marshal(&enc)
}

// UnmarshalJSON unmarshals from JSON.
func (g *Genesis) UnmarshalJSON(input []byte) error {
	type Genesis struct {
		Config       *params.ChainConfig                         `json:"config"`
		Nonce        *math.HexOrDecimal64                        `json:"nonce"`
		Timestamp    *math.HexOrDecimal64                        `json:"timestamp"`
		ExtraData    *hexutil.Bytes                              `json:"extraData"`
		GasLimit     *math.HexOrDecimal64                        `json:"gasLimit"   gencodec:"required"`
		Mixhash      *common.Hash                                `json:"mixHash"`
		Coinbase     *common.Address                             `json:"coinbase"`
		Alloc        map[common.UnprefixedAddress]GenesisAccount `json:"alloc"      gencodec:"required"`
		Deposit      *common.Address                             `json:"deposit"    gencodec:"required"`
		GasUsed      *math.HexOrDecimal64                        `json:"gasUsed"`
		ParentHashes *[]common.Hash                              `json:"parentHashes"`
		Epoch        *math.HexOrDecimal64                        `json:"epoch"`
		Slot         *math.HexOrDecimal64                        `json:"slot"`
		Height       *math.HexOrDecimal64                        `json:"height"`
		BaseFee      *math.HexOrDecimal256                       `json:"baseFeePerGas"`
	}
	var dec Genesis
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	if dec.Config != nil {
		g.Config = dec.Config
	}
	if dec.Nonce != nil {
		g.Nonce = uint64(*dec.Nonce)
	}
	if dec.Timestamp != nil {
		g.Timestamp = uint64(*dec.Timestamp)
	}
	if dec.ExtraData != nil {
		g.ExtraData = *dec.ExtraData
	}
	if dec.GasLimit == nil {
		return errors.New("missing required field 'gasLimit' for Genesis")
	}
	g.GasLimit = uint64(*dec.GasLimit)
	if dec.Mixhash != nil {
		g.Mixhash = *dec.Mixhash
	}
	if dec.Coinbase != nil {
		g.Coinbase = *dec.Coinbase
	}
	if dec.Alloc == nil {
		return errors.New("missing required field 'alloc' for Genesis")
	}
	g.Alloc = make(GenesisAlloc, len(dec.Alloc))
	for k, v := range dec.Alloc {
		g.Alloc[common.Address(k)] = v
	}
	if dec.Deposit == nil {
		return errors.New("missing required field 'deposit' for Genesis")
	}
	g.Deposit = *dec.Deposit
	if dec.GasUsed != nil {
		g.GasUsed = uint64(*dec.GasUsed)
	}
	if dec.ParentHashes != nil {
		g.ParentHashes = *dec.ParentHashes
	}
	if dec.Epoch != nil {
		g.Epoch = uint64(*dec.Epoch)
	}
	if dec.Slot != nil {
		g.Slot = uint64(*dec.Slot)
	}
	if dec.Height != nil {
		g.Height = uint64(*dec.Height)
	}
	if dec.BaseFee != nil {
		g.BaseFee = (*big.Int)(dec.BaseFee)
	}
	return nil
}
