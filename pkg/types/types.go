package types

import (
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
)

type Scheduler interface {
	Reschedule(state state.State, curBlock *proto.Block, height uint64)
}

// Miner mutates state, applying block also. We can't do it together.
// We should interrupt miner, cause block applying has higher priority.
type MinerInterrupter interface {
	Interrupt()
}
