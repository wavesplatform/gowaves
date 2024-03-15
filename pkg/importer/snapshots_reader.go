package importer

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/wavesplatform/gowaves/pkg/proto"
)

type snapshotsReader struct {
	scheme proto.Scheme
	f      *os.File
	pos    int64
}

func newSnapshotsReader(scheme proto.Scheme, snapshotsPath string) (*snapshotsReader, error) {
	f, err := os.Open(filepath.Clean(snapshotsPath))
	if err != nil {
		return nil, fmt.Errorf("failed to open snapshots file: %w", err)
	}
	return &snapshotsReader{scheme: scheme, f: f, pos: 0}, nil
}

func (sr *snapshotsReader) readSize() (uint32, error) {
	buf := make([]byte, uint32Size)
	n, err := sr.f.ReadAt(buf, sr.pos)
	if err != nil {
		return 0, fmt.Errorf("failed to read block snapshot size: %w", err)
	}
	sr.pos += int64(n)
	size := binary.BigEndian.Uint32(buf)
	if size == 0 {
		return 0, errors.New("corrupted snapshots file: invalid snapshot size")
	}
	return size, nil
}

func (sr *snapshotsReader) skip(size uint32) {
	sr.pos += int64(size)
}

func (sr *snapshotsReader) readSnapshot() (*proto.BlockSnapshot, error) {
	size, sErr := sr.readSize()
	if sErr != nil {
		return nil, sErr
	}
	buf := make([]byte, size)
	n, rErr := sr.f.ReadAt(buf, sr.pos)
	if rErr != nil {
		return nil, fmt.Errorf("failed to read snapshot: %w", rErr)
	}
	sr.pos += int64(n)
	snapshot := &proto.BlockSnapshot{}
	if err := snapshot.UnmarshalBinaryImport(buf, sr.scheme); err != nil {
		return nil, fmt.Errorf("failed to unmarshal snapshot: %w", err)
	}
	return snapshot, nil
}

func (sr *snapshotsReader) close() error {
	return sr.f.Close()
}
