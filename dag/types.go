package dag

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/dag/finalizer"
)

// ConsensusInfo represents data of consensus request
type ConsensusInfo struct {
	Epoch      uint64              `json:"epoch"`
	Slot       uint64              `json:"slot"`
	Creators   []common.Address    `json:"creators"`
	Finalizing finalizer.NrHashMap `json:"finalizing"`
}

// Copy duplicates the current storage.
func (ci *ConsensusInfo) Copy() *ConsensusInfo {
	return &ConsensusInfo{
		Epoch:      ci.Epoch,
		Slot:       ci.Slot,
		Creators:   ci.Creators,
		Finalizing: ci.Finalizing,
	}
}

// ConsensusResult represents result of handling of consensus request
type ConsensusResult struct {
	Error      *string              `json:"error"`
	Info       *map[string]string   `json:"info"`
	Candidates *finalizer.NrHashMap `json:"candidates"`
}
