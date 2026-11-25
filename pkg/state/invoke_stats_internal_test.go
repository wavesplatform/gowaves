package state

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func statTestAddressID(b byte) proto.AddressID {
	var id proto.AddressID
	for i := range id {
		id[i] = b
	}
	return id
}

func roleKey(role statRole, kind accessKind) statKey {
	return statKey{role: role, key: accessKey{kind: kind}}
}

func absKey(k accessKey) statKey { return statKey{role: statRoleNone, key: k} }

// numberedStatKeys returns n distinct absolute keys starting from the given number.
func numberedStatKeys(from, n int) []statKey {
	res := make([]statKey, 0, n)
	for i := from; i < from+n; i++ {
		res = append(res, absKey(aliasAccessKey(fmt.Sprintf("alias-%05d", i))))
	}
	return res
}

func TestNormalizeAndInstantiate(t *testing.T) {
	sender, dApp, other := statTestAddressID(1), statTestAddressID(2), statTestAddressID(3)
	asset, lease := testDigest(4), testDigest(5)
	for _, tc := range []struct {
		name     string
		sender   proto.AddressID
		dApp     proto.AddressID
		actual   accessSet
		expected []statKey
		// roundTrip is the expected result of the instantiation with the same sender and dApp.
		roundTrip accessSet
	}{
		{
			name: "empty", sender: sender, dApp: dApp,
			actual: newAccessSet(), expected: []statKey{}, roundTrip: newAccessSet(),
		},
		{
			name: "roles and absolute keys", sender: sender, dApp: dApp,
			actual: newAccessSet(
				accountBalanceAccessKey(sender), accountDataAccessKey(dApp), accountScriptAccessKey(dApp),
				accountBalanceAccessKey(dApp), accountDataAccessKey(other), assetAccessKey(asset),
				aliasAccessKey("alias"), leaseAccessKey(lease),
			),
			expected: []statKey{
				absKey(accountDataAccessKey(other)), absKey(assetAccessKey(asset)),
				absKey(aliasAccessKey("alias")), absKey(leaseAccessKey(lease)),
				roleKey(statRoleSender, accessKindAccountBalance),
				roleKey(statRoleDApp, accessKindAccountData), roleKey(statRoleDApp, accessKindAccountBalance),
				roleKey(statRoleDApp, accessKindAccountScript),
			},
			roundTrip: newAccessSet(
				accountBalanceAccessKey(sender), accountDataAccessKey(dApp), accountScriptAccessKey(dApp),
				accountBalanceAccessKey(dApp), accountDataAccessKey(other), assetAccessKey(asset),
				aliasAccessKey("alias"), leaseAccessKey(lease),
			),
		},
		{
			name: "transaction keys are dropped", sender: sender, dApp: dApp,
			actual:    newAccessSet(txAccessKey(asset.Bytes()), accountDataAccessKey(dApp)),
			expected:  []statKey{roleKey(statRoleDApp, accessKindAccountData)},
			roundTrip: newAccessSet(accountDataAccessKey(dApp)),
		},
		{
			name: "sender is dApp", sender: dApp, dApp: dApp,
			actual: newAccessSet(accountBalanceAccessKey(dApp), accountDataAccessKey(dApp),
				accountDataAccessKey(other)),
			expected: []statKey{
				absKey(accountDataAccessKey(other)),
				roleKey(statRoleSender, accessKindAccountData), roleKey(statRoleSender, accessKindAccountBalance),
				roleKey(statRoleDApp, accessKindAccountData), roleKey(statRoleDApp, accessKindAccountBalance),
			},
			roundTrip: newAccessSet(accountBalanceAccessKey(dApp), accountDataAccessKey(dApp),
				accountDataAccessKey(other)),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			keys := normalize(tc.actual, tc.sender, tc.dApp)
			assert.Equal(t, tc.expected, keys)
			stats := newInvokeStats()
			stats.update(keys, keys, false)
			reads, writes := instantiate(stats, tc.sender, tc.dApp)
			assert.Equal(t, tc.roundTrip, reads)
			assert.Equal(t, tc.roundTrip, writes)
		})
	}
}

func TestInstantiateForAnotherSender(t *testing.T) {
	sender, dApp, sender2, dApp2 := statTestAddressID(1), statTestAddressID(2), statTestAddressID(3), statTestAddressID(4)
	stats := newInvokeStats()
	stats.update(
		normalize(newAccessSet(accountBalanceAccessKey(sender), accountDataAccessKey(dApp), aliasAccessKey("a")),
			sender, dApp),
		normalize(newAccessSet(accountDataAccessKey(dApp)), sender, dApp),
		false,
	)
	reads, writes := instantiate(stats, sender2, dApp2)
	assert.Equal(t,
		newAccessSet(accountBalanceAccessKey(sender2), accountDataAccessKey(dApp2), aliasAccessKey("a")), reads)
	assert.Equal(t, newAccessSet(accountDataAccessKey(dApp2)), writes)

	// The statistics collected when the dApp invoked itself predict the keys of both roles.
	self := newInvokeStats()
	self.update(normalize(newAccessSet(accountDataAccessKey(dApp)), dApp, dApp), nil, false)
	reads, writes = instantiate(self, sender2, dApp2)
	assert.Equal(t, newAccessSet(accountDataAccessKey(sender2), accountDataAccessKey(dApp2)), reads)
	assert.Equal(t, 0, writes.len())

	reads, writes = instantiate(nil, sender, dApp)
	assert.Equal(t, 0, reads.len())
	assert.Equal(t, 0, writes.len())
}

// TestInvokeStatsUpdateReportsChanges checks that an update reports whether the predictions made from
// the statistics have changed.
func TestInvokeStatsUpdateReportsChanges(t *testing.T) {
	stats := newInvokeStats()
	assert.False(t, stats.update(nil, nil, false), "nothing is read or written")
	assert.True(t, stats.update(numberedStatKeys(0, 2), nil, false), "new read keys")
	assert.False(t, stats.update(numberedStatKeys(0, 2), nil, true), "the same keys")
	assert.False(t, stats.update(numberedStatKeys(1, 1), nil, false), "a part of the known keys")
	assert.True(t, stats.update(numberedStatKeys(0, 2), numberedStatKeys(0, 1), false), "new write key")
	assert.True(t, stats.update(numberedStatKeys(0, 3), numberedStatKeys(0, 1), false), "one more read key")

	// The eviction of the least recently seen keys changes the predictions too.
	full := newInvokeStats()
	assert.True(t, full.update(numberedStatKeys(0, invokeStatsKeysSoftCap), nil, false))
	assert.False(t, full.update(numberedStatKeys(0, invokeStatsKeysSoftCap), nil, false))
	assert.True(t, full.update(numberedStatKeys(1000, 1), nil, false))
	require.Len(t, full.reads, invokeStatsKeysSoftCap)

	assert.True(t, stats.update(numberedStatKeys(0, invokeStatsKeysHardCap+1), nil, false), "becomes wide")
	assert.False(t, stats.update(numberedStatKeys(0, 2), numberedStatKeys(5, 3), false), "wide statistics hold no keys")
}

func TestInvokeStatsUpdate(t *testing.T) {
	t.Run("counters and union", func(t *testing.T) {
		stats := newInvokeStats()
		stats.update(numberedStatKeys(0, 2), numberedStatKeys(10, 1), false)
		stats.update(numberedStatKeys(1, 2), nil, true)
		assert.Equal(t, uint64(2), stats.executions)
		assert.Equal(t, uint64(1), stats.mispredictions)
		assert.False(t, stats.wide)
		k := numberedStatKeys(0, 3)
		assert.Equal(t, statKeySet{k[0]: 1, k[1]: 2, k[2]: 2}, stats.reads)
		assert.Equal(t, statKeySet{numberedStatKeys(10, 1)[0]: 1}, stats.writes)
	})
	t.Run("eviction of least recently seen keys", func(t *testing.T) {
		stats := newInvokeStats()
		stable := numberedStatKeys(0, 4)
		// Every execution touches the stable keys and some unique keys, like IDs of new leases.
		for i := range 100 {
			keys := append(numberedStatKeys(1000+i*3, 3), stable...)
			stats.update(keys, keys, false)
			assert.LessOrEqual(t, len(stats.reads), invokeStatsKeysSoftCap)
			assert.LessOrEqual(t, len(stats.writes), invokeStatsKeysSoftCap)
		}
		assert.False(t, stats.wide)
		require.Len(t, stats.reads, invokeStatsKeysSoftCap)
		for _, k := range stable {
			assert.Equal(t, uint64(100), stats.reads[k])
		}
		// Only the most recent unique keys survive: 60 keys of the last 20 executions.
		for i := range 100 {
			for _, k := range numberedStatKeys(1000+i*3, 3) {
				_, ok := stats.reads[k]
				assert.Equal(t, i >= 80, ok, "execution %d", i)
			}
		}
		assert.Equal(t, stats.reads, stats.writes)
	})
	t.Run("ties are evicted deterministically", func(t *testing.T) {
		stats := newInvokeStats()
		stats.update(numberedStatKeys(0, invokeStatsKeysSoftCap), nil, false)
		stats.update(numberedStatKeys(1000, 2), nil, false)
		require.Len(t, stats.reads, invokeStatsKeysSoftCap)
		for i, k := range numberedStatKeys(0, invokeStatsKeysSoftCap) {
			_, ok := stats.reads[k]
			assert.Equal(t, i >= 2, ok, "key %d", i)
		}
	})
	t.Run("keys of the current execution are never evicted", func(t *testing.T) {
		stats := newInvokeStats()
		stats.update(numberedStatKeys(0, 10), nil, false)
		big := numberedStatKeys(1000, invokeStatsKeysHardCap)
		stats.update(big, nil, false)
		assert.False(t, stats.wide)
		require.Len(t, stats.reads, invokeStatsKeysHardCap)
		for _, k := range big {
			assert.Contains(t, stats.reads, k)
		}
		// The next small execution shrinks the set back to the soft cap.
		stats.update(numberedStatKeys(0, 1), nil, false)
		assert.Len(t, stats.reads, invokeStatsKeysSoftCap)
		assert.Contains(t, stats.reads, numberedStatKeys(0, 1)[0])
	})
	t.Run("wide transition", func(t *testing.T) {
		for name, wideUpdate := range map[string]func(s *invokeStats){
			"reads":  func(s *invokeStats) { s.update(numberedStatKeys(0, invokeStatsKeysHardCap+1), nil, false) },
			"writes": func(s *invokeStats) { s.update(nil, numberedStatKeys(0, invokeStatsKeysHardCap+1), true) },
		} {
			t.Run(name, func(t *testing.T) {
				stats := newInvokeStats()
				stats.update(numberedStatKeys(0, 3), numberedStatKeys(0, 3), false)
				require.False(t, stats.wide)
				wideUpdate(stats)
				assert.True(t, stats.wide)
				assert.Empty(t, stats.reads)
				assert.Empty(t, stats.writes)
				// The entry stays wide and counts executions.
				stats.update(numberedStatKeys(0, 3), numberedStatKeys(0, 3), true)
				assert.True(t, stats.wide)
				assert.Empty(t, stats.reads)
				assert.Empty(t, stats.writes)
				assert.Equal(t, uint64(3), stats.executions)
			})
		}
	})
}

func sampleInvokeStats() *invokeStats {
	sender, dApp, other := statTestAddressID(1), statTestAddressID(2), statTestAddressID(3)
	stats := newInvokeStats()
	stats.update(
		normalize(newAccessSet(
			accountBalanceAccessKey(sender), accountDataAccessKey(dApp), accountScriptAccessKey(dApp),
			accountScriptAccessKey(other), assetAccessKey(testDigest(4)), aliasAccessKey("alias"),
			aliasAccessKey("the-longest-possible-alias-123"), leaseAccessKey(testDigest(5)),
		), sender, dApp),
		normalize(newAccessSet(accountDataAccessKey(dApp), accountBalanceAccessKey(other)), sender, dApp),
		false,
	)
	stats.update(normalize(newAccessSet(accountDataAccessKey(dApp)), sender, dApp), nil, true)
	return stats
}

func TestInvokeStatsSerializationRoundTrip(t *testing.T) {
	wide := newInvokeStats()
	wide.update(numberedStatKeys(0, invokeStatsKeysHardCap+1), nil, true)
	full := newInvokeStats()
	full.update(numberedStatKeys(0, invokeStatsKeysHardCap), numberedStatKeys(0, invokeStatsKeysHardCap), false)
	for name, stats := range map[string]*invokeStats{
		"empty": newInvokeStats(), "sample": sampleInvokeStats(), "wide": wide, "full": full,
	} {
		t.Run(name, func(t *testing.T) {
			data, err := stats.marshalBinary()
			require.NoError(t, err)
			var actual invokeStats
			require.NoError(t, actual.unmarshalBinary(data))
			assert.Equal(t, stats, &actual)
			// The encoding is deterministic.
			data2, err := actual.marshalBinary()
			require.NoError(t, err)
			assert.Equal(t, data, data2)
		})
	}
}

func TestInvokeStatsUnmarshalCorrupt(t *testing.T) {
	valid, err := sampleInvokeStats().marshalBinary()
	require.NoError(t, err)
	mutate := func(f func(b []byte) []byte) []byte { return f(append([]byte(nil), valid...)) }
	accountRole := byte(statRoleSender)<<statKeyRoleShift | byte(accessKindAccountData)
	for name, data := range map[string][]byte{
		"nil":             nil,
		"version only":    {invokeStatsVersion},
		"unknown version": mutate(func(b []byte) []byte { b[0] = invokeStatsVersion + 1; return b }),
		"unknown flags":   mutate(func(b []byte) []byte { b[1] |= 0x80; return b }),
		"trailing byte":   mutate(func(b []byte) []byte { return append(b, 0) }),
		"no key sets":     {invokeStatsVersion, 0, 1, 0},
		"no write set":    {invokeStatsVersion, 0, 1, 0, 0},
		"too many keys":   {invokeStatsVersion, 0, 1, 0, 0x81, 0x02, 0},
		"missing key":     {invokeStatsVersion, 0, 1, 0, 1, 0},
		"unknown kind":    {invokeStatsVersion, 0, 1, 0, 1, 0x3f, 1, 0},
		"zero kind":       {invokeStatsVersion, 0, 1, 0, 1, 0, 1, 0},
		"unknown role":    {invokeStatsVersion, 0, 1, 0, 1, 3<<statKeyRoleShift | byte(accessKindAccountData), 1, 0},
		"role of asset":   {invokeStatsVersion, 0, 1, 0, 1, 1<<statKeyRoleShift | byte(accessKindAsset), 1, 0},
		"short payload":   {invokeStatsVersion, 0, 1, 0, 1, byte(accessKindLease), 1, 2, 3},
		"no last seen":    {invokeStatsVersion, 0, 1, 0, 1, accountRole},
		"duplicate key":   {invokeStatsVersion, 0, 1, 0, 2, accountRole, 1, accountRole, 1, 0},
		"bad uvarint":     {invokeStatsVersion, 0, 0xff},
	} {
		t.Run(name, func(t *testing.T) {
			stats := invokeStats{executions: 42}
			require.Error(t, stats.unmarshalBinary(data))
			assert.Equal(t, invokeStats{executions: 42}, stats, "failed unmarshal must not change the receiver")
		})
	}
	// Every truncation of a valid record is rejected.
	for i := range valid {
		var stats invokeStats
		assert.Error(t, stats.unmarshalBinary(valid[:i]), "truncated to %d bytes", i)
	}
}

type invokeStatsTestDB struct {
	t     *testing.T
	dir   string
	db    *keyvalue.KeyVal
	batch keyvalue.Batch
}

func newInvokeStatsTestDB(t *testing.T) *invokeStatsTestDB {
	d := &invokeStatsTestDB{t: t, dir: t.TempDir()}
	d.open()
	t.Cleanup(func() { assert.NoError(t, d.db.Close()) })
	return d
}

func (d *invokeStatsTestDB) open() {
	db, err := keyvalue.NewKeyVal(d.dir, defaultTestKeyValParams())
	require.NoError(d.t, err)
	batch, err := db.NewBatch()
	require.NoError(d.t, err)
	d.db, d.batch = db, batch
}

func (d *invokeStatsTestDB) reopen() {
	require.NoError(d.t, d.db.Close())
	d.open()
}

// write does what stateDB does after the flush of storages.
func (d *invokeStatsTestDB) write() {
	require.NoError(d.t, d.db.Flush(d.batch))
	d.batch.Reset()
}

func (d *invokeStatsTestDB) storage(maxEntries int) *invokeStatsStorage {
	return newInvokeStatsStorage(d.db, d.batch, maxEntries)
}

func TestInvokeStatsStoragePersistence(t *testing.T) {
	d := newInvokeStatsTestDB(t)
	stor := d.storage(10)
	sender, dApp := statTestAddressID(1), statTestAddressID(2)
	key := invokeStatsKey{dApp: dApp, function: "call"}
	otherKey := invokeStatsKey{dApp: dApp, function: "call2"}

	_, ok, err := stor.stats(key)
	require.NoError(t, err)
	assert.False(t, ok)

	reads := newAccessSet(accountBalanceAccessKey(sender), accountDataAccessKey(dApp), aliasAccessKey("alias"))
	writes := newAccessSet(accountDataAccessKey(dApp), txAccessKey(testDigest(7).Bytes()))
	require.NoError(t, stor.update(key, sender, reads, writes, false))
	require.NoError(t, stor.update(key, sender, reads, writes, true))

	expected := newInvokeStats()
	expected.update(normalize(reads, sender, dApp), normalize(writes, sender, dApp), false)
	expected.update(normalize(reads, sender, dApp), normalize(writes, sender, dApp), true)

	stats, ok, err := stor.stats(key)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, expected, stats)

	// Nothing is in the database until the flush and the write of the batch.
	_, err = d.db.Get(key.bytes())
	require.ErrorIs(t, err, keyvalue.ErrNotFound)
	require.NoError(t, stor.flush())
	d.write()
	stor.reset() // Reset after the successful flush keeps flushed entries.
	stats, ok, err = stor.stats(key)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, expected, stats)

	// A new storage over the same database loads the entry lazily.
	stats, ok, err = d.storage(10).stats(key)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, expected, stats)

	// The entry survives the reopening of the database and can be updated further.
	d.reopen()
	stor = d.storage(10)
	stats, ok, err = stor.stats(key)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, expected, stats)
	_, ok, err = stor.stats(otherKey)
	require.NoError(t, err)
	assert.False(t, ok)

	require.NoError(t, stor.update(key, sender, reads, writes, false))
	expected.update(normalize(reads, sender, dApp), normalize(writes, sender, dApp), false)
	require.NoError(t, stor.flush())
	d.write()
	d.reopen()
	stats, ok, err = d.storage(10).stats(key)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, expected, stats)
	assert.Equal(t, uint64(3), stats.executions)
	assert.Equal(t, uint64(1), stats.mispredictions)
}

// TestInvokeStatsStorageRevision checks that the revision of the storage is changed only when the predictions
// made from statistics may change.
func TestInvokeStatsStorageRevision(t *testing.T) {
	d := newInvokeStatsTestDB(t)
	stor := d.storage(10)
	sender, dApp := statTestAddressID(1), statTestAddressID(2)
	key := invokeStatsKey{dApp: dApp, function: "call"}
	other := invokeStatsKey{dApp: dApp, function: "other"}
	set := newAccessSet(accountDataAccessKey(dApp))
	bigger := newAccessSet(accountDataAccessKey(dApp), accountDataAccessKey(statTestAddressID(3)))

	revision := stor.revision
	changed := func() bool {
		defer func() { revision = stor.revision }()
		return stor.revision != revision
	}
	_, _, err := stor.stats(key)
	require.NoError(t, err)
	assert.False(t, changed(), "reading changes nothing")
	require.NoError(t, stor.update(key, sender, set, set, false))
	assert.True(t, changed(), "new statistics")
	require.NoError(t, stor.update(key, sender, set, set, true))
	assert.False(t, changed(), "the same keys")
	require.NoError(t, stor.update(key, sender, bigger, set, false))
	assert.True(t, changed(), "new key")
	// Statistics without keys are still new statistics.
	require.NoError(t, stor.update(other, sender, newAccessSet(), newAccessSet(), false))
	assert.True(t, changed(), "new empty statistics")
	require.NoError(t, stor.flush())
	assert.False(t, changed(), "flush changes nothing")
	stor.reset()
	assert.False(t, changed(), "nothing to drop")
	require.NoError(t, stor.update(key, sender, set, set, false))
	assert.False(t, changed())
	stor.reset()
	assert.True(t, changed(), "dropped entries are loaded again")
}

func TestInvokeStatsStorageReset(t *testing.T) {
	d := newInvokeStatsTestDB(t)
	stor := d.storage(10)
	sender, dApp := statTestAddressID(1), statTestAddressID(2)
	flushed := invokeStatsKey{dApp: dApp, function: "flushed"}
	failed := invokeStatsKey{dApp: dApp, function: "failed"}
	set := newAccessSet(accountDataAccessKey(dApp))

	require.NoError(t, stor.update(flushed, sender, set, set, false))
	require.NoError(t, stor.flush())
	d.write()

	// Updates of a failed batch of blocks are dropped: the cache returns to what is in the database.
	require.NoError(t, stor.update(flushed, sender, set, set, true))
	require.NoError(t, stor.update(failed, sender, set, set, false))
	assert.Equal(t, 2, stor.dirty)
	stor.reset()
	d.batch.Reset()
	assert.Equal(t, 0, stor.dirty)
	assert.Empty(t, stor.entries)
	assert.Equal(t, 0, stor.order.Len())

	stats, ok, err := stor.stats(flushed)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, uint64(1), stats.executions)
	assert.Equal(t, uint64(0), stats.mispredictions)
	_, ok, err = stor.stats(failed)
	require.NoError(t, err)
	assert.False(t, ok)

	// Nothing to flush after the reset.
	require.NoError(t, stor.flush())
	d.write()
	_, err = d.db.Get(failed.bytes())
	require.ErrorIs(t, err, keyvalue.ErrNotFound)
}

func TestInvokeStatsStorageCorruptRecord(t *testing.T) {
	d := newInvokeStatsTestDB(t)
	sender, dApp := statTestAddressID(1), statTestAddressID(2)
	set := newAccessSet(accountDataAccessKey(dApp))
	valid, err := sampleInvokeStats().marshalBinary()
	require.NoError(t, err)
	for i, record := range [][]byte{
		{},
		{invokeStatsVersion + 1, 0, 0, 0, 0, 0}, // A record of an unknown version.
		valid[:len(valid)-1],
		append(append([]byte(nil), valid...), 0),
	} {
		key := invokeStatsKey{dApp: dApp, function: fmt.Sprintf("corrupt-%d", i)}
		require.NoError(t, d.db.Put(key.bytes(), record))
		stor := d.storage(10)
		_, ok, sErr := stor.stats(key)
		require.NoError(t, sErr)
		assert.False(t, ok, "record %d", i)
		// The unreadable record is replaced after the next execution.
		require.NoError(t, stor.update(key, sender, set, set, false))
		require.NoError(t, stor.flush())
		d.write()
		stats, ok, sErr := d.storage(10).stats(key)
		require.NoError(t, sErr)
		require.True(t, ok, "record %d", i)
		assert.Equal(t, uint64(1), stats.executions)
	}
}

func TestInvokeStatsStorageCacheBound(t *testing.T) {
	const maxEntries = 4
	d := newInvokeStatsTestDB(t)
	stor := d.storage(maxEntries)
	sender, dApp := statTestAddressID(1), statTestAddressID(2)
	set := newAccessSet(accountDataAccessKey(dApp))
	key := func(i int) invokeStatsKey { return invokeStatsKey{dApp: dApp, function: fmt.Sprintf("f%d", i)} }

	// Dirty entries are not evicted, the cache exceeds the limit until the flush.
	for i := range 3 * maxEntries {
		require.NoError(t, stor.update(key(i), sender, set, set, false))
	}
	assert.Len(t, stor.entries, 3*maxEntries)
	require.NoError(t, stor.flush())
	d.write()
	assert.Len(t, stor.entries, maxEntries)
	assert.Equal(t, maxEntries, stor.order.Len())
	// The most recently used entries are kept.
	for i := 3*maxEntries - maxEntries; i < 3*maxEntries; i++ {
		assert.Contains(t, stor.entries, key(i))
	}

	// Reads, including the reads of absent statistics, keep the cache bounded and evict the least recently used.
	for i := range 4 * maxEntries {
		_, ok, err := stor.stats(key(i))
		require.NoError(t, err)
		assert.Equal(t, i < 3*maxEntries, ok)
		assert.LessOrEqual(t, len(stor.entries), maxEntries)
	}
	_, err := stor.entry(key(4*maxEntries - maxEntries)) // Touch the oldest entry of the cache.
	require.NoError(t, err)
	_, _, err = stor.stats(key(0))
	require.NoError(t, err)
	assert.Contains(t, stor.entries, key(4*maxEntries-maxEntries))
	assert.NotContains(t, stor.entries, key(4*maxEntries-maxEntries+1))

	// Evicted entries are loaded again with all the data.
	stats, ok, err := stor.stats(key(1))
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, uint64(1), stats.executions)
}

func TestInvokeStatsFlushWithEntitiesStorage(t *testing.T) {
	to := createStorageObjects(t, true)
	sender, dApp := statTestAddressID(1), statTestAddressID(2)
	key := invokeStatsKey{dApp: dApp, function: "call"}
	set := newAccessSet(accountDataAccessKey(dApp))

	// A failed batch of blocks: entities are reset without the flush.
	require.NoError(t, to.entities.invokeStats.update(key, sender, set, set, false))
	to.entities.reset()
	to.stateDB.reset()
	_, ok, err := to.entities.invokeStats.stats(key)
	require.NoError(t, err)
	assert.False(t, ok)

	// A successful batch of blocks.
	require.NoError(t, to.entities.invokeStats.update(key, sender, set, set, false))
	to.flush(t)
	data, err := to.db.Get(key.bytes())
	require.NoError(t, err)
	var stats invokeStats
	require.NoError(t, stats.unmarshalBinary(data))
	assert.Equal(t, uint64(1), stats.executions)
	cached, ok, err := to.entities.invokeStats.stats(key)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, &stats, cached)
}
