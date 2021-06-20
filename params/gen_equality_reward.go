// Code generated by github.com/fjl/gencodec. DO NOT EDIT.

package params

import (
	"encoding/json"
	"errors"
	"math/big"

	"github.com/SecretBlockChain/go-secret/common/math"
)

var _ = (*equalityRewardMarshaling)(nil)

// MarshalJSON marshals as JSON.
func (e EqualityReward) MarshalJSON() ([]byte, error) {
	type EqualityReward struct {
		Number uint64                `json:"number"`
		Reward *math.HexOrDecimal256 `json:"reward" gencodec:"required"`
	}
	var enc EqualityReward
	enc.Number = e.Number
	enc.Reward = (*math.HexOrDecimal256)(e.Reward)
	return json.Marshal(&enc)
}

// UnmarshalJSON unmarshals from JSON.
func (e *EqualityReward) UnmarshalJSON(input []byte) error {
	type EqualityReward struct {
		Number *uint64               `json:"number"`
		Reward *math.HexOrDecimal256 `json:"reward" gencodec:"required"`
	}
	var dec EqualityReward
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	if dec.Number != nil {
		e.Number = *dec.Number
	}
	if dec.Reward == nil {
		return errors.New("missing required field 'reward' for EqualityReward")
	}
	e.Reward = (*big.Int)(dec.Reward)
	return nil
}