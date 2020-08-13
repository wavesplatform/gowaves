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
		code:      program.Code,
		ip:        0,
		costs:     c,
		names:     n,
		callStack: make([]int, 0),
		stack:     make([]pair, 0),
		scope:     scope,
	}
	err = e.estimate()
	if err != nil {
		return 0, errors.Wrap(err, "estimate")
	}
	for _, v := range e.scope {
		if !v {
			e.add(5)
		}
	}
	return e.estimation, nil
}

type pair struct {
	branch      bool
	truthy      int
	truthyScope []int
	falsy       int
	falsyScope  []int
	ret         int
}

func (p *pair) add(n int) {
	if p.branch {
		p.truthy += n
	} else {
		p.falsy += n
	}
}

func (p *pair) mark(s int) {
	if p.branch {
		p.truthyScope = append(p.truthyScope, s)
	} else {
		p.falsyScope = append(p.falsyScope, s)
	}
}

func (p *pair) get() (int, []int) {
	if p.truthy > p.falsy {
		return p.truthy, p.truthyScope
	}
	return p.falsy, p.falsyScope
}

type estimatorV1 struct {
	code       []byte
	ip         int
	costs      func(int) int
	names      func(int) string
	estimation int
	callStack  []int
	stack      []pair
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

func (e *estimatorV1) add(n int) {
	if l := len(e.stack); l > 0 {
		e.stack[l-1].add(n)
		return
	}
	e.estimation += n
}

func (e *estimatorV1) mark(p int) {
	if l := len(e.stack); l > 0 {
		e.stack[l-1].mark(p)
	}
	e.scope[p] = true
}

func (e *estimatorV1) estimate() error {
	e.ip = 0
	for e.ip < len(e.code) {
		op := e.code[e.ip]
		e.ip++
		switch op {
		case OpPush: // Only constants can be pushed on stack, add 1 to estimation as for constant
			e.add(1)
			e.ip += 2
		case OpPop:
		case OpTrue, OpFalse: // True or False pushed to stack, add 1 as for constant
			e.add(1)
		case OpJump:
			// start false branch context
			if l := len(e.stack); l > 0 {
				e.stack[l-1].branch = false
				e.stack[l-1].ret = e.jumpPosition() + 1
			}
			e.ip += 2
		case OpJumpIfFalse:
			e.add(1)
			// start truthy branch context
			e.stack = append(e.stack, pair{branch: true})
			// remember the end position
			e.ip += 2
		case OpProperty:
			e.add(2)
			e.ip += 2
		case OpCall:
			e.callStack = append(e.callStack, e.ip+2)
			e.ip += e.jumpPosition()
		case OpExternalCall:
			id := int(e.code[e.ip])
			e.add(e.costs(id))
			e.ip += 2
		case OpLoad:
			e.add(2)
			p := e.jumpPosition()
			if b, ok := e.scope[p]; ok && b {
				e.ip += 2
			} else {
				e.mark(p)
				e.callStack = append(e.callStack, e.ip+2)
				e.ip = p
			}
		case OpLoadLocal:
			e.ip += 2
		case OpReturn:
			if l := len(e.stack); l > 0 && e.stack[l-1].ret == e.ip {
				var p pair
				p, e.stack = e.stack[l-1], e.stack[:l-1]
				be, bs := p.get()
				e.add(be)
				for _, s := range bs {
					e.scope[s] = false
				}
			}
			if l := len(e.callStack); l > 0 {
				e.add(5)
				e.ip, e.callStack = e.callStack[l-1], e.callStack[:l-1]
			} else {
				return errors.New("empty call stack")
			}
		case OpHalt:
			if l := len(e.stack); l > 0 {
				var p pair
				p, e.stack = e.stack[l-1], e.stack[:l-1]
				be, bs := p.get()
				e.add(be)
				for _, s := range bs {
					e.scope[s] = false
				}
			}
			return nil
		case OpGlobal:
			e.add(2)
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
