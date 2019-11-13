package state

// test_util.go - utilities used by unit tests of state and other packages.

import (
	"bufio"
	"encoding/binary"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

var (
	cachedBlocks []proto.Block
)

func getLocalDir() (string, error) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "", errors.Errorf("Unable to find current package file")
	}
	return filepath.Dir(filename), nil
}

func blocksPath(t *testing.T) string {
	dir, err := getLocalDir()
	assert.NoError(t, err, "getLocalDir() failed")
	return filepath.Join(dir, "testdata", "blocks-10000")
}

func readRealBlocks(t *testing.T, blocksPath string, nBlocks int) ([]proto.Block, error) {
	if len(cachedBlocks) >= nBlocks {
		return cachedBlocks[:nBlocks], nil
	}
	f, err := os.Open(blocksPath)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err = f.Close(); err != nil {
			t.Logf("Failed to close blockchain file: %v\n\n", err.Error())
		}
	}()

	sb := make([]byte, 4)
	buf := make([]byte, 2*1024*1024)
	r := bufio.NewReader(f)
	var blocks []proto.Block
	for i := 0; i < nBlocks; i++ {
		if _, err := io.ReadFull(r, sb); err != nil {
			return nil, err
		}
		s := binary.BigEndian.Uint32(sb)
		bb := buf[:s]
		if _, err = io.ReadFull(r, bb); err != nil {
			return nil, err
		}
		var block proto.Block
		if err = block.UnmarshalBinary(bb); err != nil {
			return nil, err
		}
		if !crypto.Verify(block.GenPublicKey, block.BlockSignature, bb[:len(bb)-crypto.SignatureSize]) {
			return nil, errors.Errorf("Block %d has invalid signature", i)
		}
		blocks = append(blocks, block)
	}
	cachedBlocks = blocks
	return blocks, nil
}

// This function is used for testing in other packages.
func ReadMainnetBlocksToHeight(t *testing.T, height proto.Height) []*proto.Block {
	path := blocksPath(t)
	blocks, err := readRealBlocks(t, path, int(height-1))
	assert.NoError(t, err, "readRealBlocks() failed")
	res := make([]*proto.Block, len(blocks))
	for i := range blocks {
		res[i] = &blocks[i]
	}
	return res
}
