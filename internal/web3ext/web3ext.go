// Copyright 2015 The go-ethereum Authors
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

// package web3ext contains geth specific web3.js extensions.
package web3ext

var Modules = map[string]string{
	"admin":    AdminJs,
	"ethash":   EthashJs,
	"debug":    DebugJs,
	"eth":      EthJs,
	"miner":    MinerJs,
	"net":      NetJs,
	"personal": PersonalJs,
	"rpc":      RpcJs,
	"txpool":   TxpoolJs,
	"les":      LESJs,
	"vflux":    VfluxJs,
	"dag":      DagJs,
	"wat":      WatJs,
}

const EthashJs = `
web3._extend({
	property: 'ethash',
	methods: [
		new web3._extend.Method({
			name: 'getWork',
			call: 'ethash_getWork',
			params: 0
		}),
		new web3._extend.Method({
			name: 'getHashrate',
			call: 'ethash_getHashrate',
			params: 0
		}),
		new web3._extend.Method({
			name: 'submitWork',
			call: 'ethash_submitWork',
			params: 3,
		}),
		new web3._extend.Method({
			name: 'submitHashrate',
			call: 'ethash_submitHashrate',
			params: 2,
		}),
	]
});
`

const AdminJs = `
web3._extend({
	property: 'admin',
	methods: [
		new web3._extend.Method({
			name: 'addPeer',
			call: 'admin_addPeer',
			params: 1
		}),
		new web3._extend.Method({
			name: 'removePeer',
			call: 'admin_removePeer',
			params: 1
		}),
		new web3._extend.Method({
			name: 'addTrustedPeer',
			call: 'admin_addTrustedPeer',
			params: 1
		}),
		new web3._extend.Method({
			name: 'removeTrustedPeer',
			call: 'admin_removeTrustedPeer',
			params: 1
		}),
		new web3._extend.Method({
			name: 'exportChain',
			call: 'admin_exportChain',
			params: 3,
			inputFormatter: [null, null, null]
		}),
		new web3._extend.Method({
			name: 'importChain',
			call: 'admin_importChain',
			params: 1
		}),
		new web3._extend.Method({
			name: 'sleepBlocks',
			call: 'admin_sleepBlocks',
			params: 2
		}),
		new web3._extend.Method({
			name: 'startHTTP',
			call: 'admin_startHTTP',
			params: 5,
			inputFormatter: [null, null, null, null, null]
		}),
		new web3._extend.Method({
			name: 'stopHTTP',
			call: 'admin_stopHTTP'
		}),
		// This method is deprecated.
		new web3._extend.Method({
			name: 'startRPC',
			call: 'admin_startRPC',
			params: 5,
			inputFormatter: [null, null, null, null, null]
		}),
		// This method is deprecated.
		new web3._extend.Method({
			name: 'stopRPC',
			call: 'admin_stopRPC'
		}),
		new web3._extend.Method({
			name: 'startWS',
			call: 'admin_startWS',
			params: 4,
			inputFormatter: [null, null, null, null]
		}),
		new web3._extend.Method({
			name: 'stopWS',
			call: 'admin_stopWS'
		}),
	],
	properties: [
		new web3._extend.Property({
			name: 'nodeInfo',
			getter: 'admin_nodeInfo'
		}),
		new web3._extend.Property({
			name: 'peers',
			getter: 'admin_peers'
		}),
		new web3._extend.Property({
			name: 'datadir',
			getter: 'admin_datadir'
		}),
	]
});
`

const DebugJs = `
web3._extend({
	property: 'debug',
	methods: [
		new web3._extend.Method({
			name: 'accountRange',
			call: 'debug_accountRange',
			params: 6,
			inputFormatter: [web3._extend.formatters.inputDefaultBlockNumberFormatter, null, null, null, null, null],
		}),
		new web3._extend.Method({
			name: 'printBlock',
			call: 'debug_printBlock',
			params: 1,
			outputFormatter: console.log
		}),
		new web3._extend.Method({
			name: 'getHeaderRlp',
			call: 'debug_getHeaderRlp',
			params: 1
		}),
		new web3._extend.Method({
			name: 'getBlockRlp',
			call: 'debug_getBlockRlp',
			params: 1
		}),
		new web3._extend.Method({
			name: 'setHead',
			call: 'debug_setHead',
			params: 1
		}),
		new web3._extend.Method({
			name: 'seedHash',
			call: 'debug_seedHash',
			params: 1
		}),
		new web3._extend.Method({
			name: 'dumpBlock',
			call: 'debug_dumpBlock',
			params: 1,
			inputFormatter: [web3._extend.formatters.inputBlockNumberFormatter]
		}),
		new web3._extend.Method({
			name: 'chaindbProperty',
			call: 'debug_chaindbProperty',
			params: 1,
			outputFormatter: console.log
		}),
		new web3._extend.Method({
			name: 'chaindbCompact',
			call: 'debug_chaindbCompact',
		}),
		new web3._extend.Method({
			name: 'verbosity',
			call: 'debug_verbosity',
			params: 1
		}),
		new web3._extend.Method({
			name: 'vmodule',
			call: 'debug_vmodule',
			params: 1
		}),
		new web3._extend.Method({
			name: 'backtraceAt',
			call: 'debug_backtraceAt',
			params: 1,
		}),
		new web3._extend.Method({
			name: 'stacks',
			call: 'debug_stacks',
			params: 1,
			inputFormatter: [null],
			outputFormatter: console.log
		}),
		new web3._extend.Method({
			name: 'freeOSMemory',
			call: 'debug_freeOSMemory',
			params: 0,
		}),
		new web3._extend.Method({
			name: 'setGCPercent',
			call: 'debug_setGCPercent',
			params: 1,
		}),
		new web3._extend.Method({
			name: 'memStats',
			call: 'debug_memStats',
			params: 0,
		}),
		new web3._extend.Method({
			name: 'gcStats',
			call: 'debug_gcStats',
			params: 0,
		}),
		new web3._extend.Method({
			name: 'cpuProfile',
			call: 'debug_cpuProfile',
			params: 2
		}),
		new web3._extend.Method({
			name: 'startCPUProfile',
			call: 'debug_startCPUProfile',
			params: 1
		}),
		new web3._extend.Method({
			name: 'stopCPUProfile',
			call: 'debug_stopCPUProfile',
			params: 0
		}),
		new web3._extend.Method({
			name: 'goTrace',
			call: 'debug_goTrace',
			params: 2
		}),
		new web3._extend.Method({
			name: 'startGoTrace',
			call: 'debug_startGoTrace',
			params: 1
		}),
		new web3._extend.Method({
			name: 'stopGoTrace',
			call: 'debug_stopGoTrace',
			params: 0
		}),
		new web3._extend.Method({
			name: 'blockProfile',
			call: 'debug_blockProfile',
			params: 2
		}),
		new web3._extend.Method({
			name: 'setBlockProfileRate',
			call: 'debug_setBlockProfileRate',
			params: 1
		}),
		new web3._extend.Method({
			name: 'writeBlockProfile',
			call: 'debug_writeBlockProfile',
			params: 1
		}),
		new web3._extend.Method({
			name: 'mutexProfile',
			call: 'debug_mutexProfile',
			params: 2
		}),
		new web3._extend.Method({
			name: 'setMutexProfileFraction',
			call: 'debug_setMutexProfileFraction',
			params: 1
		}),
		new web3._extend.Method({
			name: 'writeMutexProfile',
			call: 'debug_writeMutexProfile',
			params: 1
		}),
		new web3._extend.Method({
			name: 'writeMemProfile',
			call: 'debug_writeMemProfile',
			params: 1
		}),
		new web3._extend.Method({
			name: 'traceBlock',
			call: 'debug_traceBlock',
			params: 2,
			inputFormatter: [null, null]
		}),
		new web3._extend.Method({
			name: 'traceBlockFromFile',
			call: 'debug_traceBlockFromFile',
			params: 2,
			inputFormatter: [null, null]
		}),
		new web3._extend.Method({
			name: 'intermediateRoots',
			call: 'debug_intermediateRoots',
			params: 2,
			inputFormatter: [null, null]
		}),
		new web3._extend.Method({
			name: 'standardTraceBlockToFile',
			call: 'debug_standardTraceBlockToFile',
			params: 2,
			inputFormatter: [null, null]
		}),
		new web3._extend.Method({
			name: 'traceBlockByNumber',
			call: 'debug_traceBlockByNumber',
			params: 2,
			inputFormatter: [web3._extend.formatters.inputBlockNumberFormatter, null]
		}),
		new web3._extend.Method({
			name: 'traceBlockByHash',
			call: 'debug_traceBlockByHash',
			params: 2,
			inputFormatter: [null, null]
		}),
		new web3._extend.Method({
			name: 'traceTransaction',
			call: 'debug_traceTransaction',
			params: 2,
			inputFormatter: [null, null]
		}),
		new web3._extend.Method({
			name: 'traceCall',
			call: 'debug_traceCall',
			params: 3,
			inputFormatter: [null, null, null]
		}),
		new web3._extend.Method({
			name: 'preimage',
			call: 'debug_preimage',
			params: 1,
			inputFormatter: [null]
		}),
		new web3._extend.Method({
			name: 'storageRangeAt',
			call: 'debug_storageRangeAt',
			params: 5,
		}),
		new web3._extend.Method({
			name: 'getModifiedAccountsByNumber',
			call: 'debug_getModifiedAccountsByNumber',
			params: 2,
			inputFormatter: [null, null],
		}),
		new web3._extend.Method({
			name: 'getModifiedAccountsByHash',
			call: 'debug_getModifiedAccountsByHash',
			params: 2,
			inputFormatter:[null, null],
		}),
		new web3._extend.Method({
			name: 'freezeClient',
			call: 'debug_freezeClient',
			params: 1,
		}),
		new web3._extend.Method({
			name: 'getAccessibleState',
			call: 'debug_getAccessibleState',
			params: 2,
			inputFormatter:[web3._extend.formatters.inputBlockNumberFormatter, web3._extend.formatters.inputBlockNumberFormatter],
		}),
	],
	properties: []
});
`

const EthJs = `
web3._extend({
	property: 'eth',
	methods: [
		new web3._extend.Method({
			name: 'chainId',
			call: 'eth_chainId',
			params: 0
		}),
		new web3._extend.Method({
			name: 'sign',
			call: 'eth_sign',
			params: 2,
			inputFormatter: [web3._extend.formatters.inputAddressFormatter, null]
		}),
		new web3._extend.Method({
			name: 'resend',
			call: 'eth_resend',
			params: 3,
			inputFormatter: [web3._extend.formatters.inputTransactionFormatter, web3._extend.utils.fromDecimal, web3._extend.utils.fromDecimal]
		}),
		new web3._extend.Method({
			name: 'signTransaction',
			call: 'eth_signTransaction',
			params: 1,
			inputFormatter: [web3._extend.formatters.inputTransactionFormatter]
		}),
		new web3._extend.Method({
			name: 'estimateGas',
			call: 'eth_estimateGas',
			params: 2,
			inputFormatter: [web3._extend.formatters.inputCallFormatter, web3._extend.formatters.inputBlockNumberFormatter],
			outputFormatter: web3._extend.utils.toDecimal
		}),
		new web3._extend.Method({
			name: 'submitTransaction',
			call: 'eth_submitTransaction',
			params: 1,
			inputFormatter: [web3._extend.formatters.inputTransactionFormatter]
		}),
		new web3._extend.Method({
			name: 'fillTransaction',
			call: 'eth_fillTransaction',
			params: 1,
			inputFormatter: [web3._extend.formatters.inputTransactionFormatter]
		}),
		new web3._extend.Method({
			name: 'getHeaderByNumber',
			call: 'eth_getHeaderByNumber',
			params: 1,
			inputFormatter: [web3._extend.formatters.inputBlockNumberFormatter]
		}),
		new web3._extend.Method({
			name: 'getHeaderByHash',
			call: 'eth_getHeaderByHash',
			params: 1
		}),
		new web3._extend.Method({
			name: 'getBlockByNumber',
			call: 'eth_getBlockByNumber',
			params: 2,
			inputFormatter: [web3._extend.formatters.inputBlockNumberFormatter, function (val) { return !!val; }]
		}),
		new web3._extend.Method({
			name: 'getBlockByHash',
			call: 'eth_getBlockByHash',
			params: 2,
			inputFormatter: [null, function (val) { return !!val; }]
		}),
		new web3._extend.Method({
			name: 'getRawTransaction',
			call: 'eth_getRawTransactionByHash',
			params: 1
		}),
		new web3._extend.Method({
			name: 'getRawTransactionFromBlock',
			call: function(args) {
				return (web3._extend.utils.isString(args[0]) && args[0].indexOf('0x') === 0) ? 'eth_getRawTransactionByBlockHashAndIndex' : 'eth_getRawTransactionByBlockNumberAndIndex';
			},
			params: 2,
			inputFormatter: [web3._extend.formatters.inputBlockNumberFormatter, web3._extend.utils.toHex]
		}),
		new web3._extend.Method({
			name: 'getProof',
			call: 'eth_getProof',
			params: 3,
			inputFormatter: [web3._extend.formatters.inputAddressFormatter, null, web3._extend.formatters.inputBlockNumberFormatter]
		}),
		new web3._extend.Method({
			name: 'createAccessList',
			call: 'eth_createAccessList',
			params: 2,
			inputFormatter: [null, web3._extend.formatters.inputBlockNumberFormatter],
		}),
		new web3._extend.Method({
			name: 'feeHistory',
			call: 'eth_feeHistory',
			params: 3,
			inputFormatter: [null, web3._extend.formatters.inputBlockNumberFormatter, null]
		}),
	],
	properties: [
		new web3._extend.Property({
			name: 'pendingTransactions',
			getter: 'eth_pendingTransactions',
			outputFormatter: function(txs) {
				var formatted = [];
				for (var i = 0; i < txs.length; i++) {
					formatted.push(web3._extend.formatters.outputTransactionFormatter(txs[i]));
					formatted[i].blockHash = null;
				}
				return formatted;
			}
		}),
		new web3._extend.Property({
			name: 'maxPriorityFeePerGas',
			getter: 'eth_maxPriorityFeePerGas',
			outputFormatter: web3._extend.utils.toBigNumber
		}),
	]
});
`

const MinerJs = `
web3._extend({
	property: 'miner',
	methods: [
		new web3._extend.Method({
			name: 'start',
			call: 'miner_start',
			params: 1,
			inputFormatter: [null]
		}),
		new web3._extend.Method({
			name: 'stop',
			call: 'miner_stop'
		}),
		new web3._extend.Method({
			name: 'setEtherbase',
			call: 'miner_setEtherbase',
			params: 1,
			inputFormatter: [web3._extend.formatters.inputAddressFormatter]
		}),
		new web3._extend.Method({
			name: 'setExtra',
			call: 'miner_setExtra',
			params: 1
		}),
		new web3._extend.Method({
			name: 'setGasPrice',
			call: 'miner_setGasPrice',
			params: 1,
			inputFormatter: [web3._extend.utils.fromDecimal]
		}),
		new web3._extend.Method({
			name: 'setGasLimit',
			call: 'miner_setGasLimit',
			params: 1,
			inputFormatter: [web3._extend.utils.fromDecimal]
		}),
		new web3._extend.Method({
			name: 'setRecommitInterval',
			call: 'miner_setRecommitInterval',
			params: 1,
		}),
	],
	properties: []
});
`

const NetJs = `
web3._extend({
	property: 'net',
	methods: [],
	properties: [
		new web3._extend.Property({
			name: 'version',
			getter: 'net_version'
		}),
	]
});
`

const PersonalJs = `
web3._extend({
	property: 'personal',
	methods: [
		new web3._extend.Method({
			name: 'importRawKey',
			call: 'personal_importRawKey',
			params: 2
		}),
		new web3._extend.Method({
			name: 'sign',
			call: 'personal_sign',
			params: 3,
			inputFormatter: [null, web3._extend.formatters.inputAddressFormatter, null]
		}),
		new web3._extend.Method({
			name: 'ecRecover',
			call: 'personal_ecRecover',
			params: 2
		}),
		new web3._extend.Method({
			name: 'openWallet',
			call: 'personal_openWallet',
			params: 2
		}),
		new web3._extend.Method({
			name: 'deriveAccount',
			call: 'personal_deriveAccount',
			params: 3
		}),
		new web3._extend.Method({
			name: 'signTransaction',
			call: 'personal_signTransaction',
			params: 2,
			inputFormatter: [web3._extend.formatters.inputTransactionFormatter, null]
		}),
		new web3._extend.Method({
			name: 'unpair',
			call: 'personal_unpair',
			params: 2
		}),
		new web3._extend.Method({
			name: 'initializeWallet',
			call: 'personal_initializeWallet',
			params: 1
		})
	],
	properties: [
		new web3._extend.Property({
			name: 'listWallets',
			getter: 'personal_listWallets'
		}),
	]
})
`

const RpcJs = `
web3._extend({
	property: 'rpc',
	methods: [],
	properties: [
		new web3._extend.Property({
			name: 'modules',
			getter: 'rpc_modules'
		}),
	]
});
`

const TxpoolJs = `
web3._extend({
	property: 'txpool',
	methods: [],
	properties:
	[
		new web3._extend.Property({
			name: 'content',
			getter: 'txpool_content'
		}),
		new web3._extend.Property({
			name: 'inspect',
			getter: 'txpool_inspect'
		}),
		new web3._extend.Property({
			name: 'status',
			getter: 'txpool_status',
			outputFormatter: function(status) {
				status.pending = web3._extend.utils.toDecimal(status.pending);
				status.queued = web3._extend.utils.toDecimal(status.queued);
				status.processing = web3._extend.utils.toDecimal(status.processing);
				return status;
			}
		}),
		new web3._extend.Method({
			name: 'contentFrom',
			call: 'txpool_contentFrom',
			params: 1,
		}),
	]
});
`

const LESJs = `
web3._extend({
	property: 'les',
	methods:
	[
		new web3._extend.Method({
			name: 'getCheckpoint',
			call: 'les_getCheckpoint',
			params: 1
		}),
		new web3._extend.Method({
			name: 'clientInfo',
			call: 'les_clientInfo',
			params: 1
		}),
		new web3._extend.Method({
			name: 'priorityClientInfo',
			call: 'les_priorityClientInfo',
			params: 3
		}),
		new web3._extend.Method({
			name: 'setClientParams',
			call: 'les_setClientParams',
			params: 2
		}),
		new web3._extend.Method({
			name: 'setDefaultParams',
			call: 'les_setDefaultParams',
			params: 1
		}),
		new web3._extend.Method({
			name: 'addBalance',
			call: 'les_addBalance',
			params: 2
		}),
	],
	properties:
	[
		new web3._extend.Property({
			name: 'latestCheckpoint',
			getter: 'les_latestCheckpoint'
		}),
		new web3._extend.Property({
			name: 'checkpointContractAddress',
			getter: 'les_getCheckpointContractAddress'
		}),
		new web3._extend.Property({
			name: 'serverInfo',
			getter: 'les_serverInfo'
		}),
	]
});
`

const VfluxJs = `
web3._extend({
	property: 'vflux',
	methods:
	[
		new web3._extend.Method({
			name: 'distribution',
			call: 'vflux_distribution',
			params: 2
		}),
		new web3._extend.Method({
			name: 'timeout',
			call: 'vflux_timeout',
			params: 2
		}),
		new web3._extend.Method({
			name: 'value',
			call: 'vflux_value',
			params: 2
		}),
	],
	properties:
	[
		new web3._extend.Property({
			name: 'requestStats',
			getter: 'vflux_requestStats'
		}),
	]
});
`

const DagJs = `
web3._extend({
	property: 'dag',
	methods: [],
	properties: [
		new web3._extend.Method({
			name: 'sync',
			call: 'dag_sync',
			params: 1
		}),
		new web3._extend.Method({
			name: 'finalize',
			call: 'dag_finalize',
			params: 1
		}),
		new web3._extend.Method({
			name: 'coordinatedState',
			call: 'dag_coordinatedState',
			params: 0
		}),
		new web3._extend.Method({
			name: 'getCandidates',
			call: 'dag_getCandidates',
			params: 1
		}),
		new web3._extend.Method({
			name: 'getOptimisticSpines',
			call: 'dag_getOptimisticSpines',
			params: 1
		}),
		new web3._extend.Method({
			name: 'getDagHashes',
			call: 'dag_getDagHashes',
			params: 0
		}),
		new web3._extend.Method({
			name: 'headSyncReady',
			call: 'dag_headSyncReady',
			params: 1
		}),
		new web3._extend.Method({
			name: 'headSync',
			call: 'dag_headSync',
			params: 1
		}),
		new web3._extend.Method({
			name: 'validateSpines',
			call: 'dag_validateSpines',
			params: 1
		}),
		new web3._extend.Method({
			name: 'validateFinalization',
			call: 'dag_validateFinalization',
			params: 1
		}),
		new web3._extend.Method({
			name: 'syncSlotInfo',
			call: 'dag_syncSlotInfo',
			params: 1
		}),
	]
});
`

const WatJs = `
var BlsPubKeyLength = 48 * 2
var BlsSigLength = 96 * 2
var AddressLength = 20 * 2

function isHex (str) {
	return (/^(0x)?(([0-9A-Fa-f]){2,2})+$/).test(str)
}
function handleHexField (options, key, valLen) {
	val = options[key]
	if (val) {
		val = val.replace('0x', '')
		if (val.length != valLen) throw new Error(key +': invalid length. (required: ' + valLen + ')');
		if (!isHex(val)) throw new Error(key +': invalid hex.');
		options[key] = '0x' + val;
	} else {
		throw new Error(key +': field is required.');
	}
}

web3._extend({
	property: 'wat',
	methods:
	[
		// TOKEN API //
		new web3._extend.Method({
			name: 'tokenCreate',
			call: 'wat_tokenCreate',
			params: 1,
			inputFormatter: [function(options) {
				if (options.name) {
					options.name = web3._extend.utils.fromUtf8(options.name);
				} else {
					throw new Error('The name field is required.');
				}

				if (options.symbol) {
					options.symbol = web3._extend.utils.fromUtf8(options.symbol);
				} else {
					throw new Error('The symbol field is required.');
				}

				if (options.decimals) {
					options.decimals = web3._extend.utils.toHex(options.decimals);
				}
				if (options.totalSupply) {
					options.totalSupply = web3._extend.utils.toHex(options.totalSupply);
				}
				if (options.baseURI) {
					options.baseURI = web3._extend.utils.fromUtf8(options.baseURI);
				}
				return options;
			}]
		}),
		new web3._extend.Method({
			name: 'tokenProperties',
			call: 'wat_tokenProperties',
			params: 3,
			inputFormatter: [web3._extend.formatters.inputAddressFormatter, web3._extend.formatters.inputDefaultBlockNumberFormatter, function(tokenId) {
				if (tokenId) {
					return web3._extend.utils.fromDecimal(tokenId)
				} else {
					return undefined
				}
			}],
			outputFormatter: function(result) {
				result.name = web3._extend.utils.toUtf8(result.name);
				result.symbol = web3._extend.utils.toUtf8(result.symbol);
				if (result.baseURI) {
					result.baseURI = web3._extend.utils.toUtf8(result.baseURI);
				}
				if (result.decimals) {
					result.decimals = web3._extend.utils.toDecimal(result.decimals);
				}
				if (result.totalSupply) {
					result.totalSupply = web3._extend.utils.toDecimal(result.totalSupply);
				}
				if (result.byTokenId) {
					result.byTokenId.tokenURI = web3._extend.utils.toUtf8(result.byTokenId.tokenURI);
					result.byTokenId.ownerOf = web3._extend.utils.toAddress(result.byTokenId.ownerOf);
					result.byTokenId.getApproved = web3._extend.utils.toAddress(result.byTokenId.getApproved);
					result.byTokenId.metadata = web3._extend.utils.toUtf8(result.byTokenId.metadata);
				}
				return result;
			}
		}),
		new web3._extend.Method({
			name: 'tokenBalanceOf',
			call: 'wat_tokenBalanceOf',
			params: 3,
			inputFormatter: [web3._extend.formatters.inputAddressFormatter, web3._extend.formatters.inputAddressFormatter, web3._extend.formatters.inputDefaultBlockNumberFormatter],
			outputFormatter: web3._extend.utils.toDecimal
		}),
		new web3._extend.Method({
			name: 'wrc20Transfer',
			call: 'wat_wrc20Transfer',
			params: 2,
			inputFormatter: [web3._extend.formatters.inputAddressFormatter, web3._extend.utils.toHex]
		}),
		new web3._extend.Method({
			name: 'wrc20TransferFrom',
			call: 'wat_wrc20TransferFrom',
			params: 3,
			inputFormatter: [web3._extend.formatters.inputAddressFormatter, web3._extend.formatters.inputAddressFormatter, web3._extend.utils.toHex]
		}),
		new web3._extend.Method({
			name: 'wrc20Approve',
			call: 'wat_wrc20Approve',
			params: 2,
			inputFormatter: [web3._extend.formatters.inputAddressFormatter, web3._extend.utils.toHex]
		}),
		new web3._extend.Method({
			name: 'wrc20Allowance',
			call: 'wat_wrc20Allowance',
			params: 4,
			inputFormatter: [web3._extend.formatters.inputAddressFormatter, web3._extend.formatters.inputAddressFormatter, web3._extend.formatters.inputAddressFormatter, web3._extend.formatters.inputDefaultBlockNumberFormatter],
			outputFormatter: web3._extend.utils.toDecimal
		}),
		new web3._extend.Method({
			name: 'wrc721Mint',
			call: 'wat_wrc721Mint',
			params: 3,
			inputFormatter: [web3._extend.formatters.inputAddressFormatter, web3._extend.utils.toHex, web3._extend.utils.fromUtf8],
		}),
		new web3._extend.Method({
			name: 'wrc721Burn',
			call: 'wat_wrc721Burn',
			params: 1,
			inputFormatter: [web3._extend.utils.toHex],
		}),
		new web3._extend.Method({
			name: 'wrc721SetApprovalForAll',
			call: 'wat_wrc721SetApprovalForAll',
			params: 2,
			inputFormatter: [web3._extend.formatters.inputAddressFormatter, null],
		}),
		new web3._extend.Method({
			name: 'wrc721IsApprovedForAll',
			call: 'wat_wrc721IsApprovedForAll',
			params: 4,
			inputFormatter: [web3._extend.formatters.inputAddressFormatter, web3._extend.formatters.inputAddressFormatter, web3._extend.formatters.inputAddressFormatter, web3._extend.formatters.inputDefaultBlockNumberFormatter]
		}),
		new web3._extend.Method({
			name: 'wrc721Approve',
			call: 'wat_wrc721Approve',
			params: 2,
			inputFormatter: [web3._extend.formatters.inputAddressFormatter, web3._extend.utils.toHex]
		}),
		new web3._extend.Method({
			name: 'wrc721TransferFrom',
			call: 'wat_wrc721TransferFrom',
			params: 3,
			inputFormatter: [web3._extend.formatters.inputAddressFormatter, web3._extend.formatters.inputAddressFormatter, web3._extend.utils.toHex]
		}),
		new web3._extend.Method({
			name: 'tokenCost',
			call: 'wat_tokenCost',
			params: 3,
			inputFormatter: [web3._extend.formatters.inputAddressFormatter, web3._extend.utils.toHex, web3._extend.formatters.inputDefaultBlockNumberFormatter],
			outputFormatter: web3._extend.utils.toDecimal
		}),
		new web3._extend.Method({
			name: 'setPrice',
			call: 'wat_setPrice',
			params: 2,
			inputFormatter: [web3._extend.utils.toHex, web3._extend.utils.toHex],
		}),
		new web3._extend.Method({
			name: 'buy',
			call: 'wat_buy',
			params: 2,
			inputFormatter: [web3._extend.utils.toHex, web3._extend.utils.toHex],
		}),

		// VALIDATOR STORAGE API //
		new web3._extend.Method({
			name: 'getValidators',
			call: 'wat_getValidators',
			params: 1,
			inputFormatter: [null]
		}),
		new web3._extend.Method({
			name: 'getValidatorsBySlot',
			call: 'wat_getValidatorsBySlot',
			params: 1
		}),		
		new web3._extend.Method({
			name: 'validator.getInfo',
			call: 'wat_validator_GetInfo',
			params: 2,
			inputFormatter: [web3._extend.formatters.inputAddressFormatter, function(param) {
				if (param == null || param == "cp") {
					return "cp"
				}
				return web3._extend.formatters.inputDefaultBlockNumberFormatter(param)
			}]
		}),	

		// INFO API //
		new web3._extend.Method({
			name: 'getEra',
			call: 'wat_getEra',
			params: 1,
			inputFormatter: [null]
		}),
		new web3._extend.Method({
			name: 'getDagHashes',
			call: 'wat_getDagHashes',
			params: 0
		}),	
		new web3._extend.Method({
			name: 'getSlotHashes',
			call: 'wat_getSlotHashes',
			params: 1
		}),

		// VALIDATOR API //
		new web3._extend.Method({
			name: 'validator.depositData',
			call: 'wat_validator_DepositData',
			params: 1,
			inputFormatter: [function(options) {
				function handleDelegatingRules(rules) {
					function mapHandler(rules, key) {
						var upData = {}
						for (var k in rules[key]) {
							var addr = k.replace('0x', '');
							if (addr.length != AddressLength) throw new Error(key + ': invalid length of address=0x' + addr + '.');
							if (!isHex(addr)) throw new Error(key + ': invalid hex of address=0x' + addr + '.');
							var val = rules[key][k];
							val = web3._extend.utils.toDecimal(val);
							if (!Number.isInteger(val) || val < 0) throw new Error(key + ': invalid value=' + val + ' address=0x' + addr + '.\nRequire positive integer.');
							upData['0x' + addr] = val;
						}
						rules[key] = upData
					}
		
					function arrHandler(rules, key) {
						var upData = []
						for (var k in rules[key]) {
							var addr = (rules[key][k]).replace('0x', '');
							if (addr.length != AddressLength) throw new Error(key + ': invalid length of address=0x' + addr + '.');
							if (!isHex(addr)) throw new Error(key + ': invalid hex of address=0x' + addr + '.');
							upData.push('0x' + addr);
						}
						rules[key] = upData
					}
		
					if (rules.profit_share) {
						var key = 'profit_share';
						mapHandler(rules, key)
					}
					if (rules.stake_share) {
						var key = 'stake_share';
						mapHandler(rules, key)
					}
					if (Array.isArray(rules.exit)) {
						var key = 'exit';
						arrHandler(rules, key)
					}
					if (Array.isArray(rules.withdrawal)) {
						var key = 'withdrawal';
						arrHandler(rules, key)
					}
					return rules
				}
		
				// handle base deposit data
				handleHexField(options, 'pubkey', BlsPubKeyLength)
				handleHexField(options, 'creator_address', AddressLength)
				handleHexField(options, 'withdrawal_address', AddressLength)
				handleHexField(options, 'signature', BlsSigLength)

				// handle delegating stake data
				if (options.delegating_stake) {
					var dlgData = options.delegating_stake
					if (dlgData.trial_period) {
						dlgData.trial_period = web3._extend.utils.toDecimal(dlgData.trial_period);
					}
					if (dlgData.trial_rules) {
						dlgData.trial_rules = handleDelegatingRules(dlgData.trial_rules)
					}
					if (dlgData.rules) {
						dlgData.rules = handleDelegatingRules(dlgData.rules)
					}
				}
				return options;
			}]
		}),
		new web3._extend.Method({
			name: 'validator.exitData',
			call: 'wat_validator_ExitData',
			params: 1,
			inputFormatter: [function(options) {
				handleHexField(options, 'pubkey', BlsPubKeyLength)
				handleHexField(options, 'creator_address', AddressLength)
				if (options.exit_epoch !== undefined) {
					throw new Error('exit_epoch: remove deprecated field.');
				}
				return options;
			}]
		}),
		new web3._extend.Method({
			name: 'validator.withdrawalData',
			call: 'wat_validator_WithdrawalData',
			params: 1,
			inputFormatter: [function(options) {
				handleHexField(options, 'creator_address', AddressLength)
				options.amount = web3._extend.utils.toHex(options.amount);
				return options;
			}]
		}),
		new web3._extend.Method({
			name: 'validator.depositCount',
			call: 'wat_validator_DepositCount',
			params: 1,
			inputFormatter: [web3._extend.formatters.inputDefaultBlockNumberFormatter],
			outputFormatter: web3._extend.utils.toDecimal
		}),
		new web3._extend.Method({
			name: 'validator.depositAddress',
			call: 'wat_validator_DepositAddress',
		}),
		new web3._extend.Method({
			name: 'validator.getTransactionReceipt',
			call: 'wat_validator_GetTransactionReceipt',
			params: 1,
			outputFormatter: web3._extend.formatters.outputTransactionReceiptFormatter
		}),
		new web3._extend.Property({
		  name: 'info',
		  getter: 'wat_info'
		}),
	]
});
`
