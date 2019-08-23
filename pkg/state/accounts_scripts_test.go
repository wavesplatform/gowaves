package state

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/ride/evaluator/parser"
	"github.com/wavesplatform/gowaves/pkg/ride/evaluator/reader"
	"github.com/wavesplatform/gowaves/pkg/util"
)

var (
	scriptBase64 = "AgQAAAALYWxpY2VQdWJLZXkBAAAAID3+K0HJI42oXrHhtHFpHijU5PC4nn1fIFVsJp5UWrYABAAAAAlib2JQdWJLZXkBAAAAIBO1uieokBahePoeVqt4/usbhaXRq+i5EvtfsdBILNtuBAAAAAxjb29wZXJQdWJLZXkBAAAAIOfM/qkwkfi4pdngdn18n5yxNwCrBOBC3ihWaFg4gV4yBAAAAAthbGljZVNpZ25lZAMJAAH0AAAAAwgFAAAAAnR4AAAACWJvZHlCeXRlcwkAAZEAAAACCAUAAAACdHgAAAAGcHJvb2ZzAAAAAAAAAAAABQAAAAthbGljZVB1YktleQAAAAAAAAAAAQAAAAAAAAAAAAQAAAAJYm9iU2lnbmVkAwkAAfQAAAADCAUAAAACdHgAAAAJYm9keUJ5dGVzCQABkQAAAAIIBQAAAAJ0eAAAAAZwcm9vZnMAAAAAAAAAAAEFAAAACWJvYlB1YktleQAAAAAAAAAAAQAAAAAAAAAAAAQAAAAMY29vcGVyU2lnbmVkAwkAAfQAAAADCAUAAAACdHgAAAAJYm9keUJ5dGVzCQABkQAAAAIIBQAAAAJ0eAAAAAZwcm9vZnMAAAAAAAAAAAIFAAAADGNvb3BlclB1YktleQAAAAAAAAAAAQAAAAAAAAAAAAkAAGcAAAACCQAAZAAAAAIJAABkAAAAAgUAAAALYWxpY2VTaWduZWQFAAAACWJvYlNpZ25lZAUAAAAMY29vcGVyU2lnbmVkAAAAAAAAAAACqFBMLg=="
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
	accountsScripts, err := newAccountsScripts(stor.db, stor.dbBatch, stor.hs, stor.stateDB)
	if err != nil {
		return nil, path, err
	}
	return &accountsScriptsTestObjects{stor, accountsScripts}, path, nil
}

func TestSetScript(t *testing.T) {
	to, path, err := createAccountsScriptsTestObjects()
	assert.NoError(t, err, "createAccountsScriptsTestObjects() failed")

	defer func() {
		err = to.stor.stateDB.close()
		assert.NoError(t, err, "stateDB.close() failed")
		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	scriptBytes, err := base64.StdEncoding.DecodeString(scriptBase64)
	assert.NoError(t, err, "DecodeString() failed")
	correctAst, err := parser.BuildAst(reader.NewBytesReader(scriptBytes))
	assert.NoError(t, err, "BuildAst() failed")

	to.stor.addBlock(t, blockID0)
	addr := testGlobal.senderInfo.addr
	err = to.accountsScripts.setScript(addr, scriptBytes, blockID0)
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
	assert.Equal(t, correctAst, scriptAst)

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
	assert.Equal(t, correctAst, scriptAst)

	// Test stable after flushing.
	hasScript, err = to.accountsScripts.hasScript(addr, true)
	assert.NoError(t, err, "hasScript() failed")
	assert.Equal(t, true, hasScript)
	hasVerifier, err = to.accountsScripts.hasVerifier(addr, true)
	assert.NoError(t, err, "hasVerifier() failed")
	assert.Equal(t, true, hasVerifier)
	scriptAst, err = to.accountsScripts.scriptByAddr(addr, true)
	assert.NoError(t, err, "scriptByAddr() failed after flushing")
	assert.Equal(t, correctAst, scriptAst)
}
