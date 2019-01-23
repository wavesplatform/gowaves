package state

import (
	"github.com/stretchr/testify/assert"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"os"
	"path/filepath"
	"testing"
)

const (
	scheme byte = 'T'
)

func TestHeightKey(t *testing.T) {
	k := heightKey{}
	b := k.bytes()
	assert.ElementsMatch(t, []byte{0x00}, b)
}

func TestBlockInfoKey(t *testing.T) {
	k := blockInfoKey{12345}
	b := k.bytes()
	assert.ElementsMatch(t, []byte{0x01, 0x00, 0x00, 0x30, 0x39}, b)
}

func TestBlockInfoBinaryRoundTrip(t *testing.T) {
	s := crypto.Signature{0x01, 0x02, 0x03, 0x04}
	bi1 := blockInfo{
		block:             s,
		empty:             true,
		earliestTimeFrame: 12345,
	}
	b := bi1.bytes()
	var bi2 blockInfo
	err := bi2.fromBytes(b)
	assert.NoError(t, err)
	assert.ElementsMatch(t, bi1.block, bi2.block)
	assert.Equal(t, bi1.empty, bi2.empty)
	assert.Equal(t, bi1.earliestTimeFrame, bi2.earliestTimeFrame)
}

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
