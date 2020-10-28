package senate

import (
	"errors"
	"reflect"
	"strings"

	"github.com/SecretBlockChain/go-secret/common"
	"github.com/SecretBlockChain/go-secret/core/types"
)

type TxType string

const (
	EventTx TxType = "event"
)

type CustomTx interface {
	Type() TxType
	Action() string
	Decode(*types.Transaction, []byte) error
}

var (
	customTxs = []CustomTx{
		new(EventDelegate),
		new(EventBecomeCandidate),
	}
	customTxTypes = map[TxType][]CustomTx{}
)

func init() {
	for _, tx := range customTxs {
		slice, _ := customTxTypes[tx.Type()]
		customTxTypes[tx.Type()] = append(slice, tx)
	}
}

// EventDelegate delegate rights to Candidate.
// data like "senate:1:event:delegate"
// Sender of tx is Delegator, the tx.to is Candidate
type EventDelegate struct {
	Delegator common.Address
	Candidate common.Address
}

func (event *EventDelegate) Type() TxType {
	return EventTx
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

func (event *EventBecomeCandidate) Type() TxType {
	return EventTx
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

// NewCustomTx new CustomTx from transaction data.
// data format: senate:version:type:action:data
func NewCustomTx(tx *types.Transaction) (CustomTx, error) {
	slice := strings.Split(string(tx.Data()), ":")
	if len(slice) < 4 {
		return nil, errors.New("invalid custom tx data")
	}

	prefix, version, txType, action := slice[0], slice[1], TxType(slice[2]), slice[3]
	if prefix != "senate" {
		return nil, errors.New("invalid custom tx prefix")
	}
	if version != "1" {
		return nil, errors.New("invalid custom tx version")
	}

	types, ok := customTxTypes[txType]
	if !ok {
		return nil, errors.New("undefined custom tx type")
	}

	var data []byte
	if len(slice) > 4 {
		data = []byte(strings.Join(slice[4:], ""))
	}

	for _, typ := range types {
		if typ.Action() == action {
			t := reflect.TypeOf(typ)
			if t.Kind() == reflect.Ptr {
				t = t.Elem()
			}
			customTx := reflect.New(t).Interface().(CustomTx)
			if err := customTx.Decode(tx, data); err != nil {
				return nil, err
			}
			return customTx, nil
		}
	}
	return nil, errors.New("undefined custom tx action")
}
