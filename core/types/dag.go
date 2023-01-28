package types

import (
	"encoding/json"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/common/hexutil"
)

// ConsensusInfo represents data of consensus request
type ConsensusInfo struct {
	Slot       uint64           `json:"slot"`
	Creators   []common.Address `json:"creators"`
	Finalizing common.HashArray `json:"finalizing"`
}

// Copy duplicates the current storage.
func (ci *ConsensusInfo) Copy() *ConsensusInfo {
	return &ConsensusInfo{
		Slot:       ci.Slot,
		Creators:   ci.Creators,
		Finalizing: ci.Finalizing,
	}
}

type ConsensusInfoMarshaling struct {
	Slot       *hexutil.Uint64  `json:"slot"`
	Creators   []common.Address `json:"creators"`
	Finalizing common.HashArray `json:"finalizing"`
}

func (ci *ConsensusInfo) MarshalJSON() ([]byte, error) {
	out := ConsensusInfoMarshaling{
		Slot:       (*hexutil.Uint64)(&ci.Slot),
		Creators:   ci.Creators,
		Finalizing: ci.Finalizing,
	}
	return json.Marshal(out)
}

// UnmarshalJSON unmarshals from JSON.
func (ci *ConsensusInfo) UnmarshalJSON(input []byte) error {
	var dec ConsensusInfoMarshaling
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	if dec.Slot != nil {
		ci.Slot = uint64(*dec.Slot)
	}
	if dec.Creators != nil {
		ci.Creators = dec.Creators
	}
	if dec.Finalizing != nil {
		ci.Finalizing = dec.Finalizing
	}
	return nil
}

// FinalizationParams represents params of finalization request
type FinalizationParams struct {
	Spines    common.HashArray `json:"spines"`
	BaseSpine *common.Hash     `json:"baseSpine"`
}

// Copy duplicates the current storage.
func (fp *FinalizationParams) Copy() *FinalizationParams {
	cpy := &FinalizationParams{
		Spines: fp.Spines.Copy(),
	}
	if fp.BaseSpine != nil {
		cpy.BaseSpine = &common.Hash{}
		copy(cpy.BaseSpine[:], fp.BaseSpine[:])
	}
	return cpy
}

func (fp *FinalizationParams) MarshalJSON() ([]byte, error) {
	return json.Marshal(*fp)
}

// UnmarshalJSON unmarshals from JSON.
func (fp *FinalizationParams) UnmarshalJSON(input []byte) error {
	type Decoding FinalizationParams
	dec := Decoding{}
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	if dec.Spines != nil {
		fp.Spines = dec.Spines
	}
	if dec.BaseSpine != nil {
		fp.BaseSpine = dec.BaseSpine
	}
	return nil
}

// ConsensusResult represents result of handling of consensus request
type ConsensusResult struct {
	Error      *string            `json:"error"`
	Info       *map[string]string `json:"info"`
	Candidates common.HashArray   `json:"candidates"`
}

type FinalizationResult struct {
	Error   *string      `json:"error"`
	LFSpine *common.Hash `json:"lf_spine"`
}

type CandidatesResult struct {
	Error      *string          `json:"error"`
	Candidates common.HashArray `json:"candidates"`
}
