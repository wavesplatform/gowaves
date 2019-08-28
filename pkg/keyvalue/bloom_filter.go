package keyvalue

import (
	"github.com/cespare/xxhash"
	"github.com/steakknife/bloomfilter"
)

type BloomFilterParams struct {
	// N is how many items will be added to the filter.
	N int
	// FalsePositiveProbability is acceptable false positive rate {0..1}.
	FalsePositiveProbability float64
}

type bloomFilter struct {
	filter *bloomfilter.Filter
}

func newBloomFilter(params BloomFilterParams) (*bloomFilter, error) {
	bf, err := bloomfilter.NewOptimal(uint64(params.N), params.FalsePositiveProbability)
	if err != nil {
		return nil, err
	}
	return &bloomFilter{filter: bf}, nil
}

func (bf *bloomFilter) add(data []byte) error {
	f := xxhash.New()
	if _, err := f.Write(data); err != nil {
		return err
	}
	bf.filter.Add(f)
	return nil
}

func (bf *bloomFilter) notInTheSet(data []byte) (bool, error) {
	f := xxhash.New()
	if _, err := f.Write(data); err != nil {
		return false, err
	}
	return !bf.filter.Contains(f), nil
}
