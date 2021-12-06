package state

// Prefixes for batched storage (batched_storage.go).
const (
	maxTransactionIdsBatchSize = 1 * KiB
)

// Secondary keys prefixes for batched storage
const (
	transactionIdsPrefix byte = iota
)
