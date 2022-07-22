package keyvalue

import (
	"math/rand"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

const (
	n                        = 10
	falsePositiveProbability = 0.0001
)

func TestBloomFilter(t *testing.T) {
	filter, err := newBloomFilter(BloomFilterParams{n, falsePositiveProbability, nil, false})
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

func TestSaveLoad(t *testing.T) {
	dir := t.TempDir()
	cacheFile := path.Join(dir, "bloom_cache")

	params := NewBloomFilterParams(n, falsePositiveProbability, NewStore(cacheFile))
	filter, err := newBloomFilter(params)
	require.NoError(t, err)

	sig := crypto.MustSignatureFromBase58("5Sqy6nibWVBtHiok8WJfx3p2kiRkAidGuGzY7Aqb7jCZauJCSdEicvQj9HLbiopEtU33RzJ1ErXN7aB16GRuEPqB")

	err = filter.add(sig.Bytes())
	require.NoError(t, err)

	err = storeBloomFilter(filter)
	require.NoError(t, err)

	filter, err = newBloomFilter(params)
	require.NoError(t, err)

	rs, err := filter.notInTheSet(sig.Bytes())
	require.NoError(t, err)
	require.Equal(t, true, rs)

	filter, err = newBloomFilterFromStore(params)
	require.NoError(t, err)

	rs, err = filter.notInTheSet(sig.Bytes())
	require.NoError(t, err)
	require.False(t, rs)
}
