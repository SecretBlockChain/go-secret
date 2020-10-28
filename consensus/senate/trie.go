package senate

import (
	"bytes"
	"fmt"

	"github.com/SecretBlockChain/go-secret/common"
	"github.com/SecretBlockChain/go-secret/log"
	"github.com/SecretBlockChain/go-secret/trie"
)

// PrefixTrie is a Merkle Patricia Trie.
type Trie struct {
	prefix []byte
	trie   *trie.Trie
}

// New creates a trie with an existing root node from db.
func NewTrieWithPrefix(root common.Hash, prefix []byte, db *trie.Database) (*Trie, error) {
	trie, err := trie.New(root, db)
	if err != nil {
		return nil, err
	}
	return &Trie{prefix: prefix, trie: trie}, nil
}

// Hash returns the root hash of the trie. It does not write to the
// database and can be used even if the trie doesn't have one.
func (t *Trie) Hash() common.Hash {
	return t.trie.Hash()
}

// NodeIterator returns an iterator that returns nodes of the trie. Iteration starts at
// the key after the given start key.
func (t *Trie) NodeIterator(start []byte) trie.NodeIterator {
	if t.prefix != nil {
		start = append(t.prefix, start...)
	}
	return t.trie.NodeIterator(start)
}

// PrefixIterator returns an iterator that returns nodes of the trie which has the prefix path specificed
// Iteration starts at the key after the given start key.
func (t *Trie) PrefixIterator(prefix []byte) trie.NodeIterator {
	return newPrefixIterator(t, prefix)
}

// Get returns the value for key stored in the trie.
// The value bytes must not be modified by the caller.
func (t *Trie) Get(key []byte) []byte {
	res, err := t.TryGet(key)
	if err != nil {
		log.Error(fmt.Sprintf("[DPOS] Unhandled trie error: %v", err))
	}
	return res
}

// TryGet returns the value for key stored in the trie.
// The value bytes must not be modified by the caller.
// If a node was not found in the database, a MissingNodeError is returned.
func (t *Trie) TryGet(key []byte) ([]byte, error) {
	if t.prefix != nil {
		key = append(t.prefix, key...)
	}
	return t.trie.TryGet(key)
}

// Update associates key with value in the trie. Subsequent calls to
// Get will return value. If value has length zero, any existing value
// is deleted from the trie and calls to Get will return nil.
//
// The value bytes must not be modified by the caller while they are
// stored in the trie.
func (t *Trie) Update(key, value []byte) {
	if err := t.TryUpdate(key, value); err != nil {
		log.Error(fmt.Sprintf("[DPOS] Unhandled trie error: %v", err))
	}
}

// TryUpdate associates key with value in the trie. Subsequent calls to
// Get will return value. If value has length zero, any existing value
// is deleted from the trie and calls to Get will return nil.
//
// The value bytes must not be modified by the caller while they are
// stored in the trie.
//
// If a node was not found in the database, a MissingNodeError is returned.
func (t *Trie) TryUpdate(key, value []byte) error {
	if t.prefix != nil {
		key = append(t.prefix, key...)
	}
	return t.trie.TryUpdate(key, value)
}

// Delete removes any existing value for key from the trie.
func (t *Trie) Delete(key []byte) {
	if err := t.TryDelete(key); err != nil {
		log.Error(fmt.Sprintf("[DPOS] Unhandled trie error: %v", err))
	}
}

// TryDelete removes any existing value for key from the trie.
// If a node was not found in the database, a MissingNodeError is returned.
func (t *Trie) TryDelete(key []byte) error {
	if t.prefix != nil {
		key = append(t.prefix, key...)
	}
	return t.trie.TryDelete(key)
}

// Commit writes all nodes to the trie's database.
// Nodes are stored with their sha3 hash as the key.
////
//// Committing flushes nodes from memory.
//// Subsequent Get calls will load nodes from the database.
func (t *Trie) Commit(onleaf trie.LeafCallback) (root common.Hash, err error) {
	return t.trie.Commit(onleaf)
}

type prefixIterator struct {
	prefix       []byte
	nodeIterator trie.NodeIterator
}

func keybytesToHex(str []byte) []byte {
	l := len(str)*2 + 1
	var nibbles = make([]byte, l)
	for i, b := range str {
		nibbles[i*2] = b / 16
		nibbles[i*2+1] = b % 16
	}
	nibbles[l-1] = 16
	return nibbles
}

// newPrefixIterator constructs a NodeIterator, iterates over elements in trie that
// has common prefix.
func newPrefixIterator(trie *Trie, prefix []byte) trie.NodeIterator {
	emptyState := common.Hash{}
	if trie.Hash() == emptyState {
		return new(prefixIterator)
	}

	nodeIt := trie.NodeIterator(prefix)
	if prefix != nil {
		prefix = append(trie.prefix, prefix...)
	}
	prefix = keybytesToHex(prefix)
	return &prefixIterator{
		nodeIterator: nodeIt,
		prefix:       prefix[:len(prefix)-1],
	}
}

// hasPrefix return whether the nodeIterator has common prefix.
func (it *prefixIterator) hasPrefix() bool {
	return bytes.HasPrefix(it.nodeIterator.Path(), it.prefix)
}

func (it *prefixIterator) Hash() common.Hash {
	if it.hasPrefix() {
		return it.nodeIterator.Hash()
	}
	return common.Hash{}
}

func (it *prefixIterator) Parent() common.Hash {
	if it.hasPrefix() {
		it.nodeIterator.Parent()
	}
	return common.Hash{}
}

func (it *prefixIterator) Leaf() bool {
	if it.hasPrefix() {
		return it.nodeIterator.Leaf()
	}
	return false
}

func (it *prefixIterator) LeafBlob() []byte {
	if it.hasPrefix() {
		return it.nodeIterator.LeafBlob()
	}
	return nil
}

func (it *prefixIterator) LeafProof() [][]byte {
	if it.hasPrefix() {
		return it.nodeIterator.LeafProof()
	}
	return nil
}

func (it *prefixIterator) LeafKey() []byte {
	if it.hasPrefix() {
		return it.nodeIterator.LeafKey()
	}
	return nil
}

func (it *prefixIterator) Path() []byte {
	if it.hasPrefix() {
		return it.nodeIterator.Path()
	}
	return nil
}

func (it *prefixIterator) Next(descend bool) bool {
	if it.nodeIterator.Next(descend) {
		if it.hasPrefix() {
			return true
		}
	}
	return false
}

func (it *prefixIterator) Error() error {
	return it.nodeIterator.Error()
}
