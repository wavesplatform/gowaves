package types

import (
	"github.com/wavesplatform/gowaves/pkg/proto"
)

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
	ApplyBytes([]byte) error
}

// notify state that it must run synchronization
type StateHistorySynchronizer interface {
	Sync()
}

// Abstract handler that called when event happens
type Handler interface {
	Handle()
}

// UtxPool storage interface
type UtxPool interface {
	AddWithBytes(t proto.Transaction, b []byte) (added bool)
	Exists(t proto.Transaction) bool
	Pop() *TransactionWithBytes
}

type TransactionWithBytes struct {
	T proto.Transaction
	B []byte
}
