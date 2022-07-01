package retransmit

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func TestTransactionList(t *testing.T) {
	d1, _ := crypto.FastHash([]byte("1"))
	d2, _ := crypto.FastHash([]byte("2"))
	d3, _ := crypto.FastHash([]byte("3"))
	d4, _ := crypto.FastHash([]byte("4"))

	lst := NewTransactionList(2, proto.TestNetScheme)
	assert.Equal(t, 0, lst.Len())

	t1 := proto.TransferWithProofs{ID: &d1}
	lst.Add(&t1)
	assert.Equal(t, true, lst.Exists(&t1))
	assert.Equal(t, 1, lst.Len())

	t2 := proto.TransferWithProofs{ID: &d2}
	lst.Add(&t2)
	assert.Equal(t, true, lst.Exists(&t2))
	assert.Equal(t, true, lst.Exists(&t1))
	assert.Equal(t, 2, lst.Len())

	t3 := proto.TransferWithProofs{ID: &d3}
	lst.Add(&t3)
	assert.Equal(t, false, lst.Exists(&t1))
	assert.Equal(t, true, lst.Exists(&t2))
	assert.Equal(t, true, lst.Exists(&t3))
	assert.Equal(t, 2, lst.Len())

	t4 := proto.TransferWithProofs{ID: &d4}
	lst.Add(&t4)
	assert.Equal(t, false, lst.Exists(&t1))
	assert.Equal(t, false, lst.Exists(&t2))
	assert.Equal(t, true, lst.Exists(&t3))
	assert.Equal(t, true, lst.Exists(&t4))
	assert.Equal(t, 2, lst.Len())
}
