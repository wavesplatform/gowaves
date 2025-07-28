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
	state      stateWrapper
	utx        types.UtxPool
	tm         types.Time
	cancelChan <-chan struct{}
}

func newBulkValidator(state stateWrapper, utx types.UtxPool, tm types.Time, cancelChan <-chan struct{}) *bulkValidator {
	return &bulkValidator{
		state:      state,
		utx:        utx,
		tm:         tm,
		cancelChan: cancelChan,
	}
}

func (a bulkValidator) Validate() {
	transactions, err := a.validate() // Pop transactions from UTX, clean UTX
	if err != nil {
		zap.S().Debug(err)
		return
	}
	for _, t := range transactions {
		errAdd := a.utx.AddWithBytesRaw(t.T, t.B)
		if errAdd != nil {
			zap.S().Errorf("failed to add a transaction to UTX, %v", errAdd)
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

	utxLen := len(a.utx.AllTransactions())

	for i := 0; i < utxLen; i++ {
		select {
		case <-a.cancelChan:
			zap.S().Info("Validation cancelled — exiting loop early, preserving remaining UTX transactions")
			return transactions, nil
		default:
			t := a.utx.Pop()
			if t == nil {
				break
			}
			err := a.state.TxValidation(func(validation state.TxValidation) error {
				_, err := validation.ValidateNextTx(t.T, currentTimestamp, lastKnownBlock.Timestamp, lastKnownBlock.Version, false)
				return err
			})
			if stateerr.IsTxCommitmentError(err) {
				zap.S().Errorf("failed to unpack a transaction from utx, %v", err)
				// This should not happen in practice.
				// Reset state, return applied transactions to UTX.
				a.state.ResetList()
				for _, tx := range transactions {
					_ = a.utx.AddWithBytesRaw(tx.T, tx.B)
				}
				transactions = nil
				continue
			} else if err == nil {
				transactions = append(transactions, t)
			}
		}
	}
	a.state.ResetList()
	return transactions, nil
}
