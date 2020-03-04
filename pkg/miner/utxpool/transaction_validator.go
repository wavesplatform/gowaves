package utxpool

import (
	"errors"

	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/types"
)

const DELTA = 86400 * 1000 / 6 // 4 hours

type Validator interface {
	Validate(t proto.Transaction) error
}

type ValidatorImpl struct {
	state stateWrapper
	tm    types.Time
}

func NewValidator(state stateWrapper, tm types.Time) *ValidatorImpl {
	return &ValidatorImpl{
		state: state,
		tm:    tm,
	}
}

func (a *ValidatorImpl) Validate(t proto.Transaction) error {
	currentTimestamp := proto.NewTimestampFromTime(a.tm.Now())
	lastKnownBlock := a.state.TopBlock()
	if currentTimestamp-lastKnownBlock.Timestamp > DELTA {
		return errors.New("state in sync, transaction not accepted")
	}

	mu := a.state.Mutex()
	locked := mu.Lock()
	err := a.state.ValidateNextTx(t, currentTimestamp, lastKnownBlock.Timestamp, lastKnownBlock.Version)
	a.state.ResetValidationList()
	locked.Unlock()
	return err
}

type NoOpValidator struct {
}

func (a NoOpValidator) Validate(t proto.Transaction) error {
	return nil
}
