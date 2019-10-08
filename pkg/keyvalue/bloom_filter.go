package keyvalue

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"

	"github.com/cespare/xxhash"
	"github.com/pkg/errors"
	"github.com/steakknife/bloomfilter"
)

type BloomFilterParams struct {
	// N is how many items will be added to the filter.
	N int
	// FalsePositiveProbability is acceptable false positive rate {0..1}.
	FalsePositiveProbability float64
	// Path where bloom cache stored
	Path string
}

type bloomFilter struct {
	filter *bloomfilter.Filter
	params BloomFilterParams
}

func (bf *bloomFilter) WriteTo(w io.Writer) (n int64, err error) {
	return bf.filter.WriteTo(w)
}

func (bf *bloomFilter) ReadFrom(r io.Reader) (n int64, err error) {
	return bf.filter.ReadFrom(r)
}

func newBloomFilter(params BloomFilterParams) (*bloomFilter, error) {
	bf, err := bloomfilter.NewOptimal(uint64(params.N), params.FalsePositiveProbability)
	if err != nil {
		return nil, err
	}
	return &bloomFilter{filter: bf, params: params}, nil
}

func newBloomFilterFromStore(params BloomFilterParams) (*bloomFilter, error) {
	f, err := bloomfilter.NewOptimal(uint64(params.N), params.FalsePositiveProbability)
	if err != nil {
		return nil, err
	}
	bf := &bloomFilter{filter: f, params: params}
	file, err := os.Open(params.Path)
	if err != nil {
		return nil, err
	}
	bts, err := ioutil.ReadAll(file)
	if err != nil {
		_ = file.Close()
		return nil, err
	}
	err = file.Close()
	if err != nil {
		return nil, err
	}
	err = os.Remove(params.Path)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(bts[len(bts)-4:], []byte{0xaa, 0xbb, 0xcc, 0xdd}) {
		return nil, errors.New("bloomFilter: invalid data")
	}
	_, err = bf.ReadFrom(bytes.NewReader(bts[:len(bts)-4]))
	if err != nil {
		return nil, errors.Wrap(err, "UnmarshalBinary")
	}
	return bf, nil
}

func storeBloomFilter(f *bloomFilter) error {
	file, err := os.OpenFile(f.params.Path, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	err = file.Truncate(0)
	if err != nil {
		return err
	}
	_, err = file.Seek(0, 0)
	if err != nil {
		return err
	}
	_, err = f.WriteTo(file)
	if err != nil {
		return err
	}
	_, err = file.Write([]byte{0xaa, 0xbb, 0xcc, 0xdd})
	if err != nil {
		return err
	}
	return err
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
