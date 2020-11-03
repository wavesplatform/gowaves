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
	FuncDeclaration(name string, args []string) Fsm
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
	// wrapper on bytes.Buffer with handy methods.
	b *builder
	// slice of constants.
	c *constants
	// relation of variables and it's offset.
	r *references
	// way to get function id.
	f FunctionChecker
	// unique id for func params
	u *uniqid
}

func long(f Fsm, params params, value int64) Fsm {
	index := params.c.put(rideInt(value))
	params.b.push(index)
	params.b.ret()
	return f
}

func boolean(f Fsm, params params, value bool) Fsm {
	params.b.bool(value)
	return f
}

func str(a Fsm, params params, s string) Fsm {
	index := params.c.put(rideString(s))
	params.b.push(index)
	return a
}

// TODO: remove duplicate
func constant(a Fsm, params params, rideType rideType) Fsm {
	index := params.c.put(rideType)
	params.b.push(index)
	return a
}

func reference(f Fsm, params params, name string) Fsm {
	pos, ok := params.r.get(name)
	if !ok {
		//index := params.c.put(rideString(name))
		//params.b.fillContext(index)
		panic(fmt.Sprintf("reference %s not found", name))
		//return f
	}
	params.b.jump(pos)
	return f
}
