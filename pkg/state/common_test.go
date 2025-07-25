package state

import (
	"crypto/rand"
	"encoding/base64"
	"log"
	"math/big"
	"os"
	"testing"

	"github.com/mr-tron/base58/base58"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride"
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
	"github.com/wavesplatform/gowaves/pkg/ride/serialization"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

const (
	testSeedLen = 75

	testBloomFilterSize                     = 2e6
	testBloomFilterFalsePositiveProbability = 0.01
	testCacheSize                           = 2 * 1024 * 1024

	testPK   = "AfZtLRQxLNYH5iradMkTeuXGe71uAiATVbr8DpXEEQa8"
	testAddr = "3PDdGex1meSUf4Yq5bjPBpyAbx6us9PaLfo"

	issuerSeed     = "5TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk5bc"
	matcherSeed    = "4TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk4bc"
	minerSeed      = "3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc"
	senderSeed     = "2TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk2bc"
	recipientSeed  = "1TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk1bc"
	senderSKHex    = "0xffea730a62f149fd801db7966fee22c2fef23c5382cb1e4e2f1184788cef81c4"
	recipientSKHex = "0x837cd5bde5402623b2d09c9779bc585cafe5bb1a3d94b369b0b2264f7e1ef45c"

	assetStr  = "B2u2TBpTYHWCuMuKLnbQfLvdLJ3zjgPiy3iMS2TSYugZ"
	assetStr1 = "3gRJoK6f7XUV7fx5jUzHoPwdb9ZdTFjtTPy2HgDinr1N"
	assetStr3 = "6nqXFE9J94dX17MPZRB7Hkk4aYDBpybq98n25jMexYVF"

	invokeId = "B2u2TBpTYHWCuMuKLnbQfLvdLJ3zjgPiy3iMS2TSYugZ"

	defaultGenSig = "B2u2TBpTYHWCuMuKLnbQfLvdLJ3zjgPiy3iMS2TSYugZ"

	genesisSignature = "FSH8eAAzZNqnG8xgTZtz5xuLqXySsXgAjmFEC25hXMbEufiGjqWPnGCZFt6gLiVLJny16ipxRNAkkzjjhqTjBE2"

	scriptBase64 = "AgQAAAALYWxpY2VQdWJLZXkBAAAAID3+K0HJI42oXrHhtHFpHijU5PC4nn1fIFVsJp5UWrYABAAAAAlib2JQdWJLZXkBAAAAIBO1uieokBahePoeVqt4/usbhaXRq+i5EvtfsdBILNtuBAAAAAxjb29wZXJQdWJLZXkBAAAAIOfM/qkwkfi4pdngdn18n5yxNwCrBOBC3ihWaFg4gV4yBAAAAAthbGljZVNpZ25lZAMJAAH0AAAAAwgFAAAAAnR4AAAACWJvZHlCeXRlcwkAAZEAAAACCAUAAAACdHgAAAAGcHJvb2ZzAAAAAAAAAAAABQAAAAthbGljZVB1YktleQAAAAAAAAAAAQAAAAAAAAAAAAQAAAAJYm9iU2lnbmVkAwkAAfQAAAADCAUAAAACdHgAAAAJYm9keUJ5dGVzCQABkQAAAAIIBQAAAAJ0eAAAAAZwcm9vZnMAAAAAAAAAAAEFAAAACWJvYlB1YktleQAAAAAAAAAAAQAAAAAAAAAAAAQAAAAMY29vcGVyU2lnbmVkAwkAAfQAAAADCAUAAAACdHgAAAAJYm9keUJ5dGVzCQABkQAAAAIIBQAAAAJ0eAAAAAZwcm9vZnMAAAAAAAAAAAIFAAAADGNvb3BlclB1YktleQAAAAAAAAAAAQAAAAAAAAAAAAkAAGcAAAACCQAAZAAAAAIJAABkAAAAAgUAAAALYWxpY2VTaWduZWQFAAAACWJvYlNpZ25lZAUAAAAMY29vcGVyU2lnbmVkAAAAAAAAAAACqFBMLg=="
)

var (
	blockID0 = genBlockId(1)
	blockID1 = genBlockId(2)
	blockID2 = genBlockId(3)
	blockID3 = genBlockId(4)
)

type testWavesAddr interface {
	Address() proto.WavesAddress
	Recipient() proto.Recipient
	WavesKey() string
	AssetKeys() []string
}

type testWavesAddrData struct {
	sk        crypto.SecretKey
	pk        crypto.PublicKey
	addr      proto.WavesAddress
	rcp       proto.Recipient
	wavesKey  string
	assetKeys []string
}

func (t testWavesAddrData) Address() proto.WavesAddress {
	return t.addr
}

func (t testWavesAddrData) Recipient() proto.Recipient {
	return t.rcp
}

func (t testWavesAddrData) WavesKey() string {
	return t.wavesKey
}

func (t testWavesAddrData) AssetKeys() []string {
	return t.assetKeys
}

type testEthAkaWavesAddrData struct {
	sk        proto.EthereumPrivateKey
	pk        proto.EthereumPublicKey
	addr      proto.WavesAddress
	rcp       proto.Recipient
	wavesKey  string
	assetKeys []string
}

func (t testEthAkaWavesAddrData) Address() proto.WavesAddress {
	return t.addr
}

func (t testEthAkaWavesAddrData) Recipient() proto.Recipient {
	return t.rcp
}

func (t testEthAkaWavesAddrData) WavesKey() string {
	return t.wavesKey
}

func (t testEthAkaWavesAddrData) AssetKeys() []string {
	return t.assetKeys
}

func defaultBlock() *proto.BlockHeader {
	return &proto.BlockHeader{BlockSignature: blockID0.Signature(), Timestamp: defaultTimestamp}
}

func defaultBlockInfo() *proto.BlockInfo {
	genSig := crypto.MustBytesFromBase58("2eYyRDZwRCuXJhJTfwKYsqVFpBTg8v69RBppZzStWtaR")
	return proto.NewBlockInfo(proto.ProtobufBlockVersion, defaultTimestamp, 400000, 943,
		testGlobal.minerInfo.addr, testGlobal.minerInfo.pk, genSig, nil, nil)
}

func defaultDifferInfo() *differInfo {
	return &differInfo{defaultBlockInfo()}
}

func defaultAppendTxParams() *appendTxParams {
	return &appendTxParams{
		checkerInfo:   defaultCheckerInfo(),
		blockInfo:     defaultBlockInfo(),
		block:         defaultBlock(),
		acceptFailed:  false,
		validatingUtx: false,
	}
}

func defaultFallibleValidationParams() *fallibleValidationParams {
	appendTxPrms := defaultAppendTxParams()
	return &fallibleValidationParams{
		appendTxParams: appendTxPrms,
		senderScripted: false,
	}
}

func newTestWavesAddrData(seedStr string, assets []crypto.Digest) (*testWavesAddrData, error) {
	seedBytes, err := base58.Decode(seedStr)
	if err != nil {
		return nil, err
	}
	sk, pk, err := crypto.GenerateKeyPair(seedBytes)
	if err != nil {
		return nil, err
	}
	addr, err := proto.NewAddressFromPublicKey('W', pk)
	if err != nil {
		return nil, err
	}
	rcp := proto.NewRecipientFromAddress(addr)
	wavesKey := string((&wavesBalanceKey{addr.ID()}).bytes())

	assetKeys := make([]string, len(assets))
	for i, a := range assets {
		assetKeys[i] = string((&assetBalanceKey{addr.ID(), proto.AssetIDFromDigest(a)}).bytes())
	}
	return &testWavesAddrData{sk: sk, pk: pk, addr: addr, rcp: rcp, wavesKey: wavesKey, assetKeys: assetKeys}, nil
}

func newTestEthAkaWavesAddrData(ethSecretKeyHex string, assets []crypto.Digest) (*testEthAkaWavesAddrData, error) {
	sk, err := crypto.ECDSAPrivateKeyFromHexString(ethSecretKeyHex)
	if err != nil {
		return nil, err
	}

	ethSK := proto.EthereumPrivateKey(*sk)
	ethPK := proto.EthereumPublicKey(*sk.PubKey())

	addr, err := ethPK.EthereumAddress().ToWavesAddress('W')
	if err != nil {
		return nil, err
	}
	rcp := proto.NewRecipientFromAddress(addr)
	wavesKey := string((&wavesBalanceKey{addr.ID()}).bytes())

	assetKeys := make([]string, len(assets))
	for i, a := range assets {
		assetKeys[i] = string((&assetBalanceKey{addr.ID(), proto.AssetIDFromDigest(a)}).bytes())
	}
	return &testEthAkaWavesAddrData{sk: ethSK, pk: ethPK, addr: addr, rcp: rcp, wavesKey: wavesKey, assetKeys: assetKeys}, nil
}

type testAssetData struct {
	asset   *proto.OptionalAsset
	assetID crypto.Digest
}

func newTestAssetData(assetStr string) (*testAssetData, error) {
	assetID, err := crypto.NewDigestFromBase58(assetStr)
	if err != nil {
		return nil, err
	}
	asset, err := proto.NewOptionalAssetFromString(assetStr)
	if err != nil {
		return nil, err
	}
	return &testAssetData{asset, assetID}, nil
}

type testGlobalVars struct {
	asset0 *testAssetData
	asset1 *testAssetData
	asset2 *testAssetData

	issuerInfo  *testWavesAddrData
	matcherInfo *testWavesAddrData
	minerInfo   *testWavesAddrData

	senderInfo    *testWavesAddrData
	recipientInfo *testWavesAddrData

	senderEthInfo    *testEthAkaWavesAddrData
	recipientEthInfo *testEthAkaWavesAddrData

	scriptBytes []byte
	scriptAst   *ast.Tree
}

var testGlobal testGlobalVars

func TestMain(m *testing.M) {
	var err error

	testGlobal.asset0, err = newTestAssetData(assetStr)
	if err != nil {
		log.Fatalf("newTestAssetData(): %v\n", err)
	}
	testGlobal.asset1, err = newTestAssetData(assetStr1)
	if err != nil {
		log.Fatalf("newTestAssetData(): %v\n", err)
	}
	testGlobal.asset2, err = newTestAssetData(assetStr3)
	if err != nil {
		log.Fatalf("newTestAssetData(): %v\n", err)
	}

	testGlobal.issuerInfo, err = newTestWavesAddrData(issuerSeed, []crypto.Digest{testGlobal.asset0.assetID, testGlobal.asset1.assetID})
	if err != nil {
		log.Fatalf("newTestWavesAddrData(): %v\n", err)
	}
	testGlobal.matcherInfo, err = newTestWavesAddrData(matcherSeed, []crypto.Digest{testGlobal.asset0.assetID, testGlobal.asset1.assetID, testGlobal.asset2.assetID})
	if err != nil {
		log.Fatalf("newTestWavesAddrData(): %v\n", err)
	}
	testGlobal.minerInfo, err = newTestWavesAddrData(minerSeed, []crypto.Digest{testGlobal.asset0.assetID, testGlobal.asset1.assetID})
	if err != nil {
		log.Fatalf("newTestWavesAddrData(): %v\n", err)
	}

	testGlobal.senderInfo, err = newTestWavesAddrData(senderSeed, []crypto.Digest{testGlobal.asset0.assetID, testGlobal.asset1.assetID, testGlobal.asset2.assetID})
	if err != nil {
		log.Fatalf("newTestWavesAddrData(): %v\n", err)
	}
	testGlobal.recipientInfo, err = newTestWavesAddrData(recipientSeed, []crypto.Digest{testGlobal.asset0.assetID, testGlobal.asset1.assetID, testGlobal.asset2.assetID})
	if err != nil {
		log.Fatalf("newTestWavesAddrData(): %v\n", err)
	}

	testGlobal.senderEthInfo, err = newTestEthAkaWavesAddrData(senderSKHex, []crypto.Digest{testGlobal.asset0.assetID, testGlobal.asset1.assetID, testGlobal.asset2.assetID})
	if err != nil {
		log.Fatalf("newTestEthAkaWavesAddrData(): %v\n", err)
	}
	testGlobal.recipientEthInfo, err = newTestEthAkaWavesAddrData(recipientSKHex, []crypto.Digest{testGlobal.asset0.assetID, testGlobal.asset1.assetID, testGlobal.asset2.assetID})
	if err != nil {
		log.Fatalf("newTestEthAkaWavesAddrData(): %v\n", err)
	}

	scriptBytes, err := base64.StdEncoding.DecodeString(scriptBase64)
	if err != nil {
		log.Fatalf("Failed to decode script from base64: %v\n", err)
	}
	testGlobal.scriptBytes = scriptBytes
	scriptAst, err := serialization.Parse(testGlobal.scriptBytes)
	if err != nil {
		log.Fatalf("BuildAst: %v\n", err)
	}
	testGlobal.scriptAst = scriptAst

	os.Exit(m.Run())
}

func defaultTestBloomFilterParams() keyvalue.BloomFilterParams {
	return keyvalue.NewBloomFilterParams(testBloomFilterSize, testBloomFilterFalsePositiveProbability, keyvalue.NoOpStore{})
}

func defaultTestCacheParams() keyvalue.CacheParams {
	return keyvalue.CacheParams{CacheSize: testCacheSize}
}

func defaultTestKeyValParams() keyvalue.KeyValParams {
	return keyvalue.KeyValParams{CacheParams: defaultTestCacheParams(), BloomFilterParams: defaultTestBloomFilterParams()}
}

func defaultNFT(tail [proto.AssetIDTailSize]byte) *assetInfo {
	return &assetInfo{
		assetConstInfo{
			Tail:     tail,
			Issuer:   testGlobal.issuerInfo.pk,
			Decimals: 0,
			IsNFT:    true,
		},
		assetChangeableInfo{
			quantity:                 *big.NewInt(1),
			name:                     "asset",
			description:              "description",
			lastNameDescChangeHeight: 1,
			reissuable:               false,
		},
	}
}

func defaultAssetInfo(tail [12]byte, reissuable bool) *assetInfo {
	return &assetInfo{
		assetConstInfo: assetConstInfo{
			Tail:     tail,
			Issuer:   testGlobal.issuerInfo.pk,
			Decimals: 2,
		},
		assetChangeableInfo: assetChangeableInfo{
			quantity:                 *big.NewInt(10000000),
			name:                     "asset",
			description:              "description",
			lastNameDescChangeHeight: 1,
			reissuable:               reissuable,
		},
	}
}

type testStorageObjects struct {
	db       keyvalue.IterableKeyVal
	dbBatch  keyvalue.Batch
	rw       *blockReadWriter
	hs       *historyStorage
	stateDB  *stateDB
	settings *settings.BlockchainSettings

	entities *blockchainEntitiesStorage
}

func createStorageObjects(t *testing.T, amend bool) *testStorageObjects {
	return createStorageObjectsWithOptions(t, testStorageObjectsOptions{Amend: amend})
}

type testStorageObjectsOptions struct {
	Amend           bool
	Settings        *settings.BlockchainSettings
	CalculateHashes bool
}

func createStorageObjectsWithOptions(t *testing.T, options testStorageObjectsOptions) *testStorageObjects {
	if options.Settings == nil {
		options.Settings = settings.MustMainNetSettings()
	}
	db, err := keyvalue.NewKeyVal(t.TempDir(), defaultTestKeyValParams())
	require.NoError(t, err)
	// no need to close db because stateDB closes it

	dbBatch, err := db.NewBatch()
	require.NoError(t, err)

	stateDB, err := newStateDB(db, dbBatch, DefaultTestingStateParams())
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, stateDB.close())
	})

	rw, err := newBlockReadWriter(t.TempDir(), 8, 8, stateDB, options.Settings.AddressSchemeCharacter)
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, rw.close())
	})
	stateDB.setRw(rw)

	hs := newHistoryStorage(db, dbBatch, stateDB, options.Amend)

	entities, err := newBlockchainEntitiesStorage(hs, options.Settings, rw, options.CalculateHashes)
	require.NoError(t, err)

	return &testStorageObjects{db, dbBatch, rw, hs, stateDB, options.Settings, entities}
}

func (s *testStorageObjects) addRealBlock(t *testing.T, block *proto.Block) {
	blockID := block.BlockID()
	err := s.stateDB.addBlock(blockID)
	assert.NoError(t, err, "stateDB.addBlock() failed")
	err = s.rw.startBlock(blockID)
	assert.NoError(t, err, "startBlock() failed")
	err = s.rw.writeBlockHeader(&block.BlockHeader)
	assert.NoError(t, err, "writeBlockHeader() failed")
	for _, tx := range block.Transactions {
		err = s.rw.writeTransaction(tx, proto.TransactionSucceeded)
		assert.NoError(t, err, "writeTransaction() failed")
	}
	err = s.rw.finishBlock(blockID)
	assert.NoError(t, err, "finishBlock() failed")
	s.flush(t)
}

func (s *testStorageObjects) rollbackBlock(t *testing.T, blockID proto.BlockID) {
	err := s.stateDB.rollbackBlock(blockID)
	assert.NoError(t, err, "rollbackBlock() failed")
	s.flush(t)
	err = s.rw.syncWithDb()
	assert.NoError(t, err)
}

func (s *testStorageObjects) fullRollbackBlockClearCache(t *testing.T, blockID proto.BlockID) {
	s.flush(t)
	err := s.stateDB.rollback(blockID)
	assert.NoError(t, err, "rollbackBlock() failed")
	err = s.rw.syncWithDb()
	assert.NoError(t, err)
	err = s.entities.scriptsStorage.clearCache()
	assert.NoError(t, err)
	s.entities.features.clearCache()
	s.flush(t)
}

// prepareBlock makes test block officially valid (but only after batch is flushed).
func (s *testStorageObjects) prepareBlock(t *testing.T, blockID proto.BlockID) {
	err := s.stateDB.addBlock(blockID) // Assign unique block number for this block ID, add this number to the list of valid blocks.
	assert.NoError(t, err, "stateDB.addBlock() failed")
}

func (s *testStorageObjects) prepareAndStartBlock(t *testing.T, blockID proto.BlockID) {
	s.prepareBlock(t, blockID)
	err := s.rw.startBlock(blockID)
	assert.NoError(t, err, "startBlock() failed")
}

// addBlock prepares, starts and finishes fake block.
func (s *testStorageObjects) addBlock(t *testing.T, blockID proto.BlockID) {
	s.prepareAndStartBlock(t, blockID)
	s.finishBlock(t, blockID)
}

func (s *testStorageObjects) finishBlock(t *testing.T, blockID proto.BlockID) {
	err := s.rw.finishBlock(blockID)
	assert.NoError(t, err, "finishBlock() failed")
}

func (s *testStorageObjects) addBlockAndDo(t *testing.T, blockID proto.BlockID, f func(proto.BlockID)) {
	s.prepareAndStartBlock(t, blockID)
	f(blockID)
	s.finishBlock(t, blockID)
}

func (s *testStorageObjects) addBlocks(t *testing.T, blocksNum int) {
	ids := genRandBlockIds(t, blocksNum)
	for _, id := range ids {
		s.addBlock(t, id)
	}
	s.flush(t)
}

func (s *testStorageObjects) createAssetUsingInfo(t *testing.T, assetID crypto.Digest, info *assetInfo) {
	s.addBlock(t, blockID0)
	err := s.entities.assets.issueAsset(proto.AssetIDFromDigest(assetID), info, blockID0)
	assert.NoError(t, err, "issueAsset() failed")
	s.flush(t)
}

func (s *testStorageObjects) createAssetAtBlock(t *testing.T, assetID crypto.Digest, blockID proto.BlockID) *assetInfo {
	s.addBlock(t, blockID)
	assetInfo := defaultAssetInfo(proto.DigestTail(assetID), true)
	err := s.entities.assets.issueAsset(proto.AssetIDFromDigest(assetID), assetInfo, blockID)
	assert.NoError(t, err, "issueAsset() failed")
	s.flush(t)
	return assetInfo
}

func (s *testStorageObjects) createAssetWithDecimals(t *testing.T, assetID crypto.Digest, decimals int) *assetInfo {
	s.addBlock(t, blockID0)
	assetInfo := defaultAssetInfo(proto.DigestTail(assetID), true)
	require.True(t, decimals >= 0)
	assetInfo.Decimals = uint8(decimals)
	err := s.entities.assets.issueAsset(proto.AssetIDFromDigest(assetID), assetInfo, blockID0)
	assert.NoError(t, err, "issueAsset() failed")
	s.flush(t)
	return assetInfo
}

func (s *testStorageObjects) createAssetUsingRandomBlock(t *testing.T, assetID crypto.Digest) *assetInfo {
	return s.createAssetAtBlock(t, assetID, genRandBlockId(t))
}

func (s *testStorageObjects) createAsset(t *testing.T, assetID crypto.Digest) *assetInfo {
	return s.createAssetAtBlock(t, assetID, blockID0)
}

func (s *testStorageObjects) createSmartAsset(t *testing.T, assetID crypto.Digest) {
	s.addBlock(t, blockID0)
	err := s.entities.scriptsStorage.setAssetScript(assetID, testGlobal.scriptBytes, blockID0)
	assert.NoError(t, err, "setAssetScript failed")
	s.flush(t)
}

func (s *testStorageObjects) setWavesBalance(
	t *testing.T,
	addr proto.WavesAddress,
	bp balanceProfile,
	blockID proto.BlockID,
) {
	wb := newWavesValueFromProfile(bp)
	err := s.entities.balances.setWavesBalance(addr.ID(), wb, blockID)
	assert.NoError(t, err, "setWavesBalance() failed")
}

func (s *testStorageObjects) transferWaves(
	t *testing.T,
	from, to proto.WavesAddress,
	amount uint64,
	blockID proto.BlockID,
) {
	fromBP, err := s.entities.balances.newestWavesBalance(from.ID())
	require.NoError(t, err, "newestWavesBalance() failed")
	if fromBalance := fromBP.spendableBalance(); fromBalance < amount {
		require.Failf(t, "transferWaves()", "not enough balance at account '%s': %d < %d",
			from.String(), fromBalance, amount,
		)
	}
	fromBP.balance -= amount
	s.setWavesBalance(t, from, fromBP, blockID)

	toBalance, err := s.entities.balances.newestWavesBalance(to.ID())
	require.NoError(t, err, "newestWavesBalance() failed")
	toBalance.balance += amount
	s.setWavesBalance(t, to, toBalance, blockID)
}

func (s *testStorageObjects) createAlias(t testing.TB, addr proto.WavesAddress, alias string, blockID proto.BlockID) {
	err := s.entities.aliases.createAlias(alias, addr, blockID)
	assert.NoError(t, err, "createAlias() failed")
}

func storeScriptByAddress(
	stor *blockchainEntitiesStorage,
	scheme proto.Scheme,
	senderPK crypto.PublicKey,
	script proto.Script,
	se scriptEstimation,
	blockID proto.BlockID,
) error {
	senderAddr, err := proto.NewAddressFromPublicKey(scheme, senderPK)
	if err != nil {
		return errors.Wrapf(err, "failed to create addr from PK %q", senderPK.String())
	}
	if setErr := stor.scriptsStorage.setAccountScript(senderAddr, script, senderPK, blockID); setErr != nil {
		return errors.Wrapf(setErr, "failed to set account script on addr %q", senderAddr.String())
	}
	if setErr := stor.scriptsComplexity.saveComplexitiesForAddr(senderAddr, se, blockID); setErr != nil {
		return errors.Wrapf(setErr, "failed to save script complexities for addr %q", senderAddr.String())
	}
	return nil
}

func (s *testStorageObjects) setScript(t *testing.T, pk crypto.PublicKey, script proto.Script, blockID proto.BlockID) {
	var est ride.TreeEstimation
	if !script.IsEmpty() {
		tree, err := serialization.Parse(script)
		require.NoError(t, err)
		est, err = ride.EstimateTree(tree, maxEstimatorVersion)
		require.NoError(t, err)
	}
	se := scriptEstimation{
		currentEstimatorVersion: maxEstimatorVersion,
		scriptIsEmpty:           script.IsEmpty(),
		estimation:              est,
	}
	err := storeScriptByAddress(s.entities, s.settings.AddressSchemeCharacter, pk, script, se, blockID)
	require.NoError(t, err)
}

func (s *testStorageObjects) activateFeatureWithBlock(t *testing.T, featureID int16, blockID proto.BlockID) {
	activationReq := &activatedFeaturesRecord{1}
	err := s.entities.features.activateFeature(featureID, activationReq, blockID)
	assert.NoError(t, err, "activateFeature() failed")
}

func (s *testStorageObjects) activateFeature(t *testing.T, featureID int16) {
	s.addBlock(t, blockID0)
	s.activateFeatureWithBlock(t, featureID, blockID0)
	s.flush(t)
}

func (s *testStorageObjects) activateSponsorship(t *testing.T) {
	s.activateFeature(t, int16(settings.FeeSponsorship))
	windowSize := settings.MustMainNetSettings().ActivationWindowSize(1)
	s.addBlocks(t, int(windowSize))
}

func (s *testStorageObjects) flush(t *testing.T) {
	err := s.rw.flush()
	assert.NoError(t, err, "rw.flush() failed")
	s.rw.reset()
	err = s.entities.flush()
	assert.NoError(t, err, "entities.flush() failed")
	s.entities.reset()
	err = s.stateDB.flush()
	assert.NoError(t, err, "stateDB.flush() failed")
	s.stateDB.reset()
}

func genRandBlockId(t *testing.T) proto.BlockID {
	id := make([]byte, crypto.SignatureSize)
	_, err := rand.Read(id)
	assert.NoError(t, err, "rand.Read() failed")
	blockID, err := proto.NewBlockIDFromBytes(id)
	assert.NoError(t, err, "NewBlockIDFromBytes() failed")
	return blockID
}

func genRandBlockIds(t *testing.T, number int) []proto.BlockID {
	ids := make([]proto.BlockID, number)
	idsDict := make(map[proto.BlockID]bool)
	for i := range number {
		for {
			blockID := genRandBlockId(t)
			if _, ok := idsDict[blockID]; ok {
				continue
			}
			ids[i] = blockID
			idsDict[blockID] = true
			break
		}
	}
	return ids
}

func genBlockId(fillWith byte) proto.BlockID {
	var s crypto.Signature
	for i := range crypto.SignatureSize {
		s[i] = fillWith
	}
	return proto.NewBlockIDFromSignature(s)
}

func generateRandomRecipient(t *testing.T) proto.Recipient {
	seed := make([]byte, testSeedLen)
	_, err := rand.Read(seed)
	assert.NoError(t, err, "rand.Read() failed")
	_, pk, err := crypto.GenerateKeyPair(seed)
	assert.NoError(t, err)
	addr, err := proto.NewAddressFromPublicKey('W', pk)
	assert.NoError(t, err, "NewAddressFromPublicKey() failed")
	return proto.NewRecipientFromAddress(addr)
}

func existingGenesisTx(t *testing.T) proto.Transaction {
	sig, err := crypto.NewSignatureFromBase58("2DVtfgXjpMeFf2PQCqvwxAiaGbiDsxDjSdNQkc5JQ74eWxjWFYgwvqzC4dn7iB1AhuM32WxEiVi1SGijsBtYQwn8")
	assert.NoError(t, err, "NewSignatureFromBase58() failed")
	addr, err := proto.NewAddressFromString("3PAWwWa6GbwcJaFzwqXQN5KQm7H96Y7SHTQ")
	assert.NoError(t, err, "NewAddressFromString() failed")
	return &proto.Genesis{
		Type:      proto.GenesisTransaction,
		Version:   1,
		ID:        &sig,
		Signature: &sig,
		Timestamp: 1465742577614,
		Recipient: addr,
		Amount:    9999999500000000,
	}
}
