package fride

import (
	"encoding/binary"
	"fmt"

	"github.com/pkg/errors"
)

const (
	OpPush        byte = iota //00
	OpPop                     //01
	OpTrue                    //02
	OpFalse                   //03
	OpJump                    //04
	OpJumpIfFalse             //05
	OpProperty                //06
	OpCall                    //07
	OpStore                   //08
	OpLoad                    //09
	OpReturn                  //10
	OpBegin                   //11
	OpEnd                     //12
)

func Run(program *Program) (RideResult, error) {
	if program == nil {
		return nil, fmt.Errorf("empty program")
	}

	m := vm{
		code:      program.Code,
		constants: program.Constants,
		functions: program.Functions,
		stack:     make([]rideType, 0, 2),
		scopes:    make([]scope, 0, 2),
		ip:        0,
	}
	return m.run()
}

type scope map[rideString]rideType

type vm struct {
	code      []byte
	ip        int
	constants []rideType
	functions map[string]rideFunction
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
	m.scopes = append(m.scopes, make(scope))

	for m.ip < len(m.code) {
		op := m.code[m.ip]
		m.ip++
		switch op {
		case OpPush:
			m.push(m.constant())
		case OpPop:
			m.pop()
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
			obj := m.pop()
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
			in := make([]rideType, call.count)
			for i := call.count - 1; i >= 0; i-- {
				in[i] = m.pop()
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
		case OpStore:
			scope := m.scope()
			c := m.constant()
			key, ok := c.(rideString)
			if !ok {
				return nil, errors.Errorf("not a str value '%v' of type '%T'", c, c)
			}
			value := m.pop()
			scope[key] = value
		case OpLoad:
			scope := m.scope()
			c := m.constant()
			key, ok := c.(rideString)
			if !ok {
				return nil, errors.Errorf("not a str value '%v' of type '%T'", c, c)
			}
			m.push(scope[key])
		case OpBegin:
			scope := make(scope)
			m.scopes = append(m.scopes, scope)
		case OpEnd:
			m.scopes = m.scopes[:len(m.scopes)-1]
		case OpReturn:

		default:
			return nil, errors.Errorf("unknown code %#x", op)
		}
	}
	if len(m.stack) > 0 {
		v := m.pop()
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

func (m *vm) pop() rideType {
	value := m.stack[len(m.stack)-1]
	m.stack = m.stack[:len(m.stack)-1]
	return value
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

func (m *vm) scope() scope {
	if l := len(m.scopes); l > 0 {
		return m.scopes[l-1]
	}
	return nil
}
