package utxpool

import (
	"context"
	"log/slog"

	"github.com/wavesplatform/gowaves/pkg/logging"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/state/stateerr"
	"github.com/wavesplatform/gowaves/pkg/types"
)

type BulkValidator interface {
	Validate(ctx context.Context)
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

func (a bulkValidator) Validate(ctx context.Context) {
	transactions := a.validate(ctx) // Pop transactions from UTX, clean UTX
	for _, t := range transactions {
		errAdd := a.utx.AddWithBytesRaw(t.T, t.B)
		if errAdd != nil {
			slog.Error("failed to add a transaction to UTX", logging.Error(errAdd))
			return
		}
	}
}

func (a bulkValidator) validate(ctx context.Context) []*types.TransactionWithBytes {
	if a.utx.Count() == 0 {
		return nil
	}
	var transactions []*types.TransactionWithBytes
	currentTimestamp := proto.NewTimestampFromTime(a.tm.Now())
	lastKnownBlock := a.state.TopBlock()

	utxLen := len(a.utx.AllTransactions())

	for i := 0; i < utxLen; i++ {
		if ctx.Err() != nil {
			slog.Debug("Bulk validation interrupted:", logging.Error(ctx.Err()))
			return transactions
		}
		t := a.utx.Pop()
		if t == nil {
			break
		}
		err := a.state.TxValidation(func(validation state.TxValidation) error {
			_, err := validation.ValidateNextTx(t.T, currentTimestamp, lastKnownBlock.Timestamp, lastKnownBlock.Version, false)
			return err
		})
		if stateerr.IsTxCommitmentError(err) {
			slog.Error("failed to unpack a transaction from utx", logging.Error(err))
			// This should not happen in practice.
			// Reset state, return applied transactions to UTX.
			a.state.ResetList()
			for _, tx := range transactions {
				utxErr := a.utx.AddWithBytesRaw(tx.T, tx.B)
				if utxErr != nil {
					slog.Error("failed to return a transaction to UTX", logging.Error(utxErr))
				}
			}
			transactions = nil
			continue
		} else if err == nil {
			transactions = append(transactions, t)
		}
	}
	a.state.ResetList()
	return transactions
}
