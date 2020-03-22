package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func createBlockDiff(blockID proto.BlockID) blockDiff {
	return blockDiff{
		minerDiff: txDiff{testGlobal.minerInfo.wavesKey: balanceDiff{minBalance: 60, balance: 60, blockID: blockID}},
		txDiffs: []txDiff{
			{
				testGlobal.minerInfo.wavesKey:     balanceDiff{minBalance: 20, balance: 20, blockID: blockID},
				testGlobal.recipientInfo.wavesKey: balanceDiff{minBalance: -50, balance: -50, leaseOut: 200, blockID: blockID},
			},
			{
				testGlobal.minerInfo.wavesKey:     balanceDiff{minBalance: 20, balance: 20, blockID: blockID},
				testGlobal.recipientInfo.wavesKey: balanceDiff{minBalance: 500, balance: 500, blockID: blockID},
				testGlobal.senderInfo.wavesKey:    balanceDiff{minBalance: -550, balance: -550, blockID: blockID},
			},
		},
	}
}

func TestSaveBlockDiff(t *testing.T) {
	diffStor, err := newDiffStorage()
	assert.NoError(t, err, "newDiffStorage() failed")
	err = diffStor.saveBlockDiff(createBlockDiff(blockID0))
	assert.NoError(t, err, "saveBlockDiff() failed")
	minerTotalDiff := balanceDiff{minBalance: 60, balance: 100, blockID: blockID0}
	minerChange := balanceChanges{
		[]byte(testGlobal.minerInfo.wavesKey),
		[]balanceDiff{minerTotalDiff},
	}
	recipientTotalDiff := balanceDiff{minBalance: -50, balance: 450, leaseOut: 200, blockID: blockID0}
	recipientChange := balanceChanges{
		[]byte(testGlobal.recipientInfo.wavesKey),
		[]balanceDiff{recipientTotalDiff},
	}
	senderTotalDiff := balanceDiff{minBalance: -550, balance: -550, blockID: blockID0}
	senderChange := balanceChanges{
		[]byte(testGlobal.senderInfo.wavesKey),
		[]balanceDiff{senderTotalDiff},
	}
	correctAllChanges := []balanceChanges{minerChange, recipientChange, senderChange}
	assert.Equal(t, correctAllChanges, diffStor.allChanges())
	// Add another block diff to inspect how diffs are appended.
	err = diffStor.saveBlockDiff(createBlockDiff(blockID1))
	assert.NoError(t, err, "saveBlockDiff() failed")
	minerTotalDiff1 := balanceDiff{minBalance: 60, balance: 200, blockID: blockID1}
	minerChange = balanceChanges{
		[]byte(testGlobal.minerInfo.wavesKey),
		[]balanceDiff{minerTotalDiff, minerTotalDiff1},
	}
	recipientTotalDiff1 := balanceDiff{minBalance: -50, balance: 900, leaseOut: 400, blockID: blockID1}
	recipientChange = balanceChanges{
		[]byte(testGlobal.recipientInfo.wavesKey),
		[]balanceDiff{recipientTotalDiff, recipientTotalDiff1},
	}
	senderTotalDiff1 := balanceDiff{minBalance: -1100, balance: -1100, blockID: blockID1}
	senderChange = balanceChanges{
		[]byte(testGlobal.senderInfo.wavesKey),
		[]balanceDiff{senderTotalDiff, senderTotalDiff1},
	}
	correctAllChanges = []balanceChanges{minerChange, recipientChange, senderChange}
	assert.Equal(t, correctAllChanges, diffStor.allChanges())
	correctGroup := []balanceChanges{senderChange, minerChange}
	group, err := diffStor.changesByKeys([]string{testGlobal.senderInfo.wavesKey, testGlobal.minerInfo.wavesKey})
	assert.NoError(t, err, "changesByKeys() failed")
	assert.Equal(t, correctGroup, group)
}
