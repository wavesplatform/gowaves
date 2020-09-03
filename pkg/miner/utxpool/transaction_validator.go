package utxpool

import (
	"errors"

	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/types"
)

type Validator interface {
	Validate(t proto.Transaction) error
}

type ValidatorImpl struct {
	state     stateWrapper
	tm        types.Time
	outdateMs uint64
}

func NewValidator(state stateWrapper, tm types.Time, outdateMs uint64) *ValidatorImpl {
	return &ValidatorImpl{
		state:     state,
		tm:        tm,
		outdateMs: outdateMs,
	}
}

func (a *ValidatorImpl) Validate(t proto.Transaction) error {
	currentTimestamp := proto.NewTimestampFromTime(a.tm.Now())
	lastKnownBlock := a.state.TopBlock()
	if currentTimestamp-lastKnownBlock.Timestamp > a.outdateMs {
		return errors.New("state outdated, transaction not accepted")
	}
	return a.state.TxValidation(func(validation state.TxValidation) error {
		return validation.ValidateNextTx(t, currentTimestamp, lastKnownBlock.Timestamp, lastKnownBlock.Version, false)
	})
}

type NoOpValidator struct {
}

func (a NoOpValidator) Validate(t proto.Transaction) error {
	return nil
}
