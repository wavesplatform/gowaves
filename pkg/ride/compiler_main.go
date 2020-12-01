package ride

// Initial state, contains only assigments and last expression.
type MainState struct {
	params
	retAssig uint16
}

func (a MainState) retAssigment(startedAt uint16, endedAt uint16) Fsm {
	a.retAssig = startedAt
	return a
}

func (a MainState) Property(name string) Fsm {
	return propertyTransition(a, a.params, name)
}

func (a MainState) Func(name string, args []string, invoke string) Fsm {
	return funcTransition(a, a.params, name, args, invoke)
}

func (a MainState) Bytes(b []byte) Fsm {
	panic("Illegal call `Bytes` on `MainState`")
}

func (a MainState) Condition() Fsm {
	a.b.startPos()
	return conditionalTransition(a, a.params)
}

func (a MainState) TrueBranch() Fsm {
	panic("Illegal call `TrueBranch` on MainState")
}

func (a MainState) FalseBranch() Fsm {
	panic("Illegal call `FalseBranch` on MainState")
}

func (a MainState) String(s string) Fsm {
	panic("Illegal call `String` on MainState")
}

type BuildExecutable interface {
	BuildExecutable(version int) *Executable
}

func NewMain(params params) Fsm {
	return &MainState{
		params: params,
	}
}

func (a MainState) Assigment(name string) Fsm {
	return assigmentFsmTransition(a, a.params, name)
}

func (a MainState) Return() Fsm {
	a.b.ret()
	return a
}

func (a MainState) Long(int64) Fsm {
	panic("Illegal call Long on MainState")
}

func (a MainState) Call(name string, argc uint16) Fsm {
	a.b.startPos()
	return callTransition(a, a.params, name, argc)
}

func (a MainState) Reference(name string) Fsm {
	a.b.startPos()
	return reference(a, a.params, name)
}

func (a MainState) Boolean(v bool) Fsm {
	a.b.startPos()
	return boolean(a, a.params, v)
}

func (a MainState) BuildExecutable(version int) *Executable {
	startAt, code := a.b.build()
	return &Executable{
		LibVersion:  version,
		ByteCode:    code,
		References:  a.c.values,
		EntryPoints: map[string]uint16{"": startAt},
	}
}
