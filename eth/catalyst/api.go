// Copyright 2020 The go-ethereum Authors
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

// Package catalyst implements the temporary eth1/eth2 RPC integration.
package catalyst

import (
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/waterfall-foundation/gwat/common"
	"github.com/waterfall-foundation/gwat/consensus/misc"
	"github.com/waterfall-foundation/gwat/core"
	"github.com/waterfall-foundation/gwat/core/state"
	"github.com/waterfall-foundation/gwat/core/types"
	"github.com/waterfall-foundation/gwat/eth"
	"github.com/waterfall-foundation/gwat/log"
	"github.com/waterfall-foundation/gwat/node"
	chainParams "github.com/waterfall-foundation/gwat/params"
	"github.com/waterfall-foundation/gwat/rpc"
	"github.com/waterfall-foundation/gwat/trie"
)

// Register adds catalyst APIs to the node.
func Register(stack *node.Node, backend *eth.Ethereum) error {
	chainconfig := backend.BlockChain().Config()
	if chainconfig.TerminalTotalDifficulty == nil {
		return errors.New("catalyst started without valid total difficulty")
	}

	log.Warn("Catalyst mode enabled")
	stack.RegisterAPIs([]rpc.API{
		{
			Namespace: "consensus",
			Version:   "1.0",
			Service:   newConsensusAPI(backend),
			Public:    true,
		},
	})
	return nil
}

type consensusAPI struct {
	eth *eth.Ethereum
}

func newConsensusAPI(eth *eth.Ethereum) *consensusAPI {
	return &consensusAPI{eth: eth}
}

// blockExecutionEnv gathers all the data required to execute
// a block, either when assembling it or when inserting it.
type blockExecutionEnv struct {
	chain   *core.BlockChain
	state   *state.StateDB
	tcount  int
	gasPool *core.GasPool

	header   *types.Header
	txs      []*types.Transaction
	receipts []*types.Receipt
}

func (env *blockExecutionEnv) commitTransaction(tx *types.Transaction, coinbase common.Address) error {
	vmconfig := *env.chain.GetVMConfig()
	snap := env.state.Snapshot()
	receipt, err := core.ApplyTransaction(env.chain.Config(), env.chain, &coinbase, env.gasPool, env.state, env.header, tx, &env.header.GasUsed, vmconfig)
	if err != nil {
		env.state.RevertToSnapshot(snap)
		return err
	}
	env.txs = append(env.txs, tx)
	env.receipts = append(env.receipts, receipt)
	return nil
}

func (api *consensusAPI) makeEnv(parent *types.Block, header *types.Header) (*blockExecutionEnv, error) {
	state, err := api.eth.BlockChain().StateAt(parent.Root())
	if err != nil {
		return nil, err
	}
	env := &blockExecutionEnv{
		chain:   api.eth.BlockChain(),
		state:   state,
		header:  header,
		gasPool: new(core.GasPool).AddGas(header.GasLimit),
	}
	return env, nil
}

// AssembleBlock creates a new block, inserts it into the chain, and returns the "execution
// data" required for eth2 clients to process the new block.
func (api *consensusAPI) AssembleBlock(params assembleBlockParams) (*executableData, error) {
	log.Info("Producing block", "parentHashes", params.ParentHashes)

	bc := api.eth.BlockChain()
	var parent *types.Block
	for _, h := range params.ParentHashes {
		ph := bc.GetBlockByHash(h)
		if ph != nil && ph.Nr() == ph.Height() {
			parent = ph
			break
		}
	}

	if parent == nil {
		log.Warn("Cannot assemble block with parent hash to unknown block", "parentHashes", params.ParentHashes)
		return nil, fmt.Errorf("cannot assemble block with unknown parent %s", params.ParentHashes)
	}

	pool := api.eth.TxPool()

	if parent.Time() >= params.Timestamp {
		return nil, fmt.Errorf("child timestamp lower than parent's: %d >= %d", parent.Time(), params.Timestamp)
	}
	if now := uint64(time.Now().Unix()); params.Timestamp > now+1 {
		wait := time.Duration(params.Timestamp-now) * time.Second
		log.Info("Producing block too far in the future", "wait", common.PrettyDuration(wait))
		time.Sleep(wait)
	}

	pending := pool.Pending(true)

	coinbase, err := api.eth.Etherbase()
	if err != nil {
		return nil, err
	}
	header := &types.Header{
		ParentHashes: []common.Hash{parent.Hash()},
		Epoch:        0,
		Slot:         0,
		Height:       0,
		Coinbase:     coinbase,
		GasLimit:     parent.GasLimit(), // Keep the gas limit constant in this prototype
		Extra:        []byte{},
		Time:         params.Timestamp,
	}
	config := api.eth.BlockChain().Config()
	header.BaseFee = misc.CalcBaseFee(config, parent.Header())

	err = api.eth.Engine().Prepare(bc, header)
	if err != nil {
		return nil, err
	}

	env, err := api.makeEnv(parent, header)
	if err != nil {
		return nil, err
	}

	var (
		signer       = types.MakeSigner(bc.Config())
		txHeap       = types.NewTransactionsByPriceAndNonce(signer, pending, nil)
		transactions []*types.Transaction
	)
	for {
		if env.gasPool.Gas() < chainParams.TxGas {
			log.Trace("Not enough gas for further transactions", "have", env.gasPool, "want", chainParams.TxGas)
			break
		}
		tx := txHeap.Peek()
		if tx == nil {
			break
		}

		// The sender is only for logging purposes, and it doesn't really matter if it's correct.
		from, _ := types.Sender(signer, tx)

		// Execute the transaction
		env.state.Prepare(tx.Hash(), env.tcount)
		err = env.commitTransaction(tx, coinbase)
		switch err {
		case core.ErrGasLimitReached:
			// Pop the current out-of-gas transaction without shifting in the next from the account
			log.Trace("Gas limit exceeded for current block", "sender", from)
			txHeap.Pop()

		case core.ErrNonceTooLow:
			// New head notification data race between the transaction pool and miner, shift
			log.Error("Skipping transaction with low nonce", "sender", from, "nonce", tx.Nonce())
			txHeap.Shift()

		case core.ErrNonceTooHigh:
			// Reorg notification data race between the transaction pool and miner, skip account =
			log.Error("Skipping account with high nonce", "sender", from, "nonce", tx.Nonce())
			txHeap.Pop()

		case nil:
			// Everything ok, collect the logs and shift in the next transaction from the same account
			env.tcount++
			txHeap.Shift()
			transactions = append(transactions, tx)

		default:
			// Strange error, discard the transaction and get the next in line (note, the
			// nonce-too-high clause will prevent us from executing in vain).
			log.Error("Transaction failed, account skipped", "hash", tx.Hash(), "err", err)
			txHeap.Shift()
		}
	}

	// Create the block.
	block, err := api.eth.Engine().FinalizeAndAssemble(bc, header, env.state, transactions, env.receipts)
	if err != nil {
		return nil, err
	}
	return &executableData{
		BlockHash:    block.Hash(),
		ParentHashes: block.ParentHashes(),
		Epoch:        block.Epoch(),
		Slot:         block.Slot(),
		Height:       block.Height(),
		Miner:        block.Coinbase(),
		StateRoot:    block.Root(),
		GasLimit:     block.GasLimit(),
		GasUsed:      block.GasUsed(),
		Timestamp:    block.Time(),
		ReceiptRoot:  block.ReceiptHash(),
		LogsBloom:    block.Bloom().Bytes(),
		Transactions: encodeTransactions(block.Transactions()),
	}, nil
}

func encodeTransactions(txs []*types.Transaction) [][]byte {
	var enc = make([][]byte, len(txs))
	for i, tx := range txs {
		enc[i], _ = tx.MarshalBinary()
	}
	return enc
}

func decodeTransactions(enc [][]byte) ([]*types.Transaction, error) {
	var txs = make([]*types.Transaction, len(enc))
	for i, encTx := range enc {
		var tx types.Transaction
		if err := tx.UnmarshalBinary(encTx); err != nil {
			return nil, fmt.Errorf("invalid transaction %d: %v", i, err)
		}
		txs[i] = &tx
	}
	return txs, nil
}

func insertBlockParamsToBlock(config *chainParams.ChainConfig, parent *types.Header, params executableData) (*types.Block, error) {
	txs, err := decodeTransactions(params.Transactions)
	if err != nil {
		return nil, err
	}

	number := big.NewInt(0)
	number.SetUint64(params.Number)
	header := &types.Header{
		ParentHashes: params.ParentHashes,
		Epoch:        params.Epoch,
		Slot:         params.Slot,
		Height:       params.Height,
		Coinbase:     params.Miner,
		Root:         params.StateRoot,
		TxHash:       types.DeriveSha(types.Transactions(txs), trie.NewStackTrie(nil)),
		ReceiptHash:  params.ReceiptRoot,
		Bloom:        types.BytesToBloom(params.LogsBloom),
		GasLimit:     params.GasLimit,
		GasUsed:      params.GasUsed,
		Time:         params.Timestamp,
	}
	if config.IsLondon(number) {
		header.BaseFee = misc.CalcBaseFee(config, parent)
	}
	block := types.NewBlockWithHeader(header).WithBody(txs)
	return block, nil
}

// NewBlock creates an Eth1 block, inserts it in the chain, and either returns true,
// or false + an error. This is a bit redundant for go, but simplifies things on the
// eth2 side.
func (api *consensusAPI) NewBlock(params executableData) (*newBlockResponse, error) {
	var parent *types.Block
	for _, h := range params.ParentHashes {
		ph := api.eth.BlockChain().GetBlockByHash(h)
		if ph != nil && ph.Nr() == ph.Height() {
			parent = ph
			break
		}
	}
	if parent == nil {
		return &newBlockResponse{false}, fmt.Errorf("could not find parent %x", params.ParentHashes)
	}
	block, err := insertBlockParamsToBlock(api.eth.BlockChain().Config(), parent.Header(), params)
	if err != nil {
		return nil, err
	}
	_, err = api.eth.BlockChain().InsertChainWithoutSealVerification(block)
	return &newBlockResponse{err == nil}, err
}

// Used in tests to add a the list of transactions from a block to the tx pool.
func (api *consensusAPI) addBlockTxs(block *types.Block) error {
	for _, tx := range block.Transactions() {
		api.eth.TxPool().AddLocal(tx)
	}
	return nil
}

// FinalizeBlock is called to mark a block as synchronized, so
// that data that is no longer needed can be removed.
func (api *consensusAPI) FinalizeBlock(blockHash common.Hash) (*genericResponse, error) {
	return &genericResponse{true}, nil
}

// SetHead is called to perform a force choice.
func (api *consensusAPI) SetHead(newHead common.Hash) (*genericResponse, error) {
	return &genericResponse{true}, nil
}
