// Code generated by github.com/fjl/gencodec. DO NOT EDIT.

package core

import (
	"encoding/json"
	"errors"
	"math/big"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/common/hexutil"
	"gitlab.waterfall.network/waterfall/protocol/gwat/common/math"
	"gitlab.waterfall.network/waterfall/protocol/gwat/params"
)

var _ = (*genesisSpecMarshaling)(nil)

// MarshalJSON marshals as JSON.
func (g Genesis) MarshalJSON() ([]byte, error) {
	type Genesis struct {
		Config       *params.ChainConfig                         `json:"config"`
		Timestamp    math.HexOrDecimal64                         `json:"timestamp"`
		ExtraData    hexutil.Bytes                               `json:"extraData"`
		GasLimit     math.HexOrDecimal64                         `json:"gasLimit"   gencodec:"required"`
		Coinbase     common.Address                              `json:"coinbase"`
		Alloc        map[common.UnprefixedAddress]GenesisAccount `json:"alloc"      gencodec:"required"`
		Validators   []common.Address                            `json:"validators"`
		GasUsed      math.HexOrDecimal64                         `json:"gasUsed"`
		ParentHashes []common.Hash                               `json:"parentHashes"`
		Slot         math.HexOrDecimal64                         `json:"slot"`
		Height       math.HexOrDecimal64                         `json:"height"`
		BaseFee      *math.HexOrDecimal256                       `json:"baseFeePerGas"`
	}
	var enc Genesis
	enc.Config = g.Config
	enc.Timestamp = math.HexOrDecimal64(g.Timestamp)
	enc.ExtraData = g.ExtraData
	enc.GasLimit = math.HexOrDecimal64(g.GasLimit)
	enc.Coinbase = g.Coinbase
	if g.Alloc != nil {
		enc.Alloc = make(map[common.UnprefixedAddress]GenesisAccount, len(g.Alloc))
		for k, v := range g.Alloc {
			enc.Alloc[common.UnprefixedAddress(k)] = v
		}
	}
	enc.Validators = g.Validators.Addresses()
	enc.GasUsed = math.HexOrDecimal64(g.GasUsed)
	enc.ParentHashes = g.ParentHashes
	enc.Slot = math.HexOrDecimal64(g.Slot)
	enc.Height = math.HexOrDecimal64(g.Height)
	enc.BaseFee = (*math.HexOrDecimal256)(g.BaseFee)
	return json.Marshal(&enc)
}

// UnmarshalJSON unmarshals from JSON.
func (g *Genesis) UnmarshalJSON(input []byte) error {
	type Genesis struct {
		Config       *params.ChainConfig                         `json:"config"`
		Timestamp    *math.HexOrDecimal64                        `json:"timestamp"`
		ExtraData    *hexutil.Bytes                              `json:"extraData"`
		GasLimit     *math.HexOrDecimal64                        `json:"gasLimit"   gencodec:"required"`
		Coinbase     *common.Address                             `json:"coinbase"`
		Alloc        map[common.UnprefixedAddress]GenesisAccount `json:"alloc"      gencodec:"required"`
		Validators   DepositData                                 `json:"validators"`
		GasUsed      *math.HexOrDecimal64                        `json:"gasUsed"`
		ParentHashes []common.Hash                               `json:"parentHashes"`
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
	if dec.Validators != nil {
		g.Validators = dec.Validators
	}
	if dec.GasUsed != nil {
		g.GasUsed = uint64(*dec.GasUsed)
	}
	if dec.ParentHashes != nil {
		g.ParentHashes = dec.ParentHashes
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
