package importer

import (
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/wavesplatform/gowaves/pkg/proto"
)

const (
	KiB = 1024
	MiB = 1024 * KiB

	initTotalBatchSize = 5 * MiB
	sizeAdjustment     = 1 * MiB
	uint32Size         = 4

	MaxTotalBatchSize  = 20 * MiB
	MaxBlocksBatchSize = 50000
	MaxBlockSize       = 2 * MiB
)

var errNoop = errors.New("noop")

type State interface {
	AddBlocks(blocks [][]byte) error
	AddBlocksWithSnapshots(blocks [][]byte, snapshots []*proto.BlockSnapshot) error
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

type blockReader struct {
	f   *os.File
	pos int64
}

func newBlockReader(blockchainPath string) (*blockReader, error) {
	f, err := os.Open(blockchainPath)
	if err != nil {
		return nil, errors.Errorf("failed to open blockchain file: %v", err)
	}
	return &blockReader{f: f, pos: 0}, nil
}

func (br *blockReader) readSize() (uint32, error) {
	buf := make([]byte, uint32Size)
	n, err := br.f.ReadAt(buf, br.pos)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to read block size")
	}
	br.pos += int64(n)
	size := binary.BigEndian.Uint32(buf)
	if size > MaxBlockSize || size == 0 {
		return 0, errors.New("corrupted blockchain file: invalid block size")
	}
	return size, nil
}

func (br *blockReader) skip(size uint32) {
	br.pos += int64(size)
}

func (br *blockReader) readBlock(size uint32) ([]byte, error) {
	buf := make([]byte, size)
	n, err := br.f.ReadAt(buf, br.pos)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read block")
	}
	br.pos += int64(n)
	return buf, nil
}

func (br *blockReader) close() {
	if err := br.f.Close(); err != nil {
		zap.S().Fatalf("Failed to close blockchain file: %v", err)
	}
}

type snapshotReader interface {
	readSize() (uint32, error)
	readSnapshot() (*proto.BlockSnapshot, error)
	close()
}

func newSnapshotReader(scheme proto.Scheme, snapshotsPath string, lightMode bool) (snapshotReader, error) {
	if lightMode {
		f, err := os.Open(snapshotsPath)
		if err != nil {
			return nil, errors.Errorf("failed to open snapshots file: %v", err)
		}
		return &realSnapshotReader{scheme: scheme, f: f, pos: 0}, nil
	}
	return &noopSnapshotReader{}, nil
}

type noopSnapshotReader struct{}

func (sr *noopSnapshotReader) readSize() (uint32, error) {
	return 0, nil
}

func (sr *noopSnapshotReader) readSnapshot() (*proto.BlockSnapshot, error) {
	return nil, errNoop
}

func (sr *noopSnapshotReader) close() {}

type realSnapshotReader struct {
	scheme proto.Scheme
	f      *os.File
	pos    int64
}

func (sr *realSnapshotReader) readSize() (uint32, error) {
	buf := make([]byte, uint32Size)
	n, err := sr.f.ReadAt(buf, sr.pos)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to read block snapshot size")
	}
	sr.pos += int64(n)
	size := binary.BigEndian.Uint32(buf)
	if size == 0 {
		return 0, errors.New("corrupted snapshots file: invalid snapshot size")
	}
	return size, nil
}

func (sr *realSnapshotReader) readSnapshot() (*proto.BlockSnapshot, error) {
	size, sErr := sr.readSize()
	if sErr != nil {
		return nil, sErr
	}
	buf := make([]byte, size)
	n, rErr := sr.f.ReadAt(buf, sr.pos)
	if rErr != nil {
		return nil, errors.Wrapf(rErr, "failed to read snapshot")
	}
	sr.pos += int64(n)
	snapshot := &proto.BlockSnapshot{}
	if err := snapshot.UnmarshalBinaryImport(buf, sr.scheme); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal snapshot")
	}
	return snapshot, nil
}

func (sr *realSnapshotReader) close() {
	if err := sr.f.Close(); err != nil {
		zap.S().Fatalf("Failed to close snapshots file: %v", err)
	}
}

type speedRegulator struct {
	prevSpeed  float64
	speed      float64
	maxSize    int
	increasing bool
	totalSize  int
}

func newSpeedRegulator() *speedRegulator {
	return &speedRegulator{maxSize: initTotalBatchSize, increasing: true}
}

func (r *speedRegulator) updateTotalSize(size uint32) {
	r.totalSize += int(size)
}

func (r *speedRegulator) incomplete() bool {
	return r.totalSize < r.maxSize
}

func (r *speedRegulator) calculateSpeed(start time.Time) {
	elapsed := time.Since(start)
	r.speed = float64(r.totalSize) / float64(elapsed)
	r.maxSize, r.increasing = calculateNextMaxSizeAndDirection(r.maxSize, r.speed, r.prevSpeed, r.increasing)
	r.prevSpeed = r.speed
	r.totalSize = 0
}

// ApplyFromFile reads blocks from blockchainPath, applying them from height startHeight and until nBlocks+1.
// Setting optimize to true speeds up the import, but it is only safe when importing blockchain from scratch
// when no rollbacks are possible at all.
func ApplyFromFile(
	scheme proto.Scheme, st State, blockchainPath string, snapshotsPath string, nBlocks, startHeight uint64,
	isLightMode bool,
) error {
	br, brErr := newBlockReader(blockchainPath)
	if brErr != nil {
		return brErr
	}
	defer br.close()
	sr, srErr := newSnapshotReader(scheme, snapshotsPath, isLightMode)
	if srErr != nil {
		return srErr
	}
	defer sr.close()
	reg := newSpeedRegulator()

	var blocks [MaxBlocksBatchSize][]byte
	var blockSnapshots [MaxBlocksBatchSize]*proto.BlockSnapshot
	blocksIndex := 0
	for height := uint64(1); height <= nBlocks; height++ {
		// reading snapshots
		snapshot, err := sr.readSnapshot()
		if err != nil && !errors.Is(err, errNoop) {
			return err
		}
		blockSnapshots[blocksIndex] = snapshot

		size, sErr := br.readSize()
		if sErr != nil {
			return sErr
		}
		reg.updateTotalSize(size)
		if height < startHeight {
			br.skip(size)
			continue
		}
		block, rErr := br.readBlock(size)
		if rErr != nil {
			return rErr
		}
		blocks[blocksIndex] = block
		blocksIndex++
		if reg.incomplete() && (blocksIndex != MaxBlocksBatchSize) && (height != nBlocks) {
			continue
		}
		start := time.Now()
		if abErr := st.AddBlocksWithSnapshots(blocks[:blocksIndex], blockSnapshots[:blocksIndex]); abErr != nil {
			return abErr
		}
		reg.calculateSpeed(start)
		blocksIndex = 0
		if pErr := maybePersistTxs(st); pErr != nil {
			return pErr
		}
	}
	return nil
}

func CheckBalances(st State, balancesPath string) error {
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
