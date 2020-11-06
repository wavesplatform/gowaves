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
}

func (a CallSystemState) Property(name string) Fsm {
	return propertyTransition(a, a.params, name)
}

func (a CallSystemState) FuncDeclaration(name string, args []string) Fsm {
	return funcDeclarationFsmTransition(a, a.params, name, args)
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
		prev:   prev,
		params: params,
		name:   name,
		argc:   argc,
	}
}

func (a CallSystemState) Assigment(name string) Fsm {
	return assigmentFsmTransition(a, a.params, name)
}

func (a CallSystemState) Long(value int64) Fsm {
	index := a.params.c.put(rideInt(value))
	a.params.b.push(index)
	return a
}

func (a CallSystemState) Return() Fsm {

	//// check user functions
	//n, ok := a.r.get(a.name)
	//if ok {
	//	a.b.startPos()
	//	for _, pos := range a.argn {
	//		a.b.writeByte(OpSetArg)
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
		//if p, ok := a.predef.get(a.name); ok {
		//	a.b.externalCall(p.id, a.argc)
		//	return a.prev
		//}
		panic(fmt.Sprintf("system function named `%s` not found", a.name))
	}
	//for _, pos := range a.argn {
	//	a.b.writeByte(OpJump)
	//	a.b.write(encode(pos))
	//}
	a.b.externalCall(n, a.argc)
	return a.prev
}

func (a CallSystemState) Call(name string, argc uint16) Fsm {
	return callTransition(a, a.params, name, argc)
}

func (a CallSystemState) Reference(name string) Fsm {
	return reference(a, a.params, name)
}
