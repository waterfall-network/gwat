package shuffle

import (
	"crypto/sha256"
	"reflect"
	"testing"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/internal/token/testutils"
)

func TestShuffleCreators(t *testing.T) {
	indexes := make([]int, testutils.RandomInt(10, 9999))
	for i := 0; i < len(indexes); i++ {
		indexes[i] = i
	}

	input := make([]int, len(indexes))
	copy(input, indexes)

	seed := sha256.Sum256(Bytes32(uint64(testutils.RandomInt(0, 9999))))

	shuffledList, err := ShuffleCreators(input, seed)
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

func testInnerShuffleList(t *testing.T, f func([]int, [32]byte) ([]int, error), epoch uint64) {
	indexes := make([]int, uint64(testutils.RandomInt(0, 9999)))
	for i := 0; i < 100; i++ {
		indexes[i] = i
	}

	input := make([]int, len(indexes))
	copy(input, indexes)

	seed := sha256.Sum256(Bytes32(epoch))

	shuffledList, err := f(input, seed)
	if err != nil {
		t.Fatalf("error while shuffling list: %v", err)
	}

	if reflect.DeepEqual(indexes, shuffledList) {
		t.Fatalf("unexpected output: %v", shuffledList)
	}
}

func TestSwapOrNot(t *testing.T) {
	index1 := 1
	index2 := 2
	index3 := 3
	input := []int{index1, index2, index3}
	buf := make([]byte, totalSize)

	tests := []struct {
		name           string
		expectedOutput []int
		buf            []byte
		source         [32]byte
		byteV          byte
	}{
		{
			name:           "don`t swap elements",
			expectedOutput: []int{index1, index2, index3},
			buf:            make([]byte, totalSize),
			source:         [32]byte{},
			byteV:          byte(0),
		}, {
			name:           "swap elements",
			expectedOutput: []int{index1, index3, index2},
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
