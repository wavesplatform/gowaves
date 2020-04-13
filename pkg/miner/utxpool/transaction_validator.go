package utxpool

import (
	"errors"

	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
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
		return errors.New("state outdate, transaction not accepted")
	}
	stateHeight, err := a.state.Height()
	if err != nil {
		return err
	}
	vrf, err := a.state.BlockVRF(&lastKnownBlock.BlockHeader, stateHeight)
	if err != nil {
		return err
	}
	return a.state.TxValidation(func(validation state.TxValidation) error {
		return validation.ValidateNextTx(t, currentTimestamp, lastKnownBlock.Timestamp, lastKnownBlock.Version, vrf)
	})
}

type NoOpValidator struct {
}

func (a NoOpValidator) Validate(t proto.Transaction) error {
	return nil
}
