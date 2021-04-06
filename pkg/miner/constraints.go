package miner

import (
	"github.com/wavesplatform/gowaves/pkg/state"
)

type Constraints struct {
	MaxScriptRunsInBlock        int
	MaxScriptsComplexityInBlock state.MaxScriptsComplexityInBlock
	ClassicAmountOfTxsInBlock   int
	MaxTxsSizeInBytes           int
}

func DefaultConstraints() Constraints {
	return Constraints{
		MaxScriptRunsInBlock:        100,
		MaxScriptsComplexityInBlock: state.NewMaxScriptsComplexityInBlock(),
		ClassicAmountOfTxsInBlock:   100,
		MaxTxsSizeInBytes:           1 * 1024 * 1024, // 1mb
	}
}
