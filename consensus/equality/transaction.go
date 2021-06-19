package equality

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
		new(EventBecomeCandidate),
		new(EventCancelCandidate),
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
// data format: equality:version:type:action:data
func NewTransaction(tx *types.Transaction) (Transaction, error) {
	slice := strings.Split(string(tx.Data()), ":")
	if len(slice) < 4 {
		return nil, errors.New("invalid custom transaction data")
	}

	prefix, version, txType, action := slice[0], slice[1], TransactionType(slice[2]), slice[3]
	if prefix != "equality" {
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
// data like "equality:1:event:candidate"
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

// EventCancelCandidate apply to cancel Candidate.
// data like "equality:1:event:candidateQuit"
// Sender will cancel Candidate status
type EventCancelCandidate struct {
	Candidate common.Address
}

func (event *EventCancelCandidate) Type() TransactionType {
	return EventTransactionType
}

func (event *EventCancelCandidate) Action() string {
	return "candidateQuit"
}

func (event *EventCancelCandidate) Decode(tx *types.Transaction, data []byte) error {
	txSender, err := types.Sender(types.NewEIP155Signer(tx.ChainId()), tx)
	if err != nil {
		return err
	}
	event.Candidate = txSender
	return nil
}
