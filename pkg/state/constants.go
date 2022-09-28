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
	StateVersion = 14

	// Memory limit for address transactions. flush() is called when this
	// limit is exceeded.
	AddressTransactionsMemLimit = 50 * 1024 * 1024
	// Number of keys per flush() call.
	AddressTransactionsMaxKeys = 4000

	// Maximum size of transactions by addresses file.
	// Then it is sorted and saved to DB.
	MaxAddressTransactionsFileSize = 2 * 1024 * 1024 * 1024 // 2 GiB.
)
