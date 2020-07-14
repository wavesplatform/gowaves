package transpiler

type UpAction struct {
}

func NewLift() *lift {
	return &lift{}
}

type lift struct {
	fns []func() Fsm
}

func (a *lift) Up(f func() Fsm) UpAction {
	a.fns = append(a.fns, f)
	return UpAction{}
}

func (a *lift) Down() Fsm {
	last := a.fns[len(a.fns)-1]
	a.fns = a.fns[:len(a.fns)-1]
	return last()
}
