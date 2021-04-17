package senate

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"math/rand"
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
	configPrefix    = []byte("config")     // config:{params.SenateConfig}
	proposalPrefix  = []byte("proposal-")  // proposal-{hash}:{Proposal}
	declarePrefix   = []byte("declare-")   // declare-{hash}-{epoch}-{declarer}:{Declare}
)

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
	proposalTrie  *Trie
	declareTrie   *Trie
	db            *trie.Database
}

// newSnapshot creates a new empty snapshot
// only ever use if for the genesis block.
func newSnapshot(diskdb ethdb.Database) (*Snapshot, error) {
	snap := Snapshot{
		db: trie.NewDatabase(diskdb),
	}
	return &snap, nil
}

// loadSnapshot loads an existing snapshot from the database.
func loadSnapshot(diskdb ethdb.Database, root Root) (*Snapshot, error) {
	snap := Snapshot{
		root: root,
		db:   trie.NewDatabase(diskdb),
	}
	return &snap, nil
}

// ensureTrie ensure the trie has been created, trie is not nil
// the purpose is to create tire as needed.
func (snap *Snapshot) ensureTrie(prefix []byte) (*Trie, error) {
	var err error
	switch string(prefix) {
	case string(epochPrefix):
		if snap.epochTrie != nil {
			return snap.epochTrie, nil
		}
		snap.epochTrie, err = NewTrieWithPrefix(snap.root.EpochHash, prefix, snap.db)
		return snap.epochTrie, err
	case string(delegatePrefix):
		if snap.delegateTrie != nil {
			return snap.delegateTrie, nil
		}
		snap.delegateTrie, err = NewTrieWithPrefix(snap.root.DelegateHash, prefix, snap.db)
		return snap.delegateTrie, err
	case string(votePrefix):
		if snap.voteTrie != nil {
			return snap.voteTrie, nil
		}
		snap.voteTrie, err = NewTrieWithPrefix(snap.root.VoteHash, prefix, snap.db)
		return snap.voteTrie, err
	case string(candidatePrefix):
		if snap.candidateTrie != nil {
			return snap.candidateTrie, nil
		}
		snap.candidateTrie, err = NewTrieWithPrefix(snap.root.CandidateHash, prefix, snap.db)
		return snap.candidateTrie, err
	case string(mintCntPrefix):
		if snap.mintCntTrie != nil {
			return snap.mintCntTrie, nil
		}
		snap.mintCntTrie, err = NewTrieWithPrefix(snap.root.MintCntHash, prefix, snap.db)
		return snap.mintCntTrie, err
	case string(configPrefix):
		if snap.configTrie != nil {
			return snap.configTrie, nil
		}
		snap.configTrie, err = NewTrieWithPrefix(snap.root.ConfigHash, prefix, snap.db)
		return snap.configTrie, err
	case string(proposalPrefix):
		if snap.proposalTrie != nil {
			return snap.proposalTrie, nil
		}
		snap.proposalTrie, err = NewTrieWithPrefix(snap.root.ProposalHash, prefix, snap.db)
		return snap.proposalTrie, err
	case string(declarePrefix):
		if snap.declareTrie != nil {
			return snap.declareTrie, nil
		}
		snap.declareTrie, err = NewTrieWithPrefix(snap.root.DeclareHash, prefix, snap.db)
		return snap.declareTrie, err
	default:
		return nil, errors.New("unknown prefix")
	}
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
	for _, proposal := range headerExtra.CurrentBlockProposals {
		if err := snap.SubmitProposal(proposal); err != nil {
			return err
		}
	}
	for _, declare := range headerExtra.CurrentBlockDeclares {
		if err := snap.Declare(headerExtra.Epoch, declare); err != nil {
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
	root = snap.root
	if snap.epochTrie != nil {
		root.EpochHash, err = snap.epochTrie.Commit(nil)
		if err != nil {
			return Root{}, err
		}
	}

	if snap.delegateTrie != nil {
		root.DelegateHash, err = snap.delegateTrie.Commit(nil)
		if err != nil {
			return Root{}, err
		}
	}

	if snap.voteTrie != nil {
		root.VoteHash, err = snap.voteTrie.Commit(nil)
		if err != nil {
			return Root{}, err
		}
	}

	if snap.candidateTrie != nil {
		root.CandidateHash, err = snap.candidateTrie.Commit(nil)
		if err != nil {
			return Root{}, err
		}
	}

	if snap.mintCntTrie != nil {
		root.MintCntHash, err = snap.mintCntTrie.Commit(nil)
		if err != nil {
			return Root{}, err
		}
	}

	if snap.configTrie != nil {
		root.ConfigHash, err = snap.configTrie.Commit(nil)
		if err != nil {
			return Root{}, err
		}
	}

	if snap.proposalTrie != nil {
		root.ProposalHash, err = snap.proposalTrie.Commit(nil)
		if err != nil {
			return Root{}, err
		}
	}

	if snap.declareTrie != nil {
		root.DeclareHash, err = snap.declareTrie.Commit(nil)
		if err != nil {
			return Root{}, err
		}
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
	if snap.root.ProposalHash != root.ProposalHash {
		if err := snap.db.Commit(root.ProposalHash, false, nil); err != nil {
			return err
		}
	}
	if snap.root.DeclareHash != root.DeclareHash {
		if err := snap.db.Commit(root.DeclareHash, false, nil); err != nil {
			return err
		}
	}
	snap.root = root
	return nil
}

// GetChainConfig returns chain config from snapshot.
func (snap *Snapshot) GetChainConfig() (params.SenateConfig, error) {
	configTrie, err := snap.ensureTrie(configPrefix)
	if err != nil {
		return params.SenateConfig{}, err
	}

	key := []byte("config")
	data := configTrie.Get(key)
	var config params.SenateConfig
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

	configTrie, err := snap.ensureTrie(configPrefix)
	if err != nil {
		return err
	}

	key := []byte("config")
	data, err := json.Marshal(config)
	if err != nil {
		return err
	}
	return configTrie.TryUpdate(key, data)
}

// GetValidators returns validators of current epoch.
func (snap *Snapshot) GetValidators() (SortableAddresses, error) {
	epochTrie, err := snap.ensureTrie(epochPrefix)
	if err != nil {
		return nil, err
	}

	key := []byte("validator")
	var validators SortableAddresses
	validatorsRLP := epochTrie.Get(key)
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

	epochTrie, err := snap.ensureTrie(epochPrefix)
	if err != nil {
		return err
	}
	return epochTrie.TryUpdate(key, validatorsRLP)
}

// CountMinted count the minted of each validator.
func (snap *Snapshot) CountMinted(epoch uint64) (SortableAddresses, error) {
	validators, err := snap.GetValidators()
	if err != nil {
		return nil, err
	}

	mintCntTrie, err := snap.ensureTrie(mintCntPrefix)
	if err != nil {
		return nil, err
	}

	prefix := make([]byte, 8)
	binary.BigEndian.PutUint64(prefix, epoch)
	iter := trie.NewIterator(mintCntTrie.PrefixIterator(prefix))

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
	mintCntTrie, err := snap.ensureTrie(mintCntPrefix)
	if err != nil {
		return err
	}

	key := make([]byte, 16)
	binary.BigEndian.PutUint64(key[:8], epoch)
	binary.BigEndian.PutUint64(key[8:], number)
	return mintCntTrie.TryUpdate(key, validator.Bytes())
}

// CountVotes count the votes of candidate.
func (snap *Snapshot) CountVotes(state *state.StateDB, candidateAddr common.Address) (*big.Int, error) {
	delegateTrie, err := snap.ensureTrie(delegatePrefix)
	if err != nil {
		return nil, err
	}

	candidateTrie, err := snap.ensureTrie(candidatePrefix)
	if err != nil {
		return nil, err
	}

	candidate, err := candidateTrie.TryGet(candidateAddr.Bytes())
	if err != nil || candidate == nil {
		return nil, errors.New("no candidate")
	}

	votes := big.NewInt(0)
	delegateIterator := trie.NewIterator(delegateTrie.PrefixIterator(candidate))
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

	candidateTrie, err := snap.ensureTrie(candidatePrefix)
	if err != nil {
		return 0, false
	}

	iterCandidate := trie.NewIterator(candidateTrie.NodeIterator(nil))
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

	delegateTrie, err := snap.ensureTrie(delegatePrefix)
	if err != nil {
		return nil, err
	}

	candidateTrie, err := snap.ensureTrie(candidatePrefix)
	if err != nil {
		return nil, err
	}

	iterCandidate := trie.NewIterator(candidateTrie.NodeIterator(nil))
	existCandidate := iterCandidate.Next()
	if !existCandidate {
		return nil, nil
	}

	// Count of votes in election
	votes := make(map[common.Address]*big.Int)
	for existCandidate {
		candidate := iterCandidate.Value
		candidateAddr := common.BytesToAddress(candidate)
		delegateIterator := trie.NewIterator(delegateTrie.PrefixIterator(candidate))
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
// RandCandidates random return n candidates.
func (snap *Snapshot) RandCandidates(seed int64, n int) (SortableAddresses, error) {
	if n <= 0 {
		return nil, nil
	}

	candidateTrie, err := snap.ensureTrie(candidatePrefix)
	if err != nil {
		return nil, err
	}

	iterCandidate := trie.NewIterator(candidateTrie.NodeIterator(nil))
	existCandidate := iterCandidate.Next()
	if !existCandidate {
		return nil, nil
	}

	// All candidate
	candidates := make(SortableAddresses, 0)
	for existCandidate {
		candidate := iterCandidate.Value
		candidateAddr := common.BytesToAddress(candidate)
		candidates = append(candidates, SortableAddress{candidateAddr, big.NewInt(0)})
		existCandidate = iterCandidate.Next()
	}

	// Shuffle candidates
	r := rand.New(rand.NewSource(seed))
	for i := len(candidates) -1 ; i >0; i-- {
		j := int(r.Int31n(int32(i + 1)))
		candidates[i], candidates[j] = candidates[j], candidates[i]
	}
	if len(candidates) > n {
		candidates = candidates[:n]
	}
	return candidates, nil
}

// BecomeCandidate add a new candidate.
func (snap *Snapshot) BecomeCandidate(candidateAddr common.Address) error {
	candidateTrie, err := snap.ensureTrie(candidatePrefix)
	if err != nil {
		return err
	}
	candidate := candidateAddr.Bytes()
	return candidateTrie.TryUpdate(candidate, candidate)
}

// KickOutCandidate kick out existing candidate.
func (snap *Snapshot) KickOutCandidate(candidateAddr common.Address) error {
	voteTrie, err := snap.ensureTrie(votePrefix)
	if err != nil {
		return err
	}

	delegateTrie, err := snap.ensureTrie(delegatePrefix)
	if err != nil {
		return err
	}

	candidateTrie, err := snap.ensureTrie(candidatePrefix)
	if err != nil {
		return err
	}

	candidate := candidateAddr.Bytes()
	err = candidateTrie.TryDelete(candidate)
	if err != nil {
		if _, ok := err.(*trie.MissingNodeError); !ok {
			return err
		}
	}
	iter := trie.NewIterator(delegateTrie.PrefixIterator(candidate))
	for iter.Next() {
		delegator := iter.Value
		key := append(candidate, delegator...)
		err = delegateTrie.TryDelete(key)
		if err != nil {
			if _, ok := err.(*trie.MissingNodeError); !ok {
				return err
			}
		}
		v, err := voteTrie.TryGet(delegator)
		if err != nil {
			if _, ok := err.(*trie.MissingNodeError); !ok {
				return err
			}
		}
		if err == nil && bytes.Equal(v, candidate) {
			err = voteTrie.TryDelete(delegator)
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
	voteTrie, err := snap.ensureTrie(votePrefix)
	if err != nil {
		return err
	}

	delegateTrie, err := snap.ensureTrie(delegatePrefix)
	if err != nil {
		return err
	}

	candidateTrie, err := snap.ensureTrie(candidatePrefix)
	if err != nil {
		return err
	}

	delegator, candidate := delegatorAddr.Bytes(), candidateAddr.Bytes()
	candidateInTrie, err := candidateTrie.TryGet(candidate)
	if err != nil {
		return err
	}
	if candidateInTrie == nil {
		return errors.New("invalid candidate to delegate")
	}

	oldCandidate, err := voteTrie.TryGet(delegator)
	if err != nil {
		if _, ok := err.(*trie.MissingNodeError); !ok {
			return err
		}
	}
	if oldCandidate != nil {
		delegateTrie.Delete(append(oldCandidate, delegator...))
	}
	if err = delegateTrie.TryUpdate(append(candidate, delegator...), delegator); err != nil {
		return err
	}
	return voteTrie.TryUpdate(delegator, candidate)
}

// UnDelegate cancel vote for a candidate, the candidateAddr must be candidate.
func (snap *Snapshot) UnDelegate(delegatorAddr, candidateAddr common.Address) error {
	voteTrie, err := snap.ensureTrie(votePrefix)
	if err != nil {
		return err
	}

	delegateTrie, err := snap.ensureTrie(delegatePrefix)
	if err != nil {
		return err
	}

	candidateTrie, err := snap.ensureTrie(candidatePrefix)
	if err != nil {
		return err
	}

	delegator, candidate := delegatorAddr.Bytes(), candidateAddr.Bytes()
	candidateInTrie, err := candidateTrie.TryGet(candidate)
	if err != nil {
		return err
	}
	if candidateInTrie == nil {
		return errors.New("invalid candidate to undelegate")
	}

	oldCandidate, err := voteTrie.TryGet(delegator)
	if err != nil {
		return err
	}
	if !bytes.Equal(candidate, oldCandidate) {
		return errors.New("mismatch candidate to undelegate")
	}

	if err = delegateTrie.TryDelete(append(candidate, delegator...)); err != nil {
		return err
	}
	return voteTrie.TryDelete(delegator)
}

// Declare declare the decision on the proposal.
func (snap *Snapshot) Declare(epoch uint64, declare Declare) error {
	declareTrie, err := snap.ensureTrie(declarePrefix)
	if err != nil {
		return err
	}

	hash := declare.ProposalHash.Bytes()
	declarer := declare.Declarer.Bytes()
	key := make([]byte, len(hash)+8+len(declarer))
	copy(key, hash)
	binary.BigEndian.PutUint64(key[len(hash):len(hash)+8], epoch)
	copy(key[len(hash)+8:], declarer)

	jsb, err := json.Marshal(declare)
	if err != nil {
		return err
	}
	return declareTrie.TryUpdate(key, jsb)
}

// GetDeclarations returns all declarations in the epoch.
func (snap *Snapshot) GetDeclarations(proposalHash common.Hash, epoch uint64) ([]Declare, error) {
	declareTrie, err := snap.ensureTrie(declarePrefix)
	if err != nil {
		return nil, err
	}

	hash := proposalHash.Bytes()
	prefix := make([]byte, len(hash)+8)
	copy(prefix, hash)
	binary.BigEndian.PutUint64(prefix[len(hash):len(hash)+8], epoch)

	var declarations []Declare
	iter := trie.NewIterator(declareTrie.PrefixIterator(prefix))
	for iter.Next() {
		var declare Declare
		if err = json.Unmarshal(iter.Value, &declare); err != nil {
			continue
		}
		declarations = append(declarations, declare)
	}
	return declarations, nil
}

// GetProposal returns the specified proposal
// the hash is transaction hash of proposal.
func (snap *Snapshot) GetProposal(hash common.Hash) (Proposal, error) {
	proposalTrie, err := snap.ensureTrie(proposalPrefix)
	if err != nil {
		return Proposal{}, err
	}

	data := proposalTrie.Get(hash.Bytes())
	if data == nil {
		return Proposal{}, errors.New("proposal not found, hash: " + hash.String())
	}

	var proposal Proposal
	if err = json.Unmarshal(data, &proposal); err != nil {
		return Proposal{}, err
	}
	return proposal, nil
}

// SubmitProposal submit a new proposal.
func (snap *Snapshot) SubmitProposal(proposal Proposal) error {
	proposalTrie, err := snap.ensureTrie(proposalPrefix)
	if err != nil {
		return err
	}

	value, err := json.Marshal(proposal)
	if err != nil {
		return err
	}
	return proposalTrie.TryUpdate(proposal.Hash.Bytes(), value)
}

// ApproveProposal approve the proposal
// the hash is transaction hash of proposal, txHash is transaction hash of declare.
func (snap *Snapshot) ApproveProposal(hash, txHash common.Hash) (Proposal, error) {
	proposal, err := snap.GetProposal(hash)
	if err != nil {
		return Proposal{}, err
	}

	proposal.ApprovedHash = &txHash
	if err = snap.SubmitProposal(proposal); err != nil {
		return Proposal{}, err
	}
	return proposal, nil
}
