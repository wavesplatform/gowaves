package utxpool

import (
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/types"
	"go.uber.org/zap"
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
	transactions, err := a.validate()
	if err != nil {
		zap.S().Debug(err)
		return
	}
	for _, t := range transactions {
		_ = a.utx.AddWithBytes(t.T, t.B)
	}
}

func (a bulkValidator) validate() ([]*types.TransactionWithBytes, error) {
	if a.utx.Count() == 0 {
		return nil, nil
	}
	var transactions []*types.TransactionWithBytes
	currentTimestamp := proto.NewTimestampFromTime(a.tm.Now())
	lastKnownBlock := a.state.TopBlock()

	_ = a.state.Map(func(s state.NonThreadSafeState) error {
		defer s.ResetValidationList()

		for {
			t := a.utx.Pop()
			if t == nil {
				break
			}
			err := s.ValidateNextTx(t.T, currentTimestamp, lastKnownBlock.Timestamp, lastKnownBlock.Version, false)
			if state.IsTxCommitmentError(err) {
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

type noOnBulkValidator struct {
}

func (noOnBulkValidator) Validate() {
}
