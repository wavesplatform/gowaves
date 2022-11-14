package state

import (
	"encoding/base64"
	"log"
	"math/big"
	"math/rand"
	"os"
	"testing"

	"github.com/mr-tron/base58/base58"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/proto"
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
	return &proto.BlockInfo{
		Timestamp:           defaultTimestamp,
		Height:              400000,
		BaseTarget:          943,
		GenerationSignature: genSig,
		Generator:           testGlobal.minerInfo.addr,
		GeneratorPublicKey:  testGlobal.minerInfo.pk,
	}
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
	return keyvalue.CacheParams{Size: testCacheSize}
}

func defaultTestKeyValParams() keyvalue.KeyValParams {
	return keyvalue.KeyValParams{CacheParams: defaultTestCacheParams(), BloomFilterParams: defaultTestBloomFilterParams()}
}

func defaultNFT(tail [proto.AssetIDTailSize]byte) *assetInfo {
	return &assetInfo{
		assetConstInfo{
			tail:     tail,
			issuer:   testGlobal.issuerInfo.pk,
			decimals: 0,
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
			tail:     tail,
			issuer:   testGlobal.issuerInfo.pk,
			decimals: 2,
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
	db      keyvalue.IterableKeyVal
	dbBatch keyvalue.Batch
	rw      *blockReadWriter
	hs      *historyStorage
	stateDB *stateDB

	entities *blockchainEntitiesStorage
}

func createStorageObjects(t *testing.T, amend bool) *testStorageObjects {
	return createStorageObjectsWithOptions(t, testStorageObjectsOptions{Amend: amend})
}

type testStorageObjectsOptions struct {
	Amend    bool
	Scheme   proto.Scheme
	Settings *settings.BlockchainSettings
}

func createStorageObjectsWithOptions(t *testing.T, options testStorageObjectsOptions) *testStorageObjects {
	if options.Settings == nil {
		options.Settings = settings.MainNetSettings
	}
	if options.Scheme == 0 {
		options.Scheme = proto.MainNetScheme
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

	rw, err := newBlockReadWriter(t.TempDir(), 8, 8, stateDB, options.Scheme)
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, rw.close())
	})
	stateDB.setRw(rw)

	hs, err := newHistoryStorage(db, dbBatch, stateDB, options.Amend)
	require.NoError(t, err)

	entities, err := newBlockchainEntitiesStorage(hs, options.Settings, rw, false)
	require.NoError(t, err)

	return &testStorageObjects{db, dbBatch, rw, hs, stateDB, entities}
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
		err = s.rw.writeTransaction(tx, false)
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

func (s *testStorageObjects) addBlock(t *testing.T, blockID proto.BlockID) {
	err := s.stateDB.addBlock(blockID)
	assert.NoError(t, err, "stateDB.addBlock() failed")
	err = s.rw.startBlock(blockID)
	assert.NoError(t, err, "startBlock() failed")
	err = s.rw.finishBlock(blockID)
	assert.NoError(t, err, "finishBlock() failed")
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
	assetInfo.decimals = int8(decimals)
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
	err := s.entities.scriptsStorage.setAssetScript(assetID, testGlobal.scriptBytes, testGlobal.senderInfo.pk, blockID0)
	assert.NoError(t, err, "setAssetScript failed")
	s.flush(t)
}

func (s *testStorageObjects) activateFeature(t *testing.T, featureID int16) {
	s.addBlock(t, blockID0)
	activationReq := &activatedFeaturesRecord{1}
	err := s.entities.features.activateFeature(featureID, activationReq, blockID0)
	assert.NoError(t, err, "activateFeature() failed")
	s.flush(t)
}

func (s *testStorageObjects) activateSponsorship(t *testing.T) {
	s.activateFeature(t, int16(settings.FeeSponsorship))
	windowSize := settings.MainNetSettings.ActivationWindowSize(1)
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
	for i := 0; i < number; i++ {
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
	for i := 0; i < crypto.SignatureSize; i++ {
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
