package collect_writes

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCollectInt(t *testing.T) {
	b := new(bytes.Buffer)
	c := new(CollectInt)

	c.W(b.Write([]byte{1, 2, 3}))
	c.W(b.Write([]byte{1, 2, 3}))
	c.W(b.Write([]byte{1, 2, 3}))

	n, err := c.Ret()

	assert.Equal(t, 9, n)
	assert.Equal(t, nil, err)
}
