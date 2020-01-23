package runner

// synchronous for sync funcs
type synchronous struct {
}

// // Go run func synchronous
func (a synchronous) Go(f func()) {
	f()
}

// NewAsync create new synchronous
func NewSync() synchronous {
	return synchronous{}
}
