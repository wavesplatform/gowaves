package compiler

import (
	"slices"
	"strings"

	"github.com/wavesplatform/gowaves/pkg/ride/compiler/stdlib"
)

type frame struct {
	varsIndex  int
	funcsIndex int
}
type stack struct {
	frames []frame
	vars   []stdlib.Variable
	funcs  []stdlib.FunctionParams
}

func newStack() *stack {
	return &stack{
		frames: []frame{{}},
		vars:   make([]stdlib.Variable, 0),
		funcs:  make([]stdlib.FunctionParams, 0),
	}
}

func (s *stack) addFrame() {
	f := frame{
		varsIndex:  len(s.vars),
		funcsIndex: len(s.funcs),
	}
	s.frames = append(s.frames, f)
}

func (s *stack) dropFrame() {
	if len(s.frames) > 0 {
		var f frame
		f, s.frames = s.frames[len(s.frames)-1], s.frames[:len(s.frames)-1]
		s.vars = s.vars[:f.varsIndex]
		s.funcs = s.funcs[:f.funcsIndex]
	}
}

func (s *stack) pushVariable(variable stdlib.Variable) {
	s.vars = append(s.vars, variable)
}

func (s *stack) pushFunc(f stdlib.FunctionParams) {
	s.funcs = append(s.funcs, f)
}

func (s *stack) variable(name string) (stdlib.Variable, bool) {
	for _, v := range slices.Backward(s.vars) {
		if name == v.Name {
			return v, true
		}
	}
	return stdlib.Variable{}, false
}

func (s *stack) topMatchName() (string, bool) {
	for _, v := range slices.Backward(s.vars) {
		if strings.HasPrefix(v.Name, "$match") {
			return v.Name, true
		}
	}
	return "", false
}

func (s *stack) function(name string) (stdlib.FunctionParams, bool) {
	for _, v := range slices.Backward(s.funcs) {
		if name == v.ID.Name() {
			return v, true
		}
	}
	return stdlib.FunctionParams{}, false
}
