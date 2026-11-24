package ride //nolint:testpackage // White-box tests of the unexported estimator internals.

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/ride/ast"
)

func TestFsV3Shadowing(t *testing.T) {
	root := newFsV3()
	assert.Nil(t, root.functions)
	_, ok := root.get("f")
	assert.False(t, ok)

	assert.False(t, root.set("f", 1, []string{"a"}, false))
	child := root.spawn()
	assert.Nil(t, child.functions, "spawn must not allocate the map")
	fd, ok := child.get("f")
	require.True(t, ok)
	assert.Equal(t, functionDescriptor{cost: 1, usages: []string{"a"}}, fd)

	// Declaration in the child shadows the parent one, but it's not an overwrite in the child scope.
	assert.False(t, child.set("f", 2, nil, true))
	fd, ok = child.get("f")
	require.True(t, ok)
	assert.Equal(t, functionDescriptor{cost: 2, callsInvocation: true}, fd)
	assert.True(t, child.set("f", 3, nil, false))
	fd, _ = child.get("f")
	assert.Equal(t, 3, fd.cost)

	// The parent is untouched.
	fd, _ = root.get("f")
	assert.Equal(t, 1, fd.cost)

	// A declaration made in the parent after the spawn is visible from the child (lookup goes through the chain).
	grandchild := child.spawn()
	root.set("g", 5, nil, false)
	fd, ok = grandchild.get("g")
	require.True(t, ok)
	assert.Equal(t, 5, fd.cost)
}

func TestEstimationScopeV3SaveRestore(t *testing.T) {
	s := newEstimationScopeV3(CatalogueV6)
	s.setFunction("f", 1, nil, false)

	outer := s.save()
	assert.False(t, s.setFunction("f", 2, nil, false), "shadowing in a nested scope is not an overwrite")
	inner := s.save()
	s.setFunction("g", 3, nil, false)
	fd, err := s.function(ast.UserFunction("g"))
	require.NoError(t, err)
	assert.Equal(t, 3, fd.cost)

	// Restore after the nested function declarations: g is gone, f is the shadowing one.
	s.restore(inner)
	_, err = s.function(ast.UserFunction("g"))
	assert.EqualError(t, err, "user function 'g' not found")
	fd, err = s.function(ast.UserFunction("f"))
	require.NoError(t, err)
	assert.Equal(t, 2, fd.cost)

	s.restore(outer)
	fd, err = s.function(ast.UserFunction("f"))
	require.NoError(t, err)
	assert.Equal(t, 1, fd.cost)
	assert.True(t, s.setFunction("f", 4, nil, false), "redeclaration in the same scope is an overwrite")

	// Native fallback and invocation flag.
	fd, err = s.function(ast.NativeFunction("1020"))
	require.NoError(t, err)
	assert.True(t, fd.callsInvocation)
	_, err = s.function(ast.NativeFunction("unknown"))
	assert.EqualError(t, err, "native function 'unknown' not found")

	// Reset drops everything and reuses the root.
	s.save()
	s.setFunction("h", 1, nil, false)
	s.resetFunctions()
	assert.Same(t, s.root, s.functions)
	_, err = s.function(ast.UserFunction("f"))
	assert.Error(t, err)
	_, err = s.function(ast.UserFunction("h"))
	assert.Error(t, err)
	assert.False(t, s.setFunction("f", 1, nil, false))
}

func TestEstimationScopeV3Usages(t *testing.T) {
	s := newEstimationScopeV3(CatalogueV6)
	// No usages level: all operations are no-ops.
	s.use("a")
	assert.False(t, s.used("a"))
	assert.False(t, s.remove("a"))
	assert.Nil(t, s.emerge())
	s.drop()

	s.submerge()
	for _, id := range []string{"a", "b", "a", "c", "a", "d"} {
		s.use(id)
	}
	assert.True(t, s.used("a"))
	assert.False(t, s.used("x"))
	assert.False(t, s.remove("x"))
	assert.True(t, s.remove("a"))
	assert.False(t, s.used("a"))
	assert.False(t, s.remove("a"))

	// Nested level doesn't see the outer usages, and the usages are propagated in order with duplicates.
	s.submerge()
	assert.False(t, s.used("b"))
	s.use("x")
	s.use("y")
	s.use("x")
	body := s.emerge()
	assert.Equal(t, []string{"x", "y", "x"}, body)

	// The returned usages must not be affected by the reuse of the backing array.
	s.submerge()
	s.use("z")
	s.use("z")
	s.use("z")
	assert.Equal(t, []string{"z", "z", "z"}, s.emerge())
	assert.Equal(t, []string{"x", "y", "x"}, body)

	for _, u := range body {
		s.use(u)
	}
	assert.Equal(t, []string{"b", "c", "d", "x", "y", "x"}, s.emerge())

	// Empty level.
	s.submerge()
	assert.Empty(t, s.emerge())
	s.submerge()
	s.use("a")
	s.drop()
	assert.Empty(t, s.usages)
	s.submerge()
	assert.False(t, s.used("a"), "a reused level starts empty")
}

func TestEstimationScopeV3UsagesDeepNesting(t *testing.T) {
	s := newEstimationScopeV3(CatalogueV6)
	const depth = 10 // Deeper than the initial capacity of the usages stack.
	// The second round reuses the levels kept after the first one.
	for range 2 {
		for i := range depth {
			s.submerge()
			assert.False(t, s.used(fmt.Sprintf("v%d", i)), "a new level starts empty")
			assert.False(t, s.used(fmt.Sprintf("v%d", i-1)), "a nested level doesn't see the outer usages")
			s.use(fmt.Sprintf("v%d", i))
		}
		inner := s.emerge()
		assert.Equal(t, []string{fmt.Sprintf("v%d", depth-1)}, inner)
		for i := depth - 2; i >= 0; i-- {
			u := s.emerge()
			assert.Equal(t, []string{fmt.Sprintf("v%d", i)}, u)
		}
		assert.Empty(t, s.usages)
		assert.Equal(t, []string{fmt.Sprintf("v%d", depth-1)}, inner, "emerged usages are not affected by reuse")
	}
}
