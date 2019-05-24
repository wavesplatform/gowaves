package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/crypto"
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

type balancesTestObjects struct {
	stor     *storageObjects
	balances *balances
}

func createBalances() (*balancesTestObjects, []string, error) {
	stor, path, err := createStorageObjects()
	if err != nil {
		return nil, path, err
	}
	balances, err := newBalances(stor.db, stor.hs)
	if err != nil {
		return nil, path, err
	}
	return &balancesTestObjects{stor, balances}, path, nil
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

func TestCancelAllLeases(t *testing.T) {
	to, path, err := createBalances()
	assert.NoError(t, err, "createBalances() failed")

	defer func() {
		err = to.stor.stateDB.close()
		assert.NoError(t, err, "failed to close DB")
		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	to.stor.addBlock(t, blockID0)
	to.stor.addBlock(t, blockID1)
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
		err = to.balances.setWavesBalance(tc.addr, &tc.record)
		assert.NoError(t, err, "setWavesBalance() failed")
	}
	to.stor.flush(t)
	err = to.balances.cancelAllLeases()
	assert.NoError(t, err, "cancelAllLeases() failed")
	to.stor.flush(t)
	for _, tc := range tests {
		profile, err := to.balances.wavesBalance(tc.addr, true)
		assert.NoError(t, err, "wavesBalance() failed")
		assert.Equal(t, profile.balance, tc.record.balance)
		assert.Equal(t, profile.leaseIn, int64(0))
		assert.Equal(t, profile.leaseOut, int64(0))
	}
}

func TestCancelLeaseOverflows(t *testing.T) {
	to, path, err := createBalances()
	assert.NoError(t, err, "createBalances() failed")

	defer func() {
		err = to.stor.stateDB.close()
		assert.NoError(t, err, "failed to close DB")
		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	to.stor.addBlock(t, blockID0)
	to.stor.addBlock(t, blockID1)
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
		err = to.balances.setWavesBalance(addr, &tc.record)
		assert.NoError(t, err, "setWavesBalance() failed")
	}
	to.stor.flush(t)
	overflows, err := to.balances.cancelLeaseOverflows()
	assert.NoError(t, err, "cancelLeaseOverflows() failed")
	to.stor.flush(t)
	overflowsCount := 0
	for _, tc := range tests {
		addr, err := proto.NewAddressFromString(tc.addr)
		assert.NoError(t, err, "NewAddressFromString() failed")
		profile, err := to.balances.wavesBalance(addr, true)
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
	to, path, err := createBalances()
	assert.NoError(t, err, "createBalances() failed")

	defer func() {
		err = to.stor.stateDB.close()
		assert.NoError(t, err, "failed to close DB")
		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	to.stor.addBlock(t, blockID0)
	to.stor.addBlock(t, blockID1)
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
		err = to.balances.setWavesBalance(addr, &tc.record)
		assert.NoError(t, err, "setWavesBalance() failed")
		leaseIns[addr] = tc.validLeaseIn
	}
	to.stor.flush(t)
	err = to.balances.cancelInvalidLeaseIns(leaseIns)
	assert.NoError(t, err, "cancelInvalidLeaseIns() failed")
	to.stor.flush(t)
	for _, tc := range tests {
		addr, err := proto.NewAddressFromString(tc.addr)
		assert.NoError(t, err, "NewAddressFromString() failed")
		profile, err := to.balances.wavesBalance(addr, true)
		assert.NoError(t, err, "wavesBalance() failed")
		assert.Equal(t, profile.balance, tc.record.balance)
		assert.Equal(t, profile.leaseIn, tc.validLeaseIn)
		assert.Equal(t, profile.leaseOut, tc.record.leaseOut)
	}
}

func TestMinBalanceInRange(t *testing.T) {
	to, path, err := createBalances()
	assert.NoError(t, err, "createBalances() failed")

	defer func() {
		err = to.stor.stateDB.close()
		assert.NoError(t, err, "failed to close DB")
		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	addr := genAddr(1)
	for i := 0; i < totalBlocksNumber; i++ {
		blockID := genBlockID(byte(i))
		to.stor.addBlock(t, blockID)
		r := &wavesBalanceRecord{balanceProfile{uint64(i), 0, 0}, blockID}
		if err := to.balances.setWavesBalance(addr, r); err != nil {
			t.Fatalf("Faied to set waves balance: %v\n", err)
		}
	}
	to.stor.flush(t)
	minBalance, err := to.balances.minEffectiveBalanceInRange(addr, 0, totalBlocksNumber)
	if err != nil {
		t.Fatalf("minEffectiveBalanceInRange(): %v\n", err)
	}
	if minBalance != 0 {
		t.Errorf("Invalid minimum balance in range: need %d, got %d.", 0, minBalance)
	}
	minBalance, err = to.balances.minEffectiveBalanceInRange(addr, 99, 150)
	if err != nil {
		t.Fatalf("minEffectiveBalanceInRange(): %v\n", err)
	}
	if minBalance != 99 {
		t.Errorf("Invalid minimum balance in range: need %d, got %d.", 99, minBalance)
	}
}

func TestBalances(t *testing.T) {
	to, path, err := createBalances()
	assert.NoError(t, err, "createBalances() failed")

	defer func() {
		err = to.stor.stateDB.close()
		assert.NoError(t, err, "failed to close DB")
		err = util.CleanTemporaryDirs(path)
		assert.NoError(t, err, "failed to clean test data dirs")
	}()

	to.stor.addBlock(t, blockID0)
	to.stor.addBlock(t, blockID1)
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
		if err := to.balances.setWavesBalance(tc.addr, &tc.record); err != nil {
			t.Fatalf("Faied to set waves balance:%v\n", err)
		}
		to.stor.flush(t)
		profile, err := to.balances.wavesBalance(tc.addr, true)
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
		if err := to.balances.setAssetBalance(tc.addr, tc.assetID, &tc.record); err != nil {
			t.Fatalf("Faied to set asset balance:%v\n", err)
		}
		to.stor.flush(t)
		balance, err := to.balances.assetBalance(tc.addr, tc.assetID, true)
		if err != nil {
			t.Fatalf("Failed to retrieve asset balance: %v\n", err)
		}
		if balance != tc.record.balance {
			t.Errorf("Asset balances are not equal: %d and %d\n", balance, tc.record.balance)
		}
	}
}
