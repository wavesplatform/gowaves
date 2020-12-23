// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.25.0
// 	protoc        v3.14.0
// source: waves/node/grpc/accounts_api.proto

package grpc

import (
	context "context"
	proto "github.com/golang/protobuf/proto"
	waves "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	wrapperspb "google.golang.org/protobuf/types/known/wrapperspb"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// This is a compile-time assertion that a sufficiently up-to-date version
// of the legacy proto package is being used.
const _ = proto.ProtoPackageIsVersion4

type AccountRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Address []byte `protobuf:"bytes,1,opt,name=address,proto3" json:"address,omitempty"`
}

func (x *AccountRequest) Reset() {
	*x = AccountRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_waves_node_grpc_accounts_api_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *AccountRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AccountRequest) ProtoMessage() {}

func (x *AccountRequest) ProtoReflect() protoreflect.Message {
	mi := &file_waves_node_grpc_accounts_api_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AccountRequest.ProtoReflect.Descriptor instead.
func (*AccountRequest) Descriptor() ([]byte, []int) {
	return file_waves_node_grpc_accounts_api_proto_rawDescGZIP(), []int{0}
}

func (x *AccountRequest) GetAddress() []byte {
	if x != nil {
		return x.Address
	}
	return nil
}

type DataRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Address []byte `protobuf:"bytes,1,opt,name=address,proto3" json:"address,omitempty"`
	Key     string `protobuf:"bytes,2,opt,name=key,proto3" json:"key,omitempty"`
}

func (x *DataRequest) Reset() {
	*x = DataRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_waves_node_grpc_accounts_api_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *DataRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DataRequest) ProtoMessage() {}

func (x *DataRequest) ProtoReflect() protoreflect.Message {
	mi := &file_waves_node_grpc_accounts_api_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DataRequest.ProtoReflect.Descriptor instead.
func (*DataRequest) Descriptor() ([]byte, []int) {
	return file_waves_node_grpc_accounts_api_proto_rawDescGZIP(), []int{1}
}

func (x *DataRequest) GetAddress() []byte {
	if x != nil {
		return x.Address
	}
	return nil
}

func (x *DataRequest) GetKey() string {
	if x != nil {
		return x.Key
	}
	return ""
}

type BalancesRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Address []byte   `protobuf:"bytes,1,opt,name=address,proto3" json:"address,omitempty"`
	Assets  [][]byte `protobuf:"bytes,4,rep,name=assets,proto3" json:"assets,omitempty"`
}

func (x *BalancesRequest) Reset() {
	*x = BalancesRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_waves_node_grpc_accounts_api_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *BalancesRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*BalancesRequest) ProtoMessage() {}

func (x *BalancesRequest) ProtoReflect() protoreflect.Message {
	mi := &file_waves_node_grpc_accounts_api_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use BalancesRequest.ProtoReflect.Descriptor instead.
func (*BalancesRequest) Descriptor() ([]byte, []int) {
	return file_waves_node_grpc_accounts_api_proto_rawDescGZIP(), []int{2}
}

func (x *BalancesRequest) GetAddress() []byte {
	if x != nil {
		return x.Address
	}
	return nil
}

func (x *BalancesRequest) GetAssets() [][]byte {
	if x != nil {
		return x.Assets
	}
	return nil
}

type BalanceResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Types that are assignable to Balance:
	//	*BalanceResponse_Waves
	//	*BalanceResponse_Asset
	Balance isBalanceResponse_Balance `protobuf_oneof:"balance"`
}

func (x *BalanceResponse) Reset() {
	*x = BalanceResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_waves_node_grpc_accounts_api_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *BalanceResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*BalanceResponse) ProtoMessage() {}

func (x *BalanceResponse) ProtoReflect() protoreflect.Message {
	mi := &file_waves_node_grpc_accounts_api_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use BalanceResponse.ProtoReflect.Descriptor instead.
func (*BalanceResponse) Descriptor() ([]byte, []int) {
	return file_waves_node_grpc_accounts_api_proto_rawDescGZIP(), []int{3}
}

func (m *BalanceResponse) GetBalance() isBalanceResponse_Balance {
	if m != nil {
		return m.Balance
	}
	return nil
}

func (x *BalanceResponse) GetWaves() *BalanceResponse_WavesBalances {
	if x, ok := x.GetBalance().(*BalanceResponse_Waves); ok {
		return x.Waves
	}
	return nil
}

func (x *BalanceResponse) GetAsset() *waves.Amount {
	if x, ok := x.GetBalance().(*BalanceResponse_Asset); ok {
		return x.Asset
	}
	return nil
}

type isBalanceResponse_Balance interface {
	isBalanceResponse_Balance()
}

type BalanceResponse_Waves struct {
	Waves *BalanceResponse_WavesBalances `protobuf:"bytes,1,opt,name=waves,proto3,oneof"`
}

type BalanceResponse_Asset struct {
	Asset *waves.Amount `protobuf:"bytes,2,opt,name=asset,proto3,oneof"`
}

func (*BalanceResponse_Waves) isBalanceResponse_Balance() {}

func (*BalanceResponse_Asset) isBalanceResponse_Balance() {}

type DataEntryResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Address []byte                               `protobuf:"bytes,1,opt,name=address,proto3" json:"address,omitempty"`
	Entry   *waves.DataTransactionData_DataEntry `protobuf:"bytes,2,opt,name=entry,proto3" json:"entry,omitempty"`
}

func (x *DataEntryResponse) Reset() {
	*x = DataEntryResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_waves_node_grpc_accounts_api_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *DataEntryResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DataEntryResponse) ProtoMessage() {}

func (x *DataEntryResponse) ProtoReflect() protoreflect.Message {
	mi := &file_waves_node_grpc_accounts_api_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DataEntryResponse.ProtoReflect.Descriptor instead.
func (*DataEntryResponse) Descriptor() ([]byte, []int) {
	return file_waves_node_grpc_accounts_api_proto_rawDescGZIP(), []int{4}
}

func (x *DataEntryResponse) GetAddress() []byte {
	if x != nil {
		return x.Address
	}
	return nil
}

func (x *DataEntryResponse) GetEntry() *waves.DataTransactionData_DataEntry {
	if x != nil {
		return x.Entry
	}
	return nil
}

type ScriptData struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	ScriptBytes []byte `protobuf:"bytes,1,opt,name=script_bytes,json=scriptBytes,proto3" json:"script_bytes,omitempty"`
	ScriptText  string `protobuf:"bytes,2,opt,name=script_text,json=scriptText,proto3" json:"script_text,omitempty"`
	Complexity  int64  `protobuf:"varint,3,opt,name=complexity,proto3" json:"complexity,omitempty"`
}

func (x *ScriptData) Reset() {
	*x = ScriptData{}
	if protoimpl.UnsafeEnabled {
		mi := &file_waves_node_grpc_accounts_api_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ScriptData) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ScriptData) ProtoMessage() {}

func (x *ScriptData) ProtoReflect() protoreflect.Message {
	mi := &file_waves_node_grpc_accounts_api_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ScriptData.ProtoReflect.Descriptor instead.
func (*ScriptData) Descriptor() ([]byte, []int) {
	return file_waves_node_grpc_accounts_api_proto_rawDescGZIP(), []int{5}
}

func (x *ScriptData) GetScriptBytes() []byte {
	if x != nil {
		return x.ScriptBytes
	}
	return nil
}

func (x *ScriptData) GetScriptText() string {
	if x != nil {
		return x.ScriptText
	}
	return ""
}

func (x *ScriptData) GetComplexity() int64 {
	if x != nil {
		return x.Complexity
	}
	return 0
}

type BalanceResponse_WavesBalances struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Regular    int64 `protobuf:"varint,1,opt,name=regular,proto3" json:"regular,omitempty"`
	Generating int64 `protobuf:"varint,2,opt,name=generating,proto3" json:"generating,omitempty"`
	Available  int64 `protobuf:"varint,3,opt,name=available,proto3" json:"available,omitempty"`
	Effective  int64 `protobuf:"varint,4,opt,name=effective,proto3" json:"effective,omitempty"`
	LeaseIn    int64 `protobuf:"varint,5,opt,name=lease_in,json=leaseIn,proto3" json:"lease_in,omitempty"`
	LeaseOut   int64 `protobuf:"varint,6,opt,name=lease_out,json=leaseOut,proto3" json:"lease_out,omitempty"`
}

func (x *BalanceResponse_WavesBalances) Reset() {
	*x = BalanceResponse_WavesBalances{}
	if protoimpl.UnsafeEnabled {
		mi := &file_waves_node_grpc_accounts_api_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *BalanceResponse_WavesBalances) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*BalanceResponse_WavesBalances) ProtoMessage() {}

func (x *BalanceResponse_WavesBalances) ProtoReflect() protoreflect.Message {
	mi := &file_waves_node_grpc_accounts_api_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use BalanceResponse_WavesBalances.ProtoReflect.Descriptor instead.
func (*BalanceResponse_WavesBalances) Descriptor() ([]byte, []int) {
	return file_waves_node_grpc_accounts_api_proto_rawDescGZIP(), []int{3, 0}
}

func (x *BalanceResponse_WavesBalances) GetRegular() int64 {
	if x != nil {
		return x.Regular
	}
	return 0
}

func (x *BalanceResponse_WavesBalances) GetGenerating() int64 {
	if x != nil {
		return x.Generating
	}
	return 0
}

func (x *BalanceResponse_WavesBalances) GetAvailable() int64 {
	if x != nil {
		return x.Available
	}
	return 0
}

func (x *BalanceResponse_WavesBalances) GetEffective() int64 {
	if x != nil {
		return x.Effective
	}
	return 0
}

func (x *BalanceResponse_WavesBalances) GetLeaseIn() int64 {
	if x != nil {
		return x.LeaseIn
	}
	return 0
}

func (x *BalanceResponse_WavesBalances) GetLeaseOut() int64 {
	if x != nil {
		return x.LeaseOut
	}
	return 0
}

var File_waves_node_grpc_accounts_api_proto protoreflect.FileDescriptor

var file_waves_node_grpc_accounts_api_proto_rawDesc = []byte{
	0x0a, 0x22, 0x77, 0x61, 0x76, 0x65, 0x73, 0x2f, 0x6e, 0x6f, 0x64, 0x65, 0x2f, 0x67, 0x72, 0x70,
	0x63, 0x2f, 0x61, 0x63, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x73, 0x5f, 0x61, 0x70, 0x69, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0f, 0x77, 0x61, 0x76, 0x65, 0x73, 0x2e, 0x6e, 0x6f, 0x64, 0x65,
	0x2e, 0x67, 0x72, 0x70, 0x63, 0x1a, 0x26, 0x77, 0x61, 0x76, 0x65, 0x73, 0x2f, 0x6e, 0x6f, 0x64,
	0x65, 0x2f, 0x67, 0x72, 0x70, 0x63, 0x2f, 0x74, 0x72, 0x61, 0x6e, 0x73, 0x61, 0x63, 0x74, 0x69,
	0x6f, 0x6e, 0x73, 0x5f, 0x61, 0x70, 0x69, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x12, 0x77,
	0x61, 0x76, 0x65, 0x73, 0x2f, 0x61, 0x6d, 0x6f, 0x75, 0x6e, 0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x1a, 0x17, 0x77, 0x61, 0x76, 0x65, 0x73, 0x2f, 0x74, 0x72, 0x61, 0x6e, 0x73, 0x61, 0x63,
	0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1e, 0x67, 0x6f, 0x6f, 0x67,
	0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x77, 0x72, 0x61, 0x70,
	0x70, 0x65, 0x72, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x2a, 0x0a, 0x0e, 0x41, 0x63,
	0x63, 0x6f, 0x75, 0x6e, 0x74, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x18, 0x0a, 0x07,
	0x61, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x07, 0x61,
	0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x22, 0x39, 0x0a, 0x0b, 0x44, 0x61, 0x74, 0x61, 0x52, 0x65,
	0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x18, 0x0a, 0x07, 0x61, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x07, 0x61, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x12,
	0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65,
	0x79, 0x22, 0x43, 0x0a, 0x0f, 0x42, 0x61, 0x6c, 0x61, 0x6e, 0x63, 0x65, 0x73, 0x52, 0x65, 0x71,
	0x75, 0x65, 0x73, 0x74, 0x12, 0x18, 0x0a, 0x07, 0x61, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x07, 0x61, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x12, 0x16,
	0x0a, 0x06, 0x61, 0x73, 0x73, 0x65, 0x74, 0x73, 0x18, 0x04, 0x20, 0x03, 0x28, 0x0c, 0x52, 0x06,
	0x61, 0x73, 0x73, 0x65, 0x74, 0x73, 0x22, 0xcb, 0x02, 0x0a, 0x0f, 0x42, 0x61, 0x6c, 0x61, 0x6e,
	0x63, 0x65, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x46, 0x0a, 0x05, 0x77, 0x61,
	0x76, 0x65, 0x73, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x2e, 0x2e, 0x77, 0x61, 0x76, 0x65,
	0x73, 0x2e, 0x6e, 0x6f, 0x64, 0x65, 0x2e, 0x67, 0x72, 0x70, 0x63, 0x2e, 0x42, 0x61, 0x6c, 0x61,
	0x6e, 0x63, 0x65, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x2e, 0x57, 0x61, 0x76, 0x65,
	0x73, 0x42, 0x61, 0x6c, 0x61, 0x6e, 0x63, 0x65, 0x73, 0x48, 0x00, 0x52, 0x05, 0x77, 0x61, 0x76,
	0x65, 0x73, 0x12, 0x25, 0x0a, 0x05, 0x61, 0x73, 0x73, 0x65, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28,
	0x0b, 0x32, 0x0d, 0x2e, 0x77, 0x61, 0x76, 0x65, 0x73, 0x2e, 0x41, 0x6d, 0x6f, 0x75, 0x6e, 0x74,
	0x48, 0x00, 0x52, 0x05, 0x61, 0x73, 0x73, 0x65, 0x74, 0x1a, 0xbd, 0x01, 0x0a, 0x0d, 0x57, 0x61,
	0x76, 0x65, 0x73, 0x42, 0x61, 0x6c, 0x61, 0x6e, 0x63, 0x65, 0x73, 0x12, 0x18, 0x0a, 0x07, 0x72,
	0x65, 0x67, 0x75, 0x6c, 0x61, 0x72, 0x18, 0x01, 0x20, 0x01, 0x28, 0x03, 0x52, 0x07, 0x72, 0x65,
	0x67, 0x75, 0x6c, 0x61, 0x72, 0x12, 0x1e, 0x0a, 0x0a, 0x67, 0x65, 0x6e, 0x65, 0x72, 0x61, 0x74,
	0x69, 0x6e, 0x67, 0x18, 0x02, 0x20, 0x01, 0x28, 0x03, 0x52, 0x0a, 0x67, 0x65, 0x6e, 0x65, 0x72,
	0x61, 0x74, 0x69, 0x6e, 0x67, 0x12, 0x1c, 0x0a, 0x09, 0x61, 0x76, 0x61, 0x69, 0x6c, 0x61, 0x62,
	0x6c, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x03, 0x52, 0x09, 0x61, 0x76, 0x61, 0x69, 0x6c, 0x61,
	0x62, 0x6c, 0x65, 0x12, 0x1c, 0x0a, 0x09, 0x65, 0x66, 0x66, 0x65, 0x63, 0x74, 0x69, 0x76, 0x65,
	0x18, 0x04, 0x20, 0x01, 0x28, 0x03, 0x52, 0x09, 0x65, 0x66, 0x66, 0x65, 0x63, 0x74, 0x69, 0x76,
	0x65, 0x12, 0x19, 0x0a, 0x08, 0x6c, 0x65, 0x61, 0x73, 0x65, 0x5f, 0x69, 0x6e, 0x18, 0x05, 0x20,
	0x01, 0x28, 0x03, 0x52, 0x07, 0x6c, 0x65, 0x61, 0x73, 0x65, 0x49, 0x6e, 0x12, 0x1b, 0x0a, 0x09,
	0x6c, 0x65, 0x61, 0x73, 0x65, 0x5f, 0x6f, 0x75, 0x74, 0x18, 0x06, 0x20, 0x01, 0x28, 0x03, 0x52,
	0x08, 0x6c, 0x65, 0x61, 0x73, 0x65, 0x4f, 0x75, 0x74, 0x42, 0x09, 0x0a, 0x07, 0x62, 0x61, 0x6c,
	0x61, 0x6e, 0x63, 0x65, 0x22, 0x69, 0x0a, 0x11, 0x44, 0x61, 0x74, 0x61, 0x45, 0x6e, 0x74, 0x72,
	0x79, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x18, 0x0a, 0x07, 0x61, 0x64, 0x64,
	0x72, 0x65, 0x73, 0x73, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x07, 0x61, 0x64, 0x64, 0x72,
	0x65, 0x73, 0x73, 0x12, 0x3a, 0x0a, 0x05, 0x65, 0x6e, 0x74, 0x72, 0x79, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x24, 0x2e, 0x77, 0x61, 0x76, 0x65, 0x73, 0x2e, 0x44, 0x61, 0x74, 0x61, 0x54,
	0x72, 0x61, 0x6e, 0x73, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x44, 0x61, 0x74, 0x61, 0x2e, 0x44,
	0x61, 0x74, 0x61, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x05, 0x65, 0x6e, 0x74, 0x72, 0x79, 0x22,
	0x70, 0x0a, 0x0a, 0x53, 0x63, 0x72, 0x69, 0x70, 0x74, 0x44, 0x61, 0x74, 0x61, 0x12, 0x21, 0x0a,
	0x0c, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x5f, 0x62, 0x79, 0x74, 0x65, 0x73, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x0c, 0x52, 0x0b, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x42, 0x79, 0x74, 0x65, 0x73,
	0x12, 0x1f, 0x0a, 0x0b, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x5f, 0x74, 0x65, 0x78, 0x74, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0a, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x54, 0x65, 0x78,
	0x74, 0x12, 0x1e, 0x0a, 0x0a, 0x63, 0x6f, 0x6d, 0x70, 0x6c, 0x65, 0x78, 0x69, 0x74, 0x79, 0x18,
	0x03, 0x20, 0x01, 0x28, 0x03, 0x52, 0x0a, 0x63, 0x6f, 0x6d, 0x70, 0x6c, 0x65, 0x78, 0x69, 0x74,
	0x79, 0x32, 0xaa, 0x03, 0x0a, 0x0b, 0x41, 0x63, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x73, 0x41, 0x70,
	0x69, 0x12, 0x53, 0x0a, 0x0b, 0x47, 0x65, 0x74, 0x42, 0x61, 0x6c, 0x61, 0x6e, 0x63, 0x65, 0x73,
	0x12, 0x20, 0x2e, 0x77, 0x61, 0x76, 0x65, 0x73, 0x2e, 0x6e, 0x6f, 0x64, 0x65, 0x2e, 0x67, 0x72,
	0x70, 0x63, 0x2e, 0x42, 0x61, 0x6c, 0x61, 0x6e, 0x63, 0x65, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65,
	0x73, 0x74, 0x1a, 0x20, 0x2e, 0x77, 0x61, 0x76, 0x65, 0x73, 0x2e, 0x6e, 0x6f, 0x64, 0x65, 0x2e,
	0x67, 0x72, 0x70, 0x63, 0x2e, 0x42, 0x61, 0x6c, 0x61, 0x6e, 0x63, 0x65, 0x52, 0x65, 0x73, 0x70,
	0x6f, 0x6e, 0x73, 0x65, 0x30, 0x01, 0x12, 0x49, 0x0a, 0x09, 0x47, 0x65, 0x74, 0x53, 0x63, 0x72,
	0x69, 0x70, 0x74, 0x12, 0x1f, 0x2e, 0x77, 0x61, 0x76, 0x65, 0x73, 0x2e, 0x6e, 0x6f, 0x64, 0x65,
	0x2e, 0x67, 0x72, 0x70, 0x63, 0x2e, 0x41, 0x63, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x52, 0x65, 0x71,
	0x75, 0x65, 0x73, 0x74, 0x1a, 0x1b, 0x2e, 0x77, 0x61, 0x76, 0x65, 0x73, 0x2e, 0x6e, 0x6f, 0x64,
	0x65, 0x2e, 0x67, 0x72, 0x70, 0x63, 0x2e, 0x53, 0x63, 0x72, 0x69, 0x70, 0x74, 0x44, 0x61, 0x74,
	0x61, 0x12, 0x5a, 0x0a, 0x0f, 0x47, 0x65, 0x74, 0x41, 0x63, 0x74, 0x69, 0x76, 0x65, 0x4c, 0x65,
	0x61, 0x73, 0x65, 0x73, 0x12, 0x1f, 0x2e, 0x77, 0x61, 0x76, 0x65, 0x73, 0x2e, 0x6e, 0x6f, 0x64,
	0x65, 0x2e, 0x67, 0x72, 0x70, 0x63, 0x2e, 0x41, 0x63, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x52, 0x65,
	0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x24, 0x2e, 0x77, 0x61, 0x76, 0x65, 0x73, 0x2e, 0x6e, 0x6f,
	0x64, 0x65, 0x2e, 0x67, 0x72, 0x70, 0x63, 0x2e, 0x54, 0x72, 0x61, 0x6e, 0x73, 0x61, 0x63, 0x74,
	0x69, 0x6f, 0x6e, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x30, 0x01, 0x12, 0x54, 0x0a,
	0x0e, 0x47, 0x65, 0x74, 0x44, 0x61, 0x74, 0x61, 0x45, 0x6e, 0x74, 0x72, 0x69, 0x65, 0x73, 0x12,
	0x1c, 0x2e, 0x77, 0x61, 0x76, 0x65, 0x73, 0x2e, 0x6e, 0x6f, 0x64, 0x65, 0x2e, 0x67, 0x72, 0x70,
	0x63, 0x2e, 0x44, 0x61, 0x74, 0x61, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x22, 0x2e,
	0x77, 0x61, 0x76, 0x65, 0x73, 0x2e, 0x6e, 0x6f, 0x64, 0x65, 0x2e, 0x67, 0x72, 0x70, 0x63, 0x2e,
	0x44, 0x61, 0x74, 0x61, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73,
	0x65, 0x30, 0x01, 0x12, 0x49, 0x0a, 0x0c, 0x52, 0x65, 0x73, 0x6f, 0x6c, 0x76, 0x65, 0x41, 0x6c,
	0x69, 0x61, 0x73, 0x12, 0x1c, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x53, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x56, 0x61, 0x6c, 0x75,
	0x65, 0x1a, 0x1b, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x62, 0x75, 0x66, 0x2e, 0x42, 0x79, 0x74, 0x65, 0x73, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x42, 0x73,
	0x0a, 0x1a, 0x63, 0x6f, 0x6d, 0x2e, 0x77, 0x61, 0x76, 0x65, 0x73, 0x70, 0x6c, 0x61, 0x74, 0x66,
	0x6f, 0x72, 0x6d, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x67, 0x72, 0x70, 0x63, 0x5a, 0x43, 0x67, 0x69,
	0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x77, 0x61, 0x76, 0x65, 0x73, 0x70, 0x6c,
	0x61, 0x74, 0x66, 0x6f, 0x72, 0x6d, 0x2f, 0x67, 0x6f, 0x77, 0x61, 0x76, 0x65, 0x73, 0x2f, 0x70,
	0x6b, 0x67, 0x2f, 0x67, 0x72, 0x70, 0x63, 0x2f, 0x67, 0x65, 0x6e, 0x65, 0x72, 0x61, 0x74, 0x65,
	0x64, 0x2f, 0x77, 0x61, 0x76, 0x65, 0x73, 0x2f, 0x6e, 0x6f, 0x64, 0x65, 0x2f, 0x67, 0x72, 0x70,
	0x63, 0xaa, 0x02, 0x0f, 0x57, 0x61, 0x76, 0x65, 0x73, 0x2e, 0x4e, 0x6f, 0x64, 0x65, 0x2e, 0x47,
	0x72, 0x70, 0x63, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_waves_node_grpc_accounts_api_proto_rawDescOnce sync.Once
	file_waves_node_grpc_accounts_api_proto_rawDescData = file_waves_node_grpc_accounts_api_proto_rawDesc
)

func file_waves_node_grpc_accounts_api_proto_rawDescGZIP() []byte {
	file_waves_node_grpc_accounts_api_proto_rawDescOnce.Do(func() {
		file_waves_node_grpc_accounts_api_proto_rawDescData = protoimpl.X.CompressGZIP(file_waves_node_grpc_accounts_api_proto_rawDescData)
	})
	return file_waves_node_grpc_accounts_api_proto_rawDescData
}

var file_waves_node_grpc_accounts_api_proto_msgTypes = make([]protoimpl.MessageInfo, 7)
var file_waves_node_grpc_accounts_api_proto_goTypes = []interface{}{
	(*AccountRequest)(nil),                      // 0: waves.node.grpc.AccountRequest
	(*DataRequest)(nil),                         // 1: waves.node.grpc.DataRequest
	(*BalancesRequest)(nil),                     // 2: waves.node.grpc.BalancesRequest
	(*BalanceResponse)(nil),                     // 3: waves.node.grpc.BalanceResponse
	(*DataEntryResponse)(nil),                   // 4: waves.node.grpc.DataEntryResponse
	(*ScriptData)(nil),                          // 5: waves.node.grpc.ScriptData
	(*BalanceResponse_WavesBalances)(nil),       // 6: waves.node.grpc.BalanceResponse.WavesBalances
	(*waves.Amount)(nil),                        // 7: waves.Amount
	(*waves.DataTransactionData_DataEntry)(nil), // 8: waves.DataTransactionData.DataEntry
	(*wrapperspb.StringValue)(nil),              // 9: google.protobuf.StringValue
	(*TransactionResponse)(nil),                 // 10: waves.node.grpc.TransactionResponse
	(*wrapperspb.BytesValue)(nil),               // 11: google.protobuf.BytesValue
}
var file_waves_node_grpc_accounts_api_proto_depIdxs = []int32{
	6,  // 0: waves.node.grpc.BalanceResponse.waves:type_name -> waves.node.grpc.BalanceResponse.WavesBalances
	7,  // 1: waves.node.grpc.BalanceResponse.asset:type_name -> waves.Amount
	8,  // 2: waves.node.grpc.DataEntryResponse.entry:type_name -> waves.DataTransactionData.DataEntry
	2,  // 3: waves.node.grpc.AccountsApi.GetBalances:input_type -> waves.node.grpc.BalancesRequest
	0,  // 4: waves.node.grpc.AccountsApi.GetScript:input_type -> waves.node.grpc.AccountRequest
	0,  // 5: waves.node.grpc.AccountsApi.GetActiveLeases:input_type -> waves.node.grpc.AccountRequest
	1,  // 6: waves.node.grpc.AccountsApi.GetDataEntries:input_type -> waves.node.grpc.DataRequest
	9,  // 7: waves.node.grpc.AccountsApi.ResolveAlias:input_type -> google.protobuf.StringValue
	3,  // 8: waves.node.grpc.AccountsApi.GetBalances:output_type -> waves.node.grpc.BalanceResponse
	5,  // 9: waves.node.grpc.AccountsApi.GetScript:output_type -> waves.node.grpc.ScriptData
	10, // 10: waves.node.grpc.AccountsApi.GetActiveLeases:output_type -> waves.node.grpc.TransactionResponse
	4,  // 11: waves.node.grpc.AccountsApi.GetDataEntries:output_type -> waves.node.grpc.DataEntryResponse
	11, // 12: waves.node.grpc.AccountsApi.ResolveAlias:output_type -> google.protobuf.BytesValue
	8,  // [8:13] is the sub-list for method output_type
	3,  // [3:8] is the sub-list for method input_type
	3,  // [3:3] is the sub-list for extension type_name
	3,  // [3:3] is the sub-list for extension extendee
	0,  // [0:3] is the sub-list for field type_name
}

func init() { file_waves_node_grpc_accounts_api_proto_init() }
func file_waves_node_grpc_accounts_api_proto_init() {
	if File_waves_node_grpc_accounts_api_proto != nil {
		return
	}
	file_waves_node_grpc_transactions_api_proto_init()
	if !protoimpl.UnsafeEnabled {
		file_waves_node_grpc_accounts_api_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*AccountRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_waves_node_grpc_accounts_api_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*DataRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_waves_node_grpc_accounts_api_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*BalancesRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_waves_node_grpc_accounts_api_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*BalanceResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_waves_node_grpc_accounts_api_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*DataEntryResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_waves_node_grpc_accounts_api_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ScriptData); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_waves_node_grpc_accounts_api_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*BalanceResponse_WavesBalances); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	file_waves_node_grpc_accounts_api_proto_msgTypes[3].OneofWrappers = []interface{}{
		(*BalanceResponse_Waves)(nil),
		(*BalanceResponse_Asset)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_waves_node_grpc_accounts_api_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   7,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_waves_node_grpc_accounts_api_proto_goTypes,
		DependencyIndexes: file_waves_node_grpc_accounts_api_proto_depIdxs,
		MessageInfos:      file_waves_node_grpc_accounts_api_proto_msgTypes,
	}.Build()
	File_waves_node_grpc_accounts_api_proto = out.File
	file_waves_node_grpc_accounts_api_proto_rawDesc = nil
	file_waves_node_grpc_accounts_api_proto_goTypes = nil
	file_waves_node_grpc_accounts_api_proto_depIdxs = nil
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConnInterface

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion6

// AccountsApiClient is the client API for AccountsApi service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type AccountsApiClient interface {
	GetBalances(ctx context.Context, in *BalancesRequest, opts ...grpc.CallOption) (AccountsApi_GetBalancesClient, error)
	GetScript(ctx context.Context, in *AccountRequest, opts ...grpc.CallOption) (*ScriptData, error)
	GetActiveLeases(ctx context.Context, in *AccountRequest, opts ...grpc.CallOption) (AccountsApi_GetActiveLeasesClient, error)
	GetDataEntries(ctx context.Context, in *DataRequest, opts ...grpc.CallOption) (AccountsApi_GetDataEntriesClient, error)
	ResolveAlias(ctx context.Context, in *wrapperspb.StringValue, opts ...grpc.CallOption) (*wrapperspb.BytesValue, error)
}

type accountsApiClient struct {
	cc grpc.ClientConnInterface
}

func NewAccountsApiClient(cc grpc.ClientConnInterface) AccountsApiClient {
	return &accountsApiClient{cc}
}

func (c *accountsApiClient) GetBalances(ctx context.Context, in *BalancesRequest, opts ...grpc.CallOption) (AccountsApi_GetBalancesClient, error) {
	stream, err := c.cc.NewStream(ctx, &_AccountsApi_serviceDesc.Streams[0], "/waves.node.grpc.AccountsApi/GetBalances", opts...)
	if err != nil {
		return nil, err
	}
	x := &accountsApiGetBalancesClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type AccountsApi_GetBalancesClient interface {
	Recv() (*BalanceResponse, error)
	grpc.ClientStream
}

type accountsApiGetBalancesClient struct {
	grpc.ClientStream
}

func (x *accountsApiGetBalancesClient) Recv() (*BalanceResponse, error) {
	m := new(BalanceResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *accountsApiClient) GetScript(ctx context.Context, in *AccountRequest, opts ...grpc.CallOption) (*ScriptData, error) {
	out := new(ScriptData)
	err := c.cc.Invoke(ctx, "/waves.node.grpc.AccountsApi/GetScript", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *accountsApiClient) GetActiveLeases(ctx context.Context, in *AccountRequest, opts ...grpc.CallOption) (AccountsApi_GetActiveLeasesClient, error) {
	stream, err := c.cc.NewStream(ctx, &_AccountsApi_serviceDesc.Streams[1], "/waves.node.grpc.AccountsApi/GetActiveLeases", opts...)
	if err != nil {
		return nil, err
	}
	x := &accountsApiGetActiveLeasesClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type AccountsApi_GetActiveLeasesClient interface {
	Recv() (*TransactionResponse, error)
	grpc.ClientStream
}

type accountsApiGetActiveLeasesClient struct {
	grpc.ClientStream
}

func (x *accountsApiGetActiveLeasesClient) Recv() (*TransactionResponse, error) {
	m := new(TransactionResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *accountsApiClient) GetDataEntries(ctx context.Context, in *DataRequest, opts ...grpc.CallOption) (AccountsApi_GetDataEntriesClient, error) {
	stream, err := c.cc.NewStream(ctx, &_AccountsApi_serviceDesc.Streams[2], "/waves.node.grpc.AccountsApi/GetDataEntries", opts...)
	if err != nil {
		return nil, err
	}
	x := &accountsApiGetDataEntriesClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type AccountsApi_GetDataEntriesClient interface {
	Recv() (*DataEntryResponse, error)
	grpc.ClientStream
}

type accountsApiGetDataEntriesClient struct {
	grpc.ClientStream
}

func (x *accountsApiGetDataEntriesClient) Recv() (*DataEntryResponse, error) {
	m := new(DataEntryResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *accountsApiClient) ResolveAlias(ctx context.Context, in *wrapperspb.StringValue, opts ...grpc.CallOption) (*wrapperspb.BytesValue, error) {
	out := new(wrapperspb.BytesValue)
	err := c.cc.Invoke(ctx, "/waves.node.grpc.AccountsApi/ResolveAlias", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// AccountsApiServer is the server API for AccountsApi service.
type AccountsApiServer interface {
	GetBalances(*BalancesRequest, AccountsApi_GetBalancesServer) error
	GetScript(context.Context, *AccountRequest) (*ScriptData, error)
	GetActiveLeases(*AccountRequest, AccountsApi_GetActiveLeasesServer) error
	GetDataEntries(*DataRequest, AccountsApi_GetDataEntriesServer) error
	ResolveAlias(context.Context, *wrapperspb.StringValue) (*wrapperspb.BytesValue, error)
}

// UnimplementedAccountsApiServer can be embedded to have forward compatible implementations.
type UnimplementedAccountsApiServer struct {
}

func (*UnimplementedAccountsApiServer) GetBalances(*BalancesRequest, AccountsApi_GetBalancesServer) error {
	return status.Errorf(codes.Unimplemented, "method GetBalances not implemented")
}
func (*UnimplementedAccountsApiServer) GetScript(context.Context, *AccountRequest) (*ScriptData, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetScript not implemented")
}
func (*UnimplementedAccountsApiServer) GetActiveLeases(*AccountRequest, AccountsApi_GetActiveLeasesServer) error {
	return status.Errorf(codes.Unimplemented, "method GetActiveLeases not implemented")
}
func (*UnimplementedAccountsApiServer) GetDataEntries(*DataRequest, AccountsApi_GetDataEntriesServer) error {
	return status.Errorf(codes.Unimplemented, "method GetDataEntries not implemented")
}
func (*UnimplementedAccountsApiServer) ResolveAlias(context.Context, *wrapperspb.StringValue) (*wrapperspb.BytesValue, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ResolveAlias not implemented")
}

func RegisterAccountsApiServer(s *grpc.Server, srv AccountsApiServer) {
	s.RegisterService(&_AccountsApi_serviceDesc, srv)
}

func _AccountsApi_GetBalances_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(BalancesRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(AccountsApiServer).GetBalances(m, &accountsApiGetBalancesServer{stream})
}

type AccountsApi_GetBalancesServer interface {
	Send(*BalanceResponse) error
	grpc.ServerStream
}

type accountsApiGetBalancesServer struct {
	grpc.ServerStream
}

func (x *accountsApiGetBalancesServer) Send(m *BalanceResponse) error {
	return x.ServerStream.SendMsg(m)
}

func _AccountsApi_GetScript_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AccountRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AccountsApiServer).GetScript(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/waves.node.grpc.AccountsApi/GetScript",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AccountsApiServer).GetScript(ctx, req.(*AccountRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _AccountsApi_GetActiveLeases_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(AccountRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(AccountsApiServer).GetActiveLeases(m, &accountsApiGetActiveLeasesServer{stream})
}

type AccountsApi_GetActiveLeasesServer interface {
	Send(*TransactionResponse) error
	grpc.ServerStream
}

type accountsApiGetActiveLeasesServer struct {
	grpc.ServerStream
}

func (x *accountsApiGetActiveLeasesServer) Send(m *TransactionResponse) error {
	return x.ServerStream.SendMsg(m)
}

func _AccountsApi_GetDataEntries_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(DataRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(AccountsApiServer).GetDataEntries(m, &accountsApiGetDataEntriesServer{stream})
}

type AccountsApi_GetDataEntriesServer interface {
	Send(*DataEntryResponse) error
	grpc.ServerStream
}

type accountsApiGetDataEntriesServer struct {
	grpc.ServerStream
}

func (x *accountsApiGetDataEntriesServer) Send(m *DataEntryResponse) error {
	return x.ServerStream.SendMsg(m)
}

func _AccountsApi_ResolveAlias_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(wrapperspb.StringValue)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AccountsApiServer).ResolveAlias(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/waves.node.grpc.AccountsApi/ResolveAlias",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AccountsApiServer).ResolveAlias(ctx, req.(*wrapperspb.StringValue))
	}
	return interceptor(ctx, in, info, handler)
}

var _AccountsApi_serviceDesc = grpc.ServiceDesc{
	ServiceName: "waves.node.grpc.AccountsApi",
	HandlerType: (*AccountsApiServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetScript",
			Handler:    _AccountsApi_GetScript_Handler,
		},
		{
			MethodName: "ResolveAlias",
			Handler:    _AccountsApi_ResolveAlias_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "GetBalances",
			Handler:       _AccountsApi_GetBalances_Handler,
			ServerStreams: true,
		},
		{
			StreamName:    "GetActiveLeases",
			Handler:       _AccountsApi_GetActiveLeases_Handler,
			ServerStreams: true,
		},
		{
			StreamName:    "GetDataEntries",
			Handler:       _AccountsApi_GetDataEntries_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "waves/node/grpc/accounts_api.proto",
}
