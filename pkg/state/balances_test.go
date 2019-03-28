package state

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/util"
)

const (
	totalBlocksNumber = 200
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
	return height + 2, nil
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

func getBlockID(fillWith byte) crypto.Signature {
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

	key := balanceKey{address: genAddr(1)}
	for i := 2; i < totalBlocksNumber; i++ {
		blockID := getBlockID(byte(i))
		addBlock(t, rw, blockID)
		if err := stor.setAccountBalance(key.bytes(), uint64(i), blockID); err != nil {
			t.Fatalf("Faied to set account balance: %v\n", err)
		}
	}
	flush(t, stor, rw)
	minBalance, err := stor.minBalanceInRange(key.bytes(), 2, totalBlocksNumber)
	if err != nil {
		t.Fatalf("minBalanceInRange(): %v\n", err)
	}
	if minBalance != 2 {
		t.Errorf("Invalid minimum balance in range: need %d, got %d.", 2, minBalance)
	}
	minBalance, err = stor.minBalanceInRange(key.bytes(), 100, 150)
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

	// Set first balance.
	balance := uint64(100)
	blockID := getBlockID(0)
	addr := genAddr(1)
	key := balanceKey{address: addr}
	addBlock(t, rw, blockID)
	if err := stor.setAccountBalance(key.bytes(), balance, blockID); err != nil {
		t.Fatalf("Faied to set account balance:%v\n", err)
	}
	flush(t, stor, rw)
	newBalance, err := stor.accountBalance(key.bytes())
	if err != nil {
		t.Fatalf("Failed to retrieve account balance: %v\n", err)
	}
	if newBalance != balance {
		t.Errorf("Balances are not equal: %d and %d\n", balance, newBalance)
	}
	// Set balance in same block.
	balance = 2500
	addBlock(t, rw, blockID)
	if err := stor.setAccountBalance(key.bytes(), balance, blockID); err != nil {
		t.Fatalf("Faied to set account balance:%v\n", err)
	}
	flush(t, stor, rw)
	newBalance, err = stor.accountBalance(key.bytes())
	if err != nil {
		t.Fatalf("Failed to retrieve account balance: %v\n", err)
	}
	if newBalance != balance {
		t.Errorf("Balances are not equal: %d and %d\n", balance, newBalance)
	}
	// Set balance in new block.
	balance = 10
	blockID = getBlockID(1)
	addBlock(t, rw, blockID)
	if err := stor.setAccountBalance(key.bytes(), balance, blockID); err != nil {
		t.Fatalf("Faied to set account balance:%v\n", err)
	}
	flush(t, stor, rw)
	newBalance, err = stor.accountBalance(key.bytes())
	if err != nil {
		t.Fatalf("Failed to retrieve account balance: %v\n", err)
	}
	if newBalance != balance {
		t.Errorf("Balances are not equal: %d and %d\n", balance, newBalance)
	}
}
