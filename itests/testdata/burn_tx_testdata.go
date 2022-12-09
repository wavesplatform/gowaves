package testdata

import (
	"github.com/wavesplatform/gowaves/itests/config"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type BurnTestData[T any] struct {
	Account   config.AccountInfo
	AssetID   crypto.Digest
	Quantity  uint64
	ChainID   proto.Scheme
	Timestamp uint64
	Fee       uint64
	Expected  T
}

type BurnExpectedValuesPositive struct {
	WavesDiffBalance int64
	AssetBalance     int64
	_                struct{} // this field is necessary to force using explicit struct initialization
}

type BurnExpectedValuesNegative struct {
	ErrGoMsg          string
	ErrScalaMsg       string
	ErrBrdCstGoMsg    string
	ErrBrdCstScalaMsg string
	WavesDiffBalance  int64
	AssetBalance      int64
	_                 struct{} // this field is necessary to force using explicit struct initialization
}

func NewBurnTestData[T any]() *BurnTestData[T] {
	return &BurnTestData[T]{}
}
