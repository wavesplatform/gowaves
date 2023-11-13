package node

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func TestSequentialIteration(t *testing.T) {
	g := &blockGenerator{}
	bs := newBlockSequence(5)
	for i := 0; i < 5; i++ {
		b := g.next()
		ok := bs.pushID(b.BlockID())
		assert.True(t, ok)
	}
}

func TestOverCapacity(t *testing.T) {
	g := &blockGenerator{}
	bs := newBlockSequence(5)
	for i := 0; i < 5; i++ {
		b := g.next()
		ok := bs.pushID(b.BlockID())
		assert.True(t, ok)
	}
	b := g.next()
	ok := bs.pushID(b.BlockID())
	assert.False(t, ok)
}

func TestSimpleRetrieve(t *testing.T) {
	g := &blockGenerator{}
	bs := newBlockSequence(5)
	blocks := make([]*proto.Block, 5)
	for i := 0; i < 5; i++ {
		b := g.next()
		blocks[i] = b
		ok := bs.pushID(b.BlockID())
		assert.True(t, ok)
		ok = bs.putBlock(b)
		assert.True(t, ok)
	}
	assert.ElementsMatch(t, blocks, bs.blocks())
	assert.True(t, bs.full())
}

func TestRetrieve(t *testing.T) {
	g := &blockGenerator{}
	bs := newBlockSequence(5)
	blocks := make([]*proto.Block, 5)
	for i := 0; i < 5; i++ {
		b := g.next()
		blocks[i] = b
		ok := bs.pushID(b.BlockID())
		assert.True(t, ok)
	}
	ok := bs.putBlock(blocks[0])
	require.True(t, ok)
	ok = bs.putBlock(blocks[1])
	require.True(t, ok)
	assert.ElementsMatch(t, blocks[:2], bs.blocks())
	assert.False(t, bs.full())
	ok = bs.putBlock(blocks[2])
	require.True(t, ok)
	ok = bs.putBlock(blocks[3])
	require.True(t, ok)
	ok = bs.putBlock(blocks[4])
	require.True(t, ok)
	assert.ElementsMatch(t, blocks, bs.blocks())
	assert.True(t, bs.full())
}

func TestRetrieveMoreThanAvailable(t *testing.T) {
	g := &blockGenerator{}
	bs := newBlockSequence(5)
	for i := 0; i < 5; i++ {
		b := g.next()
		ok := bs.pushID(b.BlockID())
		assert.True(t, ok)
		if i < 3 {
			ok = bs.putBlock(b)
			assert.True(t, ok)
		}
	}
	assert.Equal(t, 3, len(bs.blocks()))
	assert.False(t, bs.full())
}

type blockGenerator struct {
	i int
}

func (g *blockGenerator) next() *proto.Block {
	g.i++
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(g.i))
	d := crypto.MustFastHash(b)
	id := proto.NewBlockIDFromDigest(d)
	return &proto.Block{
		BlockHeader: proto.BlockHeader{
			Version: proto.ProtobufBlockVersion,
			ID:      id,
		},
	}
}

func TestRelativeComplement(t *testing.T) {
	bg := blockGenerator{}
	id1 := bg.next().BlockID()
	id2 := bg.next().BlockID()
	id3 := bg.next().BlockID()
	id4 := bg.next().BlockID()
	id5 := bg.next().BlockID()
	for _, test := range []struct {
		first  []proto.BlockID
		second []proto.BlockID
		res    []proto.BlockID
		ok     bool
	}{
		{[]proto.BlockID{id1, id2, id3}, []proto.BlockID{id3, id4, id5}, []proto.BlockID{id4, id5}, true},
		{[]proto.BlockID{id1, id2, id3}, []proto.BlockID{id2, id3, id4, id5}, []proto.BlockID{id4, id5}, true},
		{[]proto.BlockID{id1, id2, id3}, []proto.BlockID{id4, id5}, []proto.BlockID{id4, id5}, false},
		{[]proto.BlockID{id1, id2, id3}, []proto.BlockID{id1, id2, id3}, nil, true},
		{[]proto.BlockID{id1, id2, id3}, []proto.BlockID{id3, id5, id4}, []proto.BlockID{id5, id4}, true},
	} {
		r, ok := relativeCompliment(test.first, test.second)
		assert.Equal(t, test.ok, ok)
		assert.ElementsMatch(t, test.res, r)
	}
}
