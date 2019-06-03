package state

import (
	"io/ioutil"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/importer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

const (
	maxRollbackTestBlocks = 9000
	blocksToImport        = 1000
	startScore            = "28856275329634"
)

type testCase struct {
	height uint64
	score  *big.Int
	path   string
}

func blocksPath(t *testing.T) string {
	dir, err := getLocalDir()
	assert.NoError(t, err, "getLocalDir() failed")
	return filepath.Join(dir, "testdata", "blocks-10000")
}

func bigFromStr(s string) *big.Int {
	var big big.Int
	big.SetString(s, 10)
	return &big
}

func TestGenesisConfig(t *testing.T) {
	dir, err := getLocalDir()
	if err != nil {
		t.Fatalf("Failed to get local dir: %v\n", err)
	}
	dataDir, err := ioutil.TempDir(os.TempDir(), "dataDir")
	ss := &settings.BlockchainSettings{
		Type:           settings.Custom,
		GenesisCfgPath: filepath.Join(dir, "genesis", "testnet.json"),
	}
	manager, err := newStateManager(dataDir, DefaultStateParams(), ss)
	if err != nil {
		t.Fatalf("Failed to create state manager: %v.\n", err)
	}

	defer func() {
		if err := manager.Close(); err != nil {
			t.Fatalf("Failed to close stateManager: %v\n", err)
		}
		if err := os.RemoveAll(dataDir); err != nil {
			t.Fatalf("Failed to clean dara dir: %v\n", err)
		}
	}()

	genesis, err := manager.BlockByHeight(1)
	if err != nil {
		t.Fatalf("Failed to get genesis block: %v\n", err)
	}
	if genesis.BlockSignature.String() != "5uqnLK3Z9eiot6FyYBfwUnbyid3abicQbAZjz38GQ1Q8XigQMxTK4C1zNkqS1SVw7FqSidbZKxWAKLVoEsp4nNqa" {
		t.Errorf("Genesis signature is not correct.")
	}
}

func validateTxs(st *stateManager, timestamp uint64, txs []proto.Transaction) error {
	for _, tx := range txs {
		if err := st.ValidateNextTx(tx, timestamp, timestamp); err != nil {
			return err
		}
	}
	st.ResetValidationList()
	return nil
}

func TestValidationWithoutBlocks(t *testing.T) {
	blocksPath := blocksPath(t)
	dataDir, err := ioutil.TempDir(os.TempDir(), "dataDir")
	assert.NoError(t, err, "failed to create dir for test data")
	manager, err := newStateManager(dataDir, DefaultStateParams(), settings.MainNetSettings)
	assert.NoError(t, err, "newStateManager() failed")

	defer func() {
		err := manager.Close()
		assert.NoError(t, err, "manager.Close() failed")
		err = os.RemoveAll(dataDir)
		assert.NoError(t, err, "failed to remove test data dirs")
	}()

	// Test txs from real block without this block.
	height := uint64(75)
	blocks, err := readRealBlocks(t, blocksPath, int(height+1))
	assert.NoError(t, err, "readRealBlocks() failed")
	last := blocks[len(blocks)-1]
	txs, err := proto.BytesToTransactions(last.TransactionCount, last.Transactions)
	assert.NoError(t, err, "BytesToTransactions() failed")
	err = importer.ApplyFromFile(manager, blocksPath, height, 1, false)
	assert.NoError(t, err, "ApplyFromFile() failed")
	err = validateTxs(manager, last.Timestamp, txs)
	assert.NoError(t, err, "validateTxs() failed")
}

func TestPreactivatedFeatures(t *testing.T) {
	blocksPath := blocksPath(t)
	dataDir, err := ioutil.TempDir(os.TempDir(), "dataDir")
	assert.NoError(t, err, "failed to create dir for test data")
	// Set preactivated feature.
	featureID := int16(1)
	sets := settings.MainNetSettings
	sets.PreactivatedFeatures = []int16{featureID}
	manager, err := newStateManager(dataDir, DefaultStateParams(), sets)
	assert.NoError(t, err, "newStateManager() failed")

	defer func() {
		err := manager.Close()
		assert.NoError(t, err, "manager.Close() failed")
		err = os.RemoveAll(dataDir)
		assert.NoError(t, err, "failed to remove test data dirs")
	}()

	// Check features status.
	activated, err := manager.IsActivated(featureID)
	assert.NoError(t, err, "IsActivated() failed")
	assert.Equal(t, true, activated)
	approved, err := manager.IsApproved(featureID)
	assert.NoError(t, err, "IsApproved() failed")
	assert.Equal(t, true, approved)
	// Apply blocks.
	height := uint64(75)
	err = importer.ApplyFromFile(manager, blocksPath, height, 1, false)
	assert.NoError(t, err, "ApplyFromFile() failed")
	// Check activation and approval heights.
	activationHeight, err := manager.ActivationHeight(featureID)
	assert.NoError(t, err, "ActivationHeight() failed")
	assert.Equal(t, uint64(1), activationHeight)
	approvalHeight, err := manager.ApprovalHeight(featureID)
	assert.NoError(t, err, "ApprovalHeight() failed")
	assert.Equal(t, uint64(1), approvalHeight)
}

func TestStateRollback(t *testing.T) {
	dir, err := getLocalDir()
	if err != nil {
		t.Fatalf("Failed to get local dir: %v\n", err)
	}
	blocksPath := blocksPath(t)
	dataDir, err := ioutil.TempDir(os.TempDir(), "dataDir")
	if err != nil {
		t.Fatalf("Failed to create temp dir for data: %v\n", err)
	}
	manager, err := newStateManager(dataDir, DefaultStateParams(), settings.MainNetSettings)
	if err != nil {
		t.Fatalf("Failed to create state manager: %v.\n", err)
	}

	tests := []struct {
		nextHeight        uint64
		minRollbackHeight uint64
		balancesPath      string
	}{
		{9001, 7001, filepath.Join(dir, "testdata", "accounts-9001")},
		{8001, 7001, filepath.Join(dir, "testdata", "accounts-8001")},
		{7001, 7001, filepath.Join(dir, "testdata", "accounts-7001")},
		{7501, 7001, filepath.Join(dir, "testdata", "accounts-7501")},
		{9501, 7501, filepath.Join(dir, "testdata", "accounts-9501")},
		{7501, 7501, filepath.Join(dir, "testdata", "accounts-7501")},
	}

	defer func() {
		if err := manager.Close(); err != nil {
			t.Fatalf("Failed to close stateManager: %v\n", err)
		}
		if err := os.RemoveAll(dataDir); err != nil {
			t.Fatalf("Failed to clean dara dir: %v\n", err)
		}
	}()

	for _, tc := range tests {
		height, err := manager.Height()
		if err != nil {
			t.Fatalf("Height(): %v\n", err)
		}
		if tc.nextHeight >= height {
			if err := importer.ApplyFromFile(manager, blocksPath, tc.nextHeight-1, height, false); err != nil {
				t.Fatalf("Failed to import: %v\n", err)
			}
		} else {
			if err := manager.RollbackToHeight(tc.nextHeight); err != nil {
				t.Fatalf("Rollback(): %v\n", err)
			}
		}
		if err := importer.CheckBalances(manager, tc.balancesPath); err != nil {
			t.Fatalf("CheckBalances(): %v\n", err)
		}
		if err := manager.RollbackToHeight(tc.minRollbackHeight - 1); err == nil {
			t.Fatalf("Rollback() did not fail with height less than minimum valid.")
		}
	}
}

func TestStateIntegrated(t *testing.T) {
	dir, err := getLocalDir()
	if err != nil {
		t.Fatalf("Failed to get local dir: %v\n", err)
	}
	blocksPath := blocksPath(t)
	balancesPath := filepath.Join(dir, "testdata", "accounts-1001")
	dataDir, err := ioutil.TempDir(os.TempDir(), "dataDir")
	if err != nil {
		t.Fatalf("Failed to create temp dir for data: %v\n", err)
	}
	manager, err := newStateManager(dataDir, DefaultStateParams(), settings.MainNetSettings)
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

	// Test what happens in case of failure: we add blocks starting from wrong height.
	// State should be rolled back to previous state and ready to use after.
	wrongStartHeight := uint64(100)
	if err := importer.ApplyFromFile(manager, blocksPath, blocksToImport, wrongStartHeight, false); err == nil {
		t.Errorf("Import starting from wrong height must fail but it doesn't.")
	}
	// Test normal import.
	if err := importer.ApplyFromFile(manager, blocksPath, blocksToImport, 1, false); err != nil {
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
	// Test rollback with wrong input.
	if err := manager.RollbackToHeight(0); err == nil {
		t.Fatalf("Rollback() did not fail with invalid input.")
	}
	if err := manager.RollbackToHeight(blocksToImport + 2); err == nil {
		t.Fatalf("Rollback() did not fail with invalid input.")
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

func TestStateManager_SavePeers(t *testing.T) {
	dataDir, err := ioutil.TempDir(os.TempDir(), "dataDir")
	if err != nil {
		t.Fatalf("Failed to create temp dir for data: %v\n", err)
	}
	defer os.RemoveAll(dataDir)

	manager, err := newStateManager(dataDir, DefaultStateParams(), settings.MainNetSettings)
	if err != nil {
		t.Fatalf("Failed to create state manager: %v.\n", err)
	}
	defer manager.Close()

	peers, err := manager.Peers()
	require.NoError(t, err)
	assert.Len(t, peers, 0)

	peers = []proto.TCPAddr{
		proto.NewTCPAddr(net.IPv4(127, 0, 0, 1), 65535),
		proto.NewTCPAddr(net.IPv4(83, 127, 1, 254).To4(), 80),
	}
	require.NoError(t, manager.SavePeers(peers))

	// check that peers saved
	peers2, err := manager.Peers()
	require.NoError(t, err)
	assert.Len(t, peers2, 2)
}
