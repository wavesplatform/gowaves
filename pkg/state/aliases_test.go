package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/util"
)

type aliasesTestObjects struct {
	stor    *testStorageObjects
	aliases *aliases
}

func createAliases() (*aliasesTestObjects, []string, error) {
	stor, path, err := createStorageObjects()
	if err != nil {
		return nil, path, err
	}
	aliases, err := newAliases(stor.db, stor.dbBatch, stor.hs)
	if err != nil {
		return nil, path, err
	}
	return &aliasesTestObjects{stor, aliases}, path, nil
}

func TestCreateAlias(t *testing.T) {
	to, path, err := createAliases()
	assert.NoError(t, err, "createAliases() failed")

	defer func() {
		to.stor.close(t)

		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	aliasStr := "alias"
	to.stor.addBlock(t, blockID0)
	aliasAddr, err := proto.NewAddressFromString(addr0)
	assert.NoError(t, err, "NewAddressFromString() failed")
	inf := &aliasInfo{false, aliasAddr}
	err = to.aliases.createAlias(aliasStr, inf, blockID0)
	assert.NoError(t, err, "createAlias() failed")
	addr, err := to.aliases.newestAddrByAlias(aliasStr, true)
	assert.NoError(t, err, "newestAddrByAlias() failed")
	assert.Equal(t, aliasAddr, *addr)
	to.stor.flush(t)
	addr, err = to.aliases.addrByAlias(aliasStr, true)
	assert.NoError(t, err, "addrByAlias() failed")
	assert.Equal(t, aliasAddr, *addr)
}

func TestDisableStolenAliases(t *testing.T) {
	to, path, err := createAliases()
	assert.NoError(t, err, "createAliases() failed")

	defer func() {
		to.stor.close(t)

		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	aliasStr := "alias"
	to.stor.addBlock(t, blockID0)
	aliasAddr, err := proto.NewAddressFromString(addr0)
	assert.NoError(t, err, "NewAddressFromString() failed")
	inf := &aliasInfo{true, aliasAddr}
	err = to.aliases.createAlias(aliasStr, inf, blockID0)
	assert.NoError(t, err, "createAlias() failed")
	to.stor.flush(t)

	err = to.aliases.disableStolenAliases()
	assert.NoError(t, err, "disableStolenAlises() failed")
	to.stor.flush(t)
	disabled, err := to.aliases.isDisabled(aliasStr)
	assert.NoError(t, err, "isDisabled() failed")
	assert.Equal(t, true, disabled)
	assert.Equal(t, true, to.aliases.exists(aliasStr, true))
	_, err = to.aliases.addrByAlias(aliasStr, true)
	assert.Equal(t, errAliasDisabled, err)
	_, err = to.aliases.newestAddrByAlias(aliasStr, true)
	assert.Equal(t, errAliasDisabled, err)
}
