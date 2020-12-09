package ride

import "fmt"

// Assigment: let x = 5
type AssigmentState struct {
	params
	prev Fsm
	name string
	//startedAt uint16
	//ret       uint16
	//constant rideType
	// ref id
	n uniqueid

	// Clean internal assigments.
	deferred []Deferred
	d        Deferreds
}

func (a AssigmentState) retAssigment(state Fsm) Fsm {
	a.deferred = append(a.deferred, state.(Deferred))
	return a
}

func (a AssigmentState) Property(name string) Fsm {
	panic("AssigmentState Property")
}

func (a AssigmentState) Func(name string, args []string, invoke string) Fsm {
	return funcTransition(a, a.params, name, args, invoke)
}

func (a AssigmentState) Bytes(b []byte) Fsm {
	a.deferred = append(a.deferred, a.constant(rideBytes(b)))
	return a
}

func (a AssigmentState) Condition() Fsm {
	return conditionalTransition(a, a.params, a.d)
}

func (a AssigmentState) TrueBranch() Fsm {
	panic("Illegal call `TrueBranch` on AssigmentState")
}

func (a AssigmentState) FalseBranch() Fsm {
	panic("Illegal call `FalseBranch` on AssigmentState")
}

func (a AssigmentState) String(s string) Fsm {
	a.deferred = append(a.deferred, a.constant(rideString(s)))
	return a
}

func (a AssigmentState) Boolean(v bool) Fsm {
	a.deferred = append(a.deferred, a.constant(rideBoolean(v)))
	return a
}

func assigmentFsmTransition(prev Fsm, params params, name string, n uniqueid, d Deferreds) Fsm {
	params.r.set(name, n)
	return newAssigmentFsm(prev, params, name, n, d)
}

func newAssigmentFsm(prev Fsm, p params, name string, n uniqueid, d Deferreds) Fsm {
	return AssigmentState{
		prev:   prev,
		params: p,
		name:   name,
		n:      n,
		d:      d,
	}
}

// Create new scope, so assigment in assigment can't affect global state.
func (a AssigmentState) Assigment(name string) Fsm {
	params := a.params
	params.r = newReferences(params.r)
	// TODO clear var in var
	n := a.params.u.next()
	return assigmentFsmTransition(a, params, name, n, a.d)
}

func (a AssigmentState) Return() Fsm {
	//for i := len(a.assigments) - 1; i >= 0; i-- {
	//	a.b.writeByte(OpClearCache)
	//	a.b.write(encode(a.assigments[i]))
	//}
	//// constant
	//if a.constant != nil {
	//	a.c.set(a.n, a.constant, nil, 0, true, a.name)
	//} else {
	//	a.c.set(a.n, nil, nil, a.startedAt, false, a.name)
	//	a.b.writeByte(OpCache)
	//	a.b.write(encode(a.n))
	//	a.b.ret()
	//}
	a.r.set(a.name, a.n)
	a.d.Add(a, a.n, fmt.Sprintf("ref %s", a.name))
	//return a.prev.retAssigment(a.startedAt, a.params.b.len())
	return a.prev //.retAssigment(a)
}

func (a AssigmentState) Long(value int64) Fsm {
	a.deferred = append(a.deferred, a.constant(rideInt(value)))
	return a
}

func (a AssigmentState) Call(name string, argc uint16) Fsm {
	return callTransition(a, a.params, name, argc, a.d)
}

func (a AssigmentState) Reference(name string) Fsm {
	a.deferred = append(a.deferred, reference(a, a.params, name))
	return a
}

func (a AssigmentState) Write(_ params) {
	// constant
	//if a.constant != nil {
	//	a.c.set(a.n, a.constant, nil, 0, true, a.name)
	//} else {
	//a.c.set(a.n, nil, nil, a.b.len(), false, a.name)
	//	a.b.writeByte(OpCache)
	//	a.b.write(encode(a.n))
	//	a.b.ret()
	//}
	//a.r.set(a.name, a.n)

	//for _, v := range a.deferred {
	//	v.(Clean).Clean()
	//}
	//
	//a.b.ret()
	//
	//for _, v := range reverse(a.deferred) {
	//	v.(Write).Write()
	//}

	d := a.deferred

	if len(d) == 0 {
		panic("writeDeferred len == 0")
	}
	d2 := reverse(d)

	d2[0].Write(a.params)

	for _, v := range d2 {
		v.Clean()
	}

	a.b.ret()
	for _, v := range d2[1:] {
		v.Write(a.params)
	}

	//writeDeferred(a.params, a.deferred)

	return //a.prev.retAssigment(a.startedAt, a.params.b.len())
}

func (a AssigmentState) Clean() {
	//for i := len(a.assigments) - 1; i >= 0; i-- {
	a.b.writeByte(OpClearCache)
	a.b.write(encode(a.n))
	//}
}
