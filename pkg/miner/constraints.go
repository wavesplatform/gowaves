package miner

type MaxScriptsComplexityInBlockT struct {
	BeforeRideV5 int
	AfterRideV5  int
}

type Constraints struct {
	MaxScriptRunsInBlock        int
	MaxScriptsComplexityInBlock MaxScriptsComplexityInBlockT
	ClassicAmountOfTxsInBlock   int
	MaxTxsSizeInBytes           int
}

func DefaultConstraints() Constraints {
	return Constraints{
		MaxScriptRunsInBlock:        100,
		MaxScriptsComplexityInBlock: MaxScriptsComplexityInBlockT{BeforeRideV5: 1000000, AfterRideV5: 2500000},
		ClassicAmountOfTxsInBlock:   100,
		MaxTxsSizeInBytes:           1 * 1024 * 1024, // 1mb
	}
}
