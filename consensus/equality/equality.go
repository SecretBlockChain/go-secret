package equality

import (
	"encoding/binary"
	"errors"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/SecretBlockChain/go-secret/accounts"
	"github.com/SecretBlockChain/go-secret/common"
	"github.com/SecretBlockChain/go-secret/consensus"
	"github.com/SecretBlockChain/go-secret/core/state"
	"github.com/SecretBlockChain/go-secret/core/types"
	"github.com/SecretBlockChain/go-secret/crypto"
	"github.com/SecretBlockChain/go-secret/ethdb"
	"github.com/SecretBlockChain/go-secret/log"
	"github.com/SecretBlockChain/go-secret/params"
	"github.com/SecretBlockChain/go-secret/rpc"
	"github.com/SecretBlockChain/go-secret/trie"
	lru "github.com/hashicorp/golang-lru"
)

// Equality proof-of-equality protocol constants.
var (
	extraVanity        = 32                       // Fixed number of extra-data prefix bytes reserved for signer vanity
	extraSeal          = crypto.SignatureLength   // Fixed number of extra-data suffix bytes reserved for signer seal
	defaultDifficulty  = int64(1)                 // Default difficulty
	inmemorySnapshots  = 12                       // Number of recent vote snapshots to keep in memory
	inMemorySignatures = 4096                     // Number of recent block signatures to keep in memory
	uncleHash          = types.CalcUncleHash(nil) // Always Keccak256(RLP([])) as uncles are meaningless outside of PoW.
)

// Various error messages to mark blocks invalid. These should be private to
// prevent engine specific errors from being referenced in the remainder of the
// codebase, inherently breaking if the engine is swapped out. Please put common
// error types into the consensus package.
var (
	// errUnauthorized is returned if a header is signed by a non-authorized entity.
	errUnauthorized = errors.New("unauthorized")

	// errUnknownBlock is returned when the list of signers is requested for a block
	// that is not part of the local blockchain.
	errUnknownBlock = errors.New("unknown block")

	// errMissingVanity is returned if a block's extra-data section is shorter than
	// 32 bytes, which is required to store the signer vanity.
	errMissingVanity = errors.New("extra-data 32 byte vanity prefix missing")

	// errUnclesNotAllowed is returned if uncles exists
	errUnclesNotAllowed = errors.New("uncles not allowed")

	// errMissingSignature is returned if a block's extra-data section doesn't seem
	// to contain a 65 byte secp256k1 signature.
	errMissingSignature = errors.New("extra-data 65 byte signature suffix missing")

	// errInvalidMixDigest is returned if a block's mix digest is non-zero.
	errInvalidMixDigest = errors.New("non-zero mix digest")

	// errInvalidUncleHash is returned if a block contains an non-empty uncle list.
	errInvalidUncleHash = errors.New("non empty uncle hash")

	// ErrInvalidTimestamp is returned if the timestamp of a block is lower than
	// the previous block's timestamp + the minimum block period.
	ErrInvalidTimestamp = errors.New("invalid timestamp")

	// ErrChainConfigMissing is returned if the chain config is missing
	ErrChainConfigMissing = errors.New("chain config missing")
)

type SignerFn func(accounts.Account, string, []byte) ([]byte, error)

// Equality is the proof-of-equality consensus engine.
type Equality struct {
	db         ethdb.Database         // Database to store and retrieve snapshot checkpoints
	signatures *lru.ARCCache          // Signatures of recent blocks to speed up mining
	config     *params.EqualityConfig // Consensus engine configuration parameters
	signer     common.Address         // Ethereum address of the signing key
	signFn     SignerFn               // Signer function to authorize hashes with
	lock       sync.RWMutex           // Protects the signer fields
}

// New creates a Equality proof-of-equality consensus engine with the initial
// signers set to the ones provided by the user.
func New(config *params.EqualityConfig, db ethdb.Database) *Equality {
	signatures, _ := lru.NewARC(inMemorySignatures)
	return &Equality{db: db, signatures: signatures, config: config}
}

// Close terminates any background threads maintained by the consensus engine.
func (e *Equality) Close() error {
	return nil
}

// APIs returns the RPC APIs this consensus engine provides.
func (e *Equality) APIs(chain consensus.ChainHeaderReader) []rpc.API {
	return []rpc.API{{
		Namespace: "eq",
		Version:   "1.0",
		Service:   &API{chain: chain, equality: e},
		Public:    true,
	}}
}

// Authorize injects a private key into the consensus engine to mint new blocks
// with.
func (e *Equality) Authorize(signer common.Address, signFn SignerFn) {
	e.lock.Lock()
	defer e.lock.Unlock()

	e.signer = signer
	e.signFn = signFn
}

// InTurn returns if a signer at a given block height is in-turn or not.
func (e *Equality) InTurn(lastBlockHeader *types.Header, now uint64) bool {
	config, err := e.chainConfig(lastBlockHeader)
	if err != nil {
		return false
	}
	if now <= config.GenesisTimestamp-config.Period {
		return false
	}

	// Estimate the next block time
	nexBlockTime := lastBlockHeader.Time + config.Period
	if int64(nexBlockTime) < time.Now().Unix() {
		nexBlockTime = uint64(time.Now().Unix())
	}

	e.lock.Lock()
	signer := e.signer
	e.lock.Unlock()
	return e.inTurn(config, lastBlockHeader, nexBlockTime, signer)
}

func (e *Equality) inTurn(config params.EqualityConfig,
	lastBlockHeader *types.Header, nexBlockTime uint64, signer common.Address) bool {

	validators := config.Validators
	if lastBlockHeader != nil && lastBlockHeader.Number.Int64() > 0 {
		headerExtra, err := DecodeHeaderExtra(lastBlockHeader)
		if err != nil {
			return false
		}

		snap, err := loadSnapshot(e.db, headerExtra.Root)
		if err != nil {
			return false
		}

		validators, err = snap.GetValidators()
		if err != nil {
			return false
		}
	}

	count := len(validators)
	if count == 0 {
		return false
	}

	idx := (nexBlockTime - config.GenesisTimestamp) / config.Period % uint64(len(validators))
	return validators[idx] == signer
}

// Gets the chain config for the specified block number.
func (e *Equality) chainConfig(header *types.Header) (params.EqualityConfig, error) {
	if header == nil || header.Number.Int64() == 0 {
		return *e.config, nil
	}

	headerExtra, err := DecodeHeaderExtra(header)
	if err != nil {
		return params.EqualityConfig{}, err
	}
	return e.chainConfigByHash(headerExtra.Root.ConfigHash)
}

// Gets the chain config by tire node hash value.
func (e *Equality) chainConfigByHash(configHash common.Hash) (params.EqualityConfig, error) {
	zero := common.Hash{}
	if configHash == zero {
		return *e.config, nil
	}

	snap := Snapshot{
		db:   trie.NewDatabase(e.db),
		root: Root{ConfigHash: configHash},
	}
	config, err := snap.GetChainConfig()
	if err != nil {
		return params.EqualityConfig{}, ErrChainConfigMissing
	}
	return config, nil
}

func validatorsToString(validators []common.Address) string {
	slice := make([]string, 0, len(validators))
	for _, validator := range validators {
		slice = append(slice, validator.String())
	}
	return "[" + strings.Join(slice, ",") + "]"
}

// Elect validators in first block for epoch.
func (e *Equality) tryElect(config params.EqualityConfig, header *types.Header,
	snap *Snapshot, headerExtra *HeaderExtra) error {

	// Is come to next epoch?
	number := header.Number.Uint64()
	if number != headerExtra.EpochBlock {
		return nil
	}

	// Find not active validators
	needKickOutValidators := make(SortableAddresses, 0)
	if number <= 1 {
		for _, validator := range config.Validators {
			if err := snap.BecomeCandidate(validator, 1, big.NewInt(0)); err != nil {
				return err
			}
			headerExtra.CurrentBlockCandidates = append(headerExtra.CurrentBlockCandidates, validator)
		}

		headerExtra.CurrentBlockCandidates = addressesDistinct(headerExtra.CurrentBlockCandidates)
	} else {
		minMint := big.NewInt(int64(config.Epoch / config.Period / config.MaxValidatorsCount / 2))
		validators, err := snap.CountMinted(headerExtra.Epoch - 1)
		if err != nil {
			return err
		}
		for _, validator := range validators {
			if validator.Weight.Cmp(minMint) == -1 {
				needKickOutValidators = append(needKickOutValidators, validator)
			}
		}
	}

	// Kick out not active validators
	if len(needKickOutValidators) > 0 {
		safeSize := int(config.MaxValidatorsCount*2/3 + 1)
		candidateCount, _ := snap.EnoughCandidates(safeSize + len(needKickOutValidators))
		for i, validator := range needKickOutValidators {
			// Ensure candidate count greater than or equal to safeSize
			if candidateCount <= safeSize {
				log.Info("[equality] No more candidate can be kick out",
					"prevEpochID", headerExtra.Epoch-1,
					"candidateCount", candidateCount, "needKickOutCount", len(needKickOutValidators)-i)
				break
			}

			if _, err := snap.CancelCandidate(validator.Address); err != nil {
				return err
			}

			// If kick out success, candidateCount minus 1
			candidateCount--
			headerExtra.CurrentBlockKickOutCandidates = append(headerExtra.CurrentBlockKickOutCandidates, validator.Address)
			log.Info("[equality] Kick out candidate",
				"prevEpochID", headerExtra.Epoch-1, "candidate", validator, "mintCnt", validator.Weight.String())
		}
	}

	// Shuffle candidates
	seed := int64(binary.LittleEndian.Uint32(crypto.Keccak512(header.ParentHash.Bytes())))
	candidates, err := snap.RandCandidates(seed, int(config.MaxValidatorsCount))
	if err != nil {
		return err
	}

	headerExtra.CurrentEpochValidators = append(headerExtra.CurrentEpochValidators, candidates...)
	log.Debug("[equality] Come to next epoch",
		"number", number, "epoch", headerExtra.Epoch, "validators", validatorsToString(headerExtra.CurrentEpochValidators))
	return snap.SetValidators(headerExtra.CurrentEpochValidators)
}

// Credits the coinbase of the given block with the mining reward.
func (e *Equality) accumulateRewards(config params.EqualityConfig, state *state.StateDB, header *types.Header) {
	var blockReward *big.Int
	number := header.Number.Uint64()
	for _, reward := range config.Rewards {
		blockReward = reward.Reward
		if reward.Number >= number {
			break
		}
	}

	if blockReward == nil || blockReward.Cmp(big.NewInt(0)) <= 0 {
		return
	}

	base := big.NewInt(0).Div(blockReward, big.NewInt(10))
	state.AddBalance(header.Coinbase, base)
	state.AddBalance(config.Pool, big.NewInt(0).Sub(blockReward, base))

	log.Debug("[equality] Accumulate rewards",
		"coinbase", header.Coinbase, "amount", base,
		"pool", config.Pool, "amount", big.NewInt(0).Sub(blockReward, base))
}

// Process custom transactions, write into header.Extra.
func (e *Equality) processTransactions(config params.EqualityConfig, state *state.StateDB, header *types.Header,
	snap *Snapshot, headerExtra *HeaderExtra, txs []*types.Transaction) {

	number := header.Number.Uint64()
	if number <= 1 {
		if err := snap.SetChainConfig(config); err != nil {
			panic(err)
		}
		headerExtra.ChainConfig = []params.EqualityConfig{config}
	}

	count := 0
	for _, tx := range txs {
		ctx, err := NewTransaction(tx)
		if err != nil {
			continue
		}

		switch ctx.Type() {
		case EventTransactionType:
			switch ctx.(type) {
			case *EventBecomeCandidate:
				event := ctx.(*EventBecomeCandidate)
				if state.GetBalance(event.Candidate).Cmp(config.MinCandidateBalance) == -1 {
					break
				}
				if err = snap.BecomeCandidate(event.Candidate, number, config.MinCandidateBalance); err == nil {
					state.SubBalance(event.Candidate, config.MinCandidateBalance)
					headerExtra.CurrentBlockCandidates = append(headerExtra.CurrentBlockCandidates, event.Candidate)
				}
				count++
			case *EventCancelCandidate:
				event := ctx.(*EventCancelCandidate)
				if security, err := snap.CancelCandidate(event.Delegator); err == nil {
					state.AddBalance(event.Delegator, security)
					headerExtra.CurrentBlockCancelCandidates = append(headerExtra.CurrentBlockCancelCandidates, event.Delegator)
				}
				count++
			}
		}
	}

	headerExtra.CurrentBlockCandidates = addressesDistinct(headerExtra.CurrentBlockCandidates)

	log.Trace("[equality] Processing transactions done", "txs", count)
}
