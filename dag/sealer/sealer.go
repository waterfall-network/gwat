package sealer

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru"
	"gitlab.waterfall.network/waterfall/protocol/gwat/accounts"
	"gitlab.waterfall.network/waterfall/protocol/gwat/common"
	"gitlab.waterfall.network/waterfall/protocol/gwat/common/hexutil"
	"gitlab.waterfall.network/waterfall/protocol/gwat/consensus"
	"gitlab.waterfall.network/waterfall/protocol/gwat/consensus/misc"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/state"
	"gitlab.waterfall.network/waterfall/protocol/gwat/core/types"
	"gitlab.waterfall.network/waterfall/protocol/gwat/crypto"
	"gitlab.waterfall.network/waterfall/protocol/gwat/ethdb"
	"gitlab.waterfall.network/waterfall/protocol/gwat/log"
	"gitlab.waterfall.network/waterfall/protocol/gwat/params"
	"gitlab.waterfall.network/waterfall/protocol/gwat/rlp"
	"gitlab.waterfall.network/waterfall/protocol/gwat/rpc"
	"gitlab.waterfall.network/waterfall/protocol/gwat/trie"
	"golang.org/x/crypto/sha3"
)

const (
	checkpointInterval = 1024 // Number of blocks after which to save the vote snapshot to the database
	inmemorySnapshots  = 128  // Number of recent vote snapshots to keep in memory
	inmemorySignatures = 4096 // Number of recent block signatures to keep in memory

	//wiggleTime = 500 * time.Millisecond // Random delay (per signer) to allow concurrent signers
)

// Sealer proof-of-authority protocol constants.
var (
	extraVanity = 32                     // Fixed number of extra-data prefix bytes reserved for signer vanity
	extraSeal   = crypto.SignatureLength // Fixed number of extra-data suffix bytes reserved for signer seal

	nonceAuthVote = hexutil.MustDecode("0xffffffffffffffff") // Magic nonce number to vote on adding a new signer
	nonceDropVote = hexutil.MustDecode("0x0000000000000000") // Magic nonce number to vote on removing a signer.
)

// Various error messages to mark blocks invalid. These should be private to
// prevent engine specific errors from being referenced in the remainder of the
// codebase, inherently breaking if the engine is swapped out. Please put common
// error types into the consensus package.
var (
	// errUnknownBlock is returned when the list of signers is requested for a block
	// that is not part of the local blockchain.
	errUnknownBlock = errors.New("unknown block")

	// errInvalidCheckpointBeneficiary is returned if a checkpoint/epoch transition
	// block has a beneficiary set to non-zeroes.
	errInvalidCheckpointBeneficiary = errors.New("beneficiary in checkpoint block non-zero")

	// errInvalidVote is returned if a nonce value is something else that the two
	// allowed constants of 0x00..0 or 0xff..f.
	errInvalidVote = errors.New("vote nonce not 0x00..0 or 0xff..f")

	// errInvalidMixDigest is returned if a block's mix digest is non-zero.
	errInvalidMixDigest = errors.New("non-zero mix digest")

	// errUnauthorizedSigner is returned if a header is signed by a non-authorized entity.
	errUnauthorizedSigner = errors.New("unauthorized signer")

	// errRecentlySigned is returned if a header is signed by an authorized entity
	// that already signed a header recently, thus is temporarily not allowed to.
	errRecentlySigned = errors.New("recently signed")
)

// SignerFn hashes and signs the data to be signed by a backing account.
type SignerFn func(signer accounts.Account, mimeType string, message []byte) ([]byte, error)

// ecrecover extracts the Ethereum account address from a signed header.
func ecrecover(header *types.Header, sigcache *lru.ARCCache) (common.Address, error) {
	return header.Coinbase, nil
	//
	//// If the signature's already cached, return that
	//hash := header.Hash()
	//if address, known := sigcache.Get(hash); known {
	//	return address.(common.Address), nil
	//}
	////todo
	//// Retrieve the signature from the header extra-data
	//if len(header.Extra) < extraSeal {
	//	return common.Address{}, errMissingSignature
	//}
	//signature := header.Extra[len(header.Extra)-extraSeal:]
	//
	//// Recover the public key and the Ethereum address
	//pubkey, err := crypto.Ecrecover(SealHash(header).Bytes(), signature)
	//if err != nil {
	//	return common.Address{}, err
	//}
	//var signer common.Address
	//copy(signer[:], crypto.Keccak256(pubkey[1:])[12:])
	//
	//sigcache.Add(hash, signer)
	//return signer, nil
}

// Sealer is the proof-of-authority consensus engine proposed to support the
// Ethereum testnet following the Ropsten attacks.
type Sealer struct {
	//config *params.CliqueConfig // Consensus engine configuration parameters
	db ethdb.Database // Database to store and retrieve snapshot checkpoints

	recents    *lru.ARCCache // Snapshots for recent block to speed up reorgs
	signatures *lru.ARCCache // Signatures of recent blocks to speed up block creation

	proposals map[common.Address]bool // Current list of proposals we are pushing

	signer common.Address // Ethereum address of the signing key
	signFn SignerFn       // Signer function to authorize hashes with
	lock   sync.RWMutex   // Protects the signer fields

	// The fields below are for testing only
	fakeDiff bool // Skip difficulty verifications
}

// New creates a Sealer proof-of-authority consensus engine with the initial
// signers set to the ones provided by the user.
// func New(config *params.CliqueConfig, db ethdb.Database) *Sealer {
func New(db ethdb.Database) *Sealer {
	// Allocate the snapshot caches and create the engine
	recents, _ := lru.NewARC(inmemorySnapshots)
	signatures, _ := lru.NewARC(inmemorySignatures)

	return &Sealer{
		db:         db,
		recents:    recents,
		signatures: signatures,
		proposals:  make(map[common.Address]bool),
	}
}

// Author implements consensus.Engine, returning the Ethereum address recovered
// from the signature in the header's extra-data section.
func (c *Sealer) Author(header *types.Header) (common.Address, error) {
	return ecrecover(header, c.signatures)
}

// VerifyHeader checks whether a header conforms to the consensus rules.
func (c *Sealer) VerifyHeader(chain consensus.ChainHeaderReader, header *types.Header, seal bool) error {
	//todo
	//return c.verifyHeader(chain, header, nil)
	return nil
}

// VerifyHeaders is similar to VerifyHeader, but verifies a batch of headers. The
// method returns a quit channel to abort the operations and a results channel to
// retrieve the async verifications (the order is that of the input slice).
func (c *Sealer) VerifyHeaders(chain consensus.ChainHeaderReader, headers []*types.Header, seals []bool) (chan<- struct{}, <-chan error) {
	abort := make(chan struct{})
	results := make(chan error, len(headers))

	go func() {
		//todo
		//for i, header := range headers {
		//	err := c.verifyHeader(chain, header, headers[:i])
		//
		//	select {
		//	case <-abort:
		//		return
		//	case results <- err:
		//	}
		//}
		for range headers {
			var err error = nil
			select {
			case <-abort:
				return
			case results <- err:
			}
		}
	}()
	return abort, results
}

// verifyHeader checks whether a header conforms to the consensus rules.The
// caller may optionally pass in a batch of parents (ascending order) to avoid
// looking those up from the database. This is useful for concurrently verifying
// a batch of new headers.
func (c *Sealer) verifyHeader(chain consensus.ChainHeaderReader, header *types.Header, parents []*types.Header) error {
	//if header.Number == nil {
	//	return errUnknownBlock
	//}
	//number := header.Height

	// Don't waste time checking blocks from the future
	if header.Time > uint64(time.Now().Unix()) {
		return consensus.ErrFutureBlock
	}
	// Checkpoint blocks need to enforce zero beneficiary

	//todo
	//checkpoint := (number % c.config.Epoch) == 0
	checkpoint := false

	if checkpoint && header.Coinbase != (common.Address{}) {
		return errInvalidCheckpointBeneficiary
	}
	//// Nonces must be 0x00..0 or 0xff..f, zeroes enforced on checkpoints
	//if !bytes.Equal(header.Nonce[:], nonceAuthVote) && !bytes.Equal(header.Nonce[:], nonceDropVote) {
	//	return errInvalidVote
	//}
	//if checkpoint && !bytes.Equal(header.Nonce[:], nonceDropVote) {
	//	return errInvalidCheckpointVote
	//}

	//// Check that the extra-data contains both the vanity and signature
	//if len(header.Extra) < extraVanity {
	//	return errMissingVanity
	//}
	//if len(header.Extra) < extraVanity+extraSeal {
	//	return errMissingSignature
	//}
	//
	//// Ensure that the extra-data contains a signer list on checkpoint, but none otherwise
	//signersBytes := len(header.Extra) - extraVanity - extraSeal
	//if !checkpoint && signersBytes != 0 {
	//	return errExtraSigners
	//}
	//if checkpoint && signersBytes%common.AddressLength != 0 {
	//	return errInvalidCheckpointSigners
	//}

	// Verify that the gas limit is <= 2^63-1
	cap := uint64(0x7fffffffffffffff)
	if header.GasLimit > cap {
		return fmt.Errorf("invalid gasLimit: have %v, max %v", header.GasLimit, cap)
	}
	// If all checks passed, validate any special fields for hard forks
	if err := misc.VerifyForkHashes(chain.Config(), header, false); err != nil {
		return err
	}
	// All basic checks passed, verify cascading fields
	return c.verifyCascadingFields(chain, header, parents)
}

// verifyCascadingFields verifies all the header fields that are not standalone,
// rather depend on a batch of previous headers. The caller may optionally pass
// in a batch of parents (ascending order) to avoid looking those up from the
// database. This is useful for concurrently verifying a batch of new headers.
func (c *Sealer) verifyCascadingFields(chain consensus.ChainHeaderReader, header *types.Header, parents []*types.Header) error {
	// The genesis block is the always valid dead-end
	//todo
	//number := header.Number.Uint64()
	number := uint64(0)
	if number == 0 {
		return nil
	}
	// Ensure that the block's timestamp isn't too close to its parent
	var parent *types.Header
	if len(parents) > 0 {
		parent = parents[len(parents)-1]
	} else {
		parent = chain.GetHeader(header.ParentHashes[0])
	}
	//if parent == nil || parent.Number.Uint64() != number-1 || parent.Hash() != header.ParentHashes[0] {
	//	return consensus.ErrUnknownAncestor
	//}

	//if parent.Time+c.config.Period > header.Time {
	//	return errInvalidTimestamp
	//}

	// Verify that the gasUsed is <= gasLimit
	if header.GasUsed > header.GasLimit {
		return fmt.Errorf("invalid gasUsed: have %d, gasLimit %d", header.GasUsed, header.GasLimit)
	}
	//// always IsLondon = true
	//if !chain.Config().IsLondon(header.Number) {
	//	// Verify BaseFee not present before EIP-1559 fork.
	//	if header.BaseFee != nil {
	//		return fmt.Errorf("invalid baseFee before fork: have %d, want <nil>", header.BaseFee)
	//	}
	//	if err := misc.VerifyGaslimit(parent.GasLimit, header.GasLimit); err != nil {
	//		return err
	//	}
	//} else if err := misc.VerifyEip1559Header(chain.Config(), parent, header); err != nil {
	//	// Verify the header's EIP-1559 attributes.
	//	return err
	//}
	// Get active validators number
	/// TODO: pass number of validators, max gas per block and creators per slot, when code will become reachable
	if err := misc.VerifyEip1559Header(chain.Config(), parent, header, 0, 0, 0); err != nil {
		// Verify the header's EIP-1559 attributes.
		return err
	}

	//// Retrieve the snapshot needed to verify this header and cache it
	//snap, err := c.snapshot(chain, number-1, header.ParentHashes[0], parents)
	//if err != nil {
	//	return err
	//}
	//// If the block is a checkpoint block, verify the signer list
	//if number%c.config.Epoch == 0 {
	//	signers := make([]byte, len(snap.Signers)*common.AddressLength)
	//	for i, signer := range snap.signers() {
	//		copy(signers[i*common.AddressLength:], signer[:])
	//	}
	//	extraSuffix := len(header.Extra) - extraSeal
	//	if !bytes.Equal(header.Extra[extraVanity:extraSuffix], signers) {
	//		return errMismatchingCheckpointSigners
	//	}
	//}
	// All basic checks passed, verify the seal and return
	return c.verifySeal(chain, header, parents)
}

// snapshot retrieves the authorization snapshot at a given point in time.
func (c *Sealer) snapshot(chain consensus.ChainHeaderReader, number uint64, hash common.Hash, parents []*types.Header) (*Snapshot, error) {
	// Search for a snapshot in memory or on disk for checkpoints
	var (
		headers []*types.Header
		snap    *Snapshot
	)
	for snap == nil {
		// If an in-memory snapshot was found, use that
		if s, ok := c.recents.Get(hash); ok {
			snap = s.(*Snapshot)
			break
		}
		// If an on-disk checkpoint snapshot can be found, use that
		if number%checkpointInterval == 0 {
			//if s, err := loadSnapshot(c.config, c.signatures, c.db, hash); err == nil {
			if s, err := loadSnapshot(c.signatures, c.db, hash); err == nil {
				log.Trace("Loaded voting snapshot from disk", "hash", hash)
				snap = s
				break
			}
		}

		// If we're at the genesis, snapshot the initial state. Alternatively if we're
		// at a checkpoint block without a parent (light client CHT), or we have piled
		// up more headers than allowed to be reorged (chain reinit from a freezer),
		// consider the checkpoint trusted and snapshot it.
		//if number == 0 || (number%c.config.Epoch == 0 && (len(headers) > params.FullImmutabilityThreshold || chain.GetHeaderByNumber(number-1) == nil)) {
		if number == 0 || (len(headers) > params.FullImmutabilityThreshold || chain.GetHeaderByNumber(number-1) == nil) {
			checkpoint := chain.GetHeaderByNumber(number)
			if checkpoint != nil {
				hash := checkpoint.Hash()

				signers := make([]common.Address, (len(checkpoint.Extra)-extraVanity-extraSeal)/common.AddressLength)
				for i := 0; i < len(signers); i++ {
					copy(signers[i][:], checkpoint.Extra[extraVanity+i*common.AddressLength:])
				}
				snap = newSnapshot(c.signatures, number, hash, signers)
				//snap = newSnapshot(c.config, c.signatures, number, hash, signers)
				if err := snap.store(c.db); err != nil {
					return nil, err
				}
				log.Info("Stored checkpoint snapshot to disk", "number", number, "hash", hash)
				break
			}
		}
		// No snapshot for this header, gather the header and move backward
		//var header *types.Header
		//if len(parents) > 0 {
		//	// If we have explicit parents, pick from there (enforced)
		//	header = parents[len(parents)-1]
		//	//if header.Hash() != hash || header.Number.Uint64() != number {
		//	if header.Hash() != hash {
		//		return nil, consensus.ErrUnknownAncestor
		//	}
		//	parents = parents[:len(parents)-1]
		//} else {
		//No explicit parents (or no more left), reach out to the database
		header := chain.GetHeader(hash)
		if header == nil {
			return nil, consensus.ErrUnknownAncestor
		}
		//}
		headers = append(headers, header)
		number = number - 1
		if hh := chain.GetHeaderByNumber(number); hh != nil {
			hash = hh.Hash()
		}
	}
	// Previous snapshot found, apply any pending headers on top of it
	for i := 0; i < len(headers)/2; i++ {
		headers[i], headers[len(headers)-1-i] = headers[len(headers)-1-i], headers[i]
	}
	snap, err := snap.apply(headers)
	if err != nil {
		return nil, err
	}
	c.recents.Add(snap.Hash, snap)

	// If we've generated a new checkpoint snapshot, save to disk
	if snap.Number%checkpointInterval == 0 && len(headers) > 0 {
		if err = snap.store(c.db); err != nil {
			return nil, err
		}
		log.Trace("Stored voting snapshot to disk", "number", snap.Number, "hash", snap.Hash)
	}
	return snap, err
}

// verifySeal checks whether the signature contained in the header satisfies the
// consensus protocol requirements. The method accepts an optional list of parent
// headers that aren't yet part of the local blockchain to generate the snapshots
// from.
func (c *Sealer) verifySeal(chain consensus.ChainHeaderReader, header *types.Header, parents []*types.Header) error {
	//todo
	//// Verifying the genesis block is not supported
	//number := header.Number.Uint64()
	//if number == 0 {
	//	return errUnknownBlock
	//}
	//// Retrieve the snapshot needed to verify this header and cache it
	//snap, err := c.snapshot(chain, number-1, header.ParentHashes[0], parents)
	//if err != nil {
	//	return err
	//}
	//
	//// Resolve the authorization key and check against signers
	//signer, err := ecrecover(header, c.signatures)
	//if err != nil {
	//	return err
	//}
	//if _, ok := snap.Signers[signer]; !ok {
	//	return errUnauthorizedSigner
	//}
	//for seen, recent := range snap.Recents {
	//	if recent == signer {
	//		// Signer is among recents, only fail if the current block doesn't shift it out
	//		if limit := uint64(len(snap.Signers)/2 + 1); seen > number-limit {
	//			return errRecentlySigned
	//		}
	//	}
	//}
	//// Ensure that the difficulty corresponds to the turn-ness of the signer
	//if !c.fakeDiff {
	//	inturn := snap.inturn(header.Number.Uint64(), signer)
	//	if inturn && header.Difficulty.Cmp(diffInTurn) != 0 {
	//		return errWrongDifficulty
	//	}
	//	if !inturn && header.Difficulty.Cmp(diffNoTurn) != 0 {
	//		return errWrongDifficulty
	//	}
	//}
	return nil
}

// Prepare implements consensus.Engine, preparing all the consensus fields of the
// header for running the transactions on top.
func (c *Sealer) Prepare(chain consensus.ChainHeaderReader, header *types.Header) error {
	//// If the block isn't a checkpoint, cast a random vote (good enough for now)
	//header.Coinbase = common.Address{}
	//header.Nonce = types.BlockNonce{}
	////todo
	//number := header.Height
	//// Assemble the voting snapshot to check which votes make sense
	//snap, err := c.snapshot(chain, number-1, header.ParentHash, nil)
	//if err != nil {
	//	return err
	//}
	//if number%c.config.Epoch != 0 {
	//	c.lock.RLock()
	//
	//	// Gather all the proposals that make sense voting on
	//	addresses := make([]common.Address, 0, len(c.proposals))
	//	for address, authorize := range c.proposals {
	//		if snap.validVote(address, authorize) {
	//			addresses = append(addresses, address)
	//		}
	//	}
	//	// If there's pending proposals, cast a vote on them
	//	if len(addresses) > 0 {
	//		header.Coinbase = addresses[rand.Intn(len(addresses))]
	//		if c.proposals[header.Coinbase] {
	//			copy(header.Nonce[:], nonceAuthVote)
	//		} else {
	//			copy(header.Nonce[:], nonceDropVote)
	//		}
	//	}
	//	c.lock.RUnlock()
	//}
	//// Set the correct difficulty
	//header.Difficulty = calcDifficulty(snap, c.signer)

	// Ensure the extra data has all its components
	if len(header.Extra) < extraVanity {
		header.Extra = append(header.Extra, bytes.Repeat([]byte{0x00}, extraVanity-len(header.Extra))...)
	}
	header.Extra = header.Extra[:extraVanity]

	//if number%c.config.Epoch == 0 {
	//	for _, signer := range snap.signers() {
	//		// signer - common.Address
	//		header.Extra = append(header.Extra, signer[:]...)
	//	}
	//}
	header.Extra = append(header.Extra, c.signer[:]...)

	header.Extra = append(header.Extra, make([]byte, extraSeal)...)

	//// Ensure the timestamp has the correct delay
	////todo
	//parent := chain.GetHeader(header.ParentHash, number-1)
	//if parent == nil {
	//	return consensus.ErrUnknownAncestor
	//}
	//header.Time = parent.Time + c.config.Period
	//if header.Time < uint64(time.Now().Unix()) {
	//	header.Time = uint64(time.Now().Unix())
	//}
	return nil
}

// Finalize implements consensus.Engine
// rewards given.
func (c *Sealer) Finalize(chain consensus.ChainHeaderReader, header *types.Header, state *state.StateDB, txs []*types.Transaction) {
	// No block rewards in PoA, so the state remains as is
	//header.Root = state.IntermediateRoot(chain.Config().IsEIP158(header.Number))
	header.Root = state.IntermediateRoot(true)
}

// FinalizeAndAssemble implements consensus.Engine
// nor block rewards given, and returns the final block.
func (c *Sealer) FinalizeAndAssemble(chain consensus.ChainHeaderReader, header *types.Header, state *state.StateDB, txs []*types.Transaction, receipts []*types.Receipt) (*types.Block, error) {
	// Finalize block
	c.Finalize(chain, header, state, txs)

	// Assemble and return the final block for sealing
	return types.NewBlock(header, txs, receipts, trie.NewStackTrie(nil)), nil
}

// Authorize injects a private key into the consensus engine to mint new blocks
// with.
func (c *Sealer) Authorize(signer common.Address, signFn SignerFn) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.signer = signer
	c.signFn = signFn
}

// Seal implements consensus.Engine, attempting to create a sealed block using
// the local signing credentials.
func (c *Sealer) Seal(chain consensus.ChainHeaderReader, block *types.Block, results chan<- *types.Block, stop <-chan struct{}) error {
	header := block.Header()

	// Sealing the genesis block is not supported
	if block.Height() == 0 {
		return errUnknownBlock
	}

	// Don't hold the signer fields for the entire sealing procedure
	c.lock.RLock()
	signer, signFn := c.signer, c.signFn
	c.lock.RUnlock()

	// Bail out if we're unauthorized to sign a block
	//snap, err := c.snapshot(chain, number-1, header.ParentHash, nil)
	//snap, err := c.snapshot(chain, stateHeight, stateHash, nil)
	//if err != nil {
	//	return err
	//}

	// Sign all the things!
	//sighash, err := signFn(accounts.Account{Address: signer}, accounts.MimetypeClique, CliqueRLP(header))
	_, err := signFn(accounts.Account{Address: signer}, accounts.MimetypeClique, CliqueRLP(header))
	if err != nil {
		return err
	}
	//copy(header.Extra[len(header.Extra)-extraSeal:], sighash)

	// Wait until sealing is terminated or delay timeout.
	log.Trace("Waiting for slot to sign and propagate", "delay", common.PrettyDuration(0))
	//go func() {
	select {
	case <-stop:
		return nil
	case <-time.After(0):
	}

	select {
	case results <- block.WithSeal(header):
	default:
		log.Warn("Sealing result is not read by miner", "sealhash", SealHash(header))
		return errors.New("Sealing result is not read by miner")
	}
	//}()

	return nil
}

// SealHash returns the hash of a block prior to it being sealed.
func (c *Sealer) SealHash(header *types.Header) common.Hash {
	return SealHash(header)
}

// Close implements consensus.Engine. It's a noop for clique as there are no background threads.
func (c *Sealer) Close() error {
	return nil
}

// APIs implements consensus.Engine, returning the user facing RPC API to allow
// controlling the signer voting.
func (c *Sealer) APIs(chain consensus.ChainHeaderReader) []rpc.API {
	return []rpc.API{{
		Namespace: "clique",
		Version:   "1.0",
		Service:   &API{chain: chain, clique: c},
		Public:    false,
	}}
}

// SealHash returns the hash of a block prior to it being sealed.
func SealHash(header *types.Header) (hash common.Hash) {
	hasher := sha3.NewLegacyKeccak256()
	encodeSigHeader(hasher, header)
	hasher.(crypto.KeccakState).Read(hash[:])
	return hash
}

// CliqueRLP returns the rlp bytes which needs to be signed for the proof-of-authority
// sealing. The RLP to sign consists of the entire header apart from the 65 byte signature
// contained at the end of the extra data.
//
// Note, the method requires the extra data to be at least 65 bytes, otherwise it
// panics. This is done to avoid accidentally using both forms (signature present
// or not), which could be abused to produce different hashes for the same header.
func CliqueRLP(header *types.Header) []byte {
	b := new(bytes.Buffer)
	encodeSigHeader(b, header)
	return b.Bytes()
}

func encodeSigHeader(w io.Writer, header *types.Header) {
	enc := []interface{}{
		header.ParentHashes,
		header.Height,
		header.LFHash,
		header.LFNumber,
		header.Coinbase,
		header.Root,
		header.TxHash,
		header.ReceiptHash,
		header.Bloom,
		header.GasLimit,
		header.GasUsed,
		header.BodyHash,
		header.Time,
		//header.Extra[:len(header.Extra)-crypto.SignatureLength], // Yes, this will panic if extra is too short
		header.Extra[:], // Yes, this will panic if extra is too short
	}
	if header.BaseFee != nil {
		enc = append(enc, header.BaseFee)
	}
	if err := rlp.Encode(w, enc); err != nil {
		panic("can't encode: " + err.Error())
	}
}
