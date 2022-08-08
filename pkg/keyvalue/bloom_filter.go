package keyvalue

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"runtime/debug"

	"github.com/cespare/xxhash/v2"
	"github.com/pkg/errors"
	"github.com/steakknife/bloomfilter"
	"go.uber.org/zap"
)

type BloomFilter interface {
	notInTheSet(data []byte) (bool, error)
	add(data []byte) error
	Params() BloomFilterParams
	io.WriterTo
}

type BloomFilterParams struct {
	// N is how many items will be added to the filter.
	N int
	// FalsePositiveProbability is acceptable false positive rate {0..1}.
	FalsePositiveProbability float64
	// Bloom store.
	Store store
	// Disable bloom filter.
	Disable bool
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

func (bf *bloomFilter) Params() BloomFilterParams {
	return bf.params
}

func (bf *bloomFilter) ReadFrom(r io.Reader) (n int64, err error) {
	return bf.filter.ReadFrom(r)
}

func newBloomFilter(params BloomFilterParams) (BloomFilter, error) {
	if params.Disable {
		return NewBloomFilterStub(params), nil
	}
	bf, err := bloomfilter.NewOptimal(uint64(params.N), params.FalsePositiveProbability)
	if err != nil {
		return nil, err
	}
	return &bloomFilter{filter: bf, params: params}, nil
}

func newBloomFilterFromStore(params BloomFilterParams) (BloomFilter, error) {
	if params.Disable {
		return NewBloomFilterStub(params), nil
	}
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
	debug.FreeOSMemory()
	return bf, nil
}

func storeBloomFilter(a BloomFilter) error {
	debug.FreeOSMemory()
	if a.Params().Store != nil {
		return a.Params().Store.save(a)
	}
	return nil
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
	save(to io.WriterTo) error
	load() ([]byte, error)
	WithPath(string)
}

type NoOpStore struct {
}

func (NoOpStore) WithPath(string) {}

func (a NoOpStore) save(_ io.WriterTo) error {
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

func (a *storeImpl) saveData(f io.WriterTo) error {
	file, err := os.OpenFile(a.tmpFileName(), os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	defer func() {
		if err := file.Close(); err != nil {
			zap.S().Warnf("Failed to save bloom filter: %v", err)
		}
	}()

	if err := file.Truncate(0); err != nil {
		return err
	}

	if _, err := file.Seek(0, 0); err != nil {
		return err
	}

	buffer := bufio.NewWriter(file)
	defer func() {
		if err := buffer.Flush(); err != nil {
			zap.S().Warnf("Failed to save bloom filter: %v", err)
		}
	}()

	if _, err := f.WriteTo(buffer); err != nil {
		return err
	}
	return nil
}

func (a *storeImpl) save(f io.WriterTo) error {
	if err := a.saveData(f); err != nil {
		return err
	}
	if err := os.Rename(a.tmpFileName(), a.path); err != nil {
		return err
	}
	return nil
}

func (a *storeImpl) load() ([]byte, error) {
	bts, err := os.ReadFile(a.path)
	if err != nil {
		return nil, err
	}
	return bts, os.Remove(a.path)
}

func NewStore(path string) *storeImpl {
	return &storeImpl{path: path}
}
