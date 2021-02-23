// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package ride

import (
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/types"
	"sync"
)

// Ensure, that MockRideEnvironment does implement RideEnvironment.
// If this is not the case, regenerate this file with moq.
var _ RideEnvironment = &MockRideEnvironment{}

// MockRideEnvironment is a mock implementation of RideEnvironment.
//
//     func TestSomethingThatUsesRideEnvironment(t *testing.T) {
//
//         // make and configure a mocked RideEnvironment
//         mockedRideEnvironment := &MockRideEnvironment{
//             ChooseSizeCheckFunc: func(v int)  {
// 	               panic("mock out the ChooseSizeCheck method")
//             },
//             SetInvocationFunc: func(inv rideObject)  {
// 	               panic("mock out the SetInvocation method")
//             },
//             blockFunc: func() rideObject {
// 	               panic("mock out the block method")
//             },
//             checkMessageLengthFunc: func(in1 int) bool {
// 	               panic("mock out the checkMessageLength method")
//             },
//             heightFunc: func() rideInt {
// 	               panic("mock out the height method")
//             },
//             invocationFunc: func() rideObject {
// 	               panic("mock out the invocation method")
//             },
//             schemeFunc: func() byte {
// 	               panic("mock out the scheme method")
//             },
//             setNewDAppAddressFunc: func(address proto.Address)  {
// 	               panic("mock out the setNewDAppAddress method")
//             },
//             stateFunc: func() types.SmartState {
// 	               panic("mock out the state method")
//             },
//             thisFunc: func() rideType {
// 	               panic("mock out the this method")
//             },
//             timestampFunc: func() uint64 {
// 	               panic("mock out the timestamp method")
//             },
//             transactionFunc: func() rideObject {
// 	               panic("mock out the transaction method")
//             },
//             txIDFunc: func() rideType {
// 	               panic("mock out the txID method")
//             },
//         }
//
//         // use mockedRideEnvironment in code that requires RideEnvironment
//         // and then make assertions.
//
//     }
type MockRideEnvironment struct {
	// ChooseSizeCheckFunc mocks the ChooseSizeCheck method.
	ChooseSizeCheckFunc func(v int)

	// SetInvocationFunc mocks the SetInvocation method.
	SetInvocationFunc func(inv rideObject)

	// blockFunc mocks the block method.
	blockFunc func() rideObject

	// checkMessageLengthFunc mocks the checkMessageLength method.
	checkMessageLengthFunc func(in1 int) bool

	// heightFunc mocks the height method.
	heightFunc func() rideInt

	// invocationFunc mocks the invocation method.
	invocationFunc func() rideObject

	// schemeFunc mocks the scheme method.
	schemeFunc func() byte

	// setNewDAppAddressFunc mocks the setNewDAppAddress method.
	setNewDAppAddressFunc func(address proto.Address)

	// stateFunc mocks the state method.
	stateFunc func() types.SmartState

	// thisFunc mocks the this method.
	thisFunc func() rideType

	// timestampFunc mocks the timestamp method.
	timestampFunc func() uint64

	// transactionFunc mocks the transaction method.
	transactionFunc func() rideObject

	// txIDFunc mocks the txID method.
	txIDFunc func() rideType

	// calls tracks calls to the methods.
	calls struct {
		// ChooseSizeCheck holds details about calls to the ChooseSizeCheck method.
		ChooseSizeCheck []struct {
			// V is the v argument value.
			V int
		}
		// SetInvocation holds details about calls to the SetInvocation method.
		SetInvocation []struct {
			// Inv is the inv argument value.
			Inv rideObject
		}
		// block holds details about calls to the block method.
		block []struct {
		}
		// checkMessageLength holds details about calls to the checkMessageLength method.
		checkMessageLength []struct {
			// In1 is the in1 argument value.
			In1 int
		}
		// height holds details about calls to the height method.
		height []struct {
		}
		// invocation holds details about calls to the invocation method.
		invocation []struct {
		}
		// scheme holds details about calls to the scheme method.
		scheme []struct {
		}
		// setNewDAppAddress holds details about calls to the setNewDAppAddress method.
		setNewDAppAddress []struct {
			// Address is the address argument value.
			Address proto.Address
		}
		// state holds details about calls to the state method.
		state []struct {
		}
		// this holds details about calls to the this method.
		this []struct {
		}
		// timestamp holds details about calls to the timestamp method.
		timestamp []struct {
		}
		// transaction holds details about calls to the transaction method.
		transaction []struct {
		}
		// txID holds details about calls to the txID method.
		txID []struct {
		}
	}
	lockChooseSizeCheck    sync.RWMutex
	lockSetInvocation      sync.RWMutex
	lockblock              sync.RWMutex
	lockcheckMessageLength sync.RWMutex
	lockheight             sync.RWMutex
	lockinvocation         sync.RWMutex
	lockscheme             sync.RWMutex
	locksetNewDAppAddress  sync.RWMutex
	lockstate              sync.RWMutex
	lockthis               sync.RWMutex
	locktimestamp          sync.RWMutex
	locktransaction        sync.RWMutex
	locktxID               sync.RWMutex
}

// ChooseSizeCheck calls ChooseSizeCheckFunc.
func (mock *MockRideEnvironment) ChooseSizeCheck(v int) {
	if mock.ChooseSizeCheckFunc == nil {
		panic("MockRideEnvironment.ChooseSizeCheckFunc: method is nil but RideEnvironment.ChooseSizeCheck was just called")
	}
	callInfo := struct {
		V int
	}{
		V: v,
	}
	mock.lockChooseSizeCheck.Lock()
	mock.calls.ChooseSizeCheck = append(mock.calls.ChooseSizeCheck, callInfo)
	mock.lockChooseSizeCheck.Unlock()
	mock.ChooseSizeCheckFunc(v)
}

// ChooseSizeCheckCalls gets all the calls that were made to ChooseSizeCheck.
// Check the length with:
//     len(mockedRideEnvironment.ChooseSizeCheckCalls())
func (mock *MockRideEnvironment) ChooseSizeCheckCalls() []struct {
	V int
} {
	var calls []struct {
		V int
	}
	mock.lockChooseSizeCheck.RLock()
	calls = mock.calls.ChooseSizeCheck
	mock.lockChooseSizeCheck.RUnlock()
	return calls
}

// SetInvocation calls SetInvocationFunc.
func (mock *MockRideEnvironment) SetInvocation(inv rideObject) {
	if mock.SetInvocationFunc == nil {
		panic("MockRideEnvironment.SetInvocationFunc: method is nil but RideEnvironment.SetInvocation was just called")
	}
	callInfo := struct {
		Inv rideObject
	}{
		Inv: inv,
	}
	mock.lockSetInvocation.Lock()
	mock.calls.SetInvocation = append(mock.calls.SetInvocation, callInfo)
	mock.lockSetInvocation.Unlock()
	mock.SetInvocationFunc(inv)
}

// SetInvocationCalls gets all the calls that were made to SetInvocation.
// Check the length with:
//     len(mockedRideEnvironment.SetInvocationCalls())
func (mock *MockRideEnvironment) SetInvocationCalls() []struct {
	Inv rideObject
} {
	var calls []struct {
		Inv rideObject
	}
	mock.lockSetInvocation.RLock()
	calls = mock.calls.SetInvocation
	mock.lockSetInvocation.RUnlock()
	return calls
}

// block calls blockFunc.
func (mock *MockRideEnvironment) block() rideObject {
	if mock.blockFunc == nil {
		panic("MockRideEnvironment.blockFunc: method is nil but RideEnvironment.block was just called")
	}
	callInfo := struct {
	}{}
	mock.lockblock.Lock()
	mock.calls.block = append(mock.calls.block, callInfo)
	mock.lockblock.Unlock()
	return mock.blockFunc()
}

// blockCalls gets all the calls that were made to block.
// Check the length with:
//     len(mockedRideEnvironment.blockCalls())
func (mock *MockRideEnvironment) blockCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockblock.RLock()
	calls = mock.calls.block
	mock.lockblock.RUnlock()
	return calls
}

// checkMessageLength calls checkMessageLengthFunc.
func (mock *MockRideEnvironment) checkMessageLength(in1 int) bool {
	if mock.checkMessageLengthFunc == nil {
		panic("MockRideEnvironment.checkMessageLengthFunc: method is nil but RideEnvironment.checkMessageLength was just called")
	}
	callInfo := struct {
		In1 int
	}{
		In1: in1,
	}
	mock.lockcheckMessageLength.Lock()
	mock.calls.checkMessageLength = append(mock.calls.checkMessageLength, callInfo)
	mock.lockcheckMessageLength.Unlock()
	return mock.checkMessageLengthFunc(in1)
}

// checkMessageLengthCalls gets all the calls that were made to checkMessageLength.
// Check the length with:
//     len(mockedRideEnvironment.checkMessageLengthCalls())
func (mock *MockRideEnvironment) checkMessageLengthCalls() []struct {
	In1 int
} {
	var calls []struct {
		In1 int
	}
	mock.lockcheckMessageLength.RLock()
	calls = mock.calls.checkMessageLength
	mock.lockcheckMessageLength.RUnlock()
	return calls
}

// height calls heightFunc.
func (mock *MockRideEnvironment) height() rideInt {
	if mock.heightFunc == nil {
		panic("MockRideEnvironment.heightFunc: method is nil but RideEnvironment.height was just called")
	}
	callInfo := struct {
	}{}
	mock.lockheight.Lock()
	mock.calls.height = append(mock.calls.height, callInfo)
	mock.lockheight.Unlock()
	return mock.heightFunc()
}

// heightCalls gets all the calls that were made to height.
// Check the length with:
//     len(mockedRideEnvironment.heightCalls())
func (mock *MockRideEnvironment) heightCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockheight.RLock()
	calls = mock.calls.height
	mock.lockheight.RUnlock()
	return calls
}

// invocation calls invocationFunc.
func (mock *MockRideEnvironment) invocation() rideObject {
	if mock.invocationFunc == nil {
		panic("MockRideEnvironment.invocationFunc: method is nil but RideEnvironment.invocation was just called")
	}
	callInfo := struct {
	}{}
	mock.lockinvocation.Lock()
	mock.calls.invocation = append(mock.calls.invocation, callInfo)
	mock.lockinvocation.Unlock()
	return mock.invocationFunc()
}

// invocationCalls gets all the calls that were made to invocation.
// Check the length with:
//     len(mockedRideEnvironment.invocationCalls())
func (mock *MockRideEnvironment) invocationCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockinvocation.RLock()
	calls = mock.calls.invocation
	mock.lockinvocation.RUnlock()
	return calls
}

// scheme calls schemeFunc.
func (mock *MockRideEnvironment) scheme() byte {
	if mock.schemeFunc == nil {
		panic("MockRideEnvironment.schemeFunc: method is nil but RideEnvironment.scheme was just called")
	}
	callInfo := struct {
	}{}
	mock.lockscheme.Lock()
	mock.calls.scheme = append(mock.calls.scheme, callInfo)
	mock.lockscheme.Unlock()
	return mock.schemeFunc()
}

// schemeCalls gets all the calls that were made to scheme.
// Check the length with:
//     len(mockedRideEnvironment.schemeCalls())
func (mock *MockRideEnvironment) schemeCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockscheme.RLock()
	calls = mock.calls.scheme
	mock.lockscheme.RUnlock()
	return calls
}

// setNewDAppAddress calls setNewDAppAddressFunc.
func (mock *MockRideEnvironment) setNewDAppAddress(address proto.Address) {
	if mock.setNewDAppAddressFunc == nil {
		panic("MockRideEnvironment.setNewDAppAddressFunc: method is nil but RideEnvironment.setNewDAppAddress was just called")
	}
	callInfo := struct {
		Address proto.Address
	}{
		Address: address,
	}
	mock.locksetNewDAppAddress.Lock()
	mock.calls.setNewDAppAddress = append(mock.calls.setNewDAppAddress, callInfo)
	mock.locksetNewDAppAddress.Unlock()
	mock.setNewDAppAddressFunc(address)
}

// setNewDAppAddressCalls gets all the calls that were made to setNewDAppAddress.
// Check the length with:
//     len(mockedRideEnvironment.setNewDAppAddressCalls())
func (mock *MockRideEnvironment) setNewDAppAddressCalls() []struct {
	Address proto.Address
} {
	var calls []struct {
		Address proto.Address
	}
	mock.locksetNewDAppAddress.RLock()
	calls = mock.calls.setNewDAppAddress
	mock.locksetNewDAppAddress.RUnlock()
	return calls
}

// state calls stateFunc.
func (mock *MockRideEnvironment) state() types.SmartState {
	if mock.stateFunc == nil {
		panic("MockRideEnvironment.stateFunc: method is nil but RideEnvironment.state was just called")
	}
	callInfo := struct {
	}{}
	mock.lockstate.Lock()
	mock.calls.state = append(mock.calls.state, callInfo)
	mock.lockstate.Unlock()
	return mock.stateFunc()
}

// stateCalls gets all the calls that were made to state.
// Check the length with:
//     len(mockedRideEnvironment.stateCalls())
func (mock *MockRideEnvironment) stateCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockstate.RLock()
	calls = mock.calls.state
	mock.lockstate.RUnlock()
	return calls
}

// this calls thisFunc.
func (mock *MockRideEnvironment) this() rideType {
	if mock.thisFunc == nil {
		panic("MockRideEnvironment.thisFunc: method is nil but RideEnvironment.this was just called")
	}
	callInfo := struct {
	}{}
	mock.lockthis.Lock()
	mock.calls.this = append(mock.calls.this, callInfo)
	mock.lockthis.Unlock()
	return mock.thisFunc()
}

// thisCalls gets all the calls that were made to this.
// Check the length with:
//     len(mockedRideEnvironment.thisCalls())
func (mock *MockRideEnvironment) thisCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockthis.RLock()
	calls = mock.calls.this
	mock.lockthis.RUnlock()
	return calls
}

// timestamp calls timestampFunc.
func (mock *MockRideEnvironment) timestamp() uint64 {
	if mock.timestampFunc == nil {
		panic("MockRideEnvironment.timestampFunc: method is nil but RideEnvironment.timestamp was just called")
	}
	callInfo := struct {
	}{}
	mock.locktimestamp.Lock()
	mock.calls.timestamp = append(mock.calls.timestamp, callInfo)
	mock.locktimestamp.Unlock()
	return mock.timestampFunc()
}

// timestampCalls gets all the calls that were made to timestamp.
// Check the length with:
//     len(mockedRideEnvironment.timestampCalls())
func (mock *MockRideEnvironment) timestampCalls() []struct {
} {
	var calls []struct {
	}
	mock.locktimestamp.RLock()
	calls = mock.calls.timestamp
	mock.locktimestamp.RUnlock()
	return calls
}

// transaction calls transactionFunc.
func (mock *MockRideEnvironment) transaction() rideObject {
	if mock.transactionFunc == nil {
		panic("MockRideEnvironment.transactionFunc: method is nil but RideEnvironment.transaction was just called")
	}
	callInfo := struct {
	}{}
	mock.locktransaction.Lock()
	mock.calls.transaction = append(mock.calls.transaction, callInfo)
	mock.locktransaction.Unlock()
	return mock.transactionFunc()
}

// transactionCalls gets all the calls that were made to transaction.
// Check the length with:
//     len(mockedRideEnvironment.transactionCalls())
func (mock *MockRideEnvironment) transactionCalls() []struct {
} {
	var calls []struct {
	}
	mock.locktransaction.RLock()
	calls = mock.calls.transaction
	mock.locktransaction.RUnlock()
	return calls
}

// txID calls txIDFunc.
func (mock *MockRideEnvironment) txID() rideType {
	if mock.txIDFunc == nil {
		panic("MockRideEnvironment.txIDFunc: method is nil but RideEnvironment.txID was just called")
	}
	callInfo := struct {
	}{}
	mock.locktxID.Lock()
	mock.calls.txID = append(mock.calls.txID, callInfo)
	mock.locktxID.Unlock()
	return mock.txIDFunc()
}

// txIDCalls gets all the calls that were made to txID.
// Check the length with:
//     len(mockedRideEnvironment.txIDCalls())
func (mock *MockRideEnvironment) txIDCalls() []struct {
} {
	var calls []struct {
	}
	mock.locktxID.RLock()
	calls = mock.calls.txID
	mock.locktxID.RUnlock()
	return calls
}
