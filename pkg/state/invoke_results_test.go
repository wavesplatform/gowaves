package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/util/common"
)

type invokeResultsTestObjects struct {
	stor          *testStorageObjects
	aliases       *aliases
	invokeResults *invokeResults
}

func createInvokeResults() (*invokeResultsTestObjects, []string, error) {
	stor, path, err := createStorageObjects()
	if err != nil {
		return nil, path, err
	}
	aliases, err := newAliases(stor.db, stor.dbBatch, stor.hs)
	if err != nil {
		return nil, path, err
	}
	invokeResults, err := newInvokeResults(stor.hs, aliases)
	if err != nil {
		return nil, path, err
	}
	return &invokeResultsTestObjects{stor, aliases, invokeResults}, path, nil
}

func TestSaveResult(t *testing.T) {
	to, path, err := createInvokeResults()
	assert.NoError(t, err, "createInvokeResults() failed")

	defer func() {
		to.stor.close(t)

		err = common.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	rcp := proto.NewRecipientFromAddress(testGlobal.senderInfo.addr)

	invokeID := crypto.MustDigestFromBase58(invokeId)
	to.stor.addBlock(t, blockID0)
	savedRes := &proto.ScriptResult{
		Writes: []proto.DataEntry{
			&proto.IntegerDataEntry{Key: "some key", Value: 12345},
			&proto.BooleanDataEntry{Key: "negative value", Value: false},
			&proto.StringDataEntry{Key: "some key", Value: "some value string"},
			&proto.BinaryDataEntry{Key: "k3", Value: []byte{0x24, 0x7f, 0x71, 0x14, 0x1d}},
			&proto.IntegerDataEntry{Key: "some key2", Value: -12345},
			&proto.BooleanDataEntry{Key: "negative value2", Value: true},
			&proto.StringDataEntry{Key: "some key143", Value: "some value2 string"},
			&proto.BinaryDataEntry{Key: "k5", Value: []byte{0x24, 0x7f, 0x71, 0x10, 0x1d}},
		},
		Transfers: []proto.ScriptResultTransfer{
			{Amount: 100500, Asset: *testGlobal.asset0.asset, Recipient: rcp},
			{Amount: -10, Asset: *testGlobal.asset1.asset, Recipient: rcp},
			{Amount: 0, Asset: *testGlobal.asset2.asset, Recipient: rcp},
		},
	}
	err = to.invokeResults.saveResult(invokeID, savedRes, blockID0)
	assert.NoError(t, err)
	// Flush.
	to.stor.flush(t)
	res, err := to.invokeResults.invokeResult(invokeID, true)
	assert.NoError(t, err)
	assert.Equal(t, savedRes, res)
}
