package keyvalue

import "io"

type bloomFilterStub struct {
	params BloomFilterParams
}

func NewBloomFilterStub(params BloomFilterParams) bloomFilterStub {
	return bloomFilterStub{
		params: params,
	}
}

func (bloomFilterStub) notInTheSet([]byte) (bool, error) {
	return false, nil
}

func (a bloomFilterStub) Params() BloomFilterParams {
	return a.params
}

func (a bloomFilterStub) WriteTo(to io.Writer) (int64, error) {
	return 0, nil
}

func (a bloomFilterStub) add([]byte) error {
	return nil
}
