package state

import "github.com/wavesplatform/gowaves/pkg/proto"

// Prefixes for batched storage (batched_storage.go).
const (
	maxTransactionIdsBatchSize = 1 * proto.KiB
)

// Secondary keys prefixes for batched storage
const (
	transactionIdsPrefix byte = iota
)
