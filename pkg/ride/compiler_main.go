package ride

// Initial state, contains only assigments and last expression.
type MainState struct {
	params
	retAssig uint16

	deferred  []Deferred
	deferreds *deferreds
}

func (a MainState) retAssigment(state Fsm) Fsm {
	a.deferred = append(a.deferred, state.(Deferred))
	return a
}

func (a MainState) Property(name string) Fsm {
	return propertyTransition(a, a.params, name, a.deferreds)
}

func (a MainState) Func(name string, args []string, invoke string) Fsm {
	return funcTransition(a, a.params, name, args, invoke)
}

func (a MainState) Bytes(b []byte) Fsm {
	panic("Illegal call `Bytes` on `MainState`")
}

func (a MainState) Condition() Fsm {
	//a.b.startPos()
	return conditionalTransition(a, a.params, a.deferreds)
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
		deferreds: &deferreds{
			name: "main",
		},
	}
}

func (a MainState) Assigment(name string) Fsm {
	n := a.params.u.next()
	//a.assigments = append(a.assigments, n)
	//a.r.set(name, n)
	return assigmentFsmTransition(a, a.params, name, n, a.deferreds)
}

func (a MainState) Return() Fsm {
	reversed := reverse(a.deferred)

	if f, ok := reversed[0].(FuncState); ok {
		for i := len(f.ParamIds()) - 1; i >= 0; i-- {
			a.b.writeByte(OpCache)
			a.b.write(encode(f.ParamIds()[i]))
			a.b.writeByte(OpPop)
		}
	}

	for _, v := range reversed[:1] {
		v.Write(a.params)
	}
	for _, v := range a.deferreds.Get() {
		v.deferred.Clean()
	}
	a.b.ret()
	for _, v := range reversed[1:] {
		v.Write(a.params)
		a.b.ret()
	}

	for _, v := range a.deferreds.Get() {
		pos := a.b.len()
		a.c.set(v.uniq, nil, nil, pos, false, v.debug)
		v.deferred.Write(a.params)
		a.b.ret()
	}
	return a
}

func (a MainState) Long(int64) Fsm {
	panic("Illegal call Long on MainState")
}

func (a MainState) Call(name string, argc uint16) Fsm {
	return callTransition(a, a.params, name, argc, a.deferreds)
}

func (a MainState) Reference(name string) Fsm {
	a.deferred = append(a.deferred, reference(a, a.params, name))
	return a
}

func (a MainState) Boolean(v bool) Fsm {
	a.deferred = append(a.deferred, a.constant(rideBoolean(v)))
	return a
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

func (a MainState) Write(_ params) {

}

func (a MainState) Clean() {

}
