package types

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/big"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/common/hexutil"
)

// Checkpoint represents a coordinated checkpoint
// of coorinator and gwat nodes
type Checkpoint struct {
	Epoch uint64
	Root  common.Hash
	Spine common.Hash
}

type checkpointMarshaling struct {
	Epoch *hexutil.Uint64 `json:"epoch"`
	Root  *common.Hash    `json:"root"`
	Spine *common.Hash    `json:"spine"`
}

// Bytes gets the byte representation.
func (cp *Checkpoint) Bytes() []byte {
	cpLen := 8 + common.HashLength + common.HashLength
	res := make([]byte, 0, cpLen)
	epoch := make([]byte, 8)
	binary.BigEndian.PutUint64(epoch, cp.Epoch)
	res = append(res, epoch...)
	res = append(res, cp.Root.Bytes()...)
	res = append(res, cp.Spine.Bytes()...)
	return res
}

func (cp *Checkpoint) SetBytes(data []byte) error {
	cpLen := 8 + common.HashLength + common.HashLength
	if len(data) != cpLen {
		return fmt.Errorf("bad bitlen: got=%d req=%d", len(data), cpLen)
	}
	var start, end int
	start = 0
	end += 8
	cp.Epoch = binary.BigEndian.Uint64(data[start:end])

	start = end
	end += common.HashLength
	cp.Root = common.BytesToHash(data[start:end])

	start = end
	end += common.HashLength
	cp.Spine = common.BytesToHash(data[start:end])

	return nil
}

// BytesToCheckpoint create Checkpoint from bytes.
func BytesToCheckpoint(b []byte) (*Checkpoint, error) {
	var h Checkpoint
	err := h.SetBytes(b)
	if err != nil {
		return nil, err
	}
	return &h, err
}

func (cp *Checkpoint) Copy() *Checkpoint {
	cpy := &Checkpoint{
		Epoch: cp.Epoch,
		Spine: cp.Spine,
	}
	copy(cpy.Root[:], cp.Root[:])
	copy(cpy.Spine[:], cp.Spine[:])
	return cpy
}

func (cp *Checkpoint) MarshalJSON() ([]byte, error) {
	out := checkpointMarshaling{
		Epoch: (*hexutil.Uint64)(&cp.Epoch),
		Root:  &cp.Root,
		Spine: &cp.Spine,
	}
	return json.Marshal(out)
}

func (cp *Checkpoint) UnmarshalJSON(input []byte) error {
	var dec checkpointMarshaling
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	if dec.Epoch != nil {
		cp.Epoch = uint64(*dec.Epoch)
	}
	if dec.Root != nil {
		cp.Root = *dec.Root
	}
	if dec.Spine != nil {
		cp.Spine = *dec.Spine
	}
	return nil
}

type ValidatorSyncOp uint64

const (
	Activate ValidatorSyncOp = iota
	Deactivate
	UpdateBalance
)

// ValidatorSync represents a data to perform operation of validators synchronization
// from coordinator
type ValidatorSync struct {
	OpType    ValidatorSyncOp
	ProcEpoch uint64
	Index     uint64
	Creator   common.Address
	Amount    *big.Int
	TxHash    *common.Hash
}

type validatorSyncMarshaling struct {
	OpType    *hexutil.Uint64 `json:"opType"`
	ProcEpoch *hexutil.Uint64 `json:"procEpoch"`
	Index     *hexutil.Uint64 `json:"index"`
	Creator   *common.Address `json:"creator"`
	Amount    *hexutil.Big    `json:"amount"`
	TxHash    *common.Hash    `json:"txHash"`
}

func (vs *ValidatorSync) Copy() *ValidatorSync {
	cpy := &ValidatorSync{
		OpType:    vs.OpType,
		ProcEpoch: vs.ProcEpoch,
		Index:     vs.Index,
	}
	copy(cpy.Creator[:], vs.Creator[:])
	if vs.Amount != nil {
		cpy.Amount = new(big.Int).Set(vs.Amount)
	}
	if vs.TxHash != nil {
		copy(cpy.TxHash[:], vs.TxHash[:])
	}
	return cpy
}

func (vs *ValidatorSync) Key() [28]byte {
	var key [28]byte
	if vs == nil {
		return key
	}
	enc := make([]byte, 8)
	binary.BigEndian.PutUint64(enc, uint64(vs.OpType))
	enc = append(enc, vs.Creator.Bytes()...)
	copy(key[:], enc)
	return key
}

func (vs *ValidatorSync) MarshalJSON() ([]byte, error) {
	out := validatorSyncMarshaling{
		OpType:    (*hexutil.Uint64)(&vs.OpType),
		ProcEpoch: (*hexutil.Uint64)(&vs.ProcEpoch),
		Index:     (*hexutil.Uint64)(&vs.Index),
		Creator:   &vs.Creator,
		Amount:    nil,
		TxHash:    vs.TxHash,
	}
	if vs.Amount != nil {
		out.Amount = (*hexutil.Big)(vs.Amount)
	}
	return json.Marshal(out)
}

func (vs *ValidatorSync) UnmarshalJSON(input []byte) error {
	var dec validatorSyncMarshaling
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	if dec.OpType != nil {
		vs.OpType = ValidatorSyncOp(*dec.OpType)
	}
	if dec.ProcEpoch != nil {
		vs.ProcEpoch = uint64(*dec.ProcEpoch)
	}
	if dec.Index != nil {
		vs.Index = uint64(*dec.Index)
	}
	if dec.Creator != nil {
		vs.Creator = *dec.Creator
	}
	if dec.Amount != nil {
		vs.Amount = (*big.Int)(dec.Amount)
	}
	if dec.TxHash != nil {
		vs.TxHash = dec.TxHash
	}
	return nil
}

// FinalizationParams represents params of finalization request
type FinalizationParams struct {
	Spines      common.HashArray `json:"spines"`
	BaseSpine   *common.Hash     `json:"baseSpine"`
	Checkpoint  *Checkpoint      `json:"checkpoint"`
	ValSyncData []*ValidatorSync `json:"valSyncData"`
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
	if fp.Checkpoint != nil {
		cpy.Checkpoint = fp.Checkpoint.Copy()
	}
	if fp.ValSyncData != nil {
		cpy.ValSyncData = make([]*ValidatorSync, len(fp.ValSyncData))
		for i, vs := range fp.ValSyncData {
			cpy.ValSyncData[i] = vs.Copy()
		}
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
	if dec.Checkpoint != nil {
		fp.Checkpoint = dec.Checkpoint
	}
	if dec.ValSyncData != nil {
		fp.ValSyncData = dec.ValSyncData
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
	LFSpine *common.Hash `json:"lfSpine"`
	CpEpoch *uint64      `json:"cpEpoch"`
	CpRoot  *common.Hash `json:"cpRoot"`
}

type CandidatesResult struct {
	Error      *string          `json:"error"`
	Candidates common.HashArray `json:"candidates"`
}

type OptimisticSpinesResult struct {
	Data  []common.HashArray `json:"data"`
	Error *string            `json:"error"`
}
