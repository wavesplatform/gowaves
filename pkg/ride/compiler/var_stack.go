package compiler

import s "github.com/wavesplatform/gowaves/pkg/ride/compiler/stdlib"

type VarStack struct {
	up *VarStack

	vars  []s.Variable
	funcs []s.FunctionParams
}

func NewVarStack(upperStack *VarStack) *VarStack {
	return &VarStack{
		up:   upperStack,
		vars: make([]s.Variable, 0),
	}
}

func (st *VarStack) PushVariable(variable s.Variable) {
	st.vars = append(st.vars, variable)
}

func (st *VarStack) PushFunc(f s.FunctionParams) {
	st.funcs = append(st.funcs, f)
}

func (st *VarStack) GetVariable(name string) (s.Variable, bool) {
	for i := len(st.vars) - 1; i >= 0; i-- {
		if name == st.vars[i].Name {
			return st.vars[i], true
		}
	}
	if st.up == nil {
		return s.Variable{}, false
	}
	return st.up.GetVariable(name)
}

func (st *VarStack) GetFunc(name string) (s.FunctionParams, bool) {
	for i := len(st.funcs) - 1; i >= 0; i-- {
		if name == st.funcs[i].ID.Name() {
			return st.funcs[i], true
		}
	}
	if st.up == nil {
		return s.FunctionParams{}, false
	}
	return st.up.GetFunc(name)
}
