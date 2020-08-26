package fride

//
//import (
//	"encoding/binary"
//
//	"github.com/pkg/errors"
//)
//
//// Estimator V3 works as follows:
//// It registers all expression evaluations, and sum up actually used afterward
//// User functions estimated without arguments by itself. User function call adds to expression.
//// User functions arguments estimated upon call of the function.
//
//type estimationFrameV3 interface {
//	increase(cost int)
//	cost() int
//}
//
//type functionFrameV3 struct {
//	pos  int
//	cost int
//}
//
//type forkFrameV3 struct {
//	alternative     bool // Flag that indicates to count alternative branch estimation
//	trunk           int  // Estimation of the trunk
//	branch          int  // Estimation of the alternative branch
//	nextInstruction int  // Pointer to the instruction next to an alternative branch end
//}
//
//func (e *forkFrameV3) increase(cost int) {
//	if e.alternative {
//		e.branch += cost
//	} else {
//		e.trunk += cost
//	}
//}
//
//func (e *forkFrameV3) cost() int {
//	if e.trunk > e.branch {
//		return e.trunk
//	}
//	return e.branch
//}
//
//// call structure describes how to return from the call
//type callV3 struct {
//	start int // Start position of the function/expression, used as reference value
//	ret   int // Return position, where to go after returning form function/expression
//}
//
//// estimatorScopeV3 scope management structure for the estimatorV3
//type estimatorScopeV3 struct {
//	estimations []*estimationFrameV3 // Stack of estimation frames
//	calls       []callV3             // Call stack
//	functions   map[int]int          // Dictionary of user function estimations
//	expressions map[int]int          // Dictionary of expression estimations
//}
//
//func newEstimatorScopeV3() *estimatorScopeV3 {
//	return &estimatorScopeV3{
//		estimations: make([]*estimationFrameV3, 0),
//		functions:   make(map[int]int),
//	}
//}
//
//func (s *estimatorScopeV3) add(estimation int) {
//	if l := len(s.estimations); l > 0 {
//		s.estimations[l-1].add(estimation)
//	}
//}
//
//func (s *estimatorScopeV3) push(c callV3) {
//	s.calls = append(s.calls, c)
//}
//
//func (s *estimatorScopeV3) pop() (callV3, error) {
//	l := len(s.calls)
//	if l == 0 {
//		return callV3{}, errors.New("empty call stack")
//	}
//	var c callV3
//	c, s.calls = s.calls[l-1], s.calls[:l-1]
//	return c, nil
//}
//
//func (s *estimatorScopeV3) putFunction(pos, cost int) error {
//	if e, ok := s.functions[pos]; ok {
//		return errors.Errorf("function at position %d already estimated on %d points", pos, e)
//	}
//	s.functions[pos] = cost
//	return nil
//}
//
//func (s *estimatorScopeV3) function(pos int) (int, bool) {
//	if e, ok := s.functions[pos]; ok {
//		return e, true
//	}
//	return 0, false
//}
//
//func (s *estimatorScopeV3) putExpression(pos, cost int) error {
//	if e, ok := s.expressions[pos]; ok {
//		return errors.Errorf("expression at position %d already estimated on %d points", pos, e)
//	}
//	s.expressions[pos] = cost
//	return nil
//}
//
//func (s *estimatorScopeV3) expression(pos int) (int, bool) {
//	if e, ok := s.expressions[pos]; ok {
//		return e, true
//	}
//	return 0, false
//}
//
//func (s *estimatorScopeV3) submerge() {
//	s.estimations = append(s.estimations, &estimationFrame{
//		alternative:     false,
//		trunk:           0,
//		trunkStack:      make([]int, 0),
//		branch:          0,
//		branchStack:     make([]int, 0),
//		nextInstruction: -1,
//	})
//}
//
//func (s *estimatorScopeV3) emerge() error {
//	if l := len(s.estimations); l > 0 {
//		var e *estimationFrame
//		e, s.estimations = s.estimations[l-1], s.estimations[:l-1]
//		if l = len(s.estimations); l > 0 {
//			pe := s.estimations[l-1]
//			pe.add(e.get())
//			return nil
//		}
//		return errors.New("empty estimations stack")
//	}
//	return errors.New("empty estimations stack")
//}
//
//// Check that topmost frame points on the same instruction as passed instruction pointer
//func (s *estimatorScopeV3) emergeOnEnd(ip int) error {
//	if l := len(s.estimations); l > 0 {
//		e := s.estimations[l-1]
//		if e.nextInstruction == ip {
//			//Choose the most expensive branch and collapse estimationFrame
//			s.estimations = s.estimations[:l-1]
//			if l = len(s.estimations); l > 0 {
//				pe := s.estimations[l-1]
//				pe.add(e.get())
//				return nil
//			}
//			return errors.New("empty estimations stack")
//		}
//		return nil
//	}
//	return errors.New("empty estimations stack")
//}
//
//func (s *estimatorScopeV3) switchToAlternativeBranch(ip int) error {
//	if l := len(s.estimations); l > 0 {
//		e := s.estimations[l-1]
//		e.alternative = true
//		e.nextInstruction = ip
//		return nil
//	}
//	return errors.New("empty estimations stack")
//}
//
//// returns estimation of the topmost frame on the stack
//func (s *estimatorScopeV3) current() int {
//	if l := len(s.estimations); l > 0 {
//		return s.estimations[l-1].get()
//	}
//	return 0
//}
//
//type estimatorV3 struct {
//	code  []byte
//	ip    int
//	costs func(int) int
//	names func(int) string
//	scope *estimatorScopeV3
//}
//
//func (e *estimatorV3) estimate() error {
//	e.ip = 0
//	for e.ip < len(e.code) {
//		op := e.code[e.ip]
//		// Check here that we reached the end of branch
//		err := e.scope.emergeOnEnd(e.ip)
//		if err != nil {
//			return err
//		}
//		e.ip++
//		switch op {
//		case OpHalt:
//			return nil
//		case OpReturn:
//			ret, err := e.scope.pop()
//			if err != nil {
//				return err
//			}
//			est := e.scope.current()
//			err = e.scope.emerge()
//			if err != nil {
//				return err
//			}
//			// Returning from function/expression, store its estimation
//			err = e.scope.putEstimation(ret.start, est)
//			if err != nil {
//				return err
//			}
//			e.ip = ret.ret
//		case OpPush: // Only constants can be pushed on stack, add 1 to estimation as for constant
//			e.ip += 2      // Skip constant ID
//			e.scope.add(1) // 1 for a constant
//		case OpPop:
//		case OpTrue, OpFalse: // True or False pushed to stack, add 1 as for constant
//			e.scope.add(1) // 1 for a boolean constant
//		case OpJump:
//			pos := e.arg()
//			err := e.scope.switchToAlternativeBranch(pos)
//			if err != nil {
//				return err
//			}
//		case OpJumpIfFalse:
//			e.scope.add(1) // 1 for IF statement
//			e.scope.submerge()
//			e.ip += 2 // Just skip address
//		case OpProperty:
//			e.scope.add(1) // 1 for accessing a property of an object
//			e.ip += 2      // Skip property constant ID
//		case OpExternalCall:
//			id := e.arg()
//			e.ip += 2                // Skip arguments count
//			e.scope.add(e.costs(id)) // add cost of an external function
//		case OpCall:
//			pos := e.arg()
//			args := e.arg()
//			if fe, ok := e.scope.function(pos); ok {
//				//Function estimation already known, do not estimate again, just continue
//				e.scope.add(fe)
//				e.ip += 2 // Skip arguments count
//			} else {
//				e.scope.submerge()
//				e.scope.push(call{start: pos, ret: e.ip, function: true, args: args})
//				e.ip = pos // Continue to function body
//			}
//		case OpGlobal:
//			e.scope.add(1) // 1 for referencing global expression
//			e.ip += 2      // Skip global expression ID
//		case OpLoad:
//			e.scope.add(1) // 1 for referencing expression
//			pos := e.arg()
//			if ok := e.scope.expression(pos); ok {
//				// Value estimation already exits in context, just move along
//			} else {
//				e.scope.submerge()
//				e.scope.push(call{start: pos, ret: e.ip, function: false})
//				e.ip = pos // Continue to expression's body
//			}
//		case OpLoadLocal:
//			e.scope.add(3) // 3 for every actually used function argument
//			e.ip += 2      // Skip argument number it's irrelevant
//		default:
//			return errors.Errorf("unknown operation code %#x", op)
//		}
//	}
//	return errors.New("broken byte code")
//}
//
//func (e *estimatorV3) arg() int {
//	res := binary.BigEndian.Uint16(e.code[e.ip : e.ip+2])
//	e.ip += 2
//	return int(res)
//}
