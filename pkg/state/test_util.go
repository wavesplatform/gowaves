package state

// test_util.go - utilities used by unit tests of state and other packages.

import (
	"bufio"
	"encoding/binary"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
)

var (
	cachedBlocks []proto.Block
)

func init() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	zap.ReplaceGlobals(logger)
}

func getLocalDir() (string, error) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "", errors.Errorf("unable to find current package file")
	}
	return filepath.Dir(filename), nil
}

func blocksPath() (string, error) {
	dir, err := getLocalDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "testdata", "blocks-10000"), nil
}

// readRealBlocks reads blocks. This function MUST be used ONLY for tests.
func readRealBlocks(blocksPath string, nBlocks int) ([]proto.Block, error) {
	if len(cachedBlocks) >= nBlocks {
		return cachedBlocks[:nBlocks], nil
	}
	f, err := os.Open(filepath.Clean(blocksPath))
	if err != nil {
		return nil, err
	}

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
		if err := block.UnmarshalBinary(bb, proto.MainNetScheme); err != nil {
			return nil, err
		}
		if !crypto.Verify(block.GeneratorPublicKey, block.BlockSignature, bb[:len(bb)-crypto.SignatureSize]) {
			return nil, errors.Errorf("block %d has invalid signature", i)
		}
		blocks = append(blocks, block)
	}
	cachedBlocks = blocks

	if err := f.Close(); err != nil {
		return nil, errors.Errorf("failed to close blockchain file: %v", err.Error())
	}

	return blocks, nil
}

func readBlocksFromTestPath(blocksNum int) ([]proto.Block, error) {
	path, err := blocksPath()
	if err != nil {
		return nil, err
	}
	blocks, err := readRealBlocks(path, blocksNum)
	if err != nil {
		return nil, err
	}
	return blocks, nil
}

// This function is used for testing in other packages.
// This is more convenient, because blocks are stored in state's subdir.
func ReadMainnetBlocksToHeight(height proto.Height) ([]*proto.Block, error) {
	blocks, err := readBlocksFromTestPath(int(height - 1))
	if err != nil {
		return nil, err
	}
	res := make([]*proto.Block, len(blocks))
	for i := range blocks {
		res[i] = &blocks[i]
	}
	return res, nil
}
