package validatorsync

import (
	"fmt"
	"math"
	"math/big"

	"gitlab.waterfall.network/waterfall/protocol/gwat/accounts"
	"gitlab.waterfall.network/waterfall/protocol/gwat/accounts/keystore"
	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/types"
	"gitlab.waterfall.network/waterfall/protocol/gwat/log"
	"gitlab.waterfall.network/waterfall/protocol/gwat/params"
	"gitlab.waterfall.network/waterfall/protocol/gwat/validator/operation"
)

// Backend interface provides the common API services (that are provided by
// both full and light clients) with access to necessary functions.
type Backend interface {
	BlockChain() *core.BlockChain
	CreatorAuthorize(creator common.Address) error
	AccountManager() *accounts.Manager
}

// GetPendingValidatorSyncData retrieves currently processable validators sync operations.
func CreateValidatorSyncTx(
	backend Backend,
	stateBlockHash common.Hash,
	from common.Address,
	slot uint64,
	valSyncOp *types.ValidatorSync,
	nonce uint64,
	ks *keystore.KeyStore,
) (*types.Transaction, error) {
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
	validator, err := bc.ValidatorStorage().GetValidator(stateDb, valSyncOp.Creator)
	if err != nil {
		return nil, err
	}

	// get tx.To address
	valStateAddr := bc.Config().ValidatorsStateAddress

	// validate nonce
	nonceFrom := stateDb.GetNonce(from)
	if nonceFrom > nonce {
		return nil, fmt.Errorf("nonce is too low:  fromAddr=%s fromNonce=%d txNonce=%d", from.Hex(), nonceFrom, nonce)
	}
	var withdrawalAddress *common.Address
	if valSyncOp.OpType == types.UpdateBalance {
		wa := validator.GetWithdrawalAddress()
		withdrawalAddress = wa
	}
	opVer := getValSyncVersionBySlot(bc.Config(), slot)

	log.Info("Validator sync tx data",
		"slot", slot,
		"ver", opVer,
		"Creator", valSyncOp.Creator.Hex(),
		"ProcEpoch", valSyncOp.ProcEpoch,
		"OpType", valSyncOp.OpType,
		"Amount", valSyncOp.Amount.String(),
		"Balance", valSyncOp.Balance.String(),
		"Index", valSyncOp.Index,
		"InitTxHash", valSyncOp.InitTxHash.Hex(),
	)

	valSyncTxData, err := getValSyncTxData(*valSyncOp, withdrawalAddress, opVer)
	if err != nil {
		return nil, err
	}

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

	signed, err := signTx(backend, from, tx, ks)
	if err != nil {
		return nil, err
	}
	return signed, nil
}

func ValidateValidatorSyncOp(bc *core.BlockChain, stateBlockHash common.Hash, valSyncOp *types.ValidatorSync) (bool, error) {
	if valSyncOp == nil {
		return false, fmt.Errorf("validator sync operation failed: nil data")
	}
	stateHead := bc.GetHeaderByHash(stateBlockHash)
	if stateHead == nil {
		return false, fmt.Errorf("validator sync operation failed: state block not found heash=%s", stateBlockHash.Hex())
	}
	stateEpoch := bc.GetSlotInfo().SlotToEpoch(stateHead.Slot)
	if valSyncOp.ProcEpoch < stateEpoch {
		return false, fmt.Errorf("validator sync operation failed: outdated epoch ProcEpoch=%d stateEpoch=%d", valSyncOp.ProcEpoch, stateEpoch)
	}

	stateDb, err := bc.StateAt(stateHead.Root)
	if err != nil {
		return false, err
	}
	if !stateDb.IsValidatorAddress(valSyncOp.Creator) {
		return false, fmt.Errorf("validator sync operation failed: address is not validator: %s", valSyncOp.Creator.Hex())
	}
	validator, err := bc.ValidatorStorage().GetValidator(stateDb, valSyncOp.Creator)
	if err != nil {
		return false, err
	}

	procEra := bc.EpochToEra(valSyncOp.ProcEpoch)

	switch valSyncOp.OpType {
	case types.Activate:
		if validator.GetActivationEra() < math.MaxUint64 {
			return false, fmt.Errorf("validator sync operation failed: validator already activated")
		}
	case types.Deactivate:
		if validator.GetExitEra() < math.MaxUint64 {
			return false, fmt.Errorf("validator sync operation failed: validator already deactivated")
		}
		if validator.GetActivationEra() >= procEra.Number {
			return false, fmt.Errorf("validator sync operation failed: exit epoche is too low")
		}
	case types.UpdateBalance:
		if valSyncOp.Amount == nil || valSyncOp.Amount.Sign() == 0 {
			return false, fmt.Errorf("validator sync operation failed: withdrawal amount is required")
		}
		if valSyncOp.Amount.Sign() == -1 {
			return false, fmt.Errorf("validator sync operation failed: withdrawal amount is negative")
		}
	default:
		return false, fmt.Errorf("validator sync operation failed: unknown oparation type %d", valSyncOp.OpType)
	}
	return true, nil
}

func getValSyncTxData(valSyncOp types.ValidatorSync, withdrawal *common.Address, version operation.VersionValSyncOp) ([]byte, error) {
	var (
		op  operation.Operation
		err error
	)
	if op, err = operation.NewValidatorSyncOperation(
		version,
		valSyncOp.OpType,
		valSyncOp.InitTxHash,
		valSyncOp.ProcEpoch,
		valSyncOp.Index,
		valSyncOp.Creator,
		valSyncOp.Amount,
		withdrawal,
		valSyncOp.Balance,
	); err != nil {
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
func signTx(backend Backend, addr common.Address, tx *types.Transaction, ks *keystore.KeyStore) (*types.Transaction, error) {
	// Look up the wallet containing the requested signer
	account := accounts.Account{Address: addr}

	return ks.SignTx(account, tx, (backend.BlockChain()).Config().ChainID)
}

// GetPendingValidatorSyncData retrieves currently processable validators sync operations.
func GetPendingValidatorSyncData(bc *core.BlockChain) map[common.Hash]*types.ValidatorSync {
	si := bc.GetSlotInfo()
	currEpoch := si.SlotToEpoch(si.CurrentSlot())

	valSyncOps := bc.GetNotProcessedValidatorSyncData()
	vsPending := make(map[common.Hash]*types.ValidatorSync, len(valSyncOps))
	for k, vs := range valSyncOps {
		if vs.TxHash != nil {
			continue
		}
		saved := bc.GetValidatorSyncData(vs.InitTxHash)
		if saved.TxHash != nil {
			continue
		}
		if vs.ProcEpoch == currEpoch {
			vsPending[k] = vs
		}
	}
	return vsPending
}

func getValSyncVersionBySlot(conf *params.ChainConfig, slot uint64) operation.VersionValSyncOp {
	var ver operation.VersionValSyncOp
	if conf.IsForkSlotDelegate(slot) {
		ver = operation.Ver1
	}
	return ver
}
