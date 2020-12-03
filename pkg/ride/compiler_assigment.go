package ride

// Assigment: let x = 5
type AssigmentState struct {
	params
	prev      Fsm
	name      string
	startedAt uint16
	//ret       uint16
	constant rideType
	n        uniqueid
}

func (a AssigmentState) retAssigment(startedAt uint16, endedAt uint16) Fsm {
	//a.ret = pos
	return a
}

func (a AssigmentState) Property(name string) Fsm {
	panic("AssigmentState Property")
}

func (a AssigmentState) Func(name string, args []string, invoke string) Fsm {
	return funcTransition(a, a.params, name, args, invoke)
}

func (a AssigmentState) Bytes(b []byte) Fsm {
	a.constant = rideBytes(b)
	return a
}

func (a AssigmentState) Condition() Fsm {
	return conditionalTransition(a, a.params)
}

func (a AssigmentState) TrueBranch() Fsm {
	panic("Illegal call `TrueBranch` on AssigmentState")
}

func (a AssigmentState) FalseBranch() Fsm {
	panic("Illegal call `FalseBranch` on AssigmentState")
}

func (a AssigmentState) String(s string) Fsm {
	a.constant = rideString(s)
	return a
}

func (a AssigmentState) Boolean(v bool) Fsm {
	a.constant = rideBoolean(v)
	return a
}

func assigmentFsmTransition(prev Fsm, params params, name string) Fsm {
	return newAssigmentFsm(prev, params, name)
}

func newAssigmentFsm(prev Fsm, p params, name string) Fsm {
	return AssigmentState{
		prev:      prev,
		params:    p,
		name:      name,
		startedAt: p.b.len(),
		n:         p.u.next(),
	}
}

// Create new scope, so assigment in assigment can't affect global state.
func (a AssigmentState) Assigment(name string) Fsm {
	params := a.params
	params.r = newReferences(params.r)
	return assigmentFsmTransition(a, params, name)
}

func (a AssigmentState) Return() Fsm {
	a.b.writeByte(OpCache)
	a.b.write(encode(a.n))
	a.b.ret()
	// store reference on variable and it's offset.

	if a.constant != nil {
		a.c.set(a.n, a.constant, nil, 0, a.name)
	} else {
		a.c.set(a.n, nil, nil, a.startedAt, a.name)
	}
	a.r.set(a.name, a.n)
	return a.prev.retAssigment(a.startedAt, a.params.b.len())
}

func (a AssigmentState) Long(value int64) Fsm {
	a.constant = rideInt(value)
	return a
}

func (a AssigmentState) Call(name string, argc uint16) Fsm {
	return callTransition(a, a.params, name, argc)
}

func (a AssigmentState) Reference(name string) Fsm {
	return reference(a, a.params, name)
}
