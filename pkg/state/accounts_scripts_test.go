package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/util"
)

type accountsScriptsTestObjects struct {
	stor            *testStorageObjects
	accountsScripts *accountsScripts
}

func createAccountsScriptsTestObjects() (*accountsScriptsTestObjects, []string, error) {
	stor, path, err := createStorageObjects()
	if err != nil {
		return nil, path, err
	}
	accountsScripts, err := newAccountsScripts(stor.db, stor.dbBatch, stor.hs)
	if err != nil {
		return nil, path, err
	}
	return &accountsScriptsTestObjects{stor, accountsScripts}, path, nil
}

func TestSetScript(t *testing.T) {
	to, path, err := createAccountsScriptsTestObjects()
	assert.NoError(t, err, "createAccountsScriptsTestObjects() failed")

	defer func() {
		to.stor.close(t)

		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	to.stor.addBlock(t, blockID0)
	addr := testGlobal.senderInfo.addr
	err = to.accountsScripts.setScript(addr, proto.Script(testGlobal.scriptBytes), blockID0)
	assert.NoError(t, err, "setScript() failed")

	// Test newest before flushing.
	hasScript, err := to.accountsScripts.newestHasScript(addr, true)
	assert.NoError(t, err, "newestHasScript() failed")
	assert.Equal(t, true, hasScript)
	hasVerifier, err := to.accountsScripts.newestHasVerifier(addr, true)
	assert.NoError(t, err, "newestHasVerifier() failed")
	assert.Equal(t, true, hasVerifier)
	scriptAst, err := to.accountsScripts.newestScriptByAddr(addr, true)
	assert.NoError(t, err, "newestScriptByAddr() failed")
	assert.Equal(t, testGlobal.scriptAst, scriptAst)

	// Test stable before flushing.
	hasScript, err = to.accountsScripts.hasScript(addr, true)
	assert.NoError(t, err, "hasScript() failed")
	assert.Equal(t, false, hasScript)
	hasVerifier, err = to.accountsScripts.hasVerifier(addr, true)
	assert.NoError(t, err, "hasVerifier() failed")
	assert.Equal(t, false, hasVerifier)
	_, err = to.accountsScripts.scriptByAddr(addr, true)
	assert.Error(t, err, "scriptByAddr() did not fail before flushing")

	to.stor.flush(t)

	// Test newest after flushing.
	hasScript, err = to.accountsScripts.newestHasScript(addr, true)
	assert.NoError(t, err, "newestHasScript() failed")
	assert.Equal(t, true, hasScript)
	hasVerifier, err = to.accountsScripts.newestHasVerifier(addr, true)
	assert.NoError(t, err, "newestHasVerifier() failed")
	assert.Equal(t, true, hasVerifier)
	scriptAst, err = to.accountsScripts.newestScriptByAddr(addr, true)
	assert.NoError(t, err, "newestScriptByAddr() failed")
	assert.Equal(t, testGlobal.scriptAst, scriptAst)

	// Test stable after flushing.
	hasScript, err = to.accountsScripts.hasScript(addr, true)
	assert.NoError(t, err, "hasScript() failed")
	assert.Equal(t, true, hasScript)
	hasVerifier, err = to.accountsScripts.hasVerifier(addr, true)
	assert.NoError(t, err, "hasVerifier() failed")
	assert.Equal(t, true, hasVerifier)
	scriptAst, err = to.accountsScripts.scriptByAddr(addr, true)
	assert.NoError(t, err, "scriptByAddr() failed after flushing")
	assert.Equal(t, testGlobal.scriptAst, scriptAst)
}
