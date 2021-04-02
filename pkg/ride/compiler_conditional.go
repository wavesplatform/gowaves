package ride

import "fmt"

// If-else statement.
type ConditionalState struct {
	/*
		Be aware that `x` and `y` should not be executed.

		if (true) then {
			let x = throw()
			5
		} else {
			let y = throw()
			6
		}
	*/

	originalParams params
	params
	prev State

	rets []uint16

	// Clean assigments after exit.
	deferred  []Deferred
	deferreds Deferreds

	condN uniqueid
}

func (a ConditionalState) backward(as State) State {
	// Func in func.
	if f, ok := as.(FuncState); ok {
		a.deferreds.Add(as.(Deferred), f.n, fmt.Sprintf("func `%s`in conditional", f.name))
	} else {
		a.deferred = append(a.deferred, as.(Deferred))
	}
	return a
}

func (a ConditionalState) Property(name string) State {
	return propertyTransition(a, a.params, name, a.deferreds)
}

func (a ConditionalState) Func(name string, args []string, invoke string) State {
	return funcTransition(a, a.params, name, args, invoke)
}

func (a ConditionalState) Bytes(value []byte) State {
	a.deferred = append(a.deferred, a.constant(rideBytes(value)))
	return a
}

func conditionalTransition(prev State, params params, deferreds Deferreds) State {
	return ConditionalState{
		prev:           prev,
		params:         extendParams(params),
		deferreds:      deferreds,
		originalParams: params,
	}
}

func (a ConditionalState) Condition() State {
	a.rets = append(a.rets, a.params.b.len())
	return conditionalTransition(a, a.params, a.deferreds)
}

func (a ConditionalState) TrueBranch() State {
	a.params = extendParams(a.originalParams)
	return a
}

func (a ConditionalState) FalseBranch() State {
	a.params = extendParams(a.originalParams)
	return a
}

func (a ConditionalState) Assigment(name string) State {
	n := a.params.u.next()
	return assigmentTransition(a, a.params, name, n, a.deferreds)
}

func (a ConditionalState) Long(value int64) State {
	a.deferred = append(a.deferred, a.constant(rideInt(value)))
	return a
}

func (a ConditionalState) Call(name string, argc uint16) State {
	return callTransition(a, a.params, name, argc, a.deferreds)
}

func (a ConditionalState) Reference(name string) State {
	a.deferred = append(a.deferred, reference(a, a.params, name))
	return a
}

func (a ConditionalState) Boolean(value bool) State {
	a.deferred = append(a.deferred, a.constant(rideBoolean(value)))
	return a
}

func (a ConditionalState) String(value string) State {
	a.deferred = append(a.deferred, a.constant(rideString(value)))
	return a
}

func (a ConditionalState) Return() State {
	if len(a.deferred) < 3 {
		panic("len(a.deferred) < 3")
	}
	a.condN = a.u.next()
	a.deferreds.Add(a.deferred[0], a.condN, "condition cond")
	return a.prev.backward(a)
}

func (a ConditionalState) Write(_ params, _ []byte) {
	if len(a.deferred) != 3 {
		panic("len(a.deferred) != 3")
	}

	trueB := a.deferred[1]
	falsB := a.deferred[2]

	a.b.writeByte(OpRef)
	a.b.write(encode(a.condN))

	a.b.jpmIfFalse()
	patchTruePosition := a.b.writeStub(2)
	patchFalsePosition := a.b.writeStub(2)
	patchNextPosition := a.b.writeStub(2)

	a.b.patch(patchTruePosition, encode(a.b.len()))
	trueB.Write(a.params, nil)
	a.b.ret()

	a.b.patch(patchFalsePosition, encode(a.b.len()))
	falsB.Write(a.params, nil)
	a.b.ret()

	a.b.patch(patchNextPosition, encode(a.b.len()))
}
func (a ConditionalState) Clean() {
}
