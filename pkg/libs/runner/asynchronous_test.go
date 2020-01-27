package runner

import (
	"testing"
)

// if test incorrect, it will hang forever
func TestAsynchronous_Go(t *testing.T) {
	s := NewAsync()
	ch := make(chan int)
	s.Go(func() {
		ch <- 1
	})
	<-ch
}
