package utxpool

import (
	"time"

	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/types"
)

type Validator interface {
	Validate(st types.UtxPoolValidatorState, t proto.Transaction) error
}

type ValidatorImpl struct {
	tm           types.Time
	obsolescence time.Duration
}

func NewValidator(tm types.Time, obsolescence time.Duration) (*ValidatorImpl, error) {
	if obsolescence <= 0 {
		return nil, errors.New("blockchain obsolescence period must be positive")
	}
	return &ValidatorImpl{
		tm:           tm,
		obsolescence: obsolescence,
	}, nil
}

func (a *ValidatorImpl) Validate(st types.UtxPoolValidatorState, tx proto.Transaction) error {
	now := a.tm.Now()
	lastBlock := st.TopBlock()
	lastBlockTime := time.UnixMilli(int64(lastBlock.Timestamp))
	if now.Add(-a.obsolescence).After(lastBlockTime) {
		return errors.New("state outdated, transaction not accepted")
	}
	return st.TxValidation(func(validation state.TxValidation) error {
		_, err := validation.ValidateNextTx(tx, uint64(now.UnixMilli()), lastBlock.Timestamp, lastBlock.Version, false)
		return err
	})
}

type NoOpValidator struct {
}

func (a NoOpValidator) Validate(_ types.UtxPoolValidatorState, _ proto.Transaction) error {
	return nil
}
