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

package shuffle

import (
	"encoding/binary"
	"fmt"
	"time"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/log"
)

const seedSize = int8(32)
const roundSize = int8(1)
const positionWindowSize = int8(4)
const pivotViewSize = seedSize + roundSize
const totalSize = seedSize + roundSize + positionWindowSize
const shuffleRoundCount = uint8(90)

var maxShuffleListSize uint64 = 1 << 40

// ShuffleValidators returns list of shuffled addresses in a pseudorandom permutation `p` of `0...list_size - 1` with “seed“ as entropy.
func ShuffleValidators(validators []common.Address, seed common.Hash) ([]common.Address, error) {
	start := time.Now()

	shuffledList, err := unshuffleList(validators, seed)
	if err != nil {
		return nil, err
	}

	log.Info("^^^^^^^^^^^^ TIME",
		"elapsed", common.PrettyDuration(time.Since(start)),
		"func:", "ShuffleValidators",
	)
	return shuffledList, nil
}

func unshuffleList(validators []common.Address, seed common.Hash) ([]common.Address, error) {
	return innerShuffleList(validators, seed, false /* un-shuffle */)
}

func innerShuffleList(validators []common.Address, seed common.Hash, shuffl bool) ([]common.Address, error) {
	if len(validators) <= 1 {
		return validators, nil
	}

	if uint64(len(validators)) > maxShuffleListSize {
		return nil, fmt.Errorf("list size %d out of bounds", len(validators))
	}

	rounds := shuffleRoundCount
	hashfunc := CustomSHA256Hasher()
	if rounds == 0 {
		return validators, nil
	}

	listSize := uint64(len(validators))
	buf := make([]byte, totalSize)
	r := uint8(0)
	if !shuffl {
		r = rounds - 1
	}

	copy(buf[:seedSize], seed[:])
	for {
		buf[seedSize] = r
		ph := hashfunc(buf[:pivotViewSize])
		pivot := FromBytes8(ph[:8]) % listSize
		mirror := (pivot + 1) >> 1
		binary.LittleEndian.PutUint32(buf[pivotViewSize:], uint32(pivot>>8))
		source := hashfunc(buf)
		byteV := source[(pivot&0xff)>>3]
		for i, j := uint64(0), pivot; i < mirror; i, j = i+1, j-1 {
			byteV, source = swapOrNot(buf, byteV, i, j, validators, source, hashfunc)
		}
		// Now repeat, but for the part after the pivot.
		mirror = (pivot + listSize + 1) >> 1
		end := listSize - 1
		binary.LittleEndian.PutUint32(buf[pivotViewSize:], uint32(end>>8))
		source = hashfunc(buf)
		byteV = source[(end&0xff)>>3]
		for i, j := pivot+1, end; i < mirror; i, j = i+1, j-1 {
			byteV, source = swapOrNot(buf, byteV, i, j, validators, source, hashfunc)
		}
		if shuffl {
			r++
			if r == rounds {
				break
			}
		} else {
			if r == 0 {
				break
			}
			r--
		}
	}

	return validators, nil
}

// swapOrNot describes the main algorithm behind the shuffle where we swap bytes in the inputted value
// depending on if the conditions are met.
func swapOrNot(
	buf []byte,
	byteV byte,
	i, j uint64,
	validators []common.Address,
	source common.Hash,
	hashFunc func([]byte) common.Hash,
) (byte, common.Hash) {
	if j&0xff == 0xff {
		binary.LittleEndian.PutUint32(buf[pivotViewSize:], uint32(j>>8))
		source = hashFunc(buf)
	}

	if j&0x7 == 0x7 {
		byteV = source[(j&0xff)>>3]
	}

	bitV := (byteV >> (j & 0x7)) & 0x1

	if bitV == 1 {
		validators[i], validators[j] = validators[j], validators[i]
	}

	return byteV, source
}
