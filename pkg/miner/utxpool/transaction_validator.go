package utxpool

import (
	"time"

	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/types"
)

type Validator interface {
	Validate(t proto.Transaction) error
}

type ValidatorImpl struct {
	state        stateWrapper
	tm           types.Time
	obsolescence time.Duration
}

func NewValidator(state stateWrapper, tm types.Time, obsolescence time.Duration) (*ValidatorImpl, error) {
	if obsolescence <= 0 {
		return nil, errors.New("blockchain obsolescence period must be positive")
	}
	return &ValidatorImpl{
		state:        state,
		tm:           tm,
		obsolescence: obsolescence,
	}, nil
}

func (a *ValidatorImpl) Validate(tx proto.Transaction) error {
	now := a.tm.Now()
	lastBlock := a.state.TopBlock()
	lastBlockTime := time.UnixMilli(int64(lastBlock.Timestamp))
	if now.Add(-a.obsolescence).After(lastBlockTime) {
		return errors.New("state outdated, transaction not accepted")
	}
	return a.state.TxValidation(func(validation state.TxValidation) error {
		return validation.ValidateNextTx(tx, uint64(now.UnixMilli()), lastBlock.Timestamp, lastBlock.Version, false)
	})
}

type NoOpValidator struct {
}

func (a NoOpValidator) Validate(t proto.Transaction) error {
	return nil
}
