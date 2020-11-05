package ride

// Initial state, contains only assigments and last expression.
// TODO chose better name
type DefinitionFsm struct {
	params
}

func (a DefinitionFsm) Property(name string) Fsm {
	a.b.writeByte(OpProperty)
	index := a.params.c.put(rideString(name))
	a.params.b.push(index)
	return a
}

func (a DefinitionFsm) FuncDeclaration(name string, args []string) Fsm {
	return funcDeclarationFsmTransition(a, a.params, name, args)
}

func (a DefinitionFsm) Bytes(b []byte) Fsm {
	panic("Illegal call `Bytes` on `DefinitionFsm`")
}

func (a DefinitionFsm) Condition() Fsm {
	a.b.startPos()
	return conditionalTransition(a, a.params)
}

func (a DefinitionFsm) TrueBranch() Fsm {
	panic("Illegal call `TrueBranch` on DefinitionFsm")
}

func (a DefinitionFsm) FalseBranch() Fsm {
	panic("Illegal call `FalseBranch` on DefinitionFsm")
}

func (a DefinitionFsm) String(s string) Fsm {
	panic("Illegal call `String` on DefinitionFsm")
}

type BuildExecutable interface {
	BuildExecutable(version int) *Executable
}

func NewDefinitionsFsm(params params) Fsm {
	return &DefinitionFsm{
		params: params,
	}
}

func (a DefinitionFsm) Assigment(name string) Fsm {
	return assigmentFsmTransition(a, a.params, name)
}

func (a DefinitionFsm) Return() Fsm {
	a.b.ret()
	return a
}

func (a DefinitionFsm) Long(value int64) Fsm {
	panic("Illegal call Long on DefinitionFsm")
}

func (a DefinitionFsm) Call(name string, argc uint16) Fsm {
	a.b.startPos()
	return callTransition(a, a.params, name, argc)
}

func (a DefinitionFsm) Reference(name string) Fsm {
	a.b.startPos()
	return reference(a, a.params, name)
}

func (a DefinitionFsm) Boolean(v bool) Fsm {
	a.b.startPos()
	return boolean(a, a.params, v)
}

func (a DefinitionFsm) BuildExecutable(version int) *Executable {
	startAt, code := a.b.build()
	return &Executable{
		LibVersion:  version,
		ByteCode:    code,
		Constants:   a.c.constants(),
		EntryPoints: map[string]uint16{"": startAt},
	}
}
