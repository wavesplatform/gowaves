package ride

import "fmt"

type Fsm interface {
	Assigment(name string) Fsm
	Return() Fsm
	Long(value int64) Fsm
	Call(name string, argc uint16) Fsm
	Reference(name string) Fsm
	Boolean(v bool) Fsm
	String(s string) Fsm
	Condition() Fsm
	TrueBranch() Fsm
	FalseBranch() Fsm
	Bytes(b []byte) Fsm
	Func(name string, args []string, invokeParam string) Fsm
	Property(name string) Fsm
	backward(state Fsm) Fsm
	Deferred
}

type Write interface {
	Write(params, []byte)
}

type Clean interface {
	Clean()
}

type uniqid struct {
	id uint16
}

func (a *uniqid) next() uint16 {
	if a.id < 200 {
		a.id = 200
	}
	a.id++
	return a.id
}

type FunctionChecker func(string) (uint16, bool)

type params struct {
	// Wrapper on bytes.Buffer with handy methods.
	b *builder
	// Relation of variables and it's offset.
	r *references
	// Way to get function id.
	f FunctionChecker
	// Unique id for func params.
	u *uniqid
	// Predefined variables.
	c *cell
	// Transaction ID, for debug purpose.
	txID string
}

func (a *params) addPredefined(name string, id uniqueid, fn uint16) {
	a.r.setAssigment(name, id)
	a.c.set(id, nil, fn, 0, false, name)
}

func (a *params) constant(value rideType) constantDeferred {
	switch v := value.(type) {
	case rideInt:
		if v >= 0 && v <= 100 {
			return NewConstantDeferred(uniqueid(v))
		}
	case rideBoolean:
		if v {
			return NewConstantDeferred(101)
		} else {
			return NewConstantDeferred(102)
		}
	}
	n := a.u.next()
	a.c.set(n, value, 0, 0, true, fmt.Sprintf("constant %q", value))
	return NewConstantDeferred(n)
}

func reference(_ Fsm, params params, name string) constantDeferred {
	pos, ok := params.r.getAssigment(name)
	if !ok {
		panic(fmt.Sprintf("reference %s not found, tx %s", name, params.txID))
	}
	return NewConstantDeferred(pos)
}
