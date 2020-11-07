package senate

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/SecretBlockChain/go-secret/common"
	"github.com/SecretBlockChain/go-secret/core/types"
	"github.com/SecretBlockChain/go-secret/crypto"
	"github.com/stretchr/testify/assert"
)

var testKey, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")

func TestCustomTransactionDecode(t *testing.T) {
	address := common.HexToAddress("0x47746e8acb5dafe9c00b7195d0c2d830fcc04910")

	tx := types.NewTransaction(1, address, big.NewInt(1024), 99999999, big.NewInt(1000), []byte("senate:1:event:delegate"))
	tx, err := types.SignTx(tx, types.HomesteadSigner{}, testKey)
	assert.Nil(t, err)

	ctx, err := NewTransaction(tx)
	assert.Nil(t, err)
	assert.IsType(t, new(EventDelegate), ctx)

	tx = types.NewTransaction(1, address, big.NewInt(1024), 99999999, big.NewInt(1000), []byte("senate:1:event:candidate"))
	tx, err = types.SignTx(tx, types.HomesteadSigner{}, testKey)
	assert.Nil(t, err)

	ctx, err = NewTransaction(tx)
	assert.Nil(t, err)
	assert.IsType(t, new(EventBecomeCandidate), ctx)

	proposals := [][]byte{
		[]byte("senate:1:event:proposal:period:8"),
		[]byte("senate:1:event:proposal:epoch:86400"),
		[]byte("senate:1:event:proposal:maxValidatorsCount:21"),
		[]byte("senate:1:event:proposal:minDelegatorBalance:0xde0b6b3a7640000"),
		[]byte("senate:1:event:proposal:minCandidateBalance:0x56bc75e2d63100000"),
		[]byte("senate:1:event:proposal:rewards:0x69e10de76676d0800000:0x4563918244f40000,0x13da329b6336471800000:0x1bc16d674ec80000,0x422ca8b0a00a425000000:0xde0b6b3a7640000"),
	}
	for _, proposal := range proposals {
		tx = types.NewTransaction(1, address, big.NewInt(1024), 99999999, big.NewInt(1000), proposal)
		tx, err = types.SignTx(tx, types.HomesteadSigner{}, testKey)
		assert.Nil(t, err)

		ctx, err = NewTransaction(tx)
		assert.Nil(t, err)
		assert.IsType(t, new(Proposal), ctx)
	}

	data := fmt.Sprintf("senate:1:event:declare:%s:yes", tx.Hash().String())
	tx = types.NewTransaction(1, address, big.NewInt(1024), 99999999, big.NewInt(1000), []byte(data))
	tx, err = types.SignTx(tx, types.HomesteadSigner{}, testKey)
	assert.Nil(t, err)

	ctx, err = NewTransaction(tx)
	assert.Nil(t, err)
	assert.IsType(t, new(Declare), ctx)
}
