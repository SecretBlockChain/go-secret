package senate

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sort"

	"github.com/SecretBlockChain/go-secret/common"
	"github.com/SecretBlockChain/go-secret/core/state"
	"github.com/SecretBlockChain/go-secret/core/types"
	"github.com/SecretBlockChain/go-secret/ethdb"
	"github.com/SecretBlockChain/go-secret/params"
	"github.com/SecretBlockChain/go-secret/rlp"
	"github.com/SecretBlockChain/go-secret/trie"
)

var (
	epochPrefix     = []byte("epoch-")     // epoch-validator:{validators}
	delegatePrefix  = []byte("delegate-")  // delegate-{candidateAddr}..{delegatorAddr}:{delegatorAddr}
	votePrefix      = []byte("vote-")      // vote-{delegatorAddr}:{candidateAddr}
	candidatePrefix = []byte("candidate-") // candidate-{candidateAddr}:
	mintCntPrefix   = []byte("mintCnt-")   // mintCnt-{epoch}..{validator}:{count}
	configPrefix    = []byte("config-")    // config..{params.SenateConfig}
)

func newEpochTrie(root common.Hash, db *trie.Database) (*Trie, error) {
	return NewTrieWithPrefix(root, epochPrefix, db)
}

func newDelegateTrie(root common.Hash, db *trie.Database) (*Trie, error) {
	return NewTrieWithPrefix(root, delegatePrefix, db)
}

func newVoteTrie(root common.Hash, db *trie.Database) (*Trie, error) {
	return NewTrieWithPrefix(root, votePrefix, db)
}

func newCandidateTrie(root common.Hash, db *trie.Database) (*Trie, error) {
	return NewTrieWithPrefix(root, candidatePrefix, db)
}

func newMintCntTrie(root common.Hash, db *trie.Database) (*Trie, error) {
	return NewTrieWithPrefix(root, mintCntPrefix, db)
}

func newConfigTrie(root common.Hash, db *trie.Database) (*Trie, error) {
	return NewTrieWithPrefix(root, configPrefix, db)
}

// SortableAddress sorted by votes.
type SortableAddress struct {
	Address common.Address `json:"address"`
	Weight  *big.Int       `json:"weight"`
}

// SortableAddresses sorting in descending order by weight.
type SortableAddresses []SortableAddress

func (p SortableAddresses) Swap(i, j int) { p[i], p[j] = p[j], p[i] }
func (p SortableAddresses) Len() int      { return len(p) }
func (p SortableAddresses) Less(i, j int) bool {
	if p[i].Weight.Cmp(p[j].Weight) < 0 {
		return false
	} else if p[i].Weight.Cmp(p[j].Weight) > 0 {
		return true
	} else {
		return p[i].Address.String() < p[j].Address.String()
	}
}

// Snapshot is the state of the authorization voting at a given block number.
type Snapshot struct {
	root          Root
	epochTrie     *Trie
	delegateTrie  *Trie
	voteTrie      *Trie
	candidateTrie *Trie
	mintCntTrie   *Trie
	configTrie    *Trie
	db            *trie.Database
}

// newSnapshot creates a new empty snapshot
// only ever use if for the genesis block.
func newSnapshot(diskdb ethdb.Database) (*Snapshot, error) {
	snap := Snapshot{
		db: trie.NewDatabase(diskdb),
	}

	var err error
	snap.epochTrie, err = newEpochTrie(common.Hash{}, snap.db)
	if err != nil {
		return nil, err
	}
	snap.delegateTrie, err = newDelegateTrie(common.Hash{}, snap.db)
	if err != nil {
		return nil, err
	}
	snap.voteTrie, err = newVoteTrie(common.Hash{}, snap.db)
	if err != nil {
		return nil, err
	}
	snap.candidateTrie, err = newCandidateTrie(common.Hash{}, snap.db)
	if err != nil {
		return nil, err
	}
	snap.mintCntTrie, err = newMintCntTrie(common.Hash{}, snap.db)
	if err != nil {
		return nil, err
	}
	snap.configTrie, err = newConfigTrie(common.Hash{}, snap.db)
	if err != nil {
		return nil, err
	}
	return &snap, nil
}

// loadSnapshot loads an existing snapshot from the database.
func loadSnapshot(diskdb ethdb.Database, root Root) (*Snapshot, error) {
	snap := Snapshot{
		root: root,
		db:   trie.NewDatabase(diskdb),
	}

	var err error
	snap.epochTrie, err = newEpochTrie(root.EpochHash, snap.db)
	if err != nil {
		return nil, err
	}
	snap.delegateTrie, err = newDelegateTrie(root.DelegateHash, snap.db)
	if err != nil {
		return nil, err
	}
	snap.voteTrie, err = newVoteTrie(root.VoteHash, snap.db)
	if err != nil {
		return nil, err
	}
	snap.candidateTrie, err = newCandidateTrie(root.CandidateHash, snap.db)
	if err != nil {
		return nil, err
	}
	snap.mintCntTrie, err = newMintCntTrie(root.MintCntHash, snap.db)
	if err != nil {
		return nil, err
	}
	snap.configTrie, err = newConfigTrie(root.ConfigHash, snap.db)
	if err != nil {
		return nil, err
	}
	return &snap, nil
}

// apply creates a new authorization snapshot by applying the given headers to
// the original one.
func (snap *Snapshot) apply(header *types.Header, headerExtra HeaderExtra) error {
	for _, candidate := range headerExtra.CurrentBlockCandidates {
		if err := snap.BecomeCandidate(candidate); err != nil {
			return err
		}
	}
	for _, delegate := range headerExtra.CurrentBlockDelegates {
		if err := snap.Delegate(delegate.Delegator, delegate.Candidate); err != nil {
			return err
		}
	}
	for _, candidate := range headerExtra.CurrentBlockKickOutCandidates {
		if err := snap.KickOutCandidate(candidate); err != nil {
			return err
		}
	}
	if header.Time == headerExtra.EpochTime {
		if err := snap.SetValidators(headerExtra.CurrentEpochValidators); err != nil {
			return err
		}
	}
	if len(headerExtra.ChainConfig) > 0 {
		last := len(headerExtra.ChainConfig) - 1
		if err := snap.SetChainConfig(headerExtra.ChainConfig[last]); err != nil {
			return err
		}
	}
	if err := snap.MintBlock(headerExtra.Epoch, header.Number.Uint64(), header.Coinbase); err != nil {
		return err
	}
	return nil
}

// Root returns root of snapshot trie.
func (snap *Snapshot) Root() (root Root, err error) {
	root.EpochHash, err = snap.epochTrie.Commit(nil)
	if err != nil {
		return Root{}, err
	}
	root.DelegateHash, err = snap.delegateTrie.Commit(nil)
	if err != nil {
		return Root{}, err
	}
	root.VoteHash, err = snap.voteTrie.Commit(nil)
	if err != nil {
		return Root{}, err
	}
	root.CandidateHash, err = snap.candidateTrie.Commit(nil)
	if err != nil {
		return Root{}, err
	}
	root.MintCntHash, err = snap.mintCntTrie.Commit(nil)
	if err != nil {
		return Root{}, err
	}
	root.ConfigHash, err = snap.configTrie.Commit(nil)
	if err != nil {
		return Root{}, err
	}
	return root, err
}

// Commit commit snapshot changes to database.
func (snap *Snapshot) Commit(root Root) error {
	if snap.root.EpochHash != root.EpochHash {
		if err := snap.db.Commit(root.EpochHash, false, nil); err != nil {
			return err
		}
	}
	if snap.root.DelegateHash != root.DelegateHash {
		if err := snap.db.Commit(root.DelegateHash, false, nil); err != nil {
			return err
		}
	}
	if snap.root.VoteHash != root.VoteHash {
		if err := snap.db.Commit(root.VoteHash, false, nil); err != nil {
			return err
		}
	}
	if snap.root.CandidateHash != root.CandidateHash {
		if err := snap.db.Commit(root.CandidateHash, false, nil); err != nil {
			return err
		}
	}
	if snap.root.MintCntHash != root.MintCntHash {
		if err := snap.db.Commit(root.MintCntHash, false, nil); err != nil {
			return err
		}
	}
	if snap.root.ConfigHash != root.ConfigHash {
		if err := snap.db.Commit(root.ConfigHash, false, nil); err != nil {
			return err
		}
	}
	snap.root = root
	return nil
}

// GetChainConfig get chain config from snapshot.
func (snap *Snapshot) GetChainConfig() (params.SenateConfig, error) {
	key := []byte("config")
	var config params.SenateConfig
	data := snap.configTrie.Get(key)
	if err := json.Unmarshal(data, &config); err != nil {
		return params.SenateConfig{}, err
	}
	return config, nil
}

// SetChainConfig write chain config to snapshot.
func (snap *Snapshot) SetChainConfig(config params.SenateConfig) error {
	if len(config.Rewards) == 0 {
		config.Rewards = nil
	}
	if len(config.Validators) == 0 {
		config.Validators = nil
	}

	key := []byte("config")
	data, err := json.Marshal(config)
	if err != nil {
		return err
	}
	return snap.configTrie.TryUpdate(key, data)
}

// GetValidators get validators of current epoch from snapshot.
func (snap *Snapshot) GetValidators() (SortableAddresses, error) {
	key := []byte("validator")
	var validators SortableAddresses
	validatorsRLP := snap.epochTrie.Get(key)
	if err := rlp.DecodeBytes(validatorsRLP, &validators); err != nil {
		return nil, fmt.Errorf("failed to decode validators: %s", err)
	}
	return validators, nil
}

// SetValidators write validators of current epoch to snapshot.
func (snap *Snapshot) SetValidators(validators SortableAddresses) error {
	key := []byte("validator")
	validatorsRLP, err := rlp.EncodeToBytes(validators)
	if err != nil {
		return fmt.Errorf("failed to encode validators to rlp bytes: %s", err)
	}

	return snap.epochTrie.TryUpdate(key, validatorsRLP)
}

// CountMinted count the minted of each validator.
func (snap *Snapshot) CountMinted(epoch uint64) (SortableAddresses, error) {
	validators, err := snap.GetValidators()
	if err != nil {
		return nil, err
	}

	prefix := make([]byte, 8)
	binary.BigEndian.PutUint64(prefix, epoch)
	iter := trie.NewIterator(snap.mintCntTrie.PrefixIterator(prefix))

	mapper := make(map[common.Address]int64)
	for iter.Next() {
		validator := common.BytesToAddress(iter.Value)
		count, _ := mapper[validator]
		mapper[validator] = count + 1
	}

	for idx := range validators {
		validator := &validators[idx]
		count, ok := mapper[validator.Address]
		if !ok {
			validator.Weight = big.NewInt(0)
		} else {
			validator.Weight = big.NewInt(count)
		}
	}
	sort.Sort(sort.Reverse(validators))
	return validators, nil
}

// ForgeBlock write validator of block to snapshot.
func (snap *Snapshot) MintBlock(epoch, number uint64, validator common.Address) error {
	key := make([]byte, 16)
	binary.BigEndian.PutUint64(key[:8], epoch)
	binary.BigEndian.PutUint64(key[8:], number)
	return snap.mintCntTrie.TryUpdate(key, validator.Bytes())
}

// CountVotes count the votes of candidate.
func (snap *Snapshot) CountVotes(state *state.StateDB, candidateAddr common.Address) (*big.Int, error) {
	candidate, err := snap.candidateTrie.TryGet(candidateAddr.Bytes())
	if err != nil || candidate == nil {
		return nil, errors.New("no candidate")
	}

	votes := big.NewInt(0)
	delegateIterator := trie.NewIterator(snap.delegateTrie.PrefixIterator(candidate))
	for delegateIterator.Next() {
		delegator := delegateIterator.Value
		delegatorAddr := common.BytesToAddress(delegator)
		weight := state.GetBalance(delegatorAddr)
		votes.Add(votes, weight)
	}
	return votes, nil
}

// EnoughCandidates count of candidates is greater than or equal to n.
func (snap *Snapshot) EnoughCandidates(n int) (int, bool) {
	candidateCount := 0
	if n <= 0 {
		return 0, true
	}

	iterCandidate := trie.NewIterator(snap.candidateTrie.NodeIterator(nil))
	if iterCandidate.Next() {
		candidateCount++
		if candidateCount >= n {
			return candidateCount, true
		}
	}
	return candidateCount, false
}

// TopCandidates candidates with the top N votes.
func (snap *Snapshot) TopCandidates(state *state.StateDB, n int) (SortableAddresses, error) {
	if n <= 0 {
		return nil, nil
	}

	iterCandidate := trie.NewIterator(snap.candidateTrie.NodeIterator(nil))
	existCandidate := iterCandidate.Next()
	if !existCandidate {
		return nil, nil
	}

	// Count of votes in election
	votes := make(map[common.Address]*big.Int)
	for existCandidate {
		candidate := iterCandidate.Value
		candidateAddr := common.BytesToAddress(candidate)
		delegateIterator := trie.NewIterator(snap.delegateTrie.PrefixIterator(candidate))
		existDelegator := delegateIterator.Next()
		if !existDelegator {
			votes[candidateAddr] = big.NewInt(0)
			existCandidate = iterCandidate.Next()
			continue
		}

		for existDelegator {
			delegator := delegateIterator.Value
			score, ok := votes[candidateAddr]
			if !ok {
				score = big.NewInt(0)
				votes[candidateAddr] = score
			}
			delegatorAddr := common.BytesToAddress(delegator)
			weight := state.GetBalance(delegatorAddr)
			score.Add(score, weight)
			existDelegator = delegateIterator.Next()
		}
		existCandidate = iterCandidate.Next()
	}

	// Sort candidates by votes
	candidates := make(SortableAddresses, 0, n)
	for candidate, cnt := range votes {
		candidates = append(candidates, SortableAddress{candidate, cnt})
	}
	sort.Sort(candidates)
	if len(candidates) > n {
		candidates = candidates[:n]
	}
	return candidates, nil
}

// BecomeCandidate add a new candidate.
func (snap *Snapshot) BecomeCandidate(candidateAddr common.Address) error {
	candidate := candidateAddr.Bytes()
	return snap.candidateTrie.TryUpdate(candidate, candidate)
}

// KickOutCandidate kick out existing candidate.
func (snap *Snapshot) KickOutCandidate(candidateAddr common.Address) error {
	candidate := candidateAddr.Bytes()
	err := snap.candidateTrie.TryDelete(candidate)
	if err != nil {
		if _, ok := err.(*trie.MissingNodeError); !ok {
			return err
		}
	}
	iter := trie.NewIterator(snap.delegateTrie.PrefixIterator(candidate))
	for iter.Next() {
		delegator := iter.Value
		key := append(candidate, delegator...)
		err = snap.delegateTrie.TryDelete(key)
		if err != nil {
			if _, ok := err.(*trie.MissingNodeError); !ok {
				return err
			}
		}
		v, err := snap.voteTrie.TryGet(delegator)
		if err != nil {
			if _, ok := err.(*trie.MissingNodeError); !ok {
				return err
			}
		}
		if err == nil && bytes.Equal(v, candidate) {
			err = snap.voteTrie.TryDelete(delegator)
			if err != nil {
				if _, ok := err.(*trie.MissingNodeError); !ok {
					return err
				}
			}
		}
	}
	return nil
}

// Delegate vote for a candidate, the candidateAddr must be candidate.
func (snap *Snapshot) Delegate(delegatorAddr, candidateAddr common.Address) error {
	delegator, candidate := delegatorAddr.Bytes(), candidateAddr.Bytes()

	candidateInTrie, err := snap.candidateTrie.TryGet(candidate)
	if err != nil {
		return err
	}
	if candidateInTrie == nil {
		return errors.New("invalid candidate to delegate")
	}

	oldCandidate, err := snap.voteTrie.TryGet(delegator)
	if err != nil {
		if _, ok := err.(*trie.MissingNodeError); !ok {
			return err
		}
	}
	if oldCandidate != nil {
		snap.delegateTrie.Delete(append(oldCandidate, delegator...))
	}
	if err = snap.delegateTrie.TryUpdate(append(candidate, delegator...), delegator); err != nil {
		return err
	}
	return snap.voteTrie.TryUpdate(delegator, candidate)
}

// UnDelegate cancel vote for a candidate, the candidateAddr must be candidate.
func (snap *Snapshot) UnDelegate(delegatorAddr, candidateAddr common.Address) error {
	delegator, candidate := delegatorAddr.Bytes(), candidateAddr.Bytes()

	candidateInTrie, err := snap.candidateTrie.TryGet(candidate)
	if err != nil {
		return err
	}
	if candidateInTrie == nil {
		return errors.New("invalid candidate to undelegate")
	}

	oldCandidate, err := snap.voteTrie.TryGet(delegator)
	if err != nil {
		return err
	}
	if !bytes.Equal(candidate, oldCandidate) {
		return errors.New("mismatch candidate to undelegate")
	}

	if err = snap.delegateTrie.TryDelete(append(candidate, delegator...)); err != nil {
		return err
	}
	return snap.voteTrie.TryDelete(delegator)
}
