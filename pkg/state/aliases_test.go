package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func TestCreateAlias(t *testing.T) {
	to := createStorageObjects(t, true)

	aliasStr := "alias"
	to.addBlock(t, blockID0)
	aliasAddr, err := proto.NewAddressFromString(addr0)
	assert.NoError(t, err, "NewAddressFromString() failed")
	err = to.entities.aliases.createAlias(aliasStr, aliasAddr, blockID0)
	assert.NoError(t, err, "createAlias() failed")
	addr, err := to.entities.aliases.newestAddrByAlias(aliasStr)
	assert.NoError(t, err, "newestAddrByAlias() failed")
	assert.Equal(t, aliasAddr, addr)
	to.flush(t)
	addr, err = to.entities.aliases.addrByAlias(aliasStr)
	assert.NoError(t, err, "addrByAlias() failed")
	assert.Equal(t, aliasAddr, addr)
}

func TestDisableStolenAliases(t *testing.T) {
	to := createStorageObjects(t, true)

	aliasStr := "alias"
	to.addBlock(t, blockID0)
	aliasAddr, err := proto.NewAddressFromString(addr0)
	assert.NoError(t, err, "NewAddressFromString() failed")
	err = to.entities.aliases.createAlias(aliasStr, aliasAddr, blockID0) // alias is new, it's not stolen
	assert.NoError(t, err, "createAlias() failed")
	to.flush(t)

	// no aliases have been stolen
	err = to.entities.aliases.disableStolenAliases(blockID0)
	assert.NoError(t, err, "disableStolenAliases() failed")
	to.flush(t)
	disabled, err := to.entities.aliases.isDisabled(aliasStr)
	assert.NoError(t, err, "isDisabled() failed")
	assert.Equal(t, false, disabled)
	assert.Equal(t, true, to.entities.aliases.exists(aliasStr))

	// steal alias
	stealAddr, err := proto.NewAddressFromString(addr1)
	assert.NoError(t, err, "NewAddressFromString() failed")
	err = to.entities.aliases.createAlias(aliasStr, stealAddr, blockID0) // steal alias
	assert.NoError(t, err, "createAlias() failed")

	// alias is stolen, but not disabled
	newestDisabled, err := to.entities.aliases.newestIsDisabled(aliasStr)
	assert.NoError(t, err, "newestIsDisabled() failed")
	assert.Equal(t, false, newestDisabled)
	disabled, err = to.entities.aliases.isDisabled(aliasStr)
	assert.NoError(t, err, "isDisabled() failed")
	assert.Equal(t, false, disabled)
	assert.Equal(t, true, to.entities.aliases.exists(aliasStr))
	to.flush(t)

	// disable stolen alias
	err = to.entities.aliases.disableStolenAliases(blockID0)
	assert.NoError(t, err, "disableStolenAliases() failed")
	// compare behaviour between newestIsDisabled and isDisabled
	newestDisabled, err = to.entities.aliases.newestIsDisabled(aliasStr)
	assert.NoError(t, err, "newestIsDisabled() failed")
	assert.Equal(t, true, newestDisabled) // already disabled
	disabled, err = to.entities.aliases.isDisabled(aliasStr)
	assert.NoError(t, err, "isDisabled() failed")
	assert.Equal(t, false, disabled) // returns false because changes are still not flushed
	to.flush(t)                      // flush disabled alias to DB in order to persist these changes

	// check that alias is disabled
	disabled, err = to.entities.aliases.isDisabled(aliasStr)
	assert.NoError(t, err, "isDisabled() failed")
	assert.Equal(t, true, disabled)
	newestDisabled, err = to.entities.aliases.newestIsDisabled(aliasStr)
	assert.NoError(t, err, "newestIsDisabled() failed")
	assert.Equal(t, true, newestDisabled)
	assert.Equal(t, true, to.entities.aliases.exists(aliasStr))

	// failed to get stolen alias
	_, err = to.entities.aliases.addrByAlias(aliasStr)
	assert.Equal(t, errAliasDisabled, err)
	_, err = to.entities.aliases.newestAddrByAlias(aliasStr)
	assert.Equal(t, errAliasDisabled, err)
}

func TestAddressToAliasesRecordRoundTrip(t *testing.T) {
	r := addressToAliasesRecord{aliases: []string{
		"lole", "keke", "fuuuf", "maha", "paha", "saha", "meha", "lole", "keke", "fuuuf", "maha", "paha", "saha", "meha",
		"lole", "keke", "fuuuf", "maha", "paha", "saha", "meha", "lole", "keke", "fuuuf", "maha", "paha", "saha", "meha",
		"lole", "keke", "fuuuf", "maha", "paha", "saha", "meha", "lole", "keke", "fuuuf", "maha", "paha", "saha", "meha",
		"lole", "keke", "fuuuf", "maha", "paha", "saha", "meha", "lole", "keke", "fuuuf", "maha", "paha", "saha", "meha",
		"lole", "keke", "fuuuf", "maha", "paha", "saha", "meha", "lole", "keke", "fuuuf", "maha", "paha", "saha", "meha",
		"lole", "keke", "fuuuf", "maha", "paha", "saha", "meha", "lole", "keke", "fuuuf", "maha", "paha", "saha", "meha",
		"lole", "keke", "fuuuf", "maha", "paha", "saha", "meha", "lole", "keke", "fuuuf", "maha", "paha", "saha", "meha",
		"lole", "keke", "fuuuf", "maha", "paha", "saha", "meha", "lole", "keke", "fuuuf", "maha", "paha", "saha", "meha",
		"lole", "keke", "fuuuf", "maha", "paha", "saha", "meha", "lole", "keke", "fuuuf", "maha", "paha", "saha", "meha",
		"lole", "keke", "fuuuf", "maha", "paha", "saha", "meha", "lole", "keke", "fuuuf", "maha", "paha", "saha", "meha",
	}}

	data, err := r.marshalBinary()
	require.NoError(t, err)

	var rr addressToAliasesRecord
	err = rr.unmarshalBinary(data)
	require.NoError(t, err)

	require.Equal(t, r, rr)
}

func TestAddressToAliasesRecord_removeIfExists(t *testing.T) {
	r := addressToAliasesRecord{aliases: []string{"lole", "keke", "fuuuf"}}

	ok := r.removeIfExists("keke")
	require.True(t, ok)
	require.Equal(t, []string{"lole", "fuuuf"}, r.aliases)

	ok = r.removeIfExists("keke")
	require.False(t, ok)
}
