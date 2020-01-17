package scheduler

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
)

type mockInternal struct {
}

func (a mockInternal) schedule(state state.State, keyPairs []proto.KeyPair, schema proto.Scheme, AverageBlockDelaySeconds uint64, confirmedBlock *proto.Block, confirmedBlockHeight uint64) []Emit {
	return nil
}

func TestSchedulerImpl_Emits(t *testing.T) {
	sch := newScheduler(mockInternal{}, nil, nil, nil, nil)
	sch.Reschedule()
	rs := sch.Emits()

	require.EqualValues(t, []Emit([]Emit(nil)), rs)
}
