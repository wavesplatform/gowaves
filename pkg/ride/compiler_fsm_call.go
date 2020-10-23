package ride

import "fmt"

// Function call
type CallFsm struct {
	prev Fsm
	params
	name string
	argc uint16
}

func (a CallFsm) FuncDeclaration(name string, args []string) Fsm {
	return funcDeclarationFsmTransition(a, a.params, name, args)
}

func (a CallFsm) Bytes(b []byte) Fsm {
	return constant(a, a.params, rideBytes(b))
}

func (a CallFsm) Condition() Fsm {
	return conditionalTransition(a, a.params)
}

func (a CallFsm) TrueBranch() Fsm {
	panic("Illegal call `TrueBranch` on CallFsm")
}

func (a CallFsm) FalseBranch() Fsm {
	panic("Illegal call `FalseBranch` on CallFsm")
}

func (a CallFsm) String(s string) Fsm {
	return str(a, a.params, s)
}

func (a CallFsm) Boolean(v bool) Fsm {
	return boolean(a, a.params, v)
}

func callTransition(prev Fsm, params params, name string, argc uint16) Fsm {
	return newCallFsm(prev, params, name, argc)
}

func newCallFsm(prev Fsm, params params, name string, argc uint16) Fsm {
	return &CallFsm{
		prev:   prev,
		params: params,
		name:   name,
		argc:   argc,
	}
}

func (a CallFsm) Assigment(name string) Fsm {
	return assigmentFsmTransition(a, a.params, name)
}

func (a CallFsm) Long(value int64) Fsm {
	return long(a, a.params, value)
}

func (a CallFsm) Return() Fsm {
	// check user functions
	n, ok := a.r.get(a.name)
	if ok {
		a.b.call(n, a.argc)
		return a.prev
	}

	n, ok = a.f(a.name)
	if !ok {
		panic(fmt.Sprintf("function named %s not found", a.name))
	}
	a.b.externalCall(n, a.argc)
	return a.prev
}

func (a CallFsm) Call(name string, argc uint16) Fsm {
	return callTransition(a, a.params, name, argc)
}

func (a CallFsm) Reference(name string) Fsm {
	return reference(a, a.params, name)
}
