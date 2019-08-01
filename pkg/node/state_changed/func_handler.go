package state_changed

type FuncHandler struct {
	f func()
}

func NewFuncHandler(f func()) FuncHandler {
	return FuncHandler{f: f}
}

func (a FuncHandler) Handle() {
	a.f()
}
