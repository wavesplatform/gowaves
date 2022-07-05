// Code generated by MockGen. DO NOT EDIT.
// Source: pkg/miner/utxpool/cleaner.go

// Package utxpool is a generated GoMock package.
package utxpool

import (
	gomock "github.com/golang/mock/gomock"
	proto "github.com/wavesplatform/gowaves/pkg/proto"
	state "github.com/wavesplatform/gowaves/pkg/state"
	reflect "reflect"
)

// MockstateWrapper is a mock of stateWrapper interface
type MockstateWrapper struct {
	ctrl     *gomock.Controller
	recorder *MockstateWrapperMockRecorder
}

// MockstateWrapperMockRecorder is the mock recorder for MockstateWrapper
type MockstateWrapperMockRecorder struct {
	mock *MockstateWrapper
}

// NewMockstateWrapper creates a new mock instance
func NewMockstateWrapper(ctrl *gomock.Controller) *MockstateWrapper {
	mock := &MockstateWrapper{ctrl: ctrl}
	mock.recorder = &MockstateWrapperMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockstateWrapper) EXPECT() *MockstateWrapperMockRecorder {
	return m.recorder
}

// Height mocks base method
func (m *MockstateWrapper) Height() (proto.Height, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Height")
	ret0, _ := ret[0].(proto.Height)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Height indicates an expected call of Height
func (mr *MockstateWrapperMockRecorder) Height() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Height", reflect.TypeOf((*MockstateWrapper)(nil).Height))
}

// TopBlock mocks base method
func (m *MockstateWrapper) TopBlock() *proto.Block {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "TopBlock")
	ret0, _ := ret[0].(*proto.Block)
	return ret0
}

// TopBlock indicates an expected call of TopBlock
func (mr *MockstateWrapperMockRecorder) TopBlock() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "TopBlock", reflect.TypeOf((*MockstateWrapper)(nil).TopBlock))
}

// TxValidation mocks base method
func (m *MockstateWrapper) TxValidation(arg0 func(state.TxValidation) error) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "TxValidation", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// TxValidation indicates an expected call of TxValidation
func (mr *MockstateWrapperMockRecorder) TxValidation(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "TxValidation", reflect.TypeOf((*MockstateWrapper)(nil).TxValidation), arg0)
}

// Map mocks base method
func (m *MockstateWrapper) Map(arg0 func(state.NonThreadSafeState) error) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Map", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Map indicates an expected call of Map
func (mr *MockstateWrapperMockRecorder) Map(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Map", reflect.TypeOf((*MockstateWrapper)(nil).Map), arg0)
}

// IsActivated mocks base method
func (m *MockstateWrapper) IsActivated(featureID int16) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsActivated", featureID)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// IsActivated indicates an expected call of IsActivated
func (mr *MockstateWrapperMockRecorder) IsActivated(featureID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsActivated", reflect.TypeOf((*MockstateWrapper)(nil).IsActivated), featureID)
}
