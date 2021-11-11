package peer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestID(t *testing.T) {
	assert.Equal(t, "127.0.0.1-100500", id("127.0.0.1:6868", 100500))
}
