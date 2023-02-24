package operation

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/status-im/keycard-go/hexutils"
	"testing"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
)

func TestDepositData(t *testing.T) {
	type decodedOp struct {
		pubkey             common.BlsPubKey // validator public key
		creator_address    common.Address   // attached creator account
		withdrawal_address common.Address   // attached withdrawal credentials
		signature          common.BlsSignature
		deposit_data_root  common.Hash
	}

	cases := []operationTestCase{
		{
			caseName: "OK",
			decoded: decodedOp{
				pubkey:             pubkey,
				creator_address:    creator_address,
				withdrawal_address: withdrawal_address,
				signature:          signature,
				deposit_data_root:  deposit_data_root,
			},
			encoded: hexutils.HexToBytes(depositData),
			errs:    []error{},
		},
		{
			caseName: "ErrNoPubKey",
			decoded: decodedOp{
				//pubkey:             pubkey,
				creator_address:    creator_address,
				withdrawal_address: withdrawal_address,
				signature:          signature,
				deposit_data_root:  deposit_data_root,
			},
			encoded: hexutils.HexToBytes(""),
			errs:    []error{ErrNoPubKey},
		},
		{
			caseName: "ErrNoCreatorAddress",
			decoded: decodedOp{
				pubkey: pubkey,
				//creator_address:    creator_address,
				withdrawal_address: withdrawal_address,
				signature:          signature,
				deposit_data_root:  deposit_data_root,
			},
			encoded: hexutils.HexToBytes(""),
			errs:    []error{ErrNoCreatorAddress},
		},
		{
			caseName: "ErrNoWithdrawalAddress",
			decoded: decodedOp{
				pubkey:          pubkey,
				creator_address: creator_address,
				//withdrawal_address: withdrawal_address,
				signature:         signature,
				deposit_data_root: deposit_data_root,
			},
			encoded: hexutils.HexToBytes(""),
			errs:    []error{ErrNoWithdrawalAddress},
		},
		{
			caseName: "ErrNoSignature",
			decoded: decodedOp{
				pubkey:             pubkey,
				creator_address:    creator_address,
				withdrawal_address: withdrawal_address,
				//signature:          signature,
				deposit_data_root: deposit_data_root,
			},
			encoded: hexutils.HexToBytes(""),
			errs:    []error{ErrNoSignature},
		},
		{
			caseName: "ErrNoDepositDataRoot",
			decoded: decodedOp{
				pubkey:             pubkey,
				creator_address:    creator_address,
				withdrawal_address: withdrawal_address,
				signature:          signature,
				//deposit_data_root:  deposit_data_root,
			},
			encoded: hexutils.HexToBytes(""),
			errs:    []error{ErrNoDepositDataRoot},
		},
	}

	operationEncode := func(b []byte, i interface{}) error {
		o := i.(decodedOp)
		createOp, err := NewDepositOperation(
			o.pubkey,
			o.creator_address,
			o.withdrawal_address,
			o.signature,
			o.deposit_data_root,
		)
		if err != nil {
			return err
		}

		return equalOpBytes(createOp, b)
	}

	operationDecode := func(b []byte, i interface{}) error {
		op, err := DecodeBytes(b)
		if err != nil {
			return err
		}

		o := i.(decodedOp)
		opDecoded, ok := op.(Deposit)
		if !ok {
			return errors.New("invalid operation type")
		}
		err = checkOpCode(b, opDecoded)
		if err != nil {
			return err
		}
		if !bytes.Equal(opDecoded.PubKey().Bytes(), o.pubkey.Bytes()) {
			return fmt.Errorf("pubkey do not match:\nwant: %#x\nhave: %#x", o.pubkey, opDecoded.PubKey())
		}
		if !bytes.Equal(opDecoded.CreatorAddress().Bytes(), o.creator_address.Bytes()) {
			return fmt.Errorf("creator_address do not match:\nwant: %#x\nhave: %#x", o.creator_address, opDecoded.CreatorAddress())
		}
		if !bytes.Equal(opDecoded.WithdrawalAddress().Bytes(), o.withdrawal_address.Bytes()) {
			return fmt.Errorf("withdrawal_address do not match:\nwant: %#x\nhave: %#x", o.withdrawal_address, opDecoded.WithdrawalAddress())
		}
		if !bytes.Equal(opDecoded.Signature().Bytes(), o.signature.Bytes()) {
			return fmt.Errorf("signature do not match:\nwant: %#x\nhave: %#x", o.signature, opDecoded.Signature())
		}
		if !bytes.Equal(opDecoded.DepositDataRoot().Bytes(), o.deposit_data_root.Bytes()) {
			return fmt.Errorf("deposit_data_root do not match:\nwant: %#x\nhave: %#x", o.deposit_data_root, opDecoded.DepositDataRoot())
		}
		return nil
	}

	startSubTests(t, cases, operationEncode, operationDecode)
}
