package compiler

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
	"github.com/wavesplatform/gowaves/pkg/ride/compiler/stdlib"
)

func TestStackOneFrame(t *testing.T) {
	v1 := stdlib.Variable{Name: "V1"}
	v2 := stdlib.Variable{Name: "V2"}

	f1 := stdlib.FunctionParams{ID: ast.UserFunction("F1")}

	s := newStack()
	s.pushVariable(v1)
	s.pushVariable(v2)
	s.pushFunc(f1)

	v, ok := s.variable("V1")
	assert.True(t, ok)
	assert.Equal(t, v1, v)

	v, ok = s.variable("V2")
	assert.True(t, ok)
	assert.Equal(t, v2, v)

	_, ok = s.variable("V3")
	assert.False(t, ok)

	f, ok := s.function("F1")
	assert.True(t, ok)
	assert.Equal(t, f1, f)

	_, ok = s.function("F2")
	assert.False(t, ok)
}

func TestStackFrames(t *testing.T) {
	v1 := stdlib.Variable{Name: "V1"}
	v2 := stdlib.Variable{Name: "V2"}
	v3 := stdlib.Variable{Name: "V3"}
	v4 := stdlib.Variable{Name: "V4"}
	v5 := stdlib.Variable{Name: "V5"}

	f1 := stdlib.FunctionParams{ID: ast.UserFunction("F1")}
	f2 := stdlib.FunctionParams{ID: ast.UserFunction("F2")}
	f3 := stdlib.FunctionParams{ID: ast.UserFunction("F3")}
	f4 := stdlib.FunctionParams{ID: ast.UserFunction("F4")}
	f5 := stdlib.FunctionParams{ID: ast.UserFunction("F5")}

	s := newStack()
	s.pushVariable(v1)
	s.pushVariable(v2)
	s.pushFunc(f1)
	s.pushFunc(f2)
	s.pushFunc(f3)

	s.addFrame()
	s.pushVariable(v3)
	s.pushVariable(v4)
	s.pushVariable(v5)
	s.pushFunc(f4)
	s.pushFunc(f5)

	v, ok := s.variable("V5")
	assert.True(t, ok)
	assert.Equal(t, v5, v)

	v, ok = s.variable("V2")
	assert.True(t, ok)
	assert.Equal(t, v2, v)

	f, ok := s.function("F5")
	assert.True(t, ok)
	assert.Equal(t, f5, f)

	f, ok = s.function("F3")
	assert.True(t, ok)
	assert.Equal(t, f3, f)

	s.dropFrame()

	_, ok = s.variable("V5")
	assert.False(t, ok)
	_, ok = s.variable("V4")
	assert.False(t, ok)
	_, ok = s.variable("V3")
	assert.False(t, ok)

	v, ok = s.variable("V2")
	assert.True(t, ok)
	assert.Equal(t, v2, v)

	_, ok = s.function("F5")
	assert.False(t, ok)
	_, ok = s.function("F4")
	assert.False(t, ok)

	f, ok = s.function("F3")
	assert.True(t, ok)
	assert.Equal(t, f3, f)

	s.dropFrame()

	_, ok = s.variable("V1")
	assert.False(t, ok)

	_, ok = s.function("F1")
	assert.False(t, ok)
}

func TestMatch(t *testing.T) {
	v1 := stdlib.Variable{Name: "V1"}
	m2 := stdlib.Variable{Name: "$match123"}
	v3 := stdlib.Variable{Name: "V3"}
	m4 := stdlib.Variable{Name: "$match456"}

	s := newStack()
	s.pushVariable(v1)
	s.pushVariable(m2)

	s.addFrame()
	s.pushVariable(v3)
	s.pushVariable(m4)

	m, ok := s.topMatchName()
	assert.True(t, ok)
	assert.Equal(t, m4.Name, m)

	s.dropFrame()

	m, ok = s.topMatchName()
	assert.True(t, ok)
	assert.Equal(t, m2.Name, m)

	s.dropFrame()

	_, ok = s.topMatchName()
	assert.False(t, ok)
}
