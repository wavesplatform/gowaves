package execution

import (
	"sync"
	"sync/atomic"
)

// A TaskGroup manages a collection of cooperating goroutines. Add new tasks to the group with the Run method.
// Call the Wait method to wait for the tasks to complete.
// A zero value is ready for use, but must not be copied after its first use.
//
// The group collects any errors returned by the tasks in the group.
// The first non-nil error reported by any execution and not filtered is returned from the Wait method.
type TaskGroup struct {
	wg sync.WaitGroup // Counter for active goroutines.

	// active is true when the group is "active", meaning there has been at least one call to Run since the group
	// was created or the last Wait.
	//
	// Together active and errLock work as a kind of resettable sync.Once. The fast path reads active and only
	// acquires errLock if it discovers setup is needed.
	active atomic.Bool

	errLock sync.Mutex // Guards the fields below.
	err     error      // First captured error returned from Wait.
	onError errorFunc  // Called each time a task returns non-nil error.
}

// NewTaskGroup constructs a new empty group with the specified error handler.
// See [TaskGroup.OnError] for a description of how errors are filtered. If handler is nil, no filtering is performed.
// Main properties of the TaskGroup are:
// - Cancel propagation.
// - Error propagation.
// - Waiting for all tasks to finish.
func NewTaskGroup(handler func(error) error) *TaskGroup {
	return new(TaskGroup).OnError(handler)
}

// OnError sets the error handler for TaskGroup. If handler is nil,
// the error handler is removed and errors are no longer filtered. Otherwise, each non-nil error reported by an
// execution running in g is passed to handler.
//
// Then handler is called with each reported error, and its result replaces the reported value. This permits handler to
// suppress or replace the error value selectively.
//
// Calls to handler are synchronized so that it is safe for handler to manipulate local data structures without
// additional locking. It is safe to call OnError while tasks are active in TaskGroup.
func (g *TaskGroup) OnError(handler func(error) error) *TaskGroup {
	g.errLock.Lock()
	defer g.errLock.Unlock()
	g.onError = handler
	return g
}

// Run starts an [execute] function in a new goroutine in [TaskGroup]. The execution is not interrupted by TaskGroup,
// so the [execute] function should include the interruption logic.
func (g *TaskGroup) Run(execute func() error) {
	g.wg.Add(1)
	if g.active.CompareAndSwap(false, true) {
		g.errLock.Lock()
		g.err = nil
		g.errLock.Unlock()
	}
	go func() {
		defer g.wg.Done()
		if err := execute(); err != nil {
			g.handleError(err)
		}
	}()
}

// Wait blocks until all the goroutines currently active in the TaskGroup have returned, and all reported errors have
// been delivered to the handler. It returns the first non-nil error reported by any of the goroutines in the group and
// not filtered by an OnError handler.
//
// As with sync.WaitGroup, new tasks can be added to TaskGroup during a Wait call only if the TaskGroup contains at
// least one active execution when Wait is called and continuously thereafter until the last concurrent call to
// Run returns.
//
// Wait may be called from at most one goroutine at a time. After Wait has returned, the group is ready for reuse.
func (g *TaskGroup) Wait() error {
	g.wg.Wait()
	g.errLock.Lock()
	defer g.errLock.Unlock()

	// If the group is still active, deactivate it now.
	g.active.CompareAndSwap(true, false)
	return g.err
}

// handleError synchronizes access to the error handler and captures the first non-nil error.
func (g *TaskGroup) handleError(err error) {
	g.errLock.Lock()
	defer g.errLock.Unlock()
	e := g.onError.filter(err)
	if e != nil && g.err == nil {
		g.err = e // Capture the first unfiltered error.
	}
}

// An errorFunc is called by a group each time an execution reports an error. Its return value replaces the reported
// error, so the errorFunc can filter or suppress errors by modifying or discarding the input error.
type errorFunc func(error) error

func (ef errorFunc) filter(err error) error {
	if ef == nil {
		return err
	}
	return ef(err)
}
