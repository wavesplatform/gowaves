package fride

import (
	"encoding/binary"

	"github.com/pkg/errors"
)

func EstimateV1(program *Program) (int, error) {
	if program == nil {
		return 0, errors.New("empty program")
	}
	c, err := selectCostProvider(program.LibVersion)
	if err != nil {
		return 0, errors.Wrap(err, "estimate")
	}
	n, err := selectFunctionNameProvider(program.LibVersion)
	if err != nil {
		return 0, errors.Wrap(err, "estimate")
	}
	scope, err := blocks(program.Code)
	if err != nil {
		return 0, errors.Wrap(err, "estimate")
	}
	e := estimatorV1{
		code:  program.Code,
		ip:    0,
		costs: c,
		names: n,
		stack: make([]int, 0),
		scope: scope,
	}
	err = e.estimate()
	if err != nil {
		return 0, errors.Wrap(err, "estimate")
	}
	for _, v := range e.scope {
		if !v {
			e.estimation += 5
		}
	}
	return e.estimation, nil
}

type estimatorV1 struct {
	code       []byte
	ip         int
	costs      func(int) int
	names      func(int) string
	estimation int
	stack      []int //call stack
	scope      map[int]bool
}

func blocks(code []byte) (map[int]bool, error) {
	r := make(map[int]bool)
	p := 0
	for p < len(code) {
		switch code[p] {
		case OpPop, OpTrue, OpFalse:
			p++
		case OpGlobal:
			p += 2
		case OpPush, OpJump, OpJumpIfFalse, OpCall, OpExternalCall, OpLoad, OpLoadLocal, OpProperty:
			p += 3
		case OpReturn, OpHalt:
			p++
			if p < len(code) {
				r[p] = false
			}
		default:
			return nil, errors.Errorf("unknown operation code %#x at position %d", code[p], p)
		}
	}
	return r, nil
}

func (e *estimatorV1) estimate() error {
	e.ip = 0
	for e.ip < len(e.code) {
		op := e.code[e.ip]
		e.ip++
		switch op {
		case OpPush: // Only constants can be pushed on stack, add 1 to estimation as for constant
			e.estimation++
			e.ip += 2
		case OpPop:
		case OpTrue, OpFalse: // True or False pushed to stack, add 1 as for constant
			e.estimation++
		case OpJump:
			e.ip += 2
		case OpJumpIfFalse:
			e.ip += 2
		case OpProperty:
			e.estimation += 2
			e.ip += 2
		case OpCall:
			e.stack = append(e.stack, e.ip+2)
			e.ip += e.jumpPosition()
		case OpExternalCall:
			id := int(e.code[e.ip])
			e.estimation += e.costs(id)
			e.ip += 2
		case OpLoad:
			e.estimation += 2
			p := e.jumpPosition()
			if b, ok := e.scope[p]; ok && b {
				e.ip += 2
			} else {
				e.scope[p] = true
				e.stack = append(e.stack, e.ip+2)
				e.ip = p
			}
		case OpLoadLocal:
			e.ip += 2
		case OpReturn:
			if len(e.stack) > 0 {
				e.estimation += 5
				l := len(e.stack)
				e.ip, e.stack = e.stack[l-1], e.stack[:l-1]
			} else {
				return errors.New("empty call stack")
			}
		case OpHalt:
			return nil
		case OpGlobal:
			e.estimation += 2
			e.ip++
		default:
			return errors.Errorf("unknown operation code %#x", op)
		}
	}
	return errors.New("broken byte code")
}

func (e *estimatorV1) jumpPosition() int {
	return int(binary.BigEndian.Uint16(e.code[e.ip : e.ip+2]))
}
