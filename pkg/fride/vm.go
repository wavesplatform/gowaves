package fride

import (
	"encoding/binary"

	"github.com/pkg/errors"
)

func Run(program *Program) (RideResult, error) {
	if program == nil {
		return nil, errors.New("empty program")
	}
	fs, err := functions(program.LibVersion)
	if err != nil {
		return nil, errors.Wrap(err, "run")
	}
	m := vm{
		code:      program.Code,
		constants: program.Constants,
		functions: fs,
		stack:     make([]rideType, 0, 2),
		scopes:    make([]scope, 0, 2),
		ip:        0,
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
	code      []byte
	ip        int
	constants []rideType
	functions func(int) rideFunction
	stack     []rideType
	scopes    []scope
}

func (m *vm) run() (RideResult, error) {
	if m.stack != nil {
		m.stack = m.stack[0:0]
	}
	if m.scopes != nil {
		m.scopes = m.scopes[0:0]
	}
	m.ip = 0
	m.scopes = append(m.scopes, newScope(len(m.code)))

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
			offset := m.arg16()
			m.ip += int(offset)
		case OpJumpIfFalse:
			offset := m.arg16()
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
			name, ok := c.(rideString)
			if !ok {
				return nil, errors.Errorf("not a function name but '%v' of type '%s'", c, c.instanceOf())
			}
			fp, err := m.resolve(string(name))
			if err != nil {
				return nil, errors.Wrapf(err, "failed to call user function '%s'", name)
			}
			scope := newScope(m.ip) // Creating new function scope with return position
			m.scopes = append(m.scopes, scope)
			m.ip = fp // Continue to function code
		case OpExternalCall:
			id := m.code[m.ip]
			m.ip++
			cnt := int(m.code[m.ip])
			m.ip++
			in := make([]rideType, cnt) // Prepare input parameters for external call
			for i := cnt - 1; i >= 0; i-- {
				v, err := m.pop()
				if err != nil {
					return nil, errors.Wrapf(err, "failed to call external function '%s'", functionName(int(id)))
				}
				in[i] = v
			}
			fn, err := m.fetchFunction(int(id))
			if err != nil {
				return nil, err
			}
			res, err := fn(in...)
			if err != nil {
				return nil, err
			}
			m.push(res)
		case OpStore:
			scope, n := m.scope()
			if n < 0 {
				return nil, errors.Errorf("failed to store variable: no scope")
			}
			c := m.constant()
			name, ok := c.(rideString)
			if !ok {
				return nil, errors.Errorf("not a string value '%v' of type '%T'", c, c)
			}
			value, err := m.pop()
			if err != nil {
				return nil, errors.Wrapf(err, "failed to store variable '%s'", name)
			}
			scope.variables[name] = value
		case OpLoad:
			c := m.constant()
			name, ok := c.(rideString)
			if !ok {
				return nil, errors.Errorf("not a str value '%v' of type '%T'", c, c)
			}
			v, err := m.value(name)
			if err != nil {
				return nil, errors.Wrap(err, "failed to load variable")
			}
			m.push(v)
		case OpReturn:
			m.ip = m.returnPosition()             // Continue from return position
			m.scopes = m.scopes[:len(m.scopes)-1] // Removing the current call stack frame
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

func (m *vm) fetchFunction(id int) (rideFunction, error) {
	//TODO: implement
	return nil, errors.New("not implemented")
}

func (m *vm) scope() (*scope, int) {
	n := len(m.scopes) - 1
	if n < 0 {
		return nil, n
	}
	return &m.scopes[n], n
}

func (m *vm) resolve(name string) (int, error) {
	_ = name
	//TODO: implement
	return 0, errors.New("not implemented")
}

func (m *vm) returnPosition() int {
	if l := len(m.scopes); l > 0 {
		return m.scopes[l-1].back
	}
	return len(m.code)
}

func (m *vm) value(name rideString) (rideType, error) {
	s, n := m.scope()
	switch n {
	case -1:
		return nil, errors.Errorf("no scope to look up variable '%s'", name)
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
		global := m.scopes[0]
		v, ok = global.variables[name]
		if !ok {
			return nil, errors.Errorf("variable '%s' not found", name)
		}
		return v, nil
	}
}
