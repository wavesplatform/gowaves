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
	retAssigment(startedAt uint16, endedAt uint16) Fsm
}

type uniqid struct {
	id uint16
}

func (a *uniqid) next() uint16 {
	a.id++
	return a.id
}

func (a uniqid) cur() uint16 {
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

func (a *params) addPredefined(name string, id uniqueid, fn rideFunction) {
	a.r.set(name, id)
	a.c.set(id, nil, fn, 0, false, name)
}

func (a *params) constant(value rideType) uniqueid {
	n := a.u.next()
	a.c.set(n, value, nil, 0, false, fmt.Sprintf("constant %q", value))
	return n
}

func long(f Fsm, params params, value int64) Fsm {
	params.b.push(params.constant(rideInt(value)))
	return f
}

func boolean(f Fsm, params params, value bool) Fsm {
	params.b.push(params.constant(rideBoolean(value)))
	return f
}

func bts(f Fsm, params params, value []byte) Fsm {
	params.b.push(params.constant(rideBytes(value)))
	return f
}

func str(a Fsm, params params, value string) Fsm {
	params.b.push(params.constant(rideString(value)))
	return a
}

func constant(a Fsm, params params, value rideType) Fsm {
	params.b.push(params.constant(value))
	return a
}

func putConstant(params params, rideType rideType) uniqueid {
	index := params.constant(rideType)
	return index
}

func reference(f Fsm, params params, name string) Fsm {
	pos, ok := params.r.get(name)
	if !ok {
		panic(fmt.Sprintf("reference %s not found, tx %s", name, params.txID))
	}
	//params.b
	params.b.jump(pos)
	return f
}
