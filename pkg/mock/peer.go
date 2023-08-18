// Code generated by MockGen. DO NOT EDIT.
// Source: pkg/p2p/peer/peer.go

// Package mock is a generated GoMock package.
package mock

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	conn "github.com/wavesplatform/gowaves/pkg/p2p/conn"
	peer "github.com/wavesplatform/gowaves/pkg/p2p/peer"
	proto "github.com/wavesplatform/gowaves/pkg/proto"
)

// MockPeer is a mock of Peer interface.
type MockPeer struct {
	ctrl     *gomock.Controller
	recorder *MockPeerMockRecorder
}

// MockPeerMockRecorder is the mock recorder for MockPeer.
type MockPeerMockRecorder struct {
	mock *MockPeer
}

// NewMockPeer creates a new mock instance.
func NewMockPeer(ctrl *gomock.Controller) *MockPeer {
	mock := &MockPeer{ctrl: ctrl}
	mock.recorder = &MockPeerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockPeer) EXPECT() *MockPeerMockRecorder {
	return m.recorder
}

// Close mocks base method.
func (m *MockPeer) Close() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Close")
	ret0, _ := ret[0].(error)
	return ret0
}

// Close indicates an expected call of Close.
func (mr *MockPeerMockRecorder) Close() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Close", reflect.TypeOf((*MockPeer)(nil).Close))
}

// Connection mocks base method.
func (m *MockPeer) Connection() conn.Connection {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Connection")
	ret0, _ := ret[0].(conn.Connection)
	return ret0
}

// Connection indicates an expected call of Connection.
func (mr *MockPeerMockRecorder) Connection() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Connection", reflect.TypeOf((*MockPeer)(nil).Connection))
}

// Direction mocks base method.
func (m *MockPeer) Direction() peer.Direction {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Direction")
	ret0, _ := ret[0].(peer.Direction)
	return ret0
}

// Direction indicates an expected call of Direction.
func (mr *MockPeerMockRecorder) Direction() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Direction", reflect.TypeOf((*MockPeer)(nil).Direction))
}

// Equal mocks base method.
func (m *MockPeer) Equal(arg0 peer.Peer) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Equal", arg0)
	ret0, _ := ret[0].(bool)
	return ret0
}

// Equal indicates an expected call of Equal.
func (mr *MockPeerMockRecorder) Equal(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Equal", reflect.TypeOf((*MockPeer)(nil).Equal), arg0)
}

// Handshake mocks base method.
func (m *MockPeer) Handshake() proto.Handshake {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Handshake")
	ret0, _ := ret[0].(proto.Handshake)
	return ret0
}

// Handshake indicates an expected call of Handshake.
func (mr *MockPeerMockRecorder) Handshake() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Handshake", reflect.TypeOf((*MockPeer)(nil).Handshake))
}

// ID mocks base method.
func (m *MockPeer) ID() peer.ID {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ID")
	ret0, _ := ret[0].(peer.ID)
	return ret0
}

// ID indicates an expected call of ID.
func (mr *MockPeerMockRecorder) ID() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ID", reflect.TypeOf((*MockPeer)(nil).ID))
}

// RemoteAddr mocks base method.
func (m *MockPeer) RemoteAddr() proto.TCPAddr {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RemoteAddr")
	ret0, _ := ret[0].(proto.TCPAddr)
	return ret0
}

// RemoteAddr indicates an expected call of RemoteAddr.
func (mr *MockPeerMockRecorder) RemoteAddr() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RemoteAddr", reflect.TypeOf((*MockPeer)(nil).RemoteAddr))
}

// SendMessage mocks base method.
func (m *MockPeer) SendMessage(arg0 proto.Message) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SendMessage", arg0)
}

// SendMessage indicates an expected call of SendMessage.
func (mr *MockPeerMockRecorder) SendMessage(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendMessage", reflect.TypeOf((*MockPeer)(nil).SendMessage), arg0)
}
