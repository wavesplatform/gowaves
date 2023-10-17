package node

import (
	"github.com/wavesplatform/gowaves/pkg/proto"
	storage "github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/types"
)

const (
	maxShiftFromNow = 600000 // 10 minutes.
)

func MaybeEnableExtendedAPI(state storage.State, time types.Time) error {
	lastBlock := state.TopBlock()
	return maybeEnableExtendedAPI(state, lastBlock, proto.NewTimestampFromTime(time.Now()))
}

type startProvidingExtendedAPI interface {
	StartProvidingExtendedAPI() error
}

func maybeEnableExtendedAPI(state startProvidingExtendedAPI, lastBlock *proto.Block, now proto.Timestamp) error {
	provideExtended := false
	if lastBlock.Timestamp > now {
		provideExtended = true
	} else if now-lastBlock.Timestamp < maxShiftFromNow {
		provideExtended = true
	}
	if provideExtended {
		if err := state.StartProvidingExtendedAPI(); err != nil {
			return err
		}
	}
	return nil
}
