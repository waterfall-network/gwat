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

package types

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/bits"
	"sort"
	"time"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/log"
)

/********** Tips **********/

// Tips represents the tips of dag chain.
type Tips map[common.Hash]*BlockDAG

// Add adds to tips new item or replace existed
func (tips Tips) Add(blockDag *BlockDAG) Tips {
	if blockDag == nil {
		log.Warn("Add tips received bad BlockDAG", "item", blockDag)
		return tips
	}

	blockDag.OrderedAncestorsHashes.Deduplicate()

	tips[blockDag.Hash] = blockDag
	return tips
}

// Remove removes item from tips
func (tips Tips) Remove(hash common.Hash) Tips {
	delete(tips, hash)
	return tips
}

// Get returns tips' item by hash
func (tips Tips) Get(hash common.Hash) *BlockDAG {
	return tips[hash]
}

// GetHashes returns hashes of tips' items
func (tips Tips) GetHashes() common.HashArray {
	hashes := make(common.HashArray, len(tips))
	i := 0
	for _, b := range tips {
		hashes[i] = b.Hash
		i++
	}
	return hashes
}

// Copy creates copy of the current tips.
func (tips Tips) Copy() Tips {
	cpy := make(Tips)
	for key, value := range tips {
		cpy[key] = value
	}
	return cpy
}

// GetOrderedAncestorsHashes collect all dag hashes in finalizing order
// used to create for next blockDag OrderedAncestorsHashes
func (tips Tips) GetOrderedAncestorsHashes() common.HashArray {
	allHashes := common.HashArray{}
	dagHashes := tips.getOrderedHashes()
	for _, dagHash := range dagHashes {
		dag := tips.Get(dagHash)
		allHashes = append(allHashes, dag.OrderedAncestorsHashes...)
		allHashes = append(allHashes, dagHash).Uniq()
	}
	return allHashes.Uniq()
}

// GetStableStateHash retrieve the hash of the stable state
func (tips Tips) GetStableStateHash() common.Hash {
	finDag := tips.GetFinalizingDag()
	return finDag.Hash
}

// GetFinalizingDag retrieve the top dag with stable state
// to create ordered chain for finalization process
func (tips Tips) GetFinalizingDag() *BlockDAG {
	dagHashes := tips.getOrderedHashes()
	if len(dagHashes) < 1 {
		return nil
	}
	return tips.Get(dagHashes[0])
}

// GetAncestorsHashes retrieve all OrderedAncestorsHashes
func (tips Tips) GetAncestorsHashes() common.HashArray {
	ancestors := common.HashArray{}
	for _, dag := range tips {
		ancestors = append(ancestors, dag.OrderedAncestorsHashes...)
	}
	return ancestors
}

// Print returns string representation of tips.
func (tips Tips) Print() string {
	mapB, _ := json.Marshal(tips)
	return string(mapB)
}

// getOrderedHashes sort tips in finalizing order
func (tips Tips) getOrderedHashes() common.HashArray {
	heightMap := map[uint64]common.HashArray{}
	for _, dag := range tips {
		key := dag.Height
		heightMap[key] = append(heightMap[key], dag.Hash)
	}
	// sort by height
	keys := make(common.SorterDescU64, 0, len(heightMap))
	for k := range heightMap {
		keys = append(keys, k)
	}
	sort.Sort(keys)
	sortedHashes := common.HashArray{}
	for _, k := range keys {
		// sort by hash
		sortedHashes = sortedHashes.Concat(heightMap[k].Sort())
	}
	return sortedHashes
}

// GetMaxSlot get max slot of tips
func (tips Tips) GetMaxSlot() uint64 {
	maxSlot := uint64(0)
	for _, t := range tips {
		if t.Slot > maxSlot {
			maxSlot = t.Slot
		}
	}
	return maxSlot
}

/********** BlockDAG **********/

// BlockDAG represents a currently no descendants block
// of directed acyclic graph and related data.
type BlockDAG struct {
	Hash     common.Hash
	Height   uint64
	Slot     uint64
	CpHash   common.Hash
	CpHeight uint64
	// ordered non-finalized ancestors hashes
	OrderedAncestorsHashes common.HashArray
}

// ToBytes encodes the BlockDAG structure
// to byte representation.
func (b *BlockDAG) ToBytes() []byte {
	res := []byte{}
	res = append(res, b.Hash.Bytes()...)

	height := make([]byte, 8)
	binary.BigEndian.PutUint64(height, b.Height)
	res = append(res, height...)

	slot := make([]byte, 8)
	binary.BigEndian.PutUint64(slot, b.Slot)
	res = append(res, slot...)

	res = append(res, b.CpHash.Bytes()...)
	lastFinHeight := make([]byte, 8)
	binary.BigEndian.PutUint64(lastFinHeight, b.CpHeight)
	res = append(res, lastFinHeight...)

	lenDC := make([]byte, 4)
	binary.BigEndian.PutUint32(lenDC, uint32(len(b.OrderedAncestorsHashes)))
	res = append(res, lenDC...)
	res = append(res, b.OrderedAncestorsHashes.ToBytes()...)

	return res
}

// SetBytes restores BlockDAG structure from byte representation.
func (b *BlockDAG) SetBytes(data []byte) *BlockDAG {
	start := 0
	end := common.HashLength
	b.Hash = common.BytesToHash(data[start:end])

	start = end
	end += 8
	b.Height = binary.BigEndian.Uint64(data[start:end])

	start = end
	end += 8
	b.Slot = binary.BigEndian.Uint64(data[start:end])

	start = end
	end += common.HashLength
	b.CpHash = common.BytesToHash(data[start:end])

	start = end
	end += 8
	b.CpHeight = binary.BigEndian.Uint64(data[start:end])

	start = end
	end += 4
	lenDC := binary.BigEndian.Uint32(data[start:end])
	start = end
	end += common.HashLength * int(lenDC)
	b.OrderedAncestorsHashes = common.HashArrayFromBytes(data[start:end])

	return b
}

// Print returns string representation of BlockDAG structure.
func (b *BlockDAG) Print() string {
	mapB, _ := json.Marshal(b)
	return string(mapB)
}

/********** HeaderMap **********/
// HeaderMap represents map of block headers
type HeaderMap map[common.Hash]*Header

// FromArray fills HeaderMap by data from array of headers.
func (hm HeaderMap) FromArray(headers []*Header) HeaderMap {
	for _, h := range headers {
		if h != nil {
			hm.Add(h)
		}
	}
	return hm
}

// RmEmpty removes nil-items
func (hm HeaderMap) RmEmpty() HeaderMap {
	for k, h := range hm {
		if h == nil {
			delete(hm, k)
		}
	}
	return hm
}

// ToArray casts HeaderMap to array of block headers.
func (hm HeaderMap) ToArray() []*Header {
	arr := make([]*Header, len(hm))
	i := 0
	for _, h := range hm {
		arr[i] = h
		i++
	}
	return arr
}

// Add adds header to map
func (hm HeaderMap) Add(header *Header) HeaderMap {
	if header == nil {
		log.Warn("Add HeaderMap received bad header", "item", header)
		return hm
	}
	hm[header.Hash()] = header
	return hm
}

// Hashes retrieves hashes of map
func (hm HeaderMap) Hashes() common.HashArray {
	hashes := make(common.HashArray, 0)
	for _, h := range hm {
		hashes = append(hashes, h.Hash())
	}
	return hashes
}

// GetMaxTime retrieves the max block create time
// from current HeaderMap
func (hm HeaderMap) GetMaxTime() uint64 {
	maxtipsTs := uint64(0)
	for _, p := range hm {
		if p.Time > maxtipsTs {
			maxtipsTs = p.Time
		}
	}
	return maxtipsTs
}

// AvgGasLimit retrieves average gas limit of HeaderMap.
func (hm HeaderMap) AvgGasLimit() uint64 {
	count := uint64(len(hm))
	if count < 1 {
		return 0
	}
	var sum uint64 = 0
	for _, itm := range hm {
		sum += itm.GasLimit
	}
	return sum / count
}

// ParentHashes retrieves all ParentHases from HeaderMap
func (hm HeaderMap) ParentHashes() common.HashArray {
	hashes := make(common.HashArray, 0)
	for _, h := range hm {
		hashes = hashes.Concat(h.ParentHashes)
		//hashes = append(hashes, h.Hash())
	}
	return hashes
}

// FinalizingSort sort in finalizing order
func (hm HeaderMap) FinalizingSort() common.HashArray {
	heightMap := map[uint64]common.HashArray{}
	for _, h := range hm {
		key := h.Height
		heightMap[key] = append(heightMap[key], h.Hash())
	}
	// sort by height
	keys := make(common.SorterAscU64, 0, len(heightMap))
	for k := range heightMap {
		keys = append(keys, k)
	}
	sort.Sort(keys)
	sortedHashes := common.HashArray{}
	for _, k := range keys {
		// sort by hash
		sortedHashes = sortedHashes.Concat(heightMap[k].Sort())
	}
	return sortedHashes
}

/********** BlockMap **********/

// BlockMap represents map of blocks
type BlockMap map[common.Hash]*Block

// ToArray casts BlockMap to array of blocks.
func (bm BlockMap) ToArray() []*Block {
	arr := make([]*Block, len(bm))
	i := 0
	for _, h := range bm {
		arr[i] = h
		i++
	}
	return arr
}

// Add adds block to map
func (bm BlockMap) Add(block *Block) BlockMap {
	if block == nil {
		log.Warn("Add BlockMap received bad block", "item", block)
		return bm
	}
	bm[block.Hash()] = block
	return bm
}

// Hashes retrieves hashes of map
func (bm BlockMap) Hashes() common.HashArray {
	hashes := make(common.HashArray, 0)
	for _, block := range bm {
		hashes = append(hashes, block.Hash())
	}
	return hashes
}

// Headers retrieves HeaderMap from current BlockMap.
func (bm BlockMap) Headers() HeaderMap {
	headers := HeaderMap{}
	for _, block := range bm {
		headers[block.Hash()] = block.Header()
	}
	return headers
}

// GetMaxTime retrieves the max block create time
// from current BlockMap
func (bm BlockMap) GetMaxTime() uint64 {
	return bm.Headers().GetMaxTime()
}

// AvgGasLimit calculate average GasLimit
func (bm BlockMap) AvgGasLimit() uint64 {
	return bm.Headers().AvgGasLimit()
}

// FinalizingSort sort in finalizing order
func (bm BlockMap) FinalizingSort() common.HashArray {
	return bm.Headers().FinalizingSort()
}

/********** SlotInfo **********/

type SlotInfo struct {
	GenesisTime    uint64 `json:"genesisTime"`
	SecondsPerSlot uint64 `json:"secondsPerSlot"`
	SlotsPerEpoch  uint64 `json:"slotsPerEpoch"`
}

// StartSlotTime takes the given slot to determine the start time of the slot.
func (si *SlotInfo) StartSlotTime(slot uint64) (time.Time, error) {
	genesisTimeSec := si.GenesisTime
	sps := si.SecondsPerSlot
	overflows, timeSinceGenesis := bits.Mul64(slot, sps)
	if overflows > 0 {
		return time.Unix(0, 0), fmt.Errorf("slot (%d) is in the far distant future: %w", slot, errors.New("multiplication overflows"))
	}
	sTime, carry := bits.Add64(timeSinceGenesis, genesisTimeSec, 0)
	if carry > 0 {
		return time.Unix(0, 0), fmt.Errorf("slot (%d) is in the far distant future: %w", slot, errors.New("addition overflows"))
	}
	return time.Unix(int64(sTime), 0), nil // lint:ignore uintcast -- A timestamp will not exceed int64 in your lifetime.
}

// CurrentSlot returns the current slot as determined by the local clock.
func (si *SlotInfo) CurrentSlot() uint64 {
	genesisTimeSec := si.GenesisTime
	sps := si.SecondsPerSlot
	now := time.Now().Unix()
	genesis := int64(genesisTimeSec) // lint:ignore uintcast -- Genesis timestamp will not exceed int64 in your lifetime.
	if now < genesis {
		return 0
	}
	return uint64(now-genesis) / sps
}

// IsEpochStart returns true if the given slot number is an epoch starting slot
// number.
func (si *SlotInfo) IsEpochStart(slot uint64) bool {
	return slot%si.SlotsPerEpoch == 0
}

// IsEpochEnd returns true if the given slot number is an epoch ending slot
// number.
func (si *SlotInfo) IsEpochEnd(slot uint64) bool {
	return si.IsEpochStart(slot + 1)
}

// SlotToEpoch returns the epoch number of the input slot.
func (si *SlotInfo) SlotToEpoch(slot uint64) uint64 {
	if si.SlotsPerEpoch == 0 {
		panic("integer divide by zero")
	}
	val, _ := bits.Div64(0, slot, si.SlotsPerEpoch)
	return val
}

func (si *SlotInfo) SlotInEpoch(slot uint64) uint64 {
	return slot % si.SlotsPerEpoch
}

// SlotOfEpochStart returns the first slot number of the
// current epoch.
func (si *SlotInfo) SlotOfEpochStart(epoch uint64) (uint64, error) {
	overflows, slot := bits.Mul64(si.SlotsPerEpoch, epoch)
	if overflows > 0 {
		return slot, errors.New("start slot calculation overflows: multiplication overflows")
	}
	return slot, nil
}

// SlotOfEpochEnd returns the last slot number of the
// current epoch.
func (si *SlotInfo) SlotOfEpochEnd(epoch uint64) (uint64, error) {
	if epoch == math.MaxUint64 {
		return 0, errors.New("start slot calculation overflows")
	}
	slot, err := si.SlotOfEpochStart(epoch + 1)
	if err != nil {
		return 0, err
	}
	return slot - 1, nil
}

func (si *SlotInfo) Copy() *SlotInfo {
	if si != nil {
		return &SlotInfo{
			GenesisTime:    si.GenesisTime,
			SecondsPerSlot: si.SecondsPerSlot,
			SlotsPerEpoch:  si.SlotsPerEpoch,
		}
	} else {
		return nil
	}
}
