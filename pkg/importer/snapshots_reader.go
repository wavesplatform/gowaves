package importer

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/wavesplatform/gowaves/pkg/proto"
)

type snapshotsReader struct {
	scheme   proto.Scheme
	r        *bufio.Reader
	pos      int
	closeFun func() error
}

func newSnapshotsReader(scheme proto.Scheme, snapshotsPath string) (*snapshotsReader, error) {
	f, err := os.Open(filepath.Clean(snapshotsPath))
	if err != nil {
		return nil, fmt.Errorf("failed to open snapshots file: %w", err)
	}
	r := bufio.NewReaderSize(f, bufioReaderBuffSize)
	return &snapshotsReader{scheme: scheme, r: r, pos: 0, closeFun: f.Close}, nil
}

func (sr *snapshotsReader) readSize() (uint32, error) {
	const sanityMaxBlockSnapshotSize = 100 * MiB
	var buf [uint32Size]byte
	pos := sr.pos
	n, err := io.ReadFull(sr.r, buf[:])
	if err != nil {
		return 0, fmt.Errorf("failed to read block snapshot size at pos %d: %w", pos, err)
	}
	sr.pos += n
	size := binary.BigEndian.Uint32(buf[:])
	if size > sanityMaxBlockSnapshotSize { // don't check for 0 size because it is valid
		return 0, fmt.Errorf("block snapshot size %d is too big at pos %d", size, pos)
	}
	return size, nil
}

func (sr *snapshotsReader) skip(size uint32) error {
	n, err := sr.r.Discard(int(size))
	if err != nil {
		return fmt.Errorf("failed to skip at pos %d: %w", sr.pos, err)
	}
	sr.pos += n
	return nil
}

func (sr *snapshotsReader) readSnapshot() (*proto.BlockSnapshot, error) {
	size, sErr := sr.readSize()
	if sErr != nil {
		return nil, fmt.Errorf("failed to read snapshot size: %w", sErr)
	}
	pos := sr.pos
	buf := make([]byte, size)
	n, rErr := io.ReadFull(sr.r, buf)
	if rErr != nil {
		return nil, fmt.Errorf("failed to read snapshot at pos %d: %w", pos, rErr)
	}
	sr.pos += n
	snapshot := &proto.BlockSnapshot{}
	if err := snapshot.UnmarshalBinaryImport(buf, sr.scheme); err != nil {
		return nil, fmt.Errorf("failed to unmarshal snapshot at pos %d: %w", pos, err)
	}
	return snapshot, nil
}

func (sr *snapshotsReader) close() error {
	return sr.closeFun()
}
