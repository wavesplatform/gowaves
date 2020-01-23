package state

const (
	// Prefixes for batched storage (batched_storage.go).
	transactionIdsPrefix byte = iota

	maxTransactionIdsBatchSize = 1 * KiB
)
