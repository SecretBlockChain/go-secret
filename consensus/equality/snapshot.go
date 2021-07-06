package equality

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"sort"
	"strings"

	"github.com/SecretBlockChain/go-secret/common"
	"github.com/SecretBlockChain/go-secret/core/types"
	"github.com/SecretBlockChain/go-secret/ethdb"
	"github.com/SecretBlockChain/go-secret/params"
	"github.com/SecretBlockChain/go-secret/rlp"
	"github.com/SecretBlockChain/go-secret/trie"
)

var (
	epochPrefix     = []byte("epoch-")     // key: epoch-validator:{validators}
	candidatePrefix = []byte("candidate-") // key: candidate-{candidateAddr}:{Candidate}
	mintCntPrefix   = []byte("mintCnt-")   // key: mintCnt-{epoch}..{validator}:{count}
	configPrefix    = []byte("config")     // key: config:{params.EqualityConfig}
)

// Candidate basic information
type Candidate struct {
	Security    *big.Int
	BlockNumber uint64
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
func (p SortableAddresses) String() string {
	s := make([]string, 0, len(p))
	for _, addr := range p {
		s = append(s, addr.Address.String())
	}
	return "[" + strings.Join(s, ",") + "]"
}

// Snapshot is the state of the authorization voting at a given block number.
type Snapshot struct {
	root          Root
	epochTrie     *Trie
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
	default:
		return nil, errors.New("unknown prefix")
	}
}

// apply creates a new authorization snapshot by applying the given headers to
// the original one.
func (snap *Snapshot) apply(config params.EqualityConfig, header *types.Header, headerExtra HeaderExtra) error {
	number := header.Number.Uint64()
	for _, candidate := range headerExtra.CurrentBlockCandidates {
		security := big.NewInt(0)
		if number > 1 {
			security = config.MinCandidateBalance
		}
		if _, err := snap.BecomeCandidate(candidate, number, security); err != nil {
			return err
		}
	}

	for _, candidate := range headerExtra.CurrentBlockKickOutCandidates {
		if _, err := snap.CancelCandidate(candidate); err != nil {
			return err
		}
	}

	for _, candidate := range headerExtra.CurrentBlockCancelCandidates {
		if _, err := snap.CancelCandidate(candidate); err != nil {
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
	snap.root = root
	return nil
}

// GetChainConfig returns chain config from snapshot.
func (snap *Snapshot) GetChainConfig() (params.EqualityConfig, error) {
	configTrie, err := snap.ensureTrie(configPrefix)
	if err != nil {
		return params.EqualityConfig{}, err
	}

	key := []byte("config")
	data := configTrie.Get(key)
	var config params.EqualityConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return params.EqualityConfig{}, err
	}
	return config, nil
}

// SetChainConfig write chain config to snapshot.
func (snap *Snapshot) SetChainConfig(config params.EqualityConfig) error {
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
func (snap *Snapshot) GetValidators() ([]common.Address, error) {
	epochTrie, err := snap.ensureTrie(epochPrefix)
	if err != nil {
		return nil, err
	}

	key := []byte("validator")
	var validators []common.Address
	validatorsRLP := epochTrie.Get(key)
	if err := rlp.DecodeBytes(validatorsRLP, &validators); err != nil {
		return nil, fmt.Errorf("failed to decode validators: %s", err)
	}
	return validators, nil
}

// SetValidators write validators of current epoch to snapshot.
func (snap *Snapshot) SetValidators(validators []common.Address) error {
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

	addresses := make(SortableAddresses, 0)
	for idx := range validators {
		count, ok := mapper[validators[idx]]
		if !ok {
			addresses = append(addresses, SortableAddress{Address: validators[idx], Weight: big.NewInt(0)})
		} else {
			addresses = append(addresses, SortableAddress{Address: validators[idx], Weight: big.NewInt(count)})
		}
	}
	sort.Sort(sort.Reverse(addresses))
	return addresses, nil
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

// GetCandidates returns all candidates.
func (snap *Snapshot) GetCandidates() ([]common.Address, error) {
	candidateTrie, err := snap.ensureTrie(candidatePrefix)
	if err != nil {
		return nil, err
	}

	candidates := make([]common.Address, 0)
	iterCandidate := trie.NewIterator(candidateTrie.NodeIterator(nil))
	for iterCandidate.Next() {
		candidates = append(candidates, common.BytesToAddress(iterCandidate.Key))
	}
	return candidates, nil
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
	for iterCandidate.Next() {
		candidateCount++
		if candidateCount >= n {
			return candidateCount, true
		}
	}
	return candidateCount, false
}

// RandCandidates random return n candidates.
func (snap *Snapshot) RandCandidates(seed int64, n int) ([]common.Address, error) {
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
	candidates := make([]common.Address, 0)
	for existCandidate {
		candidates = append(candidates, common.BytesToAddress(iterCandidate.Key))
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

// BecomeCandidate add a new candidate,return a bool value means address already is or not a candidate
func (snap *Snapshot) BecomeCandidate(candidateAddr common.Address, blockNumber uint64, security *big.Int) (bool, error) {
	candidateTrie, err := snap.ensureTrie(candidatePrefix)
	if err != nil {
		return false, err
	}

	key := candidateAddr.Bytes()
	candidateRLP, err := candidateTrie.TryGet(key)
	if err != nil {
		return false, err
	}
	if candidateRLP != nil {
		return true, nil
	}

	candidate := Candidate{
		Security:    security,
		BlockNumber: blockNumber,
	}

	value, err := rlp.EncodeToBytes(candidate)
	if err != nil {
		return false, err
	}
	return false, candidateTrie.TryUpdate(key, value)
}

// CancelCandidate remove a candidate
func (snap *Snapshot) CancelCandidate(candidateAddr common.Address) (*big.Int, error) {
	candidateTrie, err := snap.ensureTrie(candidatePrefix)
	if err != nil {
		return big.NewInt(0), err
	}

	key := candidateAddr.Bytes()

	var candidate Candidate
	candidateRLP := candidateTrie.Get(key)
	if err := rlp.DecodeBytes(candidateRLP, &candidate); err != nil {
		return nil, fmt.Errorf("failed to decode candidate: %s", err)
	}

	err = candidateTrie.TryDelete(key)
	if err != nil {
		if _, ok := err.(*trie.MissingNodeError); !ok {
			return big.NewInt(0), err
		}
	}
	return candidate.Security, nil
}
