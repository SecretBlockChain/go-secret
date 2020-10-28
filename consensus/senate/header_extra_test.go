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
	var extra HeaderExtra
	rand := rand.New(rand.NewSource(time.Now().Unix()))
	extra.Root.EpochHash.Generate(rand, 0)
	extra.Root.DelegateHash.Generate(rand, 0)
	extra.Root.CandidateHash.Generate(rand, 0)
	extra.Root.VoteHash.Generate(rand, 0)
	extra.Root.MintCntHash.Generate(rand, 0)

	address1 := common.HexToAddress("0xcc7c8317b21e1cea6139700c3c46c21af998d14c")
	address2 := common.HexToAddress("0x44d1ce0b7cb3588bca96151fe1bc05af38f91b6c")
	extra.CurrentEpochValidators = SortableAddresses{
		SortableAddress{Address: address1, Weight: big.NewInt(0)},
		SortableAddress{Address: address2, Weight: big.NewInt(0)},
	}
	extra.CurrentBlockDelegates = []Delegate{{Delegator: address1, Candidate: address2}}
	extra.CurrentBlockCandidates = []common.Address{address1, address2}

	data, err := extra.Encode()
	assert.Nil(t, err)

	newExtra, err := NewHeaderExtra(data)
	assert.Nil(t, err)
	assert.Equal(t, newExtra.Root.EpochHash, extra.Root.EpochHash)
	assert.Equal(t, newExtra.Root.DelegateHash, extra.Root.DelegateHash)
	assert.Equal(t, newExtra.Root.CandidateHash, extra.Root.CandidateHash)
	assert.Equal(t, newExtra.Root.VoteHash, extra.Root.VoteHash)
	assert.Equal(t, newExtra.Root.MintCntHash, extra.Root.MintCntHash)

	assert.Equal(t, newExtra.CurrentEpochValidators, extra.CurrentEpochValidators)
	assert.Equal(t, newExtra.CurrentBlockDelegates, extra.CurrentBlockDelegates)
	assert.Equal(t, newExtra.CurrentBlockCandidates, extra.CurrentBlockCandidates)
}
