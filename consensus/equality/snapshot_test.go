package equality

import (
	"math/big"
	"testing"

	"github.com/SecretBlockChain/go-secret/common"
	"github.com/SecretBlockChain/go-secret/core/rawdb"
	"github.com/SecretBlockChain/go-secret/core/state"
	"github.com/SecretBlockChain/go-secret/params"
	"github.com/stretchr/testify/assert"
)

func TestSetChainConfig(t *testing.T) {
	db := rawdb.NewMemoryDatabase()
	snap, err := loadSnapshot(db, Root{})
	assert.Nil(t, err)

	config := params.EqualityConfig{
		Period:              1024,
		MaxValidatorsCount:  21,
		MinCandidateBalance: big.NewInt(1000),
	}
	assert.Nil(t, snap.SetChainConfig(config))

	result, err := snap.GetChainConfig()
	assert.Nil(t, err)
	assert.Equal(t, config, result)
}

func TestLoadSnapshot(t *testing.T) {
	db := rawdb.NewMemoryDatabase()
	snap, err := loadSnapshot(db, Root{})
	assert.Nil(t, err)

	validator1 := common.HexToAddress("0x44d1ce0b7cb3588bca96151fe1bc05af38f91b6c")
	snap.SetValidators([]common.Address{validator1})

	root, err := snap.Root()
	assert.Nil(t, err)

	assert.Nil(t, snap.Commit(root))

	_, err = loadSnapshot(db, root)
	assert.Nil(t, err)
}

func TestSetValidators(t *testing.T) {
	db := rawdb.NewMemoryDatabase()
	snap, err := newSnapshot(db)
	assert.Nil(t, err)

	validator1 := common.HexToAddress("0x44d1ce0b7cb3588bca96151fe1bc05af38f91b6c")
	validator2 := common.HexToAddress("0xcc7c8317b21e1cea6139700c3c46c21af998d14c")
	validator3 := common.HexToAddress("0x10702d5b794d97fb720e02506ecfdb1186a804b1")
	validator4 := common.HexToAddress("0x19e28f4ca35205a5060d8375c9fca1a315f4d7b6")
	err = snap.SetValidators([]common.Address{validator1, validator2, validator3, validator4})
	assert.Nil(t, err)

	validators, err := snap.GetValidators()
	assert.Nil(t, err)
	assert.Equal(t, validators[0], validator1)
	assert.Equal(t, validators[1], validator2)
	assert.Equal(t, validators[2], validator3)
	assert.Equal(t, validators[3], validator4)
}
func TestRandCandidates(t *testing.T) {
	db := rawdb.NewMemoryDatabase()
	snap, err := newSnapshot(db)
	assert.Nil(t, err)

	candidate1 := common.HexToAddress("0xcc7c8317b21e1cea6139700c3c46c21af998d14c")
	_, err = snap.BecomeCandidate(candidate1, 1, big.NewInt(0))
	assert.Nil(t, err)

	candidate2 := common.HexToAddress("0x19e28f4ca35205a5060d8375c9fca1a315f4d7b6")
	_, err = snap.BecomeCandidate(candidate2, 1, big.NewInt(0))
	assert.Nil(t, err)

	candidate3 := common.HexToAddress("0x08317854e853facf0bff9e360583d80c1596ed7a")
	_, err = snap.BecomeCandidate(candidate3, 1, big.NewInt(0))
	assert.Nil(t, err)

	candidate4 := common.HexToAddress("0x7bee0c6d5132e39622bdb6c0fc9f16b350f09453")
	_, err = snap.BecomeCandidate(candidate4, 1, big.NewInt(0))
	assert.Nil(t, err)

	candidate5 := common.HexToAddress("0xf541c3cd1d2df407fb9bb52b3489fc2aaeedd97e")
	_, err = snap.BecomeCandidate(candidate5, 1, big.NewInt(0))
	assert.Nil(t, err)

	addresses, err := snap.RandCandidates(100, 3)

	assert.True(t, len(addresses) == 3)
}

func TestKickOutCandidate(t *testing.T) {
	db := rawdb.NewMemoryDatabase()
	snap, err := newSnapshot(db)
	assert.Nil(t, err)

	statedb, err := state.New(common.Hash{}, state.NewDatabase(db), nil)
	assert.Nil(t, err)

	candidate := common.HexToAddress("0xcc7c8317b21e1cea6139700c3c46c21af998d14c")
	delegator := common.HexToAddress("0x44d1ce0b7cb3588bca96151fe1bc05af38f91b6c")
	statedb.AddBalance(delegator, big.NewInt(10000))
	_, err = snap.BecomeCandidate(candidate, 1, big.NewInt(0))
	assert.Nil(t, err)

	candidates, err := snap.RandCandidates(100, 1)
	assert.Nil(t, err)
	assert.True(t, len(candidates) == 1)
	assert.Equal(t, candidates[0], candidate)

	_, _, err = snap.CancelCandidate(candidate)
	assert.Nil(t, err)
	candidates, err = snap.RandCandidates(100, 1)
	assert.Nil(t, err)
	assert.True(t, len(candidates) == 0)
}

func TestCountMinted(t *testing.T) {
	db := rawdb.NewMemoryDatabase()
	snap, err := newSnapshot(db)
	assert.Nil(t, err)

	validator1 := common.HexToAddress("0xcc7c8317b21e1cea6139700c3c46c21af998d14c")
	validator2 := common.HexToAddress("0x44d1ce0b7cb3588bca96151fe1bc05af38f91b6c")
	validator3 := common.HexToAddress("0xf541c3cd1d2df407fb9bb52b3489fc2aaeedd97e")
	assert.Nil(t, snap.SetValidators([]common.Address{validator1, validator2, validator3}))

	assert.Nil(t, snap.MintBlock(1, 1, validator1))
	assert.Nil(t, snap.MintBlock(1, 2, validator1))
	assert.Nil(t, snap.MintBlock(1, 3, validator1))
	assert.Nil(t, snap.MintBlock(1, 4, validator2))
	assert.Nil(t, snap.MintBlock(1, 5, validator2))
	assert.Nil(t, snap.MintBlock(1, 6, validator3))
	assert.Nil(t, snap.MintBlock(1, 7, validator3))
	assert.Nil(t, snap.MintBlock(1, 8, validator3))
	assert.Nil(t, snap.MintBlock(1, 9, validator3))

	result, err := snap.CountMinted(1)
	assert.Nil(t, err)

	assert.Equal(t, result[0].Address, validator2)
	assert.Equal(t, result[0].Weight, big.NewInt(2))

	assert.Equal(t, result[1].Address, validator1)
	assert.Equal(t, result[1].Weight, big.NewInt(3))

	assert.Equal(t, result[2].Address, validator3)
	assert.Equal(t, result[2].Weight, big.NewInt(4))
}
