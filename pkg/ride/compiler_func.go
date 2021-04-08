package ride

import (
	"fmt"
)

type arguments []string

type Deferreds interface {
	Add(Deferred, uniqueID, string)
}

type dd struct {
	deferred Deferred
	uniq     uniqueID
	debug    string
}

type deferreds struct {
	name string
	d    []dd
}

func (a *deferreds) Add(deferred2 Deferred, n uniqueID, debug string) {
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
	prev        State
	name        string
	args        arguments
	n           uniqueID
	invokeParam string
	paramIds    []uniqueID

	// References that defined inside function.
	deferred []Deferred
	defers   *deferreds
	argn     int
}

func (a FuncState) backward(as State) State {
	// Func in func.
	if f, ok := as.(FuncState); ok {
		a.defers.Add(as.(Deferred), f.n, fmt.Sprintf("func `%s`in func %s", f.name, a.name))
	} else {
		a.deferred = append(a.deferred, as.(Deferred))
	}
	return a
}

func (a FuncState) Property(name string) State {
	return propertyTransition(a, a.params, name, a.defers)
}

func funcTransition(prev State, params params, name string, args []string, invokeParam string) State {
	argn := len(args)
	n := params.u.next()
	params.r.setFunc(name, n)
	// All variable we add only visible to current scope,
	// avoid corrupting global scope.
	params.r = newReferences(params.r)

	// Function call: verifier or not.
	if invokeParam != "" {
		args = append([]string{invokeParam}, args...)
	}
	paramIds := make([]uniqueID, 0, len(args))
	for i := range args {
		e := params.u.next()
		paramIds = append(paramIds, e)
		params.r.setAssigment(args[i], e)
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

func (a FuncState) Assigment(name string) State {
	n := a.params.u.next()
	return assigmentTransition(a, a.params, name, n, a.defers)
}

func (a FuncState) ParamIds() []uniqueID {
	return a.paramIds
}

func (a FuncState) Return() State {
	return a.prev.backward(a)
}

func (a FuncState) Long(value int64) State {
	a.deferred = append(a.deferred, a.constant(rideInt(value)))
	return a
}

func (a FuncState) Call(name string, argc uint16) State {
	return callTransition(a, a.params, name, argc, a.defers)
}

func (a FuncState) Reference(name string) State {
	a.deferred = append(a.deferred, reference(a, a.params, name))
	return a
}

func (a FuncState) Boolean(value bool) State {
	a.deferred = append(a.deferred, a.constant(rideBoolean(value)))
	return a
}

func (a FuncState) String(value string) State {
	a.deferred = append(a.deferred, a.constant(rideString(value)))
	return a
}

func (a FuncState) Condition() State {
	return conditionalTransition(a, a.params, a.defers)
}

func (a FuncState) TrueBranch() State {
	panic("Illegal call `TrueBranch` on `FuncState`")
}

func (a FuncState) FalseBranch() State {
	panic("Illegal call `FalseBranch` on `FuncState`")
}

func (a FuncState) Bytes(value []byte) State {
	a.deferred = append(a.deferred, a.constant(rideBytes(value)))
	return a
}

func (a FuncState) Func(name string, args []string, invoke string) State {
	return funcTransition(a, a.params, name, args, invoke)
}

func (a FuncState) Clean() {}

func (a FuncState) Write(_ params, _ []byte) {
	pos := a.b.len()
	a.params.c.set(a.n, nil, 0, pos, fmt.Sprintf("function %s", a.name))
	if len(a.deferred) != 1 {
		panic("len(a.deferred) != 1")
	}
	// Assign function arguments from stack.
	for i := len(a.paramIds) - 1; i >= 0; i-- {
		a.b.writeByte(OpCache)
		a.b.write(encode(a.paramIds[i]))
		a.b.writeByte(OpPop)
	}
	a.deferred[0].Write(a.params, nil)

	// End of function body. Clear and write assignments.
	for _, v := range a.defers.Get() {
		v.deferred.Clean()
	}
	a.b.ret()

	for _, v := range a.defers.Get() {
		p := a.b.len()
		a.c.set(v.uniq, nil, 0, p, v.debug)
		v.deferred.Write(a.params, nil)
		a.b.ret()
	}
}
