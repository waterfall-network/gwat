// Copyright 2019 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package snapshot

import (
	"bytes"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
)

// binaryIterator is a simplistic iterator to step over the accounts or storage
// in a snapshot, which may or may not be composed of multiple layers. Performance
// wise this iterator is slow, it's meant for cross validating the fast one,
type binaryIterator struct {
	a               Iterator
	b               Iterator
	aDone           bool
	bDone           bool
	accountIterator bool
	k               common.Hash
	account         common.Hash
	fail            error
}

// Next steps the iterator forward one element, returning false if exhausted,
// or an error if iteration failed for some reason (e.g. root being iterated
// becomes stale and garbage collected).
func (it *binaryIterator) Next() bool {
	if it.aDone && it.bDone {
		return false
	}
first:
	if it.aDone {
		it.k = it.b.Hash()
		it.bDone = !it.b.Next()
		return true
	}
	if it.bDone {
		it.k = it.a.Hash()
		it.aDone = !it.a.Next()
		return true
	}
	nextA, nextB := it.a.Hash(), it.b.Hash()
	if diff := bytes.Compare(nextA[:], nextB[:]); diff < 0 {
		it.aDone = !it.a.Next()
		it.k = nextA
		return true
	} else if diff == 0 {
		// Now we need to advance one of them
		it.aDone = !it.a.Next()
		goto first
	}
	it.bDone = !it.b.Next()
	it.k = nextB
	return true
}

// Error returns any failure that occurred during iteration, which might have
// caused a premature iteration exit (e.g. snapshot stack becoming stale).
func (it *binaryIterator) Error() error {
	return it.fail
}

// Hash returns the hash of the account the iterator is currently at.
func (it *binaryIterator) Hash() common.Hash {
	return it.k
}

// Account returns the RLP encoded slim account the iterator is currently at, or
// nil if the iterated snapshot stack became stale (you can check Error after
// to see if it failed or not).
//
// Note the returned account is not a copy, please don't modify it.
func (it *binaryIterator) Account() []byte {
	if !it.accountIterator {
		return nil
	}
	// The topmost iterator must be `diffAccountIterator`
	blob, err := it.a.(*diffAccountIterator).layer.AccountRLP(it.k)
	if err != nil {
		it.fail = err
		return nil
	}
	return blob
}

// Slot returns the raw storage slot data the iterator is currently at, or
// nil if the iterated snapshot stack became stale (you can check Error after
// to see if it failed or not).
//
// Note the returned slot is not a copy, please don't modify it.
func (it *binaryIterator) Slot() []byte {
	if it.accountIterator {
		return nil
	}
	blob, err := it.a.(*diffStorageIterator).layer.Storage(it.account, it.k)
	if err != nil {
		it.fail = err
		return nil
	}
	return blob
}
