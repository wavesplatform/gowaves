package fride

import (
	"encoding/binary"

	"github.com/pkg/errors"
)

func Run(program *Program) (RideResult, error) {
	if program == nil {
		return nil, errors.New("empty program")
	}
	fs, err := selectFunctions(program.LibVersion)
	if err != nil {
		return nil, errors.Wrap(err, "run")
	}
	gcs, err := selectConstants(program.LibVersion)
	if err != nil {
		return nil, errors.Wrap(err, "run")
	}
	np, err := selectFunctionNameProvider(program.LibVersion)
	if err != nil {
		return nil, errors.Wrap(err, "run")
	}
	m := vm{
		code:         program.Code,
		constants:    program.Constants,
		functions:    fs,
		globals:      gcs,
		stack:        make([]rideType, 0, 2),
		calls:        make([]frame, 0, 2),
		ip:           0,
		functionName: np,
	}
	return m.run()
}

type frame struct {
	function bool
	back     int
	args     []rideValue
}

func newExpressionFrame(pos int) frame {
	return frame{
		back: pos,
	}
}

func newFunctionFrame(pos int, args []rideValue) frame {
	return frame{
		function: true,
		back:     pos,
		args:     args,
	}
}

type vm struct {
	code         []byte
	ip           int
	constants    []rideType
	functions    func(int) rideFunction
	globals      func(int) rideConstructor
	stack        []rideType
	calls        []frame
	functionName func(int) string
}

func (m *vm) run() (RideResult, error) {
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
			m.ip = int(pos)
		case OpJumpIfFalse:
			pos := m.arg16()
			v, ok := m.current().(rideBoolean)
			if !ok {
				return nil, errors.Errorf("not a boolean value '%v' of type '%T'", m.current(), m.current())
			}
			if !v {
				m.ip = int(pos)
			}
		case OpProperty:
			obj, err := m.pop()
			if err != nil {
				return nil, errors.Wrap(err, "failed to get object")
			}
			prop := m.constant()
			v, err := fetch(obj, prop)
			if err != nil {
				return nil, err
			}
			m.push(v)
		case OpCall:
			pos := m.arg16()
			frame := newFunctionFrame(m.ip, len(m.stack)) // Creating new function frame with return position
			m.frames = append(m.frames, frame)
			m.ip = int(pos) // Continue to function
		case OpExternalCall:
			// Before calling external function all parameters must be evaluated and placed on stack
			id := m.code[m.ip]
			m.ip++
			cnt := int(m.code[m.ip])
			m.ip++
			in := make([]rideType, cnt) // Prepare input parameters for external call
			for i := cnt - 1; i >= 0; i-- {
				v, err := m.pop()
				if err != nil {
					return nil, errors.Wrapf(err, "failed to call external function '%s'", m.functionName(int(id)))
				}
				in[i] = v
			}
			fn := m.functions(int(id))
			if fn == nil {
				return nil, errors.Errorf("external function '%s' not implemented", m.functionName(int(id)))
			}
			res, err := fn(in...)
			if err != nil {
				return nil, err
			}
			m.push(res)
		case OpLoad: // Evaluate expression behind a LET declaration
			pos := m.arg16()
			frame := newFrame(m.ip) // Creating new function frame with return position
			m.frames = append(m.frames, frame)
			m.ip = int(pos) // Continue to expression
		case OpLoadLocal:

		case OpReturn:
			l := len(m.frames)
			var f frame
			f, m.frames = m.frames[l-1], m.frames[:l-1]
			m.ip = f.back
		case OpHalt:
			if len(m.stack) > 0 {
				v, err := m.pop()
				if err != nil {
					return nil, errors.Wrap(err, "failed to get result value")
				}
				switch tv := v.(type) {
				case rideBoolean:
					return ScriptResult(tv), nil
				default:
					return nil, errors.Errorf("unexpected result value '%v' of type '%T'", v, v)
				}
			}
			return nil, errors.New("no result after script execution")
		case OpGlobal:
		case OpBlockDeclaration:
			//Meta information - nothing to do
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

func (m *vm) arg16() uint16 {
	//TODO: add check
	res := binary.BigEndian.Uint16(m.code[m.ip : m.ip+2])
	m.ip += 2
	return res
}

func (m *vm) arg8() uint8 {
	//TODO: add check
	res := m.code[m.ip]
	m.ip++
	return res
}

func (m *vm) constant() rideType {
	//TODO: add check
	return m.constants[m.arg16()]
}

func (m *vm) scope() (*frame, int) {
	n := len(m.frames) - 1
	if n < 0 {
		return nil, n
	}
	return &m.frames[n], n
}

func (m *vm) resolve(name string) (int, error) {
	_ = name
	//TODO: implement
	return 0, errors.New("not implemented")
}

func (m *vm) returnPosition() int {
	if l := len(m.frames); l > 0 {
		return m.frames[l-1].back
	}
	return len(m.code)
}

func (m *vm) value(name rideString) (rideType, error) {
	s, n := m.scope()
	switch n {
	case -1:
		return nil, errors.Errorf("no frame to look up variable '%s'", name)
	case 0:
		v, ok := s.variables[name]
		if !ok {
			return nil, errors.Errorf("variable '%s' not found", name)
		}
		return v, nil
	default:
		v, ok := s.variables[name]
		if ok {
			return v, nil
		}
		global := m.frames[0]
		v, ok = global.variables[name]
		if !ok {
			return nil, errors.Errorf("variable '%s' not found", name)
		}
		return v, nil
	}
}
