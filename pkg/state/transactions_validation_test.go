package state

import (
	"math/big"
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
	timestamp2       = settings.MainNetSettings.AllowLeasedBalanceTransferUntilTime

	genesisSignature = "FSH8eAAzZNqnG8xgTZtz5xuLqXySsXgAjmFEC25hXMbEufiGjqWPnGCZFt6gLiVLJny16ipxRNAkkzjjhqTjBE2"
	blockSig0        = "FSH8eAAzZNqnG8xgTZtz5xuLqXySsXgAjmFEC25hXMbEufiGjqWPnGCZFt6gLiVLJny16ipxRNAkkzjjhqTjBE1"
	blockSig1        = "FSH8eAAzZNqnG8xgTZtz5xuLqXySsXgAjmFEC25hXMbEufiGjqWPnGCZFt6gLiVLJny16ipxRNAkkzjjhqTjBE3"

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
	leases   *leases
	balances *balances
	tv       *transactionValidator
}

func createTestObjects(t *testing.T) (*testObjects, []string) {
	assets, path, err := createAssets()
	assert.NoError(t, err, "createAssets() failed")
	leases, path1, err := createLeases()
	assert.NoError(t, err, "createLeases() failed")
	balances, err := newBalances(assets.db, assets.dbBatch, &mock{}, &mockBlockInfo{})
	assert.NoError(t, err, "newBalances() failed")
	genesisSig, err := crypto.NewSignatureFromBase58(genesisSignature)
	assert.NoError(t, err, "NewSignatureFromBase58() failed")
	tv, err := newTransactionValidator(genesisSig, balances, assets, leases, settings.MainNetSettings)
	assert.NoError(t, err, "newTransactionValidator() failed")
	return &testObjects{assets: assets, leases: leases, balances: balances, tv: tv}, append(path, path1...)
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

type profileChange struct {
	address      string
	asset        string
	prevBalance  uint64
	prevLeaseIn  int64
	prevLeaseOut int64
	newBalance   uint64
	newLeaseIn   int64
	newLeaseOut  int64
}

func setBalances(t *testing.T, to *testObjects, profileChanges []profileChange) {
	for _, diff := range profileChanges {
		addr, err := proto.NewAddressFromString(diff.address)
		assert.NoError(t, err, "NewAddressFromString() failed")
		if diff.asset == "" {
			r := &wavesBalanceRecord{balanceProfile{diff.prevBalance, diff.prevLeaseIn, diff.prevLeaseOut}, crypto.Signature{}}
			err := to.balances.setWavesBalance(addr, r)
			assert.NoError(t, err, "setWavesBalance() failed")
		} else {
			ast, err := proto.NewOptionalAssetFromString(diff.asset)
			assert.NoError(t, err, "NewOptionalAssetFromString() failed")
			r := &assetBalanceRecord{diff.prevBalance, crypto.Signature{}}
			err = to.balances.setAssetBalance(addr, ast.ToID(), r)
			assert.NoError(t, err, "setAssetBalance() failed")
		}
	}
	flushBalances(t, to.balances)
	flushAssets(t, to.assets)
	flushLeases(t, to.leases)
}

func checkBalances(t *testing.T, balances *balances, profileChanges []profileChange) {
	for _, diff := range profileChanges {
		addr, err := proto.NewAddressFromString(diff.address)
		assert.NoError(t, err, "NewAddressFromString() failed")
		if diff.asset == "" {
			newProfile := balanceProfile{diff.newBalance, diff.newLeaseIn, diff.newLeaseOut}
			profile, err := balances.wavesBalance(addr)
			assert.NoError(t, err, "wavesBalance() failed")
			assert.Equalf(t, newProfile, *profile, "invalid waves balance profile after validation: must be %v, is %v", newProfile, profile)
		} else {
			ast, err := proto.NewOptionalAssetFromString(diff.asset)
			assert.NoError(t, err, "NewOptionalAssetFromString() failed")
			balance, err := balances.assetBalance(addr, ast.ToID())
			assert.NoError(t, err, "assetBalance() failed")
			assert.Equalf(t, balance, diff.newBalance, "invalid asset balance after validation: must be %d, is: %d", diff.newBalance, balance)
		}
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

func setBalance(t *testing.T, to *testObjects, address, asset string, profile *balanceProfile) {
	genesisSig, err := crypto.NewSignatureFromBase58(genesisSignature)
	assert.NoError(t, err, "NewSignatureFromBase58() failed")
	addr, err := proto.NewAddressFromString(address)
	assert.NoError(t, err, "NewAddressFromString() failed")
	if asset == "" {
		r := &wavesBalanceRecord{*profile, genesisSig}
		err := to.balances.setWavesBalance(addr, r)
		assert.NoError(t, err, "setWavesBalance() failed")
	} else {
		ast, err := proto.NewOptionalAssetFromString(asset)
		assert.NoError(t, err, "NewOptionalAssetFromString() failed")
		r := &assetBalanceRecord{profile.balance, genesisSig}
		err = to.balances.setAssetBalance(addr, ast.ToID(), r)
		assert.NoError(t, err, "setAssetBalance() failed")
	}
	flushBalances(t, to.balances)
	flushAssets(t, to.assets)
	flushLeases(t, to.leases)
}

func validateAndCheck(t *testing.T, to *testObjects, tx proto.Transaction, profileChanges []profileChange) {
	blocks := []block{{timestamp0, blockSig0}}
	validateTx(t, to.tv, tx, blocks, true)
	err := to.tv.performTransactions()
	assert.NoError(t, err, "performTransactions() failed")
	flushBalances(t, to.balances)
	flushAssets(t, to.assets)
	flushLeases(t, to.leases)
	checkBalances(t, to.balances, profileChanges)
}

func diffTest(t *testing.T, to *testObjects, tx proto.Transaction, profileChanges []profileChange) {
	setBalances(t, to, profileChanges)
	validateAndCheck(t, to, tx, profileChanges)
}

func createGenesis(t *testing.T, recipient string) *proto.Genesis {
	rcp, err := proto.NewAddressFromString(recipient)
	assert.NoError(t, err, "NewAddressFromString() failed")
	return proto.NewUnsignedGenesis(rcp, 100, genesisTimestamp)
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
	profileChanges := []profileChange{
		{address: recipientAddr, asset: "", prevBalance: 0, newBalance: tx.Amount},
	}
	setBalances(t, to, profileChanges)
	blocks := []block{{genesisTimestamp, genesisSignature}}
	validateTx(t, to.tv, tx, blocks, false)
	err := to.tv.performTransactions()
	assert.NoError(t, err, "performTransactions() failed")
	flushBalances(t, to.balances)
	flushAssets(t, to.assets)
	flushLeases(t, to.leases)
	checkBalances(t, to.balances, profileChanges)
}

func createPayment(t *testing.T) *proto.Payment {
	spk, err := crypto.NewPublicKeyFromBase58(senderPK)
	assert.NoError(t, err, "NewPublicKeyFromBase58() failed")
	rcp, err := proto.NewAddressFromString(recipientAddr)
	assert.NoError(t, err, "NewAddressFromString() failed")
	return proto.NewUnsignedPayment(spk, rcp, 100, 1, timestamp1)
}

type leaseTestCase struct {
	addr     string
	profile  balanceProfile
	blocks   []block
	resError bool
	failMsg  string
}

type runLeaseTest = func(*testing.T, *testObjects, proto.Transaction, *leaseTestCase)

var testInvalidLeasing runLeaseTest = func(t *testing.T, to *testObjects, tx proto.Transaction, c *leaseTestCase) {
	setBalance(t, to, c.addr, "", &c.profile)
	validateTx(t, to.tv, tx, c.blocks, true)
	err := to.tv.performTransactions()
	if c.resError {
		assert.Error(t, err, c.failMsg)
	} else {
		assert.NoError(t, err, c.failMsg)
	}
	to.reset()
}

type txTestCase struct {
	senderAddr string
	assetStr   string
	amount     uint64
	blocks     []block
	resError   bool
	failMsg    string
}

type runTest func(*testing.T, *testObjects, proto.Transaction, *txTestCase)

var testNegBalance runTest = func(t *testing.T, to *testObjects, tx proto.Transaction, c *txTestCase) {
	profile := &balanceProfile{c.amount, 0, 0}
	setBalance(t, to, c.senderAddr, c.assetStr, profile)
	validateTx(t, to.tv, tx, c.blocks, true)
	err := to.tv.performTransactions()
	if c.resError {
		assert.Error(t, err, c.failMsg)
	} else {
		assert.NoError(t, err, c.failMsg)
	}
	to.reset()
}

var testTempNegative runTest = func(t *testing.T, to *testObjects, tx proto.Transaction, c *txTestCase) {
	profile := &balanceProfile{c.amount, 0, 0}
	setBalance(t, to, c.senderAddr, c.assetStr, profile)
	// Negative balance after this Payment tx.
	validateTx(t, to.tv, tx, c.blocks, false)
	// This genesis tx 'fixes' negative balance.
	tx1 := createGenesis(t, c.senderAddr)
	validateTx(t, to.tv, tx1, c.blocks, false)
	err := to.tv.performTransactions()
	if c.resError {
		assert.Error(t, err, c.failMsg)
	} else {
		assert.NoError(t, err, c.failMsg)
	}
	to.reset()
}

var testTempNegativeUniversal runTest = func(t *testing.T, to *testObjects, tx proto.Transaction, c *txTestCase) {
	tx1 := createTransferV1(t, to, c.senderAddr)
	// Negative balance for one of txs in block with positive overall balance.
	profile := &balanceProfile{c.amount, 0, 0}
	setBalance(t, to, c.senderAddr, c.assetStr, profile)
	// Transfer to same address leads to temp negative balance.
	validateTx(t, to.tv, tx1, c.blocks, false)
	err := to.tv.performTransactions()
	if c.resError {
		assert.Error(t, err, c.failMsg)
	} else {
		assert.NoError(t, err, c.failMsg)
	}
	to.reset()
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
	tests := []struct {
		txTestCase
		f runTest
	}{
		// Set insufficient balance for sender and check failure.
		{txTestCase{senderAddr, "", tx.Amount, []block{{timestamp1, blockSig0}}, true, "performTransactions() did not fail with insufficient balance"}, testNegBalance},
		// Set insufficient balance for sender with multiple txs in same block.
		{txTestCase{senderAddr, "", tx.Amount * 2, []block{{timestamp1, blockSig0}, {timestamp1, blockSig0}}, true, "performTransactions() did not fail with insufficient balance"}, testNegBalance},
		// Set insufficient balance for sender with multiple txs in different blocks.
		{txTestCase{senderAddr, "", tx.Amount * 2, []block{{timestamp1, blockSig0}, {timestamp1, blockSig1}}, true, "performTransactions() did not fail with insufficient balance"}, testNegBalance},
		// Negative balance for one of txs in block with positive overall balance.
		{txTestCase{senderAddr, "", tx.Amount, []block{{timestamp0, genesisSignature}}, true, "performTransactions() did not fail with negative balance"}, testTempNegative},
		// Negative balance for one of txs in block with positive overall balance when this situation is allowed.
		{txTestCase{senderAddr, "", tx.Amount, []block{{timestamp1, genesisSignature}}, false, "performTransactions() failed with negative balance but it was allowed for this block"}, testTempNegative},
	}
	for _, c := range tests {
		c.f(t, to, tx, &c.txTestCase)
	}

	// Set proper balances and check result state.
	profileChanges := []profileChange{
		{address: senderAddr, asset: "", prevBalance: tx.Amount + tx.Fee, newBalance: 0},
		{address: recipientAddr, asset: "", prevBalance: 0, newBalance: tx.Amount},
		{address: minerAddr, asset: "", prevBalance: 0, newBalance: tx.Fee},
	}
	diffTest(t, to, tx, profileChanges)
}

func createAsset(t *testing.T, to *testObjects, asset *proto.OptionalAsset) *assetInfo {
	blockID, err := crypto.NewSignatureFromBase58(blockSig0)
	assert.NoError(t, err, "NewSignatureFromBase58() failed")
	assetInfo := createAssetInfo(t, true, blockID, asset.ID)
	err = to.assets.issueAsset(asset.ID, assetInfo)
	assert.NoError(t, err, "issueAset() failed")
	flushAssets(t, to.assets)
	flushLeases(t, to.leases)
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
	return proto.NewUnsignedTransferV1(spk, *asset, *asset, timestamp1, 100, 1, proto.NewRecipientFromAddress(rcp), "attachment")
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
	tests := []struct {
		txTestCase
		f runTest
	}{
		// Set insufficient balance for sender and check failure.
		{txTestCase{senderAddr, assetStr, tx.Amount, []block{{timestamp1, blockSig0}}, true, "performTransactions() did not fail with insufficient balance"}, testNegBalance},
		// Set insufficient balance for sender with multiple txs in same block.
		{txTestCase{senderAddr, assetStr, tx.Amount, []block{{timestamp1, blockSig0}, {timestamp1, blockSig0}}, true, "performTransactions() did not fail with insufficient balance"}, testNegBalance},
		// Set insufficient balance for sender with multiple txs in different blocks.
		{txTestCase{senderAddr, assetStr, tx.Amount, []block{{timestamp1, blockSig0}, {timestamp1, blockSig1}}, true, "performTransactions() did not fail with insufficient balance"}, testNegBalance},
		// Negative balance for one of txs in block with positive overall balance.
		{txTestCase{senderAddr, assetStr, tx.Amount, []block{{timestamp0, blockSig0}}, true, "performTransactions() did not fail with negative balance"}, testTempNegativeUniversal},
		// Negative balance for one of txs in block with positive overall balance when this situation is allowed.
		{txTestCase{senderAddr, assetStr, tx.Amount, []block{{timestamp1, blockSig0}}, false, "performTransactions() failed with negative balance but it was allowed for this block"}, testTempNegativeUniversal},
	}
	for _, c := range tests {
		c.f(t, to, tx, &c.txTestCase)
	}

	// Set proper balances and check result state.
	profileChanges := []profileChange{
		{address: senderAddr, asset: assetStr, prevBalance: tx.Amount + tx.Fee, newBalance: 0},
		{address: minerAddr, asset: assetStr, prevBalance: 0, newBalance: tx.Fee},
		{address: recipientAddr, asset: assetStr, prevBalance: 0, newBalance: tx.Amount},
	}
	diffTest(t, to, tx, profileChanges)
}

func createTransferV2(t *testing.T, to *testObjects, recipientAddr string) *proto.TransferV2 {
	asset, err := proto.NewOptionalAssetFromString(assetStr)
	assert.NoError(t, err, "NewOptionalAssetFromString() failed")
	createAsset(t, to, asset)
	spk, err := crypto.NewPublicKeyFromBase58(senderPK)
	assert.NoError(t, err, "NewPublicKeyFromBase58() failed")
	rcp, err := proto.NewAddressFromString(recipientAddr)
	assert.NoError(t, err, "NewAddressFromString() failed")
	return proto.NewUnsignedTransferV2(spk, *asset, *asset, timestamp1, 100, 1, proto.NewRecipientFromAddress(rcp), "attachment")
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
	tests := []struct {
		txTestCase
		f runTest
	}{
		// Set insufficient balance for sender and check failure.
		{txTestCase{senderAddr, assetStr, tx.Amount, []block{{timestamp1, blockSig0}}, true, "performTransactions() did not fail with insufficient balance"}, testNegBalance},
		// Set insufficient balance for sender with multiple txs in same block.
		{txTestCase{senderAddr, assetStr, tx.Amount, []block{{timestamp1, blockSig0}, {timestamp1, blockSig0}}, true, "performTransactions() did not fail with insufficient balance"}, testNegBalance},
		// Set insufficient balance for sender with multiple txs in different blocks.
		{txTestCase{senderAddr, assetStr, tx.Amount, []block{{timestamp1, blockSig0}, {timestamp1, blockSig1}}, true, "performTransactions() did not fail with insufficient balance"}, testNegBalance},
		// Negative balance for one of txs in block with positive overall balance.
		{txTestCase{senderAddr, assetStr, tx.Amount, []block{{timestamp0, blockSig0}}, true, "performTransactions() did not fail with negative balance"}, testTempNegativeUniversal},
		// Negative balance for one of txs in block with positive overall balance when this situation is allowed.
		{txTestCase{senderAddr, assetStr, tx.Amount, []block{{timestamp1, blockSig0}}, false, "performTransactions() failed with negative balance but it was allowed for this block"}, testTempNegativeUniversal},
	}
	for _, c := range tests {
		c.f(t, to, tx, &c.txTestCase)
	}

	// Set proper balances and check result state.
	profileChanges := []profileChange{
		{address: senderAddr, asset: assetStr, prevBalance: tx.Amount + tx.Fee, newBalance: 0},
		{address: minerAddr, asset: assetStr, prevBalance: 0, newBalance: tx.Fee},
		{address: recipientAddr, asset: assetStr, prevBalance: 0, newBalance: tx.Amount},
	}
	diffTest(t, to, tx, profileChanges)
}

func createIssueV1(t *testing.T) *proto.IssueV1 {
	spk, err := crypto.NewPublicKeyFromBase58(senderPK)
	assert.NoError(t, err, "NewPublicKeyFromBase58() failed")
	tx := proto.NewUnsignedIssueV1(spk, "name", "description", 10, 7, true, timestamp1, 1)
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
	blockID, err := crypto.NewSignatureFromBase58(blockSig0)
	assert.NoError(t, err, "NewSignatureFromBase58() failed")
	assetInfo := assetInfo{
		assetConstInfo: assetConstInfo{
			name:        tx.Name,
			description: tx.Description,
			decimals:    int8(tx.Decimals),
		},
		assetHistoryRecord: assetHistoryRecord{
			quantity:   *big.NewInt(int64(tx.Quantity)),
			reissuable: tx.Reissuable,
			blockID:    blockID,
		},
	}

	asset, err := proto.NewOptionalAssetFromDigest(*tx.ID)
	assert.NoError(t, err, "NewOptionalAssetFromString() failed")
	// Set proper balances and check result state.
	profileChanges := []profileChange{
		{address: senderAddr, asset: asset.String(), prevBalance: 0, newBalance: tx.Quantity},
		{address: senderAddr, asset: "", prevBalance: tx.Fee, newBalance: 0},
		{address: minerAddr, asset: "", prevBalance: 0, newBalance: tx.Fee},
	}
	diffTest(t, to, tx, profileChanges)

	// Check asset info.
	info, err := to.assets.assetInfo(asset.ID)
	assert.NoError(t, err, "assetInfo() failed")
	assert.Equal(t, assetInfo, *info, "invalid asset info after performing IssueV1 transaction")
}

func createIssueV2(t *testing.T) *proto.IssueV2 {
	spk, err := crypto.NewPublicKeyFromBase58(senderPK)
	assert.NoError(t, err, "NewPublicKeyFromBase58() failed")
	tx := proto.NewUnsignedIssueV2('W', spk, "name", "description", 10, 7, true, []byte{}, timestamp1, 1)
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
	blockID, err := crypto.NewSignatureFromBase58(blockSig0)
	assert.NoError(t, err, "NewSignatureFromBase58() failed")
	assetInfo := assetInfo{
		assetConstInfo: assetConstInfo{
			name:        tx.Name,
			description: tx.Description,
			decimals:    int8(tx.Decimals),
		},
		assetHistoryRecord: assetHistoryRecord{
			quantity:   *big.NewInt(int64(tx.Quantity)),
			reissuable: tx.Reissuable,
			blockID:    blockID,
		},
	}

	asset, err := proto.NewOptionalAssetFromDigest(*tx.ID)
	assert.NoError(t, err, "NewOptionalAssetFromString() failed")
	// Set proper balances and check result state.
	profileChanges := []profileChange{
		{address: senderAddr, asset: asset.String(), prevBalance: 0, newBalance: tx.Quantity},
		{address: senderAddr, asset: "", prevBalance: tx.Fee, newBalance: 0},
		{address: minerAddr, asset: "", prevBalance: 0, newBalance: tx.Fee},
	}
	diffTest(t, to, tx, profileChanges)

	// Check asset info.
	info, err := to.assets.assetInfo(asset.ID)
	assert.NoError(t, err, "assetInfo() failed")
	assert.Equal(t, assetInfo, *info, "invalid asset info after performing IssueV2 transaction")
}

func createReissueV1(t *testing.T, assetID crypto.Digest) *proto.ReissueV1 {
	spk, err := crypto.NewPublicKeyFromBase58(senderPK)
	assert.NoError(t, err, "NewPublicKeyFromBase58() failed")
	return proto.NewUnsignedReissueV1(spk, assetID, 1, false, timestamp1, 1)
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
	assetInfo.quantity.Add(&assetInfo.quantity, big.NewInt(int64(tx.Quantity)))

	// Set proper balances and check result state.
	profileChanges := []profileChange{
		{address: senderAddr, asset: asset.String(), prevBalance: 0, newBalance: tx.Quantity},
		{address: senderAddr, asset: "", prevBalance: tx.Fee, newBalance: 0},
		{address: minerAddr, asset: "", prevBalance: 0, newBalance: tx.Fee},
	}
	diffTest(t, to, tx, profileChanges)

	// Check asset info.
	info, err := to.assets.assetInfo(asset.ID)
	assert.NoError(t, err, "assetInfo() failed")
	assert.Equal(t, *assetInfo, *info, "invalid asset info after performing ReissueV1 transaction")
}

func createReissueV2(t *testing.T, assetID crypto.Digest) *proto.ReissueV2 {
	spk, err := crypto.NewPublicKeyFromBase58(senderPK)
	assert.NoError(t, err, "NewPublicKeyFromBase58() failed")
	return proto.NewUnsignedReissueV2('W', spk, assetID, 1, false, timestamp1, 1)
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
	assetInfo.quantity.Add(&assetInfo.quantity, big.NewInt(int64(tx.Quantity)))

	// Set proper balances and check result state.
	profileChanges := []profileChange{
		{address: senderAddr, asset: asset.String(), prevBalance: 0, newBalance: tx.Quantity},
		{address: senderAddr, asset: "", prevBalance: tx.Fee, newBalance: 0},
		{address: minerAddr, asset: "", prevBalance: 0, newBalance: tx.Fee},
	}
	diffTest(t, to, tx, profileChanges)

	// Check asset info.
	info, err := to.assets.assetInfo(asset.ID)
	assert.NoError(t, err, "assetInfo() failed")
	assert.Equal(t, *assetInfo, *info, "invalid asset info after performing ReissueV2 transaction")
}

func createBurnV1(t *testing.T, assetID crypto.Digest) *proto.BurnV1 {
	spk, err := crypto.NewPublicKeyFromBase58(senderPK)
	assert.NoError(t, err, "NewPublicKeyFromBase58() failed")
	return proto.NewUnsignedBurnV1(spk, assetID, 1, timestamp1, 1)
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
	assetInfo.quantity.Sub(&assetInfo.quantity, big.NewInt(int64(tx.Amount)))

	tests := []struct {
		txTestCase
		f runTest
	}{
		// Set insufficient balance for sender and check failure.
		{txTestCase{senderAddr, assetStr, tx.Amount - 1, []block{{timestamp1, blockSig0}}, true, "performTransactions() did not fail with insufficient balance"}, testNegBalance},
		// Set insufficient balance for sender with multiple txs in same block.
		{txTestCase{senderAddr, assetStr, tx.Amount*2 - 1, []block{{timestamp1, blockSig0}, {timestamp1, blockSig0}}, true, "performTransactions() did not fail with insufficient balance"}, testNegBalance},
		// Set insufficient balance for sender with multiple txs in different blocks.
		{txTestCase{senderAddr, assetStr, tx.Amount*2 - 1, []block{{timestamp1, blockSig0}, {timestamp1, blockSig1}}, true, "performTransactions() did not fail with insufficient balance"}, testNegBalance},
	}
	for _, c := range tests {
		c.f(t, to, tx, &c.txTestCase)
	}

	// Set proper balances and check result state.
	profileChanges := []profileChange{
		{address: senderAddr, asset: asset.String(), prevBalance: tx.Amount, newBalance: 0},
		{address: senderAddr, asset: "", prevBalance: tx.Fee, newBalance: 0},
		{address: minerAddr, asset: "", prevBalance: 0, newBalance: tx.Fee},
	}
	diffTest(t, to, tx, profileChanges)

	// Check asset info.
	info, err := to.assets.assetInfo(asset.ID)
	assert.NoError(t, err, "assetInfo() failed")
	assert.Equal(t, *assetInfo, *info, "invalid asset info after performing BurnV1 transaction")
}

func createBurnV2(t *testing.T, assetID crypto.Digest) *proto.BurnV2 {
	spk, err := crypto.NewPublicKeyFromBase58(senderPK)
	assert.NoError(t, err, "NewPublicKeyFromBase58() failed")
	return proto.NewUnsignedBurnV2('W', spk, assetID, 1, timestamp1, 1)
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
	assetInfo.quantity.Sub(&assetInfo.quantity, big.NewInt(int64(tx.Amount)))

	tests := []struct {
		txTestCase
		f runTest
	}{
		// Set insufficient balance for sender and check failure.
		{txTestCase{senderAddr, assetStr, tx.Amount - 1, []block{{timestamp1, blockSig0}}, true, "performTransactions() did not fail with insufficient balance"}, testNegBalance},
		// Set insufficient balance for sender with multiple txs in same block.
		{txTestCase{senderAddr, assetStr, tx.Amount*2 - 1, []block{{timestamp1, blockSig0}, {timestamp1, blockSig0}}, true, "performTransactions() did not fail with insufficient balance"}, testNegBalance},
		// Set insufficient balance for sender with multiple txs in different blocks.
		{txTestCase{senderAddr, assetStr, tx.Amount*2 - 1, []block{{timestamp1, blockSig0}, {timestamp1, blockSig1}}, true, "performTransactions() did not fail with insufficient balance"}, testNegBalance},
	}
	for _, c := range tests {
		c.f(t, to, tx, &c.txTestCase)
	}

	// Set proper balances and check result state.
	profileChanges := []profileChange{
		{address: senderAddr, asset: asset.String(), prevBalance: tx.Amount, newBalance: 0},
		{address: senderAddr, asset: "", prevBalance: tx.Fee, newBalance: 0},
		{address: minerAddr, asset: "", prevBalance: 0, newBalance: tx.Fee},
	}
	diffTest(t, to, tx, profileChanges)

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
	bo := proto.NewUnsignedOrderV1(buySender, mpk, *a, *pa, proto.Buy, 10e8, 100, 0, 0, 3)
	bo.Signature = &sig
	so := proto.NewUnsignedOrderV1(sellSender, mpk, *a, *pa, proto.Sell, 10e8, 100, 0, 0, 3)
	so.Signature = &sig
	return proto.NewUnsignedExchangeV1(*bo, *so, bo.Price, bo.Amount, 1, 2, 1, timestamp1)
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
	profileChanges := []profileChange{
		{address: recipientAddr, asset: assetStr, prevBalance: 0, newBalance: tx.Amount},
		{address: recipientAddr, asset: "", prevBalance: price + tx.BuyMatcherFee, newBalance: 0},
		{address: senderAddr, asset: assetStr, prevBalance: tx.Amount, newBalance: 0},
		{address: senderAddr, asset: "", prevBalance: tx.SellMatcherFee, newBalance: price},
		{address: minerAddr, asset: "", prevBalance: 0, newBalance: tx.Fee},
		{address: matcherAddr, asset: "", prevBalance: tx.Fee, newBalance: tx.SellMatcherFee + tx.BuyMatcherFee},
	}
	diffTest(t, to, tx, profileChanges)
}

func createExchangeV2(t *testing.T) *proto.ExchangeV2 {
	buySender, _ := crypto.NewPublicKeyFromBase58(recipientPK)
	sellSender, _ := crypto.NewPublicKeyFromBase58(senderPK)
	mpk, _ := crypto.NewPublicKeyFromBase58(matcherPK)
	a, _ := proto.NewOptionalAssetFromString(assetStr)
	pa, _ := proto.NewOptionalAssetFromString("")
	sig, _ := crypto.NewSignatureFromBase58("5pzyUowLi31yP4AEh5qzg7gRrvmsfeypiUkW84CKzc4H6UTzEF2RgGPLckBEqNbJGn5ofQXzuDmUnxwuP3utYp9L")
	bo := proto.NewUnsignedOrderV1(buySender, mpk, *a, *pa, proto.Buy, 10e8, 100, 0, 0, 3)
	bo.Signature = &sig
	so := proto.NewUnsignedOrderV1(sellSender, mpk, *a, *pa, proto.Sell, 10e8, 100, 0, 0, 3)
	so.Signature = &sig
	return proto.NewUnsignedExchangeV2(*bo, *so, bo.Price, bo.Amount, 1, 2, 1, timestamp1)
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
	profileChanges := []profileChange{
		{address: recipientAddr, asset: assetStr, prevBalance: 0, newBalance: tx.Amount},
		{address: recipientAddr, asset: "", prevBalance: price + tx.BuyMatcherFee, newBalance: 0},
		{address: senderAddr, asset: assetStr, prevBalance: tx.Amount, newBalance: 0},
		{address: senderAddr, asset: "", prevBalance: tx.SellMatcherFee, newBalance: price},
		{address: minerAddr, asset: "", prevBalance: 0, newBalance: tx.Fee},
		{address: matcherAddr, asset: "", prevBalance: tx.Fee, newBalance: tx.SellMatcherFee + tx.BuyMatcherFee},
	}
	diffTest(t, to, tx, profileChanges)
}

func createLeaseV1(t *testing.T, timestamp uint64) *proto.LeaseV1 {
	spk, err := crypto.NewPublicKeyFromBase58(senderPK)
	assert.NoError(t, err, "NewPublicKeyFromBase58() failed")
	rcp, err := proto.NewAddressFromString(recipientAddr)
	assert.NoError(t, err, "NewAddressFromString() failed")
	tx := proto.NewUnsignedLeaseV1(spk, proto.NewRecipientFromAddress(rcp), 100, 1, timestamp)
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, _ := crypto.GenerateKeyPair(seed)
	err = tx.Sign(sk)
	assert.NoError(t, err, "Sign() failed")
	return tx
}

func TestValidateLeaseV1(t *testing.T) {
	to, path := createTestObjects(t)

	defer func() {
		err := to.assets.db.Close()
		assert.NoError(t, err, "db.Close() failed")
		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createLeaseV1(t, timestamp2)
	tx1 := createTransferV1(t, to, recipientAddr)
	tx1.Timestamp = timestamp2
	tests := []struct {
		leaseTestCase
		f runLeaseTest
	}{
		{leaseTestCase{senderAddr, balanceProfile{tx.Amount, 100500, int64(tx.Amount)}, []block{{timestamp2, blockSig0}}, true, "performTransactions() did not fail with all balance leased"}, testInvalidLeasing},
		{leaseTestCase{senderAddr, balanceProfile{tx.Amount, 100500, int64(tx.Amount) - 1}, []block{{timestamp2, blockSig0}}, true, "performTransactions() did not fail with leased balance transfer"}, testInvalidLeasing},
	}
	tests[0].f(t, to, tx, &tests[0].leaseTestCase)
	tests[1].f(t, to, tx1, &tests[1].leaseTestCase)

	tx = createLeaseV1(t, timestamp1)
	// Set proper balances and check result state.
	profileChanges := []profileChange{
		{address: senderAddr, asset: "", prevBalance: tx.Fee + tx.Amount, newBalance: tx.Amount, newLeaseOut: int64(tx.Amount)},
		{address: recipientAddr, asset: "", newLeaseIn: int64(tx.Amount)},
		{address: minerAddr, asset: "", prevBalance: 0, newBalance: tx.Fee},
	}
	diffTest(t, to, tx, profileChanges)
}

func createLeaseV2(t *testing.T, timestamp uint64) *proto.LeaseV2 {
	spk, err := crypto.NewPublicKeyFromBase58(senderPK)
	assert.NoError(t, err, "NewPublicKeyFromBase58() failed")
	rcp, err := proto.NewAddressFromString(recipientAddr)
	assert.NoError(t, err, "NewAddressFromString() failed")
	tx := proto.NewUnsignedLeaseV2(spk, proto.NewRecipientFromAddress(rcp), 100, 1, timestamp)
	seed, _ := base58.Decode("3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc")
	sk, _ := crypto.GenerateKeyPair(seed)
	err = tx.Sign(sk)
	assert.NoError(t, err, "Sign() failed")
	return tx
}

func TestValidateLeaseV2(t *testing.T) {
	to, path := createTestObjects(t)

	defer func() {
		err := to.assets.db.Close()
		assert.NoError(t, err, "db.Close() failed")
		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	tx := createLeaseV2(t, timestamp2)
	tx1 := createTransferV2(t, to, recipientAddr)
	tx1.Timestamp = timestamp2
	tests := []struct {
		leaseTestCase
		f runLeaseTest
	}{
		{leaseTestCase{senderAddr, balanceProfile{tx.Amount, 100500, int64(tx.Amount)}, []block{{timestamp2, blockSig0}}, true, "performTransactions() did not fail with all balance leased"}, testInvalidLeasing},
		{leaseTestCase{senderAddr, balanceProfile{tx.Amount, 100500, int64(tx.Amount) - 1}, []block{{timestamp2, blockSig0}}, true, "performTransactions() did not fail with leased balance transfer"}, testInvalidLeasing},
	}
	tests[0].f(t, to, tx, &tests[0].leaseTestCase)
	tests[1].f(t, to, tx1, &tests[1].leaseTestCase)

	tx = createLeaseV2(t, timestamp1)
	// Set proper balances and check result state.
	profileChanges := []profileChange{
		{address: senderAddr, asset: "", prevBalance: tx.Fee + tx.Amount, newBalance: tx.Amount, newLeaseOut: int64(tx.Amount)},
		{address: recipientAddr, asset: "", newLeaseIn: int64(tx.Amount)},
		{address: minerAddr, asset: "", prevBalance: 0, newBalance: tx.Fee},
	}
	diffTest(t, to, tx, profileChanges)
}

func createLeaseCancelV1(t *testing.T, leaseID crypto.Digest, timestamp uint64) *proto.LeaseCancelV1 {
	spk, err := crypto.NewPublicKeyFromBase58(senderPK)
	assert.NoError(t, err, "NewPublicKeyFromBase58() failed")
	return proto.NewUnsignedLeaseCancelV1(spk, leaseID, 1, timestamp)
}

func TestValidateLeaseCancelV1(t *testing.T) {
	to, path := createTestObjects(t)

	defer func() {
		err := to.assets.db.Close()
		assert.NoError(t, err, "db.Close() failed")
		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	leaseTx := createLeaseV1(t, timestamp1)
	// Perform lease first.
	profileChanges := []profileChange{
		{address: senderAddr, asset: "", prevBalance: leaseTx.Fee + leaseTx.Amount, newBalance: leaseTx.Amount, newLeaseOut: int64(leaseTx.Amount)},
		{address: recipientAddr, asset: "", newLeaseIn: int64(leaseTx.Amount)},
		{address: minerAddr, asset: "", prevBalance: 0, newBalance: leaseTx.Fee},
	}
	diffTest(t, to, leaseTx, profileChanges)

	tx := createLeaseCancelV1(t, *leaseTx.ID, timestamp1)
	// Cancel it and check balance profiles.
	profileChanges = []profileChange{
		{address: senderAddr, asset: "", newBalance: leaseTx.Amount - tx.Fee, newLeaseOut: 0},
		{address: recipientAddr, asset: "", newLeaseIn: 0},
		{address: minerAddr, asset: "", newBalance: tx.Fee + leaseTx.Fee},
	}
	validateAndCheck(t, to, tx, profileChanges)
}

func createLeaseCancelV2(t *testing.T, leaseID crypto.Digest, timestamp uint64) *proto.LeaseCancelV2 {
	spk, err := crypto.NewPublicKeyFromBase58(senderPK)
	assert.NoError(t, err, "NewPublicKeyFromBase58() failed")
	return proto.NewUnsignedLeaseCancelV2('W', spk, leaseID, 1, timestamp)
}

func TestValidateLeaseCancelV2(t *testing.T) {
	to, path := createTestObjects(t)

	defer func() {
		err := to.assets.db.Close()
		assert.NoError(t, err, "db.Close() failed")
		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	leaseTx := createLeaseV2(t, timestamp1)
	// Perform lease first.
	profileChanges := []profileChange{
		{address: senderAddr, asset: "", prevBalance: leaseTx.Fee + leaseTx.Amount, newBalance: leaseTx.Amount, newLeaseOut: int64(leaseTx.Amount)},
		{address: recipientAddr, asset: "", newLeaseIn: int64(leaseTx.Amount)},
		{address: minerAddr, asset: "", prevBalance: 0, newBalance: leaseTx.Fee},
	}
	diffTest(t, to, leaseTx, profileChanges)

	tx := createLeaseCancelV2(t, *leaseTx.ID, timestamp1)
	// Cancel it and check balance profiles.
	profileChanges = []profileChange{
		{address: senderAddr, asset: "", newBalance: leaseTx.Amount - tx.Fee, newLeaseOut: 0},
		{address: recipientAddr, asset: "", newLeaseIn: 0},
		{address: minerAddr, asset: "", newBalance: tx.Fee + leaseTx.Fee},
	}
	validateAndCheck(t, to, tx, profileChanges)
}
