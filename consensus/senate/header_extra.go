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
	ProposalHash  common.Hash
	DeclareHash   common.Hash
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
	CurrentBlockProposals         []Proposal
	CurrentBlockDeclares          []Declare
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

	var headerExtra HeaderExtra
	if err := rlp.DecodeBytes(buffer.Bytes(), &headerExtra); err != nil {
		return HeaderExtra{}, err
	}
	return headerExtra, nil
}

// Encode encode header extra as rlp bytes.
func (headerExtra HeaderExtra) Encode() ([]byte, error) {
	data, err := rlp.EncodeToBytes(headerExtra)
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
func (headerExtra HeaderExtra) Equal(other HeaderExtra) bool {
	if headerExtra.Root != other.Root {
		return false
	}
	if headerExtra.Epoch != other.Epoch {
		return false
	}
	if headerExtra.EpochTime != other.EpochTime {
		return false
	}

	if len(headerExtra.ChainConfig) != len(other.ChainConfig) {
		return false
	}
	for idx, config := range headerExtra.ChainConfig {
		if !config.Equal(other.ChainConfig[idx]) {
			return false
		}
	}

	if len(headerExtra.CurrentBlockDelegates) != len(other.CurrentBlockDelegates) {
		return false
	}
	for idx, delegate := range headerExtra.CurrentBlockDelegates {
		if delegate.Candidate != other.CurrentBlockDelegates[idx].Candidate {
			return false
		}
		if delegate.Delegator != other.CurrentBlockDelegates[idx].Delegator {
			return false
		}
	}

	if len(headerExtra.CurrentBlockCandidates) != len(other.CurrentBlockCandidates) {
		return false
	}
	for idx, candidate := range headerExtra.CurrentBlockCandidates {
		if candidate != other.CurrentBlockCandidates[idx] {
			return false
		}
	}

	if len(headerExtra.CurrentBlockKickOutCandidates) != len(other.CurrentBlockKickOutCandidates) {
		return false
	}
	for idx, candidate := range headerExtra.CurrentBlockKickOutCandidates {
		if candidate != other.CurrentBlockKickOutCandidates[idx] {
			return false
		}
	}

	if len(headerExtra.CurrentBlockDeclares) != len(other.CurrentBlockDeclares) {
		return false
	}
	for idx, declare := range headerExtra.CurrentBlockDeclares {
		if declare != other.CurrentBlockDeclares[idx] {
			return false
		}
	}

	if len(headerExtra.CurrentBlockProposals) != len(other.CurrentBlockProposals) {
		return false
	}
	for idx, proposal := range headerExtra.CurrentBlockProposals {
		if proposal != other.CurrentBlockProposals[idx] {
			return false
		}
	}

	if len(headerExtra.CurrentEpochValidators) != len(other.CurrentEpochValidators) {
		return false
	}
	for idx, validator := range headerExtra.CurrentEpochValidators {
		if validator.Address != other.CurrentEpochValidators[idx].Address {
			return false
		}
	}
	return true
}

func decodeHeaderExtra(header *types.Header) (HeaderExtra, error) {
	headerExtra := header.Extra
	if len(headerExtra) < extraVanity {
		return HeaderExtra{}, errMissingVanity
	}
	if len(headerExtra) < extraVanity+extraSeal {
		return HeaderExtra{}, errMissingSignature
	}
	return NewHeaderExtra(headerExtra[extraVanity : len(headerExtra)-extraSeal])
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
