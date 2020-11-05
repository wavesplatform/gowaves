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

type FuncDeclarationFsm struct {
	params
	prev        Fsm
	name        string
	args        arguments
	offset      uint16
	globalScope *references
}

func (a FuncDeclarationFsm) Property(name string) Fsm {
	panic("FuncDeclarationFsm Property")
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

	return &FuncDeclarationFsm{
		prev:        prev,
		name:        name,
		args:        args,
		params:      params,
		offset:      params.b.len(),
		globalScope: globalScope,
	}
}

func (a FuncDeclarationFsm) Assigment(name string) Fsm {
	return assigmentFsmTransition(a, a.params, name)
}

func (a FuncDeclarationFsm) Return() Fsm {
	a.globalScope.set(a.name, a.offset)
	a.b.writeByte(OpPopCtx)
	a.b.ret()
	return a.prev
}

func (a FuncDeclarationFsm) Long(value int64) Fsm {
	index := a.params.c.put(rideInt(value))
	a.params.b.push(index)
	return a
}

func (a FuncDeclarationFsm) Call(name string, argc uint16) Fsm {
	return callTransition(a, a.params, name, argc)
}

func (a FuncDeclarationFsm) Reference(name string) Fsm {
	return reference(a, a.params, name)
}

func (a FuncDeclarationFsm) Boolean(v bool) Fsm {
	panic("implement me")
}

func (a FuncDeclarationFsm) String(s string) Fsm {
	panic("implement me")
}

func (a FuncDeclarationFsm) Condition() Fsm {
	return conditionalTransition(a, a.params)
}

func (a FuncDeclarationFsm) TrueBranch() Fsm {
	panic("implement me")
}

func (a FuncDeclarationFsm) FalseBranch() Fsm {
	panic("implement me")
}

func (a FuncDeclarationFsm) Bytes(b []byte) Fsm {
	panic("implement me")
}

func (a FuncDeclarationFsm) FuncDeclaration(name string, args []string) Fsm {
	panic("Illegal call `FuncDeclaration` is `FuncDeclarationFsm`")
}
