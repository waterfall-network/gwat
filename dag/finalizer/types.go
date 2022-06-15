package finalizer

import (
	"encoding/binary"
	"github.com/waterfall-foundation/gwat/common"
)

/********** NrHashMap  **********/

// NrHashMap represents map of finalization number to block hash
type NrHashMap map[uint64]*common.Hash

//GetMinNr search the min Nr
func (nhm *NrHashMap) GetMinNr() *uint64 {
	if len(*nhm) == 0 {
		return nil
	}
	var res uint64 = 0
	for k := range *nhm {
		if res == 0 {
			res = k
		} else if res > k {
			res = k
		}
	}
	return &res
}

//GetMaxNr search the max Nr
func (nhm *NrHashMap) GetMaxNr() *uint64 {
	if len(*nhm) == 0 {
		return nil
	}
	var res uint64 = 0
	for k := range *nhm {
		if res == 0 {
			res = k
		} else if res < k {
			res = k
		}
	}
	return &res
}

// HasGap check gap in order of nrs
func (nhm *NrHashMap) HasGap() bool {
	minNr := nhm.GetMinNr()
	if minNr == nil {
		return false
	}
	maxNr := nhm.GetMaxNr()
	for i := *minNr; i <= *maxNr; i++ {
		if (*nhm)[i] == nil {
			return true
		}
	}
	return false
}

// GetHashes retrieves all hashes
func (nhm *NrHashMap) GetHashes() *common.HashArray {
	if len(*nhm) == 0 {
		return nil
	}
	res := common.HashArray{}
	for _, h := range *nhm {
		res = append(res, *h)
	}
	return &res
}

// Copy create copy
func (nhm *NrHashMap) Copy() *NrHashMap {
	if len(*nhm) == 0 {
		return nil
	}
	res := NrHashMap{}
	for k, v := range *nhm {
		res[k] = v
	}
	return &res
}

// ToBytes encodes the NrHashMap instance
// to byte representation.
func (nhm *NrHashMap) ToBytes() []byte {
	res := []byte{}
	for nr, hash := range *nhm {
		bNr := make([]byte, 8)
		binary.BigEndian.PutUint64(bNr, nr)
		res = append(res, bNr...)
		if hash == nil {
			empty := common.Hash{}
			hash = &empty
		}
		res = append(res, hash.Bytes()...)
	}
	return res
}

// SetBytes restores NrHashMap instance from byte representation.
func (nhm *NrHashMap) SetBytes(data []byte) *NrHashMap {
	const (
		lenNr   int = 8
		lenHash int = common.HashLength
		lenItm  int = lenNr + lenHash
	)
	for k := range *nhm {
		delete(*nhm, k)
	}
	if data == nil {
		return nil
	}
	if len(data)%lenItm != 0 {
		return nil
	}
	lenmap := len(data) / lenItm
	for i := 0; i < lenmap; i++ {
		itmStart := i * lenItm
		itmEnd := itmStart + lenItm
		itm := data[itmStart:itmEnd]
		start := 0
		end := 8
		itmNr := binary.BigEndian.Uint64(itm[start:end])
		start = end
		end += lenHash
		itmHash := common.BytesToHash(itm[start:end])
		(*nhm)[itmNr] = &itmHash
		if itmHash == (common.Hash{}) {
			(*nhm)[itmNr] = nil
		}
	}
	return nhm
}
