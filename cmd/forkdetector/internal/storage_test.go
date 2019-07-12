package internal

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
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
