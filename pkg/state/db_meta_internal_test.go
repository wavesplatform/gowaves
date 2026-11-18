package state

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func requireOnlyDBMetaFile(t *testing.T, dataDir string) {
	entries, err := os.ReadDir(dataDir)
	require.NoError(t, err)
	require.Len(t, entries, 1, "temporary files must be removed")
	assert.Equal(t, dbMetaFileName, entries[0].Name())
}

func TestWriteDBMetaReplacesFile(t *testing.T) {
	dataDir := t.TempDir()
	long := stateInfo{Version: StateVersion, Amend: true, HasExtendedAPIData: true, HasStateHashes: true}
	short := stateInfo{Version: stateVersionV26}
	require.NoError(t, writeDBMeta(dataDir, long))
	// The shorter data must replace the longer one completely.
	require.NoError(t, writeDBMeta(dataDir, short))
	info, err := readDBMeta(dataDir)
	require.NoError(t, err)
	assert.Equal(t, short, info)
	requireOnlyDBMetaFile(t, dataDir)
}

func TestReadDBMetaMissingOrEmpty(t *testing.T) {
	dataDir := t.TempDir()
	_, err := readDBMeta(dataDir)
	require.Error(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dataDir, dbMetaFileName), nil, 0600))
	_, err = readDBMeta(dataDir)
	require.Error(t, err)
}

func TestCheckAndUpdateDBMeta(t *testing.T) {
	params := DefaultTestingStateParams()
	expected := stateInfo{
		Version:            StateVersion,
		HasExtendedAPIData: params.StoreExtendedApiData,
		HasStateHashes:     params.BuildStateHashes,
		DBCompressionAlgo:  params.DbParams.CompressionAlgo,
	}
	t.Run("missing", func(t *testing.T) {
		dataDir := t.TempDir()
		info, err := checkAndUpdateDBMeta(dataDir, false, params)
		require.NoError(t, err)
		assert.Equal(t, expected, info)
		stored, err := readDBMeta(dataDir)
		require.NoError(t, err)
		assert.Equal(t, expected, stored)
		requireOnlyDBMetaFile(t, dataDir)
	})
	t.Run("empty", func(t *testing.T) {
		dataDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dataDir, dbMetaFileName), nil, 0600))
		info, err := checkAndUpdateDBMeta(dataDir, false, params)
		require.NoError(t, err)
		assert.Equal(t, expected, info)
		requireOnlyDBMetaFile(t, dataDir)
	})
	t.Run("amend updated", func(t *testing.T) {
		dataDir := t.TempDir()
		require.NoError(t, writeDBMeta(dataDir, expected))
		info, err := checkAndUpdateDBMeta(dataDir, true, params)
		require.NoError(t, err)
		amended := expected
		amended.Amend = true
		assert.Equal(t, amended, info)
		stored, err := readDBMeta(dataDir)
		require.NoError(t, err)
		assert.Equal(t, amended, stored)
		requireOnlyDBMetaFile(t, dataDir)
	})
	t.Run("incompatible", func(t *testing.T) {
		dataDir := t.TempDir()
		incompatible := expected
		incompatible.HasStateHashes = !params.BuildStateHashes
		require.NoError(t, writeDBMeta(dataDir, incompatible))
		_, err := checkAndUpdateDBMeta(dataDir, false, params)
		require.Error(t, err)
		stored, err := readDBMeta(dataDir)
		require.NoError(t, err)
		assert.Equal(t, incompatible, stored, "meta file must not be changed")
	})
}
