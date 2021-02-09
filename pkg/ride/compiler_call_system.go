package ride

import "fmt"

// Function call
type CallSystemState struct {
	prev Fsm
	params
	name string
	argc uint16
	// positions of arguments
	//argn []uint16
	// Position where we started write code for current state.
	startedAt uint16
	//retAssig  uint16
	deferred  []Deferred
	deferreds Deferreds
	// Sequential function arguments.
	ns []uniqueid
}

func (a CallSystemState) backward(state Fsm) Fsm {
	a.deferred = append(a.deferred, state.(Deferred))
	return a
}

func (a CallSystemState) Property(name string) Fsm {
	return propertyTransition(a, a.params, name, a.deferreds)
}

func (a CallSystemState) Func(name string, args []string, invoke string) Fsm {
	return funcTransition(a, a.params, name, args, invoke)
}

func (a CallSystemState) Bytes(value []byte) Fsm {
	a.deferred = append(a.deferred, a.constant(rideBytes(value)))
	return a
}

func (a CallSystemState) Condition() Fsm {
	return conditionalTransition(a, a.params, a.deferreds)
}

func (a CallSystemState) TrueBranch() Fsm {
	panic("Illegal call `TrueBranch` on CallFsm")
}

func (a CallSystemState) FalseBranch() Fsm {
	panic("Illegal call `FalseBranch` on CallFsm")
}

func (a CallSystemState) String(value string) Fsm {
	a.deferred = append(a.deferred, a.constant(rideString(value)))
	return a
}

func (a CallSystemState) Boolean(value bool) Fsm {
	a.deferred = append(a.deferred, a.constant(rideBoolean(value)))
	return a
}

func callTransition(prev Fsm, params params, name string, argc uint16, d Deferreds) Fsm {
	if _, ok := params.r.getFunc(name); ok {
		return newCallUserFsm(prev, params, name, argc, d)
	}
	return newCallSystemFsm(prev, params, name, argc, d)
}

func newCallSystemFsm(prev Fsm, params params, name string, argc uint16, d Deferreds) Fsm {
	var ns []uniqueid
	for i := uint16(0); i < argc; i++ {
		ns = append(ns, params.u.next())
	}

	return &CallSystemState{
		prev:      prev,
		params:    params,
		name:      name,
		argc:      argc,
		startedAt: params.b.len(),
		deferreds: d,
		ns:        ns,
	}
}

func (a CallSystemState) Assigment(name string) Fsm {
	n := a.params.u.next()
	return assigmentFsmTransition(a, a.params, name, n, a.deferreds)
}

func (a CallSystemState) Long(value int64) Fsm {
	a.deferred = append(a.deferred, a.constant(rideInt(value)))
	return a
}

func (a CallSystemState) Return() Fsm {

	if len(a.ns) != len(a.deferred) {
		panic(fmt.Sprintf("ns %d != a.deferred %d", a.argc, len(a.deferred)))
	}

	for i, b := range a.deferred {
		if _, ok := isConstant(b); ok {
			// skip right now
		} else {
			a.deferreds.Add(b, a.ns[i], fmt.Sprintf("sys %s param #%d", a.name, i))
			//a.c.set(ns[i], nil, nil, a.b.len(), false, fmt.Sprintf("sys %s param #%d", a.name, i))
			//b.Write(a.params)
			//a.b.ret()
		}
	}

	return a.prev.backward(a)
}

func (a CallSystemState) Call(name string, argc uint16) Fsm {
	return callTransition(a, a.params, name, argc, a.deferreds)
}

func (a CallSystemState) Reference(name string) Fsm {
	a.deferred = append(a.deferred, reference(a, a.params, name))
	return a
}

func (a CallSystemState) Clean() {

}

func (a CallSystemState) Write(_ params, b []byte) {
	if int(a.argc) != len(a.deferred) {
		panic(fmt.Sprintf("argc %d != a.deferred %d", a.argc, len(a.deferred)))
	}

	for i := range a.ns {
		if n, ok := isConstant(a.deferred[i]); ok {
			a.b.writeByte(OpRef)
			a.b.write(encode(n))
			a.b.writeByte(OpCache)
			a.b.write(encode(n))
		} else {
			n := a.ns[i]
			a.b.writeByte(OpRef)
			a.b.write(encode(n))
			//a.b.writeByte(OpCache)
			//a.b.write(encode(n))
		}
	}

	n, ok := a.f(a.name)
	if !ok {
		panic(fmt.Sprintf("system function named `%s` not found", a.name))
	}
	a.b.externalCall(n, a.argc)
}
