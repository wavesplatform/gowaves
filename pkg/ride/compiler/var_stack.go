package compiler

type VarStack struct {
	up *VarStack

	vars []Variable
}

func NewVarStack(upperStack *VarStack) *VarStack {
	return &VarStack{
		up:   upperStack,
		vars: make([]Variable, 0),
	}
}

func (s *VarStack) Push(variable Variable) {
	s.vars = append(s.vars, variable)
}

func (s *VarStack) GetVariable(name string) (Variable, bool) {
	for i := len(s.vars) - 1; i >= 0; i-- {
		if name == s.vars[i].Name {
			return s.vars[i], true
		}
	}
	if s.up == nil {
		return Variable{}, false
	}
	return s.up.GetVariable(name)
}

type Variable struct {
	Name string
	Type string
}

type Func struct {
	Name    string
	RetType string
}
