package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestSaveEmptyInvokeResult(t *testing.T) {
	to, path, err := createInvokeResults()
	require.NoError(t, err, "createInvokeResults() failed")
	defer func() {
		to.stor.close(t)

		err = common.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()
	invokeID := crypto.MustDigestFromBase58(invokeId)
	to.stor.addBlock(t, blockID0)
	savedRes, err := proto.NewScriptResult(nil, proto.ScriptErrorMessage{})
	require.NoError(t, err)
	err = to.invokeResults.saveResult(invokeID, savedRes, blockID0)
	require.NoError(t, err)
	// Flush.
	to.stor.flush(t)
	res, err := to.invokeResults.invokeResult('W', invokeID, true)
	require.NoError(t, err)
	assert.EqualValues(t, savedRes, res)
}

func TestSaveResult(t *testing.T) {
	to, path, err := createInvokeResults()
	require.NoError(t, err, "createInvokeResults() failed")

	defer func() {
		to.stor.close(t)

		err = common.CleanTemporaryDirs(path)
		require.NoError(t, err, "failed to clean test data dirs")
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
		Issues: []*proto.IssueScriptAction{
			{ID: testGlobal.asset0.asset.ID, Name: "asset0", Description: "description0", Quantity: 12345, Decimals: 6, Reissuable: false, Script: []byte{}, Nonce: 7890},
			{ID: testGlobal.asset1.asset.ID, Name: "asset1", Description: "description1", Quantity: 9876, Decimals: 5, Reissuable: true, Script: []byte{}, Nonce: 4321},
		},
		Reissues: []*proto.ReissueScriptAction{
			{AssetID: testGlobal.asset0.asset.ID, Quantity: 1234567890, Reissuable: false},
			{AssetID: testGlobal.asset1.asset.ID, Quantity: 987654321, Reissuable: true},
		},
		Burns: []*proto.BurnScriptAction{
			{AssetID: testGlobal.asset0.asset.ID, Quantity: 1234567890},
			{AssetID: testGlobal.asset1.asset.ID, Quantity: 9877654321},
		},
		Sponsorships: []*proto.SponsorshipScriptAction{
			{AssetID: testGlobal.asset0.asset.ID, MinFee: 12345},
			{AssetID: testGlobal.asset0.asset.ID, MinFee: 0},
		},
	}
	err = to.invokeResults.saveResult(invokeID, savedRes, blockID0)
	require.NoError(t, err)
	to.stor.flush(t)
	res, err := to.invokeResults.invokeResult('W', invokeID, true)
	require.NoError(t, err)
	assert.EqualValues(t, savedRes, res)
}
