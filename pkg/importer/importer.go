package importer

import (
	"encoding/binary"
	"encoding/json"
	"log"
	"os"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const (
	blocksBatchSize = 250
	maxBlockSize    = 2 * 1024 * 1024
)

type State interface {
	AddNewBlocks(blocks [][]byte) error
	AddOldBlocks(blocks [][]byte) error
	WavesAddressesNumber() (uint64, error)
	AccountBalance(addr proto.Address, asset []byte) (uint64, error)
}

// ApplyFromFile reads blocks from blockchainPath, applying them from height startHeight and until nBlocks+1.
// Setting optimize to true speeds up the import, but it is only safe when importing blockchain from scratch
// when no rollbacks are possible at all.
func ApplyFromFile(st State, blockchainPath string, nBlocks, startHeight uint64, optimize bool) error {
	blockchain, err := os.Open(blockchainPath)
	if err != nil {
		return errors.Errorf("failed to open blockchain file: %v\n", err)
	}

	defer func() {
		if err := blockchain.Close(); err != nil {
			log.Fatalf("Failed to close blockchain file: %v\n", err)
		}
	}()

	sb := make([]byte, 4)
	var blocks [blocksBatchSize][]byte
	blocksIndex := 0
	readPos := int64(0)
	for height := uint64(1); height <= nBlocks; height++ {
		if _, err := blockchain.ReadAt(sb, readPos); err != nil {
			return err
		}
		size := binary.BigEndian.Uint32(sb)
		if size > maxBlockSize || size <= 0 {
			return errors.New("corrupted blockchain file: invalid block size")
		}
		readPos += 4
		if height < startHeight {
			readPos += int64(size)
			continue
		}
		block := make([]byte, size)
		if _, err := blockchain.ReadAt(block, readPos); err != nil {
			return err
		}
		readPos += int64(size)
		blocks[blocksIndex] = block
		blocksIndex++
		if blocksIndex != blocksBatchSize && height != nBlocks {
			continue
		}
		if optimize {
			if err := st.AddOldBlocks(blocks[:blocksIndex]); err != nil {
				return err
			}
		} else {
			if err := st.AddNewBlocks(blocks[:blocksIndex]); err != nil {
				return err
			}
		}
		blocksIndex = 0
	}
	return nil
}

func CheckBalances(st State, balancesPath string) error {
	balances, err := os.Open(balancesPath)
	if err != nil {
		return errors.Errorf("failed to open balances file: %v\n", err)
	}
	var state map[string]uint64
	jsonParser := json.NewDecoder(balances)
	if err := jsonParser.Decode(&state); err != nil {
		return errors.Errorf("failed to decode state: %v\n", err)
	}
	addressesNumber, err := st.WavesAddressesNumber()
	if err != nil {
		return errors.Errorf("failed to get number of waves addresses: %v\n", err)
	}
	properAddressesNumber := uint64(len(state))
	if properAddressesNumber != addressesNumber {
		return errors.Errorf("number of addresses differ: %d and %d\n", properAddressesNumber, addressesNumber)
	}
	for addrStr, properBalance := range state {
		addr, err := proto.NewAddressFromString(addrStr)
		if err != nil {
			return errors.Errorf("faied to convert string to address: %v\n", err)
		}
		balance, err := st.AccountBalance(addr, nil)
		if err != nil {
			return errors.Errorf("failed to get balance: %v\n", err)
		}
		if balance != properBalance {
			return errors.Errorf("balances for address %v differ: %d and %d\n", addr, properBalance, balance)
		}
	}
	if err := balances.Close(); err != nil {
		return errors.Errorf("failed to close balances file: %v\n", err)
	}
	return nil
}
