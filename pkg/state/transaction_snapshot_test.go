package state

import (
	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"testing"
)

// TODO send only txBalanceChanges to perfomer
func TestIssueTransactionSnapshot(t *testing.T) {
	to := createDifferTestObjects(t)

	to.stor.addBlock(t, blockID0)
	to.stor.activateFeature(t, int16(settings.NG))
	err := to.stor.entities.balances.setWavesBalance(testGlobal.senderInfo.addr.ID(), &wavesValue{profile: balanceProfile{balance: 1000 * FeeUnit * 3}}, blockID0)
	assert.NoError(t, err, "failed to sign issue tx")
	tx := proto.NewUnsignedIssueWithSig(testGlobal.senderInfo.pk, "asset0", "description", defaultQuantity, defaultDecimals, true, defaultTimestamp, uint64(1000*FeeUnit))
	err = tx.Sign(proto.TestNetScheme, testGlobal.senderInfo.sk)
	assert.NoError(t, err, "failed to sign issue tx")

	ch, err := to.td.createDiffIssueWithSig(tx, defaultDifferInfo())
	assert.NoError(t, err, "createDiffIssueWithSig() failed")
	applicationRes := &applicationResult{true, 0, ch}
	transactionSnapshot, err := to.tp.performIssueWithSig(tx, defaultPerformerInfo(), nil, applicationRes)
	assert.NoError(t, err, "performIssueWithProofs() failed")
	to.stor.flush(t)
	assert.NotNil(t, transactionSnapshot)
}
