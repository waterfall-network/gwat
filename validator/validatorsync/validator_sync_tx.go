package validatorsync

import (
	"fmt"
	"math"
	"math/big"

	"gitlab.waterfall.network/waterfall/protocol/gwat/accounts"
	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/types"
	"gitlab.waterfall.network/waterfall/protocol/gwat/log"
	"gitlab.waterfall.network/waterfall/protocol/gwat/validator/operation"
)

// Backend interface provides the common API services (that are provided by
// both full and light clients) with access to necessary functions.
type Backend interface {
	BlockChain() *core.BlockChain
	Etherbase() (eb common.Address, err error)
	CreatorAuthorize(creator common.Address) error
	AccountManager() *accounts.Manager

	//Downloader() *downloader.Downloader
}

// GetPendingValidatorSyncData retrieves currently processable validators sync operations.
func CreateValidatorSyncTx(backend Backend, stateBlockHash common.Hash, from common.Address, valSyncOp *types.ValidatorSync, nonce uint64) (*types.Transaction, error) {
	bc := backend.BlockChain()
	_, err := ValidateValidatorSyncOp(bc, stateBlockHash, valSyncOp)
	if err != nil {
		return nil, err
	}
	// collect validator data
	stateHead := bc.GetHeaderByHash(stateBlockHash)
	stateDb, err := bc.StateAt(stateHead.Root)
	if err != nil {
		return nil, err
	}
	validatorData := bc.ValidatorStorage().GetValidatorInfo(stateDb, valSyncOp.Creator)

	// get tx.To address
	valStateAddr := bc.Config().ValidatorsStateAddress

	// validate nonce
	nonceFrom := stateDb.GetNonce(from)
	if nonceFrom > nonce {
		return nil, fmt.Errorf("nonce is too low:  fromAddr=%s fromNonce=%d txNonce=%d", from.Hex(), nonceFrom, nonce)
	}
	var withdrawalAddress *common.Address
	if valSyncOp.OpType == types.Withdrawal {
		*withdrawalAddress = validatorData.GetWithdrawalAddress()
	}
	valSyncTxData, err := getValSyncTxData(*valSyncOp, withdrawalAddress)

	txData := &types.DynamicFeeTx{
		To:         valStateAddr,
		ChainID:    (backend.BlockChain()).Config().ChainID,
		Nonce:      nonce,
		Gas:        0,
		GasFeeCap:  new(big.Int).SetUint64(0),
		GasTipCap:  new(big.Int).SetUint64(0),
		Value:      new(big.Int).SetUint64(0),
		Data:       valSyncTxData,
		AccessList: types.AccessList{},
	}
	tx := types.NewTx(txData)

	signed, err := signTx(backend, from, tx)
	if err != nil {
		return nil, err
	}
	return signed, nil
}

func ValidateValidatorSyncOp(bc *core.BlockChain, stateBlockHash common.Hash, valSyncOp *types.ValidatorSync) (bool, error) {
	if valSyncOp == nil {
		return false, fmt.Errorf("validate validator sync operation failed: nil data")
	}
	stateHead := bc.GetHeaderByHash(stateBlockHash)
	if stateHead == nil {
		return false, fmt.Errorf("validate validator sync operation failed: state block not found heash=%s", stateBlockHash.Hex())
	}
	stateEpoch := bc.GetSlotInfo().SlotToEpoch(stateHead.Slot)
	if valSyncOp.ProcEpoch < stateEpoch {
		return false, fmt.Errorf("validate validator sync operation failed: outdated epoch ProcEpoch=%d stateEpoch=%d", valSyncOp.ProcEpoch, stateEpoch)
	}

	stateDb, err := bc.StateAt(stateHead.Root)
	if err != nil {
		return false, err
	}
	if !stateDb.IsValidatorAddress(valSyncOp.Creator) {
		return false, fmt.Errorf("validate validator sync operation failed: address is not validator: %s", valSyncOp.Creator.Hex())
	}
	validatorData := bc.ValidatorStorage().GetValidatorInfo(stateDb, valSyncOp.Creator)

	switch valSyncOp.OpType {
	case types.Activation:
		if validatorData.GetActivationEpoch() < math.MaxUint64 {
			return false, fmt.Errorf("validate validator sync operation failed: validator already activated")
		}
	case types.Exit:
		if validatorData.GetExitEpoch() < math.MaxUint64 {
			return false, fmt.Errorf("validate validator sync operation failed: validator already exited")
		}
		if validatorData.GetActivationEpoch() >= valSyncOp.ProcEpoch {
			return false, fmt.Errorf("validate validator sync operation failed: exit epoche is too low")
		}
	case types.Withdrawal:
		if valSyncOp.Amount == nil {
			return false, fmt.Errorf("validate validator sync operation failed: withdrawal amount is required")
		}
		if validatorData.GetActivationEpoch() >= valSyncOp.ProcEpoch || validatorData.GetExitEpoch() >= valSyncOp.ProcEpoch {
			return false, fmt.Errorf("validate validator sync operation failed: withdrowal epoche is too low")
		}
	default:
		return false, fmt.Errorf("validate validator sync operation failed: unknown oparation type %d", valSyncOp.OpType)
	}
	return true, nil
}

func getValSyncTxData(valSyncOp types.ValidatorSync, withdrawal *common.Address) ([]byte, error) {
	var (
		op  operation.Operation
		err error
	)
	if op, err = operation.NewValidatorSyncOperation(valSyncOp.OpType, valSyncOp.ProcEpoch, valSyncOp.Index, valSyncOp.Creator, valSyncOp.Amount, withdrawal); err != nil {
		return nil, err
	}
	b, err := operation.EncodeToBytes(op)
	if err != nil {
		log.Warn("Failed to encode validator sync operation", "err", err)
		return nil, err
	}
	return b, nil
}

// sign is a helper function that signs a transaction with the private key of the given address.
func signTx(backend Backend, addr common.Address, tx *types.Transaction) (*types.Transaction, error) {
	// Look up the wallet containing the requested signer
	account := accounts.Account{Address: addr}
	wallet, err := backend.AccountManager().Find(account)
	if err != nil {
		return nil, err
	}
	// Request the wallet to sign the transaction
	return wallet.SignTx(account, tx, (backend.BlockChain()).Config().ChainID)
}

// GetPendingValidatorSyncData retrieves currently processable validators sync operations.
func GetPendingValidatorSyncData(bc *core.BlockChain) map[[28]byte]*types.ValidatorSync {
	si := bc.GetSlotInfo()
	currEpoch := si.SlotToEpoch(si.CurrentSlot())

	valSyncOps := bc.GetNotProcessedValidatorSyncData()
	vsPending := make(map[[28]byte]*types.ValidatorSync, len(valSyncOps))
	for k, vs := range valSyncOps {
		if vs.ProcEpoch == currEpoch && vs.TxHash == nil {
			vsPending[k] = vs
		}
	}
	return vsPending
}
