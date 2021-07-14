package equality

import (
	"math/big"
	"sort"

	"github.com/SecretBlockChain/go-secret/common"
	"github.com/SecretBlockChain/go-secret/common/math"
	"github.com/SecretBlockChain/go-secret/consensus"
	"github.com/SecretBlockChain/go-secret/core/types"
	"github.com/SecretBlockChain/go-secret/rpc"
)

type rpcCandidate struct {
	Address     common.Address        `json:"address"`
	Staked      *math.HexOrDecimal256 `json:"staked"`
	BlockNumber *math.HexOrDecimal256 `json:"blockNumber"`
}

func unwrapBigInt(val *math.HexOrDecimal256) *big.Int {
	return (*big.Int)(val)
}

type rpcCandidateSlice []rpcCandidate

func (p rpcCandidateSlice) Len() int { return len(p) }
func (p rpcCandidateSlice) Less(i, j int) bool {
	return unwrapBigInt(p[i].BlockNumber).Cmp(unwrapBigInt(p[j].BlockNumber)) == -1
}
func (p rpcCandidateSlice) Swap(i, j int) { p[i], p[j] = p[j], p[i] }

type rpcValidator struct {
	Address     common.Address `json:"address"`
	CountMinted *big.Int       `json:"countMinted"`
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
func (api *API) loadSnapshot(number *rpc.BlockNumber) (*Snapshot, HeaderExtra, error) {
	var header *types.Header
	if number == nil || *number == rpc.LatestBlockNumber {
		header = api.chain.CurrentHeader()
	} else {
		header = api.chain.GetHeaderByNumber(uint64(number.Int64()))
	}
	if header == nil {
		return nil, HeaderExtra{}, errUnknownBlock
	}

	headerExtra, err := DecodeHeaderExtra(header)
	if err != nil {
		return nil, HeaderExtra{}, err
	}

	snap, err := loadSnapshot(api.equality.db, headerExtra.Root)
	return snap, headerExtra, err
}

// GetAddress retrieves the candidate information of the address
func (api *API) GetAddress(address common.Address, number *rpc.BlockNumber) (rpcCandidateInfo, error) {
	snap, _, err := api.loadSnapshot(number)
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
func (api *API) GetCandidates(number *rpc.BlockNumber) (rpcCandidateSlice, error) {
	snap, _, err := api.loadSnapshot(number)
	if err != nil {
		return nil, err
	}

	candidates, err := snap.GetCandidates()
	if err != nil {
		return nil, err
	}

	result := make(rpcCandidateSlice, 0, len(candidates))
	for addr, candidate := range candidates {
		c := rpcCandidate{Address: addr}
		staked := math.HexOrDecimal256(*candidate.Staked)
		c.Staked = &staked
		blockNumber := math.NewHexOrDecimal256(int64(candidate.BlockNumber))
		c.BlockNumber = blockNumber
		result = append(result, c)
	}
	sort.Sort(result)
	return result, nil
}

// GetValidators retrieves the list of the validators at specified block
func (api *API) GetValidators(number *rpc.BlockNumber) ([]rpcValidator, error) {
	snap, headerExtra, err := api.loadSnapshot(number)
	if err != nil {
		return nil, err
	}

	validators, err := snap.GetValidators()
	if err != nil {
		return nil, err
	}

	mapper := make(map[common.Address]*big.Int)
	addresses, err := snap.CountMinted(headerExtra.Epoch)
	if err != nil {
		return nil, err
	}
	for _, address := range addresses {
		mapper[address.Address] = address.Weight
	}

	result := make([]rpcValidator, 0, len(validators))
	for _, validator := range validators {
		count, _ := mapper[validator]
		v := rpcValidator{Address: validator, CountMinted: count}
		result = append(result, v)
	}
	return result, nil
}
