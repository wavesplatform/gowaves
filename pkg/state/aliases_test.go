package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/util"
)

type aliasesTestObjects struct {
	stor    *storageObjects
	aliases *aliases
}

func createAliases() (*aliasesTestObjects, []string, error) {
	stor, path, err := createStorageObjects()
	if err != nil {
		return nil, path, err
	}
	aliases, err := newAliases(stor.hs)
	if err != nil {
		return nil, path, err
	}
	return &aliasesTestObjects{stor, aliases}, path, nil
}

func TestCreateAlias(t *testing.T) {
	to, path, err := createAliases()
	assert.NoError(t, err, "createAliases() failed")

	defer func() {
		err = to.stor.stateDB.close()
		assert.NoError(t, err, "stateDB.close() failed")
		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	aliasStr := "alias"
	to.stor.addBlock(t, blockID0)
	aliasAddr, err := proto.NewAddressFromString(addr0)
	assert.NoError(t, err, "NewAddressFromString() failed")
	r := &aliasRecord{aliasAddr, blockID0}
	err = to.aliases.createAlias(aliasStr, r)
	assert.NoError(t, err, "createAlias() failed")
	addr, err := to.aliases.newestAddrByAlias(aliasStr, true)
	assert.NoError(t, err, "newestAddrByAlias() failed")
	assert.Equal(t, aliasAddr, *addr)
	to.stor.flush(t)
	addr, err = to.aliases.addrByAlias(aliasStr, true)
	assert.NoError(t, err, "addrByAlias() failed")
	assert.Equal(t, aliasAddr, *addr)
}
