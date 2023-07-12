// Code generated by MockGen. DO NOT EDIT.
// Source: pkg/node/peers/peer_manager.go

// Package mock is a generated GoMock package.
package mock

import (
	"context"
	"net"
	"reflect"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/wavesplatform/gowaves/pkg/node/peers/storage"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

// MockPeerManager is a mock of PeerManager interface.
type MockPeerManager struct {
	ctrl     *gomock.Controller
	recorder *MockPeerManagerMockRecorder
}

// MockPeerManagerMockRecorder is the mock recorder for MockPeerManager.
type MockPeerManagerMockRecorder struct {
	mock *MockPeerManager
}

// NewMockPeerManager creates a new mock instance.
func NewMockPeerManager(ctrl *gomock.Controller) *MockPeerManager {
	mock := &MockPeerManager{ctrl: ctrl}
	mock.recorder = &MockPeerManagerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockPeerManager) EXPECT() *MockPeerManagerMockRecorder {
	return m.recorder
}

// AddToBlackList mocks base method.
func (m *MockPeerManager) AddToBlackList(peer peer.Peer, blockTime time.Time, reason string) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "AddToBlackList", peer, blockTime, reason)
}

// AddToBlackList indicates an expected call of AddToBlackList.
func (mr *MockPeerManagerMockRecorder) AddToBlackList(peer, blockTime, reason interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddToBlackList", reflect.TypeOf((*MockPeerManager)(nil).AddToBlackList), peer, blockTime, reason)
}

// AskPeers mocks base method.
func (m *MockPeerManager) AskPeers() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "AskPeers")
}

// AskPeers indicates an expected call of AskPeers.
func (mr *MockPeerManagerMockRecorder) AskPeers() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AskPeers", reflect.TypeOf((*MockPeerManager)(nil).AskPeers))
}

// BlackList mocks base method.
func (m *MockPeerManager) BlackList() []storage.BlackListedPeer {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "BlackList")
	ret0, _ := ret[0].([]storage.BlackListedPeer)
	return ret0
}

// BlackList indicates an expected call of BlackList.
func (mr *MockPeerManagerMockRecorder) BlackList() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "BlackList", reflect.TypeOf((*MockPeerManager)(nil).BlackList))
}

// CheckPeerInLargestScoreGroup mocks base method.
func (m *MockPeerManager) CheckPeerInLargestScoreGroup(p peer.Peer) (peer.Peer, bool) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CheckPeerInLargestScoreGroup", p)
	ret0, _ := ret[0].(peer.Peer)
	ret1, _ := ret[1].(bool)
	return ret0, ret1
}

// CheckPeerInLargestScoreGroup indicates an expected call of CheckPeerInLargestScoreGroup.
func (mr *MockPeerManagerMockRecorder) CheckPeerInLargestScoreGroup(p interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CheckPeerInLargestScoreGroup", reflect.TypeOf((*MockPeerManager)(nil).CheckPeerInLargestScoreGroup), p)
}

// CheckPeerWithMaxScore mocks base method.
func (m *MockPeerManager) CheckPeerWithMaxScore(p peer.Peer) (peer.Peer, bool) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CheckPeerWithMaxScore", p)
	ret0, _ := ret[0].(peer.Peer)
	ret1, _ := ret[1].(bool)
	return ret0, ret1
}

// CheckPeerWithMaxScore indicates an expected call of CheckPeerWithMaxScore.
func (mr *MockPeerManagerMockRecorder) CheckPeerWithMaxScore(p interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CheckPeerWithMaxScore", reflect.TypeOf((*MockPeerManager)(nil).CheckPeerWithMaxScore), p)
}

// ClearBlackList mocks base method.
func (m *MockPeerManager) ClearBlackList() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ClearBlackList")
	ret0, _ := ret[0].(error)
	return ret0
}

// ClearBlackList indicates an expected call of ClearBlackList.
func (mr *MockPeerManagerMockRecorder) ClearBlackList() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ClearBlackList", reflect.TypeOf((*MockPeerManager)(nil).ClearBlackList))
}

// Close mocks base method.
func (m *MockPeerManager) Close() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Close")
}

// Close indicates an expected call of Close.
func (mr *MockPeerManagerMockRecorder) Close() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Close", reflect.TypeOf((*MockPeerManager)(nil).Close))
}

// Connect mocks base method.
func (m *MockPeerManager) Connect(arg0 context.Context, arg1 proto.TCPAddr) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Connect", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// Connect indicates an expected call of Connect.
func (mr *MockPeerManagerMockRecorder) Connect(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Connect", reflect.TypeOf((*MockPeerManager)(nil).Connect), arg0, arg1)
}

// ConnectedCount mocks base method.
func (m *MockPeerManager) ConnectedCount() int {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ConnectedCount")
	ret0, _ := ret[0].(int)
	return ret0
}

// ConnectedCount indicates an expected call of ConnectedCount.
func (mr *MockPeerManagerMockRecorder) ConnectedCount() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ConnectedCount", reflect.TypeOf((*MockPeerManager)(nil).ConnectedCount))
}

// Disconnect mocks base method.
func (m *MockPeerManager) Disconnect(arg0 peer.Peer) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Disconnect", arg0)
}

// Disconnect indicates an expected call of Disconnect.
func (mr *MockPeerManagerMockRecorder) Disconnect(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Disconnect", reflect.TypeOf((*MockPeerManager)(nil).Disconnect), arg0)
}

// EachConnected mocks base method.
func (m *MockPeerManager) EachConnected(arg0 func(peer.Peer, *proto.Score)) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "EachConnected", arg0)
}

// EachConnected indicates an expected call of EachConnected.
func (mr *MockPeerManagerMockRecorder) EachConnected(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "EachConnected", reflect.TypeOf((*MockPeerManager)(nil).EachConnected), arg0)
}

// KnownPeers mocks base method.
func (m *MockPeerManager) KnownPeers() []storage.KnownPeer {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "KnownPeers")
	ret0, _ := ret[0].([]storage.KnownPeer)
	return ret0
}

// KnownPeers indicates an expected call of KnownPeers.
func (mr *MockPeerManagerMockRecorder) KnownPeers() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "KnownPeers", reflect.TypeOf((*MockPeerManager)(nil).KnownPeers))
}

// NewConnection mocks base method.
func (m *MockPeerManager) NewConnection(arg0 peer.Peer) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NewConnection", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// NewConnection indicates an expected call of NewConnection.
func (mr *MockPeerManagerMockRecorder) NewConnection(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NewConnection", reflect.TypeOf((*MockPeerManager)(nil).NewConnection), arg0)
}

// Score mocks base method.
func (m *MockPeerManager) Score(p peer.Peer) (*proto.Score, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Score", p)
	ret0, _ := ret[0].(*proto.Score)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Score indicates an expected call of Score.
func (mr *MockPeerManagerMockRecorder) Score(p interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Score", reflect.TypeOf((*MockPeerManager)(nil).Score), p)
}

// SpawnIncomingConnection mocks base method.
func (m *MockPeerManager) SpawnIncomingConnection(ctx context.Context, conn net.Conn) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SpawnIncomingConnection", ctx, conn)
	ret0, _ := ret[0].(error)
	return ret0
}

// SpawnIncomingConnection indicates an expected call of SpawnIncomingConnection.
func (mr *MockPeerManagerMockRecorder) SpawnIncomingConnection(ctx, conn interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SpawnIncomingConnection", reflect.TypeOf((*MockPeerManager)(nil).SpawnIncomingConnection), ctx, conn)
}

// SpawnOutgoingConnections mocks base method.
func (m *MockPeerManager) SpawnOutgoingConnections(arg0 context.Context) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SpawnOutgoingConnections", arg0)
}

// SpawnOutgoingConnections indicates an expected call of SpawnOutgoingConnections.
func (mr *MockPeerManagerMockRecorder) SpawnOutgoingConnections(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SpawnOutgoingConnections", reflect.TypeOf((*MockPeerManager)(nil).SpawnOutgoingConnections), arg0)
}

// Spawned mocks base method.
func (m *MockPeerManager) Spawned() []proto.IpPort {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Spawned")
	ret0, _ := ret[0].([]proto.IpPort)
	return ret0
}

// Spawned indicates an expected call of Spawned.
func (mr *MockPeerManagerMockRecorder) Spawned() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Spawned", reflect.TypeOf((*MockPeerManager)(nil).Spawned))
}

// Suspend mocks base method.
func (m *MockPeerManager) Suspend(peer peer.Peer, suspendTime time.Time, reason string) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Suspend", peer, suspendTime, reason)
}

// Suspend indicates an expected call of Suspend.
func (mr *MockPeerManagerMockRecorder) Suspend(peer, suspendTime, reason interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Suspend", reflect.TypeOf((*MockPeerManager)(nil).Suspend), peer, suspendTime, reason)
}

// Suspended mocks base method.
func (m *MockPeerManager) Suspended() []storage.SuspendedPeer {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Suspended")
	ret0, _ := ret[0].([]storage.SuspendedPeer)
	return ret0
}

// Suspended indicates an expected call of Suspended.
func (mr *MockPeerManagerMockRecorder) Suspended() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Suspended", reflect.TypeOf((*MockPeerManager)(nil).Suspended))
}

// UpdateKnownPeers mocks base method.
func (m *MockPeerManager) UpdateKnownPeers(arg0 []storage.KnownPeer) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateKnownPeers", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateKnownPeers indicates an expected call of UpdateKnownPeers.
func (mr *MockPeerManagerMockRecorder) UpdateKnownPeers(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateKnownPeers", reflect.TypeOf((*MockPeerManager)(nil).UpdateKnownPeers), arg0)
}

// UpdateScore mocks base method.
func (m *MockPeerManager) UpdateScore(p peer.Peer, score *proto.Score) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateScore", p, score)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateScore indicates an expected call of UpdateScore.
func (mr *MockPeerManagerMockRecorder) UpdateScore(p, score interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateScore", reflect.TypeOf((*MockPeerManager)(nil).UpdateScore), p, score)
}
