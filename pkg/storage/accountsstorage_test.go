package storage

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const (
	TOTAL_BLOCKS_NUMBER = 200
)

func createDbDirs() (string, string, string, error) {
	dbDir0, err := ioutil.TempDir(os.TempDir(), "dbDir0")
	if err != nil {
		return "", "", "", errors.Errorf("Failed to create db dir: %v", err)
	}
	dbDir1, err := ioutil.TempDir(os.TempDir(), "dbDir1")
	if err != nil {
		return "", "", "", errors.Errorf("Failed to create db dir: %v", err)
	}
	dbDir2, err := ioutil.TempDir(os.TempDir(), "dbDir2")
	if err != nil {
		return "", "", "", errors.Errorf("Failed to create db dir: %v", err)
	}
	return dbDir0, dbDir1, dbDir2, nil
}

func cleanDbDirs(dbDir0, dbDir1, dbDir2 string, t *testing.T) {
	if err := os.RemoveAll(dbDir0); err != nil {
		t.Fatalf("Failed to close test data dirs: %v", err)
	}
	if err := os.RemoveAll(dbDir1); err != nil {
		t.Fatalf("Failed to close test data dirs: %v", err)
	}
	if err := os.RemoveAll(dbDir2); err != nil {
		t.Fatalf("Failed to close test data dirs: %v", err)
	}
}

func createAccountsStorage(dbDir0, dbDir1, dbDir2 string) (*AccountsStorage, error) {
	globalStor, err := keyvalue.NewKeyVal(dbDir0, 0)
	if err != nil {
		return nil, err
	}
	addr2Index, err := keyvalue.NewKeyVal(dbDir1, 0)
	if err != nil {
		return nil, err
	}
	asset2Index, err := keyvalue.NewKeyVal(dbDir2, 0)
	if err != nil {
		return nil, err
	}
	return NewAccountsStorage(globalStor, addr2Index, asset2Index, "")
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

func TestBalances(t *testing.T) {
	dbDir0, dbDir1, dbDir2, err := createDbDirs()
	if err != nil {
		t.Fatalf("Can not create database directories: %v\n", err)
	}
	stor, err := createAccountsStorage(dbDir0, dbDir1, dbDir2)
	if err != nil {
		t.Fatalf("Can not create AccountsStorage: %v\n", err)
	}
	// Set first balance.
	balance := uint64(100)
	blockID := genBlockID(0)
	addr := genAddr(1)
	if err := stor.SetAccountBalance(addr, nil, balance, blockID); err != nil {
		t.Fatalf("Faied to set account balance:%v\n", err)
	}
	newBalance, err := stor.AccountBalance(addr, nil)
	if err != nil {
		t.Fatalf("Failed to retrieve account balance: %v\n", err)
	}
	if newBalance != balance {
		t.Errorf("Balances are not equal: %d and %d\n", balance, newBalance)
	}
	// Set balance in same block.
	balance = 2500
	if err := stor.SetAccountBalance(addr, nil, balance, blockID); err != nil {
		t.Fatalf("Faied to set account balance:%v\n", err)
	}
	newBalance, err = stor.AccountBalance(addr, nil)
	if err != nil {
		t.Fatalf("Failed to retrieve account balance: %v\n", err)
	}
	if newBalance != balance {
		t.Errorf("Balances are not equal: %d and %d\n", balance, newBalance)
	}
	// Set balance in new block.
	balance = 10
	blockID = genBlockID(1)
	if err := stor.SetAccountBalance(addr, nil, balance, blockID); err != nil {
		t.Fatalf("Faied to set account balance:%v\n", err)
	}
	newBalance, err = stor.AccountBalance(addr, nil)
	if err != nil {
		t.Fatalf("Failed to retrieve account balance: %v\n", err)
	}
	if newBalance != balance {
		t.Errorf("Balances are not equal: %d and %d\n", balance, newBalance)
	}
	defer cleanDbDirs(dbDir0, dbDir1, dbDir2, t)
}

func TestRollbackBlock(t *testing.T) {
	dbDir0, dbDir1, dbDir2, err := createDbDirs()
	if err != nil {
		t.Fatalf("Can not create database directories: %v\n", err)
	}
	stor, err := createAccountsStorage(dbDir0, dbDir1, dbDir2)
	if err != nil {
		t.Fatalf("Can not create AccountsStorage: %v\n", err)
	}
	addr0 := genAddr(0)
	addr1 := genAddr(1)
	asset1 := genAsset(1)
	for i := 0; i < TOTAL_BLOCKS_NUMBER; i++ {
		blockID := genBlockID(byte(i))
		if err := stor.SetAccountBalance(addr0, nil, uint64(i), blockID); err != nil {
			t.Fatalf("Faied to set account balance: %v\n", err)
		}
		if err := stor.SetAccountBalance(addr1, nil, uint64(i/2), blockID); err != nil {
			t.Fatalf("Faied to set account balance: %v\n", err)
		}
		if err := stor.SetAccountBalance(addr1, asset1, uint64(i/3), blockID); err != nil {
			t.Fatalf("Faied to set account balance: %v\n", err)
		}
	}
	for i := TOTAL_BLOCKS_NUMBER - 1; i > 0; i-- {
		balance0, err := stor.AccountBalance(addr0, nil)
		if err != nil {
			t.Fatalf("Failed to retrieve account balance: %v\n", err)
		}
		balance1, err := stor.AccountBalance(addr1, nil)
		if err != nil {
			t.Fatalf("Failed to retrieve account balance: %v\n", err)
		}
		asset1Balance, err := stor.AccountBalance(addr1, asset1)
		if err != nil {
			t.Fatalf("Failed to retrieve account balance: %v\n", err)
		}
		// Check balances.
		if balance0 != uint64(i) {
			t.Errorf("Invalid balance: %d and %d\n", balance0, i)
		}
		if balance1 != uint64(i/2) {
			t.Errorf("Invalid balance: %d and %d\n", balance1, i/2)
		}
		if asset1Balance != uint64(i/3) {
			t.Errorf("Invalid balance: %d and %d\n", asset1Balance, i/3)
		}
		// Undo block.
		blockID := genBlockID(byte(i))
		if err := stor.RollbackBlock(blockID); err != nil {
			t.Fatalf("Failed to rollback block: %v\n", err)
		}
	}
	defer cleanDbDirs(dbDir0, dbDir1, dbDir2, t)
}
