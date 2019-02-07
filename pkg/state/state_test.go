package state

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/pkg/errors"
)

const (
	BATCH_SIZE    = 1000
	BLOCKS_NUMBER = 1000
)

func getLocalDir() (string, error) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "", errors.Errorf("Unable to find current package file")
	}
	return filepath.Dir(filename), nil
}

func TestBlockAccept(t *testing.T) {
	dir, err := getLocalDir()
	if err != nil {
		t.Fatalf("Failed to get local dir: %v\n", err)
	}
	blocksPath := filepath.Join(dir, "..", "storage", "testdata", "blocks-10000")
	balancesPath := filepath.Join(dir, "testdata", "accounts-1000")
	if err := CheckState(blocksPath, balancesPath, BATCH_SIZE, BLOCKS_NUMBER); err != nil {
		t.Fatalf("CheckState: %v\n", err)
	}
}
