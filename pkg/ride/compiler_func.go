package ride

import (
	"fmt"
	"math"
)

type arguments []string

func (a arguments) pos(name string) int {
	for i := range a {
		if a[i] == name {
			return i
		}
	}
	return -1
}

type FuncState struct {
	params
	prev           Fsm
	name           string
	args           arguments
	globalScope    *references
	invokeParam    string
	lastStmtOffset uint16
}

func (a FuncState) retAssigment(pos uint16) Fsm {
	a.lastStmtOffset = pos
	return a
}

func (a FuncState) Property(name string) Fsm {
	panic("FuncState Property")
}

func funcTransition(prev Fsm, params params, name string, args []string, invokeParam string) Fsm {
	// save reference to global scope, where code lower that function will be able to use it.
	globalScope := params.r
	// all variable we add only visible to current scope,
	// avoid corrupting parent state.
	params.r = newReferences(params.r)
	for i := range args {
		e := params.u.next()
		params.r.set(args[i], e)
		// set to global
		globalScope.set(fmt.Sprintf("%s$%d", name, i), e)
	}
	// assume that it's verifier
	if invokeParam != "" {
		// tx
		pos, ok := params.r.get("tx")
		if !ok {
			panic("no `tx` in function call")
		}
		params.b.writeByte(OpExternalCall)
		params.b.write(encode(math.MaxUint16))
		params.b.write(encode(0))
		params.b.writeByte(OpReturn)
		params.r.set(invokeParam, pos)
	}

	return &FuncState{
		prev:   prev,
		name:   name,
		args:   args,
		params: params,
		//offset:      params.b.len(),
		globalScope: globalScope,
		invokeParam: invokeParam,
	}
}

func (a FuncState) Assigment(name string) Fsm {
	return assigmentFsmTransition(a, a.params, name)
}

func (a FuncState) Return() Fsm {
	funcID := a.params.u.next()
	a.globalScope.set(a.name, funcID)
	a.params.c.set(funcID, nil, nil, a.lastStmtOffset)
	// TODO clean args
	a.b.ret()

	// if function has invoke param, it means no other code will be provided.
	if a.invokeParam != "" {
		a.b.startPos()
		a.b.writeByte(OpCall)
		a.b.write(encode(a.lastStmtOffset))
	}

	return a.prev.retAssigment(a.b.len())
}

func (a FuncState) Long(value int64) Fsm {
	a.lastStmtOffset = a.b.len()
	a.params.b.push(a.constant(rideInt(value)))
	return a
}

func (a FuncState) Call(name string, argc uint16) Fsm {
	a.lastStmtOffset = a.b.len()
	return callTransition(a, a.params, name, argc)
}

func (a FuncState) Reference(name string) Fsm {
	a.lastStmtOffset = a.b.len()
	return reference(a, a.params, name)
}

func (a FuncState) Boolean(v bool) Fsm {
	a.lastStmtOffset = a.b.len()
	return constant(a, a.params, rideBoolean(v))
}

func (a FuncState) String(s string) Fsm {
	a.lastStmtOffset = a.b.len()
	return constant(a, a.params, rideString(s))
}

func (a FuncState) Condition() Fsm {
	a.lastStmtOffset = a.b.len()
	return conditionalTransition(a, a.params)
}

func (a FuncState) TrueBranch() Fsm {
	panic("Illegal call `TrueBranch` on `FuncState`")
}

func (a FuncState) FalseBranch() Fsm {
	panic("Illegal call `FalseBranch` on `FuncState`")
}

func (a FuncState) Bytes(b []byte) Fsm {
	a.lastStmtOffset = a.b.len()
	return constant(a, a.params, rideBytes(b))
}

func (a FuncState) Func(name string, args []string, _ string) Fsm {
	panic("Illegal call `Func` is `FuncState`")
}
