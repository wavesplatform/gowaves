package ride

type PropertyState struct {
	prev Fsm
	name string
	params
}

func propertyTransition(prev Fsm, params params, name string) Fsm {
	return &PropertyState{
		params: params,
		prev:   prev,
		name:   name,
	}
}

func (a PropertyState) Assigment(name string) Fsm {
	panic("Illegal call `Assigment` on PropertyState")
}

func (a PropertyState) Return() Fsm {
	a.b.writeByte(OpProperty)
	index := a.params.c.put(rideString(a.name))
	a.params.b.write(encode(index))
	return a.prev
}

func (a PropertyState) Long(value int64) Fsm {
	panic("Illegal call `Long` on PropertyState")
}

func (a PropertyState) Call(name string, argc uint16) Fsm {
	panic("Illegal call `Call` on PropertyState")
}

func (a PropertyState) Reference(name string) Fsm {
	return reference(a, a.params, name)
}

func (a PropertyState) Boolean(v bool) Fsm {
	panic("Illegal call `Boolean` on PropertyState")
}

func (a PropertyState) String(s string) Fsm {
	panic("Illegal call `String` on PropertyState")
}

func (a PropertyState) Condition() Fsm {
	panic("Illegal call `Condition` on PropertyState")
}

func (a PropertyState) TrueBranch() Fsm {
	panic("Illegal call `TrueBranch` on PropertyState")
}

func (a PropertyState) FalseBranch() Fsm {
	panic("Illegal call `FalseBranch` on PropertyState")
}

func (a PropertyState) Bytes(b []byte) Fsm {
	return bts(a, a.params, b)
}

func (a PropertyState) Func(name string, args []string, invoke string) Fsm {
	panic("Illegal call `Func` on PropertyState")
}

func (a PropertyState) Property(name string) Fsm {
	panic("Illegal call `Property` on PropertyState")
}
