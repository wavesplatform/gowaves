package ride

// Assigment: let x = 5
type AssigmentState struct {
	params
	prev   Fsm
	name   string
	offset uint16
	ret    uint16
}

func (a AssigmentState) retAssigment(pos uint16) Fsm {
	a.ret = pos
	return a
}

func (a AssigmentState) Property(name string) Fsm {
	panic("AssigmentState Property")
}

func (a AssigmentState) Func(name string, args []string, invoke string) Fsm {
	return funcTransition(a, a.params, name, args, invoke)
}

func (a AssigmentState) Bytes(b []byte) Fsm {
	return bts(a, a.params, b)
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
	return constant(a, a.params, rideString(s))
}

func (a AssigmentState) Boolean(v bool) Fsm {
	return boolean(a, a.params, v)
}

func assigmentFsmTransition(prev Fsm, params params, name string) Fsm {
	return newAssigmentFsm(prev, params, name)
}

func newAssigmentFsm(prev Fsm, p params, name string) Fsm {
	return AssigmentState{
		prev:   prev,
		params: p,
		name:   name,
		offset: p.b.len(),
	}
}

// Create new scope, so assigment in assigment can't affect global state.
func (a AssigmentState) Assigment(name string) Fsm {
	params := a.params
	params.r = newReferences(params.r)
	return assigmentFsmTransition(a, params, name)
}

func (a AssigmentState) Return() Fsm {
	a.b.ret()
	// store reference on variable and it's offset.
	n := a.u.next()
	a.c.set(n, nil, nil, a.offset)
	a.r.set(a.name, n)
	return a.prev.retAssigment(a.params.b.len())
}

func (a AssigmentState) Long(value int64) Fsm {
	return long(a, a.params, value)
}

func (a AssigmentState) Call(name string, argc uint16) Fsm {
	return callTransition(a, a.params, name, argc)
}

func (a AssigmentState) Reference(name string) Fsm {
	return reference(a, a.params, name)
}
