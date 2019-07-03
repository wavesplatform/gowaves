package types

import "github.com/wavesplatform/gowaves/pkg/proto"

type Scheduler interface {
	Reschedule()
}

// Miner mutates state, applying block also. We can't do it together.
// We should interrupt miner, cause block applying has higher priority.
type MinerInterrupter interface {
	Interrupt()
}

type BlockApplier interface {
	Apply(block *proto.Block) error
}

// notify state that it must run synchronization
type StateHistorySynchronizer interface {
	Sync()
}
