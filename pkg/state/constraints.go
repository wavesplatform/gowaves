package state

type MaxScriptsComplexityInBlock struct {
	BeforeRideV5 int
	AfterRideV5  int
}

func NewMaxScriptsComplexityInBlock() MaxScriptsComplexityInBlock {
	return MaxScriptsComplexityInBlock{BeforeRideV5: 1000000, AfterRideV5: 2500000}
}

func (a MaxScriptsComplexityInBlock) GetMaxScriptsComplexityInBlock(isRiveV5Activated bool) int {
	if isRiveV5Activated {
		return a.AfterRideV5
	}
	return a.BeforeRideV5
}
