package keyvalue

import (
	"bufio"
	"bytes"
	"io"
	"log/slog"
	"os"
	"runtime/debug"

	"github.com/cespare/xxhash/v2"
	"github.com/pkg/errors"
	"github.com/steakknife/bloomfilter"

	"github.com/wavesplatform/gowaves/pkg/logging"
)

type BloomFilter interface {
	notInTheSet(data []byte) (bool, error)
	add(data []byte) error
	Params() BloomFilterParams
	io.WriterTo
}

type BloomFilterParams struct {
	// BloomFilterCapacity is how many items will be added to the filter.
	BloomFilterCapacity uint64
	// FalsePositiveProbability is acceptable false positive rate {0..1}.
	FalsePositiveProbability float64
	// Bloom store.
	BloomFilterStore store
	// Disable bloom filter.
	DisableBloomFilter bool
}

func NewBloomFilterParams(capacity uint64, probability float64, store store) BloomFilterParams {
	return BloomFilterParams{
		BloomFilterCapacity:      capacity,
		FalsePositiveProbability: probability,
		BloomFilterStore:         store,
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
	if params.DisableBloomFilter {
		return NewBloomFilterStub(params), nil
	}
	bf, err := bloomfilter.NewOptimal(params.BloomFilterCapacity, params.FalsePositiveProbability)
	if err != nil {
		return nil, err
	}
	return &bloomFilter{filter: bf, params: params}, nil
}

func newBloomFilterFromStore(params BloomFilterParams) (BloomFilter, error) {
	if params.DisableBloomFilter {
		return NewBloomFilterStub(params), nil
	}
	f, err := bloomfilter.NewOptimal(params.BloomFilterCapacity, params.FalsePositiveProbability)
	if err != nil {
		return nil, err
	}
	bf := &bloomFilter{filter: f, params: params}

	bts, err := params.BloomFilterStore.load()
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
	if a.Params().BloomFilterStore != nil {
		return a.Params().BloomFilterStore.save(a)
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
		if clErr := file.Close(); clErr != nil {
			slog.Warn("Failed to save bloom filter", logging.Error(clErr))
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
		if flErr := buffer.Flush(); flErr != nil {
			slog.Warn("Failed to save bloom filter", logging.Error(flErr))
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
