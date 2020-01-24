package runner

// Runner run asynchronous or synchronous
type Runner interface {
	Go(func())
}
