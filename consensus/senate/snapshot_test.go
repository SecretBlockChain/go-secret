package senate

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

	config := params.SenateConfig{
		Period:             1024,
		MaxValidatorsCount: 21,
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
	snap.SetValidators(SortableAddresses{
		SortableAddress{Address: validator1, Weight: big.NewInt(0)},
	})

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
	err = snap.SetValidators(SortableAddresses{
		SortableAddress{Address: validator1, Weight: big.NewInt(0)},
		SortableAddress{Address: validator2, Weight: big.NewInt(0)},
		SortableAddress{Address: validator3, Weight: big.NewInt(0)},
		SortableAddress{Address: validator4, Weight: big.NewInt(0)},
	})
	assert.Nil(t, err)

	validators, err := snap.GetValidators()
	assert.Nil(t, err)
	assert.Equal(t, validators[0].Address, validator1)
	assert.Equal(t, validators[1].Address, validator2)
	assert.Equal(t, validators[2].Address, validator3)
	assert.Equal(t, validators[3].Address, validator4)
}

func TestTopCandidates(t *testing.T) {
	db := rawdb.NewMemoryDatabase()
	snap, err := newSnapshot(db)
	assert.Nil(t, err)

	statedb, err := state.New(common.Hash{}, state.NewDatabase(db), nil)
	assert.Nil(t, err)

	delegator1 := common.HexToAddress("0x44d1ce0b7cb3588bca96151fe1bc05af38f91b6c")
	statedb.AddBalance(delegator1, big.NewInt(10000))
	candidate1 := common.HexToAddress("0xcc7c8317b21e1cea6139700c3c46c21af998d14c")
	assert.Nil(t, snap.BecomeCandidate(candidate1))
	assert.Nil(t, snap.Delegate(delegator1, candidate1))

	delegator2 := common.HexToAddress("0x10702d5b794d97fb720e02506ecfdb1186a804b1")
	statedb.AddBalance(delegator2, big.NewInt(52264))
	candidate2 := common.HexToAddress("0x19e28f4ca35205a5060d8375c9fca1a315f4d7b6")
	assert.Nil(t, snap.BecomeCandidate(candidate2))
	assert.Nil(t, snap.Delegate(delegator2, candidate2))

	delegator3 := common.HexToAddress("0xb1706a41a42a129914194261e3fe6a081719ed48")
	statedb.AddBalance(delegator3, big.NewInt(1231231))
	candidate3 := common.HexToAddress("0x08317854e853facf0bff9e360583d80c1596ed7a")
	assert.Nil(t, snap.BecomeCandidate(candidate3))
	assert.Nil(t, snap.Delegate(delegator3, candidate3))

	delegator4 := common.HexToAddress("0x47746e8acb5dafe9c00b7195d0c2d830fcc04910")
	statedb.AddBalance(delegator4, big.NewInt(562))
	candidate4 := common.HexToAddress("0x7bee0c6d5132e39622bdb6c0fc9f16b350f09453")
	assert.Nil(t, snap.BecomeCandidate(candidate4))
	assert.Nil(t, snap.Delegate(delegator4, candidate4))

	delegator5 := common.HexToAddress("0x3c8d2bbc0b9b93f396d4831ca24ea023a0acae5b")
	statedb.AddBalance(delegator5, big.NewInt(5523))
	candidate5 := common.HexToAddress("0xf541c3cd1d2df407fb9bb52b3489fc2aaeedd97e")
	assert.Nil(t, snap.BecomeCandidate(candidate5))
	assert.Nil(t, snap.Delegate(delegator5, candidate5))

	addresses, err := snap.TopCandidates(statedb, 5)
	assert.True(t, len(addresses) == 5)
	assert.Equal(t, addresses[0].Address, candidate3)
	assert.Equal(t, addresses[1].Address, candidate2)
	assert.Equal(t, addresses[2].Address, candidate1)
	assert.Equal(t, addresses[3].Address, candidate5)
	assert.Equal(t, addresses[4].Address, candidate4)

	addresses, err = snap.TopCandidates(statedb, 3)
	assert.True(t, len(addresses) == 3)
	assert.Equal(t, addresses[0].Address, candidate3)
	assert.Equal(t, addresses[1].Address, candidate2)
	assert.Equal(t, addresses[2].Address, candidate1)
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
	assert.Nil(t, snap.BecomeCandidate(candidate))

	candidates, err := snap.TopCandidates(statedb, 1)
	assert.Nil(t, err)
	assert.True(t, len(candidates) == 1)
	assert.Equal(t, candidates[0].Address, candidate)

	assert.Nil(t, snap.KickOutCandidate(candidate))
	candidates, err = snap.TopCandidates(statedb, 1)
	assert.Nil(t, err)
	assert.True(t, len(candidates) == 0)
}

func TestUnDelegate(t *testing.T) {
	db := rawdb.NewMemoryDatabase()
	snap, err := newSnapshot(db)
	assert.Nil(t, err)

	statedb, err := state.New(common.Hash{}, state.NewDatabase(db), nil)
	assert.Nil(t, err)

	candidate := common.HexToAddress("0xcc7c8317b21e1cea6139700c3c46c21af998d14c")
	delegator := common.HexToAddress("0x44d1ce0b7cb3588bca96151fe1bc05af38f91b6c")
	statedb.AddBalance(delegator, big.NewInt(10000))
	assert.Nil(t, snap.BecomeCandidate(candidate))
	assert.Nil(t, snap.Delegate(delegator, candidate))

	votes, err := snap.CountVotes(statedb, candidate)
	assert.Nil(t, err)
	assert.True(t, votes.Cmp(big.NewInt(10000)) == 0)

	candidates, err := snap.TopCandidates(statedb, 1)
	assert.Nil(t, err)
	assert.True(t, len(candidates) == 1)
	assert.Equal(t, candidates[0].Address, candidate)
	assert.True(t, candidates[0].Weight.Cmp(big.NewInt(10000)) == 0)

	assert.Nil(t, snap.UnDelegate(delegator, candidate))

	candidates, err = snap.TopCandidates(statedb, 1)
	assert.Nil(t, err)
	assert.True(t, len(candidates) == 1)
	assert.Equal(t, candidates[0].Address, candidate)
	assert.True(t, candidates[0].Weight.Cmp(big.NewInt(0)) == 0)
}

func TestCountMinted(t *testing.T) {
	db := rawdb.NewMemoryDatabase()
	snap, err := newSnapshot(db)
	assert.Nil(t, err)

	validator1 := common.HexToAddress("0xcc7c8317b21e1cea6139700c3c46c21af998d14c")
	validator2 := common.HexToAddress("0x44d1ce0b7cb3588bca96151fe1bc05af38f91b6c")
	validator3 := common.HexToAddress("0xf541c3cd1d2df407fb9bb52b3489fc2aaeedd97e")
	assert.Nil(t, snap.SetValidators(SortableAddresses{
		SortableAddress{Address: validator1, Weight: big.NewInt(0)},
		SortableAddress{Address: validator2, Weight: big.NewInt(0)},
		SortableAddress{Address: validator3, Weight: big.NewInt(0)},
	}))

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
