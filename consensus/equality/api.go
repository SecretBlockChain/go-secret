package equality

import (
	"github.com/SecretBlockChain/go-secret/common/math"

	"github.com/SecretBlockChain/go-secret/common"
	"github.com/SecretBlockChain/go-secret/consensus"
	"github.com/SecretBlockChain/go-secret/core/types"
	"github.com/SecretBlockChain/go-secret/rpc"
)

type rpcCandidate struct {
	Address     common.Address        `json:"address"`
	Staked      *math.HexOrDecimal256 `json:"staked"`
	BlockNumber *math.HexOrDecimal256 `json:"blockNumber"`
}

type rpcCandidateInfo struct {
	Address     common.Address        `json:"address"`
	IsCandidate bool                  `json:"isCandidate"`
	IsValidator bool                  `json:"isValidator"`
	Staked      *math.HexOrDecimal256 `json:"staked"`
	BlockNumber *math.HexOrDecimal256 `json:"blockNumber"`
}

// API is a user facing RPC API to allow controlling the signer and voting
// mechanisms of the proof-of-equality scheme.
type API struct {
	chain    consensus.ChainHeaderReader
	equality *Equality
}

// load a snapshot at specified block
func (api *API) loadSnapshot(number *rpc.BlockNumber) (*Snapshot, error) {
	var header *types.Header
	if number == nil || *number == rpc.LatestBlockNumber {
		header = api.chain.CurrentHeader()
	} else {
		header = api.chain.GetHeaderByNumber(uint64(number.Int64()))
	}
	if header == nil {
		return nil, errUnknownBlock
	}

	headerExtra, err := DecodeHeaderExtra(header)
	if err != nil {
		return nil, err
	}
	return loadSnapshot(api.equality.db, headerExtra.Root)
}

// GetAddress retrieves the candidate information of the address
func (api *API) GetAddress(address common.Address, number *rpc.BlockNumber) (rpcCandidateInfo, error) {
	snap, err := api.loadSnapshot(number)
	if err != nil {
		return rpcCandidateInfo{}, err
	}

	result := rpcCandidateInfo{Address: address}
	candidate, err := snap.GetCandidate(address)
	if err != nil {
		return rpcCandidateInfo{}, err
	}
	if candidate != nil {
		result.IsCandidate = true
		staked := math.HexOrDecimal256(*candidate.Staked)
		result.Staked = &staked
		blockNumber := math.NewHexOrDecimal256(int64(candidate.BlockNumber))
		result.BlockNumber = blockNumber
	}

	validators, err := snap.GetValidators()
	if err != nil {
		return rpcCandidateInfo{}, err
	}
	for _, validator := range validators {
		if validator == address {
			result.IsValidator = true
			break
		}
	}
	return result, nil
}

// GetCandidates retrieves the list of the candidates at specified block
func (api *API) GetCandidates(number *rpc.BlockNumber) ([]rpcCandidate, error) {
	snap, err := api.loadSnapshot(number)
	if err != nil {
		return nil, err
	}

	candidates, err := snap.GetCandidates()
	if err != nil {
		return nil, err
	}

	result := make([]rpcCandidate, 0, len(candidates))
	for addr, candidate := range candidates {
		c := rpcCandidate{Address: addr}
		staked := math.HexOrDecimal256(*candidate.Staked)
		c.Staked = &staked
		blockNumber := math.NewHexOrDecimal256(int64(candidate.BlockNumber))
		c.BlockNumber = blockNumber
		result = append(result, c)
	}
	return result, nil
}

// GetValidators retrieves the list of the validators at specified block
func (api *API) GetValidators(number *rpc.BlockNumber) ([]common.Address, error) {
	snap, err := api.loadSnapshot(number)
	if err != nil {
		return nil, err
	}
	return snap.GetValidators()
}
