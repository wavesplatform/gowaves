package state

//go:generate enumer -type SnapshotApplicationMode -trimprefix SnapshotApplicationMode -text -output snapshotapplicationmode_string.go
type SnapshotApplicationMode int

const (
	SnapshotApplicationModeTransactionsOnly SnapshotApplicationMode = iota
	SnapshotApplicationModeSnapshotOnly
	SnapshotApplicationModeSnapshotThenTransactions
)
