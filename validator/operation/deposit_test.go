package operation

import (
	"errors"
	"fmt"
	"github.com/status-im/keycard-go/hexutils"
	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/tests/testutils"
	"testing"
)

func TestDepositData(t *testing.T) {
	var (
		pubkey             = common.HexToBlsPubKey("0x9728bc733c8fcedde0c3a33dac12da3ebbaa0eb74d813a34b600520e7976a260d85f057687e8c923d52c78715515348d")
		creator_address    = common.HexToAddress("0xa7e558cc6efa1c41270ef4aa227b3dd6b4a3951e")
		withdrawal_address = common.HexToAddress("0xa7e558cc6efa1c41270ef4aa227b3dd6b4a3951e")
		//depositAmount      = 32000000000000
		signature = common.HexToBlsSig("0xb9221f2308c1e1655a8e1977f32241384fa77efedbb3079bcc9a95930152ee87" +
			"f341134a4e59c3e312ee5c2197732ea30d9aac2993cc4aad75335009815d07a8735f96c6dde443ba3a10f5523c4d00f6b3a7b48af" +
			"5a42795183ab5aa2f1b2dd1")
		depositData = "f4019728bc733c8fcedde0c3a33dac12da3ebbaa0eb74d813a34b600520e7976a260d85f057687e8c923d52c787155153" +
			"48da7e558cc6efa1c41270ef4aa227b3dd6b4a3951ea7e558cc6efa1c41270ef4aa227b3dd6b4a3951eb9221f2308c1e1655a8e1977" +
			"f32241384fa77efedbb3079bcc9a95930152ee87f341134a4e59c3e312ee5c2197732ea30d9aac2993cc4aad75335009815d07a8735" +
			"f96c6dde443ba3a10f5523c4d00f6b3a7b48af5a42795183ab5aa2f1b2dd1" // + "0000000400000000"
	)

	type decodedOp struct {
		pubkey             common.BlsPubKey // validator public key
		creator_address    common.Address   // attached creator account
		withdrawal_address common.Address   // attached withdrawal credentials
		signature          common.BlsSignature
	}

	cases := []operationTestCase{
		{
			caseName: "OK",
			decoded: decodedOp{
				pubkey:             pubkey,
				creator_address:    creator_address,
				withdrawal_address: withdrawal_address,
				signature:          signature,
			},
			encoded: hexutils.HexToBytes(depositData),
			errs:    []error{},
		},
		{
			caseName: "ErrNoPubKey",
			decoded: decodedOp{
				creator_address:    creator_address,
				withdrawal_address: withdrawal_address,
				signature:          signature,
			},
			encoded: hexutils.HexToBytes(""),
			errs:    []error{ErrNoPubKey},
		},
		{
			caseName: "ErrNoCreatorAddress",
			decoded: decodedOp{
				pubkey:             pubkey,
				withdrawal_address: withdrawal_address,
				signature:          signature,
			},
			encoded: hexutils.HexToBytes(""),
			errs:    []error{ErrNoCreatorAddress},
		},
		{
			caseName: "ErrNoWithdrawalAddress",
			decoded: decodedOp{
				pubkey:          pubkey,
				creator_address: creator_address,
				signature:       signature,
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
			},
			encoded: hexutils.HexToBytes(""),
			errs:    []error{ErrNoSignature},
		},
	}

	operationEncode := func(b []byte, i interface{}) error {
		o := i.(decodedOp)
		createOp, err := NewDepositOperation(
			o.pubkey,
			o.creator_address,
			o.withdrawal_address,
			o.signature,
			nil,
		)
		if err != nil {
			return err
		}

		return equalOpBytes(createOp, b)
	}

	operationDecode := func(b []byte, i interface{}) error {
		op, err := DecodeBytes(b)
		testutils.AssertNoError(t, err)

		o := i.(decodedOp)
		opDecoded, ok := op.(Deposit)
		if !ok {
			return errors.New("invalid operation type")
		}
		err = checkOpCode(b, opDecoded)
		testutils.AssertNoError(t, err)
		testutils.AssertEqual(t, opDecoded.PubKey().Bytes(), o.pubkey.Bytes())
		testutils.AssertEqual(t, opDecoded.CreatorAddress().Bytes(), o.creator_address.Bytes())
		testutils.AssertEqual(t, opDecoded.WithdrawalAddress().Bytes(), o.withdrawal_address.Bytes())
		testutils.AssertEqual(t, opDecoded.Signature().Bytes(), o.signature.Bytes())

		return nil
	}

	startSubTests(t, cases, operationEncode, operationDecode)
}

func TestDelegatingStake_Marshaling(t *testing.T) {
	profitShare, stakeShare, exit, withdrawal := TestParamsDelegatingStakeRules()

	var (
		pubkey             = common.HexToBlsPubKey("0x9728bc733c8fcedde0c3a33dac12da3ebbaa0eb74d813a34b600520e7976a260d85f057687e8c923d52c78715515348d")
		creator_address    = common.HexToAddress("0xa7e558cc6efa1c41270ef4aa227b3dd6b4a3951e")
		withdrawal_address = common.HexToAddress("0x1111111111111111111111111111111111111111")
		signature          = common.HexToBlsSig("0xb9221f2308c1e1655a8e1977f32241384fa77efedbb3079bcc9a95930152ee87" +
			"f341134a4e59c3e312ee5c2197732ea30d9aac2993cc4aad75335009815d07a8735f96c6dde443ba3a10f5523c4d00f6b3a7b48af" +
			"5a42795183ab5aa2f1b2dd1")
	)

	rules, err := NewDelegatingStakeRules(profitShare, stakeShare, exit, withdrawal)
	testutils.AssertNoError(t, err)
	trialRules, err := NewDelegatingStakeRules(profitShare, stakeShare, exit, withdrawal)
	testutils.AssertNoError(t, err)
	delegate, err := NewDelegatingStakeData(rules, 321, trialRules)
	testutils.AssertNoError(t, err)

	delegatedDeposit, err := NewDepositOperation(pubkey, creator_address, withdrawal_address, signature, delegate)
	testutils.AssertNoError(t, err)

	bin, err := delegatedDeposit.MarshalBinary()
	testutils.AssertNoError(t, err)

	//fmt.Println(fmt.Sprintf("%#x", bin))
	//fmt.Println(fmt.Sprintf("binary_size=%d", len(bin)))

	unmarshaled := &depositOperation{}
	err = unmarshaled.UnmarshalBinary(bin)
	testutils.AssertNoError(t, err)

	testutils.AssertEqual(t, delegatedDeposit.PubKey(), unmarshaled.PubKey())
	testutils.AssertEqual(t, delegatedDeposit.CreatorAddress(), unmarshaled.CreatorAddress())
	testutils.AssertEqual(t, delegatedDeposit.WithdrawalAddress(), unmarshaled.WithdrawalAddress())
	testutils.AssertEqual(t, delegatedDeposit.Signature(), unmarshaled.Signature())
	testutils.AssertEqual(t, delegatedDeposit.DelegatingStake().Rules, unmarshaled.DelegatingStake().Rules)
	testutils.AssertEqual(t, delegatedDeposit.DelegatingStake().TrialRules, unmarshaled.DelegatingStake().TrialRules)
	testutils.AssertEqual(t, delegatedDeposit.DelegatingStake().TrialPeriod, unmarshaled.DelegatingStake().TrialPeriod)
}

func TestDelegatingStake_nilMarshaling(t *testing.T) {
	var (
		pubkey             = common.HexToBlsPubKey("0x9728bc733c8fcedde0c3a33dac12da3ebbaa0eb74d813a34b600520e7976a260d85f057687e8c923d52c78715515348d")
		creator_address    = common.HexToAddress("0xa7e558cc6efa1c41270ef4aa227b3dd6b4a3951e")
		withdrawal_address = common.HexToAddress("0x1111111111111111111111111111111111111111")
		signature          = common.HexToBlsSig("0xb9221f2308c1e1655a8e1977f32241384fa77efedbb3079bcc9a95930152ee87" +
			"f341134a4e59c3e312ee5c2197732ea30d9aac2993cc4aad75335009815d07a8735f96c6dde443ba3a10f5523c4d00f6b3a7b48af" +
			"5a42795183ab5aa2f1b2dd1")
	)

	var delegate *DelegatingStakeData = nil

	delegatedDeposit, err := NewDepositOperation(pubkey, creator_address, withdrawal_address, signature, delegate)
	testutils.AssertNoError(t, err)

	bin, err := delegatedDeposit.MarshalBinary()
	testutils.AssertNoError(t, err)

	unmarshaled := &depositOperation{}
	err = unmarshaled.UnmarshalBinary(bin)
	testutils.AssertNoError(t, err)

	testutils.AssertEqual(t, delegatedDeposit.PubKey(), unmarshaled.PubKey())
	testutils.AssertEqual(t, delegatedDeposit.CreatorAddress(), unmarshaled.CreatorAddress())
	testutils.AssertEqual(t, delegatedDeposit.WithdrawalAddress(), unmarshaled.WithdrawalAddress())
	testutils.AssertEqual(t, delegatedDeposit.Signature(), unmarshaled.Signature())
	testutils.AssertEqual(t, delegate, delegatedDeposit.DelegatingStake())
	testutils.AssertEqual(t, delegatedDeposit.DelegatingStake(), unmarshaled.DelegatingStake())
}

func TestDepositData_withDelegatingStake(t *testing.T) {
	profitShare, stakeShare, exit, withdrawal := TestParamsDelegatingStakeRules()
	var (
		pubkey             = common.HexToBlsPubKey("0x9728bc733c8fcedde0c3a33dac12da3ebbaa0eb74d813a34b600520e7976a260d85f057687e8c923d52c78715515348d")
		creator_address    = common.HexToAddress("0xa7e558cc6efa1c41270ef4aa227b3dd6b4a3951e")
		withdrawal_address = common.HexToAddress("0xa7e558cc6efa1c41270ef4aa227b3dd6b4a3951e")
		signature          = common.HexToBlsSig("0xb9221f2308c1e1655a8e1977f32241384fa77efedbb3079bcc9a95930152ee87" +
			"f341134a4e59c3e312ee5c2197732ea30d9aac2993cc4aad75335009815d07a8735f96c6dde443ba3a10f5523c4d00f6b3a7b48af" +
			"5a42795183ab5aa2f1b2dd1")

		binDelegate = hexutils.HexToBytes("f4019728bc733c8fcedde0c3a33dac12da3ebbaa0eb74d813a34b600520e7976a260d85f0" +
			"57687e8c923d52c78715515348da7e558cc6efa1c41270ef4aa227b3dd6b4a3951ea7e558cc6efa1c41270ef4aa227b3dd6b4a3951" +
			"eb9221f2308c1e1655a8e1977f32241384fa77efedbb3079bcc9a95930152ee87f341134a4e59c3e312ee5c2197732ea30d9aac299" +
			"3cc4aad75335009815d07a8735f96c6dde443ba3a10f5523c4d00f6b3a7b48af5a42795183ab5aa2f1b2dd10000016400000160f90" +
			"15db8abf8a9f8939411111111111111111111111111111111111111119422222222222222222222222222222222222222229433333" +
			"3333333333333333333333333333333333394444444444444444444444444444444444444444494555555555555555555555555555" +
			"5555555555555946666666666666666666666666666666666666666947777777777777777777777777777777777777777870a1e3c0" +
			"000000087000000461e000081a081c0820141b8abf8a9f893941111111111111111111111111111111111111111942222222222222" +
			"2222222222222222222222222229433333333333333333333333333333333333333339444444444444444444444444444444444444" +
			"4444494555555555555555555555555555555555555555594666666666666666666666666666666666666666694777777777777777" +
			"7777777777777777777777777870a1e3c0000000087000000461e000081a081c0")

		binDelegate2 = hexutils.HexToBytes("f4019728bc733c8fcedde0c3a33dac12da3ebbaa0eb74d813a34b600520e7976a260d85" +
			"f057687e8c923d52c78715515348da7e558cc6efa1c41270ef4aa227b3dd6b4a3951ea7e558cc6efa1c41270ef4aa227b3dd6b4a39" +
			"51eb9221f2308c1e1655a8e1977f32241384fa77efedbb3079bcc9a95930152ee87f341134a4e59c3e312ee5c2197732ea30d9aac2" +
			"993cc4aad75335009815d07a8735f96c6dde443ba3a10f5523c4d00f6b3a7b48af5a42795183ab5aa2f1b2dd1000000bd000000b9f" +
			"8b7b8abf8a9f8939411111111111111111111111111111111111111119422222222222222222222222222222222222222229433333" +
			"3333333333333333333333333333333333394444444444444444444444444444444444444444494555555555555555555555555555" +
			"5555555555555946666666666666666666666666666666666666666947777777777777777777777777777777777777777870a1e3c0" +
			"000000087000000461e000081a081c082014186c5c080808080")

		binDelegate3 = hexutils.HexToBytes("f4019728bc733c8fcedde0c3a33dac12da3ebbaa0eb74d813a34b600520e7976a260d85f" +
			"057687e8c923d52c78715515348da7e558cc6efa1c41270ef4aa227b3dd6b4a3951ea7e558cc6efa1c41270ef4aa227b3dd6b4a395" +
			"1eb9221f2308c1e1655a8e1977f32241384fa77efedbb3079bcc9a95930152ee87f341134a4e59c3e312ee5c2197732ea30d9aac29" +
			"93cc4aad75335009815d07a8735f96c6dde443ba3a10f5523c4d00f6b3a7b48af5a42795183ab5aa2f1b2dd1000000bd000000b9f8" +
			"b786c5c080808080820141b8abf8a9f893941111111111111111111111111111111111111111942222222222222222222222222222" +
			"2222222222229433333333333333333333333333333333333333339444444444444444444444444444444444444444449455555555" +
			"5555555555555555555555555555555594666666666666666666666666666666666666666694777777777777777777777777777777" +
			"7777777777870a1e3c0000000087000000461e000081a081c0")
	)

	rules, err := NewDelegatingStakeRules(profitShare, stakeShare, exit, withdrawal)
	testutils.AssertNoError(t, err)
	trialRules, err := NewDelegatingStakeRules(profitShare, stakeShare, exit, withdrawal)
	testutils.AssertNoError(t, err)
	delegate, err := NewDelegatingStakeData(rules, 321, trialRules)
	testutils.AssertNoError(t, err)
	delegate2, err := NewDelegatingStakeData(rules, 321, nil)
	testutils.AssertNoError(t, err)
	delegate3, err := NewDelegatingStakeData(nil, 321, trialRules)
	testutils.AssertNoError(t, err)

	type decodedOp struct {
		pubKey          common.BlsPubKey // validator public key
		creatorAddress  common.Address   // attached creator account
		withdrawal      common.Address   // attached withdrawal credentials
		signature       common.BlsSignature
		delegatingStake *DelegatingStakeData
	}

	cases := []operationTestCase{
		{
			caseName: "OK",
			decoded: decodedOp{
				pubKey:          pubkey,
				creatorAddress:  creator_address,
				withdrawal:      withdrawal_address,
				signature:       signature,
				delegatingStake: delegate,
			},
			encoded: binDelegate,
			errs:    []error{},
		},
		{
			caseName: "OK-2",
			decoded: decodedOp{
				pubKey:          pubkey,
				creatorAddress:  creator_address,
				withdrawal:      withdrawal_address,
				signature:       signature,
				delegatingStake: delegate2,
			},
			encoded: binDelegate2,
			errs:    []error{},
		},
		{
			caseName: "ErrBadProfitShare",
			decoded: decodedOp{
				pubKey:          pubkey,
				creatorAddress:  creator_address,
				withdrawal:      withdrawal_address,
				signature:       signature,
				delegatingStake: delegate3,
			},
			encoded: binDelegate3,
			errs:    []error{ErrBadProfitShare},
		},
	}

	operationEncode := func(b []byte, i interface{}) error {
		o := i.(decodedOp)
		createOp, err := NewDepositOperation(
			o.pubKey,
			o.creatorAddress,
			o.withdrawal,
			o.signature,
			o.delegatingStake,
		)
		if err != nil {
			return err
		}

		return equalOpBytes(createOp, b)
	}

	operationDecode := func(b []byte, i interface{}) error {
		op, err := DecodeBytes(b)
		testutils.AssertNoError(t, err)

		o := i.(decodedOp)
		opDecoded, ok := op.(Deposit)
		if !ok {
			return fmt.Errorf("invalid operation type")
		}
		err = checkOpCode(b, opDecoded)
		testutils.AssertNoError(t, err)
		testutils.AssertEqual(t, o.pubKey.Bytes(), opDecoded.PubKey().Bytes())
		testutils.AssertEqual(t, o.creatorAddress.Bytes(), opDecoded.CreatorAddress().Bytes())
		testutils.AssertEqual(t, o.withdrawal.Bytes(), opDecoded.WithdrawalAddress().Bytes())
		testutils.AssertEqual(t, o.signature.Bytes(), opDecoded.Signature().Bytes())

		testutils.AssertEqual(t, o.delegatingStake.Copy(), opDecoded.DelegatingStake())

		o_rules, err := o.delegatingStake.MarshalBinary()
		testutils.AssertNoError(t, err)
		d_rules, err := opDecoded.DelegatingStake().MarshalBinary()
		testutils.AssertNoError(t, err)
		testutils.AssertEqual(t, o_rules, d_rules)

		return nil
	}

	startSubTests(t, cases, operationEncode, operationDecode)
}
