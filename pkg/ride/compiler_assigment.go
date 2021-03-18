package ride

import "fmt"

// Assigment: let x = 5
type AssigmentState struct {
	params
	bodyParams params
	prev       State
	name       string
	// ref id
	n uniqueid

	// Clean internal assigments.
	body Deferred
	d    Deferreds
}

func (a AssigmentState) backward(state State) State {
	a.body = state.(Deferred)
	return a
}

func (a AssigmentState) Property(name string) State {
	return propertyTransition(a, a.bodyParams, name, a.d)
}

func (a AssigmentState) Func(name string, args []string, invoke string) State {
	return funcTransition(a, a.bodyParams, name, args, invoke)
}

func (a AssigmentState) Bytes(b []byte) State {
	a.body = a.constant(rideBytes(b))
	return a
}

func (a AssigmentState) Condition() State {
	return conditionalTransition(a, a.bodyParams, a.d)
}

func (a AssigmentState) TrueBranch() State {
	panic("Illegal call `TrueBranch` on AssigmentState")
}

func (a AssigmentState) FalseBranch() State {
	panic("Illegal call `FalseBranch` on AssigmentState")
}

func (a AssigmentState) String(s string) State {
	a.body = a.constant(rideString(s))
	return a
}

func (a AssigmentState) Boolean(v bool) State {
	a.body = a.constant(rideBoolean(v))
	return a
}

func assigmentFsmTransition(prev State, params params, name string, n uniqueid, d Deferreds) State {
	return newAssigmentFsm(prev, params, name, n, d)
}

func extendParams(p params) params {
	p.r = newReferences(p.r)
	return p
}

func newAssigmentFsm(prev State, p params, name string, n uniqueid, d Deferreds) State {
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
func (a AssigmentState) Assigment(name string) State {
	//params := a.params
	//params.r = newReferences(params.r)
	// TODO clear var in var
	n := a.params.u.next()
	return assigmentFsmTransition(a, a.bodyParams, name, n, a.d)
}

func (a AssigmentState) Return() State {
	a.r.setAssigment(a.name, a.n)
	a.d.Add(a, a.n, fmt.Sprintf("ref %s", a.name))
	return a.prev
}

func (a AssigmentState) Long(value int64) State {
	a.body = a.constant(rideInt(value))
	return a
}

func (a AssigmentState) Call(name string, argc uint16) State {
	return callTransition(a, a.bodyParams, name, argc, a.d)
}

func (a AssigmentState) Reference(name string) State {
	a.body = reference(a, a.bodyParams, name)
	return a
}

func (a AssigmentState) Write(_ params, _ []byte) {
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
