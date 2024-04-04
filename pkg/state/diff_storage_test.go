package state

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state/internal"
)

func ich(v int64) internal.IntChange[int64] { return internal.NewIntChange[int64](v) }

func createBlockDiff(blockID proto.BlockID) blockDiff {
	return blockDiff{
		minerDiff: txDiff{testGlobal.minerInfo.wavesKey: balanceDiff{
			minBalance: ich(60),
			balance:    ich(60),
			blockID:    blockID,
		}},
		txDiffs: []txDiff{
			{
				testGlobal.minerInfo.wavesKey: balanceDiff{
					minBalance: ich(20),
					balance:    ich(20),
					blockID:    blockID,
				},
				testGlobal.recipientInfo.wavesKey: balanceDiff{
					minBalance: ich(-50),
					balance:    ich(-50),
					leaseOut:   ich(200),
					blockID:    blockID,
				},
			},
			{
				testGlobal.minerInfo.wavesKey: balanceDiff{
					minBalance: ich(20),
					balance:    ich(20),
					blockID:    blockID,
				},
				testGlobal.recipientInfo.wavesKey: balanceDiff{
					minBalance: ich(500),
					balance:    ich(500),
					blockID:    blockID,
				},
				testGlobal.senderInfo.wavesKey: balanceDiff{
					minBalance: ich(-550),
					balance:    ich(-550),
					blockID:    blockID,
				},
			},
		},
	}
}

func TestSaveBlockDiff(t *testing.T) {
	diffStor, err := newDiffStorage()
	assert.NoError(t, err, "newDiffStorage() failed")
	err = diffStor.saveBlockDiff(createBlockDiff(blockID0))
	assert.NoError(t, err, "saveBlockDiff() failed")
	minerTotalDiff := balanceDiff{minBalance: ich(60), balance: ich(100), blockID: blockID0}
	minerChange := balanceChanges{
		[]byte(testGlobal.minerInfo.wavesKey),
		[]balanceDiff{minerTotalDiff},
	}
	recipientTotalDiff := balanceDiff{
		minBalance: ich(-50),
		balance:    ich(450),
		leaseOut:   ich(200),
		blockID:    blockID0,
	}
	recipientChange := balanceChanges{
		[]byte(testGlobal.recipientInfo.wavesKey),
		[]balanceDiff{recipientTotalDiff},
	}
	senderTotalDiff := balanceDiff{minBalance: ich(-550), balance: ich(-550), blockID: blockID0}
	senderChange := balanceChanges{
		[]byte(testGlobal.senderInfo.wavesKey),
		[]balanceDiff{senderTotalDiff},
	}
	correctAllChanges := []balanceChanges{minerChange, recipientChange, senderChange}
	assert.Equal(t, correctAllChanges, diffStor.allChanges())
	// Add another block diff to inspect how diffs are appended.
	err = diffStor.saveBlockDiff(createBlockDiff(blockID1))
	assert.NoError(t, err, "saveBlockDiff() failed")
	minerTotalDiff1 := balanceDiff{minBalance: ich(60), balance: ich(200), blockID: blockID1}
	minerChange = balanceChanges{
		[]byte(testGlobal.minerInfo.wavesKey),
		[]balanceDiff{minerTotalDiff, minerTotalDiff1},
	}
	recipientTotalDiff1 := balanceDiff{
		minBalance: ich(-50),
		balance:    ich(900),
		leaseOut:   ich(400),
		blockID:    blockID1,
	}
	recipientChange = balanceChanges{
		[]byte(testGlobal.recipientInfo.wavesKey),
		[]balanceDiff{recipientTotalDiff, recipientTotalDiff1},
	}
	senderTotalDiff1 := balanceDiff{minBalance: ich(-1100), balance: ich(-1100), blockID: blockID1}
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
