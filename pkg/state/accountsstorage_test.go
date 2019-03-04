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

func createAccountsStorage() (*accountsStorage, []string, error) {
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
	stor, err := newAccountsStorage(genesis, globalStor)
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

func TestBalances(t *testing.T) {
	stor, path, err := createAccountsStorage()
	if err != nil {
		t.Fatalf("Can not create accountsStorage: %v\n", err)
	}

	defer func() {
		if err := stor.db.Close(); err != nil {
			t.Fatalf("Failed to close DB: %v", err)
		}
		if err := util.CleanTemporaryDirs(path); err != nil {
			t.Fatalf("Failed to clean test data dirs: %v", err)
		}
	}()

	// Set first balance.
	balance := uint64(100)
	blockID := getBlockID(0)
	addr := genAddr(1)
	key := balanceKey{address: addr}
	if err := stor.setAccountBalance(key.bytes(), balance, blockID); err != nil {
		t.Fatalf("Faied to set account balance:%v\n", err)
	}
	if err := stor.flush(); err != nil {
		t.Fatalf("flush(): %v\n", err)
	}
	newBalance, err := stor.accountBalance(key.bytes())
	if err != nil {
		t.Fatalf("Failed to retrieve account balance: %v\n", err)
	}
	if newBalance != balance {
		t.Errorf("Balances are not equal: %d and %d\n", balance, newBalance)
	}
	// Set balance in same block.
	balance = 2500
	if err := stor.setAccountBalance(key.bytes(), balance, blockID); err != nil {
		t.Fatalf("Faied to set account balance:%v\n", err)
	}
	if err := stor.flush(); err != nil {
		t.Fatalf("flush(): %v\n", err)
	}
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
	if err := stor.setAccountBalance(key.bytes(), balance, blockID); err != nil {
		t.Fatalf("Faied to set account balance:%v\n", err)
	}
	if err := stor.flush(); err != nil {
		t.Fatalf("Failed to flush DB: %v\n", err)
	}
	newBalance, err = stor.accountBalance(key.bytes())
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
		t.Fatalf("Can not create accountsStorage: %v\n", err)
	}

	defer func() {
		if err := stor.db.Close(); err != nil {
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
		blockID := getBlockID(byte(i))
		key := balanceKey{address: addr0}
		if err := stor.setAccountBalance(key.bytes(), uint64(i), blockID); err != nil {
			t.Fatalf("Faied to set account balance: %v\n", err)
		}
		key = balanceKey{address: addr1}
		if err := stor.setAccountBalance(key.bytes(), uint64(i/2), blockID); err != nil {
			t.Fatalf("Faied to set account balance: %v\n", err)
		}
		key = balanceKey{address: addr1, asset: asset1}
		if err := stor.setAccountBalance(key.bytes(), uint64(i/3), blockID); err != nil {
			t.Fatalf("Faied to set account balance: %v\n", err)
		}
		if err := stor.flush(); err != nil {
			t.Fatalf("flush(): %v\n", err)
		}
	}
	for i := totalBlocksNumber - 1; i > 0; i-- {
		key := balanceKey{address: addr0}
		balance0, err := stor.accountBalance(key.bytes())
		if err != nil {
			t.Fatalf("Failed to retrieve account balance: %v\n", err)
		}
		key = balanceKey{address: addr1}
		balance1, err := stor.accountBalance(key.bytes())
		if err != nil {
			t.Fatalf("Failed to retrieve account balance: %v\n", err)
		}
		key = balanceKey{address: addr1, asset: asset1}
		asset1Balance, err := stor.accountBalance(key.bytes())
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
		blockID := getBlockID(byte(i))
		if err := stor.rollbackBlock(blockID); err != nil {
			t.Fatalf("Failed to rollback block: %v %d\n", err, i)
		}
	}
}
