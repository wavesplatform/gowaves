package state

import (
	"encoding/binary"
	"encoding/hex"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type v26TestBalance struct {
	balance  uint64
	leaseIn  int64
	leaseOut int64
	blockNum uint32
}

func v26WavesBalanceHistory(entries ...v26TestBalance) []byte {
	res := []byte{byte(wavesBalance)}
	for _, e := range entries {
		rec := make([]byte, v26WavesBalanceRecordSize+historyBlockNumSize)
		binary.BigEndian.PutUint64(rec[0:8], e.balance)
		binary.BigEndian.PutUint64(rec[8:16], uint64(e.leaseIn))   // #nosec G115 // test data.
		binary.BigEndian.PutUint64(rec[16:24], uint64(e.leaseOut)) // #nosec G115 // test data.
		binary.BigEndian.PutUint32(rec[24:28], e.blockNum)
		res = append(res, rec...)
	}
	return res
}

func v26AssetBalanceHistory(entries ...v26TestBalance) []byte {
	res := []byte{byte(assetBalance)}
	for _, e := range entries {
		rec := make([]byte, v26AssetBalanceRecordSize+historyBlockNumSize)
		binary.BigEndian.PutUint64(rec[0:8], e.balance)
		binary.BigEndian.PutUint32(rec[8:12], e.blockNum)
		res = append(res, rec...)
	}
	return res
}

func v26LegacyStateHash(sh *proto.StateHashV1) []byte {
	id := sh.BlockID.Bytes()
	res := append([]byte{byte(len(id))}, id...)
	for _, d := range []crypto.Digest{
		sh.SumHash, sh.DataEntryHash, sh.AccountScriptHash, sh.AssetScriptHash, sh.LeaseStatusHash,
		sh.SponsorshipHash, sh.AliasesHash, sh.WavesBalanceHash, sh.AssetBalanceHash, sh.LeaseBalanceHash,
	} {
		res = append(res, d[:]...)
	}
	return res
}

func testDigest(b byte) crypto.Digest {
	var d crypto.Digest
	for i := range d {
		d[i] = b
	}
	return d
}

func testLegacyStateHash(t *testing.T, b byte, prevSum []byte) *proto.StateHashV1 {
	id := genRandBlockIds(t, 1)[0]
	sh := &proto.StateHashV1{
		BlockID:           id,
		WavesBalanceHash:  testDigest(b + 1),
		AssetBalanceHash:  testDigest(b + 2),
		DataEntryHash:     testDigest(b + 3),
		AccountScriptHash: testDigest(b + 4),
		AssetScriptHash:   testDigest(b + 5),
		LeaseBalanceHash:  testDigest(b + 6),
		LeaseStatusHash:   testDigest(b + 7),
		SponsorshipHash:   testDigest(b + 8),
		AliasesHash:       testDigest(b + 9),
	}
	require.NoError(t, sh.GenerateSumHash(prevSum))
	return sh
}

func legacyStateHashHistory(t *testing.T, entries ...[]byte) []byte {
	hr := newHistoryRecord(legacyStateHash)
	for i, e := range entries {
		hr.entries = append(hr.entries, historyEntry{data: e, blockNum: uint32(i + 1)}) // #nosec G115 // test data.
	}
	data, err := hr.marshalBinary()
	require.NoError(t, err)
	return data
}

func v27BalanceHistory(t *testing.T, entity blockchainEntity, records ...interface{ marshalBinary() ([]byte, error) },
) []byte {
	hr := newHistoryRecord(entity)
	for i, r := range records {
		data, err := r.marshalBinary()
		require.NoError(t, err)
		hr.entries = append(hr.entries, historyEntry{data: data, blockNum: uint32(i + 1)}) // #nosec G115 // test data.
	}
	data, err := hr.marshalBinary()
	require.NoError(t, err)
	return data
}

// v26LegacyHeights is the number of heights with legacy state hashes in the test state, the hashes on even heights
// are stored in the old format, the hashes on odd heights are stored in the new format, and the hash at the last
// height is StateHashV2.
const v26LegacyHeights = 6

// v26ValidBlocks is the number of valid blocks in the test state, blocks with greater numbers are rolled back.
const v26ValidBlocks = 3

type v26TestState struct {
	dataDir      string
	wavesKeys    [][]byte
	wavesHistory [][]v26TestBalance
	assetKey     []byte
	assetHistory []v26TestBalance
	// Records written in the new format by the node of version 26 after the change of format.
	newWavesKey  []byte
	newWavesData []byte
	newAssetKey  []byte
	newAssetData []byte
	hashes       []*proto.StateHashV1 // by height starting from 1
	hashV2       []byte               // at the last height
}

func createV26TestState(t *testing.T) *v26TestState {
	ts := &v26TestState{dataDir: t.TempDir()}
	db, err := keyvalue.NewKeyVal(filepath.Join(ts.dataDir, keyvalueDir), defaultTestKeyValParams())
	require.NoError(t, err)
	batch, err := db.NewBatch()
	require.NoError(t, err)

	// Waves balances, more than one batch of records to test batching.
	const addresses = migrationBatchSize + 10
	for i := range addresses {
		var id proto.AddressID
		binary.BigEndian.PutUint32(id[:], uint32(i)) // #nosec G115 // test data.
		key := (&wavesBalanceKey{address: id}).bytes()
		history := []v26TestBalance{
			{balance: uint64(i) * 100, leaseIn: 5, leaseOut: 3, blockNum: 1}, // #nosec G115 // test data.
			{balance: 1 << 40, leaseIn: 0, leaseOut: 1 << 33, blockNum: 2},
		}
		ts.wavesKeys = append(ts.wavesKeys, key)
		ts.wavesHistory = append(ts.wavesHistory, history)
		batch.Put(key, v26WavesBalanceHistory(history...))
	}
	// Asset balance.
	ts.assetKey = (&assetBalanceKey{address: proto.AddressID{1}, asset: proto.AssetID{2}}).bytes()
	ts.assetHistory = []v26TestBalance{{balance: 42, blockNum: 1}, {balance: 1 << 50, blockNum: 3}}
	batch.Put(ts.assetKey, v26AssetBalanceHistory(ts.assetHistory...))
	// Balances of the new format.
	ts.newWavesKey = (&wavesBalanceKey{address: proto.AddressID{0xff}}).bytes()
	ts.newWavesData = v27BalanceHistory(t, wavesBalance,
		&wavesBalanceRecord{balanceProfile{Balance: 1 << 40, LeaseIn: 1, LeaseOut: 2}},
		&wavesBalanceRecord{balanceProfile{Balance: 0}})
	batch.Put(ts.newWavesKey, ts.newWavesData)
	ts.newAssetKey = (&assetBalanceKey{address: proto.AddressID{0xff}, asset: proto.AssetID{2}}).bytes()
	ts.newAssetData = v27BalanceHistory(t, assetBalance, &assetBalanceRecord{Balance: 1 << 33})
	batch.Put(ts.newAssetKey, ts.newAssetData)
	// Legacy state hashes of both formats.
	var prevSum []byte
	for h := uint64(1); h < v26LegacyHeights; h++ {
		sh := testLegacyStateHash(t, byte(h*10), prevSum) // #nosec G115 // test data.
		prevSum = sh.SumHash.Bytes()
		ts.hashes = append(ts.hashes, sh)
		data := v26LegacyStateHash(sh)
		if h%2 == 1 {
			data, err = sh.MarshalBinary()
			require.NoError(t, err)
		}
		batch.Put((&legacyStateHashKey{height: h}).bytes(), legacyStateHashHistory(t, data))
	}
	shV2 := &proto.StateHashV2{BlockID: genRandBlockIds(t, 1)[0], FieldsHashesV1: ts.hashes[0].FieldsHashesV1,
		GeneratorsHash: testDigest(1), GeneratorsBalancesHash: testDigest(2)}
	require.NoError(t, shV2.GenerateSumHash(prevSum))
	ts.hashV2, err = shV2.MarshalBinary()
	require.NoError(t, err)
	batch.Put((&legacyStateHashKey{height: v26LegacyHeights}).bytes(), legacyStateHashHistory(t, ts.hashV2))
	// Valid blocks, the legacy state hashes histories refer to the first block.
	for blockNum := uint32(1); blockNum <= v26ValidBlocks; blockNum++ {
		batch.Put((&validBlockNumKey{blockNum: blockNum}).bytes(), void)
	}
	// State info.
	info := stateInfo{Version: stateVersionV26, HasStateHashes: true}
	infoBytes, err := info.marshalBinary()
	require.NoError(t, err)
	batch.Put(stateInfoKeyBytes, infoBytes)
	require.NoError(t, db.Flush(batch))
	require.NoError(t, db.Close())
	require.NoError(t, writeDBMeta(ts.dataDir, info))
	return ts
}

func (ts *v26TestState) verify(t *testing.T) {
	info, err := readDBMeta(ts.dataDir)
	require.NoError(t, err)
	assert.Equal(t, uint16(StateVersion), info.Version)
	assert.True(t, info.HasStateHashes)

	db, err := keyvalue.NewKeyVal(filepath.Join(ts.dataDir, keyvalueDir), defaultTestKeyValParams())
	require.NoError(t, err)
	defer func() { require.NoError(t, db.Close()) }()

	dbInfo, err := readStateInfoFromDB(db)
	require.NoError(t, err)
	assert.Equal(t, uint16(StateVersion), dbInfo.Version)
	_, err = db.Get([]byte{migrationProgressKeyPrefix})
	assert.ErrorIs(t, err, keyvalue.ErrNotFound)

	for i, key := range ts.wavesKeys {
		data, gErr := db.Get(key)
		require.NoError(t, gErr)
		hr, hErr := newHistoryRecordFromBytes(data)
		require.NoError(t, hErr)
		require.Len(t, hr.entries, len(ts.wavesHistory[i]))
		for j, e := range hr.entries {
			var r wavesBalanceRecord
			require.NoError(t, r.unmarshalBinary(e.data))
			expected := ts.wavesHistory[i][j]
			assert.Equal(t, expected.blockNum, e.blockNum)
			assert.Equal(t, balanceProfile{Balance: expected.balance, LeaseIn: expected.leaseIn,
				LeaseOut: expected.leaseOut}, r.balanceProfile)
		}
	}

	data, err := db.Get(ts.assetKey)
	require.NoError(t, err)
	hr, err := newHistoryRecordFromBytes(data)
	require.NoError(t, err)
	require.Len(t, hr.entries, len(ts.assetHistory))
	for j, e := range hr.entries {
		var r assetBalanceRecord
		require.NoError(t, r.unmarshalBinary(e.data))
		assert.Equal(t, ts.assetHistory[j].balance, r.Balance)
		assert.Equal(t, ts.assetHistory[j].blockNum, e.blockNum)
	}

	data, err = db.Get(ts.newWavesKey)
	require.NoError(t, err)
	assert.Equal(t, ts.newWavesData, data)
	data, err = db.Get(ts.newAssetKey)
	require.NoError(t, err)
	assert.Equal(t, ts.newAssetData, data)

	for i, expected := range ts.hashes {
		data, err = db.Get((&legacyStateHashKey{height: uint64(i + 1)}).bytes()) // #nosec G115 // test data.
		require.NoError(t, err)
		hr, err = newHistoryRecordFromBytes(data)
		require.NoError(t, err)
		require.Len(t, hr.entries, 1)
		var sh proto.StateHashV1
		require.NoError(t, sh.UnmarshalBinary(hr.entries[0].data))
		assert.True(t, expected.Equal(&sh), "height %d", i+1)
	}
	data, err = db.Get((&legacyStateHashKey{height: v26LegacyHeights}).bytes())
	require.NoError(t, err)
	hr, err = newHistoryRecordFromBytes(data)
	require.NoError(t, err)
	require.Len(t, hr.entries, 1)
	assert.Equal(t, ts.hashV2, hr.entries[0].data)
}

func TestMigrateStateFromV26(t *testing.T) {
	ts := createV26TestState(t)
	require.NoError(t, MigrateState(ts.dataDir, defaultTestKeyValParams()))
	ts.verify(t)
	// Repeated migration does nothing.
	require.NoError(t, MigrateState(ts.dataDir, defaultTestKeyValParams()))
	ts.verify(t)
}

func TestMigrateStateFromV26Resume(t *testing.T) {
	ts := createV26TestState(t)
	// Simulate the interruption after the first batch of Waves balances: the records up to the progress key are
	// converted, the rest are not.
	db, err := keyvalue.NewKeyVal(filepath.Join(ts.dataDir, keyvalueDir), defaultTestKeyValParams())
	require.NoError(t, err)
	batch, err := db.NewBatch()
	require.NoError(t, err)
	const convertedNum = 100
	for i := range convertedNum {
		data, gErr := db.Get(ts.wavesKeys[i])
		require.NoError(t, gErr)
		converted, cErr := convertV26WavesBalanceHistory(data)
		require.NoError(t, cErr)
		batch.Put(ts.wavesKeys[i], converted)
	}
	require.NoError(t, flushWithProgress(db, batch,
		migrationProgress{Version: stateVersionV26, Step: 0, LastKey: ts.wavesKeys[convertedNum-1]}))
	require.NoError(t, db.Close())

	require.NoError(t, MigrateState(ts.dataDir, defaultTestKeyValParams()))
	ts.verify(t)
}

func TestMigrateStateFromV26UnknownLegacyStateHashFormat(t *testing.T) {
	ts := createV26TestState(t)
	db, err := keyvalue.NewKeyVal(filepath.Join(ts.dataDir, keyvalueDir), defaultTestKeyValParams())
	require.NoError(t, err)
	broken := *ts.hashes[1]
	broken.AliasesHash = testDigest(0xee) // SumHash doesn't match anymore
	batch, err := db.NewBatch()
	require.NoError(t, err)
	batch.Put((&legacyStateHashKey{height: 2}).bytes(), legacyStateHashHistory(t, v26LegacyStateHash(&broken)))
	require.NoError(t, db.Flush(batch))
	require.NoError(t, db.Close())

	err = MigrateState(ts.dataDir, defaultTestKeyValParams())
	require.ErrorContains(t, err, "legacy state hash at height 2")
	info, err := readDBMeta(ts.dataDir)
	require.NoError(t, err)
	assert.Equal(t, uint16(stateVersionV26), info.Version)
}

func TestMigrateStateFromV26MetaOfCurrentVersion(t *testing.T) {
	ts := createV26TestState(t)
	// The meta file is recreated by the node of the current version, but the database is not migrated.
	info, err := readDBMeta(ts.dataDir)
	require.NoError(t, err)
	info.Version = StateVersion
	require.NoError(t, writeDBMeta(ts.dataDir, info))

	require.NoError(t, MigrateState(ts.dataDir, defaultTestKeyValParams()))
	ts.verify(t)
}

func TestMigrateStateFromV26UnsupportedVersion(t *testing.T) {
	dataDir := t.TempDir()
	require.NoError(t, writeDBMeta(dataDir, stateInfo{Version: 25}))
	require.Error(t, MigrateState(dataDir, defaultTestKeyValParams()))
	_, err := os.Stat(filepath.Join(dataDir, keyvalueDir))
	assert.ErrorIs(t, err, fs.ErrNotExist, "database must not be created")
}

func TestConvertV26BalanceHistoryAmbiguous(t *testing.T) {
	// The history of balance 37060876330 at block 1 in the format of version 26 is also the valid history of
	// balance 42 at block 1 in the format of version 27.
	data, err := hex.DecodeString("0600000008a100182a00000001")
	require.NoError(t, err)
	v26 := v26AssetBalanceHistory(v26TestBalance{balance: 37060876330, blockNum: 1})
	require.Equal(t, v26, data)
	v27 := v27BalanceHistory(t, assetBalance, &assetBalanceRecord{Balance: 42})
	require.Equal(t, v27, data)

	_, err = convertV26AssetBalanceHistory(data)
	require.ErrorContains(t, err, "ambiguous history")
}

func TestConvertV26BalanceHistoryNotIncreasingBlockNums(t *testing.T) {
	data := v26AssetBalanceHistory(v26TestBalance{balance: 1, blockNum: 2}, v26TestBalance{balance: 2, blockNum: 2})
	_, err := convertV26AssetBalanceHistory(data)
	require.ErrorContains(t, err, "not increasing")
}

func TestMigrateStateFromV26AmbiguousBalanceHistory(t *testing.T) {
	ts := createV26TestState(t)
	db, err := keyvalue.NewKeyVal(filepath.Join(ts.dataDir, keyvalueDir), defaultTestKeyValParams())
	require.NoError(t, err)
	batch, err := db.NewBatch()
	require.NoError(t, err)
	key := (&assetBalanceKey{address: proto.AddressID{3}, asset: proto.AssetID{4}}).bytes()
	batch.Put(key, v26AssetBalanceHistory(v26TestBalance{balance: 37060876330, blockNum: 1}))
	require.NoError(t, db.Flush(batch))
	require.NoError(t, db.Close())

	err = MigrateState(ts.dataDir, defaultTestKeyValParams())
	require.ErrorContains(t, err, "ambiguous history")
	info, err := readDBMeta(ts.dataDir)
	require.NoError(t, err)
	assert.Equal(t, uint16(stateVersionV26), info.Version)
}

func TestMigrateStateFromV26OrphanLegacyStateHashes(t *testing.T) {
	ts := createV26TestState(t)
	db, err := keyvalue.NewKeyVal(filepath.Join(ts.dataDir, keyvalueDir), defaultTestKeyValParams())
	require.NoError(t, err)
	batch, err := db.NewBatch()
	require.NoError(t, err)
	// The hash of the rolled back block at the height after the last one, its previous SumHash is unknown.
	orphan := testLegacyStateHash(t, 0xa0, testDigest(0xaa).Bytes())
	orphanKey := (&legacyStateHashKey{height: v26LegacyHeights + 1}).bytes()
	hr := newHistoryRecord(legacyStateHash)
	hr.entries = append(hr.entries, historyEntry{data: v26LegacyStateHash(orphan), blockNum: v26ValidBlocks + 1})
	data, err := hr.marshalBinary()
	require.NoError(t, err)
	batch.Put(orphanKey, data)
	// The hash of the rolled back block on top of the valid hash at the height 2.
	validData, err := db.Get((&legacyStateHashKey{height: 2}).bytes())
	require.NoError(t, err)
	hr, err = newHistoryRecordFromBytes(validData)
	require.NoError(t, err)
	hr.entries = append(hr.entries, historyEntry{data: v26LegacyStateHash(orphan), blockNum: v26ValidBlocks + 2})
	data, err = hr.marshalBinary()
	require.NoError(t, err)
	batch.Put((&legacyStateHashKey{height: 2}).bytes(), data)
	require.NoError(t, db.Flush(batch))
	require.NoError(t, db.Close())

	require.NoError(t, MigrateState(ts.dataDir, defaultTestKeyValParams()))
	ts.verify(t)
	db, err = keyvalue.NewKeyVal(filepath.Join(ts.dataDir, keyvalueDir), defaultTestKeyValParams())
	require.NoError(t, err)
	defer func() { require.NoError(t, db.Close()) }()
	_, err = db.Get(orphanKey)
	assert.ErrorIs(t, err, keyvalue.ErrNotFound)
}

func TestMigrateStateKeepsStoredBloomFilter(t *testing.T) {
	ts := createV26TestState(t)
	bloomPath := filepath.Join(t.TempDir(), "bloom")
	content := []byte("stored bloom filter")
	require.NoError(t, os.WriteFile(bloomPath, content, 0600))
	params := defaultTestKeyValParams()
	params.BloomFilterStore = keyvalue.NewStore(bloomPath)

	require.NoError(t, MigrateState(ts.dataDir, params))
	ts.verify(t)
	data, err := os.ReadFile(bloomPath)
	require.NoError(t, err)
	assert.Equal(t, content, data)
}
