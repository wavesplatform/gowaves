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
	if a.utx.Len() == 0 {
		slog.Debug("UTX pool is empty, nothing to validate")
		return
	}
	currentTimestamp := proto.NewTimestampFromTime(a.tm.Now())
	lastKnownBlock := a.state.TopBlock()
	checked, dropped := a.utx.Clean(ctx, func(tx proto.Transaction) bool {
		err := a.state.TxValidation(func(validation state.TxValidation) error {
			_, err := validation.ValidateNextTx(tx, currentTimestamp, lastKnownBlock.Timestamp, lastKnownBlock.Version, false)
			return err
		})
		if err != nil {
			if stateerr.IsTxCommitmentError(err) {
				slog.Error("Failed to validate transaction from UTX", logging.Error(err), logging.TxID(tx, a.scheme))
			}
			return true // Drop invalid transaction.
		}
		return false // Valid transaction - keep it in UTX.
	})
	slog.Debug("Finished UTX pool validation", slog.Int("checked", checked), slog.Int("dropped", dropped))
}
