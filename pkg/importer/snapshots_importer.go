package importer

import (
	"context"
	"fmt"
	"time"

	"github.com/wavesplatform/gowaves/pkg/proto"
)

type SnapshotsImporter struct {
	scheme proto.Scheme
	st     State

	br  *blocksReader
	sr  *snapshotsReader
	reg *speedRegulator

	h uint64
}

func NewSnapshotsImporter(scheme proto.Scheme, st State, blocksPath, snapshotsPath string) (*SnapshotsImporter, error) {
	br, err := newBlocksReader(blocksPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create snapshots importer: %w", err)
	}
	sr, err := newSnapshotsReader(scheme, snapshotsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create snapshots importer: %w", err)
	}
	return &SnapshotsImporter{scheme: scheme, st: st, br: br, sr: sr, reg: newSpeedRegulator()}, nil
}

func (imp *SnapshotsImporter) SkipToHeight(ctx context.Context, height proto.Height) error {
	imp.h = uint64(1)
	if height < imp.h {
		return fmt.Errorf("invalid initial height: %d", height)
	}
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if imp.h == height {
			return nil
		}
		size, err := imp.br.readSize()
		if err != nil {
			return fmt.Errorf("failed to skip to height %d: %w", height, err)
		}
		imp.reg.updateTotalSize(size)
		imp.br.skip(size)
		size, err = imp.sr.readSize()
		if err != nil {
			return fmt.Errorf("failed to skip to height %d: %w", height, err)
		}
		imp.sr.skip(size)
		imp.h++
	}
}

func (imp *SnapshotsImporter) Import(ctx context.Context, number uint64) error {
	var blocks [MaxBlocksBatchSize][]byte
	var snapshots [MaxBlocksBatchSize]*proto.BlockSnapshot
	index := 0
	for count := imp.h; count <= number; count++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		// reading snapshots
		snapshot, err := imp.sr.readSnapshot()
		if err != nil {
			return err
		}
		snapshots[index] = snapshot

		size, sErr := imp.br.readSize()
		if sErr != nil {
			return sErr
		}
		imp.reg.updateTotalSize(size)
		block, rErr := imp.br.readBlock(size)
		if rErr != nil {
			return rErr
		}
		blocks[index] = block
		index++
		if imp.reg.incomplete() && (index != MaxBlocksBatchSize) && (count != number) {
			continue
		}
		start := time.Now()
		if abErr := imp.st.AddBlocksWithSnapshots(blocks[:index], snapshots[:index]); abErr != nil {
			return abErr
		}
		imp.reg.calculateSpeed(start)
		index = 0
		if pErr := maybePersistTxs(imp.st); pErr != nil {
			return pErr
		}
	}
	return nil
}

func (imp *SnapshotsImporter) Close() error {
	if err := imp.sr.close(); err != nil {
		return err
	}
	return imp.br.close()
}
