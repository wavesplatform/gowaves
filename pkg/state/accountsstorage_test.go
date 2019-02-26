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

func createAccountsStorage() (*AccountsStorage, []string, error) {
	res := make([]string, 1)
	dbDir0, err := ioutil.TempDir(os.TempDir(), "dbDir0")
	if err != nil {
		return nil, res, err
	}
	globalStor, err := keyvalue.NewKeyVal(dbDir0, true)
	if err != nil {
		return nil, res, err
	}
	genesis, err := crypto.NewSignatureFromBase58(genesisSignature)
	if err != nil {
		return nil, res, err
	}
	stor, err := NewAccountsStorage(genesis, globalStor)
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

func TestBalances(t *testing.T) {
	stor, path, err := createAccountsStorage()
	if err != nil {
		t.Fatalf("Can not create AccountsStorage: %v\n", err)
	}

	defer func() {
		if err := stor.Db.Close(); err != nil {
			t.Fatalf("Failed to close DB: %v", err)
		}
		if err := util.CleanTemporaryDirs(path); err != nil {
			t.Fatalf("Failed to clean test data dirs: %v", err)
		}
	}()

	// Set first balance.
	balance := uint64(100)
	blockID := genBlockID(0)
	addr := genAddr(1)
	key := BalanceKey{Address: addr}
	if err := stor.SetAccountBalance(key.Bytes(), balance, blockID); err != nil {
		t.Fatalf("Faied to set account balance:%v\n", err)
	}
	if err := stor.Flush(); err != nil {
		t.Fatalf("Flush(): %v\n", err)
	}
	newBalance, err := stor.AccountBalance(key.Bytes())
	if err != nil {
		t.Fatalf("Failed to retrieve account balance: %v\n", err)
	}
	if newBalance != balance {
		t.Errorf("Balances are not equal: %d and %d\n", balance, newBalance)
	}
	// Set balance in same block.
	balance = 2500
	if err := stor.SetAccountBalance(key.Bytes(), balance, blockID); err != nil {
		t.Fatalf("Faied to set account balance:%v\n", err)
	}
	if err := stor.Flush(); err != nil {
		t.Fatalf("Flush(): %v\n", err)
	}
	newBalance, err = stor.AccountBalance(key.Bytes())
	if err != nil {
		t.Fatalf("Failed to retrieve account balance: %v\n", err)
	}
	if newBalance != balance {
		t.Errorf("Balances are not equal: %d and %d\n", balance, newBalance)
	}
	// Set balance in new block.
	balance = 10
	blockID = genBlockID(1)
	if err := stor.SetAccountBalance(key.Bytes(), balance, blockID); err != nil {
		t.Fatalf("Faied to set account balance:%v\n", err)
	}
	if err := stor.Flush(); err != nil {
		t.Fatalf("Failed to flush DB: %v\n", err)
	}
	newBalance, err = stor.AccountBalance(key.Bytes())
	if err != nil {
		t.Fatalf("Failed to retrieve account balance: %v\n", err)
	}
	if newBalance != balance {
		t.Errorf("Balances are not equal: %d and %d\n", balance, newBalance)
	}
}

func TestRollbackBlock(t *testing.T) {
	stor, path, err := createAccountsStorage()
	if err != nil {
		t.Fatalf("Can not create AccountsStorage: %v\n", err)
	}

	defer func() {
		if err := stor.Db.Close(); err != nil {
			t.Fatalf("Failed to close DB: %v", err)
		}
		if err := util.CleanTemporaryDirs(path); err != nil {
			t.Fatalf("Failed to clean test data dirs: %v", err)
		}
	}()

	addr0 := genAddr(0)
	addr1 := genAddr(1)
	asset1 := genAsset(1)
	for i := 0; i < totalBlocksNumber; i++ {
		blockID := genBlockID(byte(i))
		key := BalanceKey{Address: addr0}
		if err := stor.SetAccountBalance(key.Bytes(), uint64(i), blockID); err != nil {
			t.Fatalf("Faied to set account balance: %v\n", err)
		}
		key = BalanceKey{Address: addr1}
		if err := stor.SetAccountBalance(key.Bytes(), uint64(i/2), blockID); err != nil {
			t.Fatalf("Faied to set account balance: %v\n", err)
		}
		key = BalanceKey{Address: addr1, Asset: asset1}
		if err := stor.SetAccountBalance(key.Bytes(), uint64(i/3), blockID); err != nil {
			t.Fatalf("Faied to set account balance: %v\n", err)
		}
		if err := stor.Flush(); err != nil {
			t.Fatalf("Flush(): %v\n", err)
		}
	}
	for i := totalBlocksNumber - 1; i > 0; i-- {
		key := BalanceKey{Address: addr0}
		balance0, err := stor.AccountBalance(key.Bytes())
		if err != nil {
			t.Fatalf("Failed to retrieve account balance: %v\n", err)
		}
		key = BalanceKey{Address: addr1}
		balance1, err := stor.AccountBalance(key.Bytes())
		if err != nil {
			t.Fatalf("Failed to retrieve account balance: %v\n", err)
		}
		key = BalanceKey{Address: addr1, Asset: asset1}
		asset1Balance, err := stor.AccountBalance(key.Bytes())
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
			t.Fatalf("Failed to rollback block: %v %d\n", err, i)
		}
	}
}
