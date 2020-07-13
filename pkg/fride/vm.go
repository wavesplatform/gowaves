package fride

import "fmt"

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
)

func Run(program *Program) (*Result, error) {
	if program == nil {
		return nil, fmt.Errorf("empty program")
	}

	m := vm{
		code:      program.Code,
		constants: program.Constants,
		stack:     make([]interface{}, 0, 2),
		scopes:    make([]scope, 0, 2),
		ip:        0,
	}
	return m.run()
}

type scope map[string]interface{}

type call struct {
	name  string
	count int
}

type function struct {
}

type vm struct {
	code      []byte
	ip        int
	constants []interface{}
	stack     []interface{}
	scopes    []scope
}

func (m *vm) run() (*Result, error) {
	if m.stack != nil {
		m.stack = m.stack[0:0]
	}
	if m.scopes != nil {
		m.scopes = m.scopes[0:0]
	}

	for m.ip < len(m.code) {
		op := m.code[m.ip]
		switch op {
		case OpPush:
			m.push(m.constant())
		case OpPop:
			m.pop()
		}
		m.ip++
	}
	return nil, nil
}

func (m *vm) push(value interface{}) {
	m.stack = append(m.stack, value)
}

func (m *vm) pop() interface{} {
	value := m.stack[len(m.stack)-1]
	m.stack = m.stack[:len(m.stack)-1]
	return value
}

func (m *vm) current() interface{} {
	return m.stack[len(m.stack)-1]
}

func (m *vm) arg() uint16 {

	if len(m.code) >= m.ip+2 {

	}
	b0, b1 := m.code[m.ip], m.code[m.ip+1]
	m.ip += 2
	return uint16(b0) | uint16(b1)<<8
}

func (m *vm) constant() interface{} {
	return m.constants[m.arg()]
}
