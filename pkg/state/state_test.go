package state

import (
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/importer"
)

const (
	blocksToImport = 1000
	startScore     = "28856275329634"
)

type testCase struct {
	height uint64
	score  *big.Int
	path   string
}

func getLocalDir() (string, error) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "", errors.Errorf("Unable to find current package file")
	}
	return filepath.Dir(filename), nil
}

func bigFromStr(s string) *big.Int {
	var big big.Int
	big.SetString(s, 10)
	return &big
}

func TestBlockAcceptAndRollback(t *testing.T) {
	dir, err := getLocalDir()
	if err != nil {
		t.Fatalf("Failed to get local dir: %v\n", err)
	}
	blocksPath := filepath.Join(dir, "testdata", "blocks-10000")
	balancesPath := filepath.Join(dir, "testdata", "accounts-1001")
	dataDir, err := ioutil.TempDir(os.TempDir(), "dataDir")
	if err != nil {
		t.Fatalf("Failed to create temp dir for data: %v\n", err)
	}
	manager, err := newStateManager(dataDir, DefaultBlockStorageParams())
	if err != nil {
		t.Fatalf("Failed to create state manager: %v.\n", err)
	}

	tests := []testCase{
		{height: 901, score: bigFromStr("26588533320520"), path: filepath.Join(dir, "testdata", "accounts-901")},
		{height: 31, score: bigFromStr("2313166295294"), path: filepath.Join(dir, "testdata", "accounts-31")},
		{height: 1, score: bigFromStr("120000000219"), path: filepath.Join(dir, "testdata", "accounts-1")},
	}

	defer func() {
		if err := manager.Close(); err != nil {
			t.Fatalf("Failed to close stateManager: %v\n", err)
		}
		if err := os.RemoveAll(dataDir); err != nil {
			t.Fatalf("Failed to clean dara dir: %v\n", err)
		}
	}()

	if err := importer.ApplyFromFile(manager, blocksPath, blocksToImport, 1); err != nil {
		t.Fatalf("Failed to import: %v\n", err)
	}
	if err := importer.CheckBalances(manager, balancesPath); err != nil {
		t.Fatalf("CheckBalances(): %v\n", err)
	}
	score, err := manager.ScoreAtHeight(blocksToImport + 1)
	if err != nil {
		t.Fatalf("ScoreAtHeight(): %v\n", err)
	}
	if score.Cmp(bigFromStr(startScore)) != 0 {
		t.Errorf("Scores are not equal.")
	}

	for _, tc := range tests {
		if err := manager.RollbackToHeight(tc.height); err != nil {
			t.Fatalf("Rollback(): %v\n", err)
		}
		if err := importer.CheckBalances(manager, tc.path); err != nil {
			t.Fatalf("CheckBalances(): %v\n", err)
		}
		score, err = manager.ScoreAtHeight(tc.height)
		if err != nil {
			t.Fatalf("ScoreAtHeight(): %v\n", err)
		}
		if score.Cmp(tc.score) != 0 {
			t.Errorf("Scores are not equal.")
		}
		height, err := manager.Height()
		if err != nil {
			t.Fatalf("Height(): %v\n", err)
		}
		if height != tc.height {
			t.Errorf("Height after rollback is not correct.")
		}
	}
}
