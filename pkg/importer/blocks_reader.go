package importer

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type blocksReader struct {
	f   *os.File
	pos int64
}

func newBlocksReader(blockchainPath string) (*blocksReader, error) {
	f, err := os.Open(filepath.Clean(blockchainPath))
	if err != nil {
		return nil, fmt.Errorf("failed to open blocks file: %w", err)
	}
	return &blocksReader{f: f, pos: 0}, nil
}

func (br *blocksReader) readSize() (uint32, error) {
	buf := make([]byte, uint32Size)
	n, err := br.f.ReadAt(buf, br.pos)
	if err != nil {
		return 0, fmt.Errorf("failed to read block size: %w", err)
	}
	br.pos += int64(n)
	size := binary.BigEndian.Uint32(buf)
	if size > MaxBlockSize || size == 0 {
		return 0, errors.New("corrupted blockchain file: invalid block size")
	}
	return size, nil
}

func (br *blocksReader) skip(size uint32) {
	br.pos += int64(size)
}

func (br *blocksReader) readBlock(size uint32) ([]byte, error) {
	buf := make([]byte, size)
	n, err := br.f.ReadAt(buf, br.pos)
	if err != nil {
		return nil, fmt.Errorf("failed to read block: %w", err)
	}
	br.pos += int64(n)
	return buf, nil
}

func (br *blocksReader) close() error {
	return br.f.Close()
}
