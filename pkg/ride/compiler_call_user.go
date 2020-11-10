package ride

import "fmt"

// Function call
type CallUserState struct {
	prev Fsm
	params
	name string
	argc uint16
	// positions of arguments
	argn []uint16
}

func (a CallUserState) Property(name string) Fsm {
	panic("CallUserState Property")
}

func (a CallUserState) Func(name string, args []string, invoke string) Fsm {
	return funcTransition(a, a.params, name, args, invoke)
}

func (a CallUserState) Bytes(b []byte) Fsm {
	a.argn = append(a.argn, putConstant(a.params, rideBytes(b)))
	a.b.ret()
	return a
}

func (a CallUserState) Condition() Fsm {
	return conditionalTransition(a, a.params)
}

func (a CallUserState) TrueBranch() Fsm {
	panic("Illegal call `TrueBranch` on CallFsm")
}

func (a CallUserState) FalseBranch() Fsm {
	panic("Illegal call `FalseBranch` on CallFsm")
}

func (a CallUserState) String(s string) Fsm {
	a.argn = append(a.argn, putConstant(a.params, rideString(s)))
	a.b.ret()
	return a
}

func (a CallUserState) Boolean(v bool) Fsm {
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
	return &CallUserState{
		prev:   prev,
		params: params,
		name:   name,
		argc:   argc,
	}
}

func (a CallUserState) Assigment(name string) Fsm {
	return assigmentFsmTransition(a, a.params, name)
}

func (a CallUserState) Long(value int64) Fsm {
	a.argn = append(a.argn, putConstant(a.params, rideInt(value)))
	a.b.ret()
	return a
}

func (a CallUserState) Return() Fsm {
	// check user functions
	n, ok := a.r.get(a.name)
	if !ok {
		panic(fmt.Sprintf("user function `%s` not found", a.name))
	}
	a.b.startPos()
	for i, pos := range a.argn {
		a.b.writeByte(OpSetArg)
		uniqid, ok := a.r.get(fmt.Sprintf("%s$%d", a.name, i))
		if !ok {
			panic(fmt.Sprintf("no function param id `%s` stored in references", fmt.Sprintf("%s$%d", a.name, i)))
		}
		a.b.write(encode(uniqid))
		a.b.write(encode(pos))
	}

	a.b.call(n, a.argc)
	return a.prev
}

func (a CallUserState) Call(name string, argc uint16) Fsm {
	return callTransition(a, a.params, name, argc)
}

func (a CallUserState) Reference(name string) Fsm {
	return reference(a, a.params, name)
}
