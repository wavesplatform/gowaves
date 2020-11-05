package ride

// If-else statement.
type ConditionalFsm struct {
	params
	prev          Fsm
	patchPosition uint16
}

func (a ConditionalFsm) Property(name string) Fsm {
	panic("ConditionalFsm Property")
}

func (a ConditionalFsm) FuncDeclaration(name string, args []string) Fsm {
	panic("Illegal call FuncDeclaration on ConditionalFsm")
}

func (a ConditionalFsm) Bytes(b []byte) Fsm {
	return constant(a, a.params, rideBytes(b))
}

func conditionalTransition(prev Fsm, params params) Fsm {
	return ConditionalFsm{
		prev:   prev,
		params: params,
	}
}

func (a ConditionalFsm) Condition() Fsm {
	return conditionalTransition(a, a.params)
}

func (a ConditionalFsm) TrueBranch() Fsm {
	a.b.jpmIfFalse()
	a.patchPosition = a.b.writeStub(2)
	return a
}

func (a ConditionalFsm) FalseBranch() Fsm {
	a.b.ret()
	a.b.patch(a.patchPosition, encode(a.b.len()))
	return a
}

func (a ConditionalFsm) Assigment(name string) Fsm {
	return assigmentFsmTransition(a, a.params, name)
}

func (a ConditionalFsm) Return() Fsm {
	a.b.ret()
	return a.prev
}

func (a ConditionalFsm) Long(value int64) Fsm {
	return long(a, a.params, value)
}

func (a ConditionalFsm) Call(name string, argc uint16) Fsm {
	return callTransition(a, a.params, name, argc)
}

func (a ConditionalFsm) Reference(name string) Fsm {
	return reference(a, a.params, name)
}

func (a ConditionalFsm) Boolean(v bool) Fsm {
	return boolean(a, a.params, v)
}

func (a ConditionalFsm) String(s string) Fsm {
	return str(a, a.params, s)
}
