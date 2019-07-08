package state

import (
	"io/ioutil"
	"log"
	"math/big"
	"math/rand"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const (
	testSeedLen = 75

	testBloomFilterSize                     = 2e6
	testBloomFilterFalsePositiveProbability = 0.01
	testCacheSize                           = 2 * 1024 * 1024

	testPK   = "AfZtLRQxLNYH5iradMkTeuXGe71uAiATVbr8DpXEEQa8"
	testAddr = "3PDdGex1meSUf4Yq5bjPBpyAbx6us9PaLfo"

	matcherPK     = "AfZtLRQxLNYH5iradMkTeuXGe71uAiATVbr8DpXEEQa6"
	matcherAddr   = "3P9MUoSW7jfHNVFcq84rurfdWZYZuvVghVi"
	minerPK       = "AfZtLRQxLNYH5iradMkTeuXGe71uAiATVbr8DpXEEQa7"
	minerAddr     = "3PP2ywCpyvC57rN4vUZhJjQrmGMTWnjFKi7"
	senderPK      = "AfZtLRQxLNYH5iradMkTeuXGe71uAiATVbr8DpXEEQa8"
	senderAddr    = "3PNXHYoWp83VaWudq9ds9LpS5xykWuJHiHp"
	recipientPK   = "AfZtLRQxLNYH5iradMkTeuXGe71uAiATVbr8DpXEEQa9"
	recipientAddr = "3PDdGex1meSUf4Yq5bjPBpyAbx6us9PaLfo"

	assetStr  = "B2u2TBpTYHWCuMuKLnbQfLvdLJ3zjgPiy3iMS2TSYugZ"
	assetStr1 = "3gRJoK6f7XUV7fx5jUzHoPwdb9ZdTFjtTPy2HgDinr1N"

	genesisSignature = "FSH8eAAzZNqnG8xgTZtz5xuLqXySsXgAjmFEC25hXMbEufiGjqWPnGCZFt6gLiVLJny16ipxRNAkkzjjhqTjBE2"
)

var (
	blockID0 = genBlockId(1)
	blockID1 = genBlockId(2)
)

type testAddrData struct {
	pk        crypto.PublicKey
	addr      proto.Address
	wavesKey  string
	assetKey  string
	assetKey1 string
}

func newTestAddrData(pkStr, addrStr string, asset, asset1 []byte) (*testAddrData, error) {
	pk, err := crypto.NewPublicKeyFromBase58(pkStr)
	if err != nil {
		return nil, err
	}
	addr, err := proto.NewAddressFromString(addrStr)
	if err != nil {
		return nil, err
	}
	wavesKey := string((&wavesBalanceKey{addr}).bytes())
	assetKey := string((&assetBalanceKey{addr, asset}).bytes())
	assetKey1 := string((&assetBalanceKey{addr, asset1}).bytes())
	return &testAddrData{pk: pk, addr: addr, wavesKey: wavesKey, assetKey: assetKey, assetKey1: assetKey1}, nil
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

	matcherInfo   *testAddrData
	minerInfo     *testAddrData
	senderInfo    *testAddrData
	recipientInfo *testAddrData
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
	testGlobal.matcherInfo, err = newTestAddrData(matcherPK, matcherAddr, testGlobal.asset0.assetID, testGlobal.asset1.assetID)
	if err != nil {
		log.Fatalf("newTestAddrData(): %v\n", err)
	}
	testGlobal.minerInfo, err = newTestAddrData(minerPK, minerAddr, testGlobal.asset0.assetID, testGlobal.asset1.assetID)
	if err != nil {
		log.Fatalf("newTestAddrData(): %v\n", err)
	}
	testGlobal.senderInfo, err = newTestAddrData(senderPK, senderAddr, testGlobal.asset0.assetID, testGlobal.asset1.assetID)
	if err != nil {
		log.Fatalf("newTestAddrData(): %v\n", err)
	}
	testGlobal.recipientInfo, err = newTestAddrData(recipientPK, recipientAddr, testGlobal.asset0.assetID, testGlobal.asset1.assetID)
	if err != nil {
		log.Fatalf("newTestAddrData(): %v\n", err)
	}
	os.Exit(m.Run())
}

func defaultTestBloomFilterParams() keyvalue.BloomFilterParams {
	return keyvalue.BloomFilterParams{N: testBloomFilterSize, FalsePositiveProbability: testBloomFilterFalsePositiveProbability}
}

func defaultTestCacheParams() keyvalue.CacheParams {
	return keyvalue.CacheParams{Size: testCacheSize}
}

func defaultTestKeyValParams() keyvalue.KeyValParams {
	return keyvalue.KeyValParams{CacheParams: defaultTestCacheParams(), BloomFilterParams: defaultTestBloomFilterParams()}
}

type storageObjects struct {
	db      keyvalue.IterableKeyVal
	dbBatch keyvalue.Batch
	rw      *blockReadWriter
	hs      *historyStorage
	stateDB *stateDB
}

func (s *storageObjects) flush(t *testing.T) {
	err := s.rw.flush()
	assert.NoError(t, err, "rw.flush() failed")
	s.rw.reset()
	err = s.hs.flush(true)
	assert.NoError(t, err, "hs.flush() failed")
	err = s.stateDB.flush()
	assert.NoError(t, err, "stateDB.flush() failed")
	s.stateDB.reset()
}

func (s *storageObjects) addBlock(t *testing.T, blockID crypto.Signature) {
	err := s.stateDB.addBlock(blockID)
	assert.NoError(t, err, "stateDB.addBlock() failed")
	err = s.rw.startBlock(blockID)
	assert.NoError(t, err, "startBlock() failed")
	err = s.rw.finishBlock(blockID)
	assert.NoError(t, err, "finishBlock() failed")
}

func createStorageObjects() (*storageObjects, []string, error) {
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
	stateDB, err := newStateDB(db, dbBatch)
	if err != nil {
		return nil, res, err
	}
	rw, err := newBlockReadWriter(rwDir, 8, 8, db, dbBatch)
	if err != nil {
		return nil, res, err
	}
	hs, err := newHistoryStorage(db, dbBatch, rw, stateDB)
	if err != nil {
		return nil, res, err
	}
	return &storageObjects{db, dbBatch, rw, hs, stateDB}, res, nil
}

func genRandBlockIds(t *testing.T, amount int) []crypto.Signature {
	ids := make([]crypto.Signature, amount)
	for i := 0; i < amount; i++ {
		id := make([]byte, crypto.SignatureSize)
		_, err := rand.Read(id)
		assert.NoError(t, err, "rand.Read() failed")
		blockID, err := crypto.NewSignatureFromBytes(id)
		assert.NoError(t, err, "NewSignatureFromBytes() failed")
		ids[i] = blockID
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
	_, pk := crypto.GenerateKeyPair(seed)
	addr, err := proto.NewAddressFromPublicKey('W', pk)
	assert.NoError(t, err, "NewAddressFromPublicKey() failed")
	return proto.NewRecipientFromAddress(addr)
}

func defaultAssetInfo(reissuable bool) *assetInfo {
	return &assetInfo{
		assetConstInfo: assetConstInfo{
			issuer:      testGlobal.senderInfo.pk,
			name:        "asset",
			description: "description",
			decimals:    2,
		},
		assetChangeableInfo: assetChangeableInfo{
			quantity:   *big.NewInt(10000000),
			reissuable: reissuable,
		},
	}
}

func createAsset(t *testing.T, entities *blockchainEntitiesStorage, stor *storageObjects, assetID crypto.Digest) *assetInfo {
	stor.addBlock(t, blockID0)
	assetInfo := defaultAssetInfo(true)
	err := entities.assets.issueAsset(assetID, assetInfo, blockID0)
	assert.NoError(t, err, "issueAset() failed")
	stor.flush(t)
	return assetInfo
}

func activateFeature(t *testing.T, entities *blockchainEntitiesStorage, stor *storageObjects, featureID int16) {
	stor.addBlock(t, blockID0)
	blockNum, err := stor.stateDB.blockIdToNum(blockID0)
	assert.NoError(t, err, "blockIdToNum() failed")
	activationReq := &activatedFeaturesRecord{1, blockNum}
	err = entities.features.activateFeature(featureID, activationReq)
	assert.NoError(t, err, "activateFeature() failed")
	stor.flush(t)
}
