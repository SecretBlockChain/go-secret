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

	tx = types.NewTransaction(1, address, big.NewInt(1024), 99999999, big.NewInt(1000), []byte("senate:1:event:candidate"))
	tx, err = types.SignTx(tx, types.HomesteadSigner{}, testKey)
	assert.Nil(t, err)

	ctx, err = NewTransaction(tx)
	assert.Nil(t, err)
	assert.IsType(t, new(EventBecomeCandidate), ctx)

	data := fmt.Sprintf("senate:1:event:declare:%s:yes", tx.Hash().String())
	tx = types.NewTransaction(1, address, big.NewInt(1024), 99999999, big.NewInt(1000), []byte(data))
	tx, err = types.SignTx(tx, types.HomesteadSigner{}, testKey)
	assert.Nil(t, err)

	ctx, err = NewTransaction(tx)
	assert.Nil(t, err)
	assert.IsType(t, new(Declare), ctx)
}
