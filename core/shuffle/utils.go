package shuffle

import "encoding/binary"

// FromBytes8 returns an integer which is stored in the little-endian format(8, 'little')
// from a byte array.
func FromBytes8(x []byte) uint64 {
	if len(x) < 8 {
		return 0
	}
	return binary.LittleEndian.Uint64(x)
}

func Bytes32(x uint64) []byte {
	bytes := make([]byte, 32)
	binary.LittleEndian.PutUint64(bytes, x)
	return bytes
}
