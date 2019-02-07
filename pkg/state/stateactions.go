package state

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"io"
	"os"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/storage"
)

func Apply(blockchainPath string, nBlocks int, manager *StateManager) error {
	blockchain, err := os.Open(blockchainPath)
	if err != nil {
		return errors.Errorf("Failed to open blockchain file: %v\n", err)
	}
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
	if err := blockchain.Close(); err != nil {
		return errors.Errorf("Failed to close blockchain file: %v\n", err)
	}
	return nil
}

func CheckBalances(balancesPath string, stor *storage.AccountsStorage) error {
	balances, err := os.Open(balancesPath)
	if err != nil {
		return errors.Errorf("Failed to open balances file: %v\n", err)
	}
	var state map[string]uint64
	jsonParser := json.NewDecoder(balances)
	if err := jsonParser.Decode(&state); err != nil {
		return errors.Errorf("Failed to decode state: %v\n", err)
	}
	addressesNumber, err := stor.WavesAddressesNumber()
	if err != nil {
		return errors.Errorf("Failed to get number of waves addresses: %v\n", err)
	}
	properAddressesNumber := uint64(len(state))
	if properAddressesNumber != addressesNumber {
		return errors.Errorf("Number of addresses differ: %d and %d\n", properAddressesNumber, addressesNumber)
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
	if err := balances.Close(); err != nil {
		return errors.Errorf("Failed to close balances file: %v\n", err)
	}
	return nil
}
