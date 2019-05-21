package keyvalue

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	n                        = 10
	falsePositiveProbability = 0.0001
)

func TestBloomFilter(t *testing.T) {
	filter, err := newBloomFilter(BloomFilterParams{n, falsePositiveProbability})
	assert.NoError(t, err, "newBloomFilter() failed")
	for i := 0; i < n; i++ {
		data := make([]byte, 100)
		_, err := rand.Read(data)
		assert.NoError(t, err, "rand.Read() failed")
		filter.add(data)
		assert.Equal(t, filter.notInTheSet(data), false, "notInTheSet() returned wrong result")
	}
}
