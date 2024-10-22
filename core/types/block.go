// Copyright 2014 The go-ethereum Authors
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

// Package types contains data types related to Ethereum consensus.
package types

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"reflect"
	"sort"
	"sync/atomic"
	"time"

	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/common/hexutil"
	"gitlab.waterfall.network/waterfall/protocol/gwat/crypto"
	"gitlab.waterfall.network/waterfall/protocol/gwat/log"
	"gitlab.waterfall.network/waterfall/protocol/gwat/rlp"
)

var (
	EmptyRootHash = common.HexToHash("56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421")
)

// A BlockNonce is a 64-bit hash which proves (combined with the
// mix-hash) that a sufficient amount of computation has been carried
// out on a block.
type BlockNonce [8]byte
type SlotBlocksMap map[uint64]Blocks // slot: blocks
type SlotSpineMap map[uint64]*Block  // slot: block

type SlotHeadersMap map[uint64]Headers // slot: blocks

// Uint64 returns the integer value of a block nonce.
func (n BlockNonce) Uint64() uint64 {
	return binary.BigEndian.Uint64(n[:])
}

// MarshalText encodes n as a hex string with 0x prefix.
func (n BlockNonce) MarshalText() ([]byte, error) {
	return hexutil.Bytes(n[:]).MarshalText()
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (n *BlockNonce) UnmarshalText(input []byte) error {
	return hexutil.UnmarshalFixedText("BlockNonce", input, n[:])
}

//go:generate gencodec -type Header -field-override headerMarshaling -out gen_header_json.go

// Header represents a block header in the Ethereum blockchain.
type Header struct {
	//Base fields (set while create)
	ParentHashes common.HashArray `json:"parentHashes"     gencodec:"required"`
	Slot         uint64           `json:"slot"             gencodec:"required"`
	Era          uint64           `json:"era"              gencodec:"required"`
	Height       uint64           `json:"height"           gencodec:"required"`
	Coinbase     common.Address   `json:"miner"            gencodec:"required"`
	TxHash       common.Hash      `json:"transactionsRoot" gencodec:"required"`
	BodyHash     common.Hash      `json:"bodyRoot"         gencodec:"required"`
	GasLimit     uint64           `json:"gasLimit"         gencodec:"required"`
	Time         uint64           `json:"timestamp"        gencodec:"required"`
	Extra        []byte           `json:"extraData"        gencodec:"required"`
	//Last finalized block fields (set while create)
	CpHash        common.Hash `json:"cpHash"           gencodec:"required"`
	CpNumber      uint64      `json:"cpNumber"         gencodec:"required"`
	CpBaseFee     *big.Int    `json:"cpBaseFeePerGas"  gencodec:"required"`
	CpRoot        common.Hash `json:"cpStateRoot"      gencodec:"required"`
	CpReceiptHash common.Hash `json:"cpReceiptsRoot"   gencodec:"required"`
	CpGasUsed     uint64      `json:"cpGasUsed"        gencodec:"required"`
	CpBloom       Bloom       `json:"cpLogsBloom"      gencodec:"required"`
	//State fields (set while finalize)
	// BaseFee was added by EIP-1559 and is ignored in legacy headers.
	BaseFee     *big.Int    `json:"baseFeePerGas" rlp:"optional"`
	Number      *uint64     `json:"number"        rlp:"optional"`
	Root        common.Hash `json:"stateRoot"     rlp:"optional"`
	ReceiptHash common.Hash `json:"receiptsRoot"  rlp:"optional"`
	GasUsed     uint64      `json:"gasUsed"       rlp:"optional"`
	Bloom       Bloom       `json:"logsBloom"     rlp:"optional"`

	// Signature values
	V *big.Int `json:"v" rlp:"optional"`
	R *big.Int `json:"r" rlp:"optional"`
	S *big.Int `json:"s" rlp:"optional"`
}

// field type overrides for gencodec
type headerMarshaling struct {
	Height    *hexutil.Big
	CpHash    common.Hash
	CpNumber  hexutil.Uint64
	CpBaseFee *hexutil.Big
	CpGasUsed hexutil.Uint64
	GasLimit  hexutil.Uint64
	GasUsed   hexutil.Uint64
	Time      hexutil.Uint64
	Extra     hexutil.Bytes
	BaseFee   *hexutil.Big
	Hash      common.Hash `json:"hash"` // adds call to Hash() in MarshalJSON
}

// Hash returns the block hash of the header, which is simply the keccak256 hash of its
// RLP encoding.
func (h *Header) Hash() common.Hash {
	cpy := h.Copy()
	if cpy != nil {
		cpy.Number = nil
		cpy.BaseFee = nil
		cpy.GasUsed = 0
		cpy.Bloom = Bloom{}
		cpy.ReceiptHash = common.Hash{}
		cpy.Root = common.Hash{}
	}
	return rlpHash(cpy)
}

func (h *Header) UnsignedHash() common.Hash {
	cpy := h.Copy()
	if cpy != nil {
		cpy.Number = nil
		cpy.BaseFee = nil
		cpy.GasUsed = 0
		cpy.Bloom = Bloom{}
		cpy.ReceiptHash = common.Hash{}
		cpy.Root = common.Hash{}
		cpy.V = nil
		cpy.R = nil
		cpy.S = nil
	}
	return rlpHash(cpy)
}

// Copy creates copy of Header
func (h *Header) Copy() *Header {
	var cpy *Header = nil
	if h != nil {
		cpy = &Header{
			ParentHashes:  h.ParentHashes.Copy(),
			Slot:          h.Slot,
			Era:           h.Era,
			Height:        h.Height,
			CpHash:        common.BytesToHash(h.CpHash.Bytes()),
			CpNumber:      h.CpNumber,
			CpBloom:       BytesToBloom(h.CpBloom.Bytes()),
			CpRoot:        common.BytesToHash(h.CpRoot.Bytes()),
			CpReceiptHash: common.BytesToHash(h.CpReceiptHash.Bytes()),
			CpGasUsed:     h.CpGasUsed,
			Coinbase:      common.BytesToAddress(h.Coinbase.Bytes()),
			Root:          common.BytesToHash(h.Root.Bytes()),
			TxHash:        common.BytesToHash(h.TxHash.Bytes()),
			BodyHash:      common.BytesToHash(h.BodyHash.Bytes()),
			ReceiptHash:   common.BytesToHash(h.ReceiptHash.Bytes()),
			Bloom:         BytesToBloom(h.Bloom.Bytes()),
			GasLimit:      h.GasLimit,
			GasUsed:       h.GasUsed,
			Time:          h.Time,
			Extra:         make([]byte, len(h.Extra)),
		}
		if len(h.Extra) > 0 {
			copy(cpy.Extra, h.Extra)
		}
		if h.CpBaseFee != nil {
			cpy.CpBaseFee = new(big.Int).Set(h.CpBaseFee)
		}
		if h.BaseFee != nil {
			cpy.BaseFee = new(big.Int).Set(h.BaseFee)
		}
		if h.Number != nil {
			nr := *h.Number
			cpy.Number = &nr
		}
		if h.V != nil {
			cpy.V = new(big.Int).Set(h.V)
		}
		if h.R != nil {
			cpy.R = new(big.Int).Set(h.R)
		}
		if h.S != nil {
			cpy.S = new(big.Int).Set(h.S)
		}
	}
	return cpy
}

var headerSize = common.StorageSize(reflect.TypeOf(Header{}).Size())

// Nr returns finalized number if block finalized,
// otherwise 0
func (h *Header) Nr() uint64 {
	if h.Number != nil {
		return *h.Number
	}
	return 0
}

// Size returns the approximate memory used by all internal contents. It is used
// to approximate and limit the memory consumption of various caches.
func (h *Header) Size() common.StorageSize {
	return headerSize + common.StorageSize(len(h.Extra))
}

// SanityCheck checks a few basic things -- these checks are way beyond what
// any 'sane' production values should hold, and can mainly be used to prevent
// that the unbounded fields are stuffed with junk data to add processing
// overhead
func (h *Header) SanityCheck() error {
	if eLen := len(h.Extra); eLen > 0 {
		return fmt.Errorf("too large block extradata: size %d", eLen)
	}
	if h.BaseFee != nil {
		if bfLen := h.BaseFee.BitLen(); bfLen > 256 {
			return fmt.Errorf("too large base fee: bitlen %d", bfLen)
		}
	}
	return nil
}

// EmptyBody returns true if there is no additional 'body' to complete the header
// that is: no transactions and no uncles.
func (h *Header) EmptyBody() bool {
	return h.TxHash == EmptyRootHash
}

// EmptyReceipts returns true if there are no receipts for this header/block.
func (h *Header) EmptyReceipts() bool {
	return h.ReceiptHash == EmptyRootHash
}

func (h *Header) rawSignatureValues() (v, r, s *big.Int) {
	return h.V, h.R, h.S
}

func (h *Header) setSignature(sig []byte) {
	r, s, v := decodeSignature(sig)
	h.V, h.R, h.S = v, r, s
}

const uint32Length = 4

// Body is a simple (mutable, non-safe) data container for storing and moving
// a block's data contents (transactions and uncles) together.
type Body struct {
	Transactions Transactions
}

func (b *Body) ToBytes() ([]byte, error) {
	txsBytes, err := rlp.EncodeToBytes(b.Transactions)
	if err != nil {
		log.Error("can`t encode transactions from body", "error", err)
		return nil, err
	}

	txsLen := len(txsBytes)

	res := make([]byte, txsLen+uint32Length)
	copy(res[:uint32Length], common.Uint64ToBytes(uint64(txsLen)))
	copy(res[uint32Length:], txsBytes)

	return res, err
}

func (b *Body) CalculateHash() common.Hash {
	bodyBytes, err := b.ToBytes()
	if err != nil {
		return EmptyRootHash
	}

	return crypto.Keccak256Hash(bodyBytes)
}

// Block represents an entire block in the Ethereum blockchain.
type Block struct {
	header       *Header
	transactions Transactions

	// caches
	hash atomic.Value
	size atomic.Value

	// These fields are used by package eth to track
	// inter-peer block relay.
	ReceivedAt   time.Time
	ReceivedFrom interface{}
}

// "external" block encoding. used for eth protocol, etc.
type extblock struct {
	Header *Header        `json:"header"           gencodec:"required"`
	Txs    []*Transaction `json:"transactions"     gencodec:"required"`
}

// NewBlock creates a new block. The input data is copied,
// changes to header and to the field values will not affect the
// block.
//
// The values of TxHash, ReceiptHash and Bloom in header
// are ignored and set to values derived from the given txs, uncles
// and receipts.
func NewBlock(header *Header, txs []*Transaction, receipts []*Receipt, hasher TrieHasher) *Block {
	b := &Block{header: CopyHeader(header)}

	if len(txs) == 0 {
		b.header.TxHash = EmptyRootHash
	} else {
		b.header.TxHash = DeriveSha(Transactions(txs), hasher)
		b.transactions = make(Transactions, len(txs))
		copy(b.transactions, txs)
	}

	if len(receipts) == 0 {
		b.header.ReceiptHash = EmptyRootHash
	} else {
		b.header.ReceiptHash = DeriveSha(Receipts(receipts), hasher)
		b.header.Bloom = CreateBloom(receipts)
	}
	// calc BodyHash
	b.header.BodyHash = b.Body().CalculateHash()
	return b
}

// NewStatelessBlock creates a new block. The input data is copied,
// changes to header and to the field values will not affect the
// block.
//
// The values of TxHash and Bloom in header
// are ignored and set to values derived from the given txs and uncles.
func NewStatelessBlock(header *Header, txs []*Transaction, hasher TrieHasher) *Block {
	return NewBlock(header, txs, nil, hasher)
}

// NewBlockWithHeader creates a block with the given header data. The
// header data is copied, changes to header and to the field values
// will not affect the block.
func NewBlockWithHeader(header *Header) *Block {
	return &Block{header: CopyHeader(header)}
}

// CopyHeader creates a deep copy of a block header to prevent side effects from
// modifying a header variable.
func CopyHeader(h *Header) *Header {
	if h == nil {
		return nil
	}
	cpy := *h
	if h.Number != nil {
		nr := h.Nr()
		if cpy.Number = new(uint64); h.Number != nil {
			cpy.Number = &nr
		}
	} else {
		cpy.Number = nil
	}
	if h.BaseFee != nil {
		cpy.BaseFee = new(big.Int).Set(h.BaseFee)
	}
	if len(h.Extra) > 0 {
		cpy.Extra = make([]byte, len(h.Extra))
		copy(cpy.Extra, h.Extra)
	}
	return &cpy
}

// DecodeRLP decodes the Ethereum
func (b *Block) DecodeRLP(s *rlp.Stream) error {
	var eb extblock
	_, size, _ := s.Kind()
	if err := s.Decode(&eb); err != nil {
		return err
	}
	b.header, b.transactions = eb.Header, eb.Txs
	b.size.Store(common.StorageSize(rlp.ListSize(size)))
	return nil
}

// EncodeRLP serializes b into the Ethereum RLP block format.
func (b *Block) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, extblock{
		Header: b.header,
		Txs:    b.transactions,
	})
}

func (b *Block) Transactions() Transactions { return b.transactions }

func (b *Block) Transaction(hash common.Hash) *Transaction {
	for _, transaction := range b.transactions {
		if transaction.Hash() == hash {
			return transaction
		}
	}
	return nil
}

func (b *Block) GasLimit() uint64               { return b.header.GasLimit }
func (b *Block) GasUsed() uint64                { return b.header.GasUsed }
func (b *Block) Time() uint64                   { return b.header.Time }
func (b *Block) Bloom() Bloom                   { return b.header.Bloom }
func (b *Block) Coinbase() common.Address       { return b.header.Coinbase }
func (b *Block) Root() common.Hash              { return b.header.Root }
func (b *Block) ParentHashes() common.HashArray { return b.header.ParentHashes }
func (b *Block) Slot() uint64                   { return b.header.Slot }
func (b *Block) Era() uint64                    { return b.header.Era }
func (b *Block) Height() uint64                 { return b.header.Height }
func (b *Block) TxHash() common.Hash            { return b.header.TxHash }
func (b *Block) BodyHash() common.Hash          { return b.header.BodyHash }
func (b *Block) ReceiptHash() common.Hash       { return b.header.ReceiptHash }
func (b *Block) Extra() []byte                  { return common.CopyBytes(b.header.Extra) }
func (b *Block) Number() *uint64                { return b.header.Number }
func (b *Block) Nr() uint64                     { return b.header.Nr() }
func (b *Block) SetNumber(finNr *uint64)        { b.header.Number = finNr }
func (b *Block) CpHash() common.Hash            { return b.header.CpHash }
func (b *Block) CpNumber() uint64               { return b.header.CpNumber }
func (b *Block) CpBaseFee() *big.Int            { return b.header.CpBaseFee }
func (b *Block) CpBloom() Bloom                 { return b.header.CpBloom }
func (b *Block) CpRoot() common.Hash            { return b.header.CpRoot }
func (b *Block) CpReceiptHash() common.Hash     { return b.header.CpReceiptHash }
func (b *Block) CpGasUsed() uint64              { return b.header.CpGasUsed }

func (b *Block) BaseFee() *big.Int {
	if b.header.BaseFee == nil {
		return nil
	}
	return new(big.Int).Set(b.header.BaseFee)
}

func (b *Block) Header() *Header { return CopyHeader(b.header) }

func (b *Block) SetHeader(header *Header) {
	b.header = header
}

func (b *Block) SetReceipt(receipts []*Receipt, hasher TrieHasher) {
	if len(receipts) == 0 {
		b.header.ReceiptHash = EmptyRootHash
	} else {
		b.header.ReceiptHash = DeriveSha(Receipts(receipts), hasher)
		b.header.Bloom = CreateBloom(receipts)
	}
}

// Body returns the non-header content of the block.
func (b *Block) Body() *Body { return &Body{b.transactions} }

// Size returns the true RLP encoded storage size of the block, either by encoding
// and returning it, or returning a previsouly cached value.
func (b *Block) Size() common.StorageSize {
	if size := b.size.Load(); size != nil {
		return size.(common.StorageSize)
	}
	c := writeCounter(0)
	rlp.Encode(&c, b)
	b.size.Store(common.StorageSize(c))
	return common.StorageSize(c)
}

// SanityCheck can be used to prevent that unbounded fields are
// stuffed with junk data to add processing overhead
func (b *Block) SanityCheck() error {
	return b.header.SanityCheck()
}

type writeCounter common.StorageSize

func (c *writeCounter) Write(b []byte) (int, error) {
	*c += writeCounter(len(b))
	return len(b), nil
}

// WithSeal returns a new block with the data from b but the header replaced with
// the sealed one.
func (b *Block) WithSeal(header *Header) *Block {
	cpy := *header

	return &Block{
		header:       &cpy,
		transactions: b.transactions,
	}
}

// WithBody returns a new block with the given transaction and uncle contents.
func (b *Block) WithBody(transactions []*Transaction) *Block {
	block := &Block{
		header:       CopyHeader(b.header),
		transactions: make([]*Transaction, len(transactions)),
	}
	copy(block.transactions, transactions)
	return block
}

// Hash returns the keccak256 hash of b's header.
// The hash is computed on the first call and cached thereafter.
func (b *Block) Hash() common.Hash {
	if hash := b.hash.Load(); hash != nil {
		return hash.(common.Hash)
	}
	v := b.header.Hash()
	b.hash.Store(v)
	return v
}

type Blocks []*Block

func (bs *Blocks) GetMaxHeight() uint64 {
	maxHeight := uint64(0)
	for _, block := range *bs {
		if height := block.Height(); height > maxHeight {
			maxHeight = height
		}
	}
	return maxHeight
}

func (bs *Blocks) GetMaxParentHashesLen() int {
	maxLen := 0
	for _, block := range *bs {
		if phLen := len(block.ParentHashes()); phLen > maxLen {
			maxLen = phLen
		}
	}
	return maxLen
}

func (bs *Blocks) GetMaxHeightBlocks() Blocks {
	if len(*bs) == 0 {
		return nil
	}

	res := make(Blocks, 0, 1)
	maxHeight := bs.GetMaxHeight()
	for _, block := range *bs {
		if block.Height() == maxHeight {
			res = append(res, block)
		}
	}
	return res
}

func (bs *Blocks) GetMaxParentHashesLenBlocks() Blocks {
	if len(*bs) == 0 {
		return nil
	}

	res := make(Blocks, 0, 1)
	maxParentHashesLen := bs.GetMaxParentHashesLen()
	for _, block := range *bs {
		if len(block.ParentHashes()) == maxParentHashesLen {
			res = append(res, block)
		}
	}
	return res
}

func (bs *Blocks) GroupBySlot() (SlotBlocksMap, error) {
	if len(*bs) == 0 {
		return SlotBlocksMap{}, nil
	}

	res := make(SlotBlocksMap)
	for _, block := range *bs {
		if block == nil {
			return nil, errors.New("nil block found")
		}

		if block.header == nil {
			log.Error("nil header found", "block", block)
			return nil, errors.New("nil header found")
		}

		blockSlot := block.Slot()
		if _, exists := res[blockSlot]; !exists {
			res[blockSlot] = make(Blocks, 0, 1)
		}
		res[blockSlot] = append(res[blockSlot], block)
	}
	return res, nil
}

func (bs *Blocks) GetBlockByHash(hash common.Hash) *Block {
	for _, block := range *bs {
		if block.Hash() == hash {
			return block
		}
	}
	return nil
}

func (bs *Blocks) Deduplicate(rmNil bool) Blocks {
	if bs == nil {
		return Blocks{}
	}
	cpy := make(Blocks, 0, len(*bs))
	bmap := map[common.Hash]bool{}
	for _, block := range *bs {
		if block == nil {
			if rmNil {
				continue
			}
			if _, ok := bmap[common.Hash{}]; !ok {
				cpy = append(cpy, block)
				bmap[common.Hash{}] = true
			}
			continue
		}
		if _, ok := bmap[block.Hash()]; !ok {
			cpy = append(cpy, block)
			bmap[block.Hash()] = true
		}
	}
	return cpy
}

func (bs *Blocks) GetHashes() *common.HashArray {
	hashes := make(common.HashArray, 0, len(*bs))
	for _, block := range *bs {
		hashes = append(hashes, block.Hash())
	}
	return &hashes
}

func (shm *SlotBlocksMap) GetMinSlot() uint64 {
	minSlot := uint64(math.MaxUint64)
	for k := range *shm {
		if k < minSlot {
			minSlot = k
		}
	}
	return minSlot
}

func (shm *SlotBlocksMap) GetMaxSlot() uint64 {
	minSlot := uint64(0)
	for k := range *shm {
		if k > minSlot {
			minSlot = k
		}
	}
	return minSlot
}

func (shm *SlotSpineMap) GetMaxSlot() uint64 {
	maxSlot := uint64(0)
	for slot := range *shm {
		if slot > maxSlot {
			maxSlot = slot
		}
	}
	return maxSlot
}

func (shm *SlotSpineMap) GetMinSlot() uint64 {
	minClot := uint64(math.MaxUint64)
	for slot := range *shm {
		if slot < minClot {
			minClot = slot
		}
	}
	return minClot
}

func (shm *SlotSpineMap) GetOrderedHashes() *common.HashArray {
	if len(*shm) == 0 {
		return &common.HashArray{}
	}
	hashes := make(common.HashArray, 0, len(*shm))
	//sort by slots
	slots := common.SorterAscU64{}
	for sl := range *shm {
		slots = append(slots, sl)
	}
	sort.Sort(slots)
	for _, slot := range slots {
		hashes = append(hashes, (*shm)[slot].Hash())
	}
	return &hashes
}

type Headers []*Header

func (hs *Headers) GetMaxHeight() uint64 {
	maxHeight := uint64(0)
	for _, header := range *hs {
		if height := header.Height; height > maxHeight {
			maxHeight = height
		}
	}
	return maxHeight
}

func (hs *Headers) GetMaxParentHashesLen() int {
	maxLen := 0
	for _, header := range *hs {
		if phLen := len(header.ParentHashes); phLen > maxLen {
			maxLen = phLen
		}
	}
	return maxLen
}

func (hs *Headers) GetMaxHeightBlocks() Headers {
	if len(*hs) == 0 {
		return nil
	}

	res := make(Headers, 0, 1)
	maxHeight := hs.GetMaxHeight()
	for _, header := range *hs {
		if header.Height == maxHeight {
			res = append(res, header)
		}
	}
	return res
}

func (hs *Headers) GetMaxParentHashesLenBlocks() Headers {
	if len(*hs) == 0 {
		return nil
	}

	res := make(Headers, 0, 1)
	maxParentHashesLen := hs.GetMaxParentHashesLen()
	for _, header := range *hs {
		if len(header.ParentHashes) == maxParentHashesLen {
			res = append(res, header)
		}
	}
	return res
}

func (hs *Headers) GroupBySlot() (SlotHeadersMap, error) {
	if len(*hs) == 0 {
		return SlotHeadersMap{}, nil
	}

	res := make(SlotHeadersMap)
	for _, header := range *hs {
		if header == nil {
			return nil, errors.New("nil header found")
		}

		if header == nil {
			log.Error("nil header found", "header", header)
			return nil, errors.New("nil header found")
		}

		blockSlot := header.Slot
		if _, exists := res[blockSlot]; !exists {
			res[blockSlot] = make(Headers, 0, 1)
		}
		res[blockSlot] = append(res[blockSlot], header)
	}
	return res, nil
}

func (hs *Headers) GetBlockByHash(hash common.Hash) *Header {
	for _, header := range *hs {
		if header.Hash() == hash {
			return header
		}
	}
	return nil
}

func (hs *Headers) GetHashes() *common.HashArray {
	hashes := make(common.HashArray, 0, len(*hs))
	for _, header := range *hs {
		hashes = append(hashes, header.Hash())
	}
	return &hashes
}
