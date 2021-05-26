// Code generated by MockGen. DO NOT EDIT.
// Source: pkg/grpc/server/api.go

// Package mock is a generated GoMock package.
package mock

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	empty "github.com/golang/protobuf/ptypes/empty"
	wrappers "github.com/golang/protobuf/ptypes/wrappers"
	waves "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves"
	grpc "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves/node/grpc"
)

// MockGrpcHandlers is a mock of GrpcHandlers interface.
type MockGrpcHandlers struct {
	ctrl     *gomock.Controller
	recorder *MockGrpcHandlersMockRecorder
}

// MockGrpcHandlersMockRecorder is the mock recorder for MockGrpcHandlers.
type MockGrpcHandlersMockRecorder struct {
	mock *MockGrpcHandlers
}

// NewMockGrpcHandlers creates a new mock instance.
func NewMockGrpcHandlers(ctrl *gomock.Controller) *MockGrpcHandlers {
	mock := &MockGrpcHandlers{ctrl: ctrl}
	mock.recorder = &MockGrpcHandlersMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockGrpcHandlers) EXPECT() *MockGrpcHandlersMockRecorder {
	return m.recorder
}

// Broadcast mocks base method.
func (m *MockGrpcHandlers) Broadcast(arg0 context.Context, arg1 *waves.SignedTransaction) (*waves.SignedTransaction, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Broadcast", arg0, arg1)
	ret0, _ := ret[0].(*waves.SignedTransaction)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Broadcast indicates an expected call of Broadcast.
func (mr *MockGrpcHandlersMockRecorder) Broadcast(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Broadcast", reflect.TypeOf((*MockGrpcHandlers)(nil).Broadcast), arg0, arg1)
}

// GetActivationStatus mocks base method.
func (m *MockGrpcHandlers) GetActivationStatus(arg0 context.Context, arg1 *grpc.ActivationStatusRequest) (*grpc.ActivationStatusResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetActivationStatus", arg0, arg1)
	ret0, _ := ret[0].(*grpc.ActivationStatusResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetActivationStatus indicates an expected call of GetActivationStatus.
func (mr *MockGrpcHandlersMockRecorder) GetActivationStatus(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetActivationStatus", reflect.TypeOf((*MockGrpcHandlers)(nil).GetActivationStatus), arg0, arg1)
}

// GetActiveLeases mocks base method.
func (m *MockGrpcHandlers) GetActiveLeases(arg0 *grpc.AccountRequest, arg1 grpc.AccountsApi_GetActiveLeasesServer) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetActiveLeases", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// GetActiveLeases indicates an expected call of GetActiveLeases.
func (mr *MockGrpcHandlersMockRecorder) GetActiveLeases(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetActiveLeases", reflect.TypeOf((*MockGrpcHandlers)(nil).GetActiveLeases), arg0, arg1)
}

// GetBalances mocks base method.
func (m *MockGrpcHandlers) GetBalances(arg0 *grpc.BalancesRequest, arg1 grpc.AccountsApi_GetBalancesServer) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetBalances", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// GetBalances indicates an expected call of GetBalances.
func (mr *MockGrpcHandlersMockRecorder) GetBalances(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetBalances", reflect.TypeOf((*MockGrpcHandlers)(nil).GetBalances), arg0, arg1)
}

// GetBaseTarget mocks base method.
func (m *MockGrpcHandlers) GetBaseTarget(arg0 context.Context, arg1 *empty.Empty) (*grpc.BaseTargetResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetBaseTarget", arg0, arg1)
	ret0, _ := ret[0].(*grpc.BaseTargetResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetBaseTarget indicates an expected call of GetBaseTarget.
func (mr *MockGrpcHandlersMockRecorder) GetBaseTarget(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetBaseTarget", reflect.TypeOf((*MockGrpcHandlers)(nil).GetBaseTarget), arg0, arg1)
}

// GetBlock mocks base method.
func (m *MockGrpcHandlers) GetBlock(arg0 context.Context, arg1 *grpc.BlockRequest) (*grpc.BlockWithHeight, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetBlock", arg0, arg1)
	ret0, _ := ret[0].(*grpc.BlockWithHeight)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetBlock indicates an expected call of GetBlock.
func (mr *MockGrpcHandlersMockRecorder) GetBlock(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetBlock", reflect.TypeOf((*MockGrpcHandlers)(nil).GetBlock), arg0, arg1)
}

// GetBlockRange mocks base method.
func (m *MockGrpcHandlers) GetBlockRange(arg0 *grpc.BlockRangeRequest, arg1 grpc.BlocksApi_GetBlockRangeServer) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetBlockRange", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// GetBlockRange indicates an expected call of GetBlockRange.
func (mr *MockGrpcHandlersMockRecorder) GetBlockRange(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetBlockRange", reflect.TypeOf((*MockGrpcHandlers)(nil).GetBlockRange), arg0, arg1)
}

// GetCumulativeScore mocks base method.
func (m *MockGrpcHandlers) GetCumulativeScore(arg0 context.Context, arg1 *empty.Empty) (*grpc.ScoreResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCumulativeScore", arg0, arg1)
	ret0, _ := ret[0].(*grpc.ScoreResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetCumulativeScore indicates an expected call of GetCumulativeScore.
func (mr *MockGrpcHandlersMockRecorder) GetCumulativeScore(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCumulativeScore", reflect.TypeOf((*MockGrpcHandlers)(nil).GetCumulativeScore), arg0, arg1)
}

// GetCurrentHeight mocks base method.
func (m *MockGrpcHandlers) GetCurrentHeight(arg0 context.Context, arg1 *empty.Empty) (*wrappers.UInt32Value, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCurrentHeight", arg0, arg1)
	ret0, _ := ret[0].(*wrappers.UInt32Value)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetCurrentHeight indicates an expected call of GetCurrentHeight.
func (mr *MockGrpcHandlersMockRecorder) GetCurrentHeight(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCurrentHeight", reflect.TypeOf((*MockGrpcHandlers)(nil).GetCurrentHeight), arg0, arg1)
}

// GetDataEntries mocks base method.
func (m *MockGrpcHandlers) GetDataEntries(arg0 *grpc.DataRequest, arg1 grpc.AccountsApi_GetDataEntriesServer) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetDataEntries", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// GetDataEntries indicates an expected call of GetDataEntries.
func (mr *MockGrpcHandlersMockRecorder) GetDataEntries(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetDataEntries", reflect.TypeOf((*MockGrpcHandlers)(nil).GetDataEntries), arg0, arg1)
}

// GetInfo mocks base method.
func (m *MockGrpcHandlers) GetInfo(arg0 context.Context, arg1 *grpc.AssetRequest) (*grpc.AssetInfoResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetInfo", arg0, arg1)
	ret0, _ := ret[0].(*grpc.AssetInfoResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetInfo indicates an expected call of GetInfo.
func (mr *MockGrpcHandlersMockRecorder) GetInfo(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetInfo", reflect.TypeOf((*MockGrpcHandlers)(nil).GetInfo), arg0, arg1)
}

// GetNFTList mocks base method.
func (m *MockGrpcHandlers) GetNFTList(arg0 *grpc.NFTRequest, arg1 grpc.AssetsApi_GetNFTListServer) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetNFTList", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// GetNFTList indicates an expected call of GetNFTList.
func (mr *MockGrpcHandlersMockRecorder) GetNFTList(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetNFTList", reflect.TypeOf((*MockGrpcHandlers)(nil).GetNFTList), arg0, arg1)
}

// GetScript mocks base method.
func (m *MockGrpcHandlers) GetScript(arg0 context.Context, arg1 *grpc.AccountRequest) (*grpc.ScriptData, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetScript", arg0, arg1)
	ret0, _ := ret[0].(*grpc.ScriptData)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetScript indicates an expected call of GetScript.
func (mr *MockGrpcHandlersMockRecorder) GetScript(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetScript", reflect.TypeOf((*MockGrpcHandlers)(nil).GetScript), arg0, arg1)
}

// GetStateChanges mocks base method.
func (m *MockGrpcHandlers) GetStateChanges(arg0 *grpc.TransactionsRequest, arg1 grpc.TransactionsApi_GetStateChangesServer) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetStateChanges", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// GetStateChanges indicates an expected call of GetStateChanges.
func (mr *MockGrpcHandlersMockRecorder) GetStateChanges(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetStateChanges", reflect.TypeOf((*MockGrpcHandlers)(nil).GetStateChanges), arg0, arg1)
}

// GetStatuses mocks base method.
func (m *MockGrpcHandlers) GetStatuses(arg0 *grpc.TransactionsByIdRequest, arg1 grpc.TransactionsApi_GetStatusesServer) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetStatuses", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// GetStatuses indicates an expected call of GetStatuses.
func (mr *MockGrpcHandlersMockRecorder) GetStatuses(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetStatuses", reflect.TypeOf((*MockGrpcHandlers)(nil).GetStatuses), arg0, arg1)
}

// GetTransactions mocks base method.
func (m *MockGrpcHandlers) GetTransactions(arg0 *grpc.TransactionsRequest, arg1 grpc.TransactionsApi_GetTransactionsServer) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetTransactions", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// GetTransactions indicates an expected call of GetTransactions.
func (mr *MockGrpcHandlersMockRecorder) GetTransactions(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetTransactions", reflect.TypeOf((*MockGrpcHandlers)(nil).GetTransactions), arg0, arg1)
}

// GetUnconfirmed mocks base method.
func (m *MockGrpcHandlers) GetUnconfirmed(arg0 *grpc.TransactionsRequest, arg1 grpc.TransactionsApi_GetUnconfirmedServer) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetUnconfirmed", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// GetUnconfirmed indicates an expected call of GetUnconfirmed.
func (mr *MockGrpcHandlersMockRecorder) GetUnconfirmed(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetUnconfirmed", reflect.TypeOf((*MockGrpcHandlers)(nil).GetUnconfirmed), arg0, arg1)
}

// ResolveAlias mocks base method.
func (m *MockGrpcHandlers) ResolveAlias(arg0 context.Context, arg1 *wrappers.StringValue) (*wrappers.BytesValue, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ResolveAlias", arg0, arg1)
	ret0, _ := ret[0].(*wrappers.BytesValue)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ResolveAlias indicates an expected call of ResolveAlias.
func (mr *MockGrpcHandlersMockRecorder) ResolveAlias(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ResolveAlias", reflect.TypeOf((*MockGrpcHandlers)(nil).ResolveAlias), arg0, arg1)
}

// Sign mocks base method.
func (m *MockGrpcHandlers) Sign(arg0 context.Context, arg1 *grpc.SignRequest) (*waves.SignedTransaction, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Sign", arg0, arg1)
	ret0, _ := ret[0].(*waves.SignedTransaction)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Sign indicates an expected call of Sign.
func (mr *MockGrpcHandlersMockRecorder) Sign(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Sign", reflect.TypeOf((*MockGrpcHandlers)(nil).Sign), arg0, arg1)
}
