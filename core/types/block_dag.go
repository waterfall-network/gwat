package types

import (
	"encoding/binary"
	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"sort"
)

/********** Tips **********/

type Tips map[common.Hash]*BlockDAG

func (tips Tips) Add(blockDag *BlockDAG) Tips {
	if blockDag == nil {
		log.Warn("Tips::Add received BlockDAG=nil")
		return tips
	}

	blockDag.DagChainHashes = blockDag.DagChainHashes.Uniq()
	blockDag.FinalityPoints = blockDag.FinalityPoints.Uniq()

	tips[blockDag.Hash] = blockDag
	return tips
}

func (tips Tips) Remove(hash common.Hash) Tips {
	delete(tips, hash)
	return tips
}

func (tips Tips) Get(hash common.Hash) *BlockDAG {
	return tips[hash]
}

func (tips Tips) GetHashes() common.HashArray {
	hashes := make(common.HashArray, len(tips))
	i := 0
	for _, b := range tips {
		hashes[i] = b.Hash
		i++
	}
	return hashes
}

func (tips Tips) GetKey() common.Hash {
	return tips.GetHashes().Key()
}

// Copy duplicates the current storage.
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
		hashesAll := append(dag.DagChainHashes, dagHash)
		hashesAdd := hashesAll[:]
		allHashesReorg := allHashes[:]
		for i, h := range allHashes {
			ix := hashesAll.IndexOf(h)
			if ix >= 0 {
				hashesAdd = hashesAll[ix+1:]

				hashesRmd := hashesAll[:ix]
				tmpix := allHashesReorg.IndexOf(h)
				allHashesTmp := append(allHashesReorg[:tmpix], hashesRmd...)
				allHashesReorg = append(allHashesTmp, allHashes[i:]...)
				allHashesReorg = allHashesReorg.Uniq()
			}
		}
		allHashes = append(allHashesReorg, hashesAdd...)
	}
	return allHashes
}

func (tips Tips) GetLowestFinNrDag() *BlockDAG {
	var res *BlockDAG
	for _, t := range tips {
		if res == nil || res.LastFinalizedHeight > t.LastFinalizedHeight {
			res = t
		}
	}
	return res
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

func (tips Tips) Print() string {
	mapB, _ := json.Marshal(tips)
	return string(mapB)
}

//getOrderedHashes sort tips in finalizing order
func (tips Tips) getOrderedHashes() common.HashArray {
	heightMap := map[uint64]common.HashArray{}
	for _, dag := range tips {
		key := dag.Height
		heightMap[key] = append(heightMap[key], dag.Hash)
	}
	// sort by height
	keys := make(common.SorterDeskU64, 0, len(heightMap))
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

/********** BlockDAG **********/

// BlockDAG represents a currently no descendants block
// of directed acyclic graph and related data.
type BlockDAG struct {
	Hash                common.Hash
	Height              uint64
	LastFinalizedHash   common.Hash
	LastFinalizedHeight uint64
	// ordered non-finalized ancestors hashes
	DagChainHashes common.HashArray
	// ordered points of future finalization
	FinalityPoints common.HashArray
}

func (b *BlockDAG) ToBytes() []byte {
	res := []byte{}
	res = append(res, b.Hash.Bytes()...)

	height := make([]byte, 8)
	binary.BigEndian.PutUint64(height, b.Height)
	res = append(res, height...)

	res = append(res, b.LastFinalizedHash.Bytes()...)
	lastFinHeight := make([]byte, 8)
	binary.BigEndian.PutUint64(lastFinHeight, b.LastFinalizedHeight)
	res = append(res, lastFinHeight...)

	lenDC := make([]byte, 4)
	binary.BigEndian.PutUint32(lenDC, uint32(len(b.DagChainHashes)))
	res = append(res, lenDC...)
	res = append(res, b.DagChainHashes.ToBytes()...)

	lenFP := make([]byte, 4)
	binary.BigEndian.PutUint32(lenFP, uint32(len(b.FinalityPoints)))
	res = append(res, lenFP...)
	res = append(res, b.FinalityPoints.ToBytes()...)
	return res
}

func (b *BlockDAG) SetBytes(data []byte) *BlockDAG {
	start := 0
	end := common.HashLength
	b.Hash = common.BytesToHash(data[start:end])

	start = end
	end += 8
	b.Height = binary.BigEndian.Uint64(data[start:end])

	start = end
	end += common.HashLength
	b.LastFinalizedHash = common.BytesToHash(data[start:end])

	start = end
	end += 8
	b.LastFinalizedHeight = binary.BigEndian.Uint64(data[start:end])

	start = end
	end += 4
	lenDC := binary.BigEndian.Uint32(data[start:end])
	start = end
	end += common.HashLength * int(lenDC)
	b.DagChainHashes = common.HashArrayFromBytes(data[start:end])

	start = end
	end += 4
	lenFP := binary.BigEndian.Uint32(data[start:end])
	start = end
	end += common.HashLength * int(lenFP)
	b.FinalityPoints = common.HashArrayFromBytes(data[start:end])
	return b
}

//// deprecated
//// Height calculate block dag height
//func (b *BlockDAG) CalcHeight() uint64 {
//	return b.LastFinalizedHeight + uint64(len(b.DagChainHashes.Uniq())) + 1
//}

func (b *BlockDAG) Print() string {
	mapB, _ := json.Marshal(b)
	return string(mapB)
}

/********** DagGraph **********/
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

//GetFinalityPoints retrieves ordered FinalityPoints' hashes for tips item
func (gd *GraphDag) GetFinalityPoints() *common.HashArray {
	var (
		finalityPoints = &common.HashArray{}
	)
	ancestorsLoaded := gd.GetOrderedLoadedAncestors()
	if ancestorsLoaded == nil {
		return nil
	}
	lastFinGd := gd.GetLastFinalizedAncestor()
	if lastFinGd == nil {
		return nil
	}
	lastHeight := lastFinGd.Number
	for i, itm := range ancestorsLoaded {
		//log.Info("================= itm >>>>>>>>>>>>>>>>>", "condition", itm.Height == uint64(i)+lastHeight+1, "Height", itm.Height, "Number", uint64(i)+lastHeight+1, "lastHeight", lastHeight, "hash", itm.Hash.Hex())
		if itm.Height == uint64(i)+lastHeight+1 {
			*finalityPoints = append(*finalityPoints, itm.Hash)
		}
	}
	res := (*finalityPoints).Uniq()
	return &res
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
		//log.Info("================= itm =================", "condition", itm.Height == itm.Number, "Height", itm.Height, "Number", itm.Number, "lastHeight", lastHeight)
	}
	return res
}

// GetDagChainHashes retrieves ordered DagChainHashes for
//tips item
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

//GetOrderedFinalizationPoints retrieves ordered FinalizationPoints
func (gd *GraphDag) GetOrderedFinalizationPoints(lastHeight uint64) []*GraphDag {
	ancestors := gd.GetOrderedLoadedAncestors()
	if ancestors == nil {
		return nil
	}
	heightMap := map[uint64]common.HashArray{}
	fPoints := []*GraphDag{}
	for _, itm := range ancestors {
		if chLen := itm.chainLen(); chLen != nil && itm.Height == *chLen {
			key := itm.Height
			heightMap[key] = append(heightMap[key], itm.Hash)
			fPoints = append(fPoints, itm)
		}
	}
	// sort by height
	keys := make(common.SorterDeskU64, 0, len(heightMap))
	for k := range heightMap {
		keys = append(keys, k)
	}
	sort.Sort(keys)
	finHashes := common.HashArray{}
	for _, k := range keys {
		// sort by hash and select first
		sorted := heightMap[k].Sort()
		if len(sorted) > 0 {
			finHashes = append(finHashes, sorted[0])
		}
	}
	finalityPoints := []*GraphDag{}
	for _, itm := range fPoints {
		if finHashes.Has(itm.Hash) {
			finalityPoints = append(finalityPoints, itm)
		}
	}
	return fPoints
}

//GetOrderedLoadedAncestors retrieves ordered ancestors
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
	keys := make(common.SorterDeskU64, 0, len(lenMap))
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

//GetAncestors retrieves graph ancestors
func (gd *GraphDag) GetAncestorsHashes() common.HashArray {
	res := common.HashArray{}
	for _, itm := range gd.GetAncestors() {
		res = append(res, itm.Hash)
	}
	return res
}

//GetAncestors retrieves graph ancestors
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

type HeaderMap map[common.Hash]*Header

func (hm HeaderMap) FromArray(headers []*Header) HeaderMap {
	for _, h := range headers {
		if h != nil {
			hm.Add(h)
		}
	}
	return hm
}

func (hm HeaderMap) RmEmpty() HeaderMap {
	for k, h := range hm {
		if h == nil {
			delete(hm, k)
		}
	}
	return hm
}

func (hm HeaderMap) ToArray() []*Header {
	arr := make([]*Header, len(hm))
	i := 0
	for _, h := range hm {
		arr[i] = h
		i++
	}
	return arr
}

func (hm HeaderMap) Add(header *Header) HeaderMap {
	if header == nil {
		log.Warn("HeaderMap::Add received nil Header")
		return hm
	}
	hm[header.Hash()] = header
	return hm
}

func (hm HeaderMap) Hashes() common.HashArray {
	hashes := make(common.HashArray, 0)
	for _, h := range hm {
		hashes = append(hashes, h.Hash())
	}
	return hashes
}

func (hm HeaderMap) GetMinHeight() uint64 {
	var res uint64 = 0
	for _, itm := range hm {
		if res == 0 {
			res = itm.Height
		} else if res > itm.Height {
			res = itm.Height
		}
	}
	return res
}

func (hm HeaderMap) GetMaxHeight() uint64 {
	var res uint64 = 0
	for _, itm := range hm {
		if res == 0 {
			res = itm.Height
		} else if res < itm.Height {
			res = itm.Height
		}
	}
	return res
}

func (hm HeaderMap) GetMaxTime() uint64 {
	maxtipsTs := uint64(0)
	for _, p := range hm {
		if p.Time > maxtipsTs {
			maxtipsTs = p.Time
		}
	}
	return maxtipsTs
}

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

func (hm HeaderMap) ParentHashes() common.HashArray {
	hashes := make(common.HashArray, 0)
	for _, h := range hm {
		hashes = hashes.Concat(h.ParentHashes)
		//hashes = append(hashes, h.Hash())
	}
	return hashes
}

//FinalizingSort sort in finalizing order
func (hm HeaderMap) FinalizingSort() common.HashArray {
	heightMap := map[uint64]common.HashArray{}
	for _, h := range hm {
		key := h.Height
		heightMap[key] = append(heightMap[key], h.Hash())
	}
	// sort by height
	keys := make(common.SorterAskU64, 0, len(heightMap))
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

type BlockMap map[common.Hash]*Block

func (bm BlockMap) ToArray() []*Block {
	arr := make([]*Block, len(bm))
	i := 0
	for _, h := range bm {
		arr[i] = h
		i++
	}
	return arr
}

func (bm BlockMap) Add(block *Block) BlockMap {
	if block == nil {
		log.Warn("BlockMap::Add received nil Header")
		return bm
	}
	bm[block.Hash()] = block
	return bm
}

func (bm BlockMap) Hashes() common.HashArray {
	hashes := make(common.HashArray, 0)
	for _, block := range bm {
		hashes = append(hashes, block.Hash())
	}
	return hashes
}
func (bm BlockMap) Headers() HeaderMap {
	headers := HeaderMap{}
	for _, block := range bm {
		headers[block.Hash()] = block.Header()
	}
	return headers
}

func (bm BlockMap) GetMinHeight() uint64 {
	return bm.Headers().GetMinHeight()
}
func (bm BlockMap) GetMaxHeight() uint64 {
	return bm.Headers().GetMaxHeight()
}

func (bm BlockMap) GetMaxTime() uint64 {
	return bm.Headers().GetMaxTime()
}

//AvgGasLimit calculate average GasLimit
func (bm BlockMap) AvgGasLimit() uint64 {
	return bm.Headers().AvgGasLimit()
}

//FinalizingSort sort in finalizing order
func (bm BlockMap) FinalizingSort() common.HashArray {
	return bm.Headers().FinalizingSort()
}

/********** SlotInfo **********/

type SlotInfo struct {
	Epoch     int64
	EpochSlot int64
	Period    int64
	NodeTime  int64
	Slot      int64
	Remain    int64
	Current   int64
}
