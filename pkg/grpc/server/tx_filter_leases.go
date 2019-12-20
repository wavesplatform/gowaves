package server

import (
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
)

type txFilterLeases struct {
	f  *txFilter
	st state.State
}

func newTxFilterLeases(filter *txFilter, st state.State) *txFilterLeases {
	return &txFilterLeases{filter, st}
}

func (fl *txFilterLeases) filter(tx proto.Transaction) bool {
	if tx.GetTypeVersion().Type != proto.LeaseTransaction {
		return false
	}
	if !fl.f.filter(tx) {
		return false
	}
	switch t := tx.(type) {
	case *proto.LeaseV1:
		isActive, err := fl.st.IsActiveLeasing(*t.ID)
		if err != nil {
			return false
		}
		return isActive
	case *proto.LeaseV2:
		isActive, err := fl.st.IsActiveLeasing(*t.ID)
		if err != nil {
			return false
		}
		return isActive
	default:
		return false
	}
}
