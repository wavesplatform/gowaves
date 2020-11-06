package ride

// If-else statement.
type ConditionalState struct {
	params
	prev          Fsm
	patchPosition uint16
}

func (a ConditionalState) Property(name string) Fsm {
	panic("ConditionalState Property")
}

func (a ConditionalState) FuncDeclaration(name string, args []string) Fsm {
	panic("Illegal call FuncDeclaration on ConditionalState")
}

func (a ConditionalState) Bytes(b []byte) Fsm {
	return constant(a, a.params, rideBytes(b))
}

func conditionalTransition(prev Fsm, params params) Fsm {
	return ConditionalState{
		prev:   prev,
		params: params,
	}
}

func (a ConditionalState) Condition() Fsm {
	return conditionalTransition(a, a.params)
}

func (a ConditionalState) TrueBranch() Fsm {
	a.b.jpmIfFalse()
	a.patchPosition = a.b.writeStub(2)
	return a
}

func (a ConditionalState) FalseBranch() Fsm {
	a.b.ret()
	a.b.patch(a.patchPosition, encode(a.b.len()))
	return a
}

func (a ConditionalState) Assigment(name string) Fsm {
	return assigmentFsmTransition(a, a.params, name)
}

func (a ConditionalState) Return() Fsm {
	a.b.ret()
	return a.prev
}

func (a ConditionalState) Long(value int64) Fsm {
	return long(a, a.params, value)
}

func (a ConditionalState) Call(name string, argc uint16) Fsm {
	return callTransition(a, a.params, name, argc)
}

func (a ConditionalState) Reference(name string) Fsm {
	return reference(a, a.params, name)
}

func (a ConditionalState) Boolean(v bool) Fsm {
	return boolean(a, a.params, v)
}

func (a ConditionalState) String(s string) Fsm {
	return str(a, a.params, s)
}
