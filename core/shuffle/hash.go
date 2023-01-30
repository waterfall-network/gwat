package shuffle

import (
	"hash"
	"sync"

	"github.com/minio/sha256-simd"
)

var sha256Pool = sync.Pool{New: func() interface{} {
	return sha256.New()
}}

// CustomSHA256Hasher returns a hash function that uses
// an enclosed hasher. This is not safe for concurrent
// use as the same hasher is being called throughout.
func CustomSHA256Hasher() func([]byte) [32]byte {
	hasher, ok := sha256Pool.Get().(hash.Hash)
	if !ok {
		hasher = sha256.New()
	} else {
		hasher.Reset()
	}
	var h [32]byte

	return func(data []byte) [32]byte {
		hasher.Write(data)
		hasher.Sum(h[:0])
		hasher.Reset()

		return h
	}
}
