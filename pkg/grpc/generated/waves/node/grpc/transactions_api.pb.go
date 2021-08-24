// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.25.0
// 	protoc        v3.17.3
// source: waves/node/grpc/transactions_api.proto

package grpc

import (
	proto "github.com/golang/protobuf/proto"
	waves "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
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

type ApplicationStatus int32

const (
	ApplicationStatus_UNKNOWN                 ApplicationStatus = 0
	ApplicationStatus_SUCCEEDED               ApplicationStatus = 1
	ApplicationStatus_SCRIPT_EXECUTION_FAILED ApplicationStatus = 2
)

// Enum value maps for ApplicationStatus.
var (
	ApplicationStatus_name = map[int32]string{
		0: "UNKNOWN",
		1: "SUCCEEDED",
		2: "SCRIPT_EXECUTION_FAILED",
	}
	ApplicationStatus_value = map[string]int32{
		"UNKNOWN":                 0,
		"SUCCEEDED":               1,
		"SCRIPT_EXECUTION_FAILED": 2,
	}
)

func (x ApplicationStatus) Enum() *ApplicationStatus {
	p := new(ApplicationStatus)
	*p = x
	return p
}

func (x ApplicationStatus) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (ApplicationStatus) Descriptor() protoreflect.EnumDescriptor {
	return file_waves_node_grpc_transactions_api_proto_enumTypes[0].Descriptor()
}

func (ApplicationStatus) Type() protoreflect.EnumType {
	return &file_waves_node_grpc_transactions_api_proto_enumTypes[0]
}

func (x ApplicationStatus) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use ApplicationStatus.Descriptor instead.
func (ApplicationStatus) EnumDescriptor() ([]byte, []int) {
	return file_waves_node_grpc_transactions_api_proto_rawDescGZIP(), []int{0}
}

type TransactionStatus_Status int32

const (
	TransactionStatus_NOT_EXISTS  TransactionStatus_Status = 0
	TransactionStatus_UNCONFIRMED TransactionStatus_Status = 1
	TransactionStatus_CONFIRMED   TransactionStatus_Status = 2
)

// Enum value maps for TransactionStatus_Status.
var (
	TransactionStatus_Status_name = map[int32]string{
		0: "NOT_EXISTS",
		1: "UNCONFIRMED",
		2: "CONFIRMED",
	}
	TransactionStatus_Status_value = map[string]int32{
		"NOT_EXISTS":  0,
		"UNCONFIRMED": 1,
		"CONFIRMED":   2,
	}
)

func (x TransactionStatus_Status) Enum() *TransactionStatus_Status {
	p := new(TransactionStatus_Status)
	*p = x
	return p
}

func (x TransactionStatus_Status) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (TransactionStatus_Status) Descriptor() protoreflect.EnumDescriptor {
	return file_waves_node_grpc_transactions_api_proto_enumTypes[1].Descriptor()
}

func (TransactionStatus_Status) Type() protoreflect.EnumType {
	return &file_waves_node_grpc_transactions_api_proto_enumTypes[1]
}

func (x TransactionStatus_Status) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use TransactionStatus_Status.Descriptor instead.
func (TransactionStatus_Status) EnumDescriptor() ([]byte, []int) {
	return file_waves_node_grpc_transactions_api_proto_rawDescGZIP(), []int{0, 0}
}

type TransactionStatus struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id                []byte                   `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Status            TransactionStatus_Status `protobuf:"varint,2,opt,name=status,proto3,enum=waves.node.grpc.TransactionStatus_Status" json:"status,omitempty"`
	Height            int64                    `protobuf:"varint,3,opt,name=height,proto3" json:"height,omitempty"`
	ApplicationStatus ApplicationStatus        `protobuf:"varint,4,opt,name=application_status,json=applicationStatus,proto3,enum=waves.node.grpc.ApplicationStatus" json:"application_status,omitempty"`
}

func (x *TransactionStatus) Reset() {
	*x = TransactionStatus{}
	if protoimpl.UnsafeEnabled {
		mi := &file_waves_node_grpc_transactions_api_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TransactionStatus) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TransactionStatus) ProtoMessage() {}

func (x *TransactionStatus) ProtoReflect() protoreflect.Message {
	mi := &file_waves_node_grpc_transactions_api_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TransactionStatus.ProtoReflect.Descriptor instead.
func (*TransactionStatus) Descriptor() ([]byte, []int) {
	return file_waves_node_grpc_transactions_api_proto_rawDescGZIP(), []int{0}
}

func (x *TransactionStatus) GetId() []byte {
	if x != nil {
		return x.Id
	}
	return nil
}

func (x *TransactionStatus) GetStatus() TransactionStatus_Status {
	if x != nil {
		return x.Status
	}
	return TransactionStatus_NOT_EXISTS
}

func (x *TransactionStatus) GetHeight() int64 {
	if x != nil {
		return x.Height
	}
	return 0
}

func (x *TransactionStatus) GetApplicationStatus() ApplicationStatus {
	if x != nil {
		return x.ApplicationStatus
	}
	return ApplicationStatus_UNKNOWN
}

type TransactionResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id                 []byte                    `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Height             int64                     `protobuf:"varint,2,opt,name=height,proto3" json:"height,omitempty"`
	Transaction        *waves.SignedTransaction  `protobuf:"bytes,3,opt,name=transaction,proto3" json:"transaction,omitempty"`
	ApplicationStatus  ApplicationStatus         `protobuf:"varint,4,opt,name=application_status,json=applicationStatus,proto3,enum=waves.node.grpc.ApplicationStatus" json:"application_status,omitempty"`
	InvokeScriptResult *waves.InvokeScriptResult `protobuf:"bytes,5,opt,name=invoke_script_result,json=invokeScriptResult,proto3" json:"invoke_script_result,omitempty"`
}

func (x *TransactionResponse) Reset() {
	*x = TransactionResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_waves_node_grpc_transactions_api_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TransactionResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TransactionResponse) ProtoMessage() {}

func (x *TransactionResponse) ProtoReflect() protoreflect.Message {
	mi := &file_waves_node_grpc_transactions_api_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TransactionResponse.ProtoReflect.Descriptor instead.
func (*TransactionResponse) Descriptor() ([]byte, []int) {
	return file_waves_node_grpc_transactions_api_proto_rawDescGZIP(), []int{1}
}

func (x *TransactionResponse) GetId() []byte {
	if x != nil {
		return x.Id
	}
	return nil
}

func (x *TransactionResponse) GetHeight() int64 {
	if x != nil {
		return x.Height
	}
	return 0
}

func (x *TransactionResponse) GetTransaction() *waves.SignedTransaction {
	if x != nil {
		return x.Transaction
	}
	return nil
}

func (x *TransactionResponse) GetApplicationStatus() ApplicationStatus {
	if x != nil {
		return x.ApplicationStatus
	}
	return ApplicationStatus_UNKNOWN
}

func (x *TransactionResponse) GetInvokeScriptResult() *waves.InvokeScriptResult {
	if x != nil {
		return x.InvokeScriptResult
	}
	return nil
}

type TransactionsRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Sender         []byte           `protobuf:"bytes,1,opt,name=sender,proto3" json:"sender,omitempty"`
	Recipient      *waves.Recipient `protobuf:"bytes,2,opt,name=recipient,proto3" json:"recipient,omitempty"`
	TransactionIds [][]byte         `protobuf:"bytes,3,rep,name=transaction_ids,json=transactionIds,proto3" json:"transaction_ids,omitempty"`
}

func (x *TransactionsRequest) Reset() {
	*x = TransactionsRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_waves_node_grpc_transactions_api_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TransactionsRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TransactionsRequest) ProtoMessage() {}

func (x *TransactionsRequest) ProtoReflect() protoreflect.Message {
	mi := &file_waves_node_grpc_transactions_api_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TransactionsRequest.ProtoReflect.Descriptor instead.
func (*TransactionsRequest) Descriptor() ([]byte, []int) {
	return file_waves_node_grpc_transactions_api_proto_rawDescGZIP(), []int{2}
}

func (x *TransactionsRequest) GetSender() []byte {
	if x != nil {
		return x.Sender
	}
	return nil
}

func (x *TransactionsRequest) GetRecipient() *waves.Recipient {
	if x != nil {
		return x.Recipient
	}
	return nil
}

func (x *TransactionsRequest) GetTransactionIds() [][]byte {
	if x != nil {
		return x.TransactionIds
	}
	return nil
}

type TransactionsByIdRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	TransactionIds [][]byte `protobuf:"bytes,3,rep,name=transaction_ids,json=transactionIds,proto3" json:"transaction_ids,omitempty"`
}

func (x *TransactionsByIdRequest) Reset() {
	*x = TransactionsByIdRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_waves_node_grpc_transactions_api_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TransactionsByIdRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TransactionsByIdRequest) ProtoMessage() {}

func (x *TransactionsByIdRequest) ProtoReflect() protoreflect.Message {
	mi := &file_waves_node_grpc_transactions_api_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TransactionsByIdRequest.ProtoReflect.Descriptor instead.
func (*TransactionsByIdRequest) Descriptor() ([]byte, []int) {
	return file_waves_node_grpc_transactions_api_proto_rawDescGZIP(), []int{3}
}

func (x *TransactionsByIdRequest) GetTransactionIds() [][]byte {
	if x != nil {
		return x.TransactionIds
	}
	return nil
}

type CalculateFeeResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	AssetId []byte `protobuf:"bytes,1,opt,name=asset_id,json=assetId,proto3" json:"asset_id,omitempty"`
	Amount  uint64 `protobuf:"varint,2,opt,name=amount,proto3" json:"amount,omitempty"`
}

func (x *CalculateFeeResponse) Reset() {
	*x = CalculateFeeResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_waves_node_grpc_transactions_api_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CalculateFeeResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CalculateFeeResponse) ProtoMessage() {}

func (x *CalculateFeeResponse) ProtoReflect() protoreflect.Message {
	mi := &file_waves_node_grpc_transactions_api_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CalculateFeeResponse.ProtoReflect.Descriptor instead.
func (*CalculateFeeResponse) Descriptor() ([]byte, []int) {
	return file_waves_node_grpc_transactions_api_proto_rawDescGZIP(), []int{4}
}

func (x *CalculateFeeResponse) GetAssetId() []byte {
	if x != nil {
		return x.AssetId
	}
	return nil
}

func (x *CalculateFeeResponse) GetAmount() uint64 {
	if x != nil {
		return x.Amount
	}
	return 0
}

type SignRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Transaction     *waves.Transaction `protobuf:"bytes,1,opt,name=transaction,proto3" json:"transaction,omitempty"`
	SignerPublicKey []byte             `protobuf:"bytes,2,opt,name=signer_public_key,json=signerPublicKey,proto3" json:"signer_public_key,omitempty"`
}

func (x *SignRequest) Reset() {
	*x = SignRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_waves_node_grpc_transactions_api_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *SignRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SignRequest) ProtoMessage() {}

func (x *SignRequest) ProtoReflect() protoreflect.Message {
	mi := &file_waves_node_grpc_transactions_api_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SignRequest.ProtoReflect.Descriptor instead.
func (*SignRequest) Descriptor() ([]byte, []int) {
	return file_waves_node_grpc_transactions_api_proto_rawDescGZIP(), []int{5}
}

func (x *SignRequest) GetTransaction() *waves.Transaction {
	if x != nil {
		return x.Transaction
	}
	return nil
}

func (x *SignRequest) GetSignerPublicKey() []byte {
	if x != nil {
		return x.SignerPublicKey
	}
	return nil
}

type InvokeScriptResultResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Transaction *waves.SignedTransaction  `protobuf:"bytes,1,opt,name=transaction,proto3" json:"transaction,omitempty"`
	Result      *waves.InvokeScriptResult `protobuf:"bytes,2,opt,name=result,proto3" json:"result,omitempty"`
}

func (x *InvokeScriptResultResponse) Reset() {
	*x = InvokeScriptResultResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_waves_node_grpc_transactions_api_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *InvokeScriptResultResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*InvokeScriptResultResponse) ProtoMessage() {}

func (x *InvokeScriptResultResponse) ProtoReflect() protoreflect.Message {
	mi := &file_waves_node_grpc_transactions_api_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use InvokeScriptResultResponse.ProtoReflect.Descriptor instead.
func (*InvokeScriptResultResponse) Descriptor() ([]byte, []int) {
	return file_waves_node_grpc_transactions_api_proto_rawDescGZIP(), []int{6}
}

func (x *InvokeScriptResultResponse) GetTransaction() *waves.SignedTransaction {
	if x != nil {
		return x.Transaction
	}
	return nil
}

func (x *InvokeScriptResultResponse) GetResult() *waves.InvokeScriptResult {
	if x != nil {
		return x.Result
	}
	return nil
}

var File_waves_node_grpc_transactions_api_proto protoreflect.FileDescriptor

var file_waves_node_grpc_transactions_api_proto_rawDesc = []byte{
	0x0a, 0x26, 0x77, 0x61, 0x76, 0x65, 0x73, 0x2f, 0x6e, 0x6f, 0x64, 0x65, 0x2f, 0x67, 0x72, 0x70,
	0x63, 0x2f, 0x74, 0x72, 0x61, 0x6e, 0x73, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x5f, 0x61,
	0x70, 0x69, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0f, 0x77, 0x61, 0x76, 0x65, 0x73, 0x2e,
	0x6e, 0x6f, 0x64, 0x65, 0x2e, 0x67, 0x72, 0x70, 0x63, 0x1a, 0x15, 0x77, 0x61, 0x76, 0x65, 0x73,
	0x2f, 0x72, 0x65, 0x63, 0x69, 0x70, 0x69, 0x65, 0x6e, 0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x1a, 0x17, 0x77, 0x61, 0x76, 0x65, 0x73, 0x2f, 0x74, 0x72, 0x61, 0x6e, 0x73, 0x61, 0x63, 0x74,
	0x69, 0x6f, 0x6e, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x20, 0x77, 0x61, 0x76, 0x65, 0x73,
	0x2f, 0x69, 0x6e, 0x76, 0x6f, 0x6b, 0x65, 0x5f, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x5f, 0x72,
	0x65, 0x73, 0x75, 0x6c, 0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x8b, 0x02, 0x0a, 0x11,
	0x54, 0x72, 0x61, 0x6e, 0x73, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x53, 0x74, 0x61, 0x74, 0x75,
	0x73, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x02, 0x69,
	0x64, 0x12, 0x41, 0x0a, 0x06, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x18, 0x02, 0x20, 0x01, 0x28,
	0x0e, 0x32, 0x29, 0x2e, 0x77, 0x61, 0x76, 0x65, 0x73, 0x2e, 0x6e, 0x6f, 0x64, 0x65, 0x2e, 0x67,
	0x72, 0x70, 0x63, 0x2e, 0x54, 0x72, 0x61, 0x6e, 0x73, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x53,
	0x74, 0x61, 0x74, 0x75, 0x73, 0x2e, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x52, 0x06, 0x73, 0x74,
	0x61, 0x74, 0x75, 0x73, 0x12, 0x16, 0x0a, 0x06, 0x68, 0x65, 0x69, 0x67, 0x68, 0x74, 0x18, 0x03,
	0x20, 0x01, 0x28, 0x03, 0x52, 0x06, 0x68, 0x65, 0x69, 0x67, 0x68, 0x74, 0x12, 0x51, 0x0a, 0x12,
	0x61, 0x70, 0x70, 0x6c, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x5f, 0x73, 0x74, 0x61, 0x74,
	0x75, 0x73, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x22, 0x2e, 0x77, 0x61, 0x76, 0x65, 0x73,
	0x2e, 0x6e, 0x6f, 0x64, 0x65, 0x2e, 0x67, 0x72, 0x70, 0x63, 0x2e, 0x41, 0x70, 0x70, 0x6c, 0x69,
	0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x52, 0x11, 0x61, 0x70,
	0x70, 0x6c, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x22,
	0x38, 0x0a, 0x06, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x12, 0x0e, 0x0a, 0x0a, 0x4e, 0x4f, 0x54,
	0x5f, 0x45, 0x58, 0x49, 0x53, 0x54, 0x53, 0x10, 0x00, 0x12, 0x0f, 0x0a, 0x0b, 0x55, 0x4e, 0x43,
	0x4f, 0x4e, 0x46, 0x49, 0x52, 0x4d, 0x45, 0x44, 0x10, 0x01, 0x12, 0x0d, 0x0a, 0x09, 0x43, 0x4f,
	0x4e, 0x46, 0x49, 0x52, 0x4d, 0x45, 0x44, 0x10, 0x02, 0x22, 0x99, 0x02, 0x0a, 0x13, 0x54, 0x72,
	0x61, 0x6e, 0x73, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73,
	0x65, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x02, 0x69,
	0x64, 0x12, 0x16, 0x0a, 0x06, 0x68, 0x65, 0x69, 0x67, 0x68, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28,
	0x03, 0x52, 0x06, 0x68, 0x65, 0x69, 0x67, 0x68, 0x74, 0x12, 0x3a, 0x0a, 0x0b, 0x74, 0x72, 0x61,
	0x6e, 0x73, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x18,
	0x2e, 0x77, 0x61, 0x76, 0x65, 0x73, 0x2e, 0x53, 0x69, 0x67, 0x6e, 0x65, 0x64, 0x54, 0x72, 0x61,
	0x6e, 0x73, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x0b, 0x74, 0x72, 0x61, 0x6e, 0x73, 0x61,
	0x63, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x51, 0x0a, 0x12, 0x61, 0x70, 0x70, 0x6c, 0x69, 0x63, 0x61,
	0x74, 0x69, 0x6f, 0x6e, 0x5f, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x18, 0x04, 0x20, 0x01, 0x28,
	0x0e, 0x32, 0x22, 0x2e, 0x77, 0x61, 0x76, 0x65, 0x73, 0x2e, 0x6e, 0x6f, 0x64, 0x65, 0x2e, 0x67,
	0x72, 0x70, 0x63, 0x2e, 0x41, 0x70, 0x70, 0x6c, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x53,
	0x74, 0x61, 0x74, 0x75, 0x73, 0x52, 0x11, 0x61, 0x70, 0x70, 0x6c, 0x69, 0x63, 0x61, 0x74, 0x69,
	0x6f, 0x6e, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x12, 0x4b, 0x0a, 0x14, 0x69, 0x6e, 0x76, 0x6f,
	0x6b, 0x65, 0x5f, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x5f, 0x72, 0x65, 0x73, 0x75, 0x6c, 0x74,
	0x18, 0x05, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x19, 0x2e, 0x77, 0x61, 0x76, 0x65, 0x73, 0x2e, 0x49,
	0x6e, 0x76, 0x6f, 0x6b, 0x65, 0x53, 0x63, 0x72, 0x69, 0x70, 0x74, 0x52, 0x65, 0x73, 0x75, 0x6c,
	0x74, 0x52, 0x12, 0x69, 0x6e, 0x76, 0x6f, 0x6b, 0x65, 0x53, 0x63, 0x72, 0x69, 0x70, 0x74, 0x52,
	0x65, 0x73, 0x75, 0x6c, 0x74, 0x22, 0x86, 0x01, 0x0a, 0x13, 0x54, 0x72, 0x61, 0x6e, 0x73, 0x61,
	0x63, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x16, 0x0a,
	0x06, 0x73, 0x65, 0x6e, 0x64, 0x65, 0x72, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x06, 0x73,
	0x65, 0x6e, 0x64, 0x65, 0x72, 0x12, 0x2e, 0x0a, 0x09, 0x72, 0x65, 0x63, 0x69, 0x70, 0x69, 0x65,
	0x6e, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x10, 0x2e, 0x77, 0x61, 0x76, 0x65, 0x73,
	0x2e, 0x52, 0x65, 0x63, 0x69, 0x70, 0x69, 0x65, 0x6e, 0x74, 0x52, 0x09, 0x72, 0x65, 0x63, 0x69,
	0x70, 0x69, 0x65, 0x6e, 0x74, 0x12, 0x27, 0x0a, 0x0f, 0x74, 0x72, 0x61, 0x6e, 0x73, 0x61, 0x63,
	0x74, 0x69, 0x6f, 0x6e, 0x5f, 0x69, 0x64, 0x73, 0x18, 0x03, 0x20, 0x03, 0x28, 0x0c, 0x52, 0x0e,
	0x74, 0x72, 0x61, 0x6e, 0x73, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x49, 0x64, 0x73, 0x22, 0x42,
	0x0a, 0x17, 0x54, 0x72, 0x61, 0x6e, 0x73, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x42, 0x79,
	0x49, 0x64, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x27, 0x0a, 0x0f, 0x74, 0x72, 0x61,
	0x6e, 0x73, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x5f, 0x69, 0x64, 0x73, 0x18, 0x03, 0x20, 0x03,
	0x28, 0x0c, 0x52, 0x0e, 0x74, 0x72, 0x61, 0x6e, 0x73, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x49,
	0x64, 0x73, 0x22, 0x49, 0x0a, 0x14, 0x43, 0x61, 0x6c, 0x63, 0x75, 0x6c, 0x61, 0x74, 0x65, 0x46,
	0x65, 0x65, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x19, 0x0a, 0x08, 0x61, 0x73,
	0x73, 0x65, 0x74, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x07, 0x61, 0x73,
	0x73, 0x65, 0x74, 0x49, 0x64, 0x12, 0x16, 0x0a, 0x06, 0x61, 0x6d, 0x6f, 0x75, 0x6e, 0x74, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x04, 0x52, 0x06, 0x61, 0x6d, 0x6f, 0x75, 0x6e, 0x74, 0x22, 0x6f, 0x0a,
	0x0b, 0x53, 0x69, 0x67, 0x6e, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x34, 0x0a, 0x0b,
	0x74, 0x72, 0x61, 0x6e, 0x73, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x0b, 0x32, 0x12, 0x2e, 0x77, 0x61, 0x76, 0x65, 0x73, 0x2e, 0x54, 0x72, 0x61, 0x6e, 0x73, 0x61,
	0x63, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x0b, 0x74, 0x72, 0x61, 0x6e, 0x73, 0x61, 0x63, 0x74, 0x69,
	0x6f, 0x6e, 0x12, 0x2a, 0x0a, 0x11, 0x73, 0x69, 0x67, 0x6e, 0x65, 0x72, 0x5f, 0x70, 0x75, 0x62,
	0x6c, 0x69, 0x63, 0x5f, 0x6b, 0x65, 0x79, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x0f, 0x73,
	0x69, 0x67, 0x6e, 0x65, 0x72, 0x50, 0x75, 0x62, 0x6c, 0x69, 0x63, 0x4b, 0x65, 0x79, 0x22, 0x8b,
	0x01, 0x0a, 0x1a, 0x49, 0x6e, 0x76, 0x6f, 0x6b, 0x65, 0x53, 0x63, 0x72, 0x69, 0x70, 0x74, 0x52,
	0x65, 0x73, 0x75, 0x6c, 0x74, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x3a, 0x0a,
	0x0b, 0x74, 0x72, 0x61, 0x6e, 0x73, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x18, 0x2e, 0x77, 0x61, 0x76, 0x65, 0x73, 0x2e, 0x53, 0x69, 0x67, 0x6e, 0x65,
	0x64, 0x54, 0x72, 0x61, 0x6e, 0x73, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x0b, 0x74, 0x72,
	0x61, 0x6e, 0x73, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x31, 0x0a, 0x06, 0x72, 0x65, 0x73,
	0x75, 0x6c, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x19, 0x2e, 0x77, 0x61, 0x76, 0x65,
	0x73, 0x2e, 0x49, 0x6e, 0x76, 0x6f, 0x6b, 0x65, 0x53, 0x63, 0x72, 0x69, 0x70, 0x74, 0x52, 0x65,
	0x73, 0x75, 0x6c, 0x74, 0x52, 0x06, 0x72, 0x65, 0x73, 0x75, 0x6c, 0x74, 0x2a, 0x4c, 0x0a, 0x11,
	0x41, 0x70, 0x70, 0x6c, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x53, 0x74, 0x61, 0x74, 0x75,
	0x73, 0x12, 0x0b, 0x0a, 0x07, 0x55, 0x4e, 0x4b, 0x4e, 0x4f, 0x57, 0x4e, 0x10, 0x00, 0x12, 0x0d,
	0x0a, 0x09, 0x53, 0x55, 0x43, 0x43, 0x45, 0x45, 0x44, 0x45, 0x44, 0x10, 0x01, 0x12, 0x1b, 0x0a,
	0x17, 0x53, 0x43, 0x52, 0x49, 0x50, 0x54, 0x5f, 0x45, 0x58, 0x45, 0x43, 0x55, 0x54, 0x49, 0x4f,
	0x4e, 0x5f, 0x46, 0x41, 0x49, 0x4c, 0x45, 0x44, 0x10, 0x02, 0x32, 0x9f, 0x04, 0x0a, 0x0f, 0x54,
	0x72, 0x61, 0x6e, 0x73, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x41, 0x70, 0x69, 0x12, 0x5f,
	0x0a, 0x0f, 0x47, 0x65, 0x74, 0x54, 0x72, 0x61, 0x6e, 0x73, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e,
	0x73, 0x12, 0x24, 0x2e, 0x77, 0x61, 0x76, 0x65, 0x73, 0x2e, 0x6e, 0x6f, 0x64, 0x65, 0x2e, 0x67,
	0x72, 0x70, 0x63, 0x2e, 0x54, 0x72, 0x61, 0x6e, 0x73, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x73,
	0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x24, 0x2e, 0x77, 0x61, 0x76, 0x65, 0x73, 0x2e,
	0x6e, 0x6f, 0x64, 0x65, 0x2e, 0x67, 0x72, 0x70, 0x63, 0x2e, 0x54, 0x72, 0x61, 0x6e, 0x73, 0x61,
	0x63, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x30, 0x01, 0x12,
	0x6b, 0x0a, 0x0f, 0x47, 0x65, 0x74, 0x53, 0x74, 0x61, 0x74, 0x65, 0x43, 0x68, 0x61, 0x6e, 0x67,
	0x65, 0x73, 0x12, 0x24, 0x2e, 0x77, 0x61, 0x76, 0x65, 0x73, 0x2e, 0x6e, 0x6f, 0x64, 0x65, 0x2e,
	0x67, 0x72, 0x70, 0x63, 0x2e, 0x54, 0x72, 0x61, 0x6e, 0x73, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e,
	0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x2b, 0x2e, 0x77, 0x61, 0x76, 0x65, 0x73,
	0x2e, 0x6e, 0x6f, 0x64, 0x65, 0x2e, 0x67, 0x72, 0x70, 0x63, 0x2e, 0x49, 0x6e, 0x76, 0x6f, 0x6b,
	0x65, 0x53, 0x63, 0x72, 0x69, 0x70, 0x74, 0x52, 0x65, 0x73, 0x75, 0x6c, 0x74, 0x52, 0x65, 0x73,
	0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x03, 0x88, 0x02, 0x01, 0x30, 0x01, 0x12, 0x5d, 0x0a, 0x0b,
	0x47, 0x65, 0x74, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x65, 0x73, 0x12, 0x28, 0x2e, 0x77, 0x61,
	0x76, 0x65, 0x73, 0x2e, 0x6e, 0x6f, 0x64, 0x65, 0x2e, 0x67, 0x72, 0x70, 0x63, 0x2e, 0x54, 0x72,
	0x61, 0x6e, 0x73, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x42, 0x79, 0x49, 0x64, 0x52, 0x65,
	0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x22, 0x2e, 0x77, 0x61, 0x76, 0x65, 0x73, 0x2e, 0x6e, 0x6f,
	0x64, 0x65, 0x2e, 0x67, 0x72, 0x70, 0x63, 0x2e, 0x54, 0x72, 0x61, 0x6e, 0x73, 0x61, 0x63, 0x74,
	0x69, 0x6f, 0x6e, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x30, 0x01, 0x12, 0x5e, 0x0a, 0x0e, 0x47,
	0x65, 0x74, 0x55, 0x6e, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x72, 0x6d, 0x65, 0x64, 0x12, 0x24, 0x2e,
	0x77, 0x61, 0x76, 0x65, 0x73, 0x2e, 0x6e, 0x6f, 0x64, 0x65, 0x2e, 0x67, 0x72, 0x70, 0x63, 0x2e,
	0x54, 0x72, 0x61, 0x6e, 0x73, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x52, 0x65, 0x71, 0x75,
	0x65, 0x73, 0x74, 0x1a, 0x24, 0x2e, 0x77, 0x61, 0x76, 0x65, 0x73, 0x2e, 0x6e, 0x6f, 0x64, 0x65,
	0x2e, 0x67, 0x72, 0x70, 0x63, 0x2e, 0x54, 0x72, 0x61, 0x6e, 0x73, 0x61, 0x63, 0x74, 0x69, 0x6f,
	0x6e, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x30, 0x01, 0x12, 0x3e, 0x0a, 0x04, 0x53,
	0x69, 0x67, 0x6e, 0x12, 0x1c, 0x2e, 0x77, 0x61, 0x76, 0x65, 0x73, 0x2e, 0x6e, 0x6f, 0x64, 0x65,
	0x2e, 0x67, 0x72, 0x70, 0x63, 0x2e, 0x53, 0x69, 0x67, 0x6e, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73,
	0x74, 0x1a, 0x18, 0x2e, 0x77, 0x61, 0x76, 0x65, 0x73, 0x2e, 0x53, 0x69, 0x67, 0x6e, 0x65, 0x64,
	0x54, 0x72, 0x61, 0x6e, 0x73, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x3f, 0x0a, 0x09, 0x42,
	0x72, 0x6f, 0x61, 0x64, 0x63, 0x61, 0x73, 0x74, 0x12, 0x18, 0x2e, 0x77, 0x61, 0x76, 0x65, 0x73,
	0x2e, 0x53, 0x69, 0x67, 0x6e, 0x65, 0x64, 0x54, 0x72, 0x61, 0x6e, 0x73, 0x61, 0x63, 0x74, 0x69,
	0x6f, 0x6e, 0x1a, 0x18, 0x2e, 0x77, 0x61, 0x76, 0x65, 0x73, 0x2e, 0x53, 0x69, 0x67, 0x6e, 0x65,
	0x64, 0x54, 0x72, 0x61, 0x6e, 0x73, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x42, 0x73, 0x0a, 0x1a,
	0x63, 0x6f, 0x6d, 0x2e, 0x77, 0x61, 0x76, 0x65, 0x73, 0x70, 0x6c, 0x61, 0x74, 0x66, 0x6f, 0x72,
	0x6d, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x67, 0x72, 0x70, 0x63, 0x5a, 0x43, 0x67, 0x69, 0x74, 0x68,
	0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x77, 0x61, 0x76, 0x65, 0x73, 0x70, 0x6c, 0x61, 0x74,
	0x66, 0x6f, 0x72, 0x6d, 0x2f, 0x67, 0x6f, 0x77, 0x61, 0x76, 0x65, 0x73, 0x2f, 0x70, 0x6b, 0x67,
	0x2f, 0x67, 0x72, 0x70, 0x63, 0x2f, 0x67, 0x65, 0x6e, 0x65, 0x72, 0x61, 0x74, 0x65, 0x64, 0x2f,
	0x77, 0x61, 0x76, 0x65, 0x73, 0x2f, 0x6e, 0x6f, 0x64, 0x65, 0x2f, 0x67, 0x72, 0x70, 0x63, 0xaa,
	0x02, 0x0f, 0x57, 0x61, 0x76, 0x65, 0x73, 0x2e, 0x4e, 0x6f, 0x64, 0x65, 0x2e, 0x47, 0x72, 0x70,
	0x63, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_waves_node_grpc_transactions_api_proto_rawDescOnce sync.Once
	file_waves_node_grpc_transactions_api_proto_rawDescData = file_waves_node_grpc_transactions_api_proto_rawDesc
)

func file_waves_node_grpc_transactions_api_proto_rawDescGZIP() []byte {
	file_waves_node_grpc_transactions_api_proto_rawDescOnce.Do(func() {
		file_waves_node_grpc_transactions_api_proto_rawDescData = protoimpl.X.CompressGZIP(file_waves_node_grpc_transactions_api_proto_rawDescData)
	})
	return file_waves_node_grpc_transactions_api_proto_rawDescData
}

var file_waves_node_grpc_transactions_api_proto_enumTypes = make([]protoimpl.EnumInfo, 2)
var file_waves_node_grpc_transactions_api_proto_msgTypes = make([]protoimpl.MessageInfo, 7)
var file_waves_node_grpc_transactions_api_proto_goTypes = []interface{}{
	(ApplicationStatus)(0),             // 0: waves.node.grpc.ApplicationStatus
	(TransactionStatus_Status)(0),      // 1: waves.node.grpc.TransactionStatus.Status
	(*TransactionStatus)(nil),          // 2: waves.node.grpc.TransactionStatus
	(*TransactionResponse)(nil),        // 3: waves.node.grpc.TransactionResponse
	(*TransactionsRequest)(nil),        // 4: waves.node.grpc.TransactionsRequest
	(*TransactionsByIdRequest)(nil),    // 5: waves.node.grpc.TransactionsByIdRequest
	(*CalculateFeeResponse)(nil),       // 6: waves.node.grpc.CalculateFeeResponse
	(*SignRequest)(nil),                // 7: waves.node.grpc.SignRequest
	(*InvokeScriptResultResponse)(nil), // 8: waves.node.grpc.InvokeScriptResultResponse
	(*waves.SignedTransaction)(nil),    // 9: waves.SignedTransaction
	(*waves.InvokeScriptResult)(nil),   // 10: waves.InvokeScriptResult
	(*waves.Recipient)(nil),            // 11: waves.Recipient
	(*waves.Transaction)(nil),          // 12: waves.Transaction
}
var file_waves_node_grpc_transactions_api_proto_depIdxs = []int32{
	1,  // 0: waves.node.grpc.TransactionStatus.status:type_name -> waves.node.grpc.TransactionStatus.Status
	0,  // 1: waves.node.grpc.TransactionStatus.application_status:type_name -> waves.node.grpc.ApplicationStatus
	9,  // 2: waves.node.grpc.TransactionResponse.transaction:type_name -> waves.SignedTransaction
	0,  // 3: waves.node.grpc.TransactionResponse.application_status:type_name -> waves.node.grpc.ApplicationStatus
	10, // 4: waves.node.grpc.TransactionResponse.invoke_script_result:type_name -> waves.InvokeScriptResult
	11, // 5: waves.node.grpc.TransactionsRequest.recipient:type_name -> waves.Recipient
	12, // 6: waves.node.grpc.SignRequest.transaction:type_name -> waves.Transaction
	9,  // 7: waves.node.grpc.InvokeScriptResultResponse.transaction:type_name -> waves.SignedTransaction
	10, // 8: waves.node.grpc.InvokeScriptResultResponse.result:type_name -> waves.InvokeScriptResult
	4,  // 9: waves.node.grpc.TransactionsApi.GetTransactions:input_type -> waves.node.grpc.TransactionsRequest
	4,  // 10: waves.node.grpc.TransactionsApi.GetStateChanges:input_type -> waves.node.grpc.TransactionsRequest
	5,  // 11: waves.node.grpc.TransactionsApi.GetStatuses:input_type -> waves.node.grpc.TransactionsByIdRequest
	4,  // 12: waves.node.grpc.TransactionsApi.GetUnconfirmed:input_type -> waves.node.grpc.TransactionsRequest
	7,  // 13: waves.node.grpc.TransactionsApi.Sign:input_type -> waves.node.grpc.SignRequest
	9,  // 14: waves.node.grpc.TransactionsApi.Broadcast:input_type -> waves.SignedTransaction
	3,  // 15: waves.node.grpc.TransactionsApi.GetTransactions:output_type -> waves.node.grpc.TransactionResponse
	8,  // 16: waves.node.grpc.TransactionsApi.GetStateChanges:output_type -> waves.node.grpc.InvokeScriptResultResponse
	2,  // 17: waves.node.grpc.TransactionsApi.GetStatuses:output_type -> waves.node.grpc.TransactionStatus
	3,  // 18: waves.node.grpc.TransactionsApi.GetUnconfirmed:output_type -> waves.node.grpc.TransactionResponse
	9,  // 19: waves.node.grpc.TransactionsApi.Sign:output_type -> waves.SignedTransaction
	9,  // 20: waves.node.grpc.TransactionsApi.Broadcast:output_type -> waves.SignedTransaction
	15, // [15:21] is the sub-list for method output_type
	9,  // [9:15] is the sub-list for method input_type
	9,  // [9:9] is the sub-list for extension type_name
	9,  // [9:9] is the sub-list for extension extendee
	0,  // [0:9] is the sub-list for field type_name
}

func init() { file_waves_node_grpc_transactions_api_proto_init() }
func file_waves_node_grpc_transactions_api_proto_init() {
	if File_waves_node_grpc_transactions_api_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_waves_node_grpc_transactions_api_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TransactionStatus); i {
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
		file_waves_node_grpc_transactions_api_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TransactionResponse); i {
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
		file_waves_node_grpc_transactions_api_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TransactionsRequest); i {
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
		file_waves_node_grpc_transactions_api_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TransactionsByIdRequest); i {
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
		file_waves_node_grpc_transactions_api_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*CalculateFeeResponse); i {
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
		file_waves_node_grpc_transactions_api_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*SignRequest); i {
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
		file_waves_node_grpc_transactions_api_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*InvokeScriptResultResponse); i {
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
			RawDescriptor: file_waves_node_grpc_transactions_api_proto_rawDesc,
			NumEnums:      2,
			NumMessages:   7,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_waves_node_grpc_transactions_api_proto_goTypes,
		DependencyIndexes: file_waves_node_grpc_transactions_api_proto_depIdxs,
		EnumInfos:         file_waves_node_grpc_transactions_api_proto_enumTypes,
		MessageInfos:      file_waves_node_grpc_transactions_api_proto_msgTypes,
	}.Build()
	File_waves_node_grpc_transactions_api_proto = out.File
	file_waves_node_grpc_transactions_api_proto_rawDesc = nil
	file_waves_node_grpc_transactions_api_proto_goTypes = nil
	file_waves_node_grpc_transactions_api_proto_depIdxs = nil
}
