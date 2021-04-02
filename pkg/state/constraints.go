package state

type MaxScriptsComplexityInBlockT struct {
	BeforeRideV5 int
	AfterRideV5  int
}

func NewMaxScriptsComplexityInBlockT() MaxScriptsComplexityInBlockT {
	return MaxScriptsComplexityInBlockT{BeforeRideV5: 1000000, AfterRideV5: 2500000}
}

func (compl MaxScriptsComplexityInBlockT) GetMaxScriptsComplexityInBlock(isRiveV5Activated bool) int {
	if isRiveV5Activated {
		return compl.AfterRideV5
	}
	return compl.BeforeRideV5
}
