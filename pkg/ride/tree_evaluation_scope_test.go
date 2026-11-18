package ride //nolint:testpackage // White-box tests of the unexported evaluation scope.

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/ride/ast"
)

func funcPointer(f any) uintptr { return reflect.ValueOf(f).Pointer() }

// TestEvaluationScopeConstants checks that the cached scope tables resolve every constant, function and cost
// of every library version exactly as the selectors do.
func TestEvaluationScopeConstants(t *testing.T) {
	a := newEvaluatorBenchAccounts(t)
	for v := ast.LibV1; v <= ast.CurrentMaxLibraryVersion(); v++ {
		for _, rv := range evaluatorRideV6Variants {
			for _, inv := range []bool{false, true} {
				t.Run(fmt.Sprintf("V%d/%s/invocation=%t", v, rv.name, inv), func(t *testing.T) {
					checkEvaluationScopeTable(t, a, v, rv.rideV6, inv)
				})
			}
		}
	}
}

func TestEvaluationScopeUnsupportedVersion(t *testing.T) {
	a := newEvaluatorBenchAccounts(t)
	env := newEvaluatorBenchEnvironment(t, true, &ast.Tree{LibVersion: ast.LibV6}, a.transfer(t))
	for _, v := range []ast.LibraryVersion{0, ast.CurrentMaxLibraryVersion() + 1} {
		_, err := newEvaluationScope(v, env, true, nil, 0)
		require.Error(t, err)
		assert.Equal(t, EvaluationFailure, GetEvaluationErrorType(err))
		assert.EqualError(t, err, fmt.Sprintf("unsupported library version '%d'", v))
	}
}

func TestEvaluationScopeInvocationParameter(t *testing.T) {
	a := newEvaluatorBenchAccounts(t)
	env := newEvaluatorBenchEnvironment(t, true, &ast.Tree{LibVersion: ast.LibV6}, a.transfer(t))
	calls := 0
	counting := func(environment) rideType {
		calls++
		return rideInt(calls)
	}

	s, err := newEvaluationScope(ast.LibV6, env, true, nil, 0)
	require.NoError(t, err)
	s.setInvocationParameter("i", counting)
	for range 3 { // The value is created once and cached in the scope.
		r, ok := s.constant("i")
		require.True(t, ok)
		assert.Equal(t, rideInt(1), r)
	}

	// The invocation parameter overlays the library constant with the same name.
	s.setInvocationParameter("height", counting)
	r, ok := s.constant("height")
	require.True(t, ok)
	assert.Equal(t, rideInt(2), r)
	_, ok = s.constant("i")
	assert.False(t, ok)

	// Another scope sees neither the parameter nor the cached values of the first one.
	other, err := newEvaluationScope(ast.LibV6, env, true, nil, 0)
	require.NoError(t, err)
	_, ok = other.constant("i")
	assert.False(t, ok)
	r, ok = other.constant("height")
	require.True(t, ok)
	assert.Equal(t, newHeight(env), r)
	assert.Equal(t, 2, calls)
}

func TestEvaluationScopeNoLeakBetweenEvaluations(t *testing.T) {
	a := newEvaluatorBenchAccounts(t)
	tree := &ast.Tree{LibVersion: ast.LibV6}
	env1 := newEvaluatorBenchEnvironment(t, true, tree, a.transfer(t))
	env2 := newEvaluatorBenchEnvironment(t, true, tree, a.transfer(t))
	other := a.transfer(t)
	other.Amount++
	require.NoError(t, env2.SetTransaction(other))

	s1, err := newEvaluationScope(ast.LibV6, env1, true, nil, 0)
	require.NoError(t, err)
	tx1, ok := s1.constant("tx")
	require.True(t, ok)
	s2, err := newEvaluationScope(ast.LibV6, env2, true, nil, 0)
	require.NoError(t, err)
	tx2, ok := s2.constant("tx")
	require.True(t, ok)
	assert.Equal(t, newTx(env1), tx1)
	assert.Equal(t, newTx(env2), tx2)
	assert.NotEqual(t, tx1, tx2)
	// Shared read-only table stays untouched.
	assert.Same(t, s1.consts, s2.consts)
	i := s1.consts.ids["tx"]
	assert.Equal(t, funcPointer(newTx), funcPointer(s1.consts.constructors[i]))
}

func TestEvaluationScopeReservesDeclarations(t *testing.T) {
	a := newEvaluatorBenchAccounts(t)
	env := newEvaluatorBenchEnvironment(t, true, &ast.Tree{LibVersion: ast.LibV6}, a.transfer(t))
	declarations := []ast.Node{
		ast.NewAssignmentNode("a", ast.NewLongNode(1), nil),
		&ast.FunctionDeclarationNode{Name: "f", Body: ast.NewLongNode(2)},
		ast.NewAssignmentNode("b", ast.NewLongNode(3), nil),
	}
	s, err := newEvaluationScope(ast.LibV6, env, true, declarations, 2)
	require.NoError(t, err)
	require.Len(t, s.cs, 1)
	assert.Equal(t, 2, cap(s.cs))
	assert.Equal(t, 4, cap(s.cs[0]))
	assert.Equal(t, 1, cap(s.user))
	for _, d := range declarations {
		require.NoError(t, s.declare(d))
	}
	assert.Len(t, s.cs[0], 2)
	assert.Len(t, s.user, 1)
	_, _, found := s.userFunction("f")
	assert.True(t, found)

	empty, err := newEvaluationScope(ast.LibV6, env, false, nil, 0)
	require.NoError(t, err)
	require.Len(t, empty.cs, 1)
	assert.Nil(t, empty.cs[0])
	assert.Nil(t, empty.user)
}

func checkEvaluationScopeTable(t *testing.T, a evaluatorBenchAccounts, v ast.LibraryVersion, rideV6, inv bool) {
	t.Helper()
	env := newEvaluatorBenchEnvironment(t, rideV6, &ast.Tree{LibVersion: v}, a.transfer(t))
	s, err := newEvaluationScope(v, env, inv, nil, 0)
	require.NoError(t, err)

	names, err := selectConstantNames(v)
	require.NoError(t, err)
	checker, err := selectConstantsChecker(v)
	require.NoError(t, err)
	provider, err := selectConstants(v)
	require.NoError(t, err)
	require.Len(t, s.consts.constructors, len(names))
	for _, n := range names {
		id, ok := checker(n)
		require.True(t, ok)
		i, ok := s.consts.ids[n]
		require.True(t, ok, n)
		assert.Equal(t, funcPointer(provider(int(id))), funcPointer(s.consts.constructors[i]), n)
		if n == "lastBlock" {
			continue // The benchmark state doesn't provide blocks.
		}
		expected := provider(int(id))(env)
		got, ok := s.constant(n)
		require.True(t, ok, n)
		assert.Equal(t, expected, got, n)
		cached, ok := s.constant(n)
		require.True(t, ok, n)
		assert.Equal(t, got, cached, n)
	}
	_, ok := s.constant("unknown-constant")
	assert.False(t, ok)

	checkEvaluationScopeNatives(t, s, v, inv, rideV6)
}

// checkEvaluationScopeNatives checks that the combined table of system functions and costs gives the same
// lookup results as the separate function and cost providers.
func checkEvaluationScopeNatives(t *testing.T, s evaluationScope, v ast.LibraryVersion, inv, rideV6 bool) {
	t.Helper()
	fs, err := selectFunctionsByName(v, inv)
	require.NoError(t, err)
	ev := evaluatorV1
	if rideV6 {
		ev = evaluatorV2
	}
	costs, err := selectEvaluationCostsProvider(v, ev)
	require.NoError(t, err)
	names, err := selectFunctionNameProvider(v)
	require.NoError(t, err)
	all := []string{"unknown-function", "", "1020", "1021"}
	for i := 0; names(i) != ""; i++ {
		all = append(all, names(i))
	}
	for n := range costs {
		all = append(all, n)
	}
	for n := range evaluationScopeTestFunctionsMap(t, v) {
		all = append(all, n)
	}
	known := 0
	for _, n := range all {
		f, hasF := fs(n)
		cost, hasCost := costs[n]
		nf, ok := s.natives[n]
		if !hasF && !hasCost {
			assert.False(t, ok, n)
			continue
		}
		require.True(t, ok, n)
		assert.Equal(t, hasF, nf.hasF, n)
		assert.Equal(t, hasCost, nf.hasCost, n)
		assert.Equal(t, cost, nf.cost, n)
		assert.Equal(t, funcPointer(f), funcPointer(nf.f), n)
		if hasF {
			known++
		}
	}
	assert.Positive(t, known)
}

// evaluationScopeTestFunctionsMap returns the generated map of system functions behind selectFunctionsByName.
func evaluationScopeTestFunctionsMap(t *testing.T, v ast.LibraryVersion) map[string]rideFunction {
	t.Helper()
	switch v {
	case ast.LibV1, ast.LibV2:
		return _functions_map_V2
	case ast.LibV3:
		return _functions_map_V3
	case ast.LibV4:
		return _functions_map_V4
	case ast.LibV5:
		return _functions_map_V5
	case ast.LibV6:
		return _functions_map_V6
	case ast.LibV7:
		return _functions_map_V7
	case ast.LibV8:
		return _functions_map_V8
	case ast.LibV9:
		return _functions_map_V9
	default:
		require.FailNow(t, "no functions map for library version", "%d", v)
		return nil
	}
}
