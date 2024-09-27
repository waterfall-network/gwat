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
