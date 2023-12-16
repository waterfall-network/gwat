package operation

import (
	"fmt"
	"github.com/status-im/keycard-go/hexutils"
	"testing"
	"time"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/tests/testutils"
)

func TestDelegateStake_Marshaling(t *testing.T) {
	defer func(tStart time.Time) {
		fmt.Println("TOTAL TIME",
			"elapsed", common.PrettyDuration(time.Since(tStart)),
		)
	}(time.Now())
	var (
		pubkey          = common.HexToBlsPubKey("0x9728bc733c8fcedde0c3a33dac12da3ebbaa0eb74d813a34b600520e7976a260d85f057687e8c923d52c78715515348d")
		creator_address = common.HexToAddress("0xa7e558cc6efa1c41270ef4aa227b3dd6b4a3951e")
		signature       = common.HexToBlsSig("0xb9221f2308c1e1655a8e1977f32241384fa77efedbb3079bcc9a95930152ee87" +
			"f341134a4e59c3e312ee5c2197732ea30d9aac2993cc4aad75335009815d07a8735f96c6dde443ba3a10f5523c4d00f6b3a7b48af" +
			"5a42795183ab5aa2f1b2dd1")

		//DelegateStakeRules
		profitShare = map[common.Address]uint8{
			common.HexToAddress("0x1111111111111111111111111111111111111111"): 10,
			common.HexToAddress("0x2222222222222222222222222222222222222222"): 30,
			common.HexToAddress("0x3333333333333333333333333333333333333333"): 60,
		}
		stakeShare = map[common.Address]uint8{
			common.HexToAddress("0x4444444444444444444444444444444444444444"): 70,
			common.HexToAddress("0x5555555555555555555555555555555555555555"): 30,
		}
		exit       = []common.Address{common.HexToAddress("0x6666666666666666666666666666666666666666")}
		withdrawal = []common.Address{common.HexToAddress("0x7777777777777777777777777777777777777777")}
	)

	rules, err := NewDelegateStakeRules(profitShare, stakeShare, exit, withdrawal)
	testutils.AssertNoError(t, err)
	trialRules, err := NewDelegateStakeRules(profitShare, stakeShare, exit, withdrawal)
	testutils.AssertNoError(t, err)

	dsr, err := NewDelegateStakeOperation(pubkey, creator_address, signature, rules, 321, trialRules)
	testutils.AssertNoError(t, err)

	bin, err := dsr.MarshalBinary()
	testutils.AssertNoError(t, err)

	//fmt.Println(fmt.Sprintf("%#x", bin))
	//fmt.Println(fmt.Sprintf("binary_size=%d", len(bin)))

	unmarshaled := &delegateStakeOperation{}
	err = unmarshaled.UnmarshalBinary(bin)
	testutils.AssertNoError(t, err)

	testutils.AssertEqual(t, dsr.PubKey(), unmarshaled.PubKey())
	testutils.AssertEqual(t, dsr.CreatorAddress(), unmarshaled.CreatorAddress())
	testutils.AssertEqual(t, dsr.Signature(), unmarshaled.Signature())
	testutils.AssertEqual(t, dsr.Rules(), unmarshaled.Rules())
	testutils.AssertEqual(t, dsr.TrialRules(), unmarshaled.TrialRules())
}

func TestDelegateStake(t *testing.T) {
	var (
		pubkey          = common.HexToBlsPubKey("0x9728bc733c8fcedde0c3a33dac12da3ebbaa0eb74d813a34b600520e7976a260d85f057687e8c923d52c78715515348d")
		creator_address = common.HexToAddress("0xa7e558cc6efa1c41270ef4aa227b3dd6b4a3951e")
		signature       = common.HexToBlsSig("0xb9221f2308c1e1655a8e1977f32241384fa77efedbb3079bcc9a95930152ee87" +
			"f341134a4e59c3e312ee5c2197732ea30d9aac2993cc4aad75335009815d07a8735f96c6dde443ba3a10f5523c4d00f6b3a7b48af" +
			"5a42795183ab5aa2f1b2dd1")

		//DelegateStakeRules
		profitShare = map[common.Address]uint8{
			common.HexToAddress("0x1111111111111111111111111111111111111111"): 10,
			common.HexToAddress("0x2222222222222222222222222222222222222222"): 30,
			common.HexToAddress("0x3333333333333333333333333333333333333333"): 60,
		}
		stakeShare = map[common.Address]uint8{
			common.HexToAddress("0x4444444444444444444444444444444444444444"): 70,
			common.HexToAddress("0x5555555555555555555555555555555555555555"): 30,
		}
		exit       = []common.Address{common.HexToAddress("0x6666666666666666666666666666666666666666")}
		withdrawal = []common.Address{common.HexToAddress("0x7777777777777777777777777777777777777777")}

		binDelegate = hexutils.HexToBytes("f42000000208f90205b09728bc733c8fcedde0c3a33dac12da3ebbaa0eb74d813a34b6" +
			"00520e7976a260d85f057687e8c923d52c78715515348d94a7e558cc6efa1c41270ef4aa227b3dd6b4a3951eb860b9221f2308c1e" +
			"1655a8e1977f32241384fa77efedbb3079bcc9a95930152ee87f341134a4e59c3e312ee5c2197732ea30d9aac2993cc4aad753350" +
			"09815d07a8735f96c6dde443ba3a10f5523c4d00f6b3a7b48af5a42795183ab5aa2f1b2dd1b8abf8a9f8939411111111111111111" +
			"111111111111111111111119422222222222222222222222222222222222222229433333333333333333333333333333333333333" +
			"339444444444444444444444444444444444444444449455555555555555555555555555555555555555559466666666666666666" +
			"66666666666666666666666947777777777777777777777777777777777777777870a1e3c0000000087000000461e000081a081c0" +
			"820141b8abf8a9f893941111111111111111111111111111111111111111942222222222222222222222222222222222222222943" +
			"333333333333333333333333333333333333333944444444444444444444444444444444444444444945555555555555555555555" +
			"555555555555555555946666666666666666666666666666666666666666947777777777777777777777777777777777777777870" +
			"a1e3c0000000087000000461e000081a081c0")
		binDelegate_2 = hexutils.HexToBytes("f42000000162f9015fb09728bc733c8fcedde0c3a33dac12da3ebbaa0eb74d813a34b6" +
			"00520e7976a260d85f057687e8c923d52c78715515348d94a7e558cc6efa1c41270ef4aa227b3dd6b4a3951eb860b9221f2308c1e" +
			"1655a8e1977f32241384fa77efedbb3079bcc9a95930152ee87f341134a4e59c3e312ee5c2197732ea30d9aac2993cc4aad753350" +
			"09815d07a8735f96c6dde443ba3a10f5523c4d00f6b3a7b48af5a42795183ab5aa2f1b2dd1b8abf8a9f8939411111111111111111" +
			"111111111111111111111119422222222222222222222222222222222222222229433333333333333333333333333333333333333" +
			"339444444444444444444444444444444444444444449455555555555555555555555555555555555555559466666666666666666" +
			"66666666666666666666666947777777777777777777777777777777777777777870a1e3c0000000087000000461e000081a081c0" +
			"82014186c5c080800101")
	)

	rules, err := NewDelegateStakeRules(profitShare, stakeShare, exit, withdrawal)
	testutils.AssertNoError(t, err)
	trialRules, err := NewDelegateStakeRules(profitShare, stakeShare, exit, withdrawal)
	testutils.AssertNoError(t, err)

	type decodedOp struct {
		pubKey         common.BlsPubKey
		creatorAddress common.Address
		signature      common.BlsSignature
		rules          *DelegateStakeRules
		trialPeriod    uint64
		trialRules     *DelegateStakeRules
	}

	cases := []operationTestCase{
		{
			caseName: "OK",
			decoded: decodedOp{
				pubKey:         pubkey,
				creatorAddress: creator_address,
				signature:      signature,
				rules:          rules,
				trialPeriod:    321,
				trialRules:     trialRules,
			},
			encoded: binDelegate,
			errs:    []error{},
		},
		{
			caseName: "OK-2",
			decoded: decodedOp{
				pubKey:         pubkey,
				creatorAddress: creator_address,
				signature:      signature,
				rules:          rules,
				trialPeriod:    321,
				trialRules:     nil,
			},
			encoded: binDelegate_2,
			errs:    []error{},
		},
		{
			caseName: "OK",
			decoded: decodedOp{
				pubKey:         pubkey,
				creatorAddress: creator_address,
				signature:      signature,
				rules:          nil,
				trialPeriod:    321,
				trialRules:     trialRules,
			},
			encoded: binDelegate,
			errs:    []error{ErrBadProfitShare},
		},

		{
			caseName: "ErrNoPubKey",
			decoded: decodedOp{
				//pubKey:         pubkey,
				creatorAddress: creator_address,
				signature:      signature,
				rules:          rules,
				trialPeriod:    321,
				trialRules:     trialRules,
			},
			encoded: hexutils.HexToBytes(""),
			errs:    []error{ErrNoPubKey},
		},
		{
			caseName: "ErrNoCreatorAddress",
			decoded: decodedOp{
				pubKey: pubkey,
				//creatorAddress: creator_address,
				signature:   signature,
				rules:       rules,
				trialPeriod: 321,
				trialRules:  trialRules,
			},
			encoded: hexutils.HexToBytes(""),
			errs:    []error{ErrNoCreatorAddress},
		},
		{
			caseName: "ErrNoSignature",
			decoded: decodedOp{
				pubKey:         pubkey,
				creatorAddress: creator_address,
				//signature:      signature,
				rules:       rules,
				trialPeriod: 321,
				trialRules:  trialRules,
			},
			encoded: hexutils.HexToBytes(""),
			errs:    []error{ErrNoSignature},
		},
	}

	operationEncode := func(b []byte, i interface{}) error {
		o := i.(decodedOp)
		createOp, err := NewDelegateStakeOperation(
			o.pubKey,
			o.creatorAddress,
			o.signature,
			o.rules,
			o.trialPeriod,
			o.trialRules,
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
		opDecoded, ok := op.(DelegateStake)
		if !ok {
			return fmt.Errorf("invalid operation type")
		}
		err = checkOpCode(b, opDecoded)
		testutils.AssertNoError(t, err)
		testutils.AssertEqual(t, o.pubKey.Bytes(), opDecoded.PubKey().Bytes())
		testutils.AssertEqual(t, o.creatorAddress.Bytes(), opDecoded.CreatorAddress().Bytes())
		testutils.AssertEqual(t, o.signature.Bytes(), opDecoded.Signature().Bytes())

		testutils.AssertEqual(t, o.trialPeriod, opDecoded.TrialPeriod())

		o_rules, err := o.rules.MarshalBinary()
		testutils.AssertNoError(t, err)
		d_rules, err := opDecoded.Rules().MarshalBinary()
		testutils.AssertNoError(t, err)
		testutils.AssertEqual(t, o_rules, d_rules)

		return nil
	}

	startSubTests(t, cases, operationEncode, operationDecode)
}

/** DelegateStakeRules */
func TestDelegateStakeRules_init(t *testing.T) {
	var (
		profitShare = map[common.Address]uint8{
			common.HexToAddress("0x1111111111111111111111111111111111111111"): 10,
			common.HexToAddress("0x2222222222222222222222222222222222222222"): 30,
			common.HexToAddress("0x3333333333333333333333333333333333333333"): 60,
		}
		stakeShare = map[common.Address]uint8{
			common.HexToAddress("0x4444444444444444444444444444444444444444"): 70,
			common.HexToAddress("0x5555555555555555555555555555555555555555"): 30,
		}
		exit       = []common.Address{common.HexToAddress("0x6666666666666666666666666666666666666666")}
		withdrawal = []common.Address{common.HexToAddress("0x7777777777777777777777777777777777777777")}
	)

	dsr, err := NewDelegateStakeRules(profitShare, stakeShare, exit, withdrawal)
	testutils.AssertNoError(t, err)

	testutils.AssertEqual(t, profitShare, dsr.ProfitShare())
	testutils.AssertEqual(t, stakeShare, dsr.StakeShare())
	testutils.AssertEqual(t, exit, dsr.Exit())
	testutils.AssertEqual(t, withdrawal, dsr.Withdrawal())
}

func TestDelegateStakeRules_Copy(t *testing.T) {
	var (
		profitShare = map[common.Address]uint8{
			common.HexToAddress("0x1111111111111111111111111111111111111111"): 10,
			common.HexToAddress("0x2222222222222222222222222222222222222222"): 30,
			common.HexToAddress("0x3333333333333333333333333333333333333333"): 60,
		}
		stakeShare = map[common.Address]uint8{
			common.HexToAddress("0x4444444444444444444444444444444444444444"): 70,
			common.HexToAddress("0x5555555555555555555555555555555555555555"): 30,
		}
		exit       = []common.Address{common.HexToAddress("0x6666666666666666666666666666666666666666")}
		withdrawal = []common.Address{common.HexToAddress("0x7777777777777777777777777777777777777777")}
	)

	dsr, err := NewDelegateStakeRules(profitShare, stakeShare, exit, withdrawal)
	testutils.AssertNoError(t, err)

	cpy := dsr.Copy()
	testutils.AssertEqual(t, dsr.ProfitShare(), cpy.ProfitShare())
	testutils.AssertEqual(t, dsr.StakeShare(), cpy.StakeShare())
	testutils.AssertEqual(t, dsr.Exit(), cpy.Exit())
	testutils.AssertEqual(t, dsr.Withdrawal(), cpy.Withdrawal())
}

func TestDelegateStakeRules_validate(t *testing.T) {
	var (
		profitShare = map[common.Address]uint8{
			common.HexToAddress("0x1111111111111111111111111111111111111111"): 10,
			common.HexToAddress("0x2222222222222222222222222222222222222222"): 30,
			common.HexToAddress("0x3333333333333333333333333333333333333333"): 60,
		}
		stakeShare = map[common.Address]uint8{
			common.HexToAddress("0x4444444444444444444444444444444444444444"): 70,
			common.HexToAddress("0x5555555555555555555555555555555555555555"): 30,
		}
		exit       = []common.Address{common.HexToAddress("0x6666666666666666666666666666666666666666")}
		withdrawal = []common.Address{common.HexToAddress("0x7777777777777777777777777777777777777777")}
	)

	type decodedOp struct {
		profitShare map[common.Address]uint8
		stakeShare  map[common.Address]uint8
		exit        []common.Address
		withdrawal  []common.Address
	}

	cases := []operationTestCase{
		{
			caseName: "OK",
			decoded: decodedOp{
				profitShare: profitShare,
				stakeShare:  stakeShare,
				exit:        exit,
				withdrawal:  withdrawal,
			},
			encoded: nil,
			errs:    []error{},
		},
		{
			caseName: "ErrBadProfitShare",
			decoded: decodedOp{
				profitShare: nil,
				stakeShare:  stakeShare,
				exit:        exit,
				withdrawal:  withdrawal,
			},
			encoded: nil,
			errs:    []error{ErrBadProfitShare},
		},
		{
			caseName: "ErrBadProfitShare",
			decoded: decodedOp{
				profitShare: map[common.Address]uint8{
					common.HexToAddress("0x1111111111111111111111111111111111111111"): 10,
					common.HexToAddress("0x2222222222222222222222222222222222222222"): 30,
				},
				stakeShare: stakeShare,
				exit:       exit,
				withdrawal: withdrawal,
			},
			encoded: nil,
			errs:    []error{ErrBadProfitShare},
		},
		{
			caseName: "ErrBadStakeShare",
			decoded: decodedOp{
				profitShare: profitShare,
				stakeShare: map[common.Address]uint8{
					common.HexToAddress("0x1111111111111111111111111111111111111111"): 60,
					common.HexToAddress("0x2222222222222222222222222222222222222222"): 60,
				},
				exit:       exit,
				withdrawal: withdrawal,
			},
			encoded: nil,
			errs:    []error{ErrBadStakeShare},
		},
		{
			caseName: "ErrNoExitRoles",
			decoded: decodedOp{
				profitShare: profitShare,
				stakeShare:  stakeShare,
				exit:        nil,
				withdrawal:  withdrawal,
			},
			encoded: nil,
			errs:    []error{ErrNoExitRoles},
		},
		{
			caseName: "ErrNoWithdrawalRoles",
			decoded: decodedOp{
				profitShare: profitShare,
				stakeShare:  stakeShare,
				exit:        exit,
				withdrawal:  nil,
			},
			encoded: nil,
			errs:    []error{ErrNoWithdrawalRoles},
		},
	}

	operationEncode := func(b []byte, i interface{}) error {
		o := i.(decodedOp)
		createOp, err := NewDelegateStakeRules(
			o.profitShare,
			o.stakeShare,
			o.exit,
			o.withdrawal,
		)
		if err != nil {
			return err
		}
		return createOp.validate()
	}

	operationDecode := func(b []byte, i interface{}) error {
		return nil
	}

	startSubTests(t, cases, operationEncode, operationDecode)
}

func TestDelegateStakeRules_Marshaling(t *testing.T) {
	defer func(tStart time.Time) {
		fmt.Println("TOTAL TIME",
			"elapsed", common.PrettyDuration(time.Since(tStart)),
		)
	}(time.Now())
	var (
		profitShare = map[common.Address]uint8{
			common.HexToAddress("0x1111111111111111111111111111111111111111"): 10,
			common.HexToAddress("0x2222222222222222222222222222222222222222"): 30,
			common.HexToAddress("0x3333333333333333333333333333333333333333"): 60,
		}
		stakeShare = map[common.Address]uint8{
			common.HexToAddress("0x4444444444444444444444444444444444444444"): 70,
			common.HexToAddress("0x5555555555555555555555555555555555555555"): 30,
		}
		exit       = []common.Address{common.HexToAddress("0x6666666666666666666666666666666666666666")}
		withdrawal = []common.Address{common.HexToAddress("0x7777777777777777777777777777777777777777")}

		encoded = hexutils.HexToBytes("f8a9f893941111111111111111111111111111111111111111" +
			"942222222222222222222222222222222222222222" +
			"943333333333333333333333333333333333333333" +
			"944444444444444444444444444444444444444444" +
			"945555555555555555555555555555555555555555" +
			"946666666666666666666666666666666666666666" +
			"947777777777777777777777777777777777777777" +
			"870a1e3c0000000087000000461e000081a081c0")
	)

	dsr, err := NewDelegateStakeRules(profitShare, stakeShare, exit, withdrawal)
	testutils.AssertNoError(t, err)

	bin, err := dsr.MarshalBinary()
	testutils.AssertNoError(t, err)

	//fmt.Println(fmt.Sprintf("%#x", bin))
	fmt.Println(fmt.Sprintf("binary_size=%d", len(bin)))
	testutils.AssertEqual(t, encoded, bin)

	unmarshaled := &DelegateStakeRules{}
	err = unmarshaled.UnmarshalBinary(bin)
	testutils.AssertNoError(t, err)

	testutils.AssertEqual(t, dsr.ProfitShare(), unmarshaled.ProfitShare())
	testutils.AssertEqual(t, dsr.StakeShare(), unmarshaled.StakeShare())
	testutils.AssertEqual(t, dsr.Exit(), unmarshaled.Exit())
	testutils.AssertEqual(t, dsr.Withdrawal(), unmarshaled.Withdrawal())
}
