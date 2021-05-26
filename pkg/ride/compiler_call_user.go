package ride

import "fmt"

// Function call
type CallUserState struct {
	prev State
	params
	name      string
	argc      uint16
	deferreds Deferreds
	ns        []uniqueid
}

func (a CallUserState) backward(state State) State {
	num := len(a.ns)
	n := a.params.u.next()
	a.ns = append(a.ns, n)
	a.deferreds.Add(state.(Deferred), n, fmt.Sprintf("call user %s backward #%d", a.name, num))
	return a
}

func newCallUserState(prev State, params params, name string, argc uint16, d Deferreds) State {
	return &CallUserState{
		prev:      prev,
		params:    params,
		name:      name,
		argc:      argc,
		deferreds: d,
	}
}

func (a CallUserState) Property(name string) State {
	return propertyTransition(a, a.params, name, a.deferreds)
}

func (a CallUserState) Func(name string, args []string, invoke string) State {
	return funcTransition(a, a.params, name, args, invoke)
}

func (a CallUserState) Bytes(b []byte) State {
	cons := a.constant(rideBytes(b))
	a.ns = append(a.ns, cons.n)
	return a
}

func (a CallUserState) Condition() State {
	return conditionalTransition(a, a.params, a.deferreds)
}

func (a CallUserState) TrueBranch() State {
	panic("Illegal call `TrueBranch` on CallUserState")
}

func (a CallUserState) FalseBranch() State {
	panic("Illegal call `FalseBranch` on CallUserState")
}

func (a CallUserState) String(s string) State {
	cons := a.constant(rideString(s))
	a.ns = append(a.ns, cons.n)
	return a
}

func (a CallUserState) Boolean(v bool) State {
	cons := a.constant(rideBoolean(v))
	a.ns = append(a.ns, cons.n)
	return a
}

func (a CallUserState) Assigment(string) State {
	panic("CallUserState Assigment")
}

func (a CallUserState) Long(value int64) State {
	cons := a.constant(rideInt(value))
	a.ns = append(a.ns, cons.n)
	return a
}

func (a CallUserState) Return() State {
	return a.prev.backward(a)
}

func (a CallUserState) Call(name string, argc uint16) State {
	return callTransition(a, a.params, name, argc, a.deferreds)
}

func (a CallUserState) Reference(name string) State {
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
