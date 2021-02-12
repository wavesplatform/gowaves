package ride

import "fmt"

// Function call
type CallUserState struct {
	prev Fsm
	params
	name      string
	argc      uint16
	startedAt uint16

	deferreds Deferreds
	ns        []uniqueid
}

func (a CallUserState) backward(state Fsm) Fsm {
	num := len(a.ns)
	n := a.params.u.next()
	a.ns = append(a.ns, n)
	a.deferreds.Add(state.(Deferred), n, fmt.Sprintf("call user %s backward #%d", a.name, num))
	return a
}

func newCallUserFsm(prev Fsm, params params, name string, argc uint16, d Deferreds) Fsm {
	return &CallUserState{
		prev:      prev,
		params:    params,
		name:      name,
		argc:      argc,
		startedAt: params.b.len(),
		deferreds: d,
	}
}

func (a CallUserState) Property(name string) Fsm {
	return propertyTransition(a, a.params, name, a.deferreds)
}

func (a CallUserState) Func(name string, args []string, invoke string) Fsm {
	return funcTransition(a, a.params, name, args, invoke)
}

func (a CallUserState) Bytes(b []byte) Fsm {
	cons := a.constant(rideBytes(b))
	a.ns = append(a.ns, cons.n)
	return a
}

func (a CallUserState) Condition() Fsm {
	return conditionalTransition(a, a.params, a.deferreds)
}

func (a CallUserState) TrueBranch() Fsm {
	panic("Illegal call `TrueBranch` on CallFsm")
}

func (a CallUserState) FalseBranch() Fsm {
	panic("Illegal call `FalseBranch` on CallFsm")
}

func (a CallUserState) String(s string) Fsm {
	cons := a.constant(rideString(s))
	a.ns = append(a.ns, cons.n)
	return a
}

func (a CallUserState) Boolean(v bool) Fsm {
	cons := a.constant(rideBoolean(v))
	a.ns = append(a.ns, cons.n)
	return a
}

func (a CallUserState) Assigment(name string) Fsm {
	//return assigmentFsmTransition(a, a.params, name)
	panic("CallUserState Assigment")
}

func (a CallUserState) Long(value int64) Fsm {
	cons := a.constant(rideInt(value))
	a.ns = append(a.ns, cons.n)
	return a
}

func (a CallUserState) Return() Fsm {
	return a.prev.backward(a)
}

func (a CallUserState) Call(name string, argc uint16) Fsm {
	return callTransition(a, a.params, name, argc, a.deferreds)
}

func (a CallUserState) Reference(name string) Fsm {
	cons := reference(a, a.params, name)
	a.ns = append(a.ns, cons.n)
	return a
}

func (a CallUserState) Clean() {

}

func (a CallUserState) Write(_ params, b []byte) {
	// check user functions
	fn, ok := a.r.getFunc(a.name)
	if !ok {
		panic(fmt.Sprintf("user function `%s` not found", a.name))
	}
	if int(a.argc) != len(a.ns) {
		panic(fmt.Sprintf("argc %d != a.ns %d", a.argc, len(a.ns)))
	}
	for _, n := range a.ns {
		a.b.writeByte(OpRef)
		a.b.write(encode(n))
	}
	a.b.writeByte(OpRef)
	a.b.write(encode(fn))
	a.b.write(b)
}
