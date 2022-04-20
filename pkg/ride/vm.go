package ride

import (
	"encoding/binary"

	"github.com/pkg/errors"
)

type frame struct {
	function bool
	back     int
	args     []rideType
}

func newExpressionFrame(pos int) frame {
	return frame{
		back: pos,
	}
}

func newFunctionFrame(pos int, args []rideType) frame {
	return frame{
		function: true,
		back:     pos,
		args:     args,
	}
}

type vm struct {
	env          environment
	code         []byte
	ip           int
	constants    []rideType
	functions    func(int) rideFunction
	globals      func(int) rideConstructor
	stack        []rideType
	calls        []frame
	functionName func(int) string
}

func (m *vm) run() (Result, error) {
	if m.stack != nil {
		m.stack = m.stack[0:0]
	}
	if m.calls != nil {
		m.calls = m.calls[0:0]
	}
	m.ip = 0
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
			m.ip = pos
		case OpJumpIfFalse:
			pos := m.arg16()
			v, ok := m.current().(rideBoolean)
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
			cnt := m.arg16()
			in := make([]rideType, cnt)
			for i := cnt - 1; i >= 0; i-- {
				v, err := m.pop()
				if err != nil {
					return nil, errors.Wrapf(err, "failed to call function at position %d", pos)
				}
				in[i] = v
			}
			frame := newFunctionFrame(m.ip, in) // Creating new function frame with return position
			m.calls = append(m.calls, frame)
			m.ip = pos // Continue to function
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
		case OpLoad: // Evaluate expression behind a LET declaration
			pos := m.arg16()
			frame := newExpressionFrame(m.ip) // Creating new function frame with return position
			m.calls = append(m.calls, frame)
			m.ip = pos // Continue to expression
		case OpLoadLocal:
			n := m.arg16()
			for i := len(m.calls) - 1; i >= 0; i-- {

			}
			l := len(m.calls)
			if l == 0 {
				return nil, errors.New("failed to load argument on stack")
			}
			frame := m.calls[l-1]
			if l := len(frame.args); l < n+1 {
				return nil, errors.New("invalid arguments count")
			}
			m.push(frame.args[n])
		case OpReturn:
			l := len(m.calls)
			var f frame
			f, m.calls = m.calls[l-1], m.calls[:l-1]
			m.ip = f.back
		case OpHalt:
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
		case OpGlobal:
			id := m.arg16()
			constructor := m.globals(id)
			v := constructor(m.env)
			m.push(v)
		default:
			return nil, errors.Errorf("unknown code %#x", op)
		}
	}
	return nil, errors.New("broken code")
}

func (m *vm) push(v rideType) {
	m.stack = append(m.stack, v)
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

//func (m *vm) scope() (*frame, int) {
//	n := len(m.calls) - 1
//	if n < 0 {
//		return nil, n
//	}
//	return &m.calls[n], n
//}

//func (m *vm) resolve(name string) (int, error) {
//	_ = name
//	//TODO: implement
//	return 0, errors.New("not implemented")
//}

//func (m *vm) returnPosition() int {
//	if l := len(m.calls); l > 0 {
//		return m.calls[l-1].back
//	}
//	return len(m.code)
//}
