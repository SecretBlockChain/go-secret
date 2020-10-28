package senate

import (
	"bytes"
	"testing"

	"github.com/SecretBlockChain/go-secret/common"
	"github.com/SecretBlockChain/go-secret/ethdb/memorydb"
	"github.com/SecretBlockChain/go-secret/trie"
	"github.com/stretchr/testify/assert"
)

func TestUpdate(t *testing.T) {
	prefix := []byte("prefix")
	db := trie.NewDatabase(memorydb.New())
	trieWithPrefix, _ := NewTrieWithPrefix(common.Hash{}, prefix, db)

	err := trieWithPrefix.TryUpdate([]byte("120099"), []byte("zxcvzxcvzxcvzxcvzxcvzxcvzxcvzxcv"))
	assert.Nil(t, err)

	root, err := trieWithPrefix.Commit(nil)
	assert.Nil(t, err)

	trieWithPrefix, _ = NewTrieWithPrefix(root, prefix, db)
	data, err := trieWithPrefix.TryGet([]byte("120099"))
	assert.Nil(t, err)
	assert.True(t, bytes.Equal(data, []byte("zxcvzxcvzxcvzxcvzxcvzxcvzxcvzxcv")))
}

func TestDelete(t *testing.T) {
	prefix := []byte("prefix")
	db := trie.NewDatabase(memorydb.New())
	trieWithPrefix, _ := NewTrieWithPrefix(common.Hash{}, prefix, db)

	err := trieWithPrefix.TryUpdate([]byte("120099"), []byte("zxcvzxcvzxcvzxcvzxcvzxcvzxcvzxcv"))
	assert.Nil(t, err)

	data, err := trieWithPrefix.TryGet([]byte("120099"))
	assert.Nil(t, err)
	assert.True(t, bytes.Equal(data, []byte("zxcvzxcvzxcvzxcvzxcvzxcvzxcvzxcv")))

	err = trieWithPrefix.TryDelete([]byte("120099"))
	assert.Nil(t, err)

	data, err = trieWithPrefix.TryGet([]byte("120099"))
	assert.Nil(t, err)
	assert.True(t, bytes.Equal(data, nil))
}

func TestIterator(t *testing.T) {
	prefix := []byte("prefix")
	db := trie.NewDatabase(memorydb.New())
	trieWithPrefix, _ := NewTrieWithPrefix(common.Hash{}, prefix, db)

	trieWithPrefix.TryUpdate([]byte("111"), []byte("1"))
	trieWithPrefix.TryUpdate([]byte("122"), []byte("2"))
	trieWithPrefix.TryUpdate([]byte("123"), []byte("3"))
	trieWithPrefix.TryUpdate([]byte("1234"), []byte("4"))
	trieWithPrefix.TryUpdate([]byte("12345"), []byte("5"))

	count := 0
	iter := trie.NewIterator(trieWithPrefix.NodeIterator([]byte("11")))
	for iter.Next() {
		count++
	}
	assert.True(t, count == 5)
}

func TestPrefixIterator(t *testing.T) {
	prefix := []byte("prefix")
	db := trie.NewDatabase(memorydb.New())
	trieWithPrefix, _ := NewTrieWithPrefix(common.Hash{}, prefix, db)

	trieWithPrefix.TryUpdate([]byte("111"), []byte("1"))
	trieWithPrefix.TryUpdate([]byte("122"), []byte("2"))
	trieWithPrefix.TryUpdate([]byte("123"), []byte("3"))
	trieWithPrefix.TryUpdate([]byte("1734"), []byte("4"))
	trieWithPrefix.TryUpdate([]byte("12345"), []byte("5"))

	count := 0
	iter := trie.NewIterator(trieWithPrefix.PrefixIterator([]byte("11")))
	for iter.Next() {
		count++
	}
	assert.True(t, count == 1)

	count = 0
	iter = trie.NewIterator(trieWithPrefix.PrefixIterator([]byte("12")))
	for iter.Next() {
		count++
	}
	assert.True(t, count == 3)

	count = 0
	iter = trie.NewIterator(trieWithPrefix.PrefixIterator([]byte("123")))
	for iter.Next() {
		count++
	}
	assert.True(t, count == 2)
}
