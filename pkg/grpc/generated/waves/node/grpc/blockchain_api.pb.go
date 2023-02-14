// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.27.1
// 	protoc        v3.21.12
// source: waves/node/grpc/blockchain_api.proto

package grpc

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type FeatureActivationStatus_BlockchainFeatureStatus int32

const (
	FeatureActivationStatus_UNDEFINED FeatureActivationStatus_BlockchainFeatureStatus = 0
	FeatureActivationStatus_APPROVED  FeatureActivationStatus_BlockchainFeatureStatus = 1
	FeatureActivationStatus_ACTIVATED FeatureActivationStatus_BlockchainFeatureStatus = 2
)

// Enum value maps for FeatureActivationStatus_BlockchainFeatureStatus.
var (
	FeatureActivationStatus_BlockchainFeatureStatus_name = map[int32]string{
		0: "UNDEFINED",
		1: "APPROVED",
		2: "ACTIVATED",
	}
	FeatureActivationStatus_BlockchainFeatureStatus_value = map[string]int32{
		"UNDEFINED": 0,
		"APPROVED":  1,
		"ACTIVATED": 2,
	}
)

func (x FeatureActivationStatus_BlockchainFeatureStatus) Enum() *FeatureActivationStatus_BlockchainFeatureStatus {
	p := new(FeatureActivationStatus_BlockchainFeatureStatus)
	*p = x
	return p
}

func (x FeatureActivationStatus_BlockchainFeatureStatus) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (FeatureActivationStatus_BlockchainFeatureStatus) Descriptor() protoreflect.EnumDescriptor {
	return file_waves_node_grpc_blockchain_api_proto_enumTypes[0].Descriptor()
}

func (FeatureActivationStatus_BlockchainFeatureStatus) Type() protoreflect.EnumType {
	return &file_waves_node_grpc_blockchain_api_proto_enumTypes[0]
}

func (x FeatureActivationStatus_BlockchainFeatureStatus) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use FeatureActivationStatus_BlockchainFeatureStatus.Descriptor instead.
func (FeatureActivationStatus_BlockchainFeatureStatus) EnumDescriptor() ([]byte, []int) {
	return file_waves_node_grpc_blockchain_api_proto_rawDescGZIP(), []int{2, 0}
}

type FeatureActivationStatus_NodeFeatureStatus int32

const (
	FeatureActivationStatus_NOT_IMPLEMENTED FeatureActivationStatus_NodeFeatureStatus = 0
	FeatureActivationStatus_IMPLEMENTED     FeatureActivationStatus_NodeFeatureStatus = 1
	FeatureActivationStatus_VOTED           FeatureActivationStatus_NodeFeatureStatus = 2
)

// Enum value maps for FeatureActivationStatus_NodeFeatureStatus.
var (
	FeatureActivationStatus_NodeFeatureStatus_name = map[int32]string{
		0: "NOT_IMPLEMENTED",
		1: "IMPLEMENTED",
		2: "VOTED",
	}
	FeatureActivationStatus_NodeFeatureStatus_value = map[string]int32{
		"NOT_IMPLEMENTED": 0,
		"IMPLEMENTED":     1,
		"VOTED":           2,
	}
)

func (x FeatureActivationStatus_NodeFeatureStatus) Enum() *FeatureActivationStatus_NodeFeatureStatus {
	p := new(FeatureActivationStatus_NodeFeatureStatus)
	*p = x
	return p
}

func (x FeatureActivationStatus_NodeFeatureStatus) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (FeatureActivationStatus_NodeFeatureStatus) Descriptor() protoreflect.EnumDescriptor {
	return file_waves_node_grpc_blockchain_api_proto_enumTypes[1].Descriptor()
}

func (FeatureActivationStatus_NodeFeatureStatus) Type() protoreflect.EnumType {
	return &file_waves_node_grpc_blockchain_api_proto_enumTypes[1]
}

func (x FeatureActivationStatus_NodeFeatureStatus) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use FeatureActivationStatus_NodeFeatureStatus.Descriptor instead.
func (FeatureActivationStatus_NodeFeatureStatus) EnumDescriptor() ([]byte, []int) {
	return file_waves_node_grpc_blockchain_api_proto_rawDescGZIP(), []int{2, 1}
}

type ActivationStatusRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Height int32 `protobuf:"varint,1,opt,name=height,proto3" json:"height,omitempty"`
}

func (x *ActivationStatusRequest) Reset() {
	*x = ActivationStatusRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_waves_node_grpc_blockchain_api_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ActivationStatusRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ActivationStatusRequest) ProtoMessage() {}

func (x *ActivationStatusRequest) ProtoReflect() protoreflect.Message {
	mi := &file_waves_node_grpc_blockchain_api_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ActivationStatusRequest.ProtoReflect.Descriptor instead.
func (*ActivationStatusRequest) Descriptor() ([]byte, []int) {
	return file_waves_node_grpc_blockchain_api_proto_rawDescGZIP(), []int{0}
}

func (x *ActivationStatusRequest) GetHeight() int32 {
	if x != nil {
		return x.Height
	}
	return 0
}

type ActivationStatusResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Height          int32                      `protobuf:"varint,1,opt,name=height,proto3" json:"height,omitempty"`
	VotingInterval  int32                      `protobuf:"varint,2,opt,name=voting_interval,json=votingInterval,proto3" json:"voting_interval,omitempty"`
	VotingThreshold int32                      `protobuf:"varint,3,opt,name=voting_threshold,json=votingThreshold,proto3" json:"voting_threshold,omitempty"`
	NextCheck       int32                      `protobuf:"varint,4,opt,name=next_check,json=nextCheck,proto3" json:"next_check,omitempty"`
	Features        []*FeatureActivationStatus `protobuf:"bytes,5,rep,name=features,proto3" json:"features,omitempty"`
}

func (x *ActivationStatusResponse) Reset() {
	*x = ActivationStatusResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_waves_node_grpc_blockchain_api_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ActivationStatusResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ActivationStatusResponse) ProtoMessage() {}

func (x *ActivationStatusResponse) ProtoReflect() protoreflect.Message {
	mi := &file_waves_node_grpc_blockchain_api_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ActivationStatusResponse.ProtoReflect.Descriptor instead.
func (*ActivationStatusResponse) Descriptor() ([]byte, []int) {
	return file_waves_node_grpc_blockchain_api_proto_rawDescGZIP(), []int{1}
}

func (x *ActivationStatusResponse) GetHeight() int32 {
	if x != nil {
		return x.Height
	}
	return 0
}

func (x *ActivationStatusResponse) GetVotingInterval() int32 {
	if x != nil {
		return x.VotingInterval
	}
	return 0
}

func (x *ActivationStatusResponse) GetVotingThreshold() int32 {
	if x != nil {
		return x.VotingThreshold
	}
	return 0
}

func (x *ActivationStatusResponse) GetNextCheck() int32 {
	if x != nil {
		return x.NextCheck
	}
	return 0
}

func (x *ActivationStatusResponse) GetFeatures() []*FeatureActivationStatus {
	if x != nil {
		return x.Features
	}
	return nil
}

type FeatureActivationStatus struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id               int32                                           `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty"`
	Description      string                                          `protobuf:"bytes,2,opt,name=description,proto3" json:"description,omitempty"`
	BlockchainStatus FeatureActivationStatus_BlockchainFeatureStatus `protobuf:"varint,3,opt,name=blockchain_status,json=blockchainStatus,proto3,enum=waves.node.grpc.FeatureActivationStatus_BlockchainFeatureStatus" json:"blockchain_status,omitempty"`
	NodeStatus       FeatureActivationStatus_NodeFeatureStatus       `protobuf:"varint,4,opt,name=node_status,json=nodeStatus,proto3,enum=waves.node.grpc.FeatureActivationStatus_NodeFeatureStatus" json:"node_status,omitempty"`
	ActivationHeight int32                                           `protobuf:"varint,5,opt,name=activation_height,json=activationHeight,proto3" json:"activation_height,omitempty"`
	SupportingBlocks int32                                           `protobuf:"varint,6,opt,name=supporting_blocks,json=supportingBlocks,proto3" json:"supporting_blocks,omitempty"`
}

func (x *FeatureActivationStatus) Reset() {
	*x = FeatureActivationStatus{}
	if protoimpl.UnsafeEnabled {
		mi := &file_waves_node_grpc_blockchain_api_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *FeatureActivationStatus) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*FeatureActivationStatus) ProtoMessage() {}

func (x *FeatureActivationStatus) ProtoReflect() protoreflect.Message {
	mi := &file_waves_node_grpc_blockchain_api_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use FeatureActivationStatus.ProtoReflect.Descriptor instead.
func (*FeatureActivationStatus) Descriptor() ([]byte, []int) {
	return file_waves_node_grpc_blockchain_api_proto_rawDescGZIP(), []int{2}
}

func (x *FeatureActivationStatus) GetId() int32 {
	if x != nil {
		return x.Id
	}
	return 0
}

func (x *FeatureActivationStatus) GetDescription() string {
	if x != nil {
		return x.Description
	}
	return ""
}

func (x *FeatureActivationStatus) GetBlockchainStatus() FeatureActivationStatus_BlockchainFeatureStatus {
	if x != nil {
		return x.BlockchainStatus
	}
	return FeatureActivationStatus_UNDEFINED
}

func (x *FeatureActivationStatus) GetNodeStatus() FeatureActivationStatus_NodeFeatureStatus {
	if x != nil {
		return x.NodeStatus
	}
	return FeatureActivationStatus_NOT_IMPLEMENTED
}

func (x *FeatureActivationStatus) GetActivationHeight() int32 {
	if x != nil {
		return x.ActivationHeight
	}
	return 0
}

func (x *FeatureActivationStatus) GetSupportingBlocks() int32 {
	if x != nil {
		return x.SupportingBlocks
	}
	return 0
}

type BaseTargetResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	BaseTarget int64 `protobuf:"varint,1,opt,name=base_target,json=baseTarget,proto3" json:"base_target,omitempty"`
}

func (x *BaseTargetResponse) Reset() {
	*x = BaseTargetResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_waves_node_grpc_blockchain_api_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *BaseTargetResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*BaseTargetResponse) ProtoMessage() {}

func (x *BaseTargetResponse) ProtoReflect() protoreflect.Message {
	mi := &file_waves_node_grpc_blockchain_api_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use BaseTargetResponse.ProtoReflect.Descriptor instead.
func (*BaseTargetResponse) Descriptor() ([]byte, []int) {
	return file_waves_node_grpc_blockchain_api_proto_rawDescGZIP(), []int{3}
}

func (x *BaseTargetResponse) GetBaseTarget() int64 {
	if x != nil {
		return x.BaseTarget
	}
	return 0
}

type ScoreResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Score []byte `protobuf:"bytes,1,opt,name=score,proto3" json:"score,omitempty"` // BigInt
}

func (x *ScoreResponse) Reset() {
	*x = ScoreResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_waves_node_grpc_blockchain_api_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ScoreResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ScoreResponse) ProtoMessage() {}

func (x *ScoreResponse) ProtoReflect() protoreflect.Message {
	mi := &file_waves_node_grpc_blockchain_api_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ScoreResponse.ProtoReflect.Descriptor instead.
func (*ScoreResponse) Descriptor() ([]byte, []int) {
	return file_waves_node_grpc_blockchain_api_proto_rawDescGZIP(), []int{4}
}

func (x *ScoreResponse) GetScore() []byte {
	if x != nil {
		return x.Score
	}
	return nil
}

var File_waves_node_grpc_blockchain_api_proto protoreflect.FileDescriptor

var file_waves_node_grpc_blockchain_api_proto_rawDesc = []byte{
	0x0a, 0x24, 0x77, 0x61, 0x76, 0x65, 0x73, 0x2f, 0x6e, 0x6f, 0x64, 0x65, 0x2f, 0x67, 0x72, 0x70,
	0x63, 0x2f, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x63, 0x68, 0x61, 0x69, 0x6e, 0x5f, 0x61, 0x70, 0x69,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0f, 0x77, 0x61, 0x76, 0x65, 0x73, 0x2e, 0x6e, 0x6f,
	0x64, 0x65, 0x2e, 0x67, 0x72, 0x70, 0x63, 0x1a, 0x1b, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x65, 0x6d, 0x70, 0x74, 0x79, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x22, 0x31, 0x0a, 0x17, 0x41, 0x63, 0x74, 0x69, 0x76, 0x61, 0x74, 0x69,
	0x6f, 0x6e, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12,
	0x16, 0x0a, 0x06, 0x68, 0x65, 0x69, 0x67, 0x68, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x05, 0x52,
	0x06, 0x68, 0x65, 0x69, 0x67, 0x68, 0x74, 0x22, 0xeb, 0x01, 0x0a, 0x18, 0x41, 0x63, 0x74, 0x69,
	0x76, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x52, 0x65, 0x73, 0x70,
	0x6f, 0x6e, 0x73, 0x65, 0x12, 0x16, 0x0a, 0x06, 0x68, 0x65, 0x69, 0x67, 0x68, 0x74, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x05, 0x52, 0x06, 0x68, 0x65, 0x69, 0x67, 0x68, 0x74, 0x12, 0x27, 0x0a, 0x0f,
	0x76, 0x6f, 0x74, 0x69, 0x6e, 0x67, 0x5f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x76, 0x61, 0x6c, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x05, 0x52, 0x0e, 0x76, 0x6f, 0x74, 0x69, 0x6e, 0x67, 0x49, 0x6e, 0x74,
	0x65, 0x72, 0x76, 0x61, 0x6c, 0x12, 0x29, 0x0a, 0x10, 0x76, 0x6f, 0x74, 0x69, 0x6e, 0x67, 0x5f,
	0x74, 0x68, 0x72, 0x65, 0x73, 0x68, 0x6f, 0x6c, 0x64, 0x18, 0x03, 0x20, 0x01, 0x28, 0x05, 0x52,
	0x0f, 0x76, 0x6f, 0x74, 0x69, 0x6e, 0x67, 0x54, 0x68, 0x72, 0x65, 0x73, 0x68, 0x6f, 0x6c, 0x64,
	0x12, 0x1d, 0x0a, 0x0a, 0x6e, 0x65, 0x78, 0x74, 0x5f, 0x63, 0x68, 0x65, 0x63, 0x6b, 0x18, 0x04,
	0x20, 0x01, 0x28, 0x05, 0x52, 0x09, 0x6e, 0x65, 0x78, 0x74, 0x43, 0x68, 0x65, 0x63, 0x6b, 0x12,
	0x44, 0x0a, 0x08, 0x66, 0x65, 0x61, 0x74, 0x75, 0x72, 0x65, 0x73, 0x18, 0x05, 0x20, 0x03, 0x28,
	0x0b, 0x32, 0x28, 0x2e, 0x77, 0x61, 0x76, 0x65, 0x73, 0x2e, 0x6e, 0x6f, 0x64, 0x65, 0x2e, 0x67,
	0x72, 0x70, 0x63, 0x2e, 0x46, 0x65, 0x61, 0x74, 0x75, 0x72, 0x65, 0x41, 0x63, 0x74, 0x69, 0x76,
	0x61, 0x74, 0x69, 0x6f, 0x6e, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x52, 0x08, 0x66, 0x65, 0x61,
	0x74, 0x75, 0x72, 0x65, 0x73, 0x22, 0xfe, 0x03, 0x0a, 0x17, 0x46, 0x65, 0x61, 0x74, 0x75, 0x72,
	0x65, 0x41, 0x63, 0x74, 0x69, 0x76, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x53, 0x74, 0x61, 0x74, 0x75,
	0x73, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x05, 0x52, 0x02, 0x69,
	0x64, 0x12, 0x20, 0x0a, 0x0b, 0x64, 0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x69, 0x6f, 0x6e,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0b, 0x64, 0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74,
	0x69, 0x6f, 0x6e, 0x12, 0x6d, 0x0a, 0x11, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x63, 0x68, 0x61, 0x69,
	0x6e, 0x5f, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x40,
	0x2e, 0x77, 0x61, 0x76, 0x65, 0x73, 0x2e, 0x6e, 0x6f, 0x64, 0x65, 0x2e, 0x67, 0x72, 0x70, 0x63,
	0x2e, 0x46, 0x65, 0x61, 0x74, 0x75, 0x72, 0x65, 0x41, 0x63, 0x74, 0x69, 0x76, 0x61, 0x74, 0x69,
	0x6f, 0x6e, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x2e, 0x42, 0x6c, 0x6f, 0x63, 0x6b, 0x63, 0x68,
	0x61, 0x69, 0x6e, 0x46, 0x65, 0x61, 0x74, 0x75, 0x72, 0x65, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73,
	0x52, 0x10, 0x62, 0x6c, 0x6f, 0x63, 0x6b, 0x63, 0x68, 0x61, 0x69, 0x6e, 0x53, 0x74, 0x61, 0x74,
	0x75, 0x73, 0x12, 0x5b, 0x0a, 0x0b, 0x6e, 0x6f, 0x64, 0x65, 0x5f, 0x73, 0x74, 0x61, 0x74, 0x75,
	0x73, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x3a, 0x2e, 0x77, 0x61, 0x76, 0x65, 0x73, 0x2e,
	0x6e, 0x6f, 0x64, 0x65, 0x2e, 0x67, 0x72, 0x70, 0x63, 0x2e, 0x46, 0x65, 0x61, 0x74, 0x75, 0x72,
	0x65, 0x41, 0x63, 0x74, 0x69, 0x76, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x53, 0x74, 0x61, 0x74, 0x75,
	0x73, 0x2e, 0x4e, 0x6f, 0x64, 0x65, 0x46, 0x65, 0x61, 0x74, 0x75, 0x72, 0x65, 0x53, 0x74, 0x61,
	0x74, 0x75, 0x73, 0x52, 0x0a, 0x6e, 0x6f, 0x64, 0x65, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x12,
	0x2b, 0x0a, 0x11, 0x61, 0x63, 0x74, 0x69, 0x76, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x5f, 0x68, 0x65,
	0x69, 0x67, 0x68, 0x74, 0x18, 0x05, 0x20, 0x01, 0x28, 0x05, 0x52, 0x10, 0x61, 0x63, 0x74, 0x69,
	0x76, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x48, 0x65, 0x69, 0x67, 0x68, 0x74, 0x12, 0x2b, 0x0a, 0x11,
	0x73, 0x75, 0x70, 0x70, 0x6f, 0x72, 0x74, 0x69, 0x6e, 0x67, 0x5f, 0x62, 0x6c, 0x6f, 0x63, 0x6b,
	0x73, 0x18, 0x06, 0x20, 0x01, 0x28, 0x05, 0x52, 0x10, 0x73, 0x75, 0x70, 0x70, 0x6f, 0x72, 0x74,
	0x69, 0x6e, 0x67, 0x42, 0x6c, 0x6f, 0x63, 0x6b, 0x73, 0x22, 0x45, 0x0a, 0x17, 0x42, 0x6c, 0x6f,
	0x63, 0x6b, 0x63, 0x68, 0x61, 0x69, 0x6e, 0x46, 0x65, 0x61, 0x74, 0x75, 0x72, 0x65, 0x53, 0x74,
	0x61, 0x74, 0x75, 0x73, 0x12, 0x0d, 0x0a, 0x09, 0x55, 0x4e, 0x44, 0x45, 0x46, 0x49, 0x4e, 0x45,
	0x44, 0x10, 0x00, 0x12, 0x0c, 0x0a, 0x08, 0x41, 0x50, 0x50, 0x52, 0x4f, 0x56, 0x45, 0x44, 0x10,
	0x01, 0x12, 0x0d, 0x0a, 0x09, 0x41, 0x43, 0x54, 0x49, 0x56, 0x41, 0x54, 0x45, 0x44, 0x10, 0x02,
	0x22, 0x44, 0x0a, 0x11, 0x4e, 0x6f, 0x64, 0x65, 0x46, 0x65, 0x61, 0x74, 0x75, 0x72, 0x65, 0x53,
	0x74, 0x61, 0x74, 0x75, 0x73, 0x12, 0x13, 0x0a, 0x0f, 0x4e, 0x4f, 0x54, 0x5f, 0x49, 0x4d, 0x50,
	0x4c, 0x45, 0x4d, 0x45, 0x4e, 0x54, 0x45, 0x44, 0x10, 0x00, 0x12, 0x0f, 0x0a, 0x0b, 0x49, 0x4d,
	0x50, 0x4c, 0x45, 0x4d, 0x45, 0x4e, 0x54, 0x45, 0x44, 0x10, 0x01, 0x12, 0x09, 0x0a, 0x05, 0x56,
	0x4f, 0x54, 0x45, 0x44, 0x10, 0x02, 0x22, 0x35, 0x0a, 0x12, 0x42, 0x61, 0x73, 0x65, 0x54, 0x61,
	0x72, 0x67, 0x65, 0x74, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x1f, 0x0a, 0x0b,
	0x62, 0x61, 0x73, 0x65, 0x5f, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x03, 0x52, 0x0a, 0x62, 0x61, 0x73, 0x65, 0x54, 0x61, 0x72, 0x67, 0x65, 0x74, 0x22, 0x25, 0x0a,
	0x0d, 0x53, 0x63, 0x6f, 0x72, 0x65, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x14,
	0x0a, 0x05, 0x73, 0x63, 0x6f, 0x72, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x05, 0x73,
	0x63, 0x6f, 0x72, 0x65, 0x32, 0x97, 0x02, 0x0a, 0x0d, 0x42, 0x6c, 0x6f, 0x63, 0x6b, 0x63, 0x68,
	0x61, 0x69, 0x6e, 0x41, 0x70, 0x69, 0x12, 0x6a, 0x0a, 0x13, 0x47, 0x65, 0x74, 0x41, 0x63, 0x74,
	0x69, 0x76, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x12, 0x28, 0x2e,
	0x77, 0x61, 0x76, 0x65, 0x73, 0x2e, 0x6e, 0x6f, 0x64, 0x65, 0x2e, 0x67, 0x72, 0x70, 0x63, 0x2e,
	0x41, 0x63, 0x74, 0x69, 0x76, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73,
	0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x29, 0x2e, 0x77, 0x61, 0x76, 0x65, 0x73, 0x2e,
	0x6e, 0x6f, 0x64, 0x65, 0x2e, 0x67, 0x72, 0x70, 0x63, 0x2e, 0x41, 0x63, 0x74, 0x69, 0x76, 0x61,
	0x74, 0x69, 0x6f, 0x6e, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e,
	0x73, 0x65, 0x12, 0x4c, 0x0a, 0x0d, 0x47, 0x65, 0x74, 0x42, 0x61, 0x73, 0x65, 0x54, 0x61, 0x72,
	0x67, 0x65, 0x74, 0x12, 0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x1a, 0x23, 0x2e, 0x77, 0x61,
	0x76, 0x65, 0x73, 0x2e, 0x6e, 0x6f, 0x64, 0x65, 0x2e, 0x67, 0x72, 0x70, 0x63, 0x2e, 0x42, 0x61,
	0x73, 0x65, 0x54, 0x61, 0x72, 0x67, 0x65, 0x74, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65,
	0x12, 0x4c, 0x0a, 0x12, 0x47, 0x65, 0x74, 0x43, 0x75, 0x6d, 0x75, 0x6c, 0x61, 0x74, 0x69, 0x76,
	0x65, 0x53, 0x63, 0x6f, 0x72, 0x65, 0x12, 0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x1a, 0x1e,
	0x2e, 0x77, 0x61, 0x76, 0x65, 0x73, 0x2e, 0x6e, 0x6f, 0x64, 0x65, 0x2e, 0x67, 0x72, 0x70, 0x63,
	0x2e, 0x53, 0x63, 0x6f, 0x72, 0x65, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x42, 0x73,
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
	file_waves_node_grpc_blockchain_api_proto_rawDescOnce sync.Once
	file_waves_node_grpc_blockchain_api_proto_rawDescData = file_waves_node_grpc_blockchain_api_proto_rawDesc
)

func file_waves_node_grpc_blockchain_api_proto_rawDescGZIP() []byte {
	file_waves_node_grpc_blockchain_api_proto_rawDescOnce.Do(func() {
		file_waves_node_grpc_blockchain_api_proto_rawDescData = protoimpl.X.CompressGZIP(file_waves_node_grpc_blockchain_api_proto_rawDescData)
	})
	return file_waves_node_grpc_blockchain_api_proto_rawDescData
}

var file_waves_node_grpc_blockchain_api_proto_enumTypes = make([]protoimpl.EnumInfo, 2)
var file_waves_node_grpc_blockchain_api_proto_msgTypes = make([]protoimpl.MessageInfo, 5)
var file_waves_node_grpc_blockchain_api_proto_goTypes = []interface{}{
	(FeatureActivationStatus_BlockchainFeatureStatus)(0), // 0: waves.node.grpc.FeatureActivationStatus.BlockchainFeatureStatus
	(FeatureActivationStatus_NodeFeatureStatus)(0),       // 1: waves.node.grpc.FeatureActivationStatus.NodeFeatureStatus
	(*ActivationStatusRequest)(nil),                      // 2: waves.node.grpc.ActivationStatusRequest
	(*ActivationStatusResponse)(nil),                     // 3: waves.node.grpc.ActivationStatusResponse
	(*FeatureActivationStatus)(nil),                      // 4: waves.node.grpc.FeatureActivationStatus
	(*BaseTargetResponse)(nil),                           // 5: waves.node.grpc.BaseTargetResponse
	(*ScoreResponse)(nil),                                // 6: waves.node.grpc.ScoreResponse
	(*emptypb.Empty)(nil),                                // 7: google.protobuf.Empty
}
var file_waves_node_grpc_blockchain_api_proto_depIdxs = []int32{
	4, // 0: waves.node.grpc.ActivationStatusResponse.features:type_name -> waves.node.grpc.FeatureActivationStatus
	0, // 1: waves.node.grpc.FeatureActivationStatus.blockchain_status:type_name -> waves.node.grpc.FeatureActivationStatus.BlockchainFeatureStatus
	1, // 2: waves.node.grpc.FeatureActivationStatus.node_status:type_name -> waves.node.grpc.FeatureActivationStatus.NodeFeatureStatus
	2, // 3: waves.node.grpc.BlockchainApi.GetActivationStatus:input_type -> waves.node.grpc.ActivationStatusRequest
	7, // 4: waves.node.grpc.BlockchainApi.GetBaseTarget:input_type -> google.protobuf.Empty
	7, // 5: waves.node.grpc.BlockchainApi.GetCumulativeScore:input_type -> google.protobuf.Empty
	3, // 6: waves.node.grpc.BlockchainApi.GetActivationStatus:output_type -> waves.node.grpc.ActivationStatusResponse
	5, // 7: waves.node.grpc.BlockchainApi.GetBaseTarget:output_type -> waves.node.grpc.BaseTargetResponse
	6, // 8: waves.node.grpc.BlockchainApi.GetCumulativeScore:output_type -> waves.node.grpc.ScoreResponse
	6, // [6:9] is the sub-list for method output_type
	3, // [3:6] is the sub-list for method input_type
	3, // [3:3] is the sub-list for extension type_name
	3, // [3:3] is the sub-list for extension extendee
	0, // [0:3] is the sub-list for field type_name
}

func init() { file_waves_node_grpc_blockchain_api_proto_init() }
func file_waves_node_grpc_blockchain_api_proto_init() {
	if File_waves_node_grpc_blockchain_api_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_waves_node_grpc_blockchain_api_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ActivationStatusRequest); i {
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
		file_waves_node_grpc_blockchain_api_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ActivationStatusResponse); i {
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
		file_waves_node_grpc_blockchain_api_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*FeatureActivationStatus); i {
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
		file_waves_node_grpc_blockchain_api_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*BaseTargetResponse); i {
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
		file_waves_node_grpc_blockchain_api_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ScoreResponse); i {
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
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_waves_node_grpc_blockchain_api_proto_rawDesc,
			NumEnums:      2,
			NumMessages:   5,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_waves_node_grpc_blockchain_api_proto_goTypes,
		DependencyIndexes: file_waves_node_grpc_blockchain_api_proto_depIdxs,
		EnumInfos:         file_waves_node_grpc_blockchain_api_proto_enumTypes,
		MessageInfos:      file_waves_node_grpc_blockchain_api_proto_msgTypes,
	}.Build()
	File_waves_node_grpc_blockchain_api_proto = out.File
	file_waves_node_grpc_blockchain_api_proto_rawDesc = nil
	file_waves_node_grpc_blockchain_api_proto_goTypes = nil
	file_waves_node_grpc_blockchain_api_proto_depIdxs = nil
}
