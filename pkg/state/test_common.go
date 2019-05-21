package state

import (
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
)

func defaultTestBloomFilterParams() keyvalue.BloomFilterParams {
	return keyvalue.BloomFilterParams{N: 2e6, FalsePositiveProbability: 0.01}
}
