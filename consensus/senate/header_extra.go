package senate

import (
	"bytes"
	"compress/gzip"
	"io"

	"github.com/SecretBlockChain/go-secret/common"
	"github.com/SecretBlockChain/go-secret/core/types"
	"github.com/SecretBlockChain/go-secret/params"
	"github.com/SecretBlockChain/go-secret/rlp"
)

// Root is the state tree root.
type Root struct {
	EpochHash     common.Hash
	DelegateHash  common.Hash
	CandidateHash common.Hash
	VoteHash      common.Hash
	MintCntHash   common.Hash
	ConfigHash    common.Hash
}

// Delegate come from custom tx which data like "senate:1:event:delegate".
// Sender of tx is Delegator, the tx.to is Candidate.
type Delegate struct {
	Delegator common.Address
	Candidate common.Address
}

// HeaderExtra is the struct of info in header.Extra[extraVanity:len(header.extra)-extraSeal].
// HeaderExtra is the current struct.
type HeaderExtra struct {
	Root                          Root
	Epoch                         uint64
	EpochTime                     uint64
	ChainConfig                   []params.SenateConfig
	CurrentBlockDelegates         []Delegate
	CurrentBlockCandidates        []common.Address
	CurrentBlockKickOutCandidates []common.Address
	CurrentEpochValidators        SortableAddresses
}

// NewHeaderExtra new HeaderExtra from rlp bytes.
func NewHeaderExtra(data []byte) (HeaderExtra, error) {
	r, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return HeaderExtra{}, err
	}

	buffer := bytes.NewBuffer(nil)
	for {
		var temp [128]byte
		n, err := r.Read(temp[:])
		if n > 0 {
			buffer.Write(temp[:n])
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return HeaderExtra{}, err
		}
	}

	var extra HeaderExtra
	if err := rlp.DecodeBytes(buffer.Bytes(), &extra); err != nil {
		return HeaderExtra{}, err
	}
	return extra, nil
}

// Encode encode header extra as rlp bytes.
func (extra HeaderExtra) Encode() ([]byte, error) {
	data, err := rlp.EncodeToBytes(extra)
	if err != nil {
		return nil, err
	}

	buffer := bytes.NewBuffer(nil)
	w := gzip.NewWriter(buffer)
	w.Write(data)
	w.Close()
	return buffer.Bytes(), nil
}

// Equal compares two HeaderExtras for equality.
func (extra HeaderExtra) Equal(other HeaderExtra) bool {
	if extra.Root != other.Root {
		return false
	}
	if extra.Epoch != other.Epoch {
		return false
	}
	if extra.EpochTime != other.EpochTime {
		return false
	}

	if len(extra.ChainConfig) != len(other.ChainConfig) {
		return false
	}
	for idx, config := range extra.ChainConfig {
		if !config.Equal(other.ChainConfig[idx]) {
			return false
		}
	}

	if len(extra.CurrentBlockDelegates) != len(other.CurrentBlockDelegates) {
		return false
	}
	for idx, delegate := range extra.CurrentBlockDelegates {
		if delegate.Candidate != other.CurrentBlockDelegates[idx].Candidate {
			return false
		}
		if delegate.Delegator != other.CurrentBlockDelegates[idx].Delegator {
			return false
		}
	}

	if len(extra.CurrentBlockCandidates) != len(other.CurrentBlockCandidates) {
		return false
	}
	for idx, candidate := range extra.CurrentBlockCandidates {
		if candidate != other.CurrentBlockCandidates[idx] {
			return false
		}
	}

	if len(extra.CurrentBlockKickOutCandidates) != len(other.CurrentBlockKickOutCandidates) {
		return false
	}
	for idx, candidate := range extra.CurrentBlockKickOutCandidates {
		if candidate != other.CurrentBlockKickOutCandidates[idx] {
			return false
		}
	}

	if len(extra.CurrentEpochValidators) != len(other.CurrentEpochValidators) {
		return false
	}
	for idx, validator := range extra.CurrentEpochValidators {
		if validator.Address != other.CurrentEpochValidators[idx].Address {
			return false
		}
	}
	return true
}

func decodeHeaderExtra(header *types.Header) (HeaderExtra, error) {
	extra := header.Extra
	if len(extra) < extraVanity {
		return HeaderExtra{}, errMissingVanity
	}
	if len(extra) < extraVanity+extraSeal {
		return HeaderExtra{}, errMissingSignature
	}
	headerExtra, err := NewHeaderExtra(extra[extraVanity : len(extra)-extraSeal])
	if err != nil {
		return HeaderExtra{}, err
	}
	return headerExtra, nil
}

// Ensure each element of an Delegate slice are not the same.
func delegatesDistinct(slice []Delegate) []Delegate {
	if len(slice) <= 1 {
		return slice
	}

	set := make(map[Delegate]struct{})
	result := make([]Delegate, 0, len(slice))
	for _, address := range slice {
		if _, ok := set[address]; !ok {
			set[address] = struct{}{}
			result = append(result, address)
		}
	}
	return result
}

// Ensure each element of an common.Address slice are not the same.
func addressesDistinct(slice []common.Address) []common.Address {
	if len(slice) <= 1 {
		return slice
	}

	set := make(map[common.Address]struct{})
	result := make([]common.Address, 0, len(slice))
	for _, address := range slice {
		if _, ok := set[address]; !ok {
			set[address] = struct{}{}
			result = append(result, address)
		}
	}
	return result
}
