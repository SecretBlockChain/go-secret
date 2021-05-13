package senate

import (
	"errors"
	"reflect"
	"strings"

	"github.com/SecretBlockChain/go-secret/common"
	"github.com/SecretBlockChain/go-secret/core/types"
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
