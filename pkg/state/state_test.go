package state

import (
	"context"
	"fmt"
	"math/big"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/mr-tron/base58"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"

	"github.com/wavesplatform/gowaves/pkg/consensus"
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

func newTestState(t *testing.T, amend bool, params StateParams, settings *settings.BlockchainSettings) State {
	dataDir := t.TempDir()
	m, err := NewState(dataDir, amend, params, settings, false)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, m.Close(), "manager.Close() failed")
	})
	return m
}

func newTestStateManager(t *testing.T, amend bool, params StateParams, settings *settings.BlockchainSettings) *stateManager {
	dataDir := t.TempDir()
	m, err := newStateManager(dataDir, amend, params, settings, false)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, m.Close(), "manager.Close() failed")
	})
	return m
}

func TestHandleAmendFlag(t *testing.T) {
	dataDir := t.TempDir()
	// first open with false amend
	manager, err := newStateManager(dataDir, false, DefaultTestingStateParams(), settings.MainNetSettings, false)
	assert.NoError(t, err, "newStateManager() failed")
	t.Cleanup(func() {
		assert.NoError(t, manager.Close(), "manager.Close() failed")
	})
	assert.False(t, manager.stor.hs.amend)

	// open with true amend
	assert.NoError(t, manager.Close(), "manager.Close() failed")
	manager, err = newStateManager(dataDir, true, DefaultTestingStateParams(), settings.MainNetSettings, false)
	assert.NoError(t, err, "newStateManager() failed")
	assert.True(t, manager.stor.hs.amend)

	// open with false amend again. Result amend should be true
	assert.NoError(t, manager.Close(), "manager.Close() failed")
	manager, err = newStateManager(dataDir, false, DefaultTestingStateParams(), settings.MainNetSettings, false)
	assert.NoError(t, err, "newStateManager() failed")
	assert.True(t, manager.stor.hs.amend)

	// first open with true amend
	newManager, err := newStateManager(t.TempDir(), true, DefaultTestingStateParams(), settings.MainNetSettings, false)
	assert.NoError(t, err, "newStateManager() failed")
	t.Cleanup(func() {
		assert.NoError(t, newManager.Close(), "newManager.Close() failed")
	})
	assert.True(t, newManager.stor.hs.amend)
}

func TestGenesisConfig(t *testing.T) {
	ss := &settings.BlockchainSettings{
		Type:                  settings.Custom,
		Genesis:               settings.TestNetSettings.Genesis,
		FunctionalitySettings: settings.FunctionalitySettings{BlockRewardTerm: 100000, AddressSchemeCharacter: proto.TestNetScheme},
	}
	stateParams := DefaultStateParams()
	stateParams.DbParams.Store = &keyvalue.NoOpStore{}

	manager := newTestStateManager(t, true, stateParams, ss)

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
		if _, err := st.ValidateNextTx(tx, timestamp, timestamp, 3, true); err != nil {
			return err
		}
	}
	st.ResetValidationList()
	return nil
}

func TestValidationWithoutBlocks(t *testing.T) {
	blocksPath, err := blocksPath()
	assert.NoError(t, err)
	manager := newTestStateManager(t, true, DefaultTestingStateParams(), settings.MainNetSettings)

	// Test txs from real block without this block.
	height := proto.Height(75)
	blocks, err := readBlocksFromTestPath(int(height + 1))
	assert.NoError(t, err, "readBlocksFromTestPath() failed")
	last := blocks[len(blocks)-1]
	txs := last.Transactions
	err = importer.ApplyFromFile(
		context.Background(),
		importer.ImportParams{Schema: proto.MainNetScheme, BlockchainPath: blocksPath, LightNodeMode: false},
		manager,
		height, 1)
	assert.NoError(t, err, "ApplyFromFile() failed")
	err = validateTxs(manager, last.Timestamp, txs)
	assert.NoError(t, err, "validateTxs() failed")

	// Test that in case validation using ValidateNextTx() fails,
	// its diffs are not taken into account for further validation.
	// This tx tries to send more Waves than exist at all.
	invalidTx := createPayment(t)
	invalidTx.Amount = 19999999500000000
	_, err = manager.ValidateNextTx(invalidTx, defaultTimestamp, defaultTimestamp, 3, true)
	assert.Error(t, err, "ValidateNextTx did not fail with invalid tx")
	// Now set some balance for sender.
	validTx := createPayment(t)
	err = manager.stateDB.addBlock(blockID0)
	assert.NoError(t, err, "addBlock() failed")
	waves := newWavesValueFromProfile(balanceProfile{validTx.Amount + validTx.Fee, 0, 0})
	err = manager.stor.balances.setWavesBalance(testGlobal.senderInfo.addr.ID(), waves, blockID0)
	assert.NoError(t, err, "setWavesBalance() failed")
	err = manager.flush()
	assert.NoError(t, err, "manager.flush() failed")
	// Valid tx with same sender must be valid after validation of previous invalid tx.
	_, err = manager.ValidateNextTx(validTx, defaultTimestamp, defaultTimestamp, 3, true)
	assert.NoError(t, err, "ValidateNextTx failed with valid tx")

	// Check NewestBalance() results after applying `validTx` from above.
	recipientBalance, err := manager.NewestWavesBalance(proto.NewRecipientFromAddress(testGlobal.recipientInfo.addr))
	assert.NoError(t, err, "manager.NewestAccountBalance() failed")
	assert.Equal(t, validTx.Amount, recipientBalance)
	senderBalance, err := manager.NewestWavesBalance(proto.NewRecipientFromAddress(testGlobal.senderInfo.addr))
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
	manager := newTestStateManager(t, true, DefaultTestingStateParams(), settings.MainNetSettings)

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

	for _, tc := range tests {
		height, hErr := manager.Height()
		if hErr != nil {
			t.Fatalf("Height(): %v\n", hErr)
		}
		if tc.nextHeight > height {
			if aErr := importer.ApplyFromFile(
				context.Background(),
				importer.ImportParams{Schema: proto.MainNetScheme, BlockchainPath: blocksPath, LightNodeMode: false},
				manager,
				tc.nextHeight-1, height,
			); aErr != nil {
				t.Fatalf("Failed to import: %v\n", aErr)
			}
		} else {
			if rErr := manager.RollbackToHeight(tc.nextHeight); rErr != nil {
				t.Fatalf("Rollback(): %v\n", rErr)
			}
		}
		if cErr := importer.CheckBalances(manager, tc.balancesPath); cErr != nil {
			t.Fatalf("CheckBalances(): %v\n", cErr)
		}
		if rErr := manager.RollbackToHeight(tc.minRollbackHeight - 1); rErr == nil {
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
	manager := newTestStateManager(t, true, DefaultTestingStateParams(), settings.MainNetSettings)

	tests := []testCase{
		{height: 901, score: bigFromStr("26588533320520"), path: filepath.Join(dir, "testdata", "accounts-901")},
		{height: 31, score: bigFromStr("2313166295294"), path: filepath.Join(dir, "testdata", "accounts-31")},
		{height: 1, score: bigFromStr("120000000219"), path: filepath.Join(dir, "testdata", "accounts-1")},
	}

	// Test what happens in case of failure: we add blocks starting from wrong height.
	// State should be rolled back to previous state and ready to use after.
	wrongStartHeight := uint64(100)
	if aErr := importer.ApplyFromFile(
		context.Background(),
		importer.ImportParams{Schema: proto.MainNetScheme, BlockchainPath: blocksPath, LightNodeMode: false},
		manager, blocksToImport, wrongStartHeight); aErr == nil {
		t.Errorf("Import starting from wrong height must fail but it doesn't.")
	}
	// Test normal import.
	if aErr := importer.ApplyFromFile(
		context.Background(),
		importer.ImportParams{Schema: proto.MainNetScheme, BlockchainPath: blocksPath, LightNodeMode: false},
		manager, blocksToImport, 1); aErr != nil {
		t.Fatalf("Failed to import: %v\n", aErr)
	}
	if cErr := importer.CheckBalances(manager, balancesPath); cErr != nil {
		t.Fatalf("CheckBalances(): %v\n", cErr)
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

func TestPreactivatedFeatures(t *testing.T) {
	blocksPath, err := blocksPath()
	assert.NoError(t, err)
	// Set preactivated feature.
	featureID := int16(1)
	sets := settings.MainNetSettings
	sets.PreactivatedFeatures = []int16{featureID}
	manager := newTestStateManager(t, true, DefaultTestingStateParams(), sets)

	// Check features status.
	activated, err := manager.IsActivated(featureID)
	assert.NoError(t, err, "IsActivated() failed")
	assert.Equal(t, true, activated)
	approved, err := manager.IsApproved(featureID)
	assert.NoError(t, err, "IsApproved() failed")
	assert.Equal(t, true, approved)
	// Apply blocks.
	height := uint64(75)
	err = importer.ApplyFromFile(
		context.Background(),
		importer.ImportParams{Schema: proto.MainNetScheme, BlockchainPath: blocksPath, LightNodeMode: false},
		manager, height, 1)
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
	manager := newTestStateManager(t, true, DefaultTestingStateParams(), settings.MainNetSettings)

	// Apply blocks.
	height := uint64(75)
	err = importer.ApplyFromFile(
		context.Background(),
		importer.ImportParams{Schema: proto.MainNetScheme, BlockchainPath: blocksPath, LightNodeMode: false},
		manager, height, 1)
	assert.NoError(t, err, "ApplyFromFile() failed")
	// Now validate tx with ID which is already in the state.
	tx := existingGenesisTx(t)
	txID, err := tx.GetID(settings.MainNetSettings.AddressSchemeCharacter)
	assert.NoError(t, err, "tx.GetID() failed")
	expectedErrStr := fmt.Sprintf("check duplicate tx ids: transaction with ID %s already in state", base58.Encode(txID))
	_, err = manager.ValidateNextTx(tx, 1460678400000, 1460678400000, 3, true)
	assert.Error(t, err, "duplicate transaction ID was accepted by state")
	assert.EqualError(t, err, expectedErrStr)
}

func TestTransactionByID(t *testing.T) {
	blocksPath, err := blocksPath()
	assert.NoError(t, err)
	manager := newTestStateManager(t, true, DefaultTestingStateParams(), settings.MainNetSettings)

	// Apply blocks.
	height := uint64(75)
	err = importer.ApplyFromFile(
		context.Background(),
		importer.ImportParams{Schema: proto.MainNetScheme, BlockchainPath: blocksPath, LightNodeMode: false},
		manager, height, 1)
	assert.NoError(t, err, "ApplyFromFile() failed")

	// Retrieve existing MainNet genesis tx by its ID.
	correctTx := existingGenesisTx(t)
	id, err := correctTx.GetID(settings.MainNetSettings.AddressSchemeCharacter)
	assert.NoError(t, err, "GetID() failed")
	tx, err := manager.TransactionByID(id)
	assert.NoError(t, err, "TransactionByID() failed")
	assert.Equal(t, correctTx, tx)
}

func TestStateManager_TopBlock(t *testing.T) {
	blocksPath, err := blocksPath()
	assert.NoError(t, err)
	dataDir := t.TempDir()
	manager, err := newStateManager(dataDir, true, DefaultTestingStateParams(), settings.MainNetSettings, false)
	assert.NoError(t, err, "newStateManager() failed")

	t.Cleanup(func() {
		err := manager.Close()
		assert.NoError(t, err, "manager.Close() failed")
	})

	genesis, err := manager.BlockByHeight(1)
	assert.NoError(t, err)
	assert.Equal(t, genesis, manager.TopBlock())

	height := proto.Height(100)
	err = importer.ApplyFromFile(
		context.Background(),
		importer.ImportParams{Schema: proto.MainNetScheme, BlockchainPath: blocksPath, LightNodeMode: false},
		manager, height-1, 1)
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
	manager, err = newStateManager(dataDir, true, DefaultTestingStateParams(), settings.MainNetSettings, false)
	assert.NoError(t, err, "newStateManager() failed")
	assert.Equal(t, correct, manager.TopBlock())
}

func TestGenesisStateHash(t *testing.T) {
	params := DefaultTestingStateParams()
	params.BuildStateHashes = true

	manager := newTestStateManager(t, true, params, settings.MainNetSettings)

	stateHash, err := manager.LegacyStateHashAtHeight(1)
	assert.NoError(t, err, "LegacyStateHashAtHeight failed")
	var correctHashJs = `
{"sponsorshipHash":"0e5751c026e543b2e8ab2eb06099daa1d1e5df47778f7787faab45cdf12fe3a8","blockId":"FSH8eAAzZNqnG8xgTZtz5xuLqXySsXgAjmFEC25hXMbEufiGjqWPnGCZFt6gLiVLJny16ipxRNAkkzjjhqTjBE2","wavesBalanceHash":"211af58aa42c72d0cf546d11d7b9141a00c8394e0f5da2d8e7e9f4ba30e9ad37","accountScriptHash":"0e5751c026e543b2e8ab2eb06099daa1d1e5df47778f7787faab45cdf12fe3a8","aliasHash":"0e5751c026e543b2e8ab2eb06099daa1d1e5df47778f7787faab45cdf12fe3a8","stateHash":"fab947262e8f5f03807ee7a888c750e46d0544a04d5777f50cc6daaf5f4e8d19","leaseStatusHash":"0e5751c026e543b2e8ab2eb06099daa1d1e5df47778f7787faab45cdf12fe3a8","dataEntryHash":"0e5751c026e543b2e8ab2eb06099daa1d1e5df47778f7787faab45cdf12fe3a8","assetBalanceHash":"0e5751c026e543b2e8ab2eb06099daa1d1e5df47778f7787faab45cdf12fe3a8","assetScriptHash":"0e5751c026e543b2e8ab2eb06099daa1d1e5df47778f7787faab45cdf12fe3a8","leaseBalanceHash":"0e5751c026e543b2e8ab2eb06099daa1d1e5df47778f7787faab45cdf12fe3a8"}`
	var correctHash proto.StateHash
	err = correctHash.UnmarshalJSON([]byte(correctHashJs))
	assert.NoError(t, err, "failed to unmarshal correct hash JSON")
	assert.Equal(t, correctHash, *stateHash)
}

func TestStateHashAtHeight(t *testing.T) {
	params := DefaultTestingStateParams()
	params.BuildStateHashes = true
	manager := newTestStateManager(t, false, params, settings.MainNetSettings)

	blocksPath, err := blocksPath()
	assert.NoError(t, err)
	err = importer.ApplyFromFile(
		context.Background(),
		importer.ImportParams{Schema: proto.MainNetScheme, BlockchainPath: blocksPath, LightNodeMode: false},
		manager, 9499, 1)
	assert.NoError(t, err, "ApplyFromFile() failed")
	stateHash, err := manager.LegacyStateHashAtHeight(9500)
	assert.NoError(t, err, "LegacyStateHashAtHeight failed")
	var correctHashJs = `
	{"sponsorshipHash":"0e5751c026e543b2e8ab2eb06099daa1d1e5df47778f7787faab45cdf12fe3a8","blockId":"2DYapXXAwxPm9WdYjS6bAY2n2fokGWeKmvHrcJy26uDfCFMognrwNEdtWEixaDxx3AahDKcdTDRNXmPVEtVumKjY","wavesBalanceHash":"0e5751c026e543b2e8ab2eb06099daa1d1e5df47778f7787faab45cdf12fe3a8","accountScriptHash":"0e5751c026e543b2e8ab2eb06099daa1d1e5df47778f7787faab45cdf12fe3a8","aliasHash":"0e5751c026e543b2e8ab2eb06099daa1d1e5df47778f7787faab45cdf12fe3a8","stateHash":"df48986cfee70960c977d741146ef4980ca71b20401db663eeff72c332fd8825","leaseStatusHash":"0e5751c026e543b2e8ab2eb06099daa1d1e5df47778f7787faab45cdf12fe3a8","dataEntryHash":"0e5751c026e543b2e8ab2eb06099daa1d1e5df47778f7787faab45cdf12fe3a8","assetBalanceHash":"0e5751c026e543b2e8ab2eb06099daa1d1e5df47778f7787faab45cdf12fe3a8","assetScriptHash":"0e5751c026e543b2e8ab2eb06099daa1d1e5df47778f7787faab45cdf12fe3a8","leaseBalanceHash":"0e5751c026e543b2e8ab2eb06099daa1d1e5df47778f7787faab45cdf12fe3a8"}`
	var correctHash proto.StateHash
	err = correctHash.UnmarshalJSON([]byte(correctHashJs))
	assert.NoError(t, err, "failed to unmarshal correct hash JSON")
	assert.Equal(t, correctHash, *stateHash)
}

type timeMock struct{}

func (timeMock) Now() time.Time { return time.Now().UTC() }

func TestGeneratingBalanceValuesForNewestFunctions(t *testing.T) {
	createMockStateManager := func(t *testing.T, bs *settings.BlockchainSettings) (*stateManager, *testStorageObjects) {
		const (
			handleAmend               = true
			calculateHashes           = true
			enableLightNode           = false
			verificationGoroutinesNum = 2
			provideExtendedAPI        = true
		)
		toOpts := testStorageObjectsOptions{Amend: handleAmend, Settings: bs}
		to := createStorageObjectsWithOptions(t, toOpts)

		stor, err := newBlockchainEntitiesStorage(to.hs, to.settings, to.rw, calculateHashes)
		require.NoError(t, err, "newBlockchainEntitiesStorage() failed")

		blockStorageDir := t.TempDir()
		atxParams := &addressTransactionsParams{
			dir:                 blockStorageDir,
			batchedStorMemLimit: proto.KiB,
			batchedStorMaxKeys:  AddressTransactionsMaxKeys,
			maxFileSize:         2 * proto.KiB,
			providesData:        provideExtendedAPI,
		}
		atx, err := newAddressTransactions(to.db, to.stateDB, to.rw, atxParams, handleAmend)
		require.NoError(t, err, "newAddressTransactions() failed")

		state := &stateManager{
			mu:                        new(sync.RWMutex),
			lastBlock:                 atomic.Value{},
			genesis:                   new(proto.Block), // stub
			stateDB:                   to.stateDB,
			stor:                      stor,
			rw:                        to.rw,
			settings:                  to.settings,
			cv:                        nil, // filled in later
			appender:                  nil, // filled in later
			atx:                       atx,
			verificationGoroutinesNum: verificationGoroutinesNum,
			newBlocks:                 newNewBlocks(to.rw, to.settings),
			enableLightNode:           enableLightNode,
		}
		snapshotApplier := newBlockSnapshotsApplier(nil, newSnapshotApplierStorages(stor, to.rw))
		appender, err := newTxAppender(
			state,
			state.rw,
			state.stor,
			state.settings,
			state.stateDB,
			state.atx,
			&snapshotApplier,
		)
		require.NoError(t, err, "newTxAppender() failed")
		state.appender = appender
		state.cv = consensus.NewValidator(state, state.settings, timeMock{})
		return state, to
	}
	const (
		initialBalance = 100
		changedBalance = 200
	)
	prepareStateCommon := func(t *testing.T, addr proto.WavesAddress) (*stateManager, *testStorageObjects) {
		const blocksToApply = 1000

		customSettings := *settings.MainNetSettings                      // copy the mainnet settings
		customSettings.GenerationBalanceDepthFrom50To1000AfterHeight = 1 // set from the first height
		state, testObj := createMockStateManager(t, &customSettings)

		// add initial balance at first block
		testObj.addBlock(t, blockID0)
		initialBP := newWavesValueFromProfile(balanceProfile{initialBalance, 0, 0})
		err := state.stor.balances.setWavesBalance(addr.ID(), initialBP, blockID0) // height 1
		require.NoError(t, err, "setWavesBalance() failed")
		// add changed balance at second block
		testObj.addBlock(t, blockID1)
		changedBP := newWavesValueFromProfile(balanceProfile{changedBalance, 0, 0})
		err = state.stor.balances.setWavesBalance(addr.ID(), changedBP, blockID1) // height 2
		require.NoError(t, err, "setWavesBalance() failed")

		testObj.addBlocks(t, blocksToApply-2) // add 998 random blocks, 2 blocks have already been added

		nh, err := state.NewestHeight()
		require.NoError(t, err, "NewestHeight() failed")
		require.Equal(t, uint64(blocksToApply), nh) // sanity check, blockchain height should be 1000
		ah, err := state.AddingBlockHeight()
		require.NoError(t, err, "AddingBlockHeight() failed")
		require.Equal(t, nh, ah) // sanity check, adding block height should be the same
		return state, testObj
	}

	addr, aErr := proto.NewAddressFromString(addr0)
	require.NoError(t, aErr, "NewAddressFromString() failed")

	t.Run("NewestFullWavesBalance", func(t *testing.T) {
		state, testObj := prepareStateCommon(t, addr)
		rcp := proto.NewRecipientFromAddress(addr) // convert address to recipient

		fb, err := state.NewestFullWavesBalance(rcp) // height 1000
		require.NoError(t, err, "NewestFullWavesBalance() failed")
		assert.Equal(t, uint64(initialBalance), fb.Generating)

		lastBlockIDToApply := blockID2

		// blockchain height now 1000, height for NewestFullWavesBalance is 1001
		// because for NewestFullWavesBalance we take into account applying block
		testObj.prepareAndStartBlock(t, lastBlockIDToApply)
		newFb, err := state.NewestFullWavesBalance(rcp)
		require.NoError(t, err, "NewestFullWavesBalance() failed")
		assert.Equal(t, uint64(changedBalance), newFb.Generating) // should be changed balance
		// finish the block, we are not in the applying block state
		testObj.finishBlock(t, lastBlockIDToApply) // blockchain height now 1001
		newFb, err = state.NewestFullWavesBalance(rcp)
		require.NoError(t, err, "NewestFullWavesBalance() failed")
		assert.Equal(t, uint64(changedBalance), newFb.Generating) // result should be the same
	})
	t.Run("WavesBalanceProfile", func(t *testing.T) {
		state, testObj := prepareStateCommon(t, addr)

		fb, err := state.WavesBalanceProfile(addr.ID()) // height 1000
		require.NoError(t, err, "WavesBalanceProfile() failed")
		assert.Equal(t, uint64(initialBalance), fb.Generating)

		lastBlockIDToApply := blockID2

		// blockchain height now 1000, height for NewestFullWavesBalance is 1001
		// because for NewestFullWavesBalance we take into account applying block
		testObj.prepareAndStartBlock(t, lastBlockIDToApply)
		newFb, err := state.WavesBalanceProfile(addr.ID())
		require.NoError(t, err, "WavesBalanceProfile() failed")
		assert.Equal(t, uint64(changedBalance), newFb.Generating) // should be changed balance
		// finish the block, we are not in the applying block state
		testObj.finishBlock(t, lastBlockIDToApply) // blockchain height now 1001
		newFb, err = state.WavesBalanceProfile(addr.ID())
		require.NoError(t, err, "WavesBalanceProfile() failed")
		assert.Equal(t, uint64(changedBalance), newFb.Generating) // result should be the same
	})
	t.Run("NewestGeneratingBalance", func(t *testing.T) {
		state, testObj := prepareStateCommon(t, addr)
		rcp := proto.NewRecipientFromAddress(addr) // convert address to recipient
		nh, err := state.NewestHeight()
		require.NoError(t, err, "NewestHeight() failed")

		gb, err := state.NewestGeneratingBalance(rcp, nh) // height 1000
		require.NoError(t, err, "NewestGeneratingBalance() failed")
		assert.Equal(t, uint64(initialBalance), gb)

		lastBlockIDToApply := blockID2

		// blockchain height now 1000, height for NewestFullWavesBalance is 1001
		// because for NewestFullWavesBalance we take into account applying block
		testObj.prepareAndStartBlock(t, lastBlockIDToApply)
		newGB, err := state.NewestGeneratingBalance(rcp, nh)
		require.NoError(t, err, "NewestGeneratingBalance() failed")
		assert.Equal(t, uint64(initialBalance), newGB) // should be initial balance, because nh == 1000
		// check with adding block height == 1001
		ah, err := state.AddingBlockHeight()
		require.NoError(t, err, "AddingBlockHeight() failed")
		require.Equal(t, nh+1, ah) // sanity check, adding block height should be the same
		newGB, err = state.NewestGeneratingBalance(rcp, ah)
		require.NoError(t, err, "NewestGeneratingBalance() failed")
		assert.Equal(t, uint64(changedBalance), newGB) // should be changed balance now

		// finish the block, we are not in the applying block state
		testObj.finishBlock(t, lastBlockIDToApply) // blockchain height now 1001
		nh, err = state.NewestHeight()
		require.NoError(t, err, "NewestHeight() failed")
		assert.Equal(t, ah, nh) // sanity check, blockchain height should be 1001
		newGB, err = state.NewestGeneratingBalance(rcp, nh)
		require.NoError(t, err, "NewestGeneratingBalance() failed")
		assert.Equal(t, uint64(changedBalance), newGB) // result should be the same
	})
}
