package scheduler

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/types"
)

type mockInternal struct {
}

func (a mockInternal) schedule(
	state.StateInfo,
	[]proto.KeyPair,
	*settings.BlockchainSettings,
	*proto.Block,
	uint64,
	types.Time,
	bool,
) ([]Emit, error) {
	return nil, nil
}

func TestSchedulerImpl_Emits(t *testing.T) {
	sch := newScheduler(mockInternal{}, nil, nil, nil, nil, nil, 0, true)
	sch.Reschedule()
	rs := sch.Emits()

	require.EqualValues(t, []Emit([]Emit(nil)), rs)
}
