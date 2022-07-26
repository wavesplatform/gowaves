package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/util/common"
)

func TestCreateAlias(t *testing.T) {
	to, path, err := createStorageObjects(true)
	assert.NoError(t, err, "createStorageObjects() failed")

	defer func() {
		to.close(t)

		err = common.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	aliasStr := "alias"
	to.addBlock(t, blockID0)
	aliasAddr, err := proto.NewAddressFromString(addr0)
	assert.NoError(t, err, "NewAddressFromString() failed")
	inf := &aliasInfo{false, aliasAddr}
	err = to.entities.aliases.createAlias(aliasStr, inf, blockID0)
	assert.NoError(t, err, "createAlias() failed")
	addr, err := to.entities.aliases.newestAddrByAlias(aliasStr)
	assert.NoError(t, err, "newestAddrByAlias() failed")
	assert.Equal(t, aliasAddr, *addr)
	to.flush(t)
	addr, err = to.entities.aliases.addrByAlias(aliasStr)
	assert.NoError(t, err, "addrByAlias() failed")
	assert.Equal(t, aliasAddr, *addr)
}

func TestDisableStolenAliases(t *testing.T) {
	to, path, err := createStorageObjects(true)
	assert.NoError(t, err, "createStorageObjects() failed")

	defer func() {
		to.close(t)

		err = common.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	aliasStr := "alias"
	to.addBlock(t, blockID0)
	aliasAddr, err := proto.NewAddressFromString(addr0)
	assert.NoError(t, err, "NewAddressFromString() failed")
	inf := &aliasInfo{true, aliasAddr}
	err = to.entities.aliases.createAlias(aliasStr, inf, blockID0)
	assert.NoError(t, err, "createAlias() failed")
	to.flush(t)

	err = to.entities.aliases.disableStolenAliases()
	assert.NoError(t, err, "disableStolenAliases() failed")
	to.flush(t)
	disabled, err := to.entities.aliases.isDisabled(aliasStr)
	assert.NoError(t, err, "isDisabled() failed")
	assert.Equal(t, true, disabled)
	assert.Equal(t, true, to.entities.aliases.exists(aliasStr))
	_, err = to.entities.aliases.addrByAlias(aliasStr)
	assert.Equal(t, errAliasDisabled, err)
	_, err = to.entities.aliases.newestAddrByAlias(aliasStr)
	assert.Equal(t, errAliasDisabled, err)
}
