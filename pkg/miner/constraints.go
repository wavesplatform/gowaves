package miner

type Constraints struct {
	MaxScriptRunsInBlock        int
	MaxScriptsComplexityInBlock int
	ClassicAmountOfTxsInBlock   int
	MaxTxsSizeInBytes           int
}

func DefaultConstraints() Constraints {
	return Constraints{
		MaxScriptRunsInBlock:        100,
		MaxScriptsComplexityInBlock: 1000000,
		ClassicAmountOfTxsInBlock:   100,
		MaxTxsSizeInBytes:           1 * 1024 * 1024, // 1mb
	}
}
