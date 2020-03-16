package node

import (
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/types"
)

const (
	maxShiftFromNow = 600000 // 10 minutes.
)

func MaybeEnableExtendedApi(state state.State, time types.Time) error {
	lastBlock := state.TopBlock()
	return maybeEnableExtendedApi(state, lastBlock, proto.NewTimestampFromTime(time.Now()))
}

type startProvidingExtendedApi interface {
	StartProvidingExtendedApi() error
}

func maybeEnableExtendedApi(state startProvidingExtendedApi, lastBlock *proto.Block, now proto.Timestamp) error {
	provideExtended := false
	if lastBlock.Timestamp > now {
		provideExtended = true
	} else if now-lastBlock.Timestamp < maxShiftFromNow {
		provideExtended = true
	}
	if provideExtended {
		if err := state.StartProvidingExtendedApi(); err != nil {
			return err
		}
	}
	return nil
}
