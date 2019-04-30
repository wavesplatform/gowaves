package state

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/util"
)

const (
	totalBlocksNumber = 200

	addr0 = "3P9MUoSW7jfHNVFcq84rurfdWZYZuvVghVi"
	addr1 = "3PP2ywCpyvC57rN4vUZhJjQrmGMTWnjFKi7"
	addr2 = "3PNXHYoWp83VaWudq9ds9LpS5xykWuJHiHp"
	addr3 = "3PDdGex1meSUf4Yq5bjPBpyAbx6us9PaLfo"
)

var (
	blockID0 = genBlockID(0)
	blockID1 = genBlockID(1)
)

type mockBlockInfo struct {
}

func (m *mockBlockInfo) IsValidBlock(blockID crypto.Signature) (bool, error) {
	return true, nil
}

type mockHeightInfo struct {
	rw *blockReadWriter
}

func (m *mockHeightInfo) Height() (uint64, error) {
	height, err := m.rw.currentHeight()
	if err != nil {
		return 0, err
	}
	return height, nil
}

func (m *mockHeightInfo) BlockIDToHeight(blockID crypto.Signature) (uint64, error) {
	return m.rw.heightByBlockID(blockID)
}

func (m *mockHeightInfo) NewBlockIDToHeight(blockID crypto.Signature) (uint64, error) {
	return m.rw.heightByNewBlockID(blockID)
}

func (m *mockHeightInfo) RollbackMax() uint64 {
	return rollbackMaxBlocks
}

func createBalances(rw *blockReadWriter) (*balances, []string, error) {
	res := make([]string, 1)
	dbDir0, err := ioutil.TempDir(os.TempDir(), "dbDir0")
	if err != nil {
		return nil, res, err
	}
	db, err := keyvalue.NewKeyVal(dbDir0)
	if err != nil {
		return nil, res, err
	}
	dbBatch, err := db.NewBatch()
	if err != nil {
		return nil, res, err
	}
	stor, err := newBalances(db, dbBatch, &mockHeightInfo{rw: rw}, &mockBlockInfo{})
	if err != nil {
		return nil, res, err
	}
	res = []string{dbDir0}
	return stor, res, nil
}

func genAsset(fillWith byte) []byte {
	asset := make([]byte, crypto.DigestSize, crypto.DigestSize)
	for i := 0; i < crypto.DigestSize; i++ {
		asset[i] = fillWith
	}
	return asset
}

func genAddr(fillWith byte) proto.Address {
	var addr proto.Address
	for i := 0; i < proto.AddressSize; i++ {
		addr[i] = fillWith
	}
	return addr
}

func genBlockID(fillWith byte) crypto.Signature {
	var blockID crypto.Signature
	for i := 0; i < crypto.SignatureSize; i++ {
		blockID[i] = fillWith
	}
	return blockID
}

func flush(t *testing.T, stor *balances, rw *blockReadWriter) {
	if err := rw.flush(); err != nil {
		t.Fatalf("rw.flush(): %v\n", err)
	}
	rw.reset()
	if err := rw.db.Flush(rw.dbBatch); err != nil {
		t.Fatalf("db.Flush(): %v\n", err)
	}
	if err := stor.flush(); err != nil {
		t.Fatalf("flush(): %v\n", err)
	}
	stor.reset()
	if err := stor.db.Flush(stor.dbBatch); err != nil {
		t.Fatalf("db.Flush(): %v\n", err)
	}
}

func addBlock(t *testing.T, rw *blockReadWriter, blockID crypto.Signature) {
	if err := rw.startBlock(blockID); err != nil {
		t.Fatalf("startBlock(): %v\n", err)
	}
	if err := rw.finishBlock(blockID); err != nil {
		t.Fatalf("finishBlock(): %v\n", err)
	}
}

func TestCancelAllLeases(t *testing.T) {
	rw, path0, err := createBlockReadWriter(8, 8)
	assert.NoError(t, err, "createBlockReadWriter() failed")
	stor, path1, err := createBalances(rw)
	assert.NoError(t, err, "createBalances() failed")

	defer func() {
		err := rw.db.Close()
		assert.NoError(t, err, "failed to close rw DB")
		err = stor.db.Close()
		assert.NoError(t, err, "failed to close balances DB")
		err = util.CleanTemporaryDirs(append(path0, path1...))
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	addBlock(t, rw, blockID0)
	addBlock(t, rw, blockID1)
	tests := []struct {
		addr   proto.Address
		record wavesBalanceRecord
	}{
		{genAddr(1), wavesBalanceRecord{balanceProfile{100, 1, 1}, blockID0}},
		{genAddr(2), wavesBalanceRecord{balanceProfile{2500, 2, 0}, blockID0}},
		{genAddr(3), wavesBalanceRecord{balanceProfile{10, 0, 10}, blockID1}},
		{genAddr(4), wavesBalanceRecord{balanceProfile{10, 5, 3}, blockID1}},
	}
	for _, tc := range tests {
		err = stor.setWavesBalance(tc.addr, &tc.record)
		assert.NoError(t, err, "setWavesBalance() failed")
	}
	flush(t, stor, rw)
	err = stor.cancelAllLeases()
	assert.NoError(t, err, "cancelAllLeases() failed")
	flush(t, stor, rw)
	for _, tc := range tests {
		profile, err := stor.wavesBalance(tc.addr)
		assert.NoError(t, err, "wavesBalance() failed")
		assert.Equal(t, profile.balance, tc.record.balance)
		assert.Equal(t, profile.leaseIn, int64(0))
		assert.Equal(t, profile.leaseOut, int64(0))
	}
}

func TestCancelLeaseOverflows(t *testing.T) {
	rw, path0, err := createBlockReadWriter(8, 8)
	assert.NoError(t, err, "createBlockReadWriter() failed")
	stor, path1, err := createBalances(rw)
	assert.NoError(t, err, "createBalances() failed")

	defer func() {
		err := rw.db.Close()
		assert.NoError(t, err, "failed to close rw DB")
		err = stor.db.Close()
		assert.NoError(t, err, "failed to close balances DB")
		err = util.CleanTemporaryDirs(append(path0, path1...))
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	addBlock(t, rw, blockID0)
	addBlock(t, rw, blockID1)
	tests := []struct {
		addr   string
		record wavesBalanceRecord
	}{
		{addr0, wavesBalanceRecord{balanceProfile{100, 0, 1}, blockID0}},
		{addr1, wavesBalanceRecord{balanceProfile{2500, 2, 0}, blockID0}},
		{addr2, wavesBalanceRecord{balanceProfile{10, 1, 11}, blockID1}},
		{addr3, wavesBalanceRecord{balanceProfile{10, 5, 2000}, blockID1}},
	}
	for _, tc := range tests {
		addr, err := proto.NewAddressFromString(tc.addr)
		assert.NoError(t, err, "NewAddressFromString() failed")
		err = stor.setWavesBalance(addr, &tc.record)
		assert.NoError(t, err, "setWavesBalance() failed")
	}
	flush(t, stor, rw)
	overflows, err := stor.cancelLeaseOverflows()
	assert.NoError(t, err, "cancelLeaseOverflows() failed")
	flush(t, stor, rw)
	overflowsCount := 0
	for _, tc := range tests {
		addr, err := proto.NewAddressFromString(tc.addr)
		assert.NoError(t, err, "NewAddressFromString() failed")
		profile, err := stor.wavesBalance(addr)
		assert.NoError(t, err, "wavesBalance() failed")
		assert.Equal(t, profile.balance, tc.record.balance)
		assert.Equal(t, profile.leaseIn, tc.record.leaseIn)
		if uint64(tc.record.leaseOut) > tc.record.balance {
			assert.Equal(t, profile.leaseOut, int64(0))
			if _, ok := overflows[addr]; !ok {
				t.Errorf("did not include overflowed address to the list")
			}
			overflowsCount++
		} else {
			assert.Equal(t, profile.leaseOut, tc.record.leaseOut)
		}
	}
	assert.Equal(t, len(overflows), overflowsCount)
}

func TestCancelInvalidLeaseIns(t *testing.T) {
	rw, path0, err := createBlockReadWriter(8, 8)
	assert.NoError(t, err, "createBlockReadWriter() failed")
	stor, path1, err := createBalances(rw)
	assert.NoError(t, err, "createBalances() failed")

	defer func() {
		err := rw.db.Close()
		assert.NoError(t, err, "failed to close rw DB")
		err = stor.db.Close()
		assert.NoError(t, err, "failed to close balances DB")
		err = util.CleanTemporaryDirs(append(path0, path1...))
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	addBlock(t, rw, blockID0)
	addBlock(t, rw, blockID1)
	tests := []struct {
		addr         string
		record       wavesBalanceRecord
		validLeaseIn int64
	}{
		{addr0, wavesBalanceRecord{balanceProfile{100, 0, 0}, blockID0}, 1},
		{addr1, wavesBalanceRecord{balanceProfile{2500, 2, 0}, blockID0}, 3},
		{addr2, wavesBalanceRecord{balanceProfile{10, 1, 0}, blockID1}, 1},
		{addr3, wavesBalanceRecord{balanceProfile{10, 5, 0}, blockID1}, 0},
	}
	leaseIns := make(map[proto.Address]int64)
	for _, tc := range tests {
		addr, err := proto.NewAddressFromString(tc.addr)
		assert.NoError(t, err, "NewAddressFromString() failed")
		err = stor.setWavesBalance(addr, &tc.record)
		assert.NoError(t, err, "setWavesBalance() failed")
		leaseIns[addr] = tc.validLeaseIn
	}
	flush(t, stor, rw)
	err = stor.cancelInvalidLeaseIns(leaseIns)
	assert.NoError(t, err, "cancelInvalidLeaseIns() failed")
	flush(t, stor, rw)
	for _, tc := range tests {
		addr, err := proto.NewAddressFromString(tc.addr)
		assert.NoError(t, err, "NewAddressFromString() failed")
		profile, err := stor.wavesBalance(addr)
		assert.NoError(t, err, "wavesBalance() failed")
		assert.Equal(t, profile.balance, tc.record.balance)
		assert.Equal(t, profile.leaseIn, tc.validLeaseIn)
		assert.Equal(t, profile.leaseOut, tc.record.leaseOut)
	}
}

func TestMinBalanceInRange(t *testing.T) {
	rw, path0, err := createBlockReadWriter(8, 8)
	if err != nil {
		t.Fatalf("createBlockReadWriter(): %v\n", err)
	}
	stor, path1, err := createBalances(rw)
	if err != nil {
		t.Fatalf("Can not create balances: %v\n", err)
	}

	defer func() {
		if err := rw.db.Close(); err != nil {
			t.Fatalf("Failed to close DB: %v", err)
		}
		if err := stor.db.Close(); err != nil {
			t.Fatalf("Failed to close DB: %v", err)
		}
		if err := util.CleanTemporaryDirs(append(path0, path1...)); err != nil {
			t.Fatalf("Failed to clean test data dirs: %v", err)
		}
	}()

	addr := genAddr(1)
	for i := 1; i < totalBlocksNumber; i++ {
		blockID := genBlockID(byte(i))
		addBlock(t, rw, blockID)
		r := &wavesBalanceRecord{balanceProfile{uint64(i), 0, 0}, blockID}
		if err := stor.setWavesBalance(addr, r); err != nil {
			t.Fatalf("Faied to set waves balance: %v\n", err)
		}
	}
	flush(t, stor, rw)
	minBalance, err := stor.minEffectiveBalanceInRange(addr, 1, totalBlocksNumber)
	if err != nil {
		t.Fatalf("minBalanceInRange(): %v\n", err)
	}
	if minBalance != 1 {
		t.Errorf("Invalid minimum balance in range: need %d, got %d.", 1, minBalance)
	}
	minBalance, err = stor.minEffectiveBalanceInRange(addr, 100, 150)
	if err != nil {
		t.Fatalf("minBalanceInRange(): %v\n", err)
	}
	if minBalance != 100 {
		t.Errorf("Invalid minimum balance in range: need %d, got %d.", 100, minBalance)
	}
}

func TestBalances(t *testing.T) {
	rw, path0, err := createBlockReadWriter(8, 8)
	if err != nil {
		t.Fatalf("createBlockReadWriter(): %v\n", err)
	}
	stor, path1, err := createBalances(rw)
	if err != nil {
		t.Fatalf("Can not create balances: %v\n", err)
	}

	defer func() {
		if err := rw.db.Close(); err != nil {
			t.Fatalf("Failed to close DB: %v", err)
		}
		if err := stor.db.Close(); err != nil {
			t.Fatalf("Failed to close DB: %v", err)
		}
		if err := util.CleanTemporaryDirs(append(path0, path1...)); err != nil {
			t.Fatalf("Failed to clean test data dirs: %v", err)
		}
	}()

	addBlock(t, rw, blockID0)
	addBlock(t, rw, blockID1)
	wavesTests := []struct {
		addr   proto.Address
		record wavesBalanceRecord
	}{
		{genAddr(1), wavesBalanceRecord{balanceProfile{100, 0, 0}, blockID0}},
		{genAddr(1), wavesBalanceRecord{balanceProfile{2500, 0, 0}, blockID0}},
		{genAddr(1), wavesBalanceRecord{balanceProfile{10, 5, 0}, blockID1}},
		{genAddr(1), wavesBalanceRecord{balanceProfile{10, 5, 3}, blockID1}},
	}
	for _, tc := range wavesTests {
		if err := stor.setWavesBalance(tc.addr, &tc.record); err != nil {
			t.Fatalf("Faied to set waves balance:%v\n", err)
		}
		flush(t, stor, rw)
		profile, err := stor.wavesBalance(tc.addr)
		if err != nil {
			t.Fatalf("Failed to retrieve waves balance: %v\n", err)
		}
		if *profile != tc.record.balanceProfile {
			t.Errorf("Waves balance profiles are not equal: %v and %v\n", profile, tc.record.balanceProfile)
		}
	}

	assetTests := []struct {
		addr    proto.Address
		assetID []byte
		record  assetBalanceRecord
	}{
		{genAddr(1), genAsset(1), assetBalanceRecord{100, blockID0}},
		{genAddr(1), genAsset(1), assetBalanceRecord{2500, blockID0}},
		{genAddr(1), genAsset(1), assetBalanceRecord{10, blockID1}},
	}
	for _, tc := range assetTests {
		if err := stor.setAssetBalance(tc.addr, tc.assetID, &tc.record); err != nil {
			t.Fatalf("Faied to set asset balance:%v\n", err)
		}
		flush(t, stor, rw)
		balance, err := stor.assetBalance(tc.addr, tc.assetID)
		if err != nil {
			t.Fatalf("Failed to retrieve asset balance: %v\n", err)
		}
		if balance != tc.record.balance {
			t.Errorf("Asset balances are not equal: %d and %d\n", balance, tc.record.balance)
		}
	}
}
