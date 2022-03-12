// Copyright 2018 The go-ethereum Authors
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

package ethash

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/internal/testlog"
	"github.com/ethereum/go-ethereum/log"
)

// Tests whether remote HTTP servers are correctly notified of new work.
func TestRemoteNotify(t *testing.T) {
	// Start a simple web server to capture notifications.
	sink := make(chan [3]string)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		blob, err := ioutil.ReadAll(req.Body)
		if err != nil {
			t.Errorf("failed to read miner notification: %v", err)
		}
		var work [3]string
		if err := json.Unmarshal(blob, &work); err != nil {
			t.Errorf("failed to unmarshal miner notification: %v", err)
		}
		sink <- work
	}))
	defer server.Close()

	// Create the custom ethash engine.
	ethash := NewTester([]string{server.URL}, false)
	defer ethash.Close()

	// Stream a work task and ensure the notification bubbles out.
	nr := uint64(1)
	header := &types.Header{Number: &nr}
	block := types.NewBlockWithHeader(header)

	ethash.Seal(nil, block, nil, nil, nil)
	select {
	case work := <-sink:
		if want := ethash.SealHash(header).Hex(); work[0] != want {
			t.Errorf("work packet hash mismatch: have %s, want %s", work[0], want)
		}
		if want := common.BytesToHash(SeedHash(header.Nr())).Hex(); work[1] != want {
			t.Errorf("work packet seed mismatch: have %s, want %s", work[1], want)
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("notification timed out")
	}
}

// Tests whether remote HTTP servers are correctly notified of new work. (Full pending block body / --miner.notify.full)
func TestRemoteNotifyFull(t *testing.T) {
	// Start a simple web server to capture notifications.
	sink := make(chan map[string]interface{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		blob, err := ioutil.ReadAll(req.Body)
		if err != nil {
			t.Errorf("failed to read miner notification: %v", err)
		}
		var work map[string]interface{}
		if err := json.Unmarshal(blob, &work); err != nil {
			t.Errorf("failed to unmarshal miner notification: %v", err)
		}
		sink <- work
	}))
	defer server.Close()

	// Create the custom ethash engine.
	config := Config{
		PowMode:    ModeTest,
		NotifyFull: true,
		Log:        testlog.Logger(t, log.LvlWarn),
	}
	ethash := New(config, []string{server.URL}, false)
	defer ethash.Close()

	// Stream a work task and ensure the notification bubbles out.
	nr := uint64(1)
	header := &types.Header{Number: &nr}
	block := types.NewBlockWithHeader(header)

	ethash.Seal(nil, block, nil, nil, nil)
	select {
	case work := <-sink:
		if want := "0x" + strconv.FormatUint(header.Nr(), 16); work["number"] != want {
			t.Errorf("pending block number mismatch: have %v, want %v", work["number"], want)
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("notification timed out")
	}
}

// Tests that pushing work packages fast to the miner doesn't cause any data race
// issues in the notifications.
func TestRemoteMultiNotify(t *testing.T) {
	// Start a simple web server to capture notifications.
	sink := make(chan [3]string, 64)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		blob, err := ioutil.ReadAll(req.Body)
		if err != nil {
			t.Errorf("failed to read miner notification: %v", err)
		}
		var work [3]string
		if err := json.Unmarshal(blob, &work); err != nil {
			t.Errorf("failed to unmarshal miner notification: %v", err)
		}
		sink <- work
	}))
	defer server.Close()

	// Create the custom ethash engine.
	ethash := NewTester([]string{server.URL}, false)
	ethash.config.Log = testlog.Logger(t, log.LvlWarn)
	defer ethash.Close()

	// Provide a results reader.
	// Otherwise the unread results will be logged asynchronously
	// and this can happen after the test is finished, causing a panic.
	results := make(chan *types.Block, cap(sink))

	// Stream a lot of work task and ensure all the notifications bubble out.
	for i := 0; i < cap(sink); i++ {
		nr := uint64(1)
		header := &types.Header{Number: &nr}
		block := types.NewBlockWithHeader(header)
		ethash.Seal(nil, block, nil, results, nil)
	}

	for i := 0; i < cap(sink); i++ {
		select {
		case <-sink:
			<-results
		case <-time.After(10 * time.Second):
			t.Fatalf("notification %d timed out", i)
		}
	}
}

// Tests that pushing work packages fast to the miner doesn't cause any data race
// issues in the notifications. Full pending block body / --miner.notify.full)
func TestRemoteMultiNotifyFull(t *testing.T) {
	// Start a simple web server to capture notifications.
	sink := make(chan map[string]interface{}, 64)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		blob, err := ioutil.ReadAll(req.Body)
		if err != nil {
			t.Errorf("failed to read miner notification: %v", err)
		}
		var work map[string]interface{}
		if err := json.Unmarshal(blob, &work); err != nil {
			t.Errorf("failed to unmarshal miner notification: %v", err)
		}
		sink <- work
	}))
	defer server.Close()

	// Create the custom ethash engine.
	config := Config{
		PowMode:    ModeTest,
		NotifyFull: true,
		Log:        testlog.Logger(t, log.LvlWarn),
	}
	ethash := New(config, []string{server.URL}, false)
	defer ethash.Close()

	// Provide a results reader.
	// Otherwise the unread results will be logged asynchronously
	// and this can happen after the test is finished, causing a panic.
	results := make(chan *types.Block, cap(sink))

	// Stream a lot of work task and ensure all the notifications bubble out.
	for i := 0; i < cap(sink); i++ {
		nr := uint64(1)
		header := &types.Header{Number: &nr}
		block := types.NewBlockWithHeader(header)
		ethash.Seal(nil, block, nil, results, nil)
	}

	for i := 0; i < cap(sink); i++ {
		select {
		case <-sink:
			<-results
		case <-time.After(10 * time.Second):
			t.Fatalf("notification %d timed out", i)
		}
	}
}
