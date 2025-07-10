package utxpool

import (
	"go.uber.org/zap"

	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/state/stateerr"
	"github.com/wavesplatform/gowaves/pkg/types"
)

type BulkValidator interface {
	Validate()
}

type bulkValidator struct {
	state stateWrapper
	utx   types.UtxPool
	tm    types.Time
}

func newBulkValidator(state stateWrapper, utx types.UtxPool, tm types.Time) *bulkValidator {
	return &bulkValidator{
		state: state,
		utx:   utx,
		tm:    tm,
	}
}

func (a bulkValidator) Validate() {
	transactions, err := a.validate() // Pop transactions from UTX, clean UTX
	if err != nil {
		zap.S().Debug(err)
		return
	}
	for _, t := range transactions {
		err := a.utx.AddWithBytes(t.T, t.B)
		if err != nil {
			zap.S().Errorf("failed to add a transaction to UTX, %v", err)
			return
		}
	}
}

func (a bulkValidator) validate() ([]*types.TransactionWithBytes, error) {
	if a.utx.Count() == 0 {
		return nil, nil
	}
	var transactions []*types.TransactionWithBytes
	currentTimestamp := proto.NewTimestampFromTime(a.tm.Now())
	lastKnownBlock := a.state.TopBlock()

	_ = a.state.MapUnsafe(func(s state.NonThreadSafeState) error {
		defer s.ResetValidationList()
		utxLen := len(a.utx.AllTransactions())
		for i := 0; i < utxLen; i++ {
			t := a.utx.Pop()
			if t == nil {
				break
			}
			_, err := s.ValidateNextTx(t.T, currentTimestamp, lastKnownBlock.Timestamp, lastKnownBlock.Version, false)
			if stateerr.IsTxCommitmentError(err) {
				zap.S().Errorf("failed to unpack a transaction from utx, %v", err)
				// This should not happen in practice.
				// Reset state, return applied transactions to UTX.
				s.ResetValidationList()
				for _, tx := range transactions {
					_ = a.utx.AddWithBytes(tx.T, tx.B)
				}
				transactions = nil
				continue
			} else if err == nil {
				transactions = append(transactions, t)
			}
		}
		return nil
	})

	return transactions, nil
}
