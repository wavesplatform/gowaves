package ride

type PropertyState struct {
	prev Fsm
	name string
	params
	deferred  []Deferred
	deferreds Deferreds
}

func (a PropertyState) retAssigment(as Fsm) Fsm {
	panic("implement me")
	//return a
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
	//a.b.writeByte(OpProperty)
	//index := a.constant(rideString(a.name))
	//a.params.b.write(encode(index))
	//return a.prev.retAssigment(a)
	panic("aaaaa")
}

func (a PropertyState) Long(value int64) Fsm {
	panic("Illegal call `Long` on PropertyState")
}

func (a PropertyState) Call(name string, argc uint16) Fsm {
	return callTransition(a, a.params, name, argc, a.deferreds)
}

func (a PropertyState) Reference(name string) Fsm {
	a.deferred = append(a.deferred, reference(a, a.params, name))
	return a
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
	a.deferred = append(a.deferred, a.constant(rideBytes(b)))
	return a
}

func (a PropertyState) Func(name string, args []string, invoke string) Fsm {
	panic("Illegal call `Func` on PropertyState")
}

func (a PropertyState) Property(name string) Fsm {
	return propertyTransition(a, a.params, name)
}

func (a PropertyState) Clean() {

}

func (a PropertyState) Write(_ params) {
	panic("PropertyState Write")
}
