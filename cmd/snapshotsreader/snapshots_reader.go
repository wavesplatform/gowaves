package main

import (
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"github.com/wavesplatform/gowaves/pkg/logging"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"log"
	"os"

	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

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

	snapshotsSizeBytes := make([]byte, 4)
	readPos := int64(0)
	var blocksSnapshots []proto.BlockSnapshot
	for height := uint64(1); height <= uint64(*nBlocks); height++ {
		if _, err := snapshotsBody.ReadAt(snapshotsSizeBytes, readPos); err != nil {
			log.Fatal(err)
		}
		snapshotsSize := binary.BigEndian.Uint32(snapshotsSizeBytes)
		if snapshotsSize == 0 {
			readPos += 4
			continue
		}
		if snapshotsSize != 0 {
			fmt.Println()
			snapshotsInBlock := proto.BlockSnapshot{}
			snapshots := make([]byte, snapshotsSize+4) // []{snapshot, size} + 4 bytes = size of all snapshots
			if _, err := snapshotsBody.ReadAt(snapshots, readPos); err != nil {
				log.Fatal(err)
			}
			err := snapshotsInBlock.UnmarshalBinaryImport(snapshots, ss.AddressSchemeCharacter)
			if err != nil {
				return
			}
			blocksSnapshots = append(blocksSnapshots, snapshotsInBlock)
			readPos += int64(snapshotsSize) + 4
		}
	}
}
