package ride

import "fmt"

type arguments []string

type Deferreds interface {
	Add(Deferred, uniqueid, string)
}

type dd struct {
	deferred Deferred
	uniq     uniqueid
	debug    string
}

type deferreds struct {
	name string
	d    []dd
}

func (a *deferreds) Add(deferred2 Deferred, n uniqueid, debug string) {
	a.d = append(a.d, dd{
		deferred: deferred2,
		uniq:     n,
		debug:    debug,
	})
}

func (a *deferreds) Get() []dd {
	return a.d
}

type FuncState struct {
	params
	prev        Fsm
	name        string
	args        arguments
	n           uniqueid
	invokeParam string
	paramIds    []uniqueid

	// References that defined inside function.
	deferred []Deferred
	defers   *deferreds
	argn     int
}

func (a FuncState) backward(as Fsm) Fsm {
	// Func in func.
	if f, ok := as.(FuncState); ok {
		a.defers.Add(as.(Deferred), f.n, fmt.Sprintf("func `%s`in func %s", f.name, a.name))
	} else {
		a.deferred = append(a.deferred, as.(Deferred))
	}
	return a
}

func (a FuncState) Property(name string) Fsm {
	return propertyTransition(a, a.params, name, a.defers)
}

func funcTransition(prev Fsm, params params, name string, args []string, invokeParam string) Fsm {
	argn := len(args)
	n := params.u.next()
	params.r.set(name, n)
	// all variable we add only visible to current scope,
	// avoid corrupting global scope.
	params.r = newReferences(params.r)

	// Function call: verifier or not.
	if invokeParam != "" {
		args = append([]string{invokeParam}, args...)
	}
	paramIds := make([]uniqueid, 0, len(args))
	for i := range args {
		e := params.u.next()
		paramIds = append(paramIds, e)
		params.r.set(args[i], e)
	}

	return &FuncState{
		prev:        prev,
		name:        name,
		args:        args,
		params:      params,
		n:           n,
		invokeParam: invokeParam,
		defers: &deferreds{
			name: "func " + name,
		},
		paramIds: paramIds,
		argn:     argn,
	}
}

func (a FuncState) Assigment(name string) Fsm {
	n := a.params.u.next()
	return assigmentFsmTransition(a, a.params, name, n, a.defers)
}

func (a FuncState) ParamIds() []uniqueid {
	return a.paramIds
}

func (a FuncState) Return() Fsm {
	return a.prev.backward(a)
}

func (a FuncState) Long(value int64) Fsm {
	a.deferred = append(a.deferred, a.constant(rideInt(value)))
	return a
}

func (a FuncState) Call(name string, argc uint16) Fsm {
	return callTransition(a, a.params, name, argc, a.defers)
}

func (a FuncState) Reference(name string) Fsm {
	a.deferred = append(a.deferred, reference(a, a.params, name))
	return a
}

func (a FuncState) Boolean(value bool) Fsm {
	a.deferred = append(a.deferred, a.constant(rideBoolean(value)))
	return a
}

func (a FuncState) String(value string) Fsm {
	a.deferred = append(a.deferred, a.constant(rideString(value)))
	return a
}

func (a FuncState) Condition() Fsm {
	return conditionalTransition(a, a.params, a.defers)
}

func (a FuncState) TrueBranch() Fsm {
	panic("Illegal call `TrueBranch` on `FuncState`")
}

func (a FuncState) FalseBranch() Fsm {
	panic("Illegal call `FalseBranch` on `FuncState`")
}

func (a FuncState) Bytes(value []byte) Fsm {
	a.deferred = append(a.deferred, a.constant(rideBytes(value)))
	return a
}

func (a FuncState) Func(name string, args []string, invoke string) Fsm {
	return funcTransition(a, a.params, name, args, invoke)
}

func (a FuncState) Clean() {

}

func (a FuncState) Write(_ params, b []byte) {
	pos := a.b.len()
	a.params.c.set(a.n, nil, 0, pos, false, fmt.Sprintf("function %s", a.name))
	if len(a.deferred) != 1 {
		panic("len(a.deferred) != 1")
	}
	a.deferred[0].Write(a.params, nil)

	// End of function body. Clear and write assigments.
	for _, v := range a.defers.Get() {
		v.deferred.Clean()
	}
	a.b.ret()

	for _, v := range a.defers.Get() {
		pos := a.b.len()
		a.c.set(v.uniq, nil, 0, pos, false, v.debug)
		v.deferred.Write(a.params, nil)
		a.b.ret()
	}

}
