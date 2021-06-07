package equality

import (
	"testing"

	"github.com/SecretBlockChain/go-secret/accounts"
	"github.com/SecretBlockChain/go-secret/core/types"
	"github.com/SecretBlockChain/go-secret/crypto"
	lru "github.com/hashicorp/golang-lru"
	"github.com/stretchr/testify/assert"
)

func TestSealHash(t *testing.T) {
	header := types.Header{
		Extra: make([]byte, extraSeal),
	}
	signFn := func(account accounts.Account, s string, data []byte) ([]byte, error) {
		return crypto.Sign(crypto.Keccak256(data), testUserKey)
	}
	sigHash, err := signFn(accounts.Account{Address: testUserAddress}, accounts.MimetypeClique, EqualityRLP(&header))
	assert.Nil(t, err)
	copy(header.Extra, sigHash)

	signatures, _ := lru.NewARC(inMemorySignatures)
	signer, err := ecrecover(&header, signatures)
	assert.Nil(t, err)
	assert.Equal(t, signer.String(), testUserAddress.String())
}
