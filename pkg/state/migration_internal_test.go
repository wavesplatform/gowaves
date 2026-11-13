package state

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/keyvalue"
)

func TestStateMigrationsChain(t *testing.T) {
	migrations := stateMigrations()
	if len(migrations) == 0 {
		return // the state of previous versions has to be re-imported
	}
	for i := 1; i < len(migrations); i++ {
		assert.Equal(t, migrations[i-1].from+1, migrations[i].from, "migrations must be ordered and without gaps")
	}
	last := migrations[len(migrations)-1]
	assert.Equal(t, uint16(StateVersion), last.from+1,
		"StateVersion was changed, add the migration from the previous version to stateMigrations")
}

// errTestMigrationInterrupted is returned by the test migration to simulate the interruption.
var errTestMigrationInterrupted = errors.New("interrupted")

// testMigrations returns the chain of two test migrations to StateVersion. Each migration appends its source version
// to the values of records with the wavesBalanceKeyPrefix. The second migration fails once on the key interruptKey.
func testMigrations(interruptKey []byte) []stateMigration {
	appendVersion := func(v uint16, interrupt bool) func(db *keyvalue.KeyVal) []migrationStep {
		interrupted := false
		return func(_ *keyvalue.KeyVal) []migrationStep {
			return []migrationStep{{
				name:   "test",
				prefix: wavesBalanceKeyPrefix,
				convert: func(key, data []byte) ([]byte, error) {
					if interrupt && !interrupted && string(key) == string(interruptKey) {
						interrupted = true
						return nil, errTestMigrationInterrupted
					}
					return append(append([]byte{}, data...), byte(v)), nil
				},
			}}
		}
	}
	return []stateMigration{
		{from: StateVersion - 2, steps: appendVersion(StateVersion-2, false)},
		{from: StateVersion - 1, steps: appendVersion(StateVersion-1, interruptKey != nil)},
	}
}

func createTestMigrationDB(t *testing.T, version uint16, keysNum int) (*keyvalue.KeyVal, [][]byte) {
	db, err := keyvalue.NewKeyVal(filepath.Join(t.TempDir(), keyvalueDir), defaultTestKeyValParams())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	batch, err := db.NewBatch()
	require.NoError(t, err)
	keys := make([][]byte, keysNum)
	for i := range keys {
		keys[i] = []byte{wavesBalanceKeyPrefix, byte(i >> 8), byte(i)}
		batch.Put(keys[i], []byte{0xaa})
	}
	info := stateInfo{Version: version}
	infoBytes, err := info.marshalBinary()
	require.NoError(t, err)
	batch.Put(stateInfoKeyBytes, infoBytes)
	require.NoError(t, db.Flush(batch))
	return db, keys
}

func checkTestMigrationDB(t *testing.T, db *keyvalue.KeyVal, keys [][]byte, expected []byte) {
	info, err := readStateInfoFromDB(db)
	require.NoError(t, err)
	assert.Equal(t, uint16(StateVersion), info.Version)
	_, err = db.Get([]byte{migrationProgressKeyPrefix})
	assert.ErrorIs(t, err, keyvalue.ErrNotFound)
	for _, k := range keys {
		v, gErr := db.Get(k)
		require.NoError(t, gErr)
		assert.Equal(t, expected, v, "key %x", k)
	}
}

func TestMigrateDBChain(t *testing.T) {
	db, keys := createTestMigrationDB(t, StateVersion-2, 10)
	require.NoError(t, migrateDB(db, testMigrations(nil)))
	checkTestMigrationDB(t, db, keys, []byte{0xaa, StateVersion - 2, StateVersion - 1})
	// Repeated migration does nothing.
	require.NoError(t, migrateDB(db, testMigrations(nil)))
	checkTestMigrationDB(t, db, keys, []byte{0xaa, StateVersion - 2, StateVersion - 1})
}

func TestMigrateDBChainInterrupted(t *testing.T) {
	const keysNum = migrationBatchSize + 10 // interruption after the first batch of the second migration
	db, keys := createTestMigrationDB(t, StateVersion-2, keysNum)
	migrations := testMigrations(keys[keysNum-1])
	require.ErrorIs(t, migrateDB(db, migrations), errTestMigrationInterrupted)
	// The first migration is completed, the second one is in progress.
	info, err := readStateInfoFromDB(db)
	require.NoError(t, err)
	assert.Equal(t, uint16(StateVersion-1), info.Version)
	p, err := readMigrationProgress(db, StateVersion-1)
	require.NoError(t, err)
	assert.Equal(t, 0, p.Step)
	assert.Equal(t, keys[migrationBatchSize-1], p.LastKey)
	// Resumed migration doesn't convert records twice.
	require.NoError(t, migrateDB(db, migrations))
	checkTestMigrationDB(t, db, keys, []byte{0xaa, StateVersion - 2, StateVersion - 1})
}

func TestMigrateDBProgressOfAnotherVersion(t *testing.T) {
	db, _ := createTestMigrationDB(t, StateVersion-1, 1)
	batch, err := db.NewBatch()
	require.NoError(t, err)
	require.NoError(t, flushWithProgress(db, batch, migrationProgress{Version: StateVersion - 2, Step: 1}))
	err = migrateDB(db, testMigrations(nil))
	require.ErrorContains(t, err, "found progress of migration from version")
}

func TestMigrateDBUnsupportedVersions(t *testing.T) {
	for _, v := range []uint16{StateVersion - 3, StateVersion + 1} {
		db, _ := createTestMigrationDB(t, v, 1)
		require.Error(t, migrateDB(db, testMigrations(nil)), "version %d", v)
		info, err := readStateInfoFromDB(db)
		require.NoError(t, err)
		assert.Equal(t, v, info.Version)
	}
}
