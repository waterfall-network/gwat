package shuffle

import (
	"crypto/sha256"
	"reflect"
	"strconv"
	"testing"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/tests/testutils"
)

func TestShuffleCreators(t *testing.T) {
	indexes := make([]common.Address, testutils.RandomInt(10, 9999))
	for i := 0; i < len(indexes); i++ {
		indexes[i] = common.HexToAddress(strconv.Itoa(i))
	}

	input := make([]common.Address, len(indexes))
	copy(input, indexes)

	seed := sha256.Sum256(Bytes8(uint64(testutils.RandomInt(0, 9999))))

	shuffledList, err := ShuffleValidators(input, seed)
	if err != nil {
		t.Fatalf("unexpected error: %+v", err)
	}

	if reflect.DeepEqual(indexes, shuffledList) {
		t.Fatalf("got not shuffled list")
	}
}

func TestShuffleList(t *testing.T) {
	testInnerShuffleList(t, shuffleList, uint64(testutils.RandomInt(0, 9999)))
}

func TestUnshuffleList(t *testing.T) {
	testInnerShuffleList(t, unshuffleList, uint64(testutils.RandomInt(0, 9999)))
}

func testInnerShuffleList(t *testing.T, f func([]common.Address, common.Hash) ([]common.Address, error), epoch uint64) {
	validatorsCount := uint64(testutils.RandomInt(0, 9999))

	validators := make([]common.Address, validatorsCount)
	for i := 0; i < 100; i++ {
		validators[i] = common.HexToAddress(strconv.Itoa(i))
	}

	input := make([]common.Address, len(validators))
	copy(input, validators)

	seed := sha256.Sum256(Bytes8(epoch))

	shuffledList, err := f(input, seed)
	if err != nil {
		t.Fatalf("error while shuffling list: %v", err)
	}

	if reflect.DeepEqual(validators, shuffledList) {
		t.Fatalf("unexpected output: %v", shuffledList)
	}
}

func TestSwapOrNot(t *testing.T) {
	addr1 := common.HexToAddress(strconv.Itoa(1))
	addr2 := common.HexToAddress(strconv.Itoa(2))
	addr3 := common.HexToAddress(strconv.Itoa(3))
	input := []common.Address{addr1, addr2, addr3}
	buf := make([]byte, totalSize)

	tests := []struct {
		name           string
		expectedOutput []common.Address
		buf            []byte
		source         common.Hash
		byteV          byte
	}{
		{
			name:           "don`t swap elements",
			expectedOutput: []common.Address{addr1, addr2, addr3},
			buf:            make([]byte, totalSize),
			source:         common.Hash{},
			byteV:          byte(0),
		}, {
			name:           "swap elements",
			expectedOutput: []common.Address{addr1, addr3, addr2},
			buf:            make([]byte, totalSize),
			source:         common.BytesToHash([]byte{1, 2, 3}),
			byteV:          byte(4),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			swapOrNot(buf, test.byteV, 1, 2, input, test.source, CustomSHA256Hasher())
			if !reflect.DeepEqual(input, test.expectedOutput) {
				t.Errorf("expected output: %v, got: %v", test.expectedOutput, input)
			}
		})
	}
}
