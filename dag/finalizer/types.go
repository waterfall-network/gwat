package finalizer

import (
	"github.com/ethereum/go-ethereum/common"
)

/********** NrHashMap  **********/
type NrHashMap map[uint64]*common.Hash

//GetMinNr search the min Nr
func (nhm *NrHashMap) GetMinNr() *uint64 {
	if len(*nhm) == 0 {
		return nil
	}
	var res uint64 = 0
	for k, _ := range *nhm {
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
	for k, _ := range *nhm {
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
