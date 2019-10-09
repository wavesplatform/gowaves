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
	// Bloom store
	Store store
}

func NewBloomFilterParams(N int, FalsePositiveProbability float64, store store) BloomFilterParams {
	return BloomFilterParams{
		N:                        N,
		FalsePositiveProbability: FalsePositiveProbability,
		Store:                    store,
	}
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

	bts, err := params.Store.load()
	if err != nil {
		return nil, err
	}
	_, err = bf.ReadFrom(bytes.NewBuffer(bts))
	if err != nil {
		return nil, err
	}
	return bf, nil
}

func storeBloomFilter(a *bloomFilter) error {
	return a.params.Store.save(a.filter)
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

type store interface {
	save(*bloomfilter.Filter) error
	load() ([]byte, error)
	WithPath(string)
}

type NoOpStore struct {
}

func (NoOpStore) WithPath(string) {}

func (a NoOpStore) save(*bloomfilter.Filter) error {
	return nil
}

func (a NoOpStore) load() ([]byte, error) {
	return nil, errors.New("noop")
}

type storeImpl struct {
	path string
}

func (a *storeImpl) WithPath(path string) {
	a.path = path
}

func (a *storeImpl) tmpFileName() string {
	return a.path + "tmp"
}

func (a *storeImpl) saveData(f *bloomfilter.Filter) error {
	file, err := os.OpenFile(a.tmpFileName(), os.O_RDWR|os.O_CREATE, 0644)
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
	return nil
}

func (a *storeImpl) save(f *bloomfilter.Filter) error {
	if err := a.saveData(f); err != nil {
		return err
	}
	if err := os.Rename(a.tmpFileName(), a.path); err != nil {
		return err
	}
	return nil
}

func (a *storeImpl) load() ([]byte, error) {
	bts, err := ioutil.ReadFile(a.path)
	if err != nil {
		return nil, err
	}
	return bts, os.Remove(a.path)
}

func NewStore(path string) *storeImpl {
	return &storeImpl{path: path}
}
