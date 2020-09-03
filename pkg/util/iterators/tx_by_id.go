package iterators

import (
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
)

type TxByIdIterator struct {
	ids [][]byte
	id  int
	s   state.StateInfo
}

func NewTxByIdIterator(s state.StateInfo, ids [][]byte) *TxByIdIterator {
	return &TxByIdIterator{
		ids: ids,
		id:  -1,
		s:   s,
	}
}

func (a *TxByIdIterator) Transaction() (proto.Transaction, bool, error) {
	return a.s.TransactionByIDWithStatus(a.ids[a.id])
}

func (a *TxByIdIterator) Next() bool {
	a.id += 1
	return a.id < len(a.ids)
}

func (a *TxByIdIterator) Release() {

}

func (a *TxByIdIterator) Error() error {
	return nil
}
