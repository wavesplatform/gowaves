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
	prev        Fsm
	name        string
	args        arguments
	offset      uint16
	globalScope *references
	invokeParam string
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
		//params.predef = newPredefWithValue(params.predef, "tx", math.MaxUint16, tx)
		pos := params.b.len()
		params.b.writeByte(OpExternalCall)
		params.b.write(encode(math.MaxUint16))
		params.b.write(encode(0))
		params.b.writeByte(OpReturn)
		params.r.set(invokeParam, pos)
	}

	return &FuncState{
		prev:        prev,
		name:        name,
		args:        args,
		params:      params,
		offset:      params.b.len(),
		globalScope: globalScope,
		invokeParam: invokeParam,
	}
}

func (a FuncState) Assigment(name string) Fsm {
	return assigmentFsmTransition(a, a.params, name)
}

func (a FuncState) Return() Fsm {
	a.globalScope.set(a.name, a.offset)
	// TODO clean args
	a.b.ret()

	// if function has invoke param, it means no other code will be provided.
	if a.invokeParam != "" {
		a.b.writeByte(OpCall)
		a.b.write(encode(a.offset))
	}

	return a.prev
}

func (a FuncState) Long(value int64) Fsm {
	a.params.b.push(a.constant(rideInt(value)))
	return a
}

func (a FuncState) Call(name string, argc uint16) Fsm {
	return callTransition(a, a.params, name, argc)
}

func (a FuncState) Reference(name string) Fsm {
	return reference(a, a.params, name)
}

func (a FuncState) Boolean(v bool) Fsm {
	panic("implement me")
}

func (a FuncState) String(s string) Fsm {
	panic("implement me")
}

func (a FuncState) Condition() Fsm {
	return conditionalTransition(a, a.params)
}

func (a FuncState) TrueBranch() Fsm {
	panic("implement me")
}

func (a FuncState) FalseBranch() Fsm {
	panic("implement me")
}

func (a FuncState) Bytes(b []byte) Fsm {
	panic("implement me")
}

func (a FuncState) Func(name string, args []string, _ string) Fsm {
	panic("Illegal call `Func` is `FuncState`")
}
