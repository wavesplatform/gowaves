package ride

// Assigment: let x = 5
type AssigmentFsm struct {
	params
	prev   Fsm
	name   string
	offset uint16
}

func (a AssigmentFsm) Bytes(b []byte) Fsm {
	return constant(a, a.params, rideBytes(b))
}

func (a AssigmentFsm) Condition() Fsm {
	return conditionalTransition(a, a.params)
}

func (a AssigmentFsm) TrueBranch() Fsm {
	panic("Illegal call `TrueBranch` on AssigmentFsm")
}

func (a AssigmentFsm) FalseBranch() Fsm {
	panic("Illegal call `FalseBranch` on AssigmentFsm")
}

func (a AssigmentFsm) String(s string) Fsm {
	return constant(a, a.params, rideString(s))
}

func (a AssigmentFsm) Boolean(v bool) Fsm {
	return boolean(a, a.params, v)
}

func assigmentFsmTransition(prev Fsm, params params, name string) Fsm {
	return newAssigmentFsm(prev, params, name)
}

func newAssigmentFsm(prev Fsm, p params, name string) Fsm {
	return AssigmentFsm{
		prev: prev,
		params: params{
			b: p.b,
			c: p.c,
			f: p.f,
			// Create new scope, so assigment in assigment can't affect global state.
			r: newReferences(p.r),
		},
		name:   name,
		offset: p.b.len(),
	}
}

func (a AssigmentFsm) Assigment(name string) Fsm {
	return assigmentFsmTransition(a, a.params, name)
}

func (a AssigmentFsm) Return() Fsm {
	a.b.ret()
	// store reference on variable and it's offset.
	a.r.set(a.name, a.offset)
	return a.prev
}

func (a AssigmentFsm) Long(value int64) Fsm {
	return long(a, a.params, value)
}

func (a AssigmentFsm) Call(name string, argc uint16) Fsm {
	return callTransition(a, a.params, name, argc)
}

func (a AssigmentFsm) Reference(name string) Fsm {
	return reference(a, a.params, name)
}
