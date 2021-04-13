package ride

import "fmt"

type PropertyState struct {
	prev State
	name string
	params
	body      Deferred
	deferreds Deferreds
	n         uniqueID
}

func (a PropertyState) backward(as State) State {
	a.body = as
	return a
}

func propertyTransition(prev State, params params, name string, d Deferreds) State {
	return &PropertyState{
		params:    params,
		prev:      prev,
		name:      name,
		deferreds: d,
	}
}

func (a PropertyState) Assigment(name string) State {
	n := a.params.u.next()
	return assigmentTransition(a, a.params, name, n, a.deferreds)
	//panic(fmt.Sprintf("Illegal call `Assigment` on PropertyState (n=%d; prop=%s; assignment=%s)", a.n, a.name, s))
}

func (a PropertyState) Return() State {
	// 2 possible variations:
	// 1) tx.id => body is reference
	// 2) tx.sellOrder.assetPair => body is another property
	a.n = a.params.u.next()
	if n, ok := isConstant(a.body); ok { // body is reference
		a.n = n
	} else { // body is another property
		a.n = a.u.next()
		a.deferreds.Add(a.body, a.n, fmt.Sprintf("property== `%s`", a.name))
	}
	return a.prev.backward(a)
}

func (a PropertyState) Long(int64) State {
	panic("Illegal call `Long` on PropertyState")
}

func (a PropertyState) Call(name string, argc uint16) State {
	return callTransition(a, a.params, name, argc, a.deferreds)
}

func (a PropertyState) Reference(name string) State {
	a.body = reference(a, a.params, name)
	return a
}

func (a PropertyState) Boolean(bool) State {
	panic("Illegal call `Boolean` on PropertyState")
}

func (a PropertyState) String(string) State {
	panic("Illegal call `String` on PropertyState")
}

func (a PropertyState) Condition() State {
	return conditionalTransition(a, a.params, a.deferreds)
	//panic("Illegal call `Condition` on PropertyState")
}

func (a PropertyState) TrueBranch() State {
	panic("Illegal call `TrueBranch` on PropertyState")
}

func (a PropertyState) FalseBranch() State {
	panic("Illegal call `FalseBranch` on PropertyState")
}

func (a PropertyState) Bytes(b []byte) State {
	panic("PropertyState Bytes")
}

func (a PropertyState) Func(_ string, _ []string, _ string) State {
	panic("Illegal call `Func` on PropertyState")
}

func (a PropertyState) Property(name string) State {
	return propertyTransition(a, a.params, name, a.deferreds)
}

func (a PropertyState) Clean() {

}

func (a PropertyState) Write(_ params, b []byte) {
	a.b.writeByte(OpRef)
	a.b.write(encode(a.n))
	next := a.u.next()
	a.c.set(next, rideString(a.name), 0, 0, fmt.Sprintf("property?? %s", a.name))
	a.b.writeByte(OpRef)
	a.b.write(encode(next))
	a.b.writeByte(OpProperty)
	a.b.write(b)
	a.b.ret()
}
