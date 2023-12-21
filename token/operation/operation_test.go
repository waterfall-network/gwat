package operation

// Marshaling and unmarshalling of operations in the package is implemented using Ethereum rlp encoding.
// All marshal and unmarshal methods of operations suppose that an encoding prefix has already handled.
// Byte encoding of the operation should be given to the methods without the prefix.
//
// The operations are implement using a philosophy of immutable data structures. Every method that returns
// a data field of an operation always make its copy before returning. That prevents situations when the
// operation structure can be mutated accidentally.

import (
	"testing"
)

const (
	InvalidPrefix = 0xF4
)

func TestGetOpCode(t *testing.T) {
	tests := []struct {
		input    []byte
		expected Code
		err      error
	}{
		{[]byte{Prefix, CreateCode}, CreateCode, nil},
		{[]byte{Prefix, TransferCode}, TransferCode, nil},
		{[]byte{Prefix, SetPriceCode}, SetPriceCode, nil},
		{[]byte{Prefix}, 0x00, ErrRawDataShort},
		{[]byte{InvalidPrefix, CreateCode}, 0x00, ErrPrefixNotValid},
	}

	for i, test := range tests {
		result, err := GetOpCode(test.input)
		if result != test.expected {
			t.Errorf("GetOpCode: Test case %d: expected %v, got %v", i, test.expected, result)
		}
		if err != test.err {
			t.Errorf("GetOpCode: Test case %d: expected %v, got %v", i, test.err, err)
		}
	}
}