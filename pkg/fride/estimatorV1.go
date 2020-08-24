package fride

import (
	"encoding/binary"

	"github.com/pkg/errors"
)

type estimatorV1 struct {
	code  []byte
	meta  *scriptMeta
	ip    int
	costs func(int) int
	names func(int) string
	scope *estimatorScopeV1
}

func (e *estimatorV1) estimate() error {
	e.ip = 0
	for e.ip < len(e.code) {
		op := e.code[e.ip]
		e.addBlocksEstimation()
		// Check here that we reached the end of branch
		err := e.scope.emergeOnEnd(e.ip)
		if err != nil {
			return err
		}
		e.ip++
		switch op {
		case OpHalt:
			return nil
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
				est += ret.args * 5 // 5 for every declared function argument
				err = e.scope.putFunction(ret.start, est)
			} else {
				// Returning from value expression, add expression entry point on stack of estimated values
				err = e.scope.putValue(ret.start)
			}
			if err != nil {
				return err
			}
			e.ip = ret.ret
		case OpPush: // Only constants can be pushed on stack, add 1 to estimation as for constant
			e.ip += 2 // Skip constant ID
			e.scope.add(1)
		case OpPop:
		case OpTrue, OpFalse: // True or False pushed to stack, add 1 as for constant
			e.scope.add(1)
		case OpJump:
			pos := e.arg16()
			err := e.scope.switchToAlternativeBranch(pos)
			if err != nil {
				return err
			}
		case OpJumpIfFalse:
			e.scope.add(1) // 1 - for IF statement
			e.scope.submerge()
			e.ip += 2 // Just skip address
		case OpProperty:
			e.scope.add(2)
			e.ip += 2 // Skip property constant ID
		case OpExternalCall:
			id := e.arg16()
			e.ip += 2 // Skip arguments count
			e.scope.add(e.costs(id))
		case OpCall:
			pos := e.arg16()
			args := e.arg16()
			if fe, ok := e.scope.function(pos); ok {
				//Function estimation already known, do not estimate again, just continue
				e.scope.add(fe)
				e.ip += 2 // Skip arguments count
			} else {
				e.scope.submerge()
				e.scope.push(call{start: pos, ret: e.ip, function: true, args: args})
				e.ip = pos // Continue to function body
			}
		case OpGlobal:
			e.scope.add(2)
			e.ip += 2 // Skip global function ID
		case OpLoad:
			e.scope.add(2)
			pos := e.arg16()
			if ok := e.scope.value(pos); ok {
				// Value estimation already exits in context, just move along
			} else {
				e.scope.submerge()
				e.scope.push(call{start: pos, ret: e.ip, function: false})
				e.ip = pos // Continue to expression's body
			}
		case OpLoadLocal:
			e.scope.add(3) // 3 for every actually used function argument
			e.ip += 2      // Skip argument number it's irrelevant
		default:
			return errors.Errorf("unknown operation code %#x", op)
		}
	}
	return errors.New("broken byte code")
}

func (e *estimatorV1) arg16() int {
	res := binary.BigEndian.Uint16(e.code[e.ip : e.ip+2])
	e.ip += 2
	return int(res)
}

func (e *estimatorV1) addBlocksEstimation() {
	if blocks, ok := e.meta.blocks[e.ip]; ok && blocks > 0 {
		e.scope.add(blocks * 5) // Add 5 for every block declared here
	}
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
