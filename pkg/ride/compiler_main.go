package ride

// Initial state, contains only assigments and last expression.
type MainState struct {
	params

	body      []Deferred
	deferreds *deferreds
}

func (a MainState) backward(state State) State {
	a.body = append(a.body, state.(Deferred))
	return a
}

func (a MainState) Property(name string) State {
	return propertyTransition(a, a.params, name, a.deferreds)
}

func (a MainState) Func(name string, args []string, invoke string) State {
	return funcTransition(a, a.params, name, args, invoke)
}

func (a MainState) Bytes([]byte) State {
	panic("Illegal call `Bytes` on `MainState`")
}

func (a MainState) Condition() State {
	return conditionalTransition(a, a.params, a.deferreds)
}

func (a MainState) TrueBranch() State {
	panic("Illegal call `TrueBranch` on MainState")
}

func (a MainState) FalseBranch() State {
	panic("Illegal call `FalseBranch` on MainState")
}

func (a MainState) String(string) State {
	panic("Illegal call `String` on MainState")
}

type BuildExecutable interface {
	BuildExecutable(version int, isDapp bool, hasVerifier bool) *Executable
}

func NewMain(params params) State {
	return &MainState{
		params: params,
		deferreds: &deferreds{
			name: "main",
		},
	}
}

func (a MainState) Assigment(name string) State {
	n := a.params.u.next()
	return assigmentTransition(a, a.params, name, n, a.deferreds)
}

func (a MainState) Return() State {
	for _, v := range a.deferreds.Get() {
		v.deferred.Clean()
	}
	a.b.ret()

	body := a.body
	// empty script, example https://testnet.wavesexplorer.com/tx/DprupHKCwJwRhyhbHyqJqp35CvhiJdpkhjf53z1vmwHr
	if len(body) == 0 {
		return a
	}
	for {
		if f, ok := body[0].(FuncState); ok && f.invokeParam != "" {
			a.b.setStart(f.name, f.argn)
			a.b.setStart("", 0)
		} else {
			a.b.setStart("", 0)
		}

		body[0].Write(a.params, nil)
		a.b.ret()
		body = body[1:]
		if len(body) == 0 {
			break
		}
	}
	a.b.ret()

	for _, v := range a.deferreds.Get() {
		pos := a.b.len()
		a.c.set(v.uniq, nil, 0, pos, false, v.debug)
		v.deferred.Write(a.params, nil)
		a.b.ret()
	}
	return a
}

func (a MainState) Long(int64) State {
	panic("Illegal call Long on MainState")
}

func (a MainState) Call(name string, argc uint16) State {
	return callTransition(a, a.params, name, argc, a.deferreds)
}

func (a MainState) Reference(name string) State {
	a.body = append(a.body, reference(a, a.params, name))
	return a
}

func (a MainState) Boolean(v bool) State {
	a.body = append(a.body, a.constant(rideBoolean(v)))
	return a
}

func (a MainState) BuildExecutable(version int, isDapp bool, hasVerifier bool) *Executable {
	entrypoints, code := a.b.build()
	return &Executable{
		LibVersion:  version,
		ByteCode:    code,
		References:  a.c.values,
		EntryPoints: entrypoints,
		IsDapp:      isDapp,
		hasVerifier: hasVerifier,
	}
}

func (a MainState) Write(_ params, _ []byte) {

}

func (a MainState) Clean() {

}
