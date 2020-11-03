package senate

import (
	"sort"

	"github.com/SecretBlockChain/go-secret/consensus"
	"github.com/SecretBlockChain/go-secret/core/types"
	"github.com/SecretBlockChain/go-secret/rpc"
)

// API is a user facing RPC API to allow controlling the signer and voting
// mechanisms of the delegated-proof-of-stake scheme.
type API struct {
	chain  consensus.ChainHeaderReader
	senate *Senate
}

// GetValidators retrieves the list of the validators at specified block
func (api *API) GetValidators(number *rpc.BlockNumber) (SortableAddresses, error) {
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

	snap, err := loadSnapshot(api.senate.db, headerExtra.Root)
	if err != nil {
		return nil, err
	}

	validators, err := snap.GetValidators()
	if err != nil {
		return nil, err
	}
	sort.Sort(validators)
	return validators, nil
}
