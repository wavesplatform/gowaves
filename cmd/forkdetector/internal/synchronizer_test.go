package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

func TestSkipRepeatedSignatures(t *testing.T) {
	s1 := crypto.Signature{}
	s1[0] = 1
	s2 := crypto.Signature{}
	s2[0] = 2
	s3 := crypto.Signature{}
	s3[0] = 3
	pending := []crypto.Signature{s1}
	incoming := []crypto.Signature{s1, s1, s2, s3}
	unheard := skip(incoming, pending)
	assert.ElementsMatch(t, []crypto.Signature{s2, s3}, unheard)

	pending = []crypto.Signature{s1, s2}
	incoming = []crypto.Signature{s1, s1, s2, s3}
	unheard = skip(incoming, pending)
	assert.ElementsMatch(t, []crypto.Signature{s3}, unheard)
}
