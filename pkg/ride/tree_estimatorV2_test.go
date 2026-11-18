package ride //nolint:testpackage // White-box tests of the unexported estimator internals.

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/ride/ast"
	ridec "github.com/wavesplatform/gowaves/pkg/ride/compiler"
)

func TestEstimationScopeV2ValueShadowing(t *testing.T) {
	s := newEstimationScopeV2(ConstantsV3, CatalogueV3)

	// Constants are found and have no expression.
	n, err := s.value("height")
	require.NoError(t, err)
	assert.Nil(t, n)
	_, ok := s.lookup("x")
	assert.False(t, ok)
	_, err = s.value("x")
	assert.EqualError(t, err, "value 'x' not found")

	one := ast.NewLongNode(1)
	two := ast.NewLongNode(2)
	s.setValue("x", one)
	s.setValue("height", two) // Shadows the constant.
	n, err = s.value("height")
	require.NoError(t, err)
	assert.Same(t, two, n)

	// The last declaration wins.
	p, _, _ := s.save()
	s.setValue("x", two)
	n, ok = s.lookup("x")
	require.True(t, ok)
	assert.Same(t, two, n)
	s.setValue("x", nil)
	n, ok = s.lookup("x")
	require.True(t, ok)
	assert.Nil(t, n)

	// Restore brings back the outer declaration.
	s.restore(p, 0, 0)
	n, ok = s.lookup("x")
	require.True(t, ok)
	assert.Same(t, one, n)
	n, err = s.value("height")
	require.NoError(t, err)
	assert.Same(t, two, n)
}

func TestEstimationScopeV2DeleteValue(t *testing.T) {
	s := newEstimationScopeV2(ConstantsV3, CatalogueV3)
	one := ast.NewLongNode(1)
	s.setValue("a", one)
	l := len(s.values)
	s.setValue("a", nil)
	s.setValue("b", nil)
	s.setValue("a", nil)

	// Only the declarations at or above the limit are deleted, the order of the rest is kept.
	s.deleteValue("a", l)
	assert.Equal(t, []scopeValue{{id: "a", n: one}, {id: "b"}}, s.values)
	n, ok := s.lookup("a")
	require.True(t, ok)
	assert.Same(t, one, n)
}

func TestEstimationScopeV2Functions(t *testing.T) {
	s := newEstimationScopeV2(ConstantsV3, CatalogueV3)
	_, err := s.function("f")
	assert.EqualError(t, err, "function 'f' not found")

	f1 := &ast.FunctionDeclarationNode{Name: "f"}
	f2 := &ast.FunctionDeclarationNode{Name: "f"}
	g := &ast.FunctionDeclarationNode{Name: "g"}
	s.setFunction(f1)
	_, pf, _ := s.save()
	s.setFunction(f2)
	s.setFunction(g)
	f, err := s.function("f")
	require.NoError(t, err)
	assert.Same(t, f2, f)

	s.restore(len(s.values), pf, len(s.stash))
	f, err = s.function("f")
	require.NoError(t, err)
	assert.Same(t, f1, f)
	_, err = s.function("g")
	assert.Error(t, err)
}

func TestEstimationScopeV2Overlaps(t *testing.T) {
	s := newEstimationScopeV2(ConstantsV3, CatalogueV3)
	n1 := ast.NewLongNode(1)
	n2 := ast.NewLongNode(2)
	n3 := ast.NewLongNode(3)

	// Nothing to re-stack.
	s.restackOverlaps()
	assert.Empty(t, s.values)

	// A single overlap.
	s.setOverlap("a", n1)
	s.restackOverlaps()
	assert.Equal(t, []scopeValue{{id: "a", n: n1}}, s.values)

	// The last stashed value of every name is re-stacked once.
	pv, pf, po := s.save()
	s.addOverlaps([]scopeValue{{id: "b", n: n2}, {id: "a", n: n3}})
	s.setOverlap("b", nil)
	s.restackOverlaps()
	assert.Len(t, s.values, pv+2)
	n, ok := s.lookup("a")
	require.True(t, ok)
	assert.Same(t, n3, n)
	n, ok = s.lookup("b")
	require.True(t, ok)
	assert.Nil(t, n)

	// The reused set doesn't keep the names of the previous call.
	s.restore(pv, pf, po)
	s.setOverlap("b", n2)
	s.restackOverlaps()
	assert.Len(t, s.values, pv+2)
	n, ok = s.lookup("b")
	require.True(t, ok)
	assert.Same(t, n2, n)

	// Restore drops the stashed overlaps too.
	s.restore(pv, pf, po)
	assert.Equal(t, []scopeValue{{id: "a", n: n1}}, s.stash)
}

func TestEstimatorV2FunctionArgumentOverlaps(t *testing.T) {
	// Expected values are taken from the estimator before the scope optimization.
	for _, test := range []struct {
		comment     string
		source      string
		estimation  int
		verifier    int
		estimations map[string]int
	}{
		{"argument overlaps a global value", `
{-# STDLIB_VERSION 3 #-}
{-# CONTENT_TYPE EXPRESSION #-}
let x = 1 + 1
func f(x: Int) = x + 1
func g(y: Int, x: Int) = f(y) + x + f(x)
g(x, x) > f(x) + x
`, 78, 78, nil},
		{"nested calls with several overlaps", `
{-# STDLIB_VERSION 3 #-}
{-# CONTENT_TYPE EXPRESSION #-}
let a = 1 + 1
let b = a + 2
func f(a: Int, z: Int) = { let c = a + z + b; c + c }
func g(b: Int) = f(b, b) + f(a, b) + b
g(a) + f(b, a) + a + b > 0
`, 166, 166, nil},
		{"argument overlaps a constant", `
{-# STDLIB_VERSION 3 #-}
{-# CONTENT_TYPE EXPRESSION #-}
func f(height: Int) = height + 1
f(height) > height
`, 20, 20, nil},
		{"overlaps in callables and verifier", `
{-# STDLIB_VERSION 5 #-}
{-# CONTENT_TYPE DAPP #-}
{-# SCRIPT_TYPE ACCOUNT #-}
let k = 10
func h(k: Int, m: Int) = if (k > m) then k else m + k
func p(m: Int) = h(m, k) + h(k, m)
@Callable(i)
func c1(m: Int) = [IntegerEntry("a", p(m) + h(m, m))]
@Callable(i)
func c2(k: Int) = [IntegerEntry("b", p(k) + k)]
@Verifier(tx)
func v() = h(k, k) > 0
`, 130, 55, map[string]int{"c1": 130, "c2": 105}},
	} {
		t.Run(test.comment, func(t *testing.T) {
			tree, errs := ridec.CompileToTree(test.source)
			require.Empty(t, errs)
			e, err := newTreeEstimatorV2(tree)
			require.NoError(t, err)
			est, vc, fs, err := e.estimate()
			require.NoError(t, err)
			assert.Equal(t, test.estimation, est)
			assert.Equal(t, test.verifier, vc)
			if test.estimations == nil {
				assert.Nil(t, fs)
			} else {
				assert.Equal(t, test.estimations, fs)
			}
			assert.Empty(t, e.scope.args, "overlapped arguments must be released")
		})
	}
}
