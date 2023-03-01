package shuffle

import (
	"hash"
	"sync"

	"github.com/minio/sha256-simd"
	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
)

var sha256Pool = sync.Pool{New: func() interface{} {
	return sha256.New()
}}

// CustomSHA256Hasher returns a hash function that uses
// an enclosed hasher. This is not safe for concurrent
// use as the same hasher is being called throughout.
func CustomSHA256Hasher() func([]byte) common.Hash {
	hasher, ok := sha256Pool.Get().(hash.Hash)
	if !ok {
		hasher = sha256.New()
	} else {
		hasher.Reset()
	}
	var h common.Hash

	return func(data []byte) common.Hash {
		hasher.Write(data)
		hasher.Sum(h[:0])
		hasher.Reset()

		return h
	}
}
