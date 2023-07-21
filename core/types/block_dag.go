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

	blockDag.DagChainHashes.Deduplicate()

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

// GetOrderedDagChainHashes collect all dag hashes in finalizing order
// used to create for next blockDag DagChainHashes
func (tips Tips) GetOrderedDagChainHashes() common.HashArray {
	allHashes := common.HashArray{}
	dagHashes := tips.getOrderedHashes()
	for _, dagHash := range dagHashes {
		dag := tips.Get(dagHash)
		allHashes = append(allHashes, dag.DagChainHashes...)
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

// GetAncestorsHashes retrieve all DagChainHashes
func (tips Tips) GetAncestorsHashes() common.HashArray {
	ancestors := common.HashArray{}
	for _, dag := range tips {
		ancestors = append(ancestors, dag.DagChainHashes...)
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
	DagChainHashes common.HashArray
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
	binary.BigEndian.PutUint32(lenDC, uint32(len(b.DagChainHashes)))
	res = append(res, lenDC...)
	res = append(res, b.DagChainHashes.ToBytes()...)

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
	b.DagChainHashes = common.HashArrayFromBytes(data[start:end])

	return b
}

// Print returns string representation of BlockDAG structure.
func (b *BlockDAG) Print() string {
	mapB, _ := json.Marshal(b)
	return string(mapB)
}

/********** DagGraph **********/
// BlockState represents state of GraphDag structure
type BlockState uint8

const (
	BSS_NOT_LOADED BlockState = 0
	BSS_LOADED     BlockState = 1
	BSS_FINALIZED  BlockState = 2
)

// GraphDag represents Graph of DAG
type GraphDag struct {
	Hash   common.Hash
	Height uint64
	Number uint64
	Graph  []*GraphDag
	State  BlockState
	chLen  *uint64
}

// GetLastFinalizedAncestor searches the GraphDag of the last finalised ancestor
func (gd *GraphDag) GetLastFinalizedAncestor() *GraphDag {
	var (
		lastHeight uint64    = 0
		res        *GraphDag = nil
	)
	ancestors := gd.GetAncestors()
	//find height of finalized blue block
	for _, itm := range ancestors {
		if itm.Height == itm.Number && (lastHeight < itm.Number || lastHeight == 0) {
			lastHeight = itm.Number
			res = itm
		}
	}
	return res
}

// GetDagChainHashes retrieves ordered DagChainHashes for
// tips item
func (gd *GraphDag) GetDagChainHashes() *common.HashArray {
	ancestors := gd.GetOrderedLoadedAncestors()
	if ancestors == nil {
		return nil
	}
	finalityPoints := &common.HashArray{}
	for _, itm := range ancestors {
		*finalityPoints = append(*finalityPoints, itm.Hash)
	}
	res := (*finalityPoints).Uniq()
	return &res
}

// todo deprecated (don't use)
// GetOrderedLoadedAncestors retrieves ordered ancestors
func (gd *GraphDag) GetOrderedLoadedAncestors() []*GraphDag {
	//1) calculate order for parents
	lenMap := map[uint64][]*GraphDag{}
	for _, itm := range gd.Graph {
		if itm.State == BSS_NOT_LOADED {
			return nil
		}
		if itm.State == BSS_FINALIZED {
			continue
		}
		itm.chLen = itm.chainLen()
		key := itm.chLen
		if key == nil {
			return nil
		}
		lenMap[*key] = append(lenMap[*key], itm)
	}
	// sort by height
	keys := make(common.SorterDescU64, 0, len(lenMap))
	for k := range lenMap {
		keys = append(keys, k)
	}
	sort.Sort(keys)
	parents := []*GraphDag{}
	for _, k := range keys {
		// sort by hash
		hashes := common.HashArray{}
		for _, itm := range lenMap[k] {
			hashes = append(hashes, itm.Hash)
		}
		for _, h := range hashes.Sort() {
			for _, val := range lenMap[k] {
				if val.Hash == h {
					parents = append(parents, val)
					break
				}
			}
		}
	}
	//2) recursive calculate order for all ancestors
	ancestors := []*GraphDag{}
	for _, itm := range parents {
		recRes := itm.GetOrderedLoadedAncestors()
		if recRes == nil {
			return nil
		}
		ancestors = append(ancestors, recRes...)
		ancestors = append(ancestors, itm)
	}
	//3) uniq items
	check := common.HashArray{}
	uniqAncestors := []*GraphDag{}
	for _, itm := range ancestors {
		if !check.Has(itm.Hash) {
			uniqAncestors = append(uniqAncestors, itm)
			check = append(check, itm.Hash)
		}
	}
	return uniqAncestors
}

// GetAncestorsHashes retrieves graph ancestors
func (gd *GraphDag) GetAncestorsHashes() common.HashArray {
	res := common.HashArray{}
	for _, itm := range gd.GetAncestors() {
		res = append(res, itm.Hash)
	}
	return res
}

// GetAncestors retrieves graph ancestors
func (gd *GraphDag) GetAncestors() []*GraphDag {
	parents := gd.Graph
	//1) recursive calculate order for all ancestors
	ancestors := []*GraphDag{}
	for _, itm := range parents {
		recRes := itm.GetAncestors()
		if recRes == nil {
			return nil
		}
		ancestors = append(ancestors, recRes...)
		ancestors = append(ancestors, itm)
	}
	//2) uniq items
	check := common.HashArray{}
	uniqAncestors := []*GraphDag{}
	for _, itm := range ancestors {
		if !check.Has(itm.Hash) {
			uniqAncestors = append(uniqAncestors, itm)
			check = append(check, itm.Hash)
		}
	}
	return uniqAncestors
}

// chainLen calculates the length of dag chain of GraphDag
func (gd *GraphDag) chainLen() *uint64 {
	var length uint64 = 1
	if gd.State == BSS_NOT_LOADED {
		return nil
	}
	if gd.State == BSS_FINALIZED {
		return &gd.Number
	}
	if len(gd.Graph) < 1 {
		return nil
	}
	for _, itm := range gd.Graph {
		res := itm.chainLen()
		if res == nil {
			return nil
		}
		length += *res
	}
	gd.chLen = &length
	return &length
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

// ToTipsGraphDag GraphDag for tips
func (hm HeaderMap) ToTipsGraphDag(genesis *common.Hash) []*GraphDag {
	// create GraphDag fore each header
	gdMap := map[common.Hash]*GraphDag{}
	for k, h := range hm {
		if h == nil {
			gdMap[k] = &GraphDag{
				Hash:  k,
				Graph: []*GraphDag{},
				State: BSS_NOT_LOADED,
			}
			continue
		}
		state := BSS_LOADED
		if h.Nr() > 0 || (genesis != nil && h.Hash() == *genesis) {
			state = BSS_FINALIZED
		}
		gdMap[h.Hash()] = &GraphDag{
			Hash:   h.Hash(),
			Height: h.Height,
			Number: h.Nr(),
			Graph:  []*GraphDag{},
			State:  state,
		}
	}
	for _, h := range hm {
		gdMap[h.Hash()] = hm.fillGraphDagRecursive(gdMap[h.Hash()], gdMap)
	}
	// find top dags
	allHashes := hm.Hashes()
	for _, gd := range gdMap {
		ancestors := gd.GetAncestorsHashes()
		if ancestors != nil {
			allHashes = allHashes.Difference(ancestors)
		}
	}
	//make result
	res := []*GraphDag{}
	for _, h := range allHashes {
		res = append(res, gdMap[h])
	}
	return res
}

func (hm HeaderMap) fillGraphDagRecursive(gd *GraphDag, gdMap map[common.Hash]*GraphDag) *GraphDag {
	if hm[gd.Hash] == nil {
		return gd
	}
	existed := common.HashArray{}
	for _, parent := range gd.Graph {
		existed = append(existed, parent.Hash)
	}
	for _, parent := range hm[gd.Hash].ParentHashes {
		if existed.Has(parent) {
			continue
		}
		if parentGd := gdMap[parent]; parentGd != nil {
			gdParent := hm.fillGraphDagRecursive(gdMap[parent], gdMap)
			gd.Graph = append(gd.Graph, gdParent)
		}
	}
	return gd
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
