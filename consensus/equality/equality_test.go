package equality

import (
	"math/big"
	"testing"
	"time"

	"github.com/SecretBlockChain/go-secret/accounts"
	"github.com/SecretBlockChain/go-secret/common"
	"github.com/SecretBlockChain/go-secret/core/rawdb"
	"github.com/SecretBlockChain/go-secret/crypto"
	"github.com/SecretBlockChain/go-secret/params"
)

var (
	// Test accounts
	testUserKey, _  = crypto.GenerateKey()
	testUserAddress = crypto.PubkeyToAddress(testUserKey.PublicKey)
)

func TestNewEquality(t *testing.T) {
	db := rawdb.NewMemoryDatabase()
	config := params.EqualityConfig{
		Period:              60,
		Epoch:               180,
		MaxValidatorsCount:  3,
		MinCandidateBalance: big.NewInt(0),
		GenesisTimestamp:    uint64(time.Now().Unix()),
		Validators:          []common.Address{testUserAddress},
		Rewards: []params.EqualityReward{
			{Number: 100000, Reward: big.NewInt(1)},
		},
	}
	equality := New(&config, db)
	equality.Authorize(testUserAddress, func(account accounts.Account, s string, data []byte) ([]byte, error) {
		return crypto.Sign(crypto.Keccak256(data), testUserKey)
	})
}
