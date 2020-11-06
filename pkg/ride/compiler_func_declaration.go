package ride

import "fmt"

type arguments []string

func (a arguments) pos(name string) int {
	for i := range a {
		if a[i] == name {
			return i
		}
	}
	return -1
}

type FuncDeclarationState struct {
	params
	prev        Fsm
	name        string
	args        arguments
	offset      uint16
	globalScope *references
}

func (a FuncDeclarationState) Property(name string) Fsm {
	panic("FuncDeclarationState Property")
}

func funcDeclarationFsmTransition(prev Fsm, params params, name string, args []string) Fsm {
	// save reference to global scope, where code lower that function will be able to use it.
	globalScope := params.r
	//// all variable we add only visible to current scope,
	//// avoid corrupting parent state.
	params.r = newReferences(params.r)
	for i := range args {
		params.r.set(args[i], params.b.len())
		// set to global
		globalScope.set(fmt.Sprintf("%s$%d", name, i), params.u.next())
		params.b.w.WriteByte(OpUseArg)
		params.b.w.Write(encode(params.u.cur()))
		params.b.w.WriteByte(OpReturn)
	}

	return &FuncDeclarationState{
		prev:        prev,
		name:        name,
		args:        args,
		params:      params,
		offset:      params.b.len(),
		globalScope: globalScope,
	}
}

func (a FuncDeclarationState) Assigment(name string) Fsm {
	return assigmentFsmTransition(a, a.params, name)
}

func (a FuncDeclarationState) Return() Fsm {
	a.globalScope.set(a.name, a.offset)
	a.b.writeByte(OpPopCtx)
	a.b.ret()
	return a.prev
}

func (a FuncDeclarationState) Long(value int64) Fsm {
	index := a.params.c.put(rideInt(value))
	a.params.b.push(index)
	return a
}

func (a FuncDeclarationState) Call(name string, argc uint16) Fsm {
	return callTransition(a, a.params, name, argc)
}

func (a FuncDeclarationState) Reference(name string) Fsm {
	return reference(a, a.params, name)
}

func (a FuncDeclarationState) Boolean(v bool) Fsm {
	panic("implement me")
}

func (a FuncDeclarationState) String(s string) Fsm {
	panic("implement me")
}

func (a FuncDeclarationState) Condition() Fsm {
	return conditionalTransition(a, a.params)
}

func (a FuncDeclarationState) TrueBranch() Fsm {
	panic("implement me")
}

func (a FuncDeclarationState) FalseBranch() Fsm {
	panic("implement me")
}

func (a FuncDeclarationState) Bytes(b []byte) Fsm {
	panic("implement me")
}

func (a FuncDeclarationState) FuncDeclaration(name string, args []string) Fsm {
	panic("Illegal call `FuncDeclaration` is `FuncDeclarationState`")
}
