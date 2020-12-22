package ride

import "fmt"

// Function call
type CallUserState struct {
	prev Fsm
	params
	name string
	argc uint16
	// positions of arguments
	//argn      []uniqueid
	//ret       func(s CallUserState, at uint16, to uint16)
	startedAt uint16

	deferred  []Deferred
	deferreds Deferreds

	rnode []RNode
}

func (a CallUserState) backward(state Fsm) Fsm {
	a.deferred = append(a.deferred, state.(Deferred))
	return a
}

func newCallUserFsm(prev Fsm, params params, name string, argc uint16, d Deferreds) Fsm {
	return &CallUserState{
		prev:      prev,
		params:    params,
		name:      name,
		argc:      argc,
		startedAt: params.b.len(),
		deferreds: d,
	}
}

func (a CallUserState) Property(name string) Fsm {
	return propertyTransition(a, a.params, name, a.deferreds)
}

func (a CallUserState) Func(name string, args []string, invoke string) Fsm {
	return funcTransition(a, a.params, name, args, invoke)
}

func (a CallUserState) Bytes(b []byte) Fsm {
	a.deferred = append(a.deferred, a.constant(rideBytes(b)))
	return a
}

func (a CallUserState) Condition() Fsm {
	return conditionalTransition(a, a.params, a.deferreds)
}

func (a CallUserState) TrueBranch() Fsm {
	panic("Illegal call `TrueBranch` on CallFsm")
}

func (a CallUserState) FalseBranch() Fsm {
	panic("Illegal call `FalseBranch` on CallFsm")
}

func (a CallUserState) String(s string) Fsm {
	a.deferred = append(a.deferred, a.constant(rideString(s)))
	return a
}

func (a CallUserState) Boolean(v bool) Fsm {
	a.deferred = append(a.deferred, a.constant(rideBoolean(v)))
	return a
}

func (a CallUserState) Assigment(name string) Fsm {
	//return assigmentFsmTransition(a, a.params, name)
	panic("CallUserState Assigment")
}

func (a CallUserState) Long(value int64) Fsm {
	a.deferred = append(a.deferred, a.constant(rideInt(value)))
	return a
}

func (a CallUserState) Return() Fsm {
	/*
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

		_, ok = a.params.c.get(n)
		if !ok {
			panic(fmt.Sprintf("no point %d found in cell", n))
		}

		//a.b.call(point.position, a.argc)
		a.b.writeByte(OpRef)
		a.b.write(encode(n))
	*/
	return a.prev.backward(a) //.backward(a.startedAt, a.b.len())
}

func (a CallUserState) Call(name string, argc uint16) Fsm {
	//n := a.u.next()
	//a.c.set(n, nil, nil, 0, false, fmt.Sprintf("function as paramentr: %s$%d", name, n))
	//a.argn = append(a.argn, n)
	//if a.ret != nil {
	//	panic("already assigned")
	//}
	//a.ret = func(state CallUserState, startedAt uint16, endedAt uint16) {
	//	a.b.writeByte(OpCache)
	//	a.b.write(encode(n))
	//	a.b.writeByte(OpPop)
	//}
	return callTransition(a, a.params, name, argc, a.deferreds)
}

func (a CallUserState) Reference(name string) Fsm {
	a.deferred = append(a.deferred, reference(a, a.params, name))
	//rs, ok := a.r.get(name)
	//if !ok {
	//	panic("CallUserState Reference " + name + " not found")
	//}
	//a.argn = append(a.argn, rs)
	return a
}

func (a CallUserState) Clean() {

}

func (a CallUserState) Write(_ params, b []byte) {
	// check user functions
	fn, ok := a.r.get(a.name)
	if !ok {
		panic(fmt.Sprintf("user function `%s` not found", a.name))
	}

	if int(a.argc) != len(a.deferred) {
		panic(fmt.Sprintf("argc %d != a.deferred %d", a.argc, len(a.deferred)))
	}

	var ns []uniqueid
	for i := uint16(0); i < a.argc; i++ {
		if n, ok := isConstant(a.deferred[i]); ok {
			a.b.writeByte(OpRef)
			a.b.write(encode(n))
			a.b.writeByte(OpCache)
			a.b.write(encode(fn + 1 + i))
			a.b.writeByte(OpPop)
			ns = append(ns, n)
		} else {
			n := a.u.next()
			a.b.writeByte(OpRef)
			a.b.write(encode(n))
			a.b.writeByte(OpCache)
			a.b.write(encode(fn + 1 + i))
			a.b.writeByte(OpPop)
			ns = append(ns, n)
		}
	}

	a.b.writeByte(OpRef)
	a.b.write(encode(fn))
	a.b.write(b)
	a.b.ret()

	if len(ns) != len(a.deferred) {
		panic(fmt.Sprintf("ns %d != a.deferred %d", a.argc, len(a.deferred)))
	}

	for i, b := range a.deferred {
		if _, ok := isConstant(b); ok {
			// skip right now
		} else {
			a.c.set(ns[i], nil, nil, a.b.len(), false, fmt.Sprintf("sys %s param #%d", a.name, i))
			b.Write(a.params, nil)
			a.b.ret()
		}
	}
}
