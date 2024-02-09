// Copyright 2016 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"os"
	"path/filepath"
	"testing"
)

var customGenesisTests = []struct {
	genesis string
	query   string
	result  string
}{
	// Genesis file with an empty chain configuration (ensure missing fields work)
	{
		genesis: `{
			"alloc"      : {
              "0xa7e558cc6efa1c41270ef4aa227b3dd6b4a3951e": {"balance": "1000000000000000000000000000000000"},
			  "0x6e9e76fa278190cfb2404e5923d3ccd7e8f6c51d": {"balance": "1000000000000000000000000000000000"},
			  "0xe43bb1b64fc7068d313d24d01d8ccca785b22c72": {"balance": "1000000000000000000000000000000000"},
			  "0x638bfa7e4fbfa457030ff5c8c3fca1741a0e745c": {"balance": "1000000000000000000000000000000000"},
			  "0xa7062A2Bd7270740f1d15ab70B3Dee189A87b6DE": {"balance": "1000000000000000000000000000000000"},
			  "0xdcdb1c4e7c168f33af47f142bbe5a4c692a6fb57": {"balance": "1000000000000000000000000000000000"},
			  "0x6CD106e7E631939c85fa15b764eCaE787a57C26f": {"balance": "1000000000000000000000000000000000"},
			  "0xa731e0897635af790cb4566dd3a713d8c9f12952": {"balance": "1000000000000000000000000000000000"}
            },
			"coinbase"   : "0x0000000000000000000000000000000000000000",
			"difficulty" : "0x20000",
			"extraData"  : "",
			"gasLimit"   : "0x2fefd8",
			"nonce"      : "0x0000000000001338",
			"parentHash" : "0x0000000000000000000000000000000000000000000000000000000000000000",
			"timestamp"  : "0x00",
			"config"     : {
              "chainId": 333777333,
			  "secondsPerSlot": 4,
			  "slotsPerEpoch": 8,
			  "forkSlotSubNet1": 20,
			  "validatorsPerSlot": 4,
			  "epochsPerEra": 10000000,
			  "transitionPeriod": 2,
			  "effectiveBalance": 3200
			}
		}`,
		query:  "eth.getBlock(0).nonce",
		result: "undefined",
	},
	// Genesis file with specific chain configurations
	{
		genesis: `{
			"alloc"      : {
              "0xa7e558cc6efa1c41270ef4aa227b3dd6b4a3951e": {"balance": "1000000000000000000000000000000000"},
			  "0x6e9e76fa278190cfb2404e5923d3ccd7e8f6c51d": {"balance": "1000000000000000000000000000000000"},
			  "0xe43bb1b64fc7068d313d24d01d8ccca785b22c72": {"balance": "1000000000000000000000000000000000"},
			  "0x638bfa7e4fbfa457030ff5c8c3fca1741a0e745c": {"balance": "1000000000000000000000000000000000"},
			  "0xa7062A2Bd7270740f1d15ab70B3Dee189A87b6DE": {"balance": "1000000000000000000000000000000000"},
			  "0xdcdb1c4e7c168f33af47f142bbe5a4c692a6fb57": {"balance": "1000000000000000000000000000000000"},
			  "0x6CD106e7E631939c85fa15b764eCaE787a57C26f": {"balance": "1000000000000000000000000000000000"},
			  "0xa731e0897635af790cb4566dd3a713d8c9f12952": {"balance": "1000000000000000000000000000000000"}
			},
			"coinbase"   : "0x0000000000000000000000000000000000000000",
			"difficulty" : "0x20000",
			"extraData"  : "",
			"gasLimit"   : "0x2fefd8",
			"nonce"      : "0x0000000000001339",
			"parentHash" : "0x0000000000000000000000000000000000000000000000000000000000000000",
			"timestamp"  : "0x00",
			"config"     : {
              "chainId": 333777333,
			  "secondsPerSlot": 4,
			  "slotsPerEpoch": 8,
			  "forkSlotSubNet1": 20,
			  "validatorsPerSlot": 4,
			  "epochsPerEra": 10000000,
			  "transitionPeriod": 2,
			  "effectiveBalance": 3200
			}
		}`,
		query:  "eth.getBlock(0).nonce",
		result: "undefined",
	},
}

// Tests that initializing Geth with a custom genesis block and chain definitions
// work properly.
func TestCustomGenesis(t *testing.T) {
	for i, tt := range customGenesisTests {
		tmpPath := initTmpDbWithGenesis(t)
		// Create a temporary data directory to use and inspect later
		defer os.RemoveAll(tmpPath)

		// Initialize the data directory with the custom genesis block
		json := filepath.Join(tmpPath, "genesis.json")
		if err := os.WriteFile(json, []byte(tt.genesis), 0600); err != nil {
			t.Fatalf("test %d: failed to write genesis file: %v", i, err)
		}
		runGeth(t, "--datadir", tmpPath, "init", json).WaitExit()

		// Query the custom genesis block
		geth := runGeth(t, "--networkid", "1337", "--syncmode=full", "--cache", "16",
			"--datadir", tmpPath, "--maxpeers", "0", "--port", "0",
			"--nodiscover", "--nat", "none", "--ipcdisable",
			"--exec", tt.query, "console")
		geth.ExpectRegexp(tt.result)
		geth.ExpectExit()
	}
}
