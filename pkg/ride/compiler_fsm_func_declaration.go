package ride

type FuncDeclarationFsm struct {
	params
	prev        Fsm
	name        string
	args        []string
	offset      uint16
	globalScope *references
}

func funcDeclarationFsmTransition(prev Fsm, params params, name string, args []string) Fsm {
	// save reference to global scope, where code lower that function will be able to use it.
	globalScope := params.r
	// all variable we add only visible to current scope,
	// avoid corrupting parent state.
	params.r = newReferences(params.r)
	for i := range args {
		params.r.set(args[i], params.b.len())
		params.b.w.WriteByte(OpPushFromFrame)
		params.b.w.Write(encode(uint16(i)))
		params.b.ret()
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
	a.b.ret()
	return a.prev
}

func (a FuncDeclarationFsm) Long(value int64) Fsm {
	panic("implement me")
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
