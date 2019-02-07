package state

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"io"
	"log"
	"os"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/storage"
	"github.com/wavesplatform/gowaves/pkg/util"
)

func importAndApply(blockchain *os.File, nBlocks int, manager *StateManager) error {
	sb := make([]byte, 4)
	buf := make([]byte, 2*1024*1024)
	r := bufio.NewReader(blockchain)
	for i := 0; i < nBlocks; i++ {
		if _, err := io.ReadFull(r, sb); err != nil {
			return err
		}
		s := binary.BigEndian.Uint32(sb)
		block := buf[:s]
		if _, err := io.ReadFull(r, block); err != nil {
			return err
		}
		if err := manager.AcceptAndVerifyBlockBinary(block, true); err != nil {
			return err
		}
	}
	return nil
}

func decodeAndCheckBalances(stor *storage.AccountsStorage, balances *os.File) error {
	var state map[string]uint64
	jsonParser := json.NewDecoder(balances)
	if err := jsonParser.Decode(&state); err != nil {
		return errors.Errorf("Failed to decode state: %v\n", err)
	}
	for addrStr, properBalance := range state {
		addr, err := proto.NewAddressFromString(addrStr)
		if err != nil {
			return errors.Errorf("Faied to convert string to address: %v\n", err)
		}
		balance, err := stor.AccountBalance(addr, nil)
		if err != nil {
			return errors.Errorf("Failed to get balance: %v\n", err)
		}
		if balance != properBalance {
			return errors.Errorf("Balances for address %v differ: %d and %d\n", addr, properBalance, balance)
		}
	}
	return nil
}

func CheckState(blockchainPath, balancesPath string, batchSize, nBlocks int) error {
	blockchain, err := os.Open(blockchainPath)
	if err != nil {
		return errors.Errorf("Failed to open blockchain file: %v\n", err)
	}
	rw, rwPath, err := storage.CreateTestBlockReadWriter(batchSize, 8, 8)
	if err != nil {
		return errors.Errorf("CreateTesBlockReadWriter: %v\n", err)
	}
	idsFile, err := rw.BlockIdsFilePath()
	if err != nil {
		return errors.Errorf("Failed to get path of ids file: %v\n", err)
	}
	stor, storPath, err := storage.CreateTestAccountsStorage(idsFile)
	if err != nil {
		return errors.Errorf("CreateTestAccountStorage: %v\n", err)
	}

	defer func() {
		if err := rw.Close(); err != nil {
			log.Fatalf("Failed to close BlockReadWriter: %v\n", err)
		}
		if err := util.CleanTemporaryDirs(rwPath); err != nil {
			log.Fatalf("Failed to clean data dirs: %v\n", err)
		}
		if err := util.CleanTemporaryDirs(storPath); err != nil {
			log.Fatalf("Failed to clean data dirs: %v\n", err)
		}
	}()

	manager, err := NewStateManager(stor, rw)
	if err != nil {
		return errors.Errorf("Failed to create state manager: %v.\n", err)
	}
	if err := importAndApply(blockchain, nBlocks, manager); err != nil {
		return errors.Errorf("Failed to import: %v\n", err)
	}
	if err := blockchain.Close(); err != nil {
		return errors.Errorf("Failed to close blockchain file: %v\n\n", err)
	}
	if len(balancesPath) != 0 {
		balances, err := os.Open(balancesPath)
		if err != nil {
			return errors.Errorf("Failed to open balances file: %v\n", err)
		}
		if err := decodeAndCheckBalances(stor, balances); err != nil {
			return errors.Errorf("Balance checker: %v\n", err)
		}
		if err := balances.Close(); err != nil {
			return errors.Errorf("Failed to close balances file: %v\n", err)
		}
	}
	return nil
}
