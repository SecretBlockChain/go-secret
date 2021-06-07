package senate

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"sort"

	"github.com/SecretBlockChain/go-secret/common"
	"github.com/SecretBlockChain/go-secret/core/types"
	"github.com/SecretBlockChain/go-secret/ethdb"
	"github.com/SecretBlockChain/go-secret/params"
	"github.com/SecretBlockChain/go-secret/rlp"
	"github.com/SecretBlockChain/go-secret/trie"
)

var (
	epochPrefix = []byte("epoch-") // epoch-validator:{validators}
	//delegatePrefix  = []byte("delegate-")  // delegate-{candidateAddr}..{delegatorAddr}:{delegatorAddr}
	//votePrefix      = []byte("vote-")      // vote-{delegatorAddr}:{candidateAddr}
	candidatePrefix = []byte("candidate-") // candidate-{candidateAddr}:
	mintCntPrefix   = []byte("mintCnt-")   // mintCnt-{epoch}..{validator}:{count}
	configPrefix    = []byte("config")     // config:{params.SenateConfig}
	//proposalPrefix  = []byte("proposal-")  // proposal-{hash}:{Proposal}
	declarePrefix = []byte("declare-") // declare-{hash}-{epoch}-{declarer}:{Declare}
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
	for _, candidate := range headerExtra.CurrentBlockKickOutCandidates {
		if err := snap.KickOutCandidate(candidate); err != nil {
			return err
		}
	}
	for _, declare := range headerExtra.CurrentBlockDeclares {
		if err := snap.Declare(headerExtra.Epoch, declare); err != nil {
			return err
		}
	}
	if header.Number.Uint64() == headerExtra.EpochBlock {
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
	for i := len(candidates) - 1; i > 0; i-- {
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
	return nil
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
