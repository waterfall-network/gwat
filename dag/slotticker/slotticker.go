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

// File contains slot ticker and timer-related functions.
package slotticker

import (
	"time"
)

// The Ticker interface defines a type which can expose a
// receive-only channel firing slot events.
type Ticker interface {
	C() <-chan uint64
	Done()
}

// SlotTicker is a special ticker for the beacon chain block.
// The channel emits over the slot interval, and ensures that
// the ticks are in line with the genesis time. This means that
// the duration between the ticks and the genesis time are always a
// multiple of the slot duration.
// In addition, the channel returns the new slot number.
type SlotTicker struct {
	c    chan uint64
	done chan struct{}
}

// C returns the ticker channel. Call Cancel afterwards to ensure
// that the goroutine exits cleanly.
func (s *SlotTicker) C() <-chan uint64 {
	return s.c
}

// Done should be called to clean up the ticker.
func (s *SlotTicker) Done() {
	go func() {
		s.done <- struct{}{}
	}()
}

// NewSlotTicker starts and returns a new SlotTicker instance.
func NewSlotTicker(genesisTime time.Time, secondsPerSlot uint64) *SlotTicker {
	if genesisTime.IsZero() {
		panic("zero genesis time")
	}
	ticker := &SlotTicker{
		c:    make(chan uint64),
		done: make(chan struct{}),
	}
	ticker.start(genesisTime, secondsPerSlot, Since, Until, time.After)
	return ticker
}

// NewSlotTickerWithOffset starts and returns a SlotTicker instance that allows a offset of time from genesis,
// entering a offset greater than secondsPerSlot is not allowed.
func NewSlotTickerWithOffset(genesisTime time.Time, offset time.Duration, secondsPerSlot uint64) *SlotTicker {
	if genesisTime.Unix() == 0 {
		panic("zero genesis time")
	}
	if offset > time.Duration(secondsPerSlot)*time.Second {
		panic("invalid ticker offset")
	}
	ticker := &SlotTicker{
		c:    make(chan uint64),
		done: make(chan struct{}),
	}
	ticker.start(genesisTime.Add(offset), secondsPerSlot, Since, Until, time.After)
	return ticker
}

func (s *SlotTicker) start(
	genesisTime time.Time,
	secondsPerSlot uint64,
	since, until func(time.Time) time.Duration,
	after func(time.Duration) <-chan time.Time) {
	d := time.Duration(secondsPerSlot) * time.Second
	go func() {
		sinceGenesis := since(genesisTime)

		var nextTickTime time.Time
		var slot uint64
		if sinceGenesis < d {
			// Handle when the current time is before the genesis time.
			nextTickTime = genesisTime
			slot = 0
		} else {
			nextTick := sinceGenesis.Truncate(d) + d
			nextTickTime = genesisTime.Add(nextTick)
			slot = uint64(nextTick / d)
		}

		for {
			waitTime := until(nextTickTime)
			select {
			case <-after(waitTime):
				s.c <- slot
				slot++
				nextTickTime = nextTickTime.Add(d)
			case <-s.done:
				return
			}
		}
	}()
}

// Since returns the duration since t.
func Since(t time.Time) time.Duration {
	return Now().Sub(t)
}

// Until returns the duration until t.
func Until(t time.Time) time.Duration {
	return t.Sub(Now())
}

// Now returns the current local time.
func Now() time.Time {
	return time.Now()
}
