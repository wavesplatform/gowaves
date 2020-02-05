package utxpool

import (
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/types"
)

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
	mu := a.state.Mutex()
	locked := mu.Lock()
	defer locked.Unlock()
	currentTimestamp := proto.NewTimestampFromTime(a.tm.Now())
	lastKnownBlock, err := a.state.TopBlock()
	if err != nil {
		return err
	}
	err = a.state.ValidateNextTx(t, currentTimestamp, lastKnownBlock.Timestamp, lastKnownBlock.Version)
	a.state.ResetValidationList()
	return err
}

type NoOpValidator struct {
}

func (a NoOpValidator) Validate(t proto.Transaction) error {
	return nil
}
