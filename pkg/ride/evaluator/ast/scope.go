package ast

import "github.com/wavesplatform/gowaves/pkg/state"

type Scope interface {
	Clone() Scope
	AddValue(name string, expr Expr)
	FuncByShort(int16) (Callable, bool)
	FuncByName(string) (Callable, bool)
	Value(string) (Expr, bool)
	State() state.State
	Scheme() byte
}

type ScopeImpl struct {
	parent    Scope
	funcs     *FuncScope
	variables map[string]Expr
	state     state.State
	scheme    byte
}

type Callable func(Scope, Exprs) (Expr, error)

func NewScope(scheme byte, state state.State, f *FuncScope, variables map[string]Expr) *ScopeImpl {
	return &ScopeImpl{
		funcs:     f,
		variables: variables,
		state:     state,
		scheme:    scheme,
	}
}

func (a *ScopeImpl) Clone() Scope {
	return &ScopeImpl{
		funcs:  a.funcs.Clone(),
		parent: a,
		state:  a.state,
	}
}

func (a *ScopeImpl) State() state.State {
	return a.state
}

func (a *ScopeImpl) FuncByShort(id int16) (Callable, bool) {
	return a.funcs.GetByShort(id)
}

func (a *ScopeImpl) FuncByName(name string) (Callable, bool) {
	return a.funcs.GetByName(name)
}

func (a *ScopeImpl) AddValue(name string, value Expr) {
	if a.variables == nil {
		a.variables = make(map[string]Expr)
	}
	a.variables[name] = value
}

func (a *ScopeImpl) Value(name string) (Expr, bool) {
	// first look in current scope
	if a.variables != nil {
		if v, ok := a.variables[name]; ok {
			return v, true
		}
	}

	// try find in parent
	if a.parent != nil {
		return a.parent.Value(name)
	} else {
		return nil, false
	}
}

func (a *ScopeImpl) Scheme() byte {
	return a.scheme
}

type FuncScope struct {
	funcs     map[int16]Callable
	userFuncs map[string]Callable
}

func EmptyFuncScope() *FuncScope {
	return &FuncScope{
		funcs:     make(map[int16]Callable),
		userFuncs: make(map[string]Callable),
	}
}

func NewFuncScope() *FuncScope {

	funcs := make(map[int16]Callable)

	funcs[0] = NativeEq
	funcs[1] = NativeIsinstanceof
	funcs[2] = NativeThrow

	funcs[100] = NativeSumLong
	funcs[101] = NativeSubLong
	funcs[102] = NativeGtLong
	funcs[103] = NativeGeLong
	funcs[104] = NativeMulLong
	funcs[105] = NativeDivLong
	funcs[106] = NativeModLong
	funcs[107] = NativeFractionLong

	funcs[200] = NativeSizeBytes
	funcs[201] = NativeTakeBytes
	funcs[202] = NativeDropBytes
	funcs[203] = NativeConcatBytes

	funcs[300] = NativeConcatStrings
	funcs[303] = NativeTakeStrings
	funcs[304] = NativeDropStrings
	funcs[305] = NativeSizeString

	funcs[400] = NativeSizeList
	funcs[401] = NativeGetList
	funcs[410] = NativeLongToBytes
	funcs[411] = NativeStringToBytes
	funcs[412] = NativeBooleanToBytes
	funcs[420] = NativeLongToString
	funcs[421] = NativeBooleanToString

	funcs[500] = NativeSigVerify

	funcs[1001] = NativeTransactionHeightByID

	userFuncs := make(map[string]Callable)
	userFuncs["throw"] = UserThrow
	userFuncs["addressFromString"] = UserAddressFromString

	return &FuncScope{
		funcs:     funcs,
		userFuncs: userFuncs,
	}
}

func (a *FuncScope) GetByShort(id int16) (Callable, bool) {
	f, ok := a.funcs[id]
	return f, ok
}

func (a *FuncScope) GetByName(name string) (Callable, bool) {
	f, ok := a.userFuncs[name]
	return f, ok
}

func (a *FuncScope) Clone() *FuncScope {
	return a
}
