package utxpool

import (
	"github.com/wavesplatform/gowaves/pkg/proto"
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
	mu := a.state.Mutex()
	locked := mu.Lock()
	defer locked.Unlock()

	lastKnownBlock := a.state.TopBlock()
	stateHeight, err := a.state.Height()
	if err != nil {
		return nil, err
	}
	vrf, err := a.state.BlockVRF(&lastKnownBlock.BlockHeader, stateHeight)
	if err != nil {
		return nil, err
	}
	for {
		t := a.utx.Pop()
		if t == nil {
			break
		}
		if err := a.state.ValidateNextTx(t.T, currentTimestamp, lastKnownBlock.Timestamp, lastKnownBlock.Version, vrf); err == nil {
			transactions = append(transactions, t)
		}
	}
	a.state.ResetValidationList()
	return transactions, nil
}

type noOnBulkValidator struct {
}

func (noOnBulkValidator) Validate() {
}
