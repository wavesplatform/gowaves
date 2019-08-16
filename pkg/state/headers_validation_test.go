package state

import (
	"bytes"
	"io/ioutil"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

const (
	blocksBatchSize    = 500
	blockInFutureDrift = 100
)

func applyBlocks(t *testing.T, blocks []proto.Block, st State) error {
	var blocksBatch [blocksBatchSize][]byte
	blocksIndex := 0
	for height := uint64(1); height <= blocksNumber; height++ {
		block := blocks[height-1]
		blockBytes, err := block.MarshalBinary()
		if err != nil {
			t.Fatalf("block.MarshalBinary(): %v\n", err)
		}
		blocksBatch[blocksIndex] = blockBytes
		blocksIndex++
		if blocksIndex != blocksBatchSize && height != blocksNumber {
			continue
		}
		if err := st.AddOldBlocks(blocksBatch[:blocksIndex]); err != nil {
			return err
		}
		blocksIndex = 0
	}
	return nil
}

func spoilTimestampFuture(block *proto.Block) {
	block.Timestamp = uint64(time.Now().UnixNano()/1000 + blockInFutureDrift*2)
}

func spoilDelay(block *proto.Block, prevTimestamp uint64) {
	// 0 delay.
	block.Timestamp = prevTimestamp
}

func spoilGenSignature(block *proto.Block) error {
	invalidSig, err := crypto.NewDigestFromBytes(bytes.Repeat([]byte{0xff}, crypto.DigestSize))
	if err != nil {
		return err
	}
	block.GenSignature = invalidSig
	return nil
}

func spoilBaseTarget(block *proto.Block) {
	block.BaseTarget = block.BaseTarget - 1
}

func spoilBlockVersion(block *proto.Block) {
	block.Version = proto.NgBlockVersion
}

func TestHeadersValidation(t *testing.T) {
	blocks, err := readRealBlocks(t, blocksPath(t), blocksNumber)
	if err != nil {
		t.Fatalf("Can not read blocks from blockchain file: %v\n", err)
	}
	dataDir, err := ioutil.TempDir(os.TempDir(), "dataDir")
	if err != nil {
		t.Fatalf("Failed to create temp dir for data: %v\n", err)
	}
	st, err := NewState(dataDir, DefaultStateParams(), settings.MainNetSettings)
	if err != nil {
		t.Fatalf("NewState(): %v\n", err)
	}

	defer func() {
		if err := st.Close(); err != nil {
			t.Fatalf("Failed to close state: %v\n", err)
		}
		if err := os.RemoveAll(dataDir); err != nil {
			t.Fatalf("Failed to clean data dirs: %v\n", err)
		}
	}()

	err = applyBlocks(t, blocks, st)
	assert.NoError(t, err, "failed to apply correct blocks")
	err = st.RollbackToHeight(1)
	assert.NoError(t, err, "failed to rollback state")

	randN := rand.Int() % len(blocks)
	prev := blocks[randN]
	spoilTimestampFuture(&blocks[randN])
	err = applyBlocks(t, blocks, st)
	assert.Error(t, err, "did not fail with timestamp from future")
	blocks[randN] = prev
	err = st.RollbackToHeight(1)
	assert.NoError(t, err, "failed to rollback state")

	randN = (rand.Int() % (len(blocks) - 1)) + 1
	prev = blocks[randN]
	prevTimestamp := blocks[randN-1].Timestamp
	spoilDelay(&blocks[randN], prevTimestamp)
	err = applyBlocks(t, blocks, st)
	assert.Error(t, err, "did not fail with incorrect block delay")
	blocks[randN] = prev
	err = st.RollbackToHeight(1)
	assert.NoError(t, err, "failed to rollback state")

	randN = rand.Int() % len(blocks)
	prev = blocks[randN]
	if err := spoilGenSignature(&blocks[randN]); err != nil {
		t.Fatalf("spoilGenSignature(): %v\n", err)
	}
	err = applyBlocks(t, blocks, st)
	assert.Error(t, err, "did not fail with invalid geneator signature")
	blocks[randN] = prev
	err = st.RollbackToHeight(1)
	assert.NoError(t, err, "failed to rollback state")

	randN = rand.Int() % len(blocks)
	prev = blocks[randN]
	spoilBaseTarget(&blocks[randN])
	err = applyBlocks(t, blocks, st)
	assert.Error(t, err, "did not fail with incorrect base target")
	blocks[randN] = prev
	err = st.RollbackToHeight(1)
	assert.NoError(t, err, "failed to rollback state")

	randN = rand.Int() % len(blocks)
	prev = blocks[randN]
	spoilBlockVersion(&blocks[randN])
	err = applyBlocks(t, blocks, st)
	assert.Error(t, err, "did not fail with wrong block version")
	blocks[randN] = prev
	err = st.RollbackToHeight(1)
	assert.NoError(t, err, "failed to rollback state")
}
