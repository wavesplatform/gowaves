package ride

// If-else statement.
type ConditionalState struct {
	params
	prev Fsm
	/*
		Offset where true branch starts execution.
		We need this because code can look like:
		if (true) then {
			let x = throw()
			5
		} else {
			let y = throw()
			6
		}

		`X` and `y` should not be executed.

	*/
	patchTruePosition uint16
	// Same as true position.
	patchFalsePosition uint16
	// Offset where `if` code block ends.
	patchNextPosition uint16
	retAssig          uint16
	startedAt         uint16
	trueStartedAt     uint16
	falseStartedAt    uint16
	rets              []uint16
}

func (a ConditionalState) retAssigment(pos uint16) Fsm {
	a.retAssig = pos
	return a
}

func (a ConditionalState) Property(name string) Fsm {
	panic("ConditionalState Property")
}

func (a ConditionalState) Func(name string, args []string, invoke string) Fsm {
	panic("Illegal call Func on ConditionalState")
}

func (a ConditionalState) Bytes(b []byte) Fsm {
	a.rets = append(a.rets, a.params.b.len())
	return constant(a, a.params, rideBytes(b))
}

func conditionalTransition(prev Fsm, params params) Fsm {
	return ConditionalState{
		prev:      prev,
		params:    params,
		startedAt: params.b.len(),
	}
}

func (a ConditionalState) Condition() Fsm {
	a.rets = append(a.rets, a.params.b.len())
	return conditionalTransition(a, a.params)
}

func (a ConditionalState) TrueBranch() Fsm {
	a.b.jpmIfFalse()
	a.patchTruePosition = a.b.writeStub(2)
	a.patchFalsePosition = a.b.writeStub(2)
	a.patchNextPosition = a.b.writeStub(2)
	return a
}

func (a ConditionalState) FalseBranch() Fsm {
	a.b.ret()
	return a
}

func (a ConditionalState) Assigment(name string) Fsm {
	return assigmentFsmTransition(a, a.params, name)
}

func (a ConditionalState) Return() Fsm {
	a.b.ret()
	a.b.patch(a.patchTruePosition, encode(a.rets[1]))
	a.b.patch(a.patchFalsePosition, encode(a.rets[2]))
	a.b.patch(a.patchNextPosition, encode(a.b.len()))
	return a.prev.retAssigment(a.startedAt)
}

func (a ConditionalState) Long(value int64) Fsm {
	a.rets = append(a.rets, a.params.b.len())
	return long(a, a.params, value)
}

func (a ConditionalState) Call(name string, argc uint16) Fsm {
	a.rets = append(a.rets, a.params.b.len())
	// TODO check if we need ret here
	return callTransition(a, a.params, name, argc)
}

func (a ConditionalState) Reference(name string) Fsm {
	a.rets = append(a.rets, a.params.b.len())
	return reference(a, a.params, name)
}

func (a ConditionalState) Boolean(v bool) Fsm {
	a.rets = append(a.rets, a.params.b.len())
	return boolean(a, a.params, v)
}

func (a ConditionalState) String(s string) Fsm {
	a.rets = append(a.rets, a.params.b.len())
	return str(a, a.params, s)
}
