package main

import (
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/wavesplatform/gowaves/pkg/logging"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	snapshotsByteSize = 4
)

func parseSnapshots(nBlocks int, snapshotsBody *os.File, scheme proto.Scheme) []proto.BlockSnapshot {
	snapshotsSizeBytes := make([]byte, snapshotsByteSize)
	readPos := int64(0)
	var blocksSnapshots []proto.BlockSnapshot
	for height := uint64(1); height <= uint64(nBlocks); height++ {
		if _, readBerr := snapshotsBody.ReadAt(snapshotsSizeBytes, readPos); readBerr != nil {
			zap.S().Fatalf("failed to read the snapshots size in block %v", readBerr)
		}
		snapshotsSize := binary.BigEndian.Uint32(snapshotsSizeBytes)
		if snapshotsSize == 0 {
			readPos += snapshotsByteSize
			continue
		}
		if snapshotsSize != 0 {
			snapshotsInBlock := proto.BlockSnapshot{}
			snapshots := make([]byte, snapshotsSize+snapshotsByteSize) // []{snapshot, size} + 4 bytes = size of all snapshots
			if _, readRrr := snapshotsBody.ReadAt(snapshots, readPos); readRrr != nil {
				zap.S().Fatalf("failed to read the snapshots in block %v", readRrr)
			}
			unmrshlErr := snapshotsInBlock.UnmarshalBinaryImport(snapshots, scheme)
			if unmrshlErr != nil {
				zap.S().Fatalf("failed to unmarshal snapshots in block %v", unmrshlErr)
			}
			blocksSnapshots = append(blocksSnapshots, snapshotsInBlock)
			readPos += int64(snapshotsSize) + snapshotsByteSize
		}
	}
	return blocksSnapshots
}

func main() {
	const (
		defaultBlocksNumber = 1000
	)
	var (
		logLevel = zap.LevelFlag("log-level", zapcore.InfoLevel,
			"Logging level. Supported levels: DEBUG, INFO, WARN, ERROR, FATAL. Default logging level INFO.")
		blockchainType = flag.String("blockchain-type", "mainnet",
			"Blockchain type. Allowed values: mainnet/testnet/stagenet/custom. Default is 'mainnet'.")
		snapshotsPath = flag.String("snapshots-path", "", "Path to binary blockchain file.")
		nBlocks       = flag.Int("blocks-number", defaultBlocksNumber, "Number of blocks to import.")
	)
	flag.Parse()

	logger := logging.SetupSimpleLogger(*logLevel)
	defer func() {
		err := logger.Sync()
		if err != nil && errors.Is(err, os.ErrInvalid) {
			panic(fmt.Sprintf("failed to close logging subsystem: %v\n", err))
		}
	}()
	if *snapshotsPath == "" {
		zap.S().Fatalf("You must specify snapshots-path option.")
	}

	ss, err := settings.BlockchainSettingsByTypeName(*blockchainType)
	if err != nil {
		zap.S().Fatalf("failed to load blockchain settings: %v", err)
	}

	snapshotsBody, err := os.Open(*snapshotsPath)
	if err != nil {
		zap.S().Fatalf("failed to open snapshots file, %v", err)
	}
	blocksSnapshots := parseSnapshots(*nBlocks, snapshotsBody, ss.AddressSchemeCharacter)

	zap.S().Info(blocksSnapshots[0])
}
