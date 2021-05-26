package ride

type VmEstimator struct {
	builtin map[string]int
}

func NewVmEstimator(builtin map[string]int) *VmEstimator {
	return &VmEstimator{
		builtin: builtin,
	}
}

func (a VmEstimator) Ref() int {
	return 1
}

func (a VmEstimator) Builtin(n string) int {
	return a.builtin[n]
}
