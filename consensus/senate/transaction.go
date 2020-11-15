package senate

import (
	"errors"
	"math/big"
	"reflect"
	"strconv"
	"strings"

	"github.com/SecretBlockChain/go-secret/common"
	"github.com/SecretBlockChain/go-secret/core/types"
	"github.com/SecretBlockChain/go-secret/params"
)

// Transaction custom transaction interface.
type Transaction interface {
	Type() TransactionType
	Action() string
	Decode(*types.Transaction, []byte) error
}

// TransactionType custom transaction type enums.
type TransactionType string

const (
	EventTransactionType TransactionType = "event"
)

var (
	prototypes = []Transaction{
		new(Declare),
		new(Proposal),
		new(EventDelegate),
		new(EventBecomeCandidate),
	}
	prototypeMapper = map[TransactionType][]Transaction{}
)

func init() {
	for _, prototype := range prototypes {
		slice, _ := prototypeMapper[prototype.Type()]
		prototypeMapper[prototype.Type()] = append(slice, prototype)
	}
}

// NewTransaction new custom transaction from transaction data.
// data format: senate:version:type:action:data
func NewTransaction(tx *types.Transaction) (Transaction, error) {
	slice := strings.Split(string(tx.Data()), ":")
	if len(slice) < 4 {
		return nil, errors.New("invalid custom transaction data")
	}

	prefix, version, txType, action := slice[0], slice[1], TransactionType(slice[2]), slice[3]
	if prefix != "senate" {
		return nil, errors.New("invalid custom transaction prefix")
	}
	if version != "1" {
		return nil, errors.New("invalid custom transaction version")
	}

	types, ok := prototypeMapper[txType]
	if !ok {
		return nil, errors.New("undefined custom transaction type")
	}

	var data []byte
	if len(slice) > 4 {
		data = []byte(strings.Join(slice[4:], ":"))
	}

	for _, typ := range types {
		if typ.Action() == action {
			t := reflect.TypeOf(typ)
			if t.Kind() == reflect.Ptr {
				t = t.Elem()
			}
			ctx := reflect.New(t).Interface().(Transaction)
			if err := ctx.Decode(tx, data); err != nil {
				return nil, err
			}
			return ctx, nil
		}
	}
	return nil, errors.New("undefined custom transaction action")
}

// EventDelegate delegate rights to Candidate.
// data like "senate:1:event:delegate"
// Sender of tx is Delegator, the tx.to is Candidate
type EventDelegate struct {
	Delegator common.Address
	Candidate common.Address
}

func (event *EventDelegate) Type() TransactionType {
	return EventTransactionType
}

func (event *EventDelegate) Action() string {
	return "delegate"
}

func (event *EventDelegate) Decode(tx *types.Transaction, data []byte) error {
	if tx.To() == nil {
		return errors.New("missing candidate")
	}

	txSender, err := types.Sender(types.NewEIP155Signer(tx.ChainId()), tx)
	if err != nil {
		return err
	}
	event.Delegator = txSender
	event.Candidate = *tx.To()
	return nil
}

// EventBecomeCandidate apply to become Candidate.
// data like "senate:1:event:candidate"
// Sender will become a Candidate
type EventBecomeCandidate struct {
	Candidate common.Address
}

func (event *EventBecomeCandidate) Type() TransactionType {
	return EventTransactionType
}

func (event *EventBecomeCandidate) Action() string {
	return "candidate"
}

func (event *EventBecomeCandidate) Decode(tx *types.Transaction, data []byte) error {
	txSender, err := types.Sender(types.NewEIP155Signer(tx.ChainId()), tx)
	if err != nil {
		return err
	}
	event.Candidate = txSender
	return nil
}

// Proposal proposal to modify the configuration of the senate consensus.
// data like "senate:1:event:proposal:period:8"
// data like "senate:1:event:proposal:epoch:86400"
// data like "senate:1:event:proposal:maxValidatorsCount:21"
// data like "senate:1:event:proposal:minDelegatorBalance:0xde0b6b3a7640000"
// data like "senate:1:event:proposal:minCandidateBalance:0x56bc75e2d63100000"
// data like "senate:1:event:proposal:rewards:0x69e10de76676d0800000:0x4563918244f40000,0x13da329b6336471800000:0x1bc16d674ec80000,0x422ca8b0a00a425000000:0xde0b6b3a7640000"
type Proposal struct {
	Key          string         `json:"key"`
	Value        string         `json:"value"`
	Hash         common.Hash    `json:"hash"`
	Proposer     common.Address `json:"proposer"`
	ApprovedHash *common.Hash   `json:"approved_hash"`
}

func (proposal *Proposal) Type() TransactionType {
	return EventTransactionType
}

func (proposal *Proposal) Action() string {
	return "proposal"
}

func (proposal *Proposal) applyTo(config *params.SenateConfig) error {
	if len(proposal.Key) == 0 || len(proposal.Value) == 0 {
		return errors.New("invalid proposal")
	}

	var ok bool
	var err error
	switch proposal.Key {
	case "period":
		config.Period, err = strconv.ParseUint(proposal.Value, 10, 64)
		if err != nil || config.Period <= 0 {
			return errors.New("invalid value: period")
		}
	case "epoch":
		config.Epoch, err = strconv.ParseUint(proposal.Value, 10, 64)
		if err != nil || config.Epoch <= 0 {
			return errors.New("invalid value: epoch")
		}
	case "maxValidatorsCount":
		config.MaxValidatorsCount, err = strconv.ParseUint(proposal.Value, 10, 64)
		if err != nil || config.MaxValidatorsCount <= 0 {
			return errors.New("invalid value: maxValidatorsCount")
		}
	case "minDelegatorBalance":
		if len(proposal.Value) <= 2 || strings.ToLower(proposal.Value[:2]) != "0x" {
			return errors.New("invalid value: minDelegatorBalance")
		}
		config.MinDelegatorBalance, ok = big.NewInt(0).SetString(proposal.Value[2:], 16)
		if !ok || config.MinDelegatorBalance.Cmp(big.NewInt(0)) == -1 {
			return errors.New("invalid value: minDelegatorBalance")
		}
	case "minCandidateBalance":
		if len(proposal.Value) <= 2 || strings.ToLower(proposal.Value[:2]) != "0x" {
			return errors.New("invalid value: minDelegatorBalance")
		}
		config.MinCandidateBalance, ok = big.NewInt(0).SetString(proposal.Value[2:], 16)
		if !ok || config.MinCandidateBalance.Cmp(big.NewInt(0)) == -1 {
			return errors.New("invalid value: minCandidateBalance")
		}
	case "rewards":
		config.Rewards = nil
		lastHeight := big.NewInt(-1)
		for _, s := range strings.Split(proposal.Value, ",") {
			slice := strings.SplitN(s, ":", 2)
			if len(slice) != 2 || len(slice[0]) <= 2 || strings.ToLower(slice[0][:2]) != "0x" ||
				len(slice[1]) <= 2 || strings.ToLower(slice[1][:2]) != "0x" {
				return errors.New("invalid value: rewards")
			}

			height, ok := big.NewInt(0).SetString(slice[0][2:], 16)
			if !ok || height.Cmp(big.NewInt(0)) <= 0 || height.Cmp(lastHeight) <= 0 {
				return errors.New("invalid value: rewards")
			}
			reward, ok := big.NewInt(0).SetString(slice[1][2:], 16)
			if !ok || reward.Cmp(big.NewInt(0)) == -1 {
				return errors.New("invalid value: rewards")
			}
			lastHeight = height
			config.Rewards = append(config.Rewards, params.SenateReward{
				Height: height.Uint64(),
				Reward: reward,
			})
		}
	default:
		return errors.New("unknown key: " + proposal.Key)
	}
	return nil
}

func (proposal *Proposal) Decode(tx *types.Transaction, data []byte) error {
	txSender, err := types.Sender(types.NewEIP155Signer(tx.ChainId()), tx)
	if err != nil {
		return err
	}

	slice := strings.SplitN(string(data), ":", 2)
	if len(slice) != 2 {
		return errors.New("invalid proposal")
	}

	proposal.Hash = tx.Hash()
	proposal.Proposer = txSender
	proposal.Key, proposal.Value = slice[0], slice[1]
	return proposal.applyTo(new(params.SenateConfig))
}

// Declare declare come from custom tx which data like "senate:1:event:declare:hash:yes".
// proposal only come from the current candidates
// hash is the hash of proposal tx
type Declare struct {
	Hash         common.Hash    `json:"hash"`
	ProposalHash common.Hash    `json:"proposal_hash"`
	Declarer     common.Address `json:"declarer"`
	Decision     bool           `json:"decision"`
}

func (declare *Declare) Type() TransactionType {
	return EventTransactionType
}

func (declare *Declare) Action() string {
	return "declare"
}

func (declare *Declare) Decode(tx *types.Transaction, data []byte) error {
	txSender, err := types.Sender(types.NewEIP155Signer(tx.ChainId()), tx)
	if err != nil {
		return err
	}

	slice := strings.SplitN(string(data), ":", 2)
	if len(slice) != 2 {
		return errors.New("invalid declare")
	}

	declare.Hash = tx.Hash()
	declare.Declarer = txSender
	declare.ProposalHash = common.HexToHash(slice[0])
	declare.Decision = slice[1] == "yes"
	return nil
}
