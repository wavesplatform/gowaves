package retransmit

import (
	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"testing"
)

func TestTransactionList(t *testing.T) {
	d1, _ := crypto.FastHash([]byte("1"))
	d2, _ := crypto.FastHash([]byte("2"))
	d3, _ := crypto.FastHash([]byte("3"))
	d4, _ := crypto.FastHash([]byte("4"))

	lst := NewTransactionList(2)
	assert.Equal(t, 0, lst.Len())

	t1 := proto.TransferV2{ID: &d1}
	lst.Add(&t1)
	assert.Equal(t, true, lst.Exists(&t1))
	assert.Equal(t, 1, lst.Len())

	t2 := proto.TransferV2{ID: &d2}
	lst.Add(&t2)
	assert.Equal(t, true, lst.Exists(&t2))
	assert.Equal(t, true, lst.Exists(&t1))
	assert.Equal(t, 2, lst.Len())

	t3 := proto.TransferV2{ID: &d3}
	lst.Add(&t3)
	assert.Equal(t, false, lst.Exists(&t1))
	assert.Equal(t, true, lst.Exists(&t2))
	assert.Equal(t, true, lst.Exists(&t3))
	assert.Equal(t, 2, lst.Len())

	t4 := proto.TransferV2{ID: &d4}
	lst.Add(&t4)
	assert.Equal(t, false, lst.Exists(&t1))
	assert.Equal(t, false, lst.Exists(&t2))
	assert.Equal(t, true, lst.Exists(&t3))
	assert.Equal(t, true, lst.Exists(&t4))
	assert.Equal(t, 2, lst.Len())
}
