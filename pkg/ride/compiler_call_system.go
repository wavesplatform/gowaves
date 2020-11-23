package ride

import "fmt"

// Function call
type CallSystemState struct {
	prev Fsm
	params
	name string
	argc uint16
	// positions of arguments
	argn []uint16
	// Position where we started write code for current state.
	startedAt uint16
	retAssig  uint16
}

func (a CallSystemState) retAssigment(pos uint16) Fsm {
	a.retAssig = pos
	return a
}

func (a CallSystemState) Property(name string) Fsm {
	return propertyTransition(a, a.params, name)
}

func (a CallSystemState) Func(name string, args []string, invoke string) Fsm {
	return funcTransition(a, a.params, name, args, invoke)
}

func (a CallSystemState) Bytes(b []byte) Fsm {
	return constant(a, a.params, rideBytes(b))
}

func (a CallSystemState) Condition() Fsm {
	return conditionalTransition(a, a.params)
}

func (a CallSystemState) TrueBranch() Fsm {
	panic("Illegal call `TrueBranch` on CallFsm")
}

func (a CallSystemState) FalseBranch() Fsm {
	panic("Illegal call `FalseBranch` on CallFsm")
}

func (a CallSystemState) String(s string) Fsm {
	return str(a, a.params, s)
}

func (a CallSystemState) Boolean(v bool) Fsm {
	return boolean(a, a.params, v)
}

func callTransition(prev Fsm, params params, name string, argc uint16) Fsm {
	if _, ok := params.r.get(name); ok {
		return newCallUserFsm(prev, params, name, argc)
	}
	return newCallSystemFsm(prev, params, name, argc)
}

func newCallSystemFsm(prev Fsm, params params, name string, argc uint16) Fsm {
	return &CallSystemState{
		prev:      prev,
		params:    params,
		name:      name,
		argc:      argc,
		startedAt: params.b.len(),
	}
}

func (a CallSystemState) Assigment(name string) Fsm {
	return assigmentFsmTransition(a, a.params, name)
}

func (a CallSystemState) Long(value int64) Fsm {
	a.b.push(a.constant(rideInt(value)))
	return a
}

func (a CallSystemState) Return() Fsm {
	n, ok := a.f(a.name)
	if !ok {
		panic(fmt.Sprintf("system function named `%s` not found", a.name))
	}
	a.b.externalCall(n, a.argc)
	return a.prev.retAssigment(a.startedAt)
}

func (a CallSystemState) Call(name string, argc uint16) Fsm {
	return callTransition(a, a.params, name, argc)
}

func (a CallSystemState) Reference(name string) Fsm {
	return reference(a, a.params, name)
}
