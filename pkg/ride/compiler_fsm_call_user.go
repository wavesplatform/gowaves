package ride

import "fmt"

// Function call
type CallUserFsm struct {
	prev Fsm
	params
	name string
	argc uint16
	// positions of arguments
	argn []uint16
}

func (a CallUserFsm) Property(name string) Fsm {
	panic("CallUserFsm Property")
}

func (a CallUserFsm) FuncDeclaration(name string, args []string) Fsm {
	return funcDeclarationFsmTransition(a, a.params, name, args)
}

func (a CallUserFsm) Bytes(b []byte) Fsm {
	return constant(a, a.params, rideBytes(b))
}

func (a CallUserFsm) Condition() Fsm {
	return conditionalTransition(a, a.params)
}

func (a CallUserFsm) TrueBranch() Fsm {
	panic("Illegal call `TrueBranch` on CallFsm")
}

func (a CallUserFsm) FalseBranch() Fsm {
	panic("Illegal call `FalseBranch` on CallFsm")
}

func (a CallUserFsm) String(s string) Fsm {
	return str(a, a.params, s)
}

func (a CallUserFsm) Boolean(v bool) Fsm {
	pos := a.b.len()
	a.b.writeByte(OpTrue)
	a.argn = append(a.argn, pos)
	a.b.ret()
	return a
}

//func callTransition(prev Fsm, params params, name string, argc uint16) Fsm {
//	return newCallFsm(prev, params, name, argc)
//}

func newCallUserFsm(prev Fsm, params params, name string, argc uint16) Fsm {
	return &CallUserFsm{
		prev:   prev,
		params: params,
		name:   name,
		argc:   argc,
	}
}

func (a CallUserFsm) Assigment(name string) Fsm {
	return assigmentFsmTransition(a, a.params, name)
}

func (a CallUserFsm) Long(value int64) Fsm {
	pos := a.b.len()
	index := a.params.c.put(rideInt(value))
	a.params.b.push(index)
	a.argn = append(a.argn, pos)
	a.b.ret()
	return a
}

func (a CallUserFsm) Return() Fsm {

	// check user functions
	n, ok := a.r.get(a.name)
	if !ok {
		panic(fmt.Sprintf("user function `%s` not found", a.name))
	}
	//if ok {
	a.b.startPos()
	for i, pos := range a.argn {
		a.b.writeByte(OpPushArg)
		uniqid, ok := a.r.get(fmt.Sprintf("%s$%d", a.name, i))
		if !ok {
			panic(fmt.Sprintf("no function param id `%s` stored in references", fmt.Sprintf("%s$%d", a.name, i)))
		}
		a.b.write(encode(uniqid))
		a.b.write(encode(pos))
		//a.b.writeByte(OpReturn)
	}

	a.b.call(n, a.argc)
	return a.prev
	//}

	//a.b.startPos()
	//n, ok = a.f(a.name)
	//if !ok {
	//	panic(fmt.Sprintf("function named %s not found", a.name))
	//}
	//for _, pos := range a.argn {
	//	a.b.writeByte(OpJump)
	//	a.b.write(encode(pos))
	//}
	//a.b.externalCall(n, a.argc)
	//return a.prev
}

func (a CallUserFsm) Call(name string, argc uint16) Fsm {
	return callTransition(a, a.params, name, argc)
}

func (a CallUserFsm) Reference(name string) Fsm {
	return reference(a, a.params, name)
}
