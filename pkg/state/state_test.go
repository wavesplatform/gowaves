package state

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/importer"
)

const (
	BATCH_SIZE    = 1000
	BLOCKS_NUMBER = 1000
	FIRST_HEIGHT  = 900
	SECOND_HEIGHT = 30
)

func getLocalDir() (string, error) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "", errors.Errorf("Unable to find current package file")
	}
	return filepath.Dir(filename), nil
}

func TestBlockAcceptAndRollback(t *testing.T) {
	dir, err := getLocalDir()
	if err != nil {
		t.Fatalf("Failed to get local dir: %v\n", err)
	}
	blocksPath := filepath.Join(dir, "..", "storage", "testdata", "blocks-10000")
	balancesPath0 := filepath.Join(dir, "testdata", "accounts-1000")
	balancesPath1 := filepath.Join(dir, "testdata", "accounts-900")
	balancesPath2 := filepath.Join(dir, "testdata", "accounts-30")
	dataDir, err := ioutil.TempDir(os.TempDir(), "dataDir")
	if err != nil {
		t.Fatalf("Failed to create temp dir for data: %v\n", err)
	}
	manager, err := NewStateManager(dataDir, DefaultBlockStorageParams())
	if err != nil {
		t.Fatalf("Failed to create state manager: %v.\n", err)
	}

	defer func() {
		if err := manager.Close(); err != nil {
			t.Fatalf("Failed to close StateManager: %v\n", err)
		}
		if err := os.RemoveAll(dataDir); err != nil {
			t.Fatalf("Failed to clean dara dir: %v\n", err)
		}
	}()

	if err := importer.ApplyFromFile(manager, blocksPath, BLOCKS_NUMBER, true); err != nil {
		t.Fatalf("Failed to import: %v\n", err)
	}
	if err := importer.CheckBalances(manager, balancesPath0); err != nil {
		t.Fatalf("CheckBalances(): %v\n", err)
	}

	if err := manager.RollbackToHeight(FIRST_HEIGHT); err != nil {
		t.Fatalf("Rollback(): %v\n", err)
	}
	if err := importer.CheckBalances(manager, balancesPath1); err != nil {
		t.Fatalf("CheckBalances(): %v\n", err)
	}

	if err := manager.RollbackToHeight(SECOND_HEIGHT); err != nil {
		t.Fatalf("Rollback(): %v\n", err)
	}
	if err := importer.CheckBalances(manager, balancesPath2); err != nil {
		t.Fatalf("CheckBalances(): %v\n", err)
	}
}
