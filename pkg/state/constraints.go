package state

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

const FreeVerifierComplexity = 200

const MaxVerifierScriptComplexityReduced = 2000
const MaxVerifierScriptComplexity = 4000

const MaxCallableScriptComplexityV12 = 2000
const MaxCallableScriptComplexityV34 = 4000
const MaxCallableScriptComplexityV5 = 10000
