package shuffle

import (
	"encoding/binary"
	"fmt"
)

const seedSize = int8(32)
const roundSize = int8(1)
const positionWindowSize = int8(4)
const pivotViewSize = seedSize + roundSize
const totalSize = seedSize + roundSize + positionWindowSize
const shuffleRoundCount = uint8(90)

var maxShuffleListSize uint64 = 1 << 40

// ShuffleValidators returns list of shuffled addresses in a pseudorandom permutation `p` of `0...list_size - 1` with “seed“ as entropy.
func ShuffleValidators(indexes []uint64, seed [32]byte) ([]uint64, error) {
	shuffledList, err := unshuffleList(indexes, seed)
	if err != nil {
		return nil, err
	}

	return shuffledList, nil
}

func shuffleList(indexes []uint64, seed [32]byte) ([]uint64, error) {
	return innerShuffleList(indexes, seed, true /* shuffle */)
}

func unshuffleList(indexes []uint64, seed [32]byte) ([]uint64, error) {
	return innerShuffleList(indexes, seed, false /* un-shuffle */)
}

func innerShuffleList(input []uint64, seed [32]byte, shuffl bool) ([]uint64, error) {
	if len(input) <= 1 {
		return input, nil
	}

	if uint64(len(input)) > maxShuffleListSize {
		return nil, fmt.Errorf("list size %d out of bounds", len(input))
	}

	rounds := shuffleRoundCount
	hashfunc := CustomSHA256Hasher()
	if rounds == 0 {
		return input, nil
	}

	listSize := uint64(len(input))
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
			byteV, source = swapOrNot(buf, byteV, i, j, input, source, hashfunc)
		}
		// Now repeat, but for the part after the pivot.
		mirror = (pivot + listSize + 1) >> 1
		end := listSize - 1
		binary.LittleEndian.PutUint32(buf[pivotViewSize:], uint32(end>>8))
		source = hashfunc(buf)
		byteV = source[(end&0xff)>>3]
		for i, j := pivot+1, end; i < mirror; i, j = i+1, j-1 {
			byteV, source = swapOrNot(buf, byteV, i, j, input, source, hashfunc)
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

	return input, nil
}

// swapOrNot describes the main algorithm behind the shuffle where we swap bytes in the inputted value
// depending on if the conditions are met.
func swapOrNot(
	buf []byte,
	byteV byte,
	i, j uint64,
	input []uint64,
	source [32]byte,
	hashFunc func([]byte) [32]byte,
) (byte, [32]byte) {
	if j&0xff == 0xff {
		binary.LittleEndian.PutUint32(buf[pivotViewSize:], uint32(j>>8))
		source = hashFunc(buf)
	}

	if j&0x7 == 0x7 {
		byteV = source[(j&0xff)>>3]
	}

	bitV := (byteV >> (j & 0x7)) & 0x1

	if bitV == 1 {
		input[i], input[j] = input[j], input[i]
	}

	return byteV, source
}
