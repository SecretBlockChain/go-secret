package senate

import (
	"encoding/binary"
	"errors"
	"math/big"
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

// Senate delegated-proof-of-stake protocol constants.
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

// Senate is the delegated-proof-of-stake consensus engine.
type Senate struct {
	db         ethdb.Database       // Database to store and retrieve snapshot checkpoints
	signatures *lru.ARCCache        // Signatures of recent blocks to speed up mining
	config     *params.SenateConfig // Consensus engine configuration parameters
	signer     common.Address       // Ethereum address of the signing key
	signFn     SignerFn             // Signer function to authorize hashes with
	lock       sync.RWMutex         // Protects the signer fields
}

// New creates a Senate delegated-proof-of-stake consensus engine with the initial
// signers set to the ones provided by the user.
func New(config *params.SenateConfig, db ethdb.Database) *Senate {
	config.Rewards.Sort()
	signatures, _ := lru.NewARC(inMemorySignatures)
	return &Senate{db: db, signatures: signatures, config: config}
}

// Close terminates any background threads maintained by the consensus engine.
func (senate *Senate) Close() error {
	return nil
}

// APIs returns the RPC APIs this consensus engine provides.
func (senate *Senate) APIs(chain consensus.ChainHeaderReader) []rpc.API {
	return []rpc.API{{
		Namespace: "dpos",
		Version:   "1.0",
		Service:   &API{chain: chain, senate: senate},
		Public:    true,
	}}
}

// Authorize injects a private key into the consensus engine to mint new blocks
// with.
func (senate *Senate) Authorize(signer common.Address, signFn SignerFn) {
	senate.lock.Lock()
	defer senate.lock.Unlock()

	senate.signer = signer
	senate.signFn = signFn
}

// InTurn returns if a signer at a given block height is in-turn or not.
func (senate *Senate) InTurn(lastBlockHeader *types.Header, now uint64) bool {
	config, err := senate.chainConfig(lastBlockHeader)
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

	senate.lock.Lock()
	signer := senate.signer
	senate.lock.Unlock()
	return senate.inTurn(config, lastBlockHeader, nexBlockTime, signer)
}

func (senate *Senate) inTurn(config params.SenateConfig,
	lastBlockHeader *types.Header, nexBlockTime uint64, signer common.Address) bool {

	validators := config.Validators
	epochTime := config.GenesisTimestamp

	if lastBlockHeader != nil && lastBlockHeader.Number.Int64() > 0 {
		headerExtra, err := decodeHeaderExtra(lastBlockHeader)
		if err != nil {
			return false
		}

		epochTime = headerExtra.EpochTime
		snap, err := loadSnapshot(senate.db, headerExtra.Root)
		if err != nil {
			return false
		}

		addresses, err := snap.GetValidators()
		if err != nil {
			return false
		}

		validators = make([]common.Address, 0, len(addresses))
		for _, address := range addresses {
			validators = append(validators, address.Address)
		}
	}

	count := len(validators)
	if count == 0 {
		return false
	}

	idx := (nexBlockTime - epochTime) / config.Period % uint64(len(validators))
	return validators[idx] == signer
}

// Gets the chain config for the specified block height.
func (senate *Senate) chainConfig(header *types.Header) (params.SenateConfig, error) {
	if header == nil || header.Number.Int64() == 0 {
		return *senate.config, nil
	}

	headerExtra, err := decodeHeaderExtra(header)
	if err != nil {
		return params.SenateConfig{}, err
	}
	return senate.chainConfigByHash(headerExtra.Root.ConfigHash)
}

// Gets the chain config by tire node hash value.
func (senate *Senate) chainConfigByHash(configHash common.Hash) (params.SenateConfig, error) {
	zero := common.Hash{}
	if configHash == zero {
		return *senate.config, nil
	}

	snap := Snapshot{
		db:   trie.NewDatabase(senate.db),
		root: Root{ConfigHash: configHash},
	}
	config, err := snap.GetChainConfig()
	if err != nil {
		return params.SenateConfig{}, ErrChainConfigMissing
	}
	return config, nil
}

// Elect validators in first block for epoch.
func (senate *Senate) tryElect(config params.SenateConfig, state *state.StateDB, header *types.Header,
	snap *Snapshot, headerExtra *HeaderExtra) error {

	// Is come to new epoch?
	if header.Time != headerExtra.EpochTime {
		return nil
	}

	// Find not active validators
	needKickOutValidators := make(SortableAddresses, 0)
	if header.Number.Uint64() <= 1 {
		for _, validator := range config.Validators {
			if err := snap.BecomeCandidate(validator); err != nil {
				return err
			}
			if err := snap.Delegate(validator, validator); err != nil {
				return err
			}
			headerExtra.CurrentBlockDelegates = append(headerExtra.CurrentBlockDelegates, Delegate{
				Delegator: validator,
				Candidate: validator,
			})
			headerExtra.CurrentBlockCandidates = append(headerExtra.CurrentBlockCandidates, validator)
		}

		headerExtra.CurrentBlockDelegates = delegatesDistinct(headerExtra.CurrentBlockDelegates)
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
				log.Info("[DPOS] No more candidate can be kick out",
					"prevEpochID", headerExtra.Epoch-1,
					"candidateCount", candidateCount, "needKickOutCount", len(needKickOutValidators)-i)
				return nil
			}

			if err := snap.KickOutCandidate(validator.Address); err != nil {
				return err
			}

			// If kick out success, candidateCount minus 1
			candidateCount--
			headerExtra.CurrentBlockKickOutCandidates = append(headerExtra.CurrentBlockKickOutCandidates, validator.Address)
			log.Info("[DPOS] Kick out candidate",
				"prevEpochID", headerExtra.Epoch-1, "candidate", validator, "mintCnt", validator.Weight.String())
		}
	}

	// Elect next epoch validators by votes
	//candidates, err := snap.TopCandidates(state, int(config.MaxValidatorsCount))

	// Shuffle candidates
	seed := int64(binary.LittleEndian.Uint32(crypto.Keccak512(header.ParentHash.Bytes())))
	candidates, err := snap.RandCandidates(seed, int(config.MaxValidatorsCount))
	if err != nil {
		return err
	}
	//TODO test print log
	printLog(candidates)

	headerExtra.CurrentEpochValidators = append(headerExtra.CurrentEpochValidators, candidates...)
	return snap.SetValidators(headerExtra.CurrentEpochValidators)
}

func printLog(candidates SortableAddresses)  {

	addrs := ""
	for _,addr := range candidates {
		addrs = addr.Address.String() + "\n"
	}
	log.Info("rand candidates ",addrs)
}

// Credits the coinbase of the given block with the mining reward.
func (senate *Senate) accumulateRewards(config params.SenateConfig, state *state.StateDB, header *types.Header) {
	var blockReward *big.Int
	number := header.Number.Uint64()
	for _, reward := range config.Rewards {
		blockReward = reward.Reward
		if reward.Height > number {
			break
		}
	}

	if blockReward == nil || blockReward.Cmp(big.NewInt(0)) <= 0 {
		return
	}
	reward := new(big.Int).Set(blockReward)
	state.AddBalance(header.Coinbase, reward)
	log.Info("[DPOS] Accumulate rewards", "address", header.Coinbase, "amount", reward)
}

// Process custom transactions, write into header.Extra.
func (senate *Senate) processTransactions(config params.SenateConfig, state *state.StateDB, header *types.Header,
	snap *Snapshot, headerExtra *HeaderExtra, txs []*types.Transaction, receipts []*types.Receipt) {

	if header.Number.Int64() <= 1 {
		if err := snap.SetChainConfig(config); err != nil {
			panic(err)
		}
		headerExtra.ChainConfig = []params.SenateConfig{config}
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
			case *EventDelegate:
				event := ctx.(*EventDelegate)
				if state.GetBalance(event.Delegator).Cmp(config.MinDelegatorBalance) == -1 {
					break
				}
				if err = snap.Delegate(event.Delegator, event.Candidate); err == nil {
					headerExtra.CurrentBlockDelegates = append(headerExtra.CurrentBlockDelegates, Delegate{
						Delegator: event.Delegator,
						Candidate: event.Candidate,
					})
				}
				count++
			case *EventBecomeCandidate:
				event := ctx.(*EventBecomeCandidate)
				if state.GetBalance(event.Candidate).Cmp(config.MinCandidateBalance) == -1 {
					break
				}
				if err = snap.BecomeCandidate(event.Candidate); err == nil {
					headerExtra.CurrentBlockCandidates = append(headerExtra.CurrentBlockCandidates, event.Candidate)
				}
				count++
			}
		}
	}

	headerExtra.CurrentBlockDelegates = delegatesDistinct(headerExtra.CurrentBlockDelegates)
	headerExtra.CurrentBlockCandidates = addressesDistinct(headerExtra.CurrentBlockCandidates)

	log.Trace("[DPOS] Processing transactions done", "txs", count)
}
