package runner

// asynchronous for async funcs
type asynchronous struct {
}

// Go run func asynchronous
func (a asynchronous) Go(f func()) {
	go f()
}

// NewAsync create new asynchronous
func NewAsync() asynchronous {
	return asynchronous{}
}
