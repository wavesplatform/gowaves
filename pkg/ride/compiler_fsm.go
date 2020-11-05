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
	Property(name string) Fsm
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
	// Slice of constants.
	c *constants
	// Relation of variables and it's offset.
	r *references
	// Way to get function id.
	f FunctionChecker
	// Unique id for func params.
	u *uniqid
	// Predefined variables.
	predef predef
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
		if n, ok := params.predef.get(name); ok {
			params.b.writeByte(OpExternalCall)
			params.b.write(encode(n.id))
			params.b.write(encode(0))
			return f
		}
		//index := params.c.put(rideString(name))
		//params.b.fillContext(index)
		panic(fmt.Sprintf("reference %s not found", name))
		//return f
	}
	params.b.jump(pos)
	return f
}
