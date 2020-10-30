package ride

import "fmt"

// Function call
type CallSystemFsm struct {
	prev Fsm
	params
	name string
	argc uint16
	// positions of arguments
	argn []uint16
}

func (a CallSystemFsm) FuncDeclaration(name string, args []string) Fsm {
	return funcDeclarationFsmTransition(a, a.params, name, args)
}

func (a CallSystemFsm) Bytes(b []byte) Fsm {
	return constant(a, a.params, rideBytes(b))
}

func (a CallSystemFsm) Condition() Fsm {
	return conditionalTransition(a, a.params)
}

func (a CallSystemFsm) TrueBranch() Fsm {
	panic("Illegal call `TrueBranch` on CallFsm")
}

func (a CallSystemFsm) FalseBranch() Fsm {
	panic("Illegal call `FalseBranch` on CallFsm")
}

func (a CallSystemFsm) String(s string) Fsm {
	return str(a, a.params, s)
}

func (a CallSystemFsm) Boolean(v bool) Fsm {
	return boolean(a, a.params, v)
}

func callTransition(prev Fsm, params params, name string, argc uint16) Fsm {
	if _, ok := params.r.get(name); ok {
		return newCallUserFsm(prev, params, name, argc)
	}
	return newCallSystemFsm(prev, params, name, argc)
}

func newCallSystemFsm(prev Fsm, params params, name string, argc uint16) Fsm {
	return &CallSystemFsm{
		prev:   prev,
		params: params,
		name:   name,
		argc:   argc,
	}
}

func (a CallSystemFsm) Assigment(name string) Fsm {
	return assigmentFsmTransition(a, a.params, name)
}

func (a CallSystemFsm) Long(value int64) Fsm {
	index := a.params.c.put(rideInt(value))
	a.params.b.push(index)
	return a
}

func (a CallSystemFsm) Return() Fsm {

	//// check user functions
	//n, ok := a.r.get(a.name)
	//if ok {
	//	a.b.startPos()
	//	for _, pos := range a.argn {
	//		a.b.writeByte(OpPushArg)
	//		a.b.write(encode(pos))
	//		//a.b.writeByte(OpReturn)
	//	}
	//
	//	a.b.call(n, a.argc)
	//	return a.prev
	//}

	//a.b.startPos()
	n, ok := a.f(a.name)
	if !ok {
		panic(fmt.Sprintf("system function named `%s` not found", a.name))
	}
	//for _, pos := range a.argn {
	//	a.b.writeByte(OpJump)
	//	a.b.write(encode(pos))
	//}
	a.b.externalCall(n, a.argc)
	return a.prev
}

func (a CallSystemFsm) Call(name string, argc uint16) Fsm {
	return callTransition(a, a.params, name, argc)
}

func (a CallSystemFsm) Reference(name string) Fsm {
	return reference(a, a.params, name)
}
