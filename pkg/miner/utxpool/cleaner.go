package utxpool

import (
	"context"

	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/types"
)

// Cleaner is responsible for validating transactions in the UTX pool and removing invalid ones.
type Cleaner struct {
	validator BulkValidator
	state     cleanerState
}

func NewCleaner(state cleanerState, pool types.UtxPool, tm types.Time, scheme proto.Scheme) *Cleaner {
	return &Cleaner{
		validator: newBulkValidator(state, pool, tm, scheme),
		state:     state,
	}
}

func (a *Cleaner) Clean(ctx context.Context) {
	a.validator.Validate(ctx)
}

// cleanerState is an interface that provides necessary methods for the Cleaner to interact with the blockchain state.
type cleanerState interface {
	TopBlock() *proto.Block
	TxValidation(func(validation state.TxValidation) error) error
}
