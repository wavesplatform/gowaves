package state

const (
	FailFreeInvokeComplexity           = 1000
	FreeVerifierComplexity             = 200
	MaxVerifierScriptComplexityReduced = 2000
	MaxVerifierScriptComplexity        = 4000
	MaxCallableScriptComplexityV12     = 2000
	MaxCallableScriptComplexityV34     = 4000
	MaxCallableScriptComplexityV5      = 10000
	MaxCallableScriptComplexityV6      = 52000
)

type MaxScriptsComplexityInBlock struct {
	BeforeActivationRideV5Feature int
	AfterActivationRideV5Feature  int
}

func NewMaxScriptsComplexityInBlock() MaxScriptsComplexityInBlock {
	return MaxScriptsComplexityInBlock{BeforeActivationRideV5Feature: 1000000, AfterActivationRideV5Feature: 2500000}
}

func (a MaxScriptsComplexityInBlock) GetMaxScriptsComplexityInBlock(isRideV5Activated bool) int {
	if isRideV5Activated {
		return a.AfterActivationRideV5Feature
	}
	return a.BeforeActivationRideV5Feature
}
