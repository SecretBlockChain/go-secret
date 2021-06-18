package equality

import (
	"github.com/SecretBlockChain/go-secret/common"
	"github.com/SecretBlockChain/go-secret/consensus"
	"github.com/SecretBlockChain/go-secret/core/types"
	"github.com/SecretBlockChain/go-secret/rpc"
)

// API is a user facing RPC API to allow controlling the signer and voting
// mechanisms of the proof-of-equality scheme.
type API struct {
	chain    consensus.ChainHeaderReader
	equality *Equality
}

// GetCandidates retrieves the list of the candidates at specified block
func (api *API) GetCandidates(number *rpc.BlockNumber) ([]common.Address, error) {
	var header *types.Header
	if number == nil || *number == rpc.LatestBlockNumber {
		header = api.chain.CurrentHeader()
	} else {
		header = api.chain.GetHeaderByNumber(uint64(number.Int64()))
	}
	if header == nil {
		return nil, errUnknownBlock
	}

	headerExtra, err := decodeHeaderExtra(header)
	if err != nil {
		return nil, err
	}

	snap, err := loadSnapshot(api.equality.db, headerExtra.Root)
	if err != nil {
		return nil, err
	}
	return snap.GetCandidates()
}

// GetValidators retrieves the list of the validators at specified block
func (api *API) GetValidators(number *rpc.BlockNumber) ([]common.Address, error) {
	var header *types.Header
	if number == nil || *number == rpc.LatestBlockNumber {
		header = api.chain.CurrentHeader()
	} else {
		header = api.chain.GetHeaderByNumber(uint64(number.Int64()))
	}
	if header == nil {
		return nil, errUnknownBlock
	}

	headerExtra, err := decodeHeaderExtra(header)
	if err != nil {
		return nil, err
	}

	snap, err := loadSnapshot(api.equality.db, headerExtra.Root)
	if err != nil {
		return nil, err
	}

	validators, err := snap.GetValidators()
	if err != nil {
		return nil, err
	}

	result := make([]common.Address, 0, len(validators))
	for _, validator := range validators {
		result = append(result, validator.Address)
	}
	return result, nil
}
