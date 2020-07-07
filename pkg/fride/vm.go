package fride

import "fmt"

const (
	OpPush byte = iota
	OpPop
	OpRot
	OpFetch
	OpFetchMap
	OpTrue
	OpFalse
	OpNil
	OpNegate
	OpNot
	OpEqual
	OpEqualInt
	OpEqualString
	OpJump
	OpJumpIfTrue
	OpJumpIfFalse
	OpJumpBackward
	OpIn
	OpLess
	OpMore
	OpLessOrEqual
	OpMoreOrEqual
	OpAdd
	OpSubtract
	OpMultiply
	OpDivide
	OpModulo
	OpExponent
	OpRange
	OpMatches
	OpMatchesConst
	OpContains
	OpStartsWith
	OpEndsWith
	OpIndex
	OpSlice
	OpProperty
	OpCall
	OpCallFast
	OpMethod
	OpArray
	OpMap
	OpLen
	OpCast
	OpStore
	OpLoad
	OpInc
	OpBegin
	OpEnd // This opcode must be at the end of this list.
)

func Run(program *Program) (*Result, error) {
	if program == nil {
		return nil, fmt.Errorf("empty program")
	}

	m := vm{
		code:      program.Code,
		constants: program.Constanst,
		ip:        0,
	}
	return m.run()
}

type vm struct {
	code []byte
	constants
	ip int
}

func (m *vm) run() (*Result, error) {
	for m.ip < len(m.code) {
		op := m.code[m.ip]
		switch op {
		case OpPush:
		case OpPop:
		}
		m.ip++
	}
}
