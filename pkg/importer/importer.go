package importer

import (
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
)

const (
	KiB = 1024
	MiB = 1024 * KiB

	initTotalBatchSize = 5 * MiB
	sizeAdjustment     = 1 * MiB

	MaxTotalBatchSize               = 20 * MiB
	MaxTotalBatchSizeForNetworkSync = 6 * MiB
	MaxBlocksBatchSize              = 50000
	MaxBlockSize                    = 2 * MiB
)

type State interface {
	AddBlocks(blocks [][]byte) error
	WavesAddressesNumber() (uint64, error)
	WavesBalance(account proto.Recipient) (uint64, error)
	AssetBalance(account proto.Recipient, assetID proto.AssetID) (uint64, error)
	ShouldPersistAddressTransactions() (bool, error)
	PersistAddressTransactions() error
}

func maybePersistTxs(st State) error {
	// Check if we need to persist transactions for extended API.
	persistTxs, err := st.ShouldPersistAddressTransactions()
	if err != nil {
		return err
	}
	if persistTxs {
		return st.PersistAddressTransactions()
	}
	return nil
}

func calculateNextMaxSizeAndDirection(maxSize int, speed, prevSpeed float64, increasingSize bool) (int, bool) {
	if speed > prevSpeed && increasingSize {
		maxSize += sizeAdjustment
		if maxSize > MaxTotalBatchSize {
			maxSize = MaxTotalBatchSize
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
		if maxSize > MaxTotalBatchSize {
			maxSize = MaxTotalBatchSize
		}
	}
	return maxSize, increasingSize
}

// ApplyFromFile reads blocks from blockchainPath, applying them from height startHeight and until nBlocks+1.
// Setting optimize to true speeds up the import, but it is only safe when importing blockchain from scratch
// when no rollbacks are possible at all.
func ApplyFromFile(st State, blockchainPath string, nBlocks, startHeight uint64) error {
	blockchain, err := os.Open(blockchainPath) // #nosec: in this case check for prevent G304 (CWE-22) is not necessary
	if err != nil {
		return errors.Errorf("failed to open blockchain file: %v", err)
	}

	defer func() {
		if err := blockchain.Close(); err != nil {
			zap.S().Fatalf("Failed to close blockchain file: %v", err)
		}
	}()
	sb := make([]byte, 4)
	var blocks [MaxBlocksBatchSize][]byte
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
		if size > MaxBlockSize || size == 0 {
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
		if (totalSize < maxSize) && (blocksIndex != MaxBlocksBatchSize) && (height != nBlocks) {
			continue
		}
		start := time.Now()
		if err := st.AddBlocks(blocks[:blocksIndex]); err != nil {
			return err
		}
		elapsed := time.Since(start)
		speed := float64(totalSize) / float64(elapsed)
		maxSize, increasingSize = calculateNextMaxSizeAndDirection(maxSize, speed, prevSpeed, increasingSize)
		prevSpeed = speed
		totalSize = 0
		blocksIndex = 0
		if err := maybePersistTxs(st); err != nil {
			return err
		}
	}
	return nil
}

func CheckBalances(st State, balancesPath string) (err error) {
	balances, err := os.Open(filepath.Clean(balancesPath))
	if err != nil {
		return errors.Wrapf(err, "failed to open balances file %q", balancesPath)
	}
	defer func() {
		if closeErr := balances.Close(); closeErr != nil {
			zap.S().Fatalf("Failed to close balances file: %v", closeErr)
		}
	}()
	var state map[string]uint64
	jsonParser := json.NewDecoder(balances)
	if err := jsonParser.Decode(&state); err != nil {
		return errors.Errorf("failed to decode state: %v", err)
	}
	addressesNumber, err := st.WavesAddressesNumber()
	if err != nil {
		return errors.Errorf("failed to get number of waves addresses: %v", err)
	}
	properAddressesNumber := uint64(len(state))
	if properAddressesNumber != addressesNumber {
		return errors.Errorf("number of addresses differ: %d and %d", properAddressesNumber, addressesNumber)
	}
	for addrStr, properBalance := range state {
		addr, err := proto.NewAddressFromString(addrStr)
		if err != nil {
			return errors.Errorf("faied to convert string to address: %v", err)
		}
		balance, err := st.WavesBalance(proto.NewRecipientFromAddress(addr))
		if err != nil {
			return errors.Errorf("failed to get balance: %v", err)
		}
		if balance != properBalance {
			return errors.Errorf("balances for address %v differ: %d and %d", addr, properBalance, balance)
		}
	}
	return nil
}
