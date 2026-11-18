package ride //nolint:testpackage // White-box tests of the unexported estimator internals.

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/ride/ast"
	ridec "github.com/wavesplatform/gowaves/pkg/ride/compiler"
)

func TestEstimationScopeV1ValueShadowing(t *testing.T) {
	s := newEstimationScopeV1(ConstantsV3, CatalogueV3)

	// Constants are found and have no expression.
	n, err := s.value("height")
	require.NoError(t, err)
	assert.Nil(t, n)
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
	p := s.save()
	s.setValue("x", two)
	n, err = s.value("x")
	require.NoError(t, err)
	assert.Same(t, two, n)
	s.setValue("x", nil)
	n, err = s.value("x")
	require.NoError(t, err)
	assert.Nil(t, n)

	// Restore brings back the outer declaration.
	s.restore(p)
	assert.Equal(t, p, s.save())
	n, err = s.value("x")
	require.NoError(t, err)
	assert.Same(t, one, n)
	n, err = s.value("height")
	require.NoError(t, err)
	assert.Same(t, two, n)

	// Restore to the current or a higher position changes nothing.
	s.restore(p)
	s.restore(p + 1)
	assert.Equal(t, p, s.save())
}

func TestEstimationScopeV1Branches(t *testing.T) {
	s := newEstimationScopeV1(ConstantsV3, CatalogueV3)
	n1 := ast.NewLongNode(1)
	n2 := ast.NewLongNode(2)
	n3 := ast.NewLongNode(3)
	s.setValue("a", n1)

	// The values of the true branch are moved aside while the false branch is estimated.
	p := s.save()
	s.setValue("a", nil)
	s.setValue("b", n2)
	b := s.stashBranch(p)
	assert.Equal(t, []scopeValue{{id: "a", n: n1}}, s.values)
	n, err := s.value("a")
	require.NoError(t, err)
	assert.Same(t, n1, n)

	// A nested conditional in the false branch whose false branch wins.
	s.setValue("c", n3)
	q := s.save()
	s.setValue("a", n3)
	nb := s.stashBranch(q)
	s.dropBranch(nb)
	assert.Equal(t, []scopeValue{{id: "a", n: n1}, {id: "c", n: n3}}, s.values)

	// A nested conditional in the false branch whose true branch wins.
	q = s.save()
	s.setValue("d", n1)
	nb = s.stashBranch(q)
	s.setValue("e", n2)
	s.unstashBranch(q, nb)
	assert.Equal(t, []scopeValue{{id: "a", n: n1}, {id: "c", n: n3}, {id: "d", n: n1}}, s.values)

	// The outer true branch wins: its values replace the values of the false branch, in the same order.
	s.unstashBranch(p, b)
	assert.Equal(t, []scopeValue{{id: "a", n: n1}, {id: "a"}, {id: "b", n: n2}}, s.values)
	assert.Empty(t, s.branches)

	// An empty true branch restores the scope to the saved position.
	p = s.save()
	b = s.stashBranch(p)
	s.setValue("f", n1)
	s.unstashBranch(p, b)
	assert.Equal(t, p, s.save())
	assert.Empty(t, s.branches)
}

func TestEstimationScopeV1Functions(t *testing.T) {
	s := newEstimationScopeV1(ConstantsV3, CatalogueV3)
	_, err := s.function(ast.UserFunction("f"))
	assert.EqualError(t, err, "function 'f' not found")
	_, err = s.function(ast.NativeFunction("f"))
	assert.EqualError(t, err, "system function 'f' not found")

	// User functions fall back to the built-in functions.
	c, err := s.function(ast.UserFunction("getInteger"))
	require.NoError(t, err)
	assert.Equal(t, CatalogueV3["getInteger"], c)
	c, err = s.function(ast.NativeFunction("0"))
	require.NoError(t, err)
	assert.Equal(t, CatalogueV3["0"], c)

	// A user function shadows the built-in one, the last declaration wins and restore doesn't drop functions.
	p := s.save()
	s.setFunction("getInteger", 7)
	s.setFunction("f", 3)
	s.setFunction("f", 4)
	s.restore(p)
	c, err = s.function(ast.UserFunction("getInteger"))
	require.NoError(t, err)
	assert.Equal(t, 7, c)
	c, err = s.function(ast.UserFunction("f"))
	require.NoError(t, err)
	assert.Equal(t, 4, c)
}

func TestEstimatorV1Scripts(t *testing.T) {
	// Expected values are taken from the estimator before the scope optimization.
	for _, test := range []struct {
		comment     string
		source      string
		estimation  int
		verifier    int
		estimations map[string]int
	}{
		{"true branch wins and keeps its inlined references", `
{-# STDLIB_VERSION 3 #-}
{-# CONTENT_TYPE EXPRESSION #-}
let a = 1 + 1
let b = a + a
let c = if (a > 0) then b + a + b else 0
c > a + b
`, 44, 44, nil},
		{"false branch wins", `
{-# STDLIB_VERSION 3 #-}
{-# CONTENT_TYPE EXPRESSION #-}
let a = 1 + 1
let b = a + a
let c = if (height > 0) then 0 else b + a + b
c > a + b
`, 44, 44, nil},
		{"nested conditionals in the false branch", `
{-# STDLIB_VERSION 3 #-}
{-# CONTENT_TYPE EXPRESSION #-}
let a = 1 + 1
let b = a + a
let d = a * b
let c = if (height > 0) then b + d else if (a > b) then d + d + a else if (b > 0) then 0 else a
c + a + b + d > 0
`, 65, 65, nil},
		{"shadowed declarations in functions", `
{-# STDLIB_VERSION 3 #-}
{-# CONTENT_TYPE EXPRESSION #-}
let x = 1 + 1
func f(x: Int) = { let y = x + 1; if (y > 0) then y + x else x }
func g(y: Int) = { let w = y + 2; f(w) + w + x }
g(x) + f(x) + x > 0
`, 104, 104, nil},
		{"callables and verifier", `
{-# STDLIB_VERSION 3 #-}
{-# CONTENT_TYPE DAPP #-}
{-# SCRIPT_TYPE ACCOUNT #-}
let k = 10 + 1
func h(k: Int, m: Int) = if (k > m) then k else m + k
@Callable(i)
func c1(m: Int) = WriteSet([DataEntry("a", h(m, k) + k)])
@Callable(i)
func c2(k: Int) = WriteSet([DataEntry("b", if (k > 0) then h(k, k) else k)])
@Verifier(tx)
func v() = h(k, k) > k
`, 68, 53, map[string]int{"c1": 68, "c2": 67}},
	} {
		t.Run(test.comment, func(t *testing.T) {
			tree, errs := ridec.CompileToTree(test.source)
			require.Empty(t, errs)
			e, err := newTreeEstimatorV1(tree)
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
			assert.Empty(t, e.scope.branches, "stashed branches must be released")
		})
	}
}
