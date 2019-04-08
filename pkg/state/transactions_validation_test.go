package state

import (
	"testing"

	"github.com/mr-tron/base58/base58"
	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/util"
)

var (
	genesisTimestamp = uint64(1465742577614)
	timestamp0       = settings.MainNetSettings.NegativeBalanceCheckAfterTime + 1
	timestamp1       = settings.MainNetSettings.NegativeBalanceCheckAfterTime - 1

	blockID0 = "FSH8eAAzZNqnG8xgTZtz5xuLqXySsXgAjmFEC25hXMbEufiGjqWPnGCZFt6gLiVLJny16ipxRNAkkzjjhqTjBE1"
	blockID1 = "FSH8eAAzZNqnG8xgTZtz5xuLqXySsXgAjmFEC25hXMbEufiGjqWPnGCZFt6gLiVLJny16ipxRNAkkzjjhqTjBE3"

	matcherPK     = "AfZtLRQxLNYH5iradMkTeuXGe71uAiATVbr8DpXEEQa6"
	matcherAddr   = "3P9MUoSW7jfHNVFcq84rurfdWZYZuvVghVi"
	minerPK       = "AfZtLRQxLNYH5iradMkTeuXGe71uAiATVbr8DpXEEQa7"
	minerAddr     = "3PP2ywCpyvC57rN4vUZhJjQrmGMTWnjFKi7"
	senderPK      = "AfZtLRQxLNYH5iradMkTeuXGe71uAiATVbr8DpXEEQa8"
	senderAddr    = "3PNXHYoWp83VaWudq9ds9LpS5xykWuJHiHp"
	recipientPK   = "AfZtLRQxLNYH5iradMkTeuXGe71uAiATVbr8DpXEEQa9"
	recipientAddr = "3PDdGex1meSUf4Yq5bjPBpyAbx6us9PaLfo"

	assetStr = "B2u2TBpTYHWCuMuKLnbQfLvdLJ3zjgPiy3iMS2TSYugZ"
)

type testObjects struct {
	assets   *assets
	balances *balances
	tv       *transactionValidator
}

func createTestObjects(t *testing.T) (*testObjects, []string) {
	assets, path, err := createAssets()
	assert.NoError(t, err, "createAssets() failed")
	balances, err := newBalances(assets.db, assets.dbBatch, &mock{}, &mockBlockInfo{})
	assert.NoError(t, err, "newBalances() failed")
	genesisSig, err := crypto.NewSignatureFromBase58(genesisSignature)
	assert.NoError(t, err, "NewSignatureFromBase58() failed")
	tv, err := newTransactionValidator(genesisSig, balances, assets, settings.MainNetSettings)
	assert.NoError(t, err, "newTransactionValidator() failed")
	return &testObjects{assets: assets, balances: balances, tv: tv}, path
}

func (to *testObjects) reset() {
	to.assets.reset()
	to.balances.reset()
	to.tv.reset()
}

func flushBalances(t *testing.T, balances *balances) {
	err := balances.flush()
	assert.NoError(t, err, "balances.flush() failed")
	balances.reset()
}

type balanceDiff struct {
	address     string
	asset       string
	prevBalance uint64
	newBalance  uint64
}

func key(t *testing.T, addr, asset string) []byte {
	address, err := proto.NewAddressFromString(addr)
	assert.NoError(t, err, "NewAddressFromString() failed")
	ast, err := proto.NewOptionalAssetFromString(asset)
	assert.NoError(t, err, "NewOptionalAssetFromString() failed")
	balanceKey := balanceKey{address: address, asset: ast.ToID()}
	return balanceKey.bytes()
}

func setBalances(t *testing.T, to *testObjects, balanceDiffs []balanceDiff) {
	for _, diff := range balanceDiffs {
		balanceKey := key(t, diff.address, diff.asset)
		err := to.balances.setAccountBalance(balanceKey, diff.prevBalance, crypto.Signature{})
		assert.NoError(t, err, "setAccountBalance() failed")
	}
	flushBalances(t, to.balances)
	flushAssets(t, to.assets)
}

func checkBalances(t *testing.T, balances *balances, balanceDiffs []balanceDiff) {
	for _, diff := range balanceDiffs {
		balanceKey := key(t, diff.address, diff.asset)
		balance, err := balances.accountBalance(balanceKey)
		assert.NoError(t, err, "accountBalance() failed")
		assert.Equalf(t, balance, diff.newBalance, "invalid balance after validation: must be %d, is: %d", diff.newBalance, balance)
	}
}

func blankBlocks(t *testing.T, timestamp uint64, blockID crypto.Signature) (*proto.Block, *proto.Block) {
	blank := new(proto.Block)
	mpk, err := crypto.NewPublicKeyFromBase58(minerPK)
	assert.NoError(t, err, "NewPublicKeyFromBase58() failed")
	blank.GenPublicKey = mpk
	blank.Timestamp = timestamp
	blank.BlockSignature = blockID
	return blank, blank
}

type block struct {
	timestamp uint64
	sig       string
}

func validateTx(t *testing.T, tv *transactionValidator, tx proto.Transaction, blocks []block, checkTimestamp bool) {
	for _, b := range blocks {
		blockID, err := crypto.NewSignatureFromBase58(b.sig)
		assert.NoError(t, err, "NewSignatureFromBase58() failed")
		block, parent := blankBlocks(t, b.timestamp, blockID)
		err = tv.validateTransaction(block, parent, tx, true)
		assert.NoError(t, err, "validateTransaction() failed")
		if checkTimestamp {
			// Check invalid timestamp.
			block.Timestamp = 0
			parent.Timestamp = 0
			err = tv.validateTransaction(block, parent, tx, true)
			assert.Error(t, err, "validateTransaction() did not fail with invalid timestamp")
		}
	}
}

func setBalance(t *testing.T, to *testObjects, balanceKey []byte, balance uint64) {
	genesisSig, err := crypto.NewSignatureFromBase58(genesisSignature)
	assert.NoError(t, err, "NewSignatureFromBase58() failed")
	err = to.balances.setAccountBalance(balanceKey, balance, genesisSig)
	assert.NoError(t, err, "setAccountBalance() failed")
	flushBalances(t, to.balances)
	flushAssets(t, to.assets)
}

func createGenesis(t *testing.T, recipient string) *proto.Genesis {
	rcp, err := proto.NewAddressFromString(recipient)
	assert.NoError(t, err, "NewAddressFromString() failed")
	tx, err := proto.NewUnsignedGenesis(rcp, 100, genesisTimestamp)
	assert.NoError(t, err, "NewUnsignedGenesis() failed")
	return tx
}

func TestValidateGenesis(t *testing.T) {
	to, path := createTestObjects(t)

	defer func() {
		err := to.assets.db.Close()
		assert.NoError(t, err, "db.Close() failed")
		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createGenesis(t, recipientAddr)

	balanceDiffs := []balanceDiff{
		{recipientAddr, "", 0, tx.Amount},
	}
	setBalances(t, to, balanceDiffs)
	blocks := []block{{genesisTimestamp, genesisSignature}}
	validateTx(t, to.tv, tx, blocks, false)
	err := to.tv.performTransactions()
	assert.NoError(t, err, "performTransactions() failed")
	flushBalances(t, to.balances)
	flushAssets(t, to.assets)
	checkBalances(t, to.balances, balanceDiffs)
}

func createPayment(t *testing.T) *proto.Payment {
	spk, err := crypto.NewPublicKeyFromBase58(senderPK)
	assert.NoError(t, err, "NewPublicKeyFromBase58() failed")
	rcp, err := proto.NewAddressFromString(recipientAddr)
	assert.NoError(t, err, "NewAddressFromString() failed")
	tx, err := proto.NewUnsignedPayment(spk, rcp, 100, 1, timestamp1)
	assert.NoError(t, err, "NewUnsignedPayment() failed")
	return tx
}

func TestValidatePayment(t *testing.T) {
	to, path := createTestObjects(t)

	defer func() {
		err := to.assets.db.Close()
		assert.NoError(t, err, "db.Close() failed")
		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createPayment(t)
	balanceKey := key(t, senderAddr, "")

	// Set insufficient balance for sender and check failure.
	setBalance(t, to, balanceKey, tx.Amount)
	blocks := []block{{timestamp1, blockID0}}
	validateTx(t, to.tv, tx, blocks, true)
	err := to.tv.performTransactions()
	assert.Error(t, err, "performTransactions() did not fail with insufficient balance")
	to.reset()

	// Set insufficient balance for sender with multiple txs in same block.
	setBalance(t, to, balanceKey, tx.Amount*2)
	blocks = []block{{timestamp1, blockID0}, {timestamp1, blockID0}}
	validateTx(t, to.tv, tx, blocks, true)
	err = to.tv.performTransactions()
	assert.Error(t, err, "performTransactions() did not fail with insufficient balance")
	to.reset()

	// Set insufficient balance for sender with multiple txs in different blocks.
	setBalance(t, to, balanceKey, tx.Amount*2)
	blocks = []block{{timestamp1, blockID0}, {timestamp1, blockID1}}
	validateTx(t, to.tv, tx, blocks, true)
	err = to.tv.performTransactions()
	assert.Error(t, err, "performTransactions() did not fail with insufficient balance")
	to.reset()

	// Negative balance for one of txs in block with positive overall balance.
	setBalance(t, to, balanceKey, tx.Amount)
	blocks = []block{{timestamp0, genesisSignature}}
	// Negative balance after this Payment tx.
	validateTx(t, to.tv, tx, blocks, false)
	// This genesis tx 'fixes' negative balance.
	tx1 := createGenesis(t, senderAddr)
	validateTx(t, to.tv, tx1, blocks, false)
	err = to.tv.performTransactions()
	assert.Error(t, err, "performTransactions() did not fail with negative balance")
	to.reset()

	// Negative balance for one of txs in block with positive overall balance when this situation is allowed.
	setBalance(t, to, balanceKey, tx.Amount)
	blocks = []block{{timestamp1, genesisSignature}}
	// Negative balance after this Payment tx.
	validateTx(t, to.tv, tx, blocks, false)
	// This genesis tx 'fixes' negative balance.
	tx1 = createGenesis(t, senderAddr)
	validateTx(t, to.tv, tx1, blocks, false)
	err = to.tv.performTransactions()
	assert.NoError(t, err, "performTransactions() failed with negative balance but it was allowed for this block")
	to.reset()

	// Set proper balances and check result state.
	balanceDiffs := []balanceDiff{
		{senderAddr, "", tx.Amount + tx.Fee, 0},
		{recipientAddr, "", 0, tx.Amount},
		{minerAddr, "", 0, tx.Fee},
	}
	setBalances(t, to, balanceDiffs)
	blocks = []block{{timestamp0, blockID0}}
	validateTx(t, to.tv, tx, blocks, true)
	err = to.tv.performTransactions()
	assert.NoError(t, err, "performTransactions() failed")
	flushBalances(t, to.balances)
	flushAssets(t, to.assets)
	checkBalances(t, to.balances, balanceDiffs)
}

func createAsset(t *testing.T, to *testObjects, asset *proto.OptionalAsset) *assetInfo {
	blockID, err := crypto.NewSignatureFromBase58(blockID0)
	assert.NoError(t, err, "NewSignatureFromBase58() failed")
	assetInfo := createAssetInfo(t, true, blockID, asset.ID)
	err = to.assets.issueAsset(asset.ID, assetInfo)
	assert.NoError(t, err, "issueAset() failed")
	flushAssets(t, to.assets)
	return assetInfo
}

func createTransferV1(t *testing.T, to *testObjects, recipientAddr string) *proto.TransferV1 {
	asset, err := proto.NewOptionalAssetFromString(assetStr)
	assert.NoError(t, err, "NewOptionalAssetFromString() failed")
	createAsset(t, to, asset)
	spk, err := crypto.NewPublicKeyFromBase58(senderPK)
	assert.NoError(t, err, "NewPublicKeyFromBase58() failed")
	rcp, err := proto.NewAddressFromString(recipientAddr)
	assert.NoError(t, err, "NewAddressFromString() failed")
	tx, err := proto.NewUnsignedTransferV1(spk, *asset, *asset, timestamp1, 100, 1, rcp, "attachment")
	assert.NoError(t, err, "NewUnsignedTransferV1() failed")
	return tx
}

func TestValidateTransferV1(t *testing.T) {
	to, path := createTestObjects(t)

	defer func() {
		err := to.assets.db.Close()
		assert.NoError(t, err, "db.Close() failed")
		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createTransferV1(t, to, recipientAddr)
	balanceKey := key(t, senderAddr, assetStr)

	// Set insufficient balance for sender and check failure.
	setBalance(t, to, balanceKey, tx.Amount)
	blocks := []block{{timestamp1, blockID0}}
	validateTx(t, to.tv, tx, blocks, true)
	err := to.tv.performTransactions()
	assert.Error(t, err, "performTransactions() did not fail with insufficient balance")
	to.reset()

	// Set insufficient balance for sender with multiple txs in same block.
	setBalance(t, to, balanceKey, tx.Amount*2)
	blocks = []block{{timestamp1, blockID0}, {timestamp1, blockID0}}
	validateTx(t, to.tv, tx, blocks, true)
	err = to.tv.performTransactions()
	assert.Error(t, err, "performTransactions() did not fail with insufficient balance")
	to.reset()

	// Set insufficient balance for sender with multiple txs in different blocks.
	setBalance(t, to, balanceKey, tx.Amount*2)
	blocks = []block{{timestamp1, blockID0}, {timestamp1, blockID1}}
	validateTx(t, to.tv, tx, blocks, true)
	err = to.tv.performTransactions()
	assert.Error(t, err, "performTransactions() did not fail with insufficient balance")
	to.reset()

	tx1 := createTransferV1(t, to, senderAddr)
	// Negative balance for one of txs in block with positive overall balance.
	setBalance(t, to, balanceKey, tx1.Amount)
	blocks = []block{{timestamp0, blockID0}}
	// Transfer to same address leads to temp negative balance.
	validateTx(t, to.tv, tx1, blocks, false)
	err = to.tv.performTransactions()
	assert.Error(t, err, "performTransactions() did not fail with negative balance")
	to.reset()

	// Negative balance for one of txs in block with positive overall balance when this situation is allowed.
	setBalance(t, to, balanceKey, tx1.Amount)
	blocks = []block{{timestamp1, blockID0}}
	// Transfer to same address leads to temp negative balance.
	validateTx(t, to.tv, tx1, blocks, false)
	err = to.tv.performTransactions()
	assert.NoError(t, err, "performTransactions() failed with negative balance but it was allowed for this block")
	to.reset()

	// Set proper balances and check result state.
	balanceDiffs := []balanceDiff{
		{senderAddr, assetStr, tx.Amount + tx.Fee, 0},
		{recipientAddr, assetStr, 0, tx.Amount},
		{minerAddr, assetStr, 0, tx.Fee},
	}
	setBalances(t, to, balanceDiffs)
	blocks = []block{{timestamp0, blockID0}}
	validateTx(t, to.tv, tx, blocks, true)
	err = to.tv.performTransactions()
	assert.NoError(t, err, "performTransactions() failed")
	flushBalances(t, to.balances)
	flushAssets(t, to.assets)
	checkBalances(t, to.balances, balanceDiffs)
}

func createTransferV2(t *testing.T, to *testObjects, recipientAddr string) *proto.TransferV2 {
	asset, err := proto.NewOptionalAssetFromString(assetStr)
	assert.NoError(t, err, "NewOptionalAssetFromString() failed")
	createAsset(t, to, asset)
	spk, err := crypto.NewPublicKeyFromBase58(senderPK)
	assert.NoError(t, err, "NewPublicKeyFromBase58() failed")
	rcp, err := proto.NewAddressFromString(recipientAddr)
	assert.NoError(t, err, "NewAddressFromString() failed")
	tx, err := proto.NewUnsignedTransferV2(spk, *asset, *asset, timestamp1, 100, 1, rcp, "attachment")
	assert.NoError(t, err, "NewUnsignedTransferV2() failed")
	return tx
}

func TestValidateTransferV2(t *testing.T) {
	to, path := createTestObjects(t)

	defer func() {
		err := to.assets.db.Close()
		assert.NoError(t, err, "db.Close() failed")
		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createTransferV2(t, to, recipientAddr)
	balanceKey := key(t, senderAddr, assetStr)

	// Set insufficient balance for sender and check failure.
	setBalance(t, to, balanceKey, tx.Amount)
	blocks := []block{{timestamp1, blockID0}}
	validateTx(t, to.tv, tx, blocks, true)
	err := to.tv.performTransactions()
	assert.Error(t, err, "performTransactions() did not fail with insufficient balance")
	to.reset()

	// Set insufficient balance for sender with multiple txs in same block.
	setBalance(t, to, balanceKey, tx.Amount*2)
	blocks = []block{{timestamp1, blockID0}, {timestamp1, blockID0}}
	validateTx(t, to.tv, tx, blocks, true)
	err = to.tv.performTransactions()
	assert.Error(t, err, "performTransactions() did not fail with insufficient balance")
	to.reset()

	// Set insufficient balance for sender with multiple txs in different blocks.
	setBalance(t, to, balanceKey, tx.Amount*2)
	blocks = []block{{timestamp1, blockID0}, {timestamp1, blockID1}}
	validateTx(t, to.tv, tx, blocks, true)
	err = to.tv.performTransactions()
	assert.Error(t, err, "performTransactions() did not fail with insufficient balance")
	to.reset()

	tx1 := createTransferV2(t, to, senderAddr)
	// Negative balance for one of txs in block with positive overall balance.
	setBalance(t, to, balanceKey, tx1.Amount)
	blocks = []block{{timestamp0, blockID0}}
	// Transfer to same address leads to temp negative balance.
	validateTx(t, to.tv, tx1, blocks, true)
	err = to.tv.performTransactions()
	assert.Error(t, err, "performTransactions() did not fail with negative balance")
	to.reset()

	// Negative balance for one of txs in block with positive overall balance when this situation is allowed.
	setBalance(t, to, balanceKey, tx1.Amount)
	blocks = []block{{timestamp1, blockID0}}
	// Transfer to same address leads to temp negative balance.
	validateTx(t, to.tv, tx1, blocks, true)
	err = to.tv.performTransactions()
	assert.NoError(t, err, "performTransactions() failed with negative balance but it was allowed for this block")
	to.reset()

	// Set proper balances and check result state.
	balanceDiffs := []balanceDiff{
		{senderAddr, assetStr, tx.Amount + tx.Fee, 0},
		{recipientAddr, assetStr, 0, tx.Amount},
		{minerAddr, assetStr, 0, tx.Fee},
	}
	setBalances(t, to, balanceDiffs)
	blocks = []block{{timestamp0, blockID0}}
	validateTx(t, to.tv, tx, blocks, true)
	err = to.tv.performTransactions()
	assert.NoError(t, err, "performTransactions() failed")
	flushBalances(t, to.balances)
	flushAssets(t, to.assets)
	checkBalances(t, to.balances, balanceDiffs)
}

func createIssueV1(t *testing.T) *proto.IssueV1 {
	spk, err := crypto.NewPublicKeyFromBase58(senderPK)
	assert.NoError(t, err, "NewPublicKeyFromBase58() failed")
	tx, err := proto.NewUnsignedIssueV1(spk, "name", "description", 10, 7, true, timestamp1, 1)
	assert.NoError(t, err, "NewUnsignedIssueV1() failed")
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, _ := crypto.GenerateKeyPair(seed)
	err = tx.Sign(sk)
	assert.NoError(t, err, "Sign() failed")
	return tx
}

func TestValidateIssueV1(t *testing.T) {
	to, path := createTestObjects(t)

	defer func() {
		err := to.assets.db.Close()
		assert.NoError(t, err, "db.Close() failed")
		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createIssueV1(t)

	blockID, err := crypto.NewSignatureFromBase58(blockID0)
	assert.NoError(t, err, "NewSignatureFromBase58() failed")
	assetInfo := assetInfo{
		assetConstInfo: assetConstInfo{
			name:        tx.Name,
			description: tx.Description,
			decimals:    int8(tx.Decimals),
		},
		assetHistoryRecord: assetHistoryRecord{
			quantity:   tx.Quantity,
			reissuable: tx.Reissuable,
			blockID:    blockID,
		},
	}

	asset, err := proto.NewOptionalAssetFromDigest(*tx.ID)
	assert.NoError(t, err, "NewOptionalAssetFromString() failed")
	// Set proper balances and check result state.
	balanceDiffs := []balanceDiff{
		{senderAddr, asset.String(), 0, tx.Quantity},
		{senderAddr, "", tx.Fee, 0},
		{minerAddr, "", 0, tx.Fee},
	}
	setBalances(t, to, balanceDiffs)
	blocks := []block{{timestamp0, blockID0}}
	validateTx(t, to.tv, tx, blocks, true)
	err = to.tv.performTransactions()
	assert.NoError(t, err, "performTransactions() failed")
	flushBalances(t, to.balances)
	flushAssets(t, to.assets)
	checkBalances(t, to.balances, balanceDiffs)

	// Check asset info.
	info, err := to.assets.assetInfo(asset.ID)
	assert.NoError(t, err, "assetInfo() failed")
	assert.Equal(t, assetInfo, *info, "invalid asset info after performing IssueV1 transaction")
}

func createIssueV2(t *testing.T) *proto.IssueV2 {
	spk, err := crypto.NewPublicKeyFromBase58(senderPK)
	assert.NoError(t, err, "NewPublicKeyFromBase58() failed")
	tx, err := proto.NewUnsignedIssueV2('W', spk, "name", "description", 10, 7, true, []byte{}, timestamp1, 1)
	assert.NoError(t, err, "NewUnsignedIssueV2() failed")
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, _ := crypto.GenerateKeyPair(seed)
	err = tx.Sign(sk)
	assert.NoError(t, err, "Sign() failed")
	return tx
}

func TestValidateIssueV2(t *testing.T) {
	to, path := createTestObjects(t)

	defer func() {
		err := to.assets.db.Close()
		assert.NoError(t, err, "db.Close() failed")
		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createIssueV2(t)

	blockID, err := crypto.NewSignatureFromBase58(blockID0)
	assert.NoError(t, err, "NewSignatureFromBase58() failed")
	assetInfo := assetInfo{
		assetConstInfo: assetConstInfo{
			name:        tx.Name,
			description: tx.Description,
			decimals:    int8(tx.Decimals),
		},
		assetHistoryRecord: assetHistoryRecord{
			quantity:   tx.Quantity,
			reissuable: tx.Reissuable,
			blockID:    blockID,
		},
	}

	asset, err := proto.NewOptionalAssetFromDigest(*tx.ID)
	assert.NoError(t, err, "NewOptionalAssetFromString() failed")
	// Set proper balances and check result state.
	balanceDiffs := []balanceDiff{
		{senderAddr, asset.String(), 0, tx.Quantity},
		{senderAddr, "", tx.Fee, 0},
		{minerAddr, "", 0, tx.Fee},
	}
	setBalances(t, to, balanceDiffs)
	blocks := []block{{timestamp0, blockID0}}
	validateTx(t, to.tv, tx, blocks, true)
	err = to.tv.performTransactions()
	assert.NoError(t, err, "performTransactions() failed")
	flushBalances(t, to.balances)
	flushAssets(t, to.assets)
	checkBalances(t, to.balances, balanceDiffs)

	// Check asset info.
	info, err := to.assets.assetInfo(asset.ID)
	assert.NoError(t, err, "assetInfo() failed")
	assert.Equal(t, assetInfo, *info, "invalid asset info after performing IssueV2 transaction")
}

func createReissueV1(t *testing.T, assetID crypto.Digest) *proto.ReissueV1 {
	spk, err := crypto.NewPublicKeyFromBase58(senderPK)
	assert.NoError(t, err, "NewPublicKeyFromBase58() failed")
	tx, err := proto.NewUnsignedReissueV1(spk, assetID, 1, false, timestamp1, 1)
	assert.NoError(t, err, "NewUnsignedReissueV1() failed")
	return tx
}

func TestValidateReissueV1(t *testing.T) {
	to, path := createTestObjects(t)

	defer func() {
		err := to.assets.db.Close()
		assert.NoError(t, err, "db.Close() failed")
		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	// Create asset.
	asset, err := proto.NewOptionalAssetFromString(assetStr)
	assert.NoError(t, err, "NewOptionalAssetFromString() failed")
	assetInfo := createAsset(t, to, asset)
	tx := createReissueV1(t, asset.ID)
	// Reissue asset.
	assetInfo.reissuable = tx.Reissuable
	assetInfo.quantity = assetInfo.quantity + tx.Quantity

	// Set proper balances and check result state.
	balanceDiffs := []balanceDiff{
		{senderAddr, asset.String(), 0, tx.Quantity},
		{senderAddr, "", tx.Fee, 0},
		{minerAddr, "", 0, tx.Fee},
	}
	setBalances(t, to, balanceDiffs)
	blocks := []block{{timestamp0, blockID0}}
	validateTx(t, to.tv, tx, blocks, true)
	err = to.tv.performTransactions()
	assert.NoError(t, err, "performTransactions() failed")
	flushBalances(t, to.balances)
	flushAssets(t, to.assets)
	checkBalances(t, to.balances, balanceDiffs)

	// Check asset info.
	info, err := to.assets.assetInfo(asset.ID)
	assert.NoError(t, err, "assetInfo() failed")
	assert.Equal(t, *assetInfo, *info, "invalid asset info after performing ReissueV1 transaction")
}

func createReissueV2(t *testing.T, assetID crypto.Digest) *proto.ReissueV2 {
	spk, err := crypto.NewPublicKeyFromBase58(senderPK)
	assert.NoError(t, err, "NewPublicKeyFromBase58() failed")
	tx, err := proto.NewUnsignedReissueV2('W', spk, assetID, 1, false, timestamp1, 1)
	assert.NoError(t, err, "NewUnsignedReissueV2() failed")
	return tx
}

func TestValidateReissueV2(t *testing.T) {
	to, path := createTestObjects(t)

	defer func() {
		err := to.assets.db.Close()
		assert.NoError(t, err, "db.Close() failed")
		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	// Create asset.
	asset, err := proto.NewOptionalAssetFromString(assetStr)
	assert.NoError(t, err, "NewOptionalAssetFromString() failed")
	assetInfo := createAsset(t, to, asset)
	tx := createReissueV2(t, asset.ID)
	// Reissue asset.
	assetInfo.reissuable = tx.Reissuable
	assetInfo.quantity = assetInfo.quantity + tx.Quantity

	// Set proper balances and check result state.
	balanceDiffs := []balanceDiff{
		{senderAddr, asset.String(), 0, tx.Quantity},
		{senderAddr, "", tx.Fee, 0},
		{minerAddr, "", 0, tx.Fee},
	}
	setBalances(t, to, balanceDiffs)
	blocks := []block{{timestamp0, blockID0}}
	validateTx(t, to.tv, tx, blocks, true)
	err = to.tv.performTransactions()
	assert.NoError(t, err, "performTransactions() failed")
	flushBalances(t, to.balances)
	flushAssets(t, to.assets)
	checkBalances(t, to.balances, balanceDiffs)

	// Check asset info.
	info, err := to.assets.assetInfo(asset.ID)
	assert.NoError(t, err, "assetInfo() failed")
	assert.Equal(t, *assetInfo, *info, "invalid asset info after performing ReissueV2 transaction")
}

func createBurnV1(t *testing.T, assetID crypto.Digest) *proto.BurnV1 {
	spk, err := crypto.NewPublicKeyFromBase58(senderPK)
	assert.NoError(t, err, "NewPublicKeyFromBase58() failed")
	tx, err := proto.NewUnsignedBurnV1(spk, assetID, 1, timestamp1, 1)
	assert.NoError(t, err, "NewUnsignedBurnV1() failed")
	return tx
}

func TestValidateBurnV1(t *testing.T) {
	to, path := createTestObjects(t)

	defer func() {
		err := to.assets.db.Close()
		assert.NoError(t, err, "db.Close() failed")
		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	// Create asset.
	asset, err := proto.NewOptionalAssetFromString(assetStr)
	assert.NoError(t, err, "NewOptionalAssetFromString() failed")
	assetInfo := createAsset(t, to, asset)
	tx := createBurnV1(t, asset.ID)
	// Burn asset.
	assetInfo.quantity = assetInfo.quantity - tx.Amount

	balanceKey := key(t, senderAddr, assetStr)

	// Set insufficient balance for sender and check failure.
	setBalance(t, to, balanceKey, tx.Amount-1)
	blocks := []block{{timestamp1, blockID0}}
	validateTx(t, to.tv, tx, blocks, true)
	err = to.tv.performTransactions()
	assert.Error(t, err, "performTransactions() did not fail with insufficient balance")
	to.reset()

	// Set insufficient balance for sender with multiple txs in same block.
	setBalance(t, to, balanceKey, tx.Amount*2-1)
	blocks = []block{{timestamp1, blockID0}, {timestamp1, blockID0}}
	validateTx(t, to.tv, tx, blocks, true)
	err = to.tv.performTransactions()
	assert.Error(t, err, "performTransactions() did not fail with insufficient balance")
	to.reset()

	// Set insufficient balance for sender with multiple txs in different blocks.
	setBalance(t, to, balanceKey, tx.Amount*2-1)
	blocks = []block{{timestamp1, blockID0}, {timestamp1, blockID1}}
	validateTx(t, to.tv, tx, blocks, true)
	err = to.tv.performTransactions()
	assert.Error(t, err, "performTransactions() did not fail with insufficient balance")
	to.reset()

	// Set proper balances and check result state.
	balanceDiffs := []balanceDiff{
		{senderAddr, asset.String(), tx.Amount, 0},
		{senderAddr, "", tx.Fee, 0},
		{minerAddr, "", 0, tx.Fee},
	}
	setBalances(t, to, balanceDiffs)
	blocks = []block{{timestamp0, blockID0}}
	validateTx(t, to.tv, tx, blocks, true)
	err = to.tv.performTransactions()
	assert.NoError(t, err, "performTransactions() failed")
	flushBalances(t, to.balances)
	flushAssets(t, to.assets)
	checkBalances(t, to.balances, balanceDiffs)

	// Check asset info.
	info, err := to.assets.assetInfo(asset.ID)
	assert.NoError(t, err, "assetInfo() failed")
	assert.Equal(t, *assetInfo, *info, "invalid asset info after performing BurnV1 transaction")
}

func createBurnV2(t *testing.T, assetID crypto.Digest) *proto.BurnV2 {
	spk, err := crypto.NewPublicKeyFromBase58(senderPK)
	assert.NoError(t, err, "NewPublicKeyFromBase58() failed")
	tx, err := proto.NewUnsignedBurnV2('W', spk, assetID, 1, timestamp1, 1)
	assert.NoError(t, err, "NewUnsignedBurnV2() failed")
	return tx
}

func TestValidateBurnV2(t *testing.T) {
	to, path := createTestObjects(t)

	defer func() {
		err := to.assets.db.Close()
		assert.NoError(t, err, "db.Close() failed")
		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	// Create asset.
	asset, err := proto.NewOptionalAssetFromString(assetStr)
	assert.NoError(t, err, "NewOptionalAssetFromString() failed")
	assetInfo := createAsset(t, to, asset)
	tx := createBurnV2(t, asset.ID)
	// Burn asset.
	assetInfo.quantity = assetInfo.quantity - tx.Amount

	balanceKey := key(t, senderAddr, assetStr)

	// Set insufficient balance for sender and check failure.
	setBalance(t, to, balanceKey, tx.Amount-1)
	blocks := []block{{timestamp1, blockID0}}
	validateTx(t, to.tv, tx, blocks, true)
	err = to.tv.performTransactions()
	assert.Error(t, err, "performTransactions() did not fail with insufficient balance")
	to.reset()

	// Set insufficient balance for sender with multiple txs in same block.
	setBalance(t, to, balanceKey, tx.Amount*2-1)
	blocks = []block{{timestamp1, blockID0}, {timestamp1, blockID0}}
	validateTx(t, to.tv, tx, blocks, true)
	err = to.tv.performTransactions()
	assert.Error(t, err, "performTransactions() did not fail with insufficient balance")
	to.reset()

	// Set insufficient balance for sender with multiple txs in different blocks.
	setBalance(t, to, balanceKey, tx.Amount*2-1)
	blocks = []block{{timestamp1, blockID0}, {timestamp1, blockID1}}
	validateTx(t, to.tv, tx, blocks, true)
	err = to.tv.performTransactions()
	assert.Error(t, err, "performTransactions() did not fail with insufficient balance")
	to.reset()

	// Set proper balances and check result state.
	balanceDiffs := []balanceDiff{
		{senderAddr, asset.String(), tx.Amount, 0},
		{senderAddr, "", tx.Fee, 0},
		{minerAddr, "", 0, tx.Fee},
	}
	setBalances(t, to, balanceDiffs)
	blocks = []block{{timestamp0, blockID0}}
	validateTx(t, to.tv, tx, blocks, true)
	err = to.tv.performTransactions()
	assert.NoError(t, err, "performTransactions() failed")
	flushBalances(t, to.balances)
	flushAssets(t, to.assets)
	checkBalances(t, to.balances, balanceDiffs)

	// Check asset info.
	info, err := to.assets.assetInfo(asset.ID)
	assert.NoError(t, err, "assetInfo() failed")
	assert.Equal(t, *assetInfo, *info, "invalid asset info after performing BurnV2 transaction")
}

func createExchangeV1(t *testing.T) *proto.ExchangeV1 {
	buySender, _ := crypto.NewPublicKeyFromBase58(recipientPK)
	sellSender, _ := crypto.NewPublicKeyFromBase58(senderPK)
	mpk, _ := crypto.NewPublicKeyFromBase58(matcherPK)
	a, _ := proto.NewOptionalAssetFromString(assetStr)
	pa, _ := proto.NewOptionalAssetFromString("")
	sig, _ := crypto.NewSignatureFromBase58("5pzyUowLi31yP4AEh5qzg7gRrvmsfeypiUkW84CKzc4H6UTzEF2RgGPLckBEqNbJGn5ofQXzuDmUnxwuP3utYp9L")
	bo, _ := proto.NewUnsignedOrderV1(buySender, mpk, *a, *pa, proto.Buy, 10e8, 100, 0, 0, 3)
	bo.Signature = &sig
	so, _ := proto.NewUnsignedOrderV1(sellSender, mpk, *a, *pa, proto.Sell, 10e8, 100, 0, 0, 3)
	so.Signature = &sig
	tx, err := proto.NewUnsignedExchangeV1(*bo, *so, bo.Price, bo.Amount, 1, 2, 1, timestamp1)
	assert.NoError(t, err, "NewUnsignedExchangeV1() failed")
	return tx
}

func TestValidateExchangeV1(t *testing.T) {
	to, path := createTestObjects(t)

	defer func() {
		err := to.assets.db.Close()
		assert.NoError(t, err, "db.Close() failed")
		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	// Create assets.
	asset, err := proto.NewOptionalAssetFromString(assetStr)
	assert.NoError(t, err, "NewOptionalAssetFromString() failed")
	createAsset(t, to, asset)
	tx := createExchangeV1(t)

	price := tx.Price * tx.Amount / priceConstant
	// Set proper balances and check result state.
	balanceDiffs := []balanceDiff{
		{recipientAddr, assetStr, 0, tx.Amount},
		{recipientAddr, "", price + tx.BuyMatcherFee, 0},
		{senderAddr, assetStr, tx.Amount, 0},
		{senderAddr, "", tx.SellMatcherFee, price},
		{minerAddr, "", 0, tx.Fee},
		{matcherAddr, "", tx.Fee, tx.SellMatcherFee + tx.BuyMatcherFee},
	}
	setBalances(t, to, balanceDiffs)
	blocks := []block{{timestamp0, blockID0}}
	validateTx(t, to.tv, tx, blocks, true)
	err = to.tv.performTransactions()
	assert.NoError(t, err, "performTransactions() failed")
	flushBalances(t, to.balances)
	flushAssets(t, to.assets)
	checkBalances(t, to.balances, balanceDiffs)
}

func createExchangeV2(t *testing.T) *proto.ExchangeV2 {
	buySender, _ := crypto.NewPublicKeyFromBase58(recipientPK)
	sellSender, _ := crypto.NewPublicKeyFromBase58(senderPK)
	mpk, _ := crypto.NewPublicKeyFromBase58(matcherPK)
	a, _ := proto.NewOptionalAssetFromString(assetStr)
	pa, _ := proto.NewOptionalAssetFromString("")
	sig, _ := crypto.NewSignatureFromBase58("5pzyUowLi31yP4AEh5qzg7gRrvmsfeypiUkW84CKzc4H6UTzEF2RgGPLckBEqNbJGn5ofQXzuDmUnxwuP3utYp9L")
	bo, _ := proto.NewUnsignedOrderV1(buySender, mpk, *a, *pa, proto.Buy, 10e8, 100, 0, 0, 3)
	bo.Signature = &sig
	so, _ := proto.NewUnsignedOrderV1(sellSender, mpk, *a, *pa, proto.Sell, 10e8, 100, 0, 0, 3)
	so.Signature = &sig
	tx, err := proto.NewUnsignedExchangeV2(*bo, *so, bo.Price, bo.Amount, 1, 2, 1, timestamp1)
	assert.NoError(t, err, "NewUnsignedExchangeV2() failed")
	return tx
}

func TestValidateExchangeV2(t *testing.T) {
	to, path := createTestObjects(t)

	defer func() {
		err := to.assets.db.Close()
		assert.NoError(t, err, "db.Close() failed")
		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	// Create assets.
	asset, err := proto.NewOptionalAssetFromString(assetStr)
	assert.NoError(t, err, "NewOptionalAssetFromString() failed")
	createAsset(t, to, asset)
	tx := createExchangeV2(t)

	price := tx.Price * tx.Amount / priceConstant
	// Set proper balances and check result state.
	balanceDiffs := []balanceDiff{
		{recipientAddr, assetStr, 0, tx.Amount},
		{recipientAddr, "", price + tx.BuyMatcherFee, 0},
		{senderAddr, assetStr, tx.Amount, 0},
		{senderAddr, "", tx.SellMatcherFee, price},
		{minerAddr, "", 0, tx.Fee},
		{matcherAddr, "", tx.Fee, tx.SellMatcherFee + tx.BuyMatcherFee},
	}
	setBalances(t, to, balanceDiffs)
	blocks := []block{{timestamp0, blockID0}}
	validateTx(t, to.tv, tx, blocks, true)
	err = to.tv.performTransactions()
	assert.NoError(t, err, "performTransactions() failed")
	flushBalances(t, to.balances)
	flushAssets(t, to.assets)
	checkBalances(t, to.balances, balanceDiffs)
}
