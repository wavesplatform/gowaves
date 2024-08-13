package importer

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type blocksReader struct {
	r        *bufio.Reader
	pos      int
	closeFun func() error
}

func newBlocksReader(blockchainPath string) (*blocksReader, error) {
	f, err := os.Open(filepath.Clean(blockchainPath))
	if err != nil {
		return nil, fmt.Errorf("failed to open blocks file: %w", err)
	}
	r := bufio.NewReaderSize(f, bufioReaderBuffSize)
	return &blocksReader{r: r, pos: 0, closeFun: f.Close}, nil
}

func (br *blocksReader) readSize() (uint32, error) {
	var buf [uint32Size]byte
	pos := br.pos
	n, err := io.ReadFull(br.r, buf[:])
	if err != nil {
		return 0, fmt.Errorf("failed to read block size at pos %d: %w", pos, err)
	}
	br.pos += n
	size := binary.BigEndian.Uint32(buf[:])
	if size > MaxBlockSize || size == 0 {
		return 0, fmt.Errorf("corrupted blockchain file: invalid block size %d at pos %d", size, pos)
	}
	return size, nil
}

func (br *blocksReader) readBlock(size uint32) ([]byte, error) {
	buf := make([]byte, size)
	n, err := io.ReadFull(br.r, buf)
	if err != nil {
		return nil, fmt.Errorf("failed to read block at pos %d: %w", br.pos, err)
	}
	br.pos += n
	return buf, nil
}

func (br *blocksReader) skip(size uint32) error {
	n, err := br.r.Discard(int(size))
	if err != nil {
		return fmt.Errorf("failed to skip at pos %d: %w", br.pos, err)
	}
	br.pos += n
	return nil
}

func (br *blocksReader) close() error {
	return br.closeFun()
}
