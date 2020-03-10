package state

import (
	"io/ioutil"
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
	"github.com/wavesplatform/gowaves/pkg/ride/evaluator/ast"
	"github.com/wavesplatform/gowaves/pkg/ride/evaluator/reader"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

const (
	testSeedLen = 75

	testBloomFilterSize                     = 2e6
	testBloomFilterFalsePositiveProbability = 0.01
	testCacheSize                           = 2 * 1024 * 1024

	testPK   = "AfZtLRQxLNYH5iradMkTeuXGe71uAiATVbr8DpXEEQa8"
	testAddr = "3PDdGex1meSUf4Yq5bjPBpyAbx6us9PaLfo"

	issuerSeed    = "5TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk5bc"
	matcherSeed   = "4TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk4bc"
	minerSeed     = "3TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk3bc"
	senderSeed    = "2TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk2bc"
	recipientSeed = "1TUPTbbpiM5UmZDhMmzdsKKNgMvyHwZQncKWfJrxk1bc"

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
)

type testAddrData struct {
	sk        crypto.SecretKey
	pk        crypto.PublicKey
	addr      proto.Address
	wavesKey  string
	assetKeys []string
}

func newTestAddrData(seedStr string, assets [][]byte) (*testAddrData, error) {
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
	wavesKey := string((&wavesBalanceKey{addr}).bytes())

	assetKeys := make([]string, len(assets))
	for i, a := range assets {
		assetKeys[i] = string((&assetBalanceKey{addr, a}).bytes())
	}
	return &testAddrData{sk: sk, pk: pk, addr: addr, wavesKey: wavesKey, assetKeys: assetKeys}, nil
}

type testAssetData struct {
	asset   *proto.OptionalAsset
	assetID []byte
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
	return &testAssetData{asset, assetID.Bytes()}, nil
}

type testGlobalVars struct {
	asset0 *testAssetData
	asset1 *testAssetData
	asset2 *testAssetData

	issuerInfo    *testAddrData
	matcherInfo   *testAddrData
	minerInfo     *testAddrData
	senderInfo    *testAddrData
	recipientInfo *testAddrData

	scriptBytes []byte
	scriptAst   ast.Script
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
	testGlobal.issuerInfo, err = newTestAddrData(issuerSeed, [][]byte{testGlobal.asset0.assetID, testGlobal.asset1.assetID})
	if err != nil {
		log.Fatalf("newTestAddrData(): %v\n", err)
	}
	testGlobal.matcherInfo, err = newTestAddrData(matcherSeed, [][]byte{testGlobal.asset0.assetID, testGlobal.asset1.assetID, testGlobal.asset2.assetID})
	if err != nil {
		log.Fatalf("newTestAddrData(): %v\n", err)
	}
	testGlobal.minerInfo, err = newTestAddrData(minerSeed, [][]byte{testGlobal.asset0.assetID, testGlobal.asset1.assetID})
	if err != nil {
		log.Fatalf("newTestAddrData(): %v\n", err)
	}
	testGlobal.senderInfo, err = newTestAddrData(senderSeed, [][]byte{testGlobal.asset0.assetID, testGlobal.asset1.assetID, testGlobal.asset2.assetID})
	if err != nil {
		log.Fatalf("newTestAddrData(): %v\n", err)
	}
	testGlobal.recipientInfo, err = newTestAddrData(recipientSeed, [][]byte{testGlobal.asset0.assetID, testGlobal.asset1.assetID, testGlobal.asset2.assetID})
	if err != nil {
		log.Fatalf("newTestAddrData(): %v\n", err)
	}
	scriptBytes, err := reader.ScriptBytesFromBase64Str(scriptBase64)
	if err != nil {
		log.Fatalf("Failed to decode script from base64: %v\n", err)
	}
	testGlobal.scriptBytes = scriptBytes
	scriptAst, err := ast.BuildScript(reader.NewBytesReader(testGlobal.scriptBytes))
	if err != nil {
		log.Fatalf("BuildAst: %v\n", err)
	}
	testGlobal.scriptAst = *scriptAst
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

func defaultAssetInfo(reissuable bool) *assetInfo {
	return &assetInfo{
		assetConstInfo: assetConstInfo{
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

func createStorageObjects() (*testStorageObjects, []string, error) {
	res := make([]string, 2)
	dbDir0, err := ioutil.TempDir(os.TempDir(), "dbDir0")
	if err != nil {
		return nil, nil, err
	}
	res[0] = dbDir0
	rwDir, err := ioutil.TempDir(os.TempDir(), "rw_dir")
	if err != nil {
		return nil, res, err
	}
	res[1] = rwDir
	db, err := keyvalue.NewKeyVal(dbDir0, defaultTestKeyValParams())
	if err != nil {
		return nil, res, err
	}
	dbBatch, err := db.NewBatch()
	if err != nil {
		return nil, res, err
	}
	rw, err := newBlockReadWriter(rwDir, 8, 8, db, dbBatch, proto.MainNetScheme)
	if err != nil {
		return nil, res, err
	}
	stateDB, err := newStateDB(db, dbBatch, rw, false)
	if err != nil {
		return nil, res, err
	}
	hs, err := newHistoryStorage(db, dbBatch, stateDB)
	if err != nil {
		return nil, res, err
	}
	entities, err := newBlockchainEntitiesStorage(hs, settings.MainNetSettings, rw)
	if err != nil {
		return nil, res, err
	}
	return &testStorageObjects{db, dbBatch, rw, hs, stateDB, entities}, res, nil
}

func (s *testStorageObjects) addBlock(t *testing.T, blockID crypto.Signature) {
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

func (s *testStorageObjects) createAssetAtBlock(t *testing.T, assetID crypto.Digest, blockID crypto.Signature) *assetInfo {
	s.addBlock(t, blockID)
	assetInfo := defaultAssetInfo(true)
	err := s.entities.assets.issueAsset(assetID, assetInfo, blockID)
	assert.NoError(t, err, "issueAsset() failed")
	s.flush(t)
	return assetInfo
}

func (s *testStorageObjects) createAssetWithDecimals(t *testing.T, assetID crypto.Digest, decimals int) *assetInfo {
	s.addBlock(t, blockID0)
	assetInfo := defaultAssetInfo(true)
	require.True(t, decimals >= 0)
	assetInfo.decimals = int8(decimals)
	err := s.entities.assets.issueAsset(assetID, assetInfo, blockID0)
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
	err = s.entities.flush(true)
	assert.NoError(t, err, "entities.flush() failed")
	s.entities.reset()
	err = s.stateDB.flush()
	assert.NoError(t, err, "stateDB.flush() failed")
	s.stateDB.reset()
}

func (s *testStorageObjects) close(t *testing.T) {
	err := s.rw.close()
	assert.NoError(t, err)
	err = s.stateDB.close()
	assert.NoError(t, err)
}

func genRandBlockId(t *testing.T) crypto.Signature {
	id := make([]byte, crypto.SignatureSize)
	_, err := rand.Read(id)
	assert.NoError(t, err, "rand.Read() failed")
	blockID, err := crypto.NewSignatureFromBytes(id)
	assert.NoError(t, err, "NewSignatureFromBytes() failed")
	return blockID
}

func genRandBlockIds(t *testing.T, number int) []crypto.Signature {
	ids := make([]crypto.Signature, number)
	idsDict := make(map[crypto.Signature]bool)
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

func genBlockId(fillWith byte) crypto.Signature {
	var blockID crypto.Signature
	for i := 0; i < crypto.SignatureSize; i++ {
		blockID[i] = fillWith
	}
	return blockID
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
