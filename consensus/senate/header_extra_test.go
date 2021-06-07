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
	headerExtra.Root.CandidateHash.Generate(rand, 0)
	headerExtra.Root.MintCntHash.Generate(rand, 0)

	address1 := common.HexToAddress("0xcc7c8317b21e1cea6139700c3c46c21af998d14c")
	address2 := common.HexToAddress("0x44d1ce0b7cb3588bca96151fe1bc05af38f91b6c")
	headerExtra.CurrentEpochValidators = SortableAddresses{
		SortableAddress{Address: address1, Weight: big.NewInt(0)},
		SortableAddress{Address: address2, Weight: big.NewInt(0)},
	}
	headerExtra.CurrentBlockCandidates = []common.Address{address1, address2}

	data, err := headerExtra.Encode()
	assert.Nil(t, err)

	newHeaderExtra, err := NewHeaderExtra(data)
	assert.Nil(t, err)
	assert.Equal(t, newHeaderExtra.Root.EpochHash, headerExtra.Root.EpochHash)
	assert.Equal(t, newHeaderExtra.Root.CandidateHash, headerExtra.Root.CandidateHash)
	assert.Equal(t, newHeaderExtra.Root.MintCntHash, headerExtra.Root.MintCntHash)

	assert.Equal(t, newHeaderExtra.CurrentEpochValidators, headerExtra.CurrentEpochValidators)
	assert.Equal(t, newHeaderExtra.CurrentBlockCandidates, headerExtra.CurrentBlockCandidates)
}

func TestHeaderExtraEqual(t *testing.T) {
	var headerExtra HeaderExtra
	var otherHeaderExtra HeaderExtra

	assert.True(t, headerExtra.Equal(otherHeaderExtra))

	headerExtra.CurrentBlockCandidates = append(headerExtra.CurrentBlockCandidates, common.HexToAddress("0xcc7c8317b21e1cea6139700c3c46c21af998d14c"))
	assert.False(t, headerExtra.Equal(otherHeaderExtra))
	otherHeaderExtra.CurrentBlockCandidates = append(otherHeaderExtra.CurrentBlockCandidates, headerExtra.CurrentBlockCandidates[0])
	assert.True(t, headerExtra.Equal(otherHeaderExtra))

	headerExtra.CurrentBlockKickOutCandidates = append(headerExtra.CurrentBlockKickOutCandidates, common.HexToAddress("0xcc7c8317b21e1cea6139700c3c46c21af998d14c"))
	assert.False(t, headerExtra.Equal(otherHeaderExtra))
	otherHeaderExtra.CurrentBlockKickOutCandidates = append(otherHeaderExtra.CurrentBlockKickOutCandidates, headerExtra.CurrentBlockKickOutCandidates[0])
	assert.True(t, headerExtra.Equal(otherHeaderExtra))

	headerExtra.CurrentEpochValidators = append(headerExtra.CurrentEpochValidators, SortableAddress{
		Address: common.HexToAddress("0xcc7c8317b21e1cea6139700c3c46c21af998d14c"),
		Weight:  big.NewInt(1223),
	})
	assert.False(t, headerExtra.Equal(otherHeaderExtra))
	otherHeaderExtra.CurrentEpochValidators = append(otherHeaderExtra.CurrentEpochValidators, headerExtra.CurrentEpochValidators[0])
	assert.True(t, headerExtra.Equal(otherHeaderExtra))
}
