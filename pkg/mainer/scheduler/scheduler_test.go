package scheduler

import (
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
)

type mockInternal struct {
}

func (a mockInternal) schedule(state state.State, keyPairs []proto.KeyPair, schema proto.Schema, AverageBlockDelaySeconds uint64, confirmedBlock *proto.Block, confirmedBlockHeight uint64) []Emit {
	// TODO fix me
	panic("implement me")
}
