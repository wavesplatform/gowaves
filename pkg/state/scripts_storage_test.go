package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type scriptsStorageTestObjects struct {
	stor           *testStorageObjects
	scriptsStorage *scriptsStorage
}

func createScriptsStorageTestObjects(t *testing.T) *scriptsStorageTestObjects {
	stor := createStorageObjects(t, true)
	scriptsStorage, err := newScriptsStorage(stor.hs, proto.TestNetScheme, true)
	require.NoError(t, err)
	return &scriptsStorageTestObjects{stor, scriptsStorage}
}

func TestSetAccountScript(t *testing.T) {
	to := createScriptsStorageTestObjects(t)

	to.stor.addBlock(t, blockID0)
	addr := testGlobal.senderInfo.addr
	err := to.scriptsStorage.setAccountScript(addr, testGlobal.scriptBytes, testGlobal.senderInfo.pk, blockID0)
	assert.NoError(t, err, "setAccountScript() failed")

	// Test newest before flushing.
	accountHasScript, err := to.scriptsStorage.newestAccountHasScript(addr)
	assert.NoError(t, err, "newestAccountHasScript() failed")
	assert.Equal(t, true, accountHasScript)
	accountHasVerifier, err := to.scriptsStorage.newestAccountHasVerifier(addr)
	assert.NoError(t, err, "newestAccountHasVerifier() failed")
	assert.Equal(t, true, accountHasVerifier)
	scriptAst, err := to.scriptsStorage.newestScriptByAddr(addr)
	assert.NoError(t, err, "newestScriptByAddr() failed")
	assert.Equal(t, testGlobal.scriptAst, scriptAst)

	// Test stable before flushing.
	accountHasScript, err = to.scriptsStorage.accountHasScript(addr)
	assert.NoError(t, err, "accountHasScript() failed")
	assert.Equal(t, false, accountHasScript)
	accountHasVerifier, err = to.scriptsStorage.accountHasVerifier(addr)
	assert.NoError(t, err, "accountHasVerifier() failed")
	assert.Equal(t, false, accountHasVerifier)
	_, err = to.scriptsStorage.scriptByAddr(addr)
	assert.Error(t, err, "scriptByAddr() did not fail before flushing")

	to.stor.flush(t)

	// Test newest after flushing.
	accountHasScript, err = to.scriptsStorage.newestAccountHasScript(addr)
	assert.NoError(t, err, "newestAccountHasScript() failed")
	assert.Equal(t, true, accountHasScript)
	accountHasVerifier, err = to.scriptsStorage.newestAccountHasVerifier(addr)
	assert.NoError(t, err, "newestAccountHasVerifier() failed")
	assert.Equal(t, true, accountHasVerifier)
	scriptAst, err = to.scriptsStorage.newestScriptByAddr(addr)
	assert.NoError(t, err, "newestScriptByAddr() failed")
	assert.Equal(t, testGlobal.scriptAst, scriptAst)

	// Test stable after flushing.
	accountHasScript, err = to.scriptsStorage.accountHasScript(addr)
	assert.NoError(t, err, "accountHasScript() failed")
	assert.Equal(t, true, accountHasScript)
	accountHasVerifier, err = to.scriptsStorage.accountHasVerifier(addr)
	assert.NoError(t, err, "accountHasVerifier() failed")
	assert.Equal(t, true, accountHasVerifier)
	scriptAst, err = to.scriptsStorage.scriptByAddr(addr)
	assert.NoError(t, err, "scriptByAddr() failed after flushing")
	assert.Equal(t, testGlobal.scriptAst, scriptAst)

	// Test discarding script.
	err = to.scriptsStorage.setAccountScript(addr, proto.Script{}, testGlobal.senderInfo.pk, blockID0)
	assert.NoError(t, err, "setAccountScript() failed")

	// Test newest before flushing.
	accountHasScript, err = to.scriptsStorage.newestAccountHasScript(addr)
	assert.NoError(t, err, "newestAccountHasScript() failed")
	assert.Equal(t, false, accountHasScript)
	accountHasVerifier, err = to.scriptsStorage.newestAccountHasVerifier(addr)
	assert.NoError(t, err, "newestAccountHasVerifier() failed")
	assert.Equal(t, false, accountHasVerifier)
	_, err = to.scriptsStorage.newestScriptByAddr(addr)
	assert.Error(t, err)

	// Test stable before flushing.
	accountHasScript, err = to.scriptsStorage.accountHasScript(addr)
	assert.NoError(t, err, "accountHasScript() failed")
	assert.Equal(t, true, accountHasScript)
	accountHasVerifier, err = to.scriptsStorage.accountHasVerifier(addr)
	assert.NoError(t, err, "accountHasVerifier() failed")
	assert.Equal(t, true, accountHasVerifier)
	scriptAst, err = to.scriptsStorage.scriptByAddr(addr)
	assert.NoError(t, err)
	assert.Equal(t, testGlobal.scriptAst, scriptAst)

	to.stor.flush(t)

	// Test newest after flushing.
	accountHasScript, err = to.scriptsStorage.newestAccountHasScript(addr)
	assert.NoError(t, err, "newestAccountHasScript() failed")
	assert.Equal(t, false, accountHasScript)
	accountHasVerifier, err = to.scriptsStorage.newestAccountHasVerifier(addr)
	assert.NoError(t, err, "newestAccountHasVerifier() failed")
	assert.Equal(t, false, accountHasVerifier)
	_, err = to.scriptsStorage.newestScriptByAddr(addr)
	assert.Error(t, err)

	// Test stable after flushing.
	accountHasScript, err = to.scriptsStorage.accountHasScript(addr)
	assert.NoError(t, err, "accountHasScript() failed")
	assert.Equal(t, false, accountHasScript)
	accountHasVerifier, err = to.scriptsStorage.accountHasVerifier(addr)
	assert.NoError(t, err, "accountHasVerifier() failed")
	assert.Equal(t, false, accountHasVerifier)
	_, err = to.scriptsStorage.scriptByAddr(addr)
	assert.Error(t, err)
}

func TestSetAssetScript(t *testing.T) {
	to := createScriptsStorageTestObjects(t)

	to.stor.addBlock(t, blockID0)

	fullAssetID := testGlobal.asset0.asset.ID
	shortAssetID := proto.AssetIDFromDigest(fullAssetID)

	err := to.scriptsStorage.setAssetScript(fullAssetID, testGlobal.scriptBytes, testGlobal.senderInfo.pk, blockID0)
	assert.NoError(t, err, "setAssetScript() failed")

	// Test newest before flushing.
	isSmartAsset, err := to.scriptsStorage.newestIsSmartAsset(shortAssetID)
	assert.NoError(t, err)
	assert.Equal(t, true, isSmartAsset)
	scriptAst, err := to.scriptsStorage.newestScriptByAsset(shortAssetID)
	assert.NoError(t, err, "newestScriptByAsset() failed")
	assert.Equal(t, testGlobal.scriptAst, scriptAst)

	// Test stable before flushing.
	isSmartAsset, err = to.scriptsStorage.isSmartAsset(shortAssetID)
	assert.NoError(t, err, "isSmartAsset() failed")
	assert.Equal(t, false, isSmartAsset)
	_, err = to.scriptsStorage.scriptByAsset(shortAssetID)
	assert.Error(t, err, "scriptByAsset() did not fail before flushing")

	to.stor.flush(t)

	// Test newest after flushing.
	isSmartAsset, err = to.scriptsStorage.newestIsSmartAsset(shortAssetID)
	assert.NoError(t, err)
	assert.Equal(t, true, isSmartAsset)
	scriptAst, err = to.scriptsStorage.newestScriptByAsset(shortAssetID)
	assert.NoError(t, err, "newestScriptByAsset() failed")
	assert.Equal(t, testGlobal.scriptAst, scriptAst)

	// Test stable after flushing.
	isSmartAsset, err = to.scriptsStorage.isSmartAsset(shortAssetID)
	assert.NoError(t, err, "isSmartAsset() failed")
	assert.Equal(t, true, isSmartAsset)
	scriptAst, err = to.scriptsStorage.scriptByAsset(shortAssetID)
	assert.NoError(t, err, "scriptByAsset() failed after flushing")
	assert.Equal(t, testGlobal.scriptAst, scriptAst)

	// Test discarding script.
	err = to.scriptsStorage.setAssetScript(fullAssetID, proto.Script{}, testGlobal.senderInfo.pk, blockID0)
	assert.NoError(t, err, "setAssetScript() failed")

	// Test newest before flushing.
	isSmartAsset, err = to.scriptsStorage.newestIsSmartAsset(shortAssetID)
	assert.NoError(t, err)
	assert.Equal(t, false, isSmartAsset)
	_, err = to.scriptsStorage.newestScriptByAsset(shortAssetID)
	assert.Error(t, err)

	// Test stable before flushing.
	isSmartAsset, err = to.scriptsStorage.isSmartAsset(shortAssetID)
	assert.NoError(t, err, "isSmartAsset() failed")
	assert.Equal(t, true, isSmartAsset)
	scriptAst, err = to.scriptsStorage.scriptByAsset(shortAssetID)
	assert.NoError(t, err)
	assert.Equal(t, testGlobal.scriptAst, scriptAst)

	to.stor.flush(t)

	// Test newest after flushing.
	isSmartAsset, err = to.scriptsStorage.newestIsSmartAsset(shortAssetID)
	assert.NoError(t, err)
	assert.Equal(t, false, isSmartAsset)
	_, err = to.scriptsStorage.newestScriptByAsset(shortAssetID)
	assert.Error(t, err)

	// Test stable after flushing.
	isSmartAsset, err = to.scriptsStorage.isSmartAsset(shortAssetID)
	assert.NoError(t, err, "isSmartAsset() failed")
	assert.Equal(t, false, isSmartAsset)
	_, err = to.scriptsStorage.scriptByAsset(shortAssetID)
	assert.Error(t, err)

	// Test uncertain with empty script.
	err = to.scriptsStorage.setAssetScriptUncertain(fullAssetID, proto.Script{}, testGlobal.senderInfo.pk)
	assert.NoError(t, err)
	isSmartAsset, err = to.scriptsStorage.newestIsSmartAsset(shortAssetID)
	assert.NoError(t, err)
	assert.Equal(t, false, isSmartAsset)
	_, err = to.scriptsStorage.newestScriptByAsset(shortAssetID)
	assert.EqualError(t, err, proto.ErrNotFound.Error(), "newestScriptByAsset() failed")
	to.scriptsStorage.dropUncertain()
	_, err = to.scriptsStorage.newestScriptByAsset(shortAssetID)
	assert.EqualError(t, err, proto.ErrNotFound.Error(), "newestScriptByAsset() failed")
	isSmartAsset, err = to.scriptsStorage.newestIsSmartAsset(shortAssetID)
	assert.NoError(t, err)
	assert.Equal(t, false, isSmartAsset)
	// Test uncertain.
	err = to.scriptsStorage.setAssetScriptUncertain(fullAssetID, testGlobal.scriptBytes, testGlobal.senderInfo.pk)
	assert.NoError(t, err)
	isSmartAsset, err = to.scriptsStorage.newestIsSmartAsset(shortAssetID)
	assert.NoError(t, err)
	assert.Equal(t, true, isSmartAsset)
	scriptAst, err = to.scriptsStorage.newestScriptByAsset(shortAssetID)
	assert.NoError(t, err, "newestScriptByAsset() failed")
	assert.Equal(t, testGlobal.scriptAst, scriptAst)
	to.scriptsStorage.dropUncertain()
	_, err = to.scriptsStorage.newestScriptByAsset(shortAssetID)
	assert.EqualError(t, err, proto.ErrNotFound.Error(), "newestScriptByAsset() failed")
	isSmartAsset, err = to.scriptsStorage.newestIsSmartAsset(shortAssetID)
	assert.NoError(t, err)
	assert.Equal(t, false, isSmartAsset)
	// Test after commit.
	err = to.scriptsStorage.setAssetScriptUncertain(fullAssetID, testGlobal.scriptBytes, testGlobal.senderInfo.pk)
	assert.NoError(t, err)
	err = to.scriptsStorage.commitUncertain(blockID0)
	assert.NoError(t, err, "commitUncertain() failed")
	to.scriptsStorage.dropUncertain()
	isSmartAsset, err = to.scriptsStorage.newestIsSmartAsset(shortAssetID)
	assert.NoError(t, err)
	assert.Equal(t, true, isSmartAsset)
	scriptAst, err = to.scriptsStorage.newestScriptByAsset(shortAssetID)
	assert.NoError(t, err, "newestScriptByAsset() failed")
	assert.Equal(t, testGlobal.scriptAst, scriptAst)
	// Test after flush.
	to.stor.flush(t)
	isSmartAsset, err = to.scriptsStorage.isSmartAsset(shortAssetID)
	assert.NoError(t, err, "isSmartAsset() failed")
	assert.Equal(t, true, isSmartAsset)
	scriptAst, err = to.scriptsStorage.scriptByAsset(shortAssetID)
	assert.NoError(t, err)
	assert.Equal(t, testGlobal.scriptAst, scriptAst)
}
