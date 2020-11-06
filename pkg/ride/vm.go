package ride

import (
	"encoding/binary"

	//im "github.com/frozen/immutable_map"
	"github.com/pkg/errors"
)

//type Context interface {
//	add(name string, rideType2 rideType) Context
//	get(name string) (rideType, bool)
//}
//type ContextImpl struct {
//	m *im.Map
//}
//
//func newContext() Context {
//	return ContextImpl{m: im.New()}
//}
//
//func (a ContextImpl) add(name string, value rideType) Context {
//	return ContextImpl{
//		m: a.m.Insert([]byte(name), value),
//	}
//}
//
//func (a ContextImpl) get(name string) (rideType, bool) {
//	v, ok := a.m.Get([]byte(name))
//	if !ok {
//		return nil, ok
//	}
//	return v.(rideType), ok
//}

type frame struct {
	function bool
	back     int
	context  []int
	future   []int
}

func newExpressionFrame(pos int) frame {
	return frame{
		back: pos,
	}
}

func newFrameContext(pos int, context args, future args) frame {
	return frame{
		back:    pos,
		context: context,
		future:  future,
	}
}

//func newFunctionFrame(pos int, args []int) frame {
//	return frame{
//		function: true,
//		back:     pos,
//		args:     args,
//	}
//}

type args = []int

type vm struct {
	env          RideEnvironment
	code         []byte
	ip           int
	constants    []rideType
	functions    func(int) rideFunction
	globals      func(int) rideConstructor
	stack        []rideType
	functionName func(int) string
	jpms         []int
	mem          map[uint16]uint16
}

func (m *vm) run() (RideResult, error) {
	if m.stack != nil {
		m.stack = m.stack[0:0]
	}
	m.mem = make(map[uint16]uint16)
	for m.ip < len(m.code) {
		op := m.code[m.ip]
		m.ip++
		switch op {
		case OpPush:
			m.push(m.constant())
		case OpPop:
			_, err := m.pop()
			if err != nil {
				return nil, errors.Wrap(err, "failed to pop value")
			}
		case OpTrue:
			m.push(rideBoolean(true))
		case OpFalse:
			m.push(rideBoolean(false))
		case OpJump:
			pos := m.arg16()
			m.jpms = append(m.jpms, m.ip)
			m.ip = pos
		case OpJumpIfFalse:
			pos := m.arg16()
			val, err := m.pop()
			if err != nil {
				return nil, errors.Wrap(err, "OpJumpIfFalse")
			}
			v, ok := val.(rideBoolean)
			if !ok {
				return nil, errors.Errorf("not a boolean value '%v' of type '%T'", m.current(), m.current())
			}
			if !v {
				m.ip = pos
			}
		case OpProperty:
			obj, err := m.pop()
			if err != nil {
				return nil, errors.Wrap(err, "failed to get object")
			}
			prop := m.constant()
			p, ok := prop.(rideString)
			if !ok {
				return nil, errors.Errorf("invalid property name type '%s'", prop.instanceOf())
			}
			v, err := obj.get(string(p))
			if err != nil {
				return nil, err
			}
			m.push(v)
		case OpCall:
			pos := m.arg16()
			m.jpms = append(m.jpms, m.ip)
			m.ip = pos

		case OpExternalCall:
			// Before calling external function all parameters must be evaluated and placed on stack
			id := m.arg16()
			cnt := m.arg16()
			in := make([]rideType, cnt) // Prepare input parameters for external call
			for i := cnt - 1; i >= 0; i-- {
				v, err := m.pop()
				if err != nil {
					return nil, errors.Wrapf(err, "failed to call external function '%s'", m.functionName(id))
				}
				in[i] = v
			}
			fn := m.functions(id)
			if fn == nil {
				return nil, errors.Errorf("external function '%s' not implemented", m.functionName(id))
			}
			res, err := fn(m.env, in...)
			if err != nil {
				return nil, err
			}
			m.push(res)
		case OpReturn:
			l := len(m.jpms)
			if l == 0 {
				if len(m.stack) > 0 {
					v, err := m.pop()
					if err != nil {
						return nil, errors.Wrap(err, "failed to get result value")
					}
					switch tv := v.(type) {
					case rideBoolean:
						return ScriptResult{res: bool(tv)}, nil
					default:
						return nil, errors.Errorf("unexpected result value '%v' of type '%T'", v, v)
					}
				}
				return nil, errors.New("no result after script execution")
			}
			m.ip, m.jpms = m.jpms[l-1], m.jpms[:l-1]

		case OpSetArg:
			id := m.arg16()
			value := m.arg16()
			m.mem[uint16(id)] = uint16(value)

		case OpUseArg:
			argid := m.arg16()
			m.jpms = append(m.jpms, m.ip)
			m.ip = int(m.mem[uint16(argid)])

		default:
			return nil, errors.Errorf("unknown code %#x", op)
		}
	}
	return nil, errors.New("broken code")
}

func (m *vm) push(v rideType) constid {
	m.stack = append(m.stack, v)
	return uint16(len(m.stack) - 1)
}

func (m *vm) pop() (rideType, error) {
	if len(m.stack) == 0 {
		return nil, errors.New("empty callStack")
	}
	value := m.stack[len(m.stack)-1]
	m.stack = m.stack[:len(m.stack)-1]
	return value, nil
}

func (m *vm) current() rideType {
	return m.stack[len(m.stack)-1]
}

func (m *vm) arg16() int {
	//TODO: add check
	res := binary.BigEndian.Uint16(m.code[m.ip : m.ip+2])
	m.ip += 2
	return int(res)
}

func (m *vm) constant() rideType {
	//TODO: add check
	return m.constants[m.arg16()]
}
