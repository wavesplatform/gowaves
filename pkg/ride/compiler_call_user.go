package ride

import "fmt"

// Function call
type CallUserState struct {
	prev Fsm
	params
	name string
	argc uint16
	// positions of arguments
	argn      []uniqueid
	ret       func(s CallUserState, at uint16, to uint16)
	startedAt uint16
}

func (a CallUserState) retAssigment(startedAt uint16, endedAt uint16) Fsm {
	if a.ret != nil {
		a.ret(a, startedAt, endedAt)
	}
	a.ret = nil
	return a
}

func newCallUserFsm(prev Fsm, params params, name string, argc uint16) Fsm {
	return &CallUserState{
		prev:      prev,
		params:    params,
		name:      name,
		argc:      argc,
		startedAt: params.b.len(),
	}
}

func (a CallUserState) Property(name string) Fsm {
	panic("CallUserState Property")
}

func (a CallUserState) Func(name string, args []string, invoke string) Fsm {
	return funcTransition(a, a.params, name, args, invoke)
}

func (a CallUserState) Bytes(b []byte) Fsm {
	a.argn = append(a.argn, putConstant(a.params, rideBytes(b)))
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
	return a
}

func (a CallUserState) Boolean(v bool) Fsm {
	a.argn = append(a.argn, putConstant(a.params, rideBoolean(v)))
	return a
}

func (a CallUserState) Assigment(name string) Fsm {
	//return assigmentFsmTransition(a, a.params, name)
	panic("illegal transition")
}

func (a CallUserState) Long(value int64) Fsm {
	a.argn = append(a.argn, putConstant(a.params, rideInt(value)))
	return a
}

func (a CallUserState) Return() Fsm {
	// check user functions
	n, ok := a.r.get(a.name)
	if !ok {
		panic(fmt.Sprintf("user function `%s` not found", a.name))
	}
	for i, pos := range a.argn {
		a.b.writeByte(OpSetArg)
		funcParamID, ok := a.r.get(fmt.Sprintf("%s$%d", a.name, i))
		if !ok {
			panic(fmt.Sprintf("no function param id `%s` stored in references", fmt.Sprintf("%s$%d", a.name, i)))
		}
		a.b.write(encode(pos))
		a.b.write(encode(funcParamID))
	}

	point, ok := a.params.c.get(n)
	if !ok {
		panic(fmt.Sprintf("no point %d found in cell", n))
	}

	a.b.call(point.position, a.argc)
	return a.prev.retAssigment(a.startedAt, a.b.len())
}

func (a CallUserState) Call(name string, argc uint16) Fsm {
	n := a.u.next()
	a.c.set(n, nil, nil, 0, false, fmt.Sprintf("function as paramentr: %s$%d", name, n))
	a.argn = append(a.argn, n)
	if a.ret != nil {
		panic("already assigned")
	}
	a.ret = func(state CallUserState, startedAt uint16, endedAt uint16) {
		a.b.writeByte(OpCache)
		a.b.write(encode(n))
		a.b.writeByte(OpPop)
	}
	return callTransition(a, a.params, name, argc)
}

func (a CallUserState) Reference(name string) Fsm {
	rs, ok := a.r.get(name)
	if !ok {
		panic("CallUserState Reference " + name + " not found")
	}
	a.argn = append(a.argn, rs)
	return a
}
