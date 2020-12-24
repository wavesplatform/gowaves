package ride

import "fmt"

type PropertyState struct {
	prev Fsm
	name string
	params
	body      Deferred
	deferreds Deferreds
	n         uniqueid
}

func (a PropertyState) backward(as Fsm) Fsm {
	a.body = as
	return a
}

func propertyTransition(prev Fsm, params params, name string, d Deferreds) Fsm {
	return &PropertyState{
		params:    params,
		prev:      prev,
		name:      name,
		deferreds: d,
	}
}

func (a PropertyState) Assigment(name string) Fsm {
	panic("Illegal call `Assigment` on PropertyState")
}

func (a PropertyState) Return() Fsm {
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

func (a PropertyState) Long(value int64) Fsm {
	panic("Illegal call `Long` on PropertyState")
}

func (a PropertyState) Call(name string, argc uint16) Fsm {
	return callTransition(a, a.params, name, argc, a.deferreds)
}

func (a PropertyState) Reference(name string) Fsm {
	a.body = reference(a, a.params, name)
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
	panic("PropertyState Bytes")
}

func (a PropertyState) Func(name string, args []string, invoke string) Fsm {
	panic("Illegal call `Func` on PropertyState")
}

func (a PropertyState) Property(name string) Fsm {
	return propertyTransition(a, a.params, name, a.deferreds)
}

func (a PropertyState) Clean() {

}

func (a PropertyState) Write(_ params, b []byte) {
	a.b.writeByte(OpRef)
	a.b.write(encode(a.n))
	next := a.u.next()
	a.c.set(next, rideString(a.name), 0, 0, true, fmt.Sprintf("property?? %s", a.name))
	a.b.writeByte(OpRef)
	a.b.write(encode(next))
	a.b.writeByte(OpProperty)
	a.b.write(b)
	a.b.ret()
}
