package ride

import "fmt"

// Function call
type CallSystemState struct {
	prev State
	params
	name      string
	argc      uint16
	deferred  []Deferred
	deferreds Deferreds
	// Sequential function arguments.
	ns []uniqueID
}

func (a CallSystemState) backward(state State) State {
	a.deferred = append(a.deferred, state.(Deferred))
	return a
}

func (a CallSystemState) Property(name string) State {
	return propertyTransition(a, a.params, name, a.deferreds)
}

func (a CallSystemState) Func(name string, args []string, invoke string) State {
	return funcTransition(a, a.params, name, args, invoke)
}

func (a CallSystemState) Bytes(value []byte) State {
	a.deferred = append(a.deferred, a.constant(rideBytes(value)))
	return a
}

func (a CallSystemState) Condition() State {
	return conditionalTransition(a, a.params, a.deferreds)
}

func (a CallSystemState) TrueBranch() State {
	panic("Illegal call `TrueBranch` on CallSystemState")
}

func (a CallSystemState) FalseBranch() State {
	panic("Illegal call `FalseBranch` on CallSystemState")
}

func (a CallSystemState) String(value string) State {
	a.deferred = append(a.deferred, a.constant(rideString(value)))
	return a
}

func (a CallSystemState) Boolean(value bool) State {
	a.deferred = append(a.deferred, a.constant(rideBoolean(value)))
	return a
}

func callTransition(prev State, params params, name string, argc uint16, d Deferreds) State {
	if _, ok := params.r.getFunc(name); ok {
		return newCallUserState(prev, params, name, argc, d)
	}
	return newCallSystemState(prev, params, name, argc, d)
}

func newCallSystemState(prev State, params params, name string, argc uint16, d Deferreds) State {
	var ns []uniqueID
	for i := uint16(0); i < argc; i++ {
		ns = append(ns, params.u.next())
	}

	return &CallSystemState{
		prev:      prev,
		params:    params,
		name:      name,
		argc:      argc,
		deferreds: d,
		ns:        ns,
	}
}

func (a CallSystemState) Assigment(name string) State {
	n := a.params.u.next()
	return assigmentTransition(a, a.params, name, n, a.deferreds)
}

func (a CallSystemState) Long(value int64) State {
	a.deferred = append(a.deferred, a.constant(rideInt(value)))
	return a
}

func (a CallSystemState) Return() State {

	if len(a.ns) != len(a.deferred) {
		panic(fmt.Sprintf("ns %d != a.deferred %d", a.argc, len(a.deferred)))
	}

	for i, b := range a.deferred {
		if _, ok := isConstant(b); ok {
			// skip right now
		} else {
			a.deferreds.Add(b, a.ns[i], fmt.Sprintf("sys %s param #%d", a.name, i))
		}
	}

	return a.prev.backward(a)
}

func (a CallSystemState) Call(name string, argc uint16) State {
	return callTransition(a, a.params, name, argc, a.deferreds)
}

func (a CallSystemState) Reference(name string) State {
	a.deferred = append(a.deferred, reference(a, a.params, name))
	return a
}

func (a CallSystemState) Clean() {

}

func (a CallSystemState) Write(_ params, _ []byte) {
	if int(a.argc) != len(a.deferred) {
		panic(fmt.Sprintf("argc %d != a.deferred %d", a.argc, len(a.deferred)))
	}

	for i := range a.ns {
		if n, ok := isConstant(a.deferred[i]); ok {
			a.b.writeByte(OpRef)
			a.b.write(encode(n))
		} else {
			n := a.ns[i]
			a.b.writeByte(OpRef)
			a.b.write(encode(n))
		}
	}

	n, ok := a.f(a.name)
	if !ok {
		panic(fmt.Sprintf("%s system function named `%s` not found", a.params.txID, a.name))
	}
	a.b.externalCall(n, a.argc)
}
