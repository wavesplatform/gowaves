package server

import (
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
)

type txFilterInvoke struct {
	f  *txFilter
	st state.State
}

func newTxFilterInvoke(filter *txFilter, st state.State) *txFilterInvoke {
	return &txFilterInvoke{filter, st}
}

func (fl *txFilterInvoke) filter(tx proto.Transaction) bool {
	switch t := tx.(type) {
	case *proto.InvokeScriptV1:
		return fl.f.filter(t)
	default:
		return false
	}
}
