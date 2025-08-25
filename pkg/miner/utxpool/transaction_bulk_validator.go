package utxpool

import (
	"context"
	"log/slog"

	"github.com/mr-tron/base58"

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

func txIDSlogAttr(t proto.Transaction, scheme proto.Scheme) slog.Attr {
	id, err := t.GetID(scheme)
	if err != nil {
		return slog.Group("tx-get-id", logging.Error(err))
	}
	return slog.String("tx-id", base58.Encode(id))
}

func (a bulkValidator) Validate(ctx context.Context) {
	transactions := a.validate(ctx) // Pop transactions from UTX, clean UTX
	for _, t := range transactions {
		errAdd := a.utx.AddWithBytesRaw(t.T, t.B)
		if errAdd != nil {
			slog.Error("failed to add a validated transaction to UTX",
				logging.Error(errAdd), txIDSlogAttr(t.T, a.scheme),
			)
		}
	}
}

func (a bulkValidator) validate(ctx context.Context) []*types.TransactionWithBytes {
	utxLen := a.utx.Count()
	if utxLen == 0 {
		slog.Debug("UTX pool is empty, nothing to validate")
		return nil
	}
	var transactions []*types.TransactionWithBytes
	currentTimestamp := proto.NewTimestampFromTime(a.tm.Now())
	lastKnownBlock := a.state.TopBlock()
	defer a.state.ResetList()
	for range utxLen {
		if ctx.Err() != nil {
			slog.Debug("Bulk validation interrupted:", logging.Error(context.Cause(ctx)))
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
					slog.Error("failed to return a transaction to UTX",
						logging.Error(utxErr), txIDSlogAttr(t.T, a.scheme),
					)
				}
			}
			clear(transactions)             // Clear the slice to avoid memory leak
			transactions = transactions[:0] // Reset the slice to empty
			continue
		} else if err == nil {
			transactions = append(transactions, t)
		}
	}
	return transactions
}
