package state

import (
	"fmt"
	"io/ioutil"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/mr-tron/base58"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/importer"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
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

func bigFromStr(s string) *big.Int {
	var i big.Int
	i.SetString(s, 10)
	return &i
}

func TestGenesisConfig(t *testing.T) {
	dataDir, err := ioutil.TempDir(os.TempDir(), "dataDir")
	if err != nil {
		t.Fatalf("TempDir(): %v.\n", err)
	}
	ss := &settings.BlockchainSettings{
		Type:                  settings.Custom,
		Genesis:               settings.TestNetSettings.Genesis,
		FunctionalitySettings: settings.FunctionalitySettings{BlockRewardTerm: 100000},
	}
	stateParams := DefaultStateParams()
	stateParams.DbParams.Store = &keyvalue.NoOpStore{}
	manager, err := newStateManager(dataDir, stateParams, ss)
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
		if err := st.ValidateNextTx(tx, timestamp, timestamp, 3, true); err != nil {
			return err
		}
	}
	st.ResetValidationList()
	return nil
}

func TestValidationWithoutBlocks(t *testing.T) {
	blocksPath, err := blocksPath()
	assert.NoError(t, err)
	dataDir, err := ioutil.TempDir(os.TempDir(), "dataDir")
	assert.NoError(t, err, "failed to create dir for test data")
	manager, err := newStateManager(dataDir, DefaultTestingStateParams(), settings.MainNetSettings)
	assert.NoError(t, err, "newStateManager() failed")

	defer func() {
		err := manager.Close()
		assert.NoError(t, err, "manager.Close() failed")
		err = os.RemoveAll(dataDir)
		assert.NoError(t, err, "failed to remove test data dirs")
	}()

	// Test txs from real block without this block.
	height := proto.Height(75)
	blocks, err := readBlocksFromTestPath(int(height + 1))
	assert.NoError(t, err, "readBlocksFromTestPath() failed")
	last := blocks[len(blocks)-1]
	txs := last.Transactions
	err = importer.ApplyFromFile(manager, blocksPath, height, 1, false)
	assert.NoError(t, err, "ApplyFromFile() failed")
	err = validateTxs(manager, last.Timestamp, txs)
	assert.NoError(t, err, "validateTxs() failed")

	// Test that in case validation using ValidateNextTx() fails,
	// its diffs are not taken into account for further validation.
	// This tx tries to send more Waves than exist at all.
	invalidTx := createPayment(t)
	invalidTx.Amount = 19999999500000000
	err = manager.ValidateNextTx(invalidTx, defaultTimestamp, defaultTimestamp, 3, true)
	assert.Error(t, err, "ValidateNextTx did not fail with invalid tx")
	// Now set some balance for sender.
	validTx := createPayment(t)
	err = manager.stateDB.addBlock(blockID0)
	assert.NoError(t, err, "addBlock() failed")
	waves := newWavesValueFromProfile(balanceProfile{validTx.Amount + validTx.Fee, 0, 0})
	err = manager.stor.balances.setWavesBalance(testGlobal.senderInfo.addr, waves, blockID0)
	assert.NoError(t, err, "setWavesBalance() failed")
	err = manager.flush(false)
	assert.NoError(t, err, "manager.flush() failed")
	// Valid tx with same sender must be valid after validation of previous invalid tx.
	err = manager.ValidateNextTx(validTx, defaultTimestamp, defaultTimestamp, 3, true)
	assert.NoError(t, err, "ValidateNextTx failed with valid tx")

	// Check NewestBalance() results after applying `validTx` from above.
	recipientBalance, err := manager.NewestAccountBalance(proto.NewRecipientFromAddress(testGlobal.recipientInfo.addr), nil)
	assert.NoError(t, err, "manager.NewestAccountBalance() failed")
	assert.Equal(t, validTx.Amount, recipientBalance)
	senderBalance, err := manager.NewestAccountBalance(proto.NewRecipientFromAddress(testGlobal.senderInfo.addr), nil)
	assert.NoError(t, err, "manager.NewestAccountBalance() failed")
	assert.Equal(t, uint64(0), senderBalance)
}

func TestStateRollback(t *testing.T) {
	dir, err := getLocalDir()
	if err != nil {
		t.Fatalf("Failed to get local dir: %v\n", err)
	}
	blocksPath, err := blocksPath()
	assert.NoError(t, err)
	dataDir, err := ioutil.TempDir(os.TempDir(), "dataDir")
	if err != nil {
		t.Fatalf("Failed to create temp dir for data: %v\n", err)
	}
	manager, err := newStateManager(dataDir, DefaultTestingStateParams(), settings.MainNetSettings)
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
		if tc.nextHeight > height {
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
	blocksPath, err := blocksPath()
	assert.NoError(t, err)
	balancesPath := filepath.Join(dir, "testdata", "accounts-1001")
	dataDir, err := ioutil.TempDir(os.TempDir(), "dataDir")
	if err != nil {
		t.Fatalf("Failed to create temp dir for data: %v\n", err)
	}
	manager, err := newStateManager(dataDir, DefaultTestingStateParams(), settings.MainNetSettings)
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
			t.Errorf("Height after rollback is not correct: %d; must be %d", height, tc.height)
		}
		height, err = manager.NewestHeight()
		if err != nil {
			t.Fatalf("NewestHeight(): %v\n", err)
		}
		if height != tc.height {
			t.Errorf("Height after rollback is not correct: %d; must be %d", height, tc.height)
		}
	}
}

func TestStateManager_SavePeers(t *testing.T) {
	dataDir, err := ioutil.TempDir(os.TempDir(), "dataDir")
	if err != nil {
		t.Fatalf("Failed to create temp dir for data: %v\n", err)
	}
	defer func() {
		err = os.RemoveAll(dataDir)
		require.NoError(t, err)
	}()

	manager, err := newStateManager(dataDir, DefaultTestingStateParams(), settings.MainNetSettings)
	if err != nil {
		t.Fatalf("Failed to create state manager: %v.\n", err)
	}
	defer func() {
		err := manager.Close()
		require.NoError(t, err)
	}()

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

func TestPreactivatedFeatures(t *testing.T) {
	blocksPath, err := blocksPath()
	assert.NoError(t, err)
	dataDir, err := ioutil.TempDir(os.TempDir(), "dataDir")
	assert.NoError(t, err, "failed to create dir for test data")
	// Set preactivated feature.
	featureID := int16(1)
	sets := settings.MainNetSettings
	sets.PreactivatedFeatures = []int16{featureID}
	manager, err := newStateManager(dataDir, DefaultTestingStateParams(), sets)
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

func TestDisallowDuplicateTxIds(t *testing.T) {
	blocksPath, err := blocksPath()
	assert.NoError(t, err)
	dataDir, err := ioutil.TempDir(os.TempDir(), "dataDir")
	assert.NoError(t, err, "failed to create dir for test data")
	manager, err := newStateManager(dataDir, DefaultTestingStateParams(), settings.MainNetSettings)
	assert.NoError(t, err, "newStateManager() failed")

	defer func() {
		err := manager.Close()
		assert.NoError(t, err, "manager.Close() failed")
		err = os.RemoveAll(dataDir)
		assert.NoError(t, err, "failed to remove test data dirs")
	}()

	// Apply blocks.
	height := uint64(75)
	err = importer.ApplyFromFile(manager, blocksPath, height, 1, false)
	assert.NoError(t, err, "ApplyFromFile() failed")
	// Now validate tx with ID which is already in the state.
	tx := existingGenesisTx(t)
	txID, err := tx.GetID(settings.MainNetSettings.AddressSchemeCharacter)
	assert.NoError(t, err, "tx.GetID() failed")
	expectedErrStr := fmt.Sprintf("check duplicate tx ids: transaction with ID %s already in state", base58.Encode(txID))
	err = manager.ValidateNextTx(tx, 1460678400000, 1460678400000, 3, true)
	assert.Error(t, err, "duplicate transaction ID was accepted by state")
	assert.EqualError(t, err, expectedErrStr)
}

func TestTransactionByID(t *testing.T) {
	blocksPath, err := blocksPath()
	assert.NoError(t, err)
	dataDir, err := ioutil.TempDir(os.TempDir(), "dataDir")
	assert.NoError(t, err, "failed to create dir for test data")
	manager, err := newStateManager(dataDir, DefaultTestingStateParams(), settings.MainNetSettings)
	assert.NoError(t, err, "newStateManager() failed")

	defer func() {
		err := manager.Close()
		assert.NoError(t, err, "manager.Close() failed")
		err = os.RemoveAll(dataDir)
		assert.NoError(t, err, "failed to remove test data dirs")
	}()

	// Apply blocks.
	height := uint64(75)
	err = importer.ApplyFromFile(manager, blocksPath, height, 1, false)
	assert.NoError(t, err, "ApplyFromFile() failed")

	// Retrieve existing MainNet genesis tx by its ID.
	correctTx := existingGenesisTx(t)
	id, err := correctTx.GetID(settings.MainNetSettings.AddressSchemeCharacter)
	assert.NoError(t, err, "GetID() failed")
	tx, err := manager.TransactionByID(id)
	assert.NoError(t, err, "TransactionByID() failed")
	assert.Equal(t, correctTx, tx)
}

func TestStateManager_Mutex(t *testing.T) {
	dataDir, err := ioutil.TempDir(os.TempDir(), "dataDir")
	if err != nil {
		t.Fatalf("Failed to create temp dir for data: %v\n", err)
	}
	defer func() {
		err := os.RemoveAll(dataDir)
		require.NoError(t, err)
	}()

	manager, err := newStateManager(dataDir, DefaultTestingStateParams(), settings.MainNetSettings)
	if err != nil {
		t.Fatalf("Failed to create state manager: %v.\n", err)
	}
	defer func() {
		err := manager.Close()
		require.NoError(t, err)
	}()

	mu := manager.Mutex()
	mu.Lock().Unlock()
}

func TestStateManager_TopBlock(t *testing.T) {
	blocksPath, err := blocksPath()
	assert.NoError(t, err)
	dataDir, err := ioutil.TempDir(os.TempDir(), "dataDir")
	assert.NoError(t, err, "failed to create dir for test data")
	manager, err := newStateManager(dataDir, DefaultTestingStateParams(), settings.MainNetSettings)
	assert.NoError(t, err, "newStateManager() failed")

	defer func() {
		err := manager.Close()
		assert.NoError(t, err, "manager.Close() failed")
		err = os.RemoveAll(dataDir)
		assert.NoError(t, err, "failed to remove test data dirs")
	}()

	genesis, err := manager.BlockByHeight(1)
	assert.NoError(t, err)
	assert.Equal(t, genesis, manager.TopBlock())

	height := proto.Height(100)
	err = importer.ApplyFromFile(manager, blocksPath, height-1, 1, false)
	assert.NoError(t, err, "ApplyFromFile() failed")

	correct, err := manager.BlockByHeight(height)
	assert.NoError(t, err)
	assert.Equal(t, correct, manager.TopBlock())

	height = proto.Height(30)
	err = manager.RollbackToHeight(height)
	assert.NoError(t, err)

	correct, err = manager.BlockByHeight(height)
	assert.NoError(t, err)
	assert.Equal(t, correct, manager.TopBlock())

	// Test after closure.
	err = manager.Close()
	assert.NoError(t, err, "manager.Close() failed")

	manager, err = newStateManager(dataDir, DefaultTestingStateParams(), settings.MainNetSettings)
	assert.NoError(t, err, "newStateManager() failed")
	assert.Equal(t, correct, manager.TopBlock())
}

func TestGenesisStateHash(t *testing.T) {
	dataDir, err := ioutil.TempDir(os.TempDir(), "dataDir")
	assert.NoError(t, err, "failed to create dir for test data")
	params := DefaultTestingStateParams()
	params.BuildStateHashes = true
	manager, err := newStateManager(dataDir, params, settings.MainNetSettings)
	assert.NoError(t, err, "newStateManager() failed")

	defer func() {
		err := manager.Close()
		assert.NoError(t, err, "manager.Close() failed")
		err = os.RemoveAll(dataDir)
		assert.NoError(t, err, "failed to remove test data dirs")
	}()

	stateHash, err := manager.StateHashAtHeight(1)
	assert.NoError(t, err, "StateHashAtHeight failed")
	var correctHashJs = `
{"sponsorshipHash":"0e5751c026e543b2e8ab2eb06099daa1d1e5df47778f7787faab45cdf12fe3a8","blockId":"FSH8eAAzZNqnG8xgTZtz5xuLqXySsXgAjmFEC25hXMbEufiGjqWPnGCZFt6gLiVLJny16ipxRNAkkzjjhqTjBE2","wavesBalanceHash":"211af58aa42c72d0cf546d11d7b9141a00c8394e0f5da2d8e7e9f4ba30e9ad37","accountScriptHash":"0e5751c026e543b2e8ab2eb06099daa1d1e5df47778f7787faab45cdf12fe3a8","aliasHash":"0e5751c026e543b2e8ab2eb06099daa1d1e5df47778f7787faab45cdf12fe3a8","stateHash":"fab947262e8f5f03807ee7a888c750e46d0544a04d5777f50cc6daaf5f4e8d19","leaseStatusHash":"0e5751c026e543b2e8ab2eb06099daa1d1e5df47778f7787faab45cdf12fe3a8","dataEntryHash":"0e5751c026e543b2e8ab2eb06099daa1d1e5df47778f7787faab45cdf12fe3a8","assetBalanceHash":"0e5751c026e543b2e8ab2eb06099daa1d1e5df47778f7787faab45cdf12fe3a8","assetScriptHash":"0e5751c026e543b2e8ab2eb06099daa1d1e5df47778f7787faab45cdf12fe3a8","leaseBalanceHash":"0e5751c026e543b2e8ab2eb06099daa1d1e5df47778f7787faab45cdf12fe3a8"}`
	var correctHash proto.StateHash
	err = correctHash.UnmarshalJSON([]byte(correctHashJs))
	assert.NoError(t, err, "failed to unmarshal correct hash JSON")
	assert.Equal(t, correctHash, *stateHash)
}

func TestStateHashAtHeight(t *testing.T) {
	dataDir, err := ioutil.TempDir(os.TempDir(), "dataDir")
	assert.NoError(t, err, "failed to create dir for test data")
	params := DefaultTestingStateParams()
	params.BuildStateHashes = true
	manager, err := newStateManager(dataDir, params, settings.MainNetSettings)
	assert.NoError(t, err, "newStateManager() failed")

	defer func() {
		err := manager.Close()
		assert.NoError(t, err, "manager.Close() failed")
		err = os.RemoveAll(dataDir)
		assert.NoError(t, err, "failed to remove test data dirs")
	}()

	blocksPath, err := blocksPath()
	assert.NoError(t, err)
	err = importer.ApplyFromFile(manager, blocksPath, 9499, 1, true)
	assert.NoError(t, err, "ApplyFromFile() failed")
	stateHash, err := manager.StateHashAtHeight(9500)
	assert.NoError(t, err, "StateHashAtHeight failed")
	var correctHashJs = `
	{"sponsorshipHash":"0e5751c026e543b2e8ab2eb06099daa1d1e5df47778f7787faab45cdf12fe3a8","blockId":"2DYapXXAwxPm9WdYjS6bAY2n2fokGWeKmvHrcJy26uDfCFMognrwNEdtWEixaDxx3AahDKcdTDRNXmPVEtVumKjY","wavesBalanceHash":"0e5751c026e543b2e8ab2eb06099daa1d1e5df47778f7787faab45cdf12fe3a8","accountScriptHash":"0e5751c026e543b2e8ab2eb06099daa1d1e5df47778f7787faab45cdf12fe3a8","aliasHash":"0e5751c026e543b2e8ab2eb06099daa1d1e5df47778f7787faab45cdf12fe3a8","stateHash":"df48986cfee70960c977d741146ef4980ca71b20401db663eeff72c332fd8825","leaseStatusHash":"0e5751c026e543b2e8ab2eb06099daa1d1e5df47778f7787faab45cdf12fe3a8","dataEntryHash":"0e5751c026e543b2e8ab2eb06099daa1d1e5df47778f7787faab45cdf12fe3a8","assetBalanceHash":"0e5751c026e543b2e8ab2eb06099daa1d1e5df47778f7787faab45cdf12fe3a8","assetScriptHash":"0e5751c026e543b2e8ab2eb06099daa1d1e5df47778f7787faab45cdf12fe3a8","leaseBalanceHash":"0e5751c026e543b2e8ab2eb06099daa1d1e5df47778f7787faab45cdf12fe3a8"}`
	var correctHash proto.StateHash
	err = correctHash.UnmarshalJSON([]byte(correctHashJs))
	assert.NoError(t, err, "failed to unmarshal correct hash JSON")
	assert.Equal(t, correctHash, *stateHash)
}
