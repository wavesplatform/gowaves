package state

const (
	// Default values.
	// Cache parameters.
	// 500 MiB.
	DefaultCacheSize = 500 * 1024 * 1024

	// Bloom filter parameters.
	// Number of elements in Bloom Filter.
	DefaultBloomFilterSize = 2e8
	// Acceptable false positive for Bloom Filter (0.01%).
	DefaultBloomFilterFalsePositiveProbability = 0.0001

	// Db parameters.
	DefaultWriteBuffer         = 32 * 1024 * 1024
	DefaultCompactionTableSize = 8 * 1024 * 1024
	DefaultCompactionTotalSize = 10 * 1024 * 1024

	// Block storage parameters.
	// DefaultOffsetLen is the amount of bytes needed to store offset of transactions in blockchain file.
	DefaultOffsetLen = 8
	// DefaultHeaderOffsetLen is the amount of bytes needed to store offset of headers in headers file.
	DefaultHeaderOffsetLen = 8

	// StateVersion is current version of state internal storage formats.
	// It increases when backward compatibility with previous storage version is lost.
	// Version 27 changes the formats of Waves and asset balances records and the order of fields hashes of legacy
	// state hashes, see migration_v26.go.
	// Version 28 stores the sizes of entries of variable size history records as uvarint, see migration_v27.go.
	StateVersion = 28

	// Memory limit for address transactions. flush() is called when this
	// limit is exceeded.
	AddressTransactionsMemLimit = 50 * 1024 * 1024
	// Number of keys per flush() call.
	AddressTransactionsMaxKeys = 4000

	// Maximum size of transactions by addresses file.
	// Then it is sorted and saved to DB.
	MaxAddressTransactionsFileSize = 2 * 1024 * 1024 * 1024 // 2 GiB.

	// Invoke access statistics parameters (see invoke_stats.go).
	// invokeStatsKeysSoftCap is the number of read keys and the number of write keys kept in the statistics
	// of a function, the least recently seen keys above it are evicted.
	invokeStatsKeysSoftCap = 64
	// invokeStatsKeysHardCap is the maximum number of read keys or write keys of a single execution, the function
	// that exceeds it is marked as wide and its keys are not tracked anymore.
	invokeStatsKeysHardCap = 256
	// invokeStatsCacheSize is the number of functions which statistics are kept in memory.
	invokeStatsCacheSize = 4096
	// invokeSpeculationLookahead is the maximum number of upcoming transactions of a block considered for
	// a batch of speculatively evaluated invokes (see invoke_scheduler.go).
	invokeSpeculationLookahead = 64
)
