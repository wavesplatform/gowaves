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
		err = filter.add(data)
		assert.NoError(t, err)
		notInTheSet, err := filter.notInTheSet(data)
		assert.NoError(t, err)
		assert.Equal(t, notInTheSet, false, "notInTheSet() returned wrong result")
	}
}
