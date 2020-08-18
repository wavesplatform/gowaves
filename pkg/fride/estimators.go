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
	e := estimatorV1{
		code:  program.Code,
		ip:    0,
		costs: c,
		names: n,
		scope: newEstimatorScopeV1(),
	}
	e.scope.submerge() // Add initial scope to stack
	err = e.estimate()
	if err != nil {
		return 0, errors.Wrap(err, "estimate")
	}
	return e.scope.current(), nil
}

type estimate struct {
	ip  int
	est int
}
type expressionDescriptor struct {
	startPosition  int
	returnPosition int
	function       bool
}

type callStack struct {
	stack []expressionDescriptor
}

func newCallStack() *callStack {
	return &callStack{stack: make([]expressionDescriptor, 0)}
}

func (s *callStack) push(r expressionDescriptor) {
	s.stack = append(s.stack, r)
}

func (s *callStack) pop() (expressionDescriptor, error) {
	l := len(s.stack)
	if l == 0 {
		return expressionDescriptor{}, errors.New("empty call stack")
	}
	var r expressionDescriptor
	r, s.stack = s.stack[l-1], s.stack[:l-1]
	return r, nil
}

type valueStack struct {
	items []estimate
}

func newValueStack() *valueStack {
	return &valueStack{items: make([]estimate, 0)}
}

func (s *valueStack) push(e estimate) {
	s.items = append(s.items, e)
}

func (s *valueStack) pop() (estimate, error) {
	l := len(s.items)
	if l == 0 {
		return estimate{}, errors.New("empty value stack")
	}
	var e estimate
	e, s.items = s.items[l-1], s.items[:l-1]
	return e, nil
}

func (s *valueStack) mark() int {
	return len(s.items) - 1
}

func (s *valueStack) stash(i int) []estimate {
	var r []estimate
	r, s.items = s.items[i+1:], s.items[:i+1]
	return r
}

func (s *valueStack) get(p int) (int, bool) {
	for i := len(s.items) - 1; i >= 0; i-- {
		if s.items[i].ip == p {
			return s.items[i].est, true
		}
	}
	return 0, false
}

type branch struct {
	alternative     bool
	truthy          int
	falsy           int
	nextInstruction int
	stackPosition   int
}

func (p *branch) add(n int) {
	if p.alternative {
		p.falsy += n
	} else {
		p.truthy += n
	}
}

func (p *branch) get() int {
	if p.truthy > p.falsy {
		return p.truthy
	}
	return p.falsy
}

type branchesStack struct {
	stack []*branch
}

func (s *branchesStack) push(b *branch) {
	s.stack = append(s.stack, b)
}

func (s *branchesStack) pop() (*branch, error) {
	l := len(s.stack)
	if l == 0 {
		return nil, errors.New("empty branches stack")
	}
	var b *branch
	b, s.stack = s.stack[l-1], s.stack[:l-1]
	return b, nil
}

func (s *branchesStack) peek() (*branch, error) {
	l := len(s.stack)
	if l == 0 {
		return nil, errors.Errorf("empty branches stack")
	}
	return s.stack[l-1], nil
}

func newBranchesStack(b *branch) *branchesStack {
	return &branchesStack{stack: []*branch{b}}
}

type estimatorV1 struct {
	code  []byte
	ip    int
	costs func(int) int
	names func(int) string
	scope *estimatorScopeV1
}

func (e *estimatorV1) estimate() error {
	e.ip = 0
	for e.ip < len(e.code) {
		op := e.code[e.ip]
		// Check here that we reached the end of branch
		err := e.scope.emergeOnEnd(e.ip)
		if err != nil {
			return err
		}
		e.ip++
		switch op {
		case OpPush: // Only constants can be pushed on stack, add 1 to estimation as for constant
			e.scope.add(1)
			e.ip += 2
		case OpPop:
		case OpTrue, OpFalse: // True or False pushed to stack, add 1 as for constant
			e.scope.add(1)
		case OpJump:
			err := e.scope.switchToAlternativeBranch(e.jumpPosition())
			if err != nil {
				return err
			}
			e.ip += 2
		case OpJumpIfFalse:
			e.scope.add(1)
			e.scope.submerge()
			e.ip += 2
		case OpProperty:
			e.scope.add(2)
			e.ip += 2
		case OpCall:
			jp := e.jumpPosition()
			if fe, ok := e.scope.function(jp); ok {
				//Function estimation already known, do not estimate again, just continue
				e.scope.add(fe)
				e.ip += 2
			} else {
				e.scope.submerge()
				e.scope.push(call{start: jp, ret: e.ip + 2, function: true})
				e.ip = jp // Continue to function body
			}
		case OpExternalCall:
			id := int(e.code[e.ip])
			e.scope.add(e.costs(id))
			e.ip += 2
		case OpLoad:
			e.scope.add(2)
			jp := e.jumpPosition()
			if ok := e.scope.value(jp); ok {
				// Value estimation already exits in context, just move along
				e.ip += 2
			} else {
				e.scope.submerge()
				e.scope.push(call{start: jp, ret: e.ip + 2, function: false})
				e.ip = jp // Continue to expression's body
			}
		case OpLoadLocal:
			e.scope.add(8) // 5 for every function argument
			e.ip += 2
		case OpReturn:
			ret, err := e.scope.pop()
			if err != nil {
				return err
			}
			est := e.scope.current()
			err = e.scope.emerge()
			if err != nil {
				return err
			}
			if ret.function {
				// Returning from function, store its estimation
				err = e.scope.putFunction(ret.start, est)
			} else {
				// Returning from value expression, add expression entry point on stack of estimated values
				err = e.scope.putValue(ret.start)
			}
			if err != nil {
				return err
			}
			e.ip = ret.ret
		case OpHalt:
			return nil
		case OpGlobal:
			e.scope.add(2)
			e.ip++
		case OpBlockDeclaration:
			e.scope.add(5)
		default:
			return errors.Errorf("unknown operation code %#x", op)
		}
	}
	return errors.New("broken byte code")
}

func (e *estimatorV1) jumpPosition() int {
	return int(binary.BigEndian.Uint16(e.code[e.ip : e.ip+2]))
}

// call structure describes how to return from the call
type call struct {
	start    int  // Start position of the function/expression, used as reference value
	ret      int  // Return position, where to go after returning form function/expression
	function bool // Flag to distinguish function and expression calls
}

// estimationFrame holds values stacks and estimations for both main and alternative branches of execution flow
type estimationFrame struct {
	alternative     bool  // Flag that indicates to count alternative branch estimation
	trunk           int   // Estimation of the trunk
	trunkStack      []int // Values stack of trunk
	branch          int   // Estimation of the alternative branch
	branchStack     []int // Values stack of alternative branch
	nextInstruction int   // Pointer to the instruction next to an alternative branch end
}

func (e *estimationFrame) add(estimation int) {
	if e.alternative {
		e.branch += estimation
	} else {
		e.trunk += estimation
	}
}

func (e *estimationFrame) get() int {
	if e.trunk > e.branch {
		return e.trunk
	}
	return e.branch
}

func (e *estimationFrame) put(pos int) {
	if e.alternative {
		e.branchStack = append(e.branchStack, pos)
	} else {
		e.trunkStack = append(e.trunkStack, pos)
	}
}

func (e *estimationFrame) value(pos int) bool {
	var stack []int
	if e.alternative {
		stack = e.branchStack
	} else {
		stack = e.trunkStack
	}
	for i := len(stack) - 1; i >= 0; i-- {
		if stack[i] == pos {
			return true
		}
	}
	return false
}

// estimatorScopeV1 scope management structure for the V1
type estimatorScopeV1 struct {
	estimations []*estimationFrame // Stack of estimation frames
	calls       []call             // Call stack
	functions   map[int]int        // Dictionary of user function estimations
}

func newEstimatorScopeV1() *estimatorScopeV1 {
	return &estimatorScopeV1{
		estimations: make([]*estimationFrame, 0),
		functions:   make(map[int]int),
	}
}

func (s *estimatorScopeV1) add(estimation int) {
	if l := len(s.estimations); l > 0 {
		s.estimations[l-1].add(estimation)
	}
}

func (s *estimatorScopeV1) push(c call) {
	s.calls = append(s.calls, c)
}

func (s *estimatorScopeV1) pop() (call, error) {
	l := len(s.calls)
	if l == 0 {
		return call{}, errors.New("empty call stack")
	}
	var c call
	c, s.calls = s.calls[l-1], s.calls[:l-1]
	return c, nil
}

func (s *estimatorScopeV1) putFunction(pos, cost int) error {
	if e, ok := s.functions[pos]; ok {
		return errors.Errorf("function at position %d already estimated on %d points", pos, e)
	}
	s.functions[pos] = cost
	return nil
}

func (s *estimatorScopeV1) function(pos int) (int, bool) {
	if e, ok := s.functions[pos]; ok {
		return e, true
	}
	return 0, false
}

func (s *estimatorScopeV1) putValue(pos int) error {
	el := len(s.estimations)
	if el == 0 {
		return errors.New("empty estimations stack")
	}
	s.estimations[el-1].put(pos)
	return nil
}

func (s *estimatorScopeV1) value(pos int) bool {
	for i := len(s.estimations) - 1; i >= 0; i-- {
		if s.estimations[i].value(pos) {
			return true
		}
	}
	return false
}

func (s *estimatorScopeV1) submerge() {
	s.estimations = append(s.estimations, &estimationFrame{
		alternative:     false,
		trunk:           0,
		trunkStack:      make([]int, 0),
		branch:          0,
		branchStack:     make([]int, 0),
		nextInstruction: -1,
	})
}

func (s *estimatorScopeV1) emerge() error {
	if l := len(s.estimations); l > 0 {
		var e *estimationFrame
		e, s.estimations = s.estimations[l-1], s.estimations[:l-1]
		if l = len(s.estimations); l > 0 {
			pe := s.estimations[l-1]
			pe.add(e.get())
			return nil
		}
		return errors.New("empty estimations stack")
	}
	return errors.New("empty estimations stack")
}

// Check that topmost frame points on the same instruction as passed instruction pointer
func (s *estimatorScopeV1) emergeOnEnd(ip int) error {
	if l := len(s.estimations); l > 0 {
		e := s.estimations[l-1]
		if e.nextInstruction == ip {
			//Choose the most expensive branch and collapse estimationFrame
			s.estimations = s.estimations[:l-1]
			if l = len(s.estimations); l > 0 {
				pe := s.estimations[l-1]
				pe.add(e.get())
				return nil
			}
			return errors.New("empty estimations stack")
		}
		return nil
	}
	return errors.New("empty estimations stack")
}

func (s *estimatorScopeV1) switchToAlternativeBranch(ip int) error {
	if l := len(s.estimations); l > 0 {
		e := s.estimations[l-1]
		e.alternative = true
		e.nextInstruction = ip
		return nil
	}
	return errors.New("empty estimations stack")
}

// returns estimation of the topmost frame on the stack
func (s *estimatorScopeV1) current() int {
	if l := len(s.estimations); l > 0 {
		return s.estimations[l-1].get()
	}
	return 0
}
