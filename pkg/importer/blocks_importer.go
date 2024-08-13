package importer

import (
	"context"
	"fmt"
	"time"

	"github.com/wavesplatform/gowaves/pkg/proto"
)

type BlocksImporter struct {
	scheme proto.Scheme
	st     State

	br  *blocksReader
	reg *speedRegulator

	h uint64
}

func NewBlocksImporter(scheme proto.Scheme, st State, blocksPath string) (*BlocksImporter, error) {
	br, err := newBlocksReader(blocksPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create blocks importer: %w", err)
	}
	return &BlocksImporter{scheme: scheme, st: st, br: br, reg: newSpeedRegulator()}, nil
}

func (imp *BlocksImporter) SkipToHeight(ctx context.Context, height proto.Height) error {
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
		if skipErr := imp.br.skip(size); skipErr != nil {
			return fmt.Errorf("failed to skip to height %d: %w", height, skipErr)
		}
		imp.h++
	}
}

func (imp *BlocksImporter) Import(ctx context.Context, number uint64) error {
	var blocks [MaxBlocksBatchSize][]byte
	index := uint64(0)
	for height := imp.h; height <= number; height++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		size, err := imp.br.readSize()
		if err != nil {
			return fmt.Errorf("failed to import: %w", err)
		}
		imp.reg.updateTotalSize(size)
		block, err := imp.br.readBlock(size)
		if err != nil {
			return fmt.Errorf("failed to import: %w", err)
		}
		blocks[index] = block
		index++
		if imp.reg.incomplete() && (index != MaxBlocksBatchSize) && (height != number) {
			continue
		}
		start := time.Now()
		if abErr := imp.st.AddBlocks(blocks[:index]); abErr != nil {
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

func (imp *BlocksImporter) Close() error {
	return imp.br.close()
}
