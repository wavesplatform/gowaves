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
	stateHeight, err := a.state.Height()
	if err != nil {
		return nil, err
	}

	vrf, err := a.state.BlockVRF(&lastKnownBlock.BlockHeader, stateHeight)
	if err != nil {
		return nil, err
	}

	_ = a.state.TxValidation(func(validation state.TxValidation) error {
		for {
			t := a.utx.Pop()
			if t == nil {
				break
			}
			if err := validation.ValidateNextTx(t.T, currentTimestamp, lastKnownBlock.Timestamp, lastKnownBlock.Version, vrf); err == nil {
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
