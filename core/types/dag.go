package types

import (
	"encoding/json"

	"github.com/waterfall-foundation/gwat/common"
	"github.com/waterfall-foundation/gwat/common/hexutil"
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

// ConsensusResult represents result of handling of consensus request
type ConsensusResult struct {
	Error      *string            `json:"error"`
	Info       *map[string]string `json:"info"`
	Candidates common.HashArray   `json:"candidates"`
}

type FinalizationResult struct {
	Error *string `json:"error"`
}

type CandidatesResult struct {
	Error      *string          `json:"error"`
	Candidates common.HashArray `json:"candidates"`
}
