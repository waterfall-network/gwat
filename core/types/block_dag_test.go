package types

import (
	"fmt"
	"testing"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
)

func TestBlockDAG(t *testing.T) {
	blockDag := &BlockDAG{
		Hash: common.Hash{
			0x33, 0x33, 0x33, 0x33, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x33,
		},
		Height: uint64(300),
		Slot:   uint64(150),
		CpHash: common.Hash{
			0x11, 0x11, 0x11, 0x11, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x11,
		},
		CpHeight: uint64(255),
		OrderedAncestorsHashes: common.HashArray{
			common.Hash{
				0x11, 0x11, 0x11, 0x11, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x11,
			},
			common.Hash{
				0x22, 0x22, 0x22, 0x22, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x22,
			},
			common.Hash{
				0x11, 0x11, 0x11, 0x11, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x11,
			},
			common.Hash{
				0x22, 0x22, 0x22, 0x22, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x22,
			},
			common.Hash{
				0x11, 0x11, 0x11, 0x11, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x11,
			},
			common.Hash{
				0x22, 0x22, 0x22, 0x22, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x22,
			},
			common.Hash{
				0x11, 0x11, 0x11, 0x11, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x11,
			},
			common.Hash{
				0x22, 0x22, 0x22, 0x22, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x22,
			},
			common.Hash{
				0x11, 0x11, 0x11, 0x11, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x11,
			},
			common.Hash{
				0x22, 0x22, 0x22, 0x22, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x22,
			},
			common.Hash{
				0x11, 0x11, 0x11, 0x11, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x11,
			},
			common.Hash{
				0x22, 0x22, 0x22, 0x22, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x22,
			},
		},
	}
	enc := []byte{
		0x33, 0x33, 0x33, 0x33, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x33,

		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x2c,

		0x11, 0x11, 0x11, 0x11, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x11,

		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xff,

		0x00, 0x00, 0x00, 0x0c,

		0x11, 0x11, 0x11, 0x11, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x11,

		0x22, 0x22, 0x22, 0x22, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x22,

		0x11, 0x11, 0x11, 0x11, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x11,

		0x22, 0x22, 0x22, 0x22, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x22,

		0x11, 0x11, 0x11, 0x11, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x11,

		0x22, 0x22, 0x22, 0x22, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x22,

		0x11, 0x11, 0x11, 0x11, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x11,

		0x22, 0x22, 0x22, 0x22, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x22,

		0x11, 0x11, 0x11, 0x11, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x11,

		0x22, 0x22, 0x22, 0x22, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x22,

		0x11, 0x11, 0x11, 0x11, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x11,

		0x22, 0x22, 0x22, 0x22, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x22,

		0x00, 0x00, 0x00, 0x04,

		0x11, 0x11, 0x11, 0x11, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x11,

		0x22, 0x22, 0x22, 0x22, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x22,

		0x11, 0x11, 0x11, 0x11, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x11,

		0x22, 0x22, 0x22, 0x22, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x22,
	}
	// blockDag.ToBytes()
	res := fmt.Sprintf("%v", blockDag.ToBytes())
	exp := fmt.Sprintf("%v", enc)
	if res != exp {
		t.Fatalf("blockDag.ToBytes failed, got %v != %v", res, exp)
	}

	// blockDag.SetBytes()
	restored := new(BlockDAG).SetBytes(blockDag.ToBytes())
	res = fmt.Sprintf("%v", restored)
	exp = fmt.Sprintf("%v", blockDag)
	if res != exp {
		t.Fatalf("empty uncle hash is wrong, got %#v != %#v", res, exp)
	}

	test := common.HashArray{
		common.Hash{
			0x11, 0x11, 0x11, 0x11, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x11,
		},
		common.Hash{
			0x22, 0x22, 0x22, 0x22, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x22,
		},
		common.Hash{
			0x33, 0x11, 0x11, 0x11, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x11,
		},
		common.Hash{
			0x44, 0x22, 0x22, 0x22, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x22,
		}}

	val := test[len(test)-1-2]
	fmt.Print("\n")
	fmt.Printf("len(test)-4=%v \n", len(test) <= 4)
	fmt.Print(val)
	fmt.Print("\n")
}

func TestTips(t *testing.T) {
	// Nr= 1807722  (blue)
	blockDag0 := &BlockDAG{
		Hash:     common.HexToHash("0xa659fcd4ed3f3ad9cd43ab36eb29080a4655328fe16f045962afab1d66a5da09"),
		Height:   uint64(1807722),
		Slot:     uint64(1800022),
		CpHash:   common.HexToHash("0x7b5f02d2646f4d0758e78ae1d211b0bb3a0724952ff04d588c7a57bfc2ba7070"),
		CpHeight: uint64(1807706),

		OrderedAncestorsHashes: common.HashArray{
			common.HexToHash("0x0c449b3b974d1c33081eda15010740a4c79b194ee10846311970d150b7cd07de"), // 1807717 (blue)
			common.HexToHash("0x72fc4b6693d51f0a10913ca9ecfb6baa6774e54134b5aa23f49e45c5443841db"), // 1807717
			common.HexToHash("0xae0e83f6c7249349bd4c71f4aaccf16c57b3ba7a7f5a2a3b6618f2f86e155a99"), // 1807717
			common.HexToHash("0xc05186da87308720aef815a8a91a5195f104f89cb99e3fe5943f42c0856da7d1"), // 1807717
			common.HexToHash("0xe198206aa59e3f274851cefb00e648bf179d1b160b781f33c9d47752a1d86297"), // 1807717
		},
	}
	// Nr= 1807723  (red)
	blockDag1 := &BlockDAG{
		Hash:     common.HexToHash("0xd6a047d2fa3483d042741ef2cc0856d323f5d0432d48e48568d2de3c52310286"), //common.Hash{0x66, 0x66, 0x66, 0x66},
		Height:   uint64(1807722),
		Slot:     uint64(1800022),
		CpHash:   common.HexToHash("0x7b5f02d2646f4d0758e78ae1d211b0bb3a0724952ff04d588c7a57bfc2ba7070"),
		CpHeight: uint64(1807706),

		OrderedAncestorsHashes: common.HashArray{
			common.HexToHash("0x880b8b458fb7fbb4660292d9ecaf484536849972712f5a5628a207f318cae307"), //> 1807706 => 1807707
			common.HexToHash("0xadd020939a5f00207c95147c7d1f89fdcf84dfde08ac099bd90b5c571c26649e"), //> 1807706 => 1807708
			common.HexToHash("0xfa4c39399200727eec17c5eb32f859321daa8b1856dfe5e634aed0ffabfdc09d"), //> 1807706 => 1807709
			common.HexToHash("0xfb39360c4c5b41b1685315b6ea423cba96afa9ab88bdd75bd74ca19e4c669a8b"), //> 1807706 => 1807710
			common.HexToHash("0xfdf4a98d8f823eb6fa922340ffe5fd0a3ffa22590f8353e9d8ef873e745a86c7"), //> 1807706 => 1807711

			common.HexToHash("0x1c2dcc132ccfea02842d60c9c23de2efcfb614c4c8d95ff1f39ab5c58a7efda4"), //> 1807712 => 1807712 (blue)
			common.HexToHash("0x633fe4a676d864d89e75f61bffa31b632e99a595d8e71e55158591d752c09f57"), //> 1807712 => 1807713
			common.HexToHash("0x9fe01e79fd826b02f7308b23731003ecde3178fe9862ddd97bcd11819aa7171f"), //> 1807712 => 1807714
			common.HexToHash("0xd4d302747ccadaa7348062fc75b5a69331b05479fe844fae914faa3e992cdee7"), //> 1807712 => 1807715
			common.HexToHash("0xe39bfd9d8c1ec623fd00942484d2b3fbd54b0b70ad13035096e3b67771d71c6c"), //> 1807712 => 1807716

			common.HexToHash("0x0c449b3b974d1c33081eda15010740a4c79b194ee10846311970d150b7cd07de"), // 1807717 (blue)
			common.HexToHash("0x72fc4b6693d51f0a10913ca9ecfb6baa6774e54134b5aa23f49e45c5443841db"), // 1807717
			common.HexToHash("0xae0e83f6c7249349bd4c71f4aaccf16c57b3ba7a7f5a2a3b6618f2f86e155a99"), // 1807717
			common.HexToHash("0xc05186da87308720aef815a8a91a5195f104f89cb99e3fe5943f42c0856da7d1"), // 1807717
			common.HexToHash("0xe198206aa59e3f274851cefb00e648bf179d1b160b781f33c9d47752a1d86297"), // 1807717
		},
	}
	// Nr= 1807724  (red)
	blockDag2 := &BlockDAG{
		Hash:     common.HexToHash("0xddcc2c3e0530a6a3d8e60d35bd17c05b22a4826b78c44f55438cac6b47f4bfde"), //common.Hash{0x66, 0x66, 0x66, 0x66},
		Height:   uint64(1807722),
		Slot:     uint64(1800022),
		CpHash:   common.HexToHash("0x7b5f02d2646f4d0758e78ae1d211b0bb3a0724952ff04d588c7a57bfc2ba7070"),
		CpHeight: uint64(1807706),
		OrderedAncestorsHashes: common.HashArray{
			common.HexToHash("0x880b8b458fb7fbb4660292d9ecaf484536849972712f5a5628a207f318cae307"), //> 1807706 => 1807707
			common.HexToHash("0xadd020939a5f00207c95147c7d1f89fdcf84dfde08ac099bd90b5c571c26649e"), //> 1807706 => 1807708
			common.HexToHash("0xfa4c39399200727eec17c5eb32f859321daa8b1856dfe5e634aed0ffabfdc09d"), //> 1807706 => 1807709
			common.HexToHash("0xfb39360c4c5b41b1685315b6ea423cba96afa9ab88bdd75bd74ca19e4c669a8b"), //> 1807706 => 1807710
			common.HexToHash("0xfdf4a98d8f823eb6fa922340ffe5fd0a3ffa22590f8353e9d8ef873e745a86c7"), //> 1807706 => 1807711

			common.HexToHash("0x1c2dcc132ccfea02842d60c9c23de2efcfb614c4c8d95ff1f39ab5c58a7efda4"), //> 1807712 => 1807712 (blue)
			common.HexToHash("0x633fe4a676d864d89e75f61bffa31b632e99a595d8e71e55158591d752c09f57"), //> 1807712 => 1807713
			common.HexToHash("0x9fe01e79fd826b02f7308b23731003ecde3178fe9862ddd97bcd11819aa7171f"), //> 1807712 => 1807714
			common.HexToHash("0xd4d302747ccadaa7348062fc75b5a69331b05479fe844fae914faa3e992cdee7"), //> 1807712 => 1807715
			common.HexToHash("0xe39bfd9d8c1ec623fd00942484d2b3fbd54b0b70ad13035096e3b67771d71c6c"), //> 1807712 => 1807716

			common.HexToHash("0x0c449b3b974d1c33081eda15010740a4c79b194ee10846311970d150b7cd07de"), // 1807717 (blue)
			common.HexToHash("0x72fc4b6693d51f0a10913ca9ecfb6baa6774e54134b5aa23f49e45c5443841db"), // 1807717
			common.HexToHash("0xae0e83f6c7249349bd4c71f4aaccf16c57b3ba7a7f5a2a3b6618f2f86e155a99"), // 1807717
			common.HexToHash("0xc05186da87308720aef815a8a91a5195f104f89cb99e3fe5943f42c0856da7d1"), // 1807717
			common.HexToHash("0xe198206aa59e3f274851cefb00e648bf179d1b160b781f33c9d47752a1d86297"), // 1807717
		},
	}
	// Nr= 1807725  (red)
	blockDag3 := &BlockDAG{
		Hash:     common.HexToHash("0xe29272257bd82f1beb90ce048361e5063481830f16dacdb4a2781164ede114e1"), //common.Hash{0x66, 0x66, 0x66, 0x66},
		Height:   uint64(1807722),
		Slot:     uint64(1807722),
		CpHash:   common.HexToHash("0x7b5f02d2646f4d0758e78ae1d211b0bb3a0724952ff04d588c7a57bfc2ba7070"),
		CpHeight: uint64(1807706),
		OrderedAncestorsHashes: common.HashArray{
			common.HexToHash("0x880b8b458fb7fbb4660292d9ecaf484536849972712f5a5628a207f318cae307"), //> 1807706 => 1807707
			common.HexToHash("0xadd020939a5f00207c95147c7d1f89fdcf84dfde08ac099bd90b5c571c26649e"), //> 1807706 => 1807708
			common.HexToHash("0xfa4c39399200727eec17c5eb32f859321daa8b1856dfe5e634aed0ffabfdc09d"), //> 1807706 => 1807709
			common.HexToHash("0xfb39360c4c5b41b1685315b6ea423cba96afa9ab88bdd75bd74ca19e4c669a8b"), //> 1807706 => 1807710
			common.HexToHash("0xfdf4a98d8f823eb6fa922340ffe5fd0a3ffa22590f8353e9d8ef873e745a86c7"), //> 1807706 => 1807711

			common.HexToHash("0x1c2dcc132ccfea02842d60c9c23de2efcfb614c4c8d95ff1f39ab5c58a7efda4"), //> 1807712 => 1807712 (blue)
			common.HexToHash("0x633fe4a676d864d89e75f61bffa31b632e99a595d8e71e55158591d752c09f57"), //> 1807712 => 1807713
			common.HexToHash("0x9fe01e79fd826b02f7308b23731003ecde3178fe9862ddd97bcd11819aa7171f"), //> 1807712 => 1807714
			common.HexToHash("0xd4d302747ccadaa7348062fc75b5a69331b05479fe844fae914faa3e992cdee7"), //> 1807712 => 1807715
			common.HexToHash("0xe39bfd9d8c1ec623fd00942484d2b3fbd54b0b70ad13035096e3b67771d71c6c"), //> 1807712 => 1807716

			common.HexToHash("0x0c449b3b974d1c33081eda15010740a4c79b194ee10846311970d150b7cd07de"), // 1807717 (blue)
			common.HexToHash("0x72fc4b6693d51f0a10913ca9ecfb6baa6774e54134b5aa23f49e45c5443841db"), // 1807717
			common.HexToHash("0xae0e83f6c7249349bd4c71f4aaccf16c57b3ba7a7f5a2a3b6618f2f86e155a99"), // 1807717
			common.HexToHash("0xc05186da87308720aef815a8a91a5195f104f89cb99e3fe5943f42c0856da7d1"), // 1807717
			common.HexToHash("0xe198206aa59e3f274851cefb00e648bf179d1b160b781f33c9d47752a1d86297"), // 1807717
		},
	}
	// Nr= 1807726  (red)
	blockDag4 := &BlockDAG{
		Hash:     common.HexToHash("0xeb1a3ad9ad8136fc6290de8227dde86699f7e2d1944259783e4f1241c5eb1610"), //common.Hash{0x66, 0x66, 0x66, 0x66},
		Height:   uint64(1807722),
		Slot:     uint64(1807722),
		CpHash:   common.HexToHash("0x7b5f02d2646f4d0758e78ae1d211b0bb3a0724952ff04d588c7a57bfc2ba7070"),
		CpHeight: uint64(1807706),
		OrderedAncestorsHashes: common.HashArray{
			common.HexToHash("0x880b8b458fb7fbb4660292d9ecaf484536849972712f5a5628a207f318cae307"), //> 1807706 => 1807707
			common.HexToHash("0xadd020939a5f00207c95147c7d1f89fdcf84dfde08ac099bd90b5c571c26649e"), //> 1807706 => 1807708
			common.HexToHash("0xfa4c39399200727eec17c5eb32f859321daa8b1856dfe5e634aed0ffabfdc09d"), //> 1807706 => 1807709
			common.HexToHash("0xfb39360c4c5b41b1685315b6ea423cba96afa9ab88bdd75bd74ca19e4c669a8b"), //> 1807706 => 1807710
			common.HexToHash("0xfdf4a98d8f823eb6fa922340ffe5fd0a3ffa22590f8353e9d8ef873e745a86c7"), //> 1807706 => 1807711

			common.HexToHash("0x1c2dcc132ccfea02842d60c9c23de2efcfb614c4c8d95ff1f39ab5c58a7efda4"), //> 1807712 => 1807712 (blue)
			common.HexToHash("0x633fe4a676d864d89e75f61bffa31b632e99a595d8e71e55158591d752c09f57"), //> 1807712 => 1807713
			common.HexToHash("0x9fe01e79fd826b02f7308b23731003ecde3178fe9862ddd97bcd11819aa7171f"), //> 1807712 => 1807714
			common.HexToHash("0xd4d302747ccadaa7348062fc75b5a69331b05479fe844fae914faa3e992cdee7"), //> 1807712 => 1807715
			common.HexToHash("0xe39bfd9d8c1ec623fd00942484d2b3fbd54b0b70ad13035096e3b67771d71c6c"), //> 1807712 => 1807716

			common.HexToHash("0x0c449b3b974d1c33081eda15010740a4c79b194ee10846311970d150b7cd07de"), // 1807717 (blue)
			common.HexToHash("0x72fc4b6693d51f0a10913ca9ecfb6baa6774e54134b5aa23f49e45c5443841db"), // 1807717
			common.HexToHash("0xae0e83f6c7249349bd4c71f4aaccf16c57b3ba7a7f5a2a3b6618f2f86e155a99"), // 1807717
			common.HexToHash("0xc05186da87308720aef815a8a91a5195f104f89cb99e3fe5943f42c0856da7d1"), // 1807717
			common.HexToHash("0xe198206aa59e3f274851cefb00e648bf179d1b160b781f33c9d47752a1d86297"), // 1807717
		},
	}

	tips := Tips{}.Add(blockDag1).Add(blockDag2).Add(blockDag4).Add(blockDag0).Add(blockDag3)

	// sort by (1) Height desc (2) Hash asc
	orderedHashes := tips.getOrderedHashes()
	res := fmt.Sprintf("%v", orderedHashes)
	exp := fmt.Sprintf("%v", common.HashArray{blockDag0.Hash, blockDag1.Hash, blockDag2.Hash, blockDag3.Hash, blockDag4.Hash})
	if res != exp {
		t.Fatalf("Tips.getOrderedHashes failed, got %v != %v", res, exp)
	}

	ordChain := tips.GetOrderedAncestorsHashes()
	res = fmt.Sprintf("%v", ordChain.Uniq())

	expHashes := common.HashArray{
		common.HexToHash("0x0c449b3b974d1c33081eda15010740a4c79b194ee10846311970d150b7cd07de"),
		common.HexToHash("0x72fc4b6693d51f0a10913ca9ecfb6baa6774e54134b5aa23f49e45c5443841db"),
		common.HexToHash("0xae0e83f6c7249349bd4c71f4aaccf16c57b3ba7a7f5a2a3b6618f2f86e155a99"),
		common.HexToHash("0xc05186da87308720aef815a8a91a5195f104f89cb99e3fe5943f42c0856da7d1"),
		common.HexToHash("0xe198206aa59e3f274851cefb00e648bf179d1b160b781f33c9d47752a1d86297"),
		common.HexToHash("0xa659fcd4ed3f3ad9cd43ab36eb29080a4655328fe16f045962afab1d66a5da09"),
		common.HexToHash("0x880b8b458fb7fbb4660292d9ecaf484536849972712f5a5628a207f318cae307"),
		common.HexToHash("0xadd020939a5f00207c95147c7d1f89fdcf84dfde08ac099bd90b5c571c26649e"),
		common.HexToHash("0xfa4c39399200727eec17c5eb32f859321daa8b1856dfe5e634aed0ffabfdc09d"),
		common.HexToHash("0xfb39360c4c5b41b1685315b6ea423cba96afa9ab88bdd75bd74ca19e4c669a8b"),
		common.HexToHash("0xfdf4a98d8f823eb6fa922340ffe5fd0a3ffa22590f8353e9d8ef873e745a86c7"),
		common.HexToHash("0x1c2dcc132ccfea02842d60c9c23de2efcfb614c4c8d95ff1f39ab5c58a7efda4"),
		common.HexToHash("0x633fe4a676d864d89e75f61bffa31b632e99a595d8e71e55158591d752c09f57"),
		common.HexToHash("0x9fe01e79fd826b02f7308b23731003ecde3178fe9862ddd97bcd11819aa7171f"),
		common.HexToHash("0xd4d302747ccadaa7348062fc75b5a69331b05479fe844fae914faa3e992cdee7"),
		common.HexToHash("0xe39bfd9d8c1ec623fd00942484d2b3fbd54b0b70ad13035096e3b67771d71c6c"),
		common.HexToHash("0xd6a047d2fa3483d042741ef2cc0856d323f5d0432d48e48568d2de3c52310286"),
		common.HexToHash("0xddcc2c3e0530a6a3d8e60d35bd17c05b22a4826b78c44f55438cac6b47f4bfde"),
		common.HexToHash("0xe29272257bd82f1beb90ce048361e5063481830f16dacdb4a2781164ede114e1"),
		common.HexToHash("0xeb1a3ad9ad8136fc6290de8227dde86699f7e2d1944259783e4f1241c5eb1610"),
	}
	exp = fmt.Sprintf("%v", expHashes.Uniq())
	if res != exp {
		t.Fatalf("Tips.GetOrderedAncestorsHashes failed, got %v != %v", res, exp)
	}

	// tips.GetStableStateHash()
	stateHash := tips.GetStableStateHash()
	res = fmt.Sprintf("%v", stateHash)
	exp = fmt.Sprintf("%v", blockDag0.Hash)
	if res != exp {
		t.Fatalf("Tips.GetStableStateHash failed, got %v != %v", res, exp)
	}
}

// TestTipsSkippedSlots test case happen while extremal network loading
func TestTipsSkippedSlots(t *testing.T) {
	blockDag0 := &BlockDAG{
		Hash:     common.HexToHash("0x073e11c4fbb1c9e774cc54366af5a299f73aaf28fcbbe856fb18e158e2f5bc13"),
		Height:   uint64(3190),
		Slot:     uint64(1500),
		CpHash:   common.HexToHash("0x4264174f75eb9ed6cee56681adf1ab68adc2baf928ab5bbf2125e43f8504115a"),
		CpHeight: uint64(3170),
		OrderedAncestorsHashes: common.HashArray{
			common.HexToHash("0x5316ca87acd66d92f8766daf0e28508577b2048d67477d22430408ffd13663c8"), //
			common.HexToHash("0x8b69c9581d484978b094748fff612faaa8756fe2007c2e51288a57044d7dc2f2"), //
			common.HexToHash("0xa0c5206e647d2a3d268cf854f69491c3dbe2001d071d368e64d02cf14a5a904c"), //
			common.HexToHash("0xd9ae23892099540c543c9c503b49e8c5f67cb20982bb70993e7b4acb0b0fc92b"), //
			common.HexToHash("0xec43cf6fb475dde5e03d9b2fd5d0e68b4030a964a051508925b691fdec2cd6f4"), //
			common.HexToHash("0xa2ddb6f0a78ea1cf16eedc6610e32f6c23a49fc8e115ec3567797331258f78ef"), //
			common.HexToHash("0x08d31423142e35cc78c4936ff519e787362c1d6ea52115f074dae836e2be92bb"), //
			common.HexToHash("0x0a1e9e55bd566d51f7568510fdcc78a7d2360c476d4864d3467a7c8f8fdb918c"), //
			common.HexToHash("0x23906fa2ee492206571b5de2f82bee4d1ec683a91ccf16404ed54724fd62e5b2"), //
			common.HexToHash("0x30dda434434828308c6764663d652809f3610ae7dd8249d726c4075955f81766"), //
			common.HexToHash("0x4fc97dc76a92d730d1c75d4a393876e929e029af4e138bc7fbac6addd57d8114"), //
			common.HexToHash("0x877f52c39eaab61b59ff8a490814c7e5dba5a227e74ff5740f6fa0f303386d16"), //
			common.HexToHash("0x0535435a497014812e55b4c620eb204fa161db73883158b5223ef49e2795f966"), //
			common.HexToHash("0x598d8a6ec5ebe0ca390ffdde469c6f7931496b1c4c4df786c7af9df86e00cdd7"), //
			common.HexToHash("0x75b27a30e9b130796ce2e2fd798c23c73326eeb662eda7bb43067c5abfce5830"), //
			common.HexToHash("0x8b4b466ec19eb5712aad0415ae3bf803c9fedc764997b3ae9a23a372c9eb6a48"), //
			common.HexToHash("0xe3a24badcb0229a3e34c69cf2ce0257accfd470ac7cd8185a34d582856335c0a"), //
			common.HexToHash("0x79b751c964596724907788aabc3bf8fbc79f6d83d155f40f09975a1ae2a199cc"), //
			common.HexToHash("0x16aaae1457bd5ead8b573f3d033b37d56f61eaf7da264ddcfdaf963d1ed6b834"), //
		},
	}
	//
	blockDag1 := &BlockDAG{
		Hash:     common.HexToHash("0x08fd673456f56bf7f4370cc297c68e78dafe4ee0002b8b423c7f25b07a8b39ab"),
		Height:   uint64(3190),
		Slot:     uint64(3190),
		CpHash:   common.HexToHash("0x4264174f75eb9ed6cee56681adf1ab68adc2baf928ab5bbf2125e43f8504115a"),
		CpHeight: uint64(3170),
		OrderedAncestorsHashes: common.HashArray{
			common.HexToHash("0x5316ca87acd66d92f8766daf0e28508577b2048d67477d22430408ffd13663c8"), //
			common.HexToHash("0x8b69c9581d484978b094748fff612faaa8756fe2007c2e51288a57044d7dc2f2"), //
			common.HexToHash("0xa0c5206e647d2a3d268cf854f69491c3dbe2001d071d368e64d02cf14a5a904c"), //
			common.HexToHash("0xd9ae23892099540c543c9c503b49e8c5f67cb20982bb70993e7b4acb0b0fc92b"), //
			common.HexToHash("0xec43cf6fb475dde5e03d9b2fd5d0e68b4030a964a051508925b691fdec2cd6f4"), //
			common.HexToHash("0xa2ddb6f0a78ea1cf16eedc6610e32f6c23a49fc8e115ec3567797331258f78ef"), //
			common.HexToHash("0x08d31423142e35cc78c4936ff519e787362c1d6ea52115f074dae836e2be92bb"), //
			common.HexToHash("0x0a1e9e55bd566d51f7568510fdcc78a7d2360c476d4864d3467a7c8f8fdb918c"), //
			common.HexToHash("0x23906fa2ee492206571b5de2f82bee4d1ec683a91ccf16404ed54724fd62e5b2"), //
			common.HexToHash("0x30dda434434828308c6764663d652809f3610ae7dd8249d726c4075955f81766"), //
			common.HexToHash("0x4fc97dc76a92d730d1c75d4a393876e929e029af4e138bc7fbac6addd57d8114"), //
			common.HexToHash("0x877f52c39eaab61b59ff8a490814c7e5dba5a227e74ff5740f6fa0f303386d16"), //
			common.HexToHash("0x0535435a497014812e55b4c620eb204fa161db73883158b5223ef49e2795f966"), //
			common.HexToHash("0x598d8a6ec5ebe0ca390ffdde469c6f7931496b1c4c4df786c7af9df86e00cdd7"), //
			common.HexToHash("0x75b27a30e9b130796ce2e2fd798c23c73326eeb662eda7bb43067c5abfce5830"), //
			common.HexToHash("0x8b4b466ec19eb5712aad0415ae3bf803c9fedc764997b3ae9a23a372c9eb6a48"), //
			common.HexToHash("0xe3a24badcb0229a3e34c69cf2ce0257accfd470ac7cd8185a34d582856335c0a"), //
			common.HexToHash("0x79b751c964596724907788aabc3bf8fbc79f6d83d155f40f09975a1ae2a199cc"), //
			common.HexToHash("0x16aaae1457bd5ead8b573f3d033b37d56f61eaf7da264ddcfdaf963d1ed6b834"), //
		},
	}
	//
	blockDag2 := &BlockDAG{
		Hash:     common.HexToHash("0x2be07efb8721fc50e66ccd717300fdf2f89df92cb61027e7730c6f2bdf60b00d"),
		Height:   uint64(3190),
		Slot:     uint64(1500),
		CpHash:   common.HexToHash("0x4264174f75eb9ed6cee56681adf1ab68adc2baf928ab5bbf2125e43f8504115a"),
		CpHeight: uint64(3170),
		OrderedAncestorsHashes: common.HashArray{
			common.HexToHash("0x5316ca87acd66d92f8766daf0e28508577b2048d67477d22430408ffd13663c8"), //
			common.HexToHash("0x8b69c9581d484978b094748fff612faaa8756fe2007c2e51288a57044d7dc2f2"), //
			common.HexToHash("0xa0c5206e647d2a3d268cf854f69491c3dbe2001d071d368e64d02cf14a5a904c"), //
			common.HexToHash("0xd9ae23892099540c543c9c503b49e8c5f67cb20982bb70993e7b4acb0b0fc92b"), //
			common.HexToHash("0xec43cf6fb475dde5e03d9b2fd5d0e68b4030a964a051508925b691fdec2cd6f4"), //
			common.HexToHash("0xa2ddb6f0a78ea1cf16eedc6610e32f6c23a49fc8e115ec3567797331258f78ef"), //
			common.HexToHash("0x08d31423142e35cc78c4936ff519e787362c1d6ea52115f074dae836e2be92bb"), //
			common.HexToHash("0x0a1e9e55bd566d51f7568510fdcc78a7d2360c476d4864d3467a7c8f8fdb918c"), //
			common.HexToHash("0x23906fa2ee492206571b5de2f82bee4d1ec683a91ccf16404ed54724fd62e5b2"), //
			common.HexToHash("0x30dda434434828308c6764663d652809f3610ae7dd8249d726c4075955f81766"), //
			common.HexToHash("0x4fc97dc76a92d730d1c75d4a393876e929e029af4e138bc7fbac6addd57d8114"), //
			common.HexToHash("0x877f52c39eaab61b59ff8a490814c7e5dba5a227e74ff5740f6fa0f303386d16"), //
			common.HexToHash("0x0535435a497014812e55b4c620eb204fa161db73883158b5223ef49e2795f966"), //
			common.HexToHash("0x598d8a6ec5ebe0ca390ffdde469c6f7931496b1c4c4df786c7af9df86e00cdd7"), //
			common.HexToHash("0x75b27a30e9b130796ce2e2fd798c23c73326eeb662eda7bb43067c5abfce5830"), //
			common.HexToHash("0x8b4b466ec19eb5712aad0415ae3bf803c9fedc764997b3ae9a23a372c9eb6a48"), //
			common.HexToHash("0xe3a24badcb0229a3e34c69cf2ce0257accfd470ac7cd8185a34d582856335c0a"), //
			common.HexToHash("0x79b751c964596724907788aabc3bf8fbc79f6d83d155f40f09975a1ae2a199cc"), //
			common.HexToHash("0x16aaae1457bd5ead8b573f3d033b37d56f61eaf7da264ddcfdaf963d1ed6b834"), //
		},
	}
	//
	blockDag3 := &BlockDAG{
		Hash:     common.HexToHash("0x484a40592f60ebe1e26a8bcd6d9e33e706e1ac6dee0c608fb34cf7520083492e"),
		Height:   uint64(3184),
		Slot:     uint64(1500),
		CpHash:   common.HexToHash("0x4264174f75eb9ed6cee56681adf1ab68adc2baf928ab5bbf2125e43f8504115a"),
		CpHeight: uint64(3170),
		OrderedAncestorsHashes: common.HashArray{
			common.HexToHash("0x5316ca87acd66d92f8766daf0e28508577b2048d67477d22430408ffd13663c8"), //
			common.HexToHash("0x8b69c9581d484978b094748fff612faaa8756fe2007c2e51288a57044d7dc2f2"), //
			common.HexToHash("0xa0c5206e647d2a3d268cf854f69491c3dbe2001d071d368e64d02cf14a5a904c"), //
			common.HexToHash("0xd9ae23892099540c543c9c503b49e8c5f67cb20982bb70993e7b4acb0b0fc92b"), //
			common.HexToHash("0xec43cf6fb475dde5e03d9b2fd5d0e68b4030a964a051508925b691fdec2cd6f4"), //
			common.HexToHash("0xa2ddb6f0a78ea1cf16eedc6610e32f6c23a49fc8e115ec3567797331258f78ef"), //
			common.HexToHash("0x08d31423142e35cc78c4936ff519e787362c1d6ea52115f074dae836e2be92bb"), //
			common.HexToHash("0x0a1e9e55bd566d51f7568510fdcc78a7d2360c476d4864d3467a7c8f8fdb918c"), //
			common.HexToHash("0x16aaae1457bd5ead8b573f3d033b37d56f61eaf7da264ddcfdaf963d1ed6b834"), //
			common.HexToHash("0x23906fa2ee492206571b5de2f82bee4d1ec683a91ccf16404ed54724fd62e5b2"), //
			common.HexToHash("0x30dda434434828308c6764663d652809f3610ae7dd8249d726c4075955f81766"), //
			common.HexToHash("0x4fc97dc76a92d730d1c75d4a393876e929e029af4e138bc7fbac6addd57d8114"), //
			common.HexToHash("0x877f52c39eaab61b59ff8a490814c7e5dba5a227e74ff5740f6fa0f303386d16"), //
		},
	}
	//
	blockDag4 := &BlockDAG{
		Hash:     common.HexToHash("0x4c5993fd09b323b67b0e8ab2ac734792b242870403a1721c19907e9434013639"),
		Height:   uint64(3186),
		Slot:     uint64(1500),
		CpHash:   common.HexToHash("0x4264174f75eb9ed6cee56681adf1ab68adc2baf928ab5bbf2125e43f8504115a"),
		CpHeight: uint64(3170),
		OrderedAncestorsHashes: common.HashArray{
			common.HexToHash("0x5316ca87acd66d92f8766daf0e28508577b2048d67477d22430408ffd13663c8"), //
			common.HexToHash("0x8b69c9581d484978b094748fff612faaa8756fe2007c2e51288a57044d7dc2f2"), //
			common.HexToHash("0xa0c5206e647d2a3d268cf854f69491c3dbe2001d071d368e64d02cf14a5a904c"), //
			common.HexToHash("0xd9ae23892099540c543c9c503b49e8c5f67cb20982bb70993e7b4acb0b0fc92b"), //
			common.HexToHash("0xec43cf6fb475dde5e03d9b2fd5d0e68b4030a964a051508925b691fdec2cd6f4"), //
			common.HexToHash("0xa2ddb6f0a78ea1cf16eedc6610e32f6c23a49fc8e115ec3567797331258f78ef"), //
			common.HexToHash("0x08d31423142e35cc78c4936ff519e787362c1d6ea52115f074dae836e2be92bb"), //
			common.HexToHash("0x0a1e9e55bd566d51f7568510fdcc78a7d2360c476d4864d3467a7c8f8fdb918c"), //
			common.HexToHash("0x23906fa2ee492206571b5de2f82bee4d1ec683a91ccf16404ed54724fd62e5b2"), //
			common.HexToHash("0x30dda434434828308c6764663d652809f3610ae7dd8249d726c4075955f81766"), //
			common.HexToHash("0x4fc97dc76a92d730d1c75d4a393876e929e029af4e138bc7fbac6addd57d8114"), //
			common.HexToHash("0x877f52c39eaab61b59ff8a490814c7e5dba5a227e74ff5740f6fa0f303386d16"), //
			common.HexToHash("0x8b4b466ec19eb5712aad0415ae3bf803c9fedc764997b3ae9a23a372c9eb6a48"), //
			common.HexToHash("0x79b751c964596724907788aabc3bf8fbc79f6d83d155f40f09975a1ae2a199cc"), //
			common.HexToHash("0x16aaae1457bd5ead8b573f3d033b37d56f61eaf7da264ddcfdaf963d1ed6b834"), //
		},
	}
	//
	blockDag5 := &BlockDAG{
		Hash:     common.HexToHash("0x877afac335140d86bbbfc97b7e5f8a2156bc03862cc28e17afdb2187ec2df3ba"),
		Height:   uint64(3190),
		Slot:     uint64(1500),
		CpHash:   common.HexToHash("0x4264174f75eb9ed6cee56681adf1ab68adc2baf928ab5bbf2125e43f8504115a"),
		CpHeight: uint64(3170),
		OrderedAncestorsHashes: common.HashArray{
			common.HexToHash("0x5316ca87acd66d92f8766daf0e28508577b2048d67477d22430408ffd13663c8"), //
			common.HexToHash("0x8b69c9581d484978b094748fff612faaa8756fe2007c2e51288a57044d7dc2f2"), //
			common.HexToHash("0xa0c5206e647d2a3d268cf854f69491c3dbe2001d071d368e64d02cf14a5a904c"), //
			common.HexToHash("0xd9ae23892099540c543c9c503b49e8c5f67cb20982bb70993e7b4acb0b0fc92b"), //
			common.HexToHash("0xec43cf6fb475dde5e03d9b2fd5d0e68b4030a964a051508925b691fdec2cd6f4"), //
			common.HexToHash("0xa2ddb6f0a78ea1cf16eedc6610e32f6c23a49fc8e115ec3567797331258f78ef"), //
			common.HexToHash("0x08d31423142e35cc78c4936ff519e787362c1d6ea52115f074dae836e2be92bb"), //
			common.HexToHash("0x0a1e9e55bd566d51f7568510fdcc78a7d2360c476d4864d3467a7c8f8fdb918c"), //
			common.HexToHash("0x23906fa2ee492206571b5de2f82bee4d1ec683a91ccf16404ed54724fd62e5b2"), //
			common.HexToHash("0x30dda434434828308c6764663d652809f3610ae7dd8249d726c4075955f81766"), //
			common.HexToHash("0x4fc97dc76a92d730d1c75d4a393876e929e029af4e138bc7fbac6addd57d8114"), //
			common.HexToHash("0x877f52c39eaab61b59ff8a490814c7e5dba5a227e74ff5740f6fa0f303386d16"), //
			common.HexToHash("0x0535435a497014812e55b4c620eb204fa161db73883158b5223ef49e2795f966"), //
			common.HexToHash("0x598d8a6ec5ebe0ca390ffdde469c6f7931496b1c4c4df786c7af9df86e00cdd7"), //
			common.HexToHash("0x75b27a30e9b130796ce2e2fd798c23c73326eeb662eda7bb43067c5abfce5830"), //
			common.HexToHash("0x8b4b466ec19eb5712aad0415ae3bf803c9fedc764997b3ae9a23a372c9eb6a48"), //
			common.HexToHash("0xe3a24badcb0229a3e34c69cf2ce0257accfd470ac7cd8185a34d582856335c0a"), //
			common.HexToHash("0x79b751c964596724907788aabc3bf8fbc79f6d83d155f40f09975a1ae2a199cc"), //
			common.HexToHash("0x16aaae1457bd5ead8b573f3d033b37d56f61eaf7da264ddcfdaf963d1ed6b834"), //
		},
	}
	//
	blockDag6 := &BlockDAG{
		Hash:     common.HexToHash("0x8d90760ae9acce0ef73f74bb37c3913cbab7c439c3ea78f9e6b3e50e86d5abc7"),
		Height:   uint64(3190),
		Slot:     uint64(1500),
		CpHash:   common.HexToHash("0x4264174f75eb9ed6cee56681adf1ab68adc2baf928ab5bbf2125e43f8504115a"),
		CpHeight: uint64(3170),
		OrderedAncestorsHashes: common.HashArray{
			common.HexToHash("0x5316ca87acd66d92f8766daf0e28508577b2048d67477d22430408ffd13663c8"), //
			common.HexToHash("0x8b69c9581d484978b094748fff612faaa8756fe2007c2e51288a57044d7dc2f2"), //
			common.HexToHash("0xa0c5206e647d2a3d268cf854f69491c3dbe2001d071d368e64d02cf14a5a904c"), //
			common.HexToHash("0xd9ae23892099540c543c9c503b49e8c5f67cb20982bb70993e7b4acb0b0fc92b"), //
			common.HexToHash("0xec43cf6fb475dde5e03d9b2fd5d0e68b4030a964a051508925b691fdec2cd6f4"), //
			common.HexToHash("0xa2ddb6f0a78ea1cf16eedc6610e32f6c23a49fc8e115ec3567797331258f78ef"), //
			common.HexToHash("0x08d31423142e35cc78c4936ff519e787362c1d6ea52115f074dae836e2be92bb"), //
			common.HexToHash("0x0a1e9e55bd566d51f7568510fdcc78a7d2360c476d4864d3467a7c8f8fdb918c"), //
			common.HexToHash("0x23906fa2ee492206571b5de2f82bee4d1ec683a91ccf16404ed54724fd62e5b2"), //
			common.HexToHash("0x30dda434434828308c6764663d652809f3610ae7dd8249d726c4075955f81766"), //
			common.HexToHash("0x4fc97dc76a92d730d1c75d4a393876e929e029af4e138bc7fbac6addd57d8114"), //
			common.HexToHash("0x877f52c39eaab61b59ff8a490814c7e5dba5a227e74ff5740f6fa0f303386d16"), //
			common.HexToHash("0x0535435a497014812e55b4c620eb204fa161db73883158b5223ef49e2795f966"), //
			common.HexToHash("0x598d8a6ec5ebe0ca390ffdde469c6f7931496b1c4c4df786c7af9df86e00cdd7"), //
			common.HexToHash("0x75b27a30e9b130796ce2e2fd798c23c73326eeb662eda7bb43067c5abfce5830"), //
			common.HexToHash("0x8b4b466ec19eb5712aad0415ae3bf803c9fedc764997b3ae9a23a372c9eb6a48"), //
			common.HexToHash("0xe3a24badcb0229a3e34c69cf2ce0257accfd470ac7cd8185a34d582856335c0a"), //
			common.HexToHash("0x79b751c964596724907788aabc3bf8fbc79f6d83d155f40f09975a1ae2a199cc"), //
			common.HexToHash("0x16aaae1457bd5ead8b573f3d033b37d56f61eaf7da264ddcfdaf963d1ed6b834"), //
		},
	}
	//
	blockDag7 := &BlockDAG{
		Hash:     common.HexToHash("0xc9f3797c356525c8ceea13b205f65c30b082141b83520b05802df5ba2864c82f"),
		Height:   uint64(3190),
		Slot:     uint64(1500),
		CpHash:   common.HexToHash("0x4264174f75eb9ed6cee56681adf1ab68adc2baf928ab5bbf2125e43f8504115a"),
		CpHeight: uint64(3170),
		OrderedAncestorsHashes: common.HashArray{
			common.HexToHash("0x5316ca87acd66d92f8766daf0e28508577b2048d67477d22430408ffd13663c8"), //
			common.HexToHash("0x8b69c9581d484978b094748fff612faaa8756fe2007c2e51288a57044d7dc2f2"), //
			common.HexToHash("0xa0c5206e647d2a3d268cf854f69491c3dbe2001d071d368e64d02cf14a5a904c"), //
			common.HexToHash("0xd9ae23892099540c543c9c503b49e8c5f67cb20982bb70993e7b4acb0b0fc92b"), //
			common.HexToHash("0xec43cf6fb475dde5e03d9b2fd5d0e68b4030a964a051508925b691fdec2cd6f4"), //
			common.HexToHash("0xa2ddb6f0a78ea1cf16eedc6610e32f6c23a49fc8e115ec3567797331258f78ef"), //
			common.HexToHash("0x08d31423142e35cc78c4936ff519e787362c1d6ea52115f074dae836e2be92bb"), //
			common.HexToHash("0x0a1e9e55bd566d51f7568510fdcc78a7d2360c476d4864d3467a7c8f8fdb918c"), //
			common.HexToHash("0x23906fa2ee492206571b5de2f82bee4d1ec683a91ccf16404ed54724fd62e5b2"), //
			common.HexToHash("0x30dda434434828308c6764663d652809f3610ae7dd8249d726c4075955f81766"), //
			common.HexToHash("0x4fc97dc76a92d730d1c75d4a393876e929e029af4e138bc7fbac6addd57d8114"), //
			common.HexToHash("0x877f52c39eaab61b59ff8a490814c7e5dba5a227e74ff5740f6fa0f303386d16"), //
			common.HexToHash("0x0535435a497014812e55b4c620eb204fa161db73883158b5223ef49e2795f966"), //
			common.HexToHash("0x598d8a6ec5ebe0ca390ffdde469c6f7931496b1c4c4df786c7af9df86e00cdd7"), //
			common.HexToHash("0x75b27a30e9b130796ce2e2fd798c23c73326eeb662eda7bb43067c5abfce5830"), //
			common.HexToHash("0x8b4b466ec19eb5712aad0415ae3bf803c9fedc764997b3ae9a23a372c9eb6a48"), //
			common.HexToHash("0xe3a24badcb0229a3e34c69cf2ce0257accfd470ac7cd8185a34d582856335c0a"), //
			common.HexToHash("0x79b751c964596724907788aabc3bf8fbc79f6d83d155f40f09975a1ae2a199cc"), //
			common.HexToHash("0x16aaae1457bd5ead8b573f3d033b37d56f61eaf7da264ddcfdaf963d1ed6b834"), //
		},
	}

	tips := Tips{}.Add(blockDag1).Add(blockDag2).Add(blockDag4).Add(blockDag0).Add(blockDag3).Add(blockDag4).Add(blockDag5).Add(blockDag6).Add(blockDag7)

	// sort by (1) Height desc (2) Hash asc
	orderedHashes := tips.getOrderedHashes()
	res := fmt.Sprintf("%v", orderedHashes)
	exp := fmt.Sprintf("%v", common.HashArray{
		common.HexToHash("0x073e11c4fbb1c9e774cc54366af5a299f73aaf28fcbbe856fb18e158e2f5bc13"),
		common.HexToHash("0x08fd673456f56bf7f4370cc297c68e78dafe4ee0002b8b423c7f25b07a8b39ab"),
		common.HexToHash("0x2be07efb8721fc50e66ccd717300fdf2f89df92cb61027e7730c6f2bdf60b00d"),
		common.HexToHash("0x877afac335140d86bbbfc97b7e5f8a2156bc03862cc28e17afdb2187ec2df3ba"),
		common.HexToHash("0x8d90760ae9acce0ef73f74bb37c3913cbab7c439c3ea78f9e6b3e50e86d5abc7"),
		common.HexToHash("0xc9f3797c356525c8ceea13b205f65c30b082141b83520b05802df5ba2864c82f"),
		common.HexToHash("0x4c5993fd09b323b67b0e8ab2ac734792b242870403a1721c19907e9434013639"),
		common.HexToHash("0x484a40592f60ebe1e26a8bcd6d9e33e706e1ac6dee0c608fb34cf7520083492e"),
	})
	if res != exp {
		t.Fatalf("Tips.getOrderedHashes failed, got %v != %v", res, exp)
	}

	// tips.GetStableStateHash()
	stateHash := tips.GetStableStateHash()
	res = fmt.Sprintf("%v", stateHash)
	exp = fmt.Sprintf("%v", blockDag0.Hash)
	if res != exp {
		t.Fatalf("Tips.GetStableStateHash failed, got %v != %v", res, exp)
	}

	// tips.GetOrderedAncestorsHashes()
	ordChain := tips.GetOrderedAncestorsHashes()
	res = fmt.Sprintf("%v", ordChain.Uniq())

	expHashes := common.HashArray{
		common.HexToHash("0x5316ca87acd66d92f8766daf0e28508577b2048d67477d22430408ffd13663c8"),
		common.HexToHash("0x8b69c9581d484978b094748fff612faaa8756fe2007c2e51288a57044d7dc2f2"),
		common.HexToHash("0xa0c5206e647d2a3d268cf854f69491c3dbe2001d071d368e64d02cf14a5a904c"),
		common.HexToHash("0xd9ae23892099540c543c9c503b49e8c5f67cb20982bb70993e7b4acb0b0fc92b"),
		common.HexToHash("0xec43cf6fb475dde5e03d9b2fd5d0e68b4030a964a051508925b691fdec2cd6f4"),
		common.HexToHash("0xa2ddb6f0a78ea1cf16eedc6610e32f6c23a49fc8e115ec3567797331258f78ef"),
		common.HexToHash("0x08d31423142e35cc78c4936ff519e787362c1d6ea52115f074dae836e2be92bb"),
		common.HexToHash("0x0a1e9e55bd566d51f7568510fdcc78a7d2360c476d4864d3467a7c8f8fdb918c"),

		common.HexToHash("0x23906fa2ee492206571b5de2f82bee4d1ec683a91ccf16404ed54724fd62e5b2"),
		common.HexToHash("0x30dda434434828308c6764663d652809f3610ae7dd8249d726c4075955f81766"),
		common.HexToHash("0x4fc97dc76a92d730d1c75d4a393876e929e029af4e138bc7fbac6addd57d8114"),
		common.HexToHash("0x877f52c39eaab61b59ff8a490814c7e5dba5a227e74ff5740f6fa0f303386d16"),
		common.HexToHash("0x0535435a497014812e55b4c620eb204fa161db73883158b5223ef49e2795f966"),
		common.HexToHash("0x598d8a6ec5ebe0ca390ffdde469c6f7931496b1c4c4df786c7af9df86e00cdd7"),
		common.HexToHash("0x75b27a30e9b130796ce2e2fd798c23c73326eeb662eda7bb43067c5abfce5830"),

		common.HexToHash("0x8b4b466ec19eb5712aad0415ae3bf803c9fedc764997b3ae9a23a372c9eb6a48"),
		common.HexToHash("0xe3a24badcb0229a3e34c69cf2ce0257accfd470ac7cd8185a34d582856335c0a"),
		common.HexToHash("0x79b751c964596724907788aabc3bf8fbc79f6d83d155f40f09975a1ae2a199cc"),
		common.HexToHash("0x16aaae1457bd5ead8b573f3d033b37d56f61eaf7da264ddcfdaf963d1ed6b834"),
		common.HexToHash("0x073e11c4fbb1c9e774cc54366af5a299f73aaf28fcbbe856fb18e158e2f5bc13"),
		common.HexToHash("0x08fd673456f56bf7f4370cc297c68e78dafe4ee0002b8b423c7f25b07a8b39ab"),
		common.HexToHash("0x2be07efb8721fc50e66ccd717300fdf2f89df92cb61027e7730c6f2bdf60b00d"),
		common.HexToHash("0x877afac335140d86bbbfc97b7e5f8a2156bc03862cc28e17afdb2187ec2df3ba"),
		common.HexToHash("0x8d90760ae9acce0ef73f74bb37c3913cbab7c439c3ea78f9e6b3e50e86d5abc7"),

		common.HexToHash("0xc9f3797c356525c8ceea13b205f65c30b082141b83520b05802df5ba2864c82f"),
		common.HexToHash("0x4c5993fd09b323b67b0e8ab2ac734792b242870403a1721c19907e9434013639"),
		common.HexToHash("0x484a40592f60ebe1e26a8bcd6d9e33e706e1ac6dee0c608fb34cf7520083492e"),
	}

	exp = fmt.Sprintf("%v", expHashes.Uniq())
	if res != exp {
		t.Fatalf("Tips.GetOrderedAncestorsHashes failed, got %v != %v", res, exp)
	}
}

func TestHeaderMap(t *testing.T) {
	block0 := NewBlockWithHeader(&Header{
		Extra:    []byte("test Block"),
		GasLimit: 10,
		Time:     10,
		Height:   10,
	})
	block1 := NewBlockWithHeader(&Header{
		Extra:    []byte("test Block"),
		GasLimit: 20,
		Time:     20,
		Height:   20,
	})
	block2 := NewBlockWithHeader(&Header{
		Extra:    []byte("test Block"),
		GasLimit: 30,
		Time:     30,
		Height:   30,
	})
	hm := HeaderMap{}.Add(block0.Header()).Add(block1.Header()).Add(block2.Header())
	// AvgGasLimit()
	res := hm.AvgGasLimit()
	exp := uint64(20)
	if res != exp {
		t.Fatalf("AvgGasLimit failed, got %v != %v", res, exp)
	}
	// GetMaxTime()
	res = hm.GetMaxTime()
	exp = uint64(30)
	if res != exp {
		t.Fatalf("GetMaxTime failed, got %v != %v", res, exp)
	}
}
