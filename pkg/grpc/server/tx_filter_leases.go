package server

import (
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
)

type txFilterLeases struct {
	f  *txFilter
	st state.StateInfo
}

func newTxFilterLeases(filter *txFilter, st state.StateInfo) *txFilterLeases {
	return &txFilterLeases{filter, st}
}

func (fl *txFilterLeases) filterLease(tx proto.Transaction, id crypto.Digest) bool {
	if !fl.f.filter(tx) {
		return false
	}
	isActive, err := fl.st.IsActiveLeasing(id)
	if err != nil {
		return false
	}
	return isActive
}

func (fl *txFilterLeases) filter(tx proto.Transaction) bool {
	switch t := tx.(type) {
	case *proto.LeaseWithSig:
		return fl.filterLease(t, *t.ID)
	case *proto.LeaseWithProofs:
		return fl.filterLease(t, *t.ID)
	default:
		return false
	}
}
