package state

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

// referenceBlockIDKeyBytes builds a block ID key the way it was built before blockIDKeyBytes.
func referenceBlockIDKeyBytes(prefix byte, blockID proto.BlockID) []byte {
	idBytes := blockID.Bytes()
	buf := make([]byte, 1+len(idBytes))
	buf[0] = prefix
	copy(buf[1:], idBytes)
	return buf
}

func TestBlockIDKeysBytes(t *testing.T) {
	var dig crypto.Digest
	for i := range dig {
		dig[i] = byte(i + 7)
	}
	ids := map[string]proto.BlockID{
		"signature":     genBlockId(0x5a),
		"digest":        proto.NewBlockIDFromDigest(dig),
		"uninitialized": {},
	}
	for name, id := range ids {
		t.Run(name, func(t *testing.T) {
			k1 := blockIdToNumKey{id}
			assert.Equal(t, referenceBlockIDKeyBytes(blockIdToNumKeyPrefix, id), k1.bytes())
			k2 := blockOffsetKey{id}
			assert.Equal(t, referenceBlockIDKeyBytes(blockOffsetKeyPrefix, id), k2.bytes())
			k3 := blocksInfoKey{id}
			assert.Equal(t, referenceBlockIDKeyBytes(blocksInfoKeyPrefix, id), k3.bytes())
			k4 := patchKey{id}
			assert.Equal(t, referenceBlockIDKeyBytes(patchKeyPrefix, id), k4.bytes())

			key := blockIDKeyBytes(blockIdToNumKeyPrefix, id)
			assert.Equal(t, 1+id.Len(), cap(key)) // Exactly sized.
			allocs := testing.AllocsPerRun(100, func() {
				key = blockIDKeyBytes(blockIdToNumKeyPrefix, id)
			})
			assert.LessOrEqual(t, allocs, 1.0)
		})
	}
}
