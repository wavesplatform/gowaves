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
	invokeResults *invokeResults
}

func createInvokeResults() (*invokeResultsTestObjects, []string, error) {
	stor, path, err := createStorageObjects()
	if err != nil {
		return nil, path, err
	}
	invokeResults, err := newInvokeResults(stor.hs)
	if err != nil {
		return nil, path, err
	}
	return &invokeResultsTestObjects{stor, invokeResults}, path, nil
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
		DataEntries: []*proto.DataEntryScriptAction{
			{Entry: &proto.IntegerDataEntry{Key: "some key", Value: 12345}},
			{Entry: &proto.BooleanDataEntry{Key: "negative value", Value: false}},
			{Entry: &proto.StringDataEntry{Key: "some key", Value: "some value string"}},
			{Entry: &proto.BinaryDataEntry{Key: "k3", Value: []byte{0x24, 0x7f, 0x71, 0x14, 0x1d}}},
			{Entry: &proto.IntegerDataEntry{Key: "some key2", Value: -12345}},
			{Entry: &proto.BooleanDataEntry{Key: "negative value2", Value: true}},
			{Entry: &proto.StringDataEntry{Key: "some key143", Value: "some value2 string"}},
			{Entry: &proto.BinaryDataEntry{Key: "k5", Value: []byte{0x24, 0x7f, 0x71, 0x10, 0x1d}}},
		},
		Transfers: []*proto.TransferScriptAction{
			{Amount: 100500, Asset: *testGlobal.asset0.asset, Recipient: rcp},
			{Amount: 10, Asset: *testGlobal.asset1.asset, Recipient: rcp},
			{Amount: 0, Asset: *testGlobal.asset2.asset, Recipient: rcp},
		},
		Issues:   make([]*proto.IssueScriptAction, 0),
		Reissues: make([]*proto.ReissueScriptAction, 0),
		Burns:    make([]*proto.BurnScriptAction, 0),
	}
	err = to.invokeResults.saveResult(invokeID, savedRes, blockID0)
	assert.NoError(t, err)
	// Flush.
	to.stor.flush(t)
	res, err := to.invokeResults.invokeResult('W', invokeID, true)
	assert.NoError(t, err)
	assert.EqualValues(t, savedRes, res)
}
