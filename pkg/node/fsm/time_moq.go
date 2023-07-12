// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package fsm

import (
	"sync"
	"time"

	"github.com/wavesplatform/gowaves/pkg/types"
)

// Ensure, that MockTime does implement types.Time.
// If this is not the case, regenerate this file with moq.
var _ types.Time = &MockTime{}

// MockTime is a mock implementation of types.Time.
//
//	func TestSomethingThatUsesTime(t *testing.T) {
//
//		// make and configure a mocked types.Time
//		mockedTime := &MockTime{
//			NowFunc: func() time.Time {
//				panic("mock out the Now method")
//			},
//		}
//
//		// use mockedTime in code that requires types.Time
//		// and then make assertions.
//
//	}
type MockTime struct {
	// NowFunc mocks the Now method.
	NowFunc func() time.Time

	// calls tracks calls to the methods.
	calls struct {
		// Now holds details about calls to the Now method.
		Now []struct {
		}
	}
	lockNow sync.RWMutex
}

// Now calls NowFunc.
func (mock *MockTime) Now() time.Time {
	if mock.NowFunc == nil {
		panic("MockTime.NowFunc: method is nil but Time.Now was just called")
	}
	callInfo := struct {
	}{}
	mock.lockNow.Lock()
	mock.calls.Now = append(mock.calls.Now, callInfo)
	mock.lockNow.Unlock()
	return mock.NowFunc()
}

// NowCalls gets all the calls that were made to Now.
// Check the length with:
//
//	len(mockedTime.NowCalls())
func (mock *MockTime) NowCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockNow.RLock()
	calls = mock.calls.Now
	mock.lockNow.RUnlock()
	return calls
}
