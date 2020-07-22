package fride

import (
	"encoding/binary"
	"fmt"

	"github.com/pkg/errors"
)

const (
	OpPush        byte = iota //00 - 3
	OpPop                     //01 - 1
	OpTrue                    //02 - 1
	OpFalse                   //03 - 1
	OpJump                    //04 - 3
	OpJumpIfFalse             //05 - 3
	OpProperty                //06 - 3
	OpCall                    //07 - 3
	OpStore                   //08 - 3
	OpLoad                    //09 - 3
	OpBegin                   //0a - 1
	OpEnd                     //0b - 1
)

func Run(program *Program) (RideResult, error) {
	if program == nil {
		return nil, fmt.Errorf("empty program")
	}
	var functions map[string]rideFunction
	switch program.LibVersion {
	case 1, 2:
		functions = functionsV12()
	case 3:
		functions = functionsV3()
	case 4:
		functions = functionsV4()
	default:
		return nil, errors.Errorf("unsupported library version %d", program.LibVersion)
	}

	m := vm{
		code:          program.Code,
		constants:     program.Constants,
		functions:     functions,
		userFunctions: program.Functions,
		stack:         make([]rideType, 0, 2),
		scopes:        make([]scope, 0, 2),
		ip:            0,
		entryPoint:    program.EntryPoint,
	}
	return m.run()
}

type scope struct {
	back      int
	variables map[rideString]rideType
}

func newScope(pos int) scope {
	return scope{
		back:      pos,
		variables: make(map[rideString]rideType),
	}
}

type vm struct {
	code          []byte
	ip            int
	constants     []rideType
	functions     map[string]rideFunction
	userFunctions map[string]int
	entryPoint    int
	stack         []rideType
	scopes        []scope
}

func (m *vm) run() (RideResult, error) {
	if m.stack != nil {
		m.stack = m.stack[0:0]
	}
	if m.scopes != nil {
		m.scopes = m.scopes[0:0]
	}
	m.ip = m.entryPoint
	m.scopes = append(m.scopes, newScope(0))

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
			offset := m.arg()
			m.ip += int(offset)
		case OpJumpIfFalse:
			offset := m.arg()
			v, ok := m.current().(rideBoolean)
			if !ok {
				return nil, errors.Errorf("not a boolean value '%v' of type '%T'", m.current(), m.current())
			}
			if !v {
				m.ip += int(offset)
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
			c := m.constant()
			call, ok := c.(rideCall)
			if !ok {
				return nil, errors.Errorf("not a call descriptor '%v' of type '%T'", c, c)
			}
			fp, ok := m.userFunctions[call.name]
			if ok {
				m.push(rideLong(m.ip))
				m.ip = fp
			} else {
				in := make([]rideType, call.count)
				for i := call.count - 1; i >= 0; i-- {
					v, err := m.pop()
					if err != nil {
						return nil, errors.Wrapf(err, "failed to call function '%s'", call.name)
					}
					in[i] = v
				}
				fn, err := m.fetchFunction(call.name)
				if err != nil {
					return nil, err
				}
				res, err := fn(in...)
				if err != nil {
					return nil, err
				}
				m.push(res)
			}
		case OpStore:
			scope := m.scope()
			c := m.constant()
			key, ok := c.(rideString)
			if !ok {
				return nil, errors.Errorf("not a str value '%v' of type '%T'", c, c)
			}
			value, err := m.pop()
			if err != nil {
				return nil, errors.Wrapf(err, "failed to store variable '%s'", key)
			}
			scope.variables[key] = value
		case OpLoad:
			scope := m.scope()
			c := m.constant()
			key, ok := c.(rideString)
			if !ok {
				return nil, errors.Errorf("not a str value '%v' of type '%T'", c, c)
			}
			m.push(scope.variables[key])
		case OpBegin:
			pos, err := m.pop()
			if err != nil {
				return nil, errors.Wrap(err, "failed to start function execution")
			}
			p, ok := pos.(rideLong)
			if !ok {
				return nil, errors.Errorf("invalid return position '%v' of type '%T'", pos, pos)
			}
			scope := newScope(int(p))
			m.scopes = append(m.scopes, scope)
		case OpEnd:
			s := m.scope()
			if s == nil {
				return nil, errors.New("failed to exit from function: no scope")
			}
			m.ip = s.back
			m.scopes = m.scopes[:len(m.scopes)-1]
		default:
			return nil, errors.Errorf("unknown code %#x", op)
		}
	}
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
}

func (m *vm) push(v rideType) {
	m.stack = append(m.stack, v)
}

func (m *vm) pop() (rideType, error) {
	if len(m.stack) == 0 {
		return nil, errors.New("empty stack")
	}
	value := m.stack[len(m.stack)-1]
	m.stack = m.stack[:len(m.stack)-1]
	return value, nil
}

func (m *vm) current() rideType {
	return m.stack[len(m.stack)-1]
}

func (m *vm) arg() uint16 {
	//TODO: add check
	res := binary.BigEndian.Uint16(m.code[m.ip : m.ip+2])
	m.ip += 2
	return res
}

func (m *vm) constant() rideType {
	return m.constants[m.arg()]
}

func (m *vm) fetchFunction(name string) (rideFunction, error) {
	f, ok := m.functions[name]
	if !ok {
		return nil, errors.Errorf("function '%s' not found", name)
	}
	return f, nil
}

func (m *vm) scope() *scope {
	if l := len(m.scopes); l > 0 {
		return &m.scopes[l-1]
	}
	return nil
}
