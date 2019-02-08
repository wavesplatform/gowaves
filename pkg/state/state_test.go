package state

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/storage"
	"github.com/wavesplatform/gowaves/pkg/util"
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
	rw, rwPath, err := storage.CreateTestBlockReadWriter(BATCH_SIZE, 8, 8)
	if err != nil {
		t.Fatalf("CreateTesBlockReadWriter: %v\n", err)
	}
	idsFile, err := rw.BlockIdsFilePath()
	if err != nil {
		t.Fatalf("Failed to get path of ids file: %v\n", err)
	}
	stor, storPath, err := storage.CreateTestAccountsStorage(idsFile)
	if err != nil {
		t.Fatalf("CreateTestAccountStorage: %v\n", err)
	}

	defer func() {
		if err := rw.Close(); err != nil {
			t.Fatalf("Failed to close BlockReadWriter: %v\n", err)
		}
		if err := util.CleanTemporaryDirs(rwPath); err != nil {
			t.Fatalf("Failed to clean data dirs: %v\n", err)
		}
		if err := util.CleanTemporaryDirs(storPath); err != nil {
			t.Fatalf("Failed to clean data dirs: %v\n", err)
		}
	}()

	manager, err := NewStateManager(stor, rw)
	if err != nil {
		t.Fatalf("Failed to create state manager: %v.\n", err)
	}
	if err := Apply(blocksPath, BLOCKS_NUMBER, manager, true); err != nil {
		t.Fatalf("Failed to import: %v\n", err)
	}
	if err := CheckBalances(balancesPath0, stor); err != nil {
		t.Fatalf("CheckBalances(): %v\n", err)
	}

	if err := manager.RollbackToHeight(FIRST_HEIGHT); err != nil {
		t.Fatalf("Rollback(): %v\n", err)
	}
	if err := CheckBalances(balancesPath1, stor); err != nil {
		t.Fatalf("CheckBalances(): %v\n", err)
	}

	if err := manager.RollbackToHeight(SECOND_HEIGHT); err != nil {
		t.Fatalf("Rollback(): %v\n", err)
	}
	if err := CheckBalances(balancesPath2, stor); err != nil {
		t.Fatalf("CheckBalances(): %v\n", err)
	}
}
