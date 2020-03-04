package server

import (
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type txFilterInvoke struct {
	f *txFilter
}

func newTxFilterInvoke(filter *txFilter) *txFilterInvoke {
	return &txFilterInvoke{filter}
}

func (fl *txFilterInvoke) filter(tx proto.Transaction) bool {
	switch t := tx.(type) {
	case *proto.InvokeScriptWithProofs:
		return fl.f.filter(t)
	default:
		return false
	}
}
