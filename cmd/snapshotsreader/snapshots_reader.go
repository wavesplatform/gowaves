package main

import (
	"bufio"
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/wavesplatform/gowaves/pkg/logging"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

const (
	snapshotsByteSize = 4
)

type SnapshotAtHeight struct {
	Height        proto.Height
	BlockSnapshot proto.BlockSnapshot
}

func parseSnapshots(start, end uint64, snapshotsBody io.Reader, scheme proto.Scheme) []SnapshotAtHeight {
	var buf []byte
	snapshotsSizeBytes := make([]byte, snapshotsByteSize)
	var blocksSnapshots []SnapshotAtHeight
	for height := uint64(2); height < end; height++ {
		if _, readBerr := io.ReadFull(snapshotsBody, snapshotsSizeBytes); readBerr != nil {
			zap.S().Fatalf("failed to read the snapshots size in block: %v", readBerr)
		}
		snapshotsSize := binary.BigEndian.Uint32(snapshotsSizeBytes)
		if snapshotsSize == 0 { // add empty block snapshot
			if height >= start {
				blocksSnapshots = append(blocksSnapshots, SnapshotAtHeight{
					Height:        height,
					BlockSnapshot: proto.BlockSnapshot{},
				})
			}
			continue
		}

		if cap(buf) < int(snapshotsSize) {
			buf = make([]byte, snapshotsSize)
		}
		buf = buf[:snapshotsSize]

		if _, readRrr := io.ReadFull(snapshotsBody, buf); readRrr != nil {
			zap.S().Fatalf("failed to read the snapshots in block: %v", readRrr)
		}
		if height < start {
			continue
		}

		snapshotsInBlock := proto.BlockSnapshot{}
		unmrshlErr := snapshotsInBlock.UnmarshalBinaryImport(buf, scheme)
		if unmrshlErr != nil {
			zap.S().Fatalf("failed to unmarshal snapshots in block: %v", unmrshlErr)
		}
		blocksSnapshots = append(blocksSnapshots, SnapshotAtHeight{
			Height:        height,
			BlockSnapshot: snapshotsInBlock,
		})
	}
	return blocksSnapshots
}

func main() {
	var (
		logLevel = zap.LevelFlag("log-level", zapcore.InfoLevel,
			"Logging level. Supported levels: DEBUG, INFO, WARN, ERROR, FATAL. Default logging level INFO.")
		blockchainType = flag.String("blockchain-type", "mainnet",
			"Blockchain type. Allowed values: mainnet/testnet/stagenet. Default is 'mainnet'.")
		snapshotsPath = flag.String("snapshots-path", "", "Path to binary blockchain file.")
		blocksStart   = flag.Uint64("blocks-start", 0,
			"Start block number. Should be greater than 1, because the snapshots file doesn't include genesis.")
		nBlocks = flag.Uint64("blocks-number", 1, "Number of blocks to read since 'blocks-start'.")
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
	if *blocksStart < 2 {
		zap.S().Fatalf("'blocks-start' must be greater than 1.")
	}

	ss, err := settings.BlockchainSettingsByTypeName(*blockchainType)
	if err != nil {
		zap.S().Fatalf("failed to load blockchain settings: %v", err)
	}

	snapshotsBody, err := os.Open(*snapshotsPath)
	if err != nil {
		zap.S().Fatalf("failed to open snapshots file, %v", err)
	}
	defer func(snapshotsBody *os.File) {
		if clErr := snapshotsBody.Close(); clErr != nil {
			zap.S().Fatalf("failed to close snapshots file, %v", clErr)
		}
	}(snapshotsBody)
	const MB = 1 << 20
	var (
		start = *blocksStart
		end   = start + *nBlocks
	)
	blocksSnapshots := parseSnapshots(start, end, bufio.NewReaderSize(snapshotsBody, MB), ss.AddressSchemeCharacter)

	zap.S().Infof("%+v", blocksSnapshots)
}
