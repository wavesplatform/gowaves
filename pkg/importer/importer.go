package importer

import (
	"encoding/binary"
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const (
	KiB = 1024
	MiB = 1024 * KiB

	initTotalBatchSize = 5 * MiB
	sizeAdjustment     = 1 * MiB

	maxTotalBatchSize  = 32 * MiB
	maxBlocksBatchSize = 50000
	maxBlockSize       = 2 * MiB
)

type State interface {
	AddNewBlocks(blocks [][]byte) error
	AddOldBlocks(blocks [][]byte) error
	WavesAddressesNumber() (uint64, error)
	AccountBalance(addr proto.Address, asset []byte) (uint64, error)
}

func calculateNextMaxSizeAndDirection(maxSize int, speed, prevSpeed float64, increasingSize bool) (int, bool) {
	if speed > prevSpeed && increasingSize {
		maxSize += sizeAdjustment
		if maxSize > maxTotalBatchSize {
			maxSize = maxTotalBatchSize
		}
	} else if speed > prevSpeed && !increasingSize {
		maxSize -= sizeAdjustment
		if maxSize < initTotalBatchSize {
			maxSize = initTotalBatchSize
		}
	} else if speed < prevSpeed && increasingSize {
		increasingSize = false
		maxSize -= sizeAdjustment
		if maxSize < initTotalBatchSize {
			maxSize = initTotalBatchSize
		}
	} else if speed < prevSpeed && !increasingSize {
		increasingSize = true
		maxSize += sizeAdjustment
		if maxSize > maxTotalBatchSize {
			maxSize = maxTotalBatchSize
		}
	}
	return maxSize, increasingSize
}

// ApplyFromFile reads blocks from blockchainPath, applying them from height startHeight and until nBlocks+1.
// Setting optimize to true speeds up the import, but it is only safe when importing blockchain from scratch
// when no rollbacks are possible at all.
// If the state was rolled back at least once before, `optimize` MUST BE false.
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
	var blocks [maxBlocksBatchSize][]byte
	blocksIndex := 0
	readPos := int64(0)
	totalSize := 0
	prevSpeed := float64(0)
	increasingSize := true
	maxSize := initTotalBatchSize
	for height := uint64(1); height <= nBlocks; height++ {
		if _, err := blockchain.ReadAt(sb, readPos); err != nil {
			return err
		}
		size := binary.BigEndian.Uint32(sb)
		if size > maxBlockSize || size <= 0 {
			return errors.New("corrupted blockchain file: invalid block size")
		}
		totalSize += int(size)
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
		if (totalSize < maxSize) && (blocksIndex != maxBlocksBatchSize) && (height != nBlocks) {
			continue
		}
		start := time.Now()
		if optimize {
			if err := st.AddOldBlocks(blocks[:blocksIndex]); err != nil {
				return err
			}
		} else {
			if err := st.AddNewBlocks(blocks[:blocksIndex]); err != nil {
				return err
			}
		}
		elapsed := time.Since(start)
		speed := float64(totalSize) / float64(elapsed)
		maxSize, increasingSize = calculateNextMaxSizeAndDirection(maxSize, speed, prevSpeed, increasingSize)
		prevSpeed = speed
		totalSize = 0
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
