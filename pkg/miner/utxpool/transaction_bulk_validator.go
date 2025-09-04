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
	state  stateWrapper
	utx    types.UtxPool
	tm     types.Time
	scheme proto.Scheme
}

func newBulkValidator(state stateWrapper, utx types.UtxPool, tm types.Time, scheme proto.Scheme) *bulkValidator {
	return &bulkValidator{
		state:  state,
		utx:    utx,
		tm:     tm,
		scheme: scheme,
	}
}

func (a bulkValidator) Validate(ctx context.Context) {
	transactions := a.validate(ctx) // Pop transactions from UTX, clean UTX
	for _, t := range transactions {
		errAdd := a.utx.AddWithBytesRaw(t.T, t.B)
		if errAdd != nil {
			// can happen because other nodes can send copies of txs while they are being validated
			slog.Warn("Failed to add a validated transaction to UTX",
				logging.Error(errAdd), logging.TxID(t.T, a.scheme),
			)
		}
	}
}

func (a bulkValidator) validate(ctx context.Context) []*types.TransactionWithBytes {
	if a.utx.Count() == 0 {
		slog.Debug("UTX pool is empty, nothing to validate")
		return nil
	}
	var transactions []*types.TransactionWithBytes
	currentTimestamp := proto.NewTimestampFromTime(a.tm.Now())
	lastKnownBlock := a.state.TopBlock()
	dropped := 0
	for checked := 0; ; checked++ { // just a counter for logging, checking until UTX is empty or context is done
		if ctx.Err() != nil {
			slog.Debug("Bulk validation interrupted", logging.Error(context.Cause(ctx)),
				slog.Int("valid", len(transactions)),
				slog.Int("checked", checked),
				slog.Int("dropped", dropped),
			)
			return transactions
		}
		t := a.utx.Pop()
		if t == nil {
			slog.Debug("UTX pool is empty, finished validating",
				slog.Int("valid", len(transactions)),
				slog.Int("checked", checked),
				slog.Int("dropped", dropped),
			)
			break
		}
		err := a.state.TxValidation(func(validation state.TxValidation) error {
			_, err := validation.ValidateNextTx(t.T, currentTimestamp, lastKnownBlock.Timestamp, lastKnownBlock.Version, false)
			return err
		})
		switch {
		case stateerr.IsTxCommitmentError(err):
			slog.Error("Failed to unpack a transaction from UTX",
				logging.Error(err), logging.TxID(t.T, a.scheme),
				slog.Int("valid", len(transactions)),
				slog.Int("checked", checked),
				slog.Int("dropped", dropped),
			)
			dropped++ // drop this tx
			// This should not happen in practice.
			// Reset state, return applied transactions to UTX.
			for _, tx := range transactions {
				utxErr := a.utx.AddWithBytesRaw(tx.T, tx.B)
				if utxErr != nil {
					dropped++ // drop this tx
					slog.Error("Failed to return a transaction to UTX",
						logging.Error(utxErr), logging.TxID(t.T, a.scheme),
					)
				}
			}
			slog.Debug("Valid transactions returned to UTX, resetting list and continuing",
				slog.Int("returned", len(transactions)),
				slog.Int("checked", checked),
				slog.Int("dropped", dropped),
			)
			clear(transactions)             // Clear the slice to avoid memory leak
			transactions = transactions[:0] // Reset the slice to empty
			continue
		case err != nil: // invalid tx, just drop it
			dropped++
		default: // valid tx
			transactions = append(transactions, t)
		}
	}
	return transactions
}
