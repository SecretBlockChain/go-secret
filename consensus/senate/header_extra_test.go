package senate

import (
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/SecretBlockChain/go-secret/common"
	"github.com/stretchr/testify/assert"
)

func TestEncodeHeaderExtra(t *testing.T) {
	var headerExtra HeaderExtra
	rand := rand.New(rand.NewSource(time.Now().Unix()))
	headerExtra.Root.EpochHash.Generate(rand, 0)
	headerExtra.Root.DelegateHash.Generate(rand, 0)
	headerExtra.Root.CandidateHash.Generate(rand, 0)
	headerExtra.Root.VoteHash.Generate(rand, 0)
	headerExtra.Root.MintCntHash.Generate(rand, 0)

	address1 := common.HexToAddress("0xcc7c8317b21e1cea6139700c3c46c21af998d14c")
	address2 := common.HexToAddress("0x44d1ce0b7cb3588bca96151fe1bc05af38f91b6c")
	headerExtra.CurrentEpochValidators = SortableAddresses{
		SortableAddress{Address: address1, Weight: big.NewInt(0)},
		SortableAddress{Address: address2, Weight: big.NewInt(0)},
	}
	headerExtra.CurrentBlockDelegates = []Delegate{{Delegator: address1, Candidate: address2}}
	headerExtra.CurrentBlockCandidates = []common.Address{address1, address2}

	data, err := headerExtra.Encode()
	assert.Nil(t, err)

	newHeaderExtra, err := NewHeaderExtra(data)
	assert.Nil(t, err)
	assert.Equal(t, newHeaderExtra.Root.EpochHash, headerExtra.Root.EpochHash)
	assert.Equal(t, newHeaderExtra.Root.DelegateHash, headerExtra.Root.DelegateHash)
	assert.Equal(t, newHeaderExtra.Root.CandidateHash, headerExtra.Root.CandidateHash)
	assert.Equal(t, newHeaderExtra.Root.VoteHash, headerExtra.Root.VoteHash)
	assert.Equal(t, newHeaderExtra.Root.MintCntHash, headerExtra.Root.MintCntHash)

	assert.Equal(t, newHeaderExtra.CurrentEpochValidators, headerExtra.CurrentEpochValidators)
	assert.Equal(t, newHeaderExtra.CurrentBlockDelegates, headerExtra.CurrentBlockDelegates)
	assert.Equal(t, newHeaderExtra.CurrentBlockCandidates, headerExtra.CurrentBlockCandidates)
}

func TestHeaderExtraEqual(t *testing.T) {
	var headerExtra HeaderExtra
	var otherHeaderExtra HeaderExtra

	headerExtra.CurrentBlockDelegates = append(headerExtra.CurrentBlockDelegates, Delegate{
		Delegator: common.HexToAddress("0xcc7c8317b21e1cea6139700c3c46c21af998d14c"),
		Candidate: common.HexToAddress("0xcc7c8317b21e1cea6139700c3c46c21af998d14c"),
	})
	assert.False(t, headerExtra.Equal(otherHeaderExtra))
	otherHeaderExtra.CurrentBlockDelegates = append(otherHeaderExtra.CurrentBlockDelegates, headerExtra.CurrentBlockDelegates[0])
	assert.True(t, headerExtra.Equal(otherHeaderExtra))

	headerExtra.CurrentBlockCandidates = append(headerExtra.CurrentBlockCandidates, common.HexToAddress("0xcc7c8317b21e1cea6139700c3c46c21af998d14c"))
	assert.False(t, headerExtra.Equal(otherHeaderExtra))
	otherHeaderExtra.CurrentBlockCandidates = append(otherHeaderExtra.CurrentBlockCandidates, headerExtra.CurrentBlockCandidates[0])
	assert.True(t, headerExtra.Equal(otherHeaderExtra))

	headerExtra.CurrentBlockKickOutCandidates = append(headerExtra.CurrentBlockKickOutCandidates, common.HexToAddress("0xcc7c8317b21e1cea6139700c3c46c21af998d14c"))
	assert.False(t, headerExtra.Equal(otherHeaderExtra))
	otherHeaderExtra.CurrentBlockKickOutCandidates = append(otherHeaderExtra.CurrentBlockKickOutCandidates, headerExtra.CurrentBlockKickOutCandidates[0])
	assert.True(t, headerExtra.Equal(otherHeaderExtra))

	headerExtra.CurrentBlockProposals = append(headerExtra.CurrentBlockProposals, Proposal{
		Key:      "a",
		Value:    "b",
		Hash:     common.HexToHash("0x90fcc640d56532c8d4f1255a44533b8d097149c67e298fc7baa1d920925e235f"),
		Proposer: common.HexToAddress("0xcc7c8317b21e1cea6139700c3c46c21af998d14c"),
	})
	assert.False(t, headerExtra.Equal(otherHeaderExtra))
	otherHeaderExtra.CurrentBlockProposals = append(otherHeaderExtra.CurrentBlockProposals, headerExtra.CurrentBlockProposals[0])
	assert.True(t, headerExtra.Equal(otherHeaderExtra))

	headerExtra.CurrentBlockDeclares = append(headerExtra.CurrentBlockDeclares, Declare{
		ProposalHash: common.HexToHash("0x90fcc640d56532c8d4f1255a44533b8d097149c67e298fc7baa1d920925e235f"),
		Declarer:     common.HexToAddress("0xcc7c8317b21e1cea6139700c3c46c21af998d14c"),
		Decision:     true,
	})
	assert.False(t, headerExtra.Equal(otherHeaderExtra))
	otherHeaderExtra.CurrentBlockDeclares = append(otherHeaderExtra.CurrentBlockDeclares, headerExtra.CurrentBlockDeclares[0])
	assert.True(t, headerExtra.Equal(otherHeaderExtra))

	headerExtra.CurrentEpochValidators = append(headerExtra.CurrentEpochValidators, SortableAddress{
		Address: common.HexToAddress("0xcc7c8317b21e1cea6139700c3c46c21af998d14c"),
		Weight:  big.NewInt(1223),
	})
	assert.False(t, headerExtra.Equal(otherHeaderExtra))
	otherHeaderExtra.CurrentEpochValidators = append(otherHeaderExtra.CurrentEpochValidators, headerExtra.CurrentEpochValidators[0])
	assert.True(t, headerExtra.Equal(otherHeaderExtra))
}
