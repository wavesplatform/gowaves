package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func TestSkipRepeatedSignatures(t *testing.T) {
	s1 := crypto.Signature{}
	s1[0] = 1
	id1 := proto.NewBlockIDFromSignature(s1)
	s2 := crypto.Signature{}
	s2[0] = 2
	id2 := proto.NewBlockIDFromSignature(s2)
	s3 := crypto.Signature{}
	s3[0] = 3
	id3 := proto.NewBlockIDFromSignature(s3)
	pending := []proto.BlockID{id1}
	incoming := []proto.BlockID{id1, id1, id2, id3}
	unheard := skip(incoming, pending)
	assert.ElementsMatch(t, []proto.BlockID{id2, id3}, unheard)

	pending = []proto.BlockID{id1, id2}
	incoming = []proto.BlockID{id1, id1, id2, id3}
	unheard = skip(incoming, pending)
	assert.ElementsMatch(t, []proto.BlockID{id3}, unheard)
}
