package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/util/common"
)

type scriptsStorageTestObjects struct {
	stor           *testStorageObjects
	scriptsStorage *scriptsStorage
}

func createScriptsStorageTestObjects() (*scriptsStorageTestObjects, []string, error) {
	stor, path, err := createStorageObjects()
	if err != nil {
		return nil, path, err
	}
	scriptsStorage, err := newScriptsStorage(stor.hs, true)
	if err != nil {
		return nil, path, err
	}
	return &scriptsStorageTestObjects{stor, scriptsStorage}, path, nil
}

func TestSetAccountScript(t *testing.T) {
	to, path, err := createScriptsStorageTestObjects()
	assert.NoError(t, err, "createScriptsStorageTestObjects() failed")

	defer func() {
		to.stor.close(t)

		err = common.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	to.stor.addBlock(t, blockID0)
	addr := testGlobal.senderInfo.addr
	err = to.scriptsStorage.setAccountScript(addr, testGlobal.scriptBytes, testGlobal.senderInfo.pk, blockID0)
	assert.NoError(t, err, "setAccountScript() failed")

	// Test newest before flushing.
	accountHasScript, err := to.scriptsStorage.newestAccountHasScript(addr, true)
	assert.NoError(t, err, "newestAccountHasScript() failed")
	assert.Equal(t, true, accountHasScript)
	accountHasVerifier, err := to.scriptsStorage.newestAccountHasVerifier(addr, true)
	assert.NoError(t, err, "newestAccountHasVerifier() failed")
	assert.Equal(t, true, accountHasVerifier)
	scriptAst, err := to.scriptsStorage.newestScriptByAddr(addr, true)
	assert.NoError(t, err, "newestScriptByAddr() failed")
	assert.Equal(t, testGlobal.scriptAst, scriptAst)

	// Test stable before flushing.
	accountHasScript, err = to.scriptsStorage.accountHasScript(addr, true)
	assert.NoError(t, err, "accountHasScript() failed")
	assert.Equal(t, false, accountHasScript)
	accountHasVerifier, err = to.scriptsStorage.accountHasVerifier(addr, true)
	assert.NoError(t, err, "accountHasVerifier() failed")
	assert.Equal(t, false, accountHasVerifier)
	_, err = to.scriptsStorage.scriptByAddr(addr, true)
	assert.Error(t, err, "scriptByAddr() did not fail before flushing")

	to.stor.flush(t)

	// Test newest after flushing.
	accountHasScript, err = to.scriptsStorage.newestAccountHasScript(addr, true)
	assert.NoError(t, err, "newestAccountHasScript() failed")
	assert.Equal(t, true, accountHasScript)
	accountHasVerifier, err = to.scriptsStorage.newestAccountHasVerifier(addr, true)
	assert.NoError(t, err, "newestAccountHasVerifier() failed")
	assert.Equal(t, true, accountHasVerifier)
	scriptAst, err = to.scriptsStorage.newestScriptByAddr(addr, true)
	assert.NoError(t, err, "newestScriptByAddr() failed")
	assert.Equal(t, testGlobal.scriptAst, scriptAst)

	// Test stable after flushing.
	accountHasScript, err = to.scriptsStorage.accountHasScript(addr, true)
	assert.NoError(t, err, "accountHasScript() failed")
	assert.Equal(t, true, accountHasScript)
	accountHasVerifier, err = to.scriptsStorage.accountHasVerifier(addr, true)
	assert.NoError(t, err, "accountHasVerifier() failed")
	assert.Equal(t, true, accountHasVerifier)
	scriptAst, err = to.scriptsStorage.scriptByAddr(addr, true)
	assert.NoError(t, err, "scriptByAddr() failed after flushing")
	assert.Equal(t, testGlobal.scriptAst, scriptAst)

	// Test discarding script.
	err = to.scriptsStorage.setAccountScript(addr, proto.Script{}, testGlobal.senderInfo.pk, blockID0)
	assert.NoError(t, err, "setAccountScript() failed")

	// Test newest before flushing.
	accountHasScript, err = to.scriptsStorage.newestAccountHasScript(addr, true)
	assert.NoError(t, err, "newestAccountHasScript() failed")
	assert.Equal(t, false, accountHasScript)
	accountHasVerifier, err = to.scriptsStorage.newestAccountHasVerifier(addr, true)
	assert.NoError(t, err, "newestAccountHasVerifier() failed")
	assert.Equal(t, false, accountHasVerifier)
	_, err = to.scriptsStorage.newestScriptByAddr(addr, true)
	assert.Error(t, err)

	// Test stable before flushing.
	accountHasScript, err = to.scriptsStorage.accountHasScript(addr, true)
	assert.NoError(t, err, "accountHasScript() failed")
	assert.Equal(t, true, accountHasScript)
	accountHasVerifier, err = to.scriptsStorage.accountHasVerifier(addr, true)
	assert.NoError(t, err, "accountHasVerifier() failed")
	assert.Equal(t, true, accountHasVerifier)
	scriptAst, err = to.scriptsStorage.scriptByAddr(addr, true)
	assert.NoError(t, err)
	assert.Equal(t, testGlobal.scriptAst, scriptAst)

	to.stor.flush(t)

	// Test newest after flushing.
	accountHasScript, err = to.scriptsStorage.newestAccountHasScript(addr, true)
	assert.NoError(t, err, "newestAccountHasScript() failed")
	assert.Equal(t, false, accountHasScript)
	accountHasVerifier, err = to.scriptsStorage.newestAccountHasVerifier(addr, true)
	assert.NoError(t, err, "newestAccountHasVerifier() failed")
	assert.Equal(t, false, accountHasVerifier)
	_, err = to.scriptsStorage.newestScriptByAddr(addr, true)
	assert.Error(t, err)

	// Test stable after flushing.
	accountHasScript, err = to.scriptsStorage.accountHasScript(addr, true)
	assert.NoError(t, err, "accountHasScript() failed")
	assert.Equal(t, false, accountHasScript)
	accountHasVerifier, err = to.scriptsStorage.accountHasVerifier(addr, true)
	assert.NoError(t, err, "accountHasVerifier() failed")
	assert.Equal(t, false, accountHasVerifier)
	_, err = to.scriptsStorage.scriptByAddr(addr, true)
	assert.Error(t, err)
}

func TestSetAssetScript(t *testing.T) {
	to, path, err := createScriptsStorageTestObjects()
	assert.NoError(t, err, "createScriptsStorageTestObjects() failed")

	defer func() {
		to.stor.close(t)

		err = common.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	to.stor.addBlock(t, blockID0)
	assetID := testGlobal.asset0.asset.ID
	err = to.scriptsStorage.setAssetScript(assetID, testGlobal.scriptBytes, testGlobal.senderInfo.pk, blockID0)
	assert.NoError(t, err, "setAssetScript() failed")

	// Test newest before flushing.
	isSmartAsset, err := to.scriptsStorage.newestIsSmartAsset(assetID, true)
	assert.NoError(t, err)
	assert.Equal(t, true, isSmartAsset)
	scriptAst, err := to.scriptsStorage.newestScriptByAsset(assetID, true)
	assert.NoError(t, err, "newestScriptByAsset() failed")
	assert.Equal(t, testGlobal.scriptAst, scriptAst)

	// Test stable before flushing.
	isSmartAsset, err = to.scriptsStorage.isSmartAsset(assetID, true)
	assert.NoError(t, err, "isSmartAsset() failed")
	assert.Equal(t, false, isSmartAsset)
	_, err = to.scriptsStorage.scriptByAsset(assetID, true)
	assert.Error(t, err, "scriptByAsset() did not fail before flushing")

	to.stor.flush(t)

	// Test newest after flushing.
	isSmartAsset, err = to.scriptsStorage.newestIsSmartAsset(assetID, true)
	assert.NoError(t, err)
	assert.Equal(t, true, isSmartAsset)
	scriptAst, err = to.scriptsStorage.newestScriptByAsset(assetID, true)
	assert.NoError(t, err, "newestScriptByAsset() failed")
	assert.Equal(t, testGlobal.scriptAst, scriptAst)

	// Test stable after flushing.
	isSmartAsset, err = to.scriptsStorage.isSmartAsset(assetID, true)
	assert.NoError(t, err, "isSmartAsset() failed")
	assert.Equal(t, true, isSmartAsset)
	scriptAst, err = to.scriptsStorage.scriptByAsset(assetID, true)
	assert.NoError(t, err, "scriptByAsset() failed after flushing")
	assert.Equal(t, testGlobal.scriptAst, scriptAst)

	// Test discarding script.
	err = to.scriptsStorage.setAssetScript(assetID, proto.Script{}, testGlobal.senderInfo.pk, blockID0)
	assert.NoError(t, err, "setAssetScript() failed")

	// Test newest before flushing.
	isSmartAsset, err = to.scriptsStorage.newestIsSmartAsset(assetID, true)
	assert.NoError(t, err)
	assert.Equal(t, false, isSmartAsset)
	_, err = to.scriptsStorage.newestScriptByAsset(assetID, true)
	assert.Error(t, err)

	// Test stable before flushing.
	isSmartAsset, err = to.scriptsStorage.isSmartAsset(assetID, true)
	assert.NoError(t, err, "isSmartAsset() failed")
	assert.Equal(t, true, isSmartAsset)
	scriptAst, err = to.scriptsStorage.scriptByAsset(assetID, true)
	assert.NoError(t, err)
	assert.Equal(t, testGlobal.scriptAst, scriptAst)

	to.stor.flush(t)

	// Test newest after flushing.
	isSmartAsset, err = to.scriptsStorage.newestIsSmartAsset(assetID, true)
	assert.NoError(t, err)
	assert.Equal(t, false, isSmartAsset)
	_, err = to.scriptsStorage.newestScriptByAsset(assetID, true)
	assert.Error(t, err)

	// Test stable after flushing.
	isSmartAsset, err = to.scriptsStorage.isSmartAsset(assetID, true)
	assert.NoError(t, err, "isSmartAsset() failed")
	assert.Equal(t, false, isSmartAsset)
	_, err = to.scriptsStorage.scriptByAsset(assetID, true)
	assert.Error(t, err)

	// Test uncertain.
	to.scriptsStorage.setAssetScriptUncertain(assetID, testGlobal.scriptBytes, testGlobal.senderInfo.pk)
	isSmartAsset, err = to.scriptsStorage.newestIsSmartAsset(assetID, true)
	assert.NoError(t, err)
	assert.Equal(t, true, isSmartAsset)
	scriptAst, err = to.scriptsStorage.newestScriptByAsset(assetID, true)
	assert.NoError(t, err, "newestScriptByAsset() failed")
	assert.Equal(t, testGlobal.scriptAst, scriptAst)
	to.scriptsStorage.dropUncertain()
	_, err = to.scriptsStorage.newestScriptByAsset(assetID, true)
	assert.Error(t, err)
	isSmartAsset, err = to.scriptsStorage.newestIsSmartAsset(assetID, true)
	assert.NoError(t, err)
	assert.Equal(t, false, isSmartAsset)
	// Test after commit.
	to.scriptsStorage.setAssetScriptUncertain(assetID, testGlobal.scriptBytes, testGlobal.senderInfo.pk)
	err = to.scriptsStorage.commitUncertain(blockID0)
	assert.NoError(t, err, "commitUncertain() failed")
	to.scriptsStorage.dropUncertain()
	isSmartAsset, err = to.scriptsStorage.newestIsSmartAsset(assetID, true)
	assert.NoError(t, err)
	assert.Equal(t, true, isSmartAsset)
	scriptAst, err = to.scriptsStorage.newestScriptByAsset(assetID, true)
	assert.NoError(t, err, "newestScriptByAsset() failed")
	assert.Equal(t, testGlobal.scriptAst, scriptAst)
	// Test after flush.
	to.stor.flush(t)
	isSmartAsset, err = to.scriptsStorage.isSmartAsset(assetID, true)
	assert.NoError(t, err, "isSmartAsset() failed")
	assert.Equal(t, true, isSmartAsset)
	scriptAst, err = to.scriptsStorage.scriptByAsset(assetID, true)
	assert.NoError(t, err)
	assert.Equal(t, testGlobal.scriptAst, scriptAst)
}
