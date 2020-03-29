package state

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"os"
	"path/filepath"
	"testing"
)

const (
	scheme byte = 'T'
)

func openDB(t *testing.T, name string) (*leveldb.DB, func()) {
	path := filepath.Join(os.TempDir(), name)
	opts := opt.Options{ErrorIfExist: true}
	db, err := leveldb.OpenFile(path, &opts)
	assert.NoError(t, err)
	return db, func() {
		err = db.Close()
		assert.NoError(t, err)
		err = os.RemoveAll(path)
		assert.NoError(t, err)
	}
}

func TestBlocks(t *testing.T) {
	db, closeDB := openDB(t, "wmd-blocks-state-db")
	defer closeDB()

	sig1, err := crypto.NewSignatureFromBase58("4nCQbL7D8UZEUiZJxriGDSAY2YwyjLB4h6xCwCkm9Y5DDPKexta9aYR4vCtKsMK4VmovGa6wHGA4kVWCfRvkU87e")
	require.NoError(t, err)
	b1 := proto.NewBlockIDFromSignature(sig1)
	sig2, err := crypto.NewSignatureFromBase58("2iNPrfoUyF6CRxHrTFAT3H7dut1PQ3nJxuhNsuYbd3nCtaHvoXBdw5NqV77CGk6X8xmKkmKd1YsB4czzrWXZbusD")
	require.NoError(t, err)
	b2 := proto.NewBlockIDFromSignature(sig2)
	sig3, err := crypto.NewSignatureFromBase58("297ogBEykGSTASsAn5LUDoA58egcQ2JxPK9BV6jXh6oyoPcqrC2PjqwnqcP9ZD8fwvG9epCA7hJFN7syt8u1Cwa3")
	require.NoError(t, err)
	b3 := proto.NewBlockIDFromSignature(sig3)

	batch := new(leveldb.Batch)
	err = putBlock(batch, 1, b1)
	require.NoError(t, err)
	err = db.Write(batch, nil)
	require.NoError(t, err)
	if snapshot, err := db.GetSnapshot(); assert.NoError(t, err) {
		h, err := height(snapshot)
		require.NoError(t, err)
		assert.Equal(t, 1, h)
		b, ok, err := block(snapshot, 1)
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, b1, b)
	}

	batch = new(leveldb.Batch)
	err = putBlock(batch, 2, b2)
	require.NoError(t, err)
	err = db.Write(batch, nil)
	require.NoError(t, err)
	if snapshot, err := db.GetSnapshot(); assert.NoError(t, err) {
		h, err := height(snapshot)
		require.NoError(t, err)
		assert.Equal(t, 2, h)
		b, ok, err := block(snapshot, 1)
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, b1, b)
		b, ok, err = block(snapshot, 2)
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, b2, b)
	}

	batch = new(leveldb.Batch)
	err = putBlock(batch, 3, b3)
	require.NoError(t, err)
	err = db.Write(batch, nil)
	require.NoError(t, err)
	if snapshot, err := db.GetSnapshot(); assert.NoError(t, err) {
		h, err := height(snapshot)
		require.NoError(t, err)
		assert.Equal(t, 3, h)
		b, ok, err := block(snapshot, 1)
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, b1, b)
		b, ok, err = block(snapshot, 2)
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, b2, b)
		b, ok, err = block(snapshot, 3)
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, b3, b)
	}

	snapshot, err := db.GetSnapshot()
	require.NoError(t, err)
	batch = new(leveldb.Batch)
	err = rollbackBlocks(snapshot, batch, 2)
	require.NoError(t, err)
	err = db.Write(batch, nil)
	require.NoError(t, err)
	if snapshot, err := db.GetSnapshot(); assert.NoError(t, err) {
		h, err := height(snapshot)
		require.NoError(t, err)
		assert.Equal(t, 1, h)
		b, ok, err := block(snapshot, 1)
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, b1, b)
		_, ok, err = block(snapshot, 2)
		require.NoError(t, err)
		assert.False(t, ok)
		_, ok, err = block(snapshot, 3)
		require.NoError(t, err)
		assert.False(t, ok)
	}
}
