package state

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

// referenceV27HistoryEncoding encodes the history in the format of version 27 independently of marshalV27History:
// the size of each entry of variable size history is stored as 4 bytes big-endian uint32.
func referenceV27HistoryEncoding(entity blockchainEntity, fixed bool, entries []historyEntry) []byte {
	res := []byte{byte(entity)}
	for _, e := range entries {
		if !fixed {
			res = binary.BigEndian.AppendUint32(res, uint32(len(e.data)+historyBlockNumSize)) // #nosec G115 // test.
		}
		res = append(res, e.data...)
		res = binary.BigEndian.AppendUint32(res, e.blockNum)
	}
	return res
}

func variableSizeEntities() []blockchainEntity {
	var res []blockchainEntity
	for _, e := range allBlockchainEntities() {
		if p, _ := entityProperties(e); !p.fixedSize {
			res = append(res, e)
		}
	}
	return res
}

func TestConvertV27HistoryAllEntities(t *testing.T) {
	for _, e := range allBlockchainEntities() {
		p, ok := entityProperties(e)
		require.True(t, ok)
		for _, n := range []int{1, 2, 10} {
			entries := testHistoryEntries(e, p, n)
			v27 := referenceV27HistoryEncoding(e, p.fixedSize, entries)
			converted, err := convertV27History(v27)
			require.NoError(t, err, "entity %d", e)
			if p.fixedSize {
				assert.Equal(t, v27, converted, "fixed size history of entity %d must be unchanged", e)
				continue
			}
			expected := referenceHistoryRecordEncoding(e, false, entries)
			assert.Equal(t, expected, converted, "entity %d", e)
			assert.Less(t, len(converted), len(v27), "entity %d", e)
			// Conversion is idempotent.
			again, err := convertV27History(converted)
			require.NoError(t, err)
			assert.Equal(t, converted, again, "entity %d", e)
			// The old decoder and encoder round trip.
			hr, err := decodeV27History(v27)
			require.NoError(t, err)
			assert.Equal(t, e, hr.entityType)
			require.Len(t, hr.entries, n)
			for i := range entries {
				assert.Equal(t, entries[i].blockNum, hr.entries[i].blockNum)
				assert.True(t, bytes.Equal(entries[i].data, hr.entries[i].data))
			}
			encoded, err := marshalV27History(hr)
			require.NoError(t, err)
			assert.Equal(t, v27, encoded)
		}
	}
}

func TestConvertV27HistoryByEntityTypeOfRecord(t *testing.T) {
	// Histories of addresses to aliases are written with the entity type of score, the conversion uses the
	// entity type stored in the record.
	entries := []historyEntry{{data: []byte{1, 2, 3}, blockNum: 7}}
	converted, err := convertV27History(referenceV27HistoryEncoding(score, false, entries))
	require.NoError(t, err)
	assert.Equal(t, referenceHistoryRecordEncoding(score, false, entries), converted)
}

func TestConvertV27HistoryEmpty(t *testing.T) {
	data := []byte{byte(dataEntry)}
	converted, err := convertV27History(data)
	require.NoError(t, err)
	assert.Equal(t, data, converted)
}

func TestConvertV27HistoryInvalid(t *testing.T) {
	for _, tc := range []struct {
		name string
		data []byte
		err  string
	}{
		{"empty", nil, "empty history record"},
		{"unknown entity", []byte{0xff, 0, 0, 0, 4, 0, 0, 0, 1}, "bad entity type"},
		{"zero entity", []byte{0, 0, 0, 0, 4, 0, 0, 0, 1}, "bad entity type"},
		{"invalid in both formats", []byte{byte(dataEntry), 0, 0, 0, 10, 1}, "invalid history of entity"},
		{"truncated size", []byte{byte(dataEntry), 0, 0}, "invalid history of entity"},
	} {
		_, err := convertV27History(tc.data)
		assert.ErrorContains(t, err, tc.err, tc.name)
	}
	_, err := decodeV27History([]byte{byte(alias), 1, 2, 3})
	assert.ErrorContains(t, err, "fixed size")
	_, err = decodeV27History([]byte{0xff})
	assert.ErrorContains(t, err, "bad entity type")
	_, err = decodeV27History(nil)
	assert.ErrorIs(t, err, errInvalidDataSize)
	_, err = marshalV27History(&historyRecord{entityType: 0xff})
	assert.ErrorContains(t, err, "bad entity type")
}

func TestMarshalV27HistoryFixedSize(t *testing.T) {
	p, _ := entityProperties(ordersVolume)
	entries := testHistoryEntries(ordersVolume, p, 3)
	data, err := marshalV27History(&historyRecord{entityType: ordersVolume, entries: entries})
	require.NoError(t, err)
	assert.Equal(t, referenceV27HistoryEncoding(ordersVolume, true, entries), data)
}

func TestConvertV27HistoryAmbiguous(t *testing.T) {
	if testing.Short() {
		t.Skip("allocates 64 MiB")
	}
	// The only entry of the history of version 27 has the size 0x04000000. In version 28 the same bytes are
	// the entry of size 4 followed by the entry, which takes the rest of the data.
	const v27Size = 0x04000000
	data := make([]byte, 1+4+v27Size)
	data[0] = byte(dataEntry)
	binary.BigEndian.PutUint32(data[1:5], v27Size)
	const v28SecondSize = v27Size - 1 - 4 // the first entry takes 1+4 bytes, the size of the second takes 4 bytes
	require.Equal(t, 4, binary.PutUvarint(data[6:], v28SecondSize))
	_, err := decodeV27History(data)
	require.NoError(t, err)
	n, err := variableSizeEntriesCount(data)
	require.NoError(t, err)
	require.Equal(t, 2, n)

	_, err = convertV27History(data)
	assert.ErrorContains(t, err, "ambiguous history")
}

func TestMigrationStepsFromV27(t *testing.T) {
	steps := migrationStepsFromV27(nil)
	entities := variableSizeEntities()
	require.Len(t, steps, len(entities))
	prefixes := make(map[byte]struct{}, len(steps))
	for i, e := range entities {
		prefix, err := prefixByEntity(e)
		require.NoError(t, err)
		assert.Equal(t, prefix, []byte{steps[i].prefix}, "entity %d", e)
		assert.NotContains(t, prefixes, steps[i].prefix, "duplicate prefix")
		prefixes[steps[i].prefix] = struct{}{}
	}
	assert.Contains(t, prefixes, addressToAliasesPrefix)
	for _, e := range allBlockchainEntities() {
		if p, _ := entityProperties(e); p.fixedSize {
			prefix, err := prefixByEntity(e)
			require.NoError(t, err)
			assert.NotContains(t, prefixes, prefix[0], "no step for fixed size entity %d", e)
		}
	}
	require.Equal(t, stateVersionV27, int(stateMigrations()[len(stateMigrations())-1].from))
	require.Equal(t, StateVersion, stateVersionV27+1)
}

// v27TestState is the raw database of version 27 with histories of all variable size entities.
type v27TestState struct {
	dataDir  string
	v27      map[string][]byte // records as written in version 27
	expected map[string][]byte // the same records in version 28
	keys     [][]byte          // keys of variable size histories in the order of migration
}

func v27TestKey(prefix byte, i int) []byte {
	return binary.BigEndian.AppendUint32([]byte{prefix}, uint32(i)) // #nosec G115 // test data.
}

func createV27TestState(t *testing.T) *v27TestState {
	ts := &v27TestState{dataDir: t.TempDir(), v27: make(map[string][]byte), expected: make(map[string][]byte)}
	db, err := keyvalue.NewKeyVal(filepath.Join(ts.dataDir, keyvalueDir), defaultTestKeyValParams())
	require.NoError(t, err)
	batch, err := db.NewBatch()
	require.NoError(t, err)
	put := func(key, v27, expected []byte) {
		batch.Put(key, v27)
		ts.v27[string(key)] = v27
		ts.expected[string(key)] = expected
	}
	for _, step := range migrationStepsFromV27(nil) {
		count := 3
		if step.prefix == accountsDataStorKeyPrefix {
			count = migrationBatchSize + 10 // more than one batch to test batching
		}
		for i := range count {
			entity := blockchainEntity(i%int(generators) + 1) // #nosec G115 // test data.
			p, _ := entityProperties(entity)
			if p.fixedSize {
				entity = dataEntry
			}
			entries := testHistoryEntries(entity, blockchainEntityProperties{}, 1+i%4)
			key := v27TestKey(step.prefix, i)
			put(key, referenceV27HistoryEncoding(entity, false, entries),
				referenceHistoryRecordEncoding(entity, false, entries))
			ts.keys = append(ts.keys, key)
		}
		// The history, which is already of the new format.
		entries := testHistoryEntries(dataEntry, blockchainEntityProperties{}, 2)
		newData := referenceHistoryRecordEncoding(dataEntry, false, entries)
		put(v27TestKey(step.prefix, count), newData, newData)
	}
	// Fixed size histories are not changed.
	for _, e := range allBlockchainEntities() {
		p, _ := entityProperties(e)
		if !p.fixedSize {
			continue
		}
		prefix, pErr := prefixByEntity(e)
		require.NoError(t, pErr)
		data := referenceV27HistoryEncoding(e, true, testHistoryEntries(e, p, 2))
		put(v27TestKey(prefix[0], 0), data, data)
	}
	info := stateInfo{Version: stateVersionV27, HasStateHashes: true}
	infoBytes, err := info.marshalBinary()
	require.NoError(t, err)
	batch.Put(stateInfoKeyBytes, infoBytes)
	require.NoError(t, db.Flush(batch))
	require.NoError(t, db.Close())
	require.NoError(t, writeDBMeta(ts.dataDir, info))
	return ts
}

func (ts *v27TestState) verify(t *testing.T) {
	info, err := readDBMeta(ts.dataDir)
	require.NoError(t, err)
	assert.Equal(t, uint16(StateVersion), info.Version)

	db, err := keyvalue.NewKeyVal(filepath.Join(ts.dataDir, keyvalueDir), defaultTestKeyValParams())
	require.NoError(t, err)
	defer func() { require.NoError(t, db.Close()) }()
	dbInfo, err := readStateInfoFromDB(db)
	require.NoError(t, err)
	assert.Equal(t, uint16(StateVersion), dbInfo.Version)
	_, err = db.Get([]byte{migrationProgressKeyPrefix})
	assert.ErrorIs(t, err, keyvalue.ErrNotFound)
	for _, key := range slices.Sorted(maps.Keys(ts.expected)) {
		data, gErr := db.Get([]byte(key))
		require.NoError(t, gErr)
		require.Equal(t, ts.expected[key], data, "key %x", key)
	}
}

func TestMigrateStateFromV27(t *testing.T) {
	ts := createV27TestState(t)
	require.NoError(t, MigrateState(ts.dataDir, defaultTestKeyValParams()))
	ts.verify(t)
	// Repeated migration does nothing.
	require.NoError(t, MigrateState(ts.dataDir, defaultTestKeyValParams()))
	ts.verify(t)
}

func TestMigrateStateFromV27Resume(t *testing.T) {
	steps := migrationStepsFromV27(nil)
	for _, stepIndex := range []int{0, slices.IndexFunc(steps, func(s migrationStep) bool {
		return s.prefix == accountsDataStorKeyPrefix
	}), len(steps) - 1} {
		t.Run(fmt.Sprintf("step_%d", stepIndex), func(t *testing.T) {
			ts := createV27TestState(t)
			// Simulate the interruption in the middle of the step: the records of previous steps and the records
			// of the step up to the progress key are converted, the rest are not.
			db, err := keyvalue.NewKeyVal(filepath.Join(ts.dataDir, keyvalueDir), defaultTestKeyValParams())
			require.NoError(t, err)
			batch, err := db.NewBatch()
			require.NoError(t, err)
			stepPrefix := steps[stepIndex].prefix
			var lastKey []byte
			for _, key := range ts.keys {
				if key[0] == stepPrefix && binary.BigEndian.Uint32(key[1:]) > 1 {
					continue // not converted yet
				}
				if key[0] == stepPrefix {
					lastKey = key
				}
				if key[0] == stepPrefix || slices.IndexFunc(steps, func(s migrationStep) bool {
					return s.prefix == key[0]
				}) < stepIndex {
					batch.Put(key, ts.expected[string(key)])
				}
			}
			require.NotNil(t, lastKey)
			require.NoError(t, flushWithProgress(db, batch,
				migrationProgress{Version: stateVersionV27, Step: stepIndex, LastKey: lastKey}))
			require.NoError(t, db.Close())

			require.NoError(t, MigrateState(ts.dataDir, defaultTestKeyValParams()))
			ts.verify(t)
		})
	}
}

func TestMigrateStateFromV27InvalidHistory(t *testing.T) {
	ts := createV27TestState(t)
	db, err := keyvalue.NewKeyVal(filepath.Join(ts.dataDir, keyvalueDir), defaultTestKeyValParams())
	require.NoError(t, err)
	batch, err := db.NewBatch()
	require.NoError(t, err)
	batch.Put(v27TestKey(scoreKeyPrefix, 100), []byte{byte(score), 0, 0, 0, 10, 1})
	require.NoError(t, db.Flush(batch))
	require.NoError(t, db.Close())

	err = MigrateState(ts.dataDir, defaultTestKeyValParams())
	require.ErrorContains(t, err, "failed to migrate state from version 27 to 28")
	info, err := readDBMeta(ts.dataDir)
	require.NoError(t, err)
	assert.Equal(t, uint16(stateVersionV27), info.Version)
}

// TestMigrateStateFromV27OpensAsV28 downgrades the state created by the node to the format of version 27,
// migrates it back and checks that the node opens the migrated state and reads the same values.
func TestMigrateStateFromV27OpensAsV28(t *testing.T) {
	dataDir := t.TempDir()
	bs := settings.MustMainNetSettings()
	params := DefaultTestingStateParams()
	params.BuildStateHashes = true
	sm, err := newStateManager(t.Context(), dataDir, true, params, bs, false, nil)
	require.NoError(t, err)
	genesisTx, ok := bs.Genesis.Transactions[0].(*proto.Genesis)
	require.True(t, ok)
	recipient := proto.NewRecipientFromAddress(genesisTx.Recipient)
	balance, err := sm.NewestWavesBalance(recipient)
	require.NoError(t, err)
	require.Equal(t, genesisTx.Amount, balance)
	score, err := sm.ScoreAtHeight(1)
	require.NoError(t, err)
	require.NoError(t, sm.Close())

	// Downgrade the state to version 27.
	meta, err := readDBMeta(dataDir)
	require.NoError(t, err)
	kvParams := params.DbParams
	kvParams.CompressionAlgo = meta.DBCompressionAlgo
	kvParams.DisableBloomFilter = true
	kvParams.BloomFilterStore = keyvalue.NoOpStore{}
	db, err := keyvalue.NewKeyVal(filepath.Join(dataDir, keyvalueDir), kvParams)
	require.NoError(t, err)
	batch, err := db.NewBatch()
	require.NoError(t, err)
	v28 := make(map[string][]byte)
	for _, step := range migrationStepsFromV27(nil) {
		iter, iErr := db.NewKeyIterator([]byte{step.prefix})
		require.NoError(t, iErr)
		for iter.Next() {
			value := bytes.Clone(iter.Value())
			hr, dErr := newHistoryRecordFromBytes(value)
			require.NoError(t, dErr)
			v27, mErr := marshalV27History(hr)
			require.NoError(t, mErr)
			key := keyvalue.SafeKey(iter)
			v28[string(key)] = value
			batch.Put(key, v27)
		}
		require.NoError(t, iter.Error())
		iter.Release()
	}
	require.NotEmpty(t, v28)
	info, err := readStateInfoFromDB(db)
	require.NoError(t, err)
	info.Version = stateVersionV27
	infoBytes, err := info.marshalBinary()
	require.NoError(t, err)
	batch.Put(stateInfoKeyBytes, infoBytes)
	require.NoError(t, db.Flush(batch))
	require.NoError(t, db.Close())
	meta.Version = stateVersionV27
	require.NoError(t, writeDBMeta(dataDir, meta))

	_, err = newStateManager(t.Context(), dataDir, true, params, bs, false, nil)
	require.Error(t, err, "the state of version 27 must not be opened")

	require.NoError(t, MigrateState(dataDir, kvParams))

	db, err = keyvalue.NewKeyVal(filepath.Join(dataDir, keyvalueDir), kvParams)
	require.NoError(t, err)
	for key, expected := range v28 {
		data, gErr := db.Get([]byte(key))
		require.NoError(t, gErr)
		assert.Equal(t, expected, data, "key %x", key)
	}
	require.NoError(t, db.Close())

	sm, err = newStateManager(t.Context(), dataDir, true, params, bs, false, nil)
	require.NoError(t, err)
	defer func() { require.NoError(t, sm.Close()) }()
	balance, err = sm.NewestWavesBalance(recipient)
	require.NoError(t, err)
	assert.Equal(t, genesisTx.Amount, balance)
	migratedScore, err := sm.ScoreAtHeight(1)
	require.NoError(t, err)
	assert.Equal(t, score, migratedScore)
}
