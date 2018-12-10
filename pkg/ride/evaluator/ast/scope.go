package ast

type Scope interface {
	Clone() Scope
	AddValue(name string, expr Expr)
	FuncByShort(int16) (Callable, bool)
	FuncByName(string) (Callable, bool)
	Value(string) (Expr, bool)
}

type ScopeImpl struct {
	parent    Scope
	funcs     *FuncScope
	variables map[string]Expr
}

type Callable func(Scope, Exprs) (Expr, error)

func NewScope(f *FuncScope, variables map[string]Expr) *ScopeImpl {
	return &ScopeImpl{
		funcs:     f,
		variables: variables,
	}
}

func (a *ScopeImpl) Clone() Scope {
	return &ScopeImpl{
		funcs:  a.funcs.Clone(),
		parent: a,
	}
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

	funcs[100] = NativeSumLong
	funcs[101] = NativeSubLong
	funcs[102] = NativeGtLong
	funcs[103] = NativeGeLong

	funcs[401] = NativeGetList

	// TODO
	funcs[500] = Native500

	userFuncs := make(map[string]Callable)
	userFuncs["throw"] = USER_THROW
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
