package ride

import "fmt"

// Assigment: let x = 5
type AssigmentState struct {
	params
	bodyParams params
	prev       Fsm
	name       string
	// ref id
	n uniqueid

	// Clean internal assigments.
	body Deferred
	d    Deferreds
}

func (a AssigmentState) backward(state Fsm) Fsm {
	a.body = state.(Deferred)
	return a
}

func (a AssigmentState) Property(name string) Fsm {
	return propertyTransition(a, a.bodyParams, name, a.d)
}

func (a AssigmentState) Func(name string, args []string, invoke string) Fsm {
	return funcTransition(a, a.bodyParams, name, args, invoke)
}

func (a AssigmentState) Bytes(b []byte) Fsm {
	a.body = a.constant(rideBytes(b))
	return a
}

func (a AssigmentState) Condition() Fsm {
	return conditionalTransition(a, a.bodyParams, a.d)
}

func (a AssigmentState) TrueBranch() Fsm {
	panic("Illegal call `TrueBranch` on AssigmentState")
}

func (a AssigmentState) FalseBranch() Fsm {
	panic("Illegal call `FalseBranch` on AssigmentState")
}

func (a AssigmentState) String(s string) Fsm {
	a.body = a.constant(rideString(s))
	return a
}

func (a AssigmentState) Boolean(v bool) Fsm {
	a.body = a.constant(rideBoolean(v))
	return a
}

func assigmentFsmTransition(prev Fsm, params params, name string, n uniqueid, d Deferreds) Fsm {
	params.r.setAssigment(name, n)
	return newAssigmentFsm(prev, params, name, n, d)
}

func extendParams(p params) params {
	p.r = newReferences(p.r)
	return p
}

func newAssigmentFsm(prev Fsm, p params, name string, n uniqueid, d Deferreds) Fsm {
	return AssigmentState{
		prev:       prev,
		params:     p,
		bodyParams: extendParams(p),
		name:       name,
		n:          n,
		d:          d,
	}
}

// Create new scope, so assigment in assigment can't affect global state.
func (a AssigmentState) Assigment(name string) Fsm {
	//params := a.params
	//params.r = newReferences(params.r)
	// TODO clear var in var
	n := a.params.u.next()
	return assigmentFsmTransition(a, a.bodyParams, name, n, a.d)
}

func (a AssigmentState) Return() Fsm {
	a.r.setAssigment(a.name, a.n)
	a.d.Add(a, a.n, fmt.Sprintf("ref %s", a.name))
	return a.prev
}

func (a AssigmentState) Long(value int64) Fsm {
	a.body = a.constant(rideInt(value))
	return a
}

func (a AssigmentState) Call(name string, argc uint16) Fsm {
	return callTransition(a, a.bodyParams, name, argc, a.d)
}

func (a AssigmentState) Reference(name string) Fsm {
	a.body = reference(a, a.bodyParams, name)
	return a
}

func (a AssigmentState) Write(_ params, b []byte) {
	if a.body == nil {
		panic("no body for assigment")
	}
	a.body.Write(a.params, nil)
	a.b.writeByte(OpCache)
	a.b.write(encode(a.n))
	a.b.ret()
}

func (a AssigmentState) Clean() {
	a.b.writeByte(OpClearCache)
	a.b.write(encode(a.n))
}
