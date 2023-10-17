package state

import (
	"bytes"
	"math/rand"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/libs/ntptime"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

const (
	blocksBatchSize    = 500
	blockInFutureDrift = 100
)

func applyBlocks(t *testing.T, blocks []proto.Block, st State, scheme proto.Scheme) error {
	var blocksBatch [blocksBatchSize][]byte
	blocksIndex := 0
	for height := uint64(1); height <= blocksNumber; height++ {
		block := blocks[height-1]
		blockBytes, err := block.MarshalBinary(scheme)
		if err != nil {
			t.Fatalf("block.MarshalBinary(): %v\n", err)
		}
		blocksBatch[blocksIndex] = blockBytes
		blocksIndex++
		if blocksIndex != blocksBatchSize && height != blocksNumber {
			continue
		}
		if err := st.AddBlocks(blocksBatch[:blocksIndex]); err != nil {
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
	invalidSig := bytes.Repeat([]byte{0xff}, crypto.DigestSize)
	block.GenSignature = invalidSig
	return nil
}

func spoilBaseTarget(block *proto.Block) {
	block.BaseTarget = block.BaseTarget - 1
}

func spoilBlockVersion(block *proto.Block) {
	block.Version = proto.NgBlockVersion
}

func stateParams() StateParams {
	s := DefaultStorageParams()
	s.DbParams.Store = keyvalue.NoOpStore{}
	return StateParams{
		StorageParams: s,
		ValidationParams: ValidationParams{
			VerificationGoroutinesNum: runtime.NumCPU() * 2,
			Time:                      ntptime.Stub{},
		},
	}
}

func TestHeadersValidation(t *testing.T) {
	blocks, err := readBlocksFromTestPath(blocksNumber)
	if err != nil {
		t.Fatalf("Can not read blocks from blockchain file: %v\n", err)
	}
	var (
		sets   = settings.MainNetSettings
		scheme = sets.AddressSchemeCharacter
		st     = newTestState(t, true, stateParams(), sets)
	)

	err = applyBlocks(t, blocks, st, scheme)
	assert.NoError(t, err, "failed to apply correct blocks")
	err = st.RollbackToHeight(1)
	assert.NoError(t, err, "failed to rollback state")

	randN := rand.Int() % len(blocks)
	prev := blocks[randN]
	spoilTimestampFuture(&blocks[randN])
	err = applyBlocks(t, blocks, st, scheme)
	assert.Error(t, err, "did not fail with timestamp from future")
	blocks[randN] = prev
	err = st.RollbackToHeight(1)
	assert.NoError(t, err, "failed to rollback state")

	randN = (rand.Int() % (len(blocks) - 1)) + 1
	prev = blocks[randN]
	prevTimestamp := blocks[randN-1].Timestamp
	spoilDelay(&blocks[randN], prevTimestamp)
	err = applyBlocks(t, blocks, st, scheme)
	assert.Error(t, err, "did not fail with incorrect block delay")
	blocks[randN] = prev
	err = st.RollbackToHeight(1)
	assert.NoError(t, err, "failed to rollback state")

	randN = rand.Int() % len(blocks)
	prev = blocks[randN]
	if err := spoilGenSignature(&blocks[randN]); err != nil {
		t.Fatalf("spoilGenSignature(): %v\n", err)
	}
	err = applyBlocks(t, blocks, st, scheme)
	assert.Error(t, err, "did not fail with invalid geneator signature")
	blocks[randN] = prev
	err = st.RollbackToHeight(1)
	assert.NoError(t, err, "failed to rollback state")

	randN = rand.Int() % len(blocks)
	prev = blocks[randN]
	spoilBaseTarget(&blocks[randN])
	err = applyBlocks(t, blocks, st, scheme)
	assert.Error(t, err, "did not fail with incorrect base target")
	blocks[randN] = prev
	err = st.RollbackToHeight(1)
	assert.NoError(t, err, "failed to rollback state")

	randN = rand.Int() % len(blocks)
	prev = blocks[randN]
	spoilBlockVersion(&blocks[randN])
	err = applyBlocks(t, blocks, st, scheme)
	assert.Error(t, err, "did not fail with wrong block version")
	blocks[randN] = prev
	err = st.RollbackToHeight(1)
	assert.NoError(t, err, "failed to rollback state")
}
