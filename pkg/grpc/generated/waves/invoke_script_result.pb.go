// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.25.0
// 	protoc        v3.12.3
// source: waves/invoke_script_result.proto

package waves

import (
	proto "github.com/golang/protobuf/proto"
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

type InvokeScriptResult struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Data         []*DataTransactionData_DataEntry `protobuf:"bytes,1,rep,name=data,proto3" json:"data,omitempty"`
	Transfers    []*InvokeScriptResult_Payment    `protobuf:"bytes,2,rep,name=transfers,proto3" json:"transfers,omitempty"`
	Issues       []*InvokeScriptResult_Issue      `protobuf:"bytes,3,rep,name=issues,proto3" json:"issues,omitempty"`
	Reissues     []*InvokeScriptResult_Reissue    `protobuf:"bytes,4,rep,name=reissues,proto3" json:"reissues,omitempty"`
	Burns        []*InvokeScriptResult_Burn       `protobuf:"bytes,5,rep,name=burns,proto3" json:"burns,omitempty"`
	ErrorMessage *InvokeScriptResult_ErrorMessage `protobuf:"bytes,6,opt,name=error_message,json=errorMessage,proto3" json:"error_message,omitempty"`
	SponsorFees  []*InvokeScriptResult_SponsorFee `protobuf:"bytes,7,rep,name=sponsor_fees,json=sponsorFees,proto3" json:"sponsor_fees,omitempty"`
}

func (x *InvokeScriptResult) Reset() {
	*x = InvokeScriptResult{}
	if protoimpl.UnsafeEnabled {
		mi := &file_waves_invoke_script_result_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *InvokeScriptResult) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*InvokeScriptResult) ProtoMessage() {}

func (x *InvokeScriptResult) ProtoReflect() protoreflect.Message {
	mi := &file_waves_invoke_script_result_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use InvokeScriptResult.ProtoReflect.Descriptor instead.
func (*InvokeScriptResult) Descriptor() ([]byte, []int) {
	return file_waves_invoke_script_result_proto_rawDescGZIP(), []int{0}
}

func (x *InvokeScriptResult) GetData() []*DataTransactionData_DataEntry {
	if x != nil {
		return x.Data
	}
	return nil
}

func (x *InvokeScriptResult) GetTransfers() []*InvokeScriptResult_Payment {
	if x != nil {
		return x.Transfers
	}
	return nil
}

func (x *InvokeScriptResult) GetIssues() []*InvokeScriptResult_Issue {
	if x != nil {
		return x.Issues
	}
	return nil
}

func (x *InvokeScriptResult) GetReissues() []*InvokeScriptResult_Reissue {
	if x != nil {
		return x.Reissues
	}
	return nil
}

func (x *InvokeScriptResult) GetBurns() []*InvokeScriptResult_Burn {
	if x != nil {
		return x.Burns
	}
	return nil
}

func (x *InvokeScriptResult) GetErrorMessage() *InvokeScriptResult_ErrorMessage {
	if x != nil {
		return x.ErrorMessage
	}
	return nil
}

func (x *InvokeScriptResult) GetSponsorFees() []*InvokeScriptResult_SponsorFee {
	if x != nil {
		return x.SponsorFees
	}
	return nil
}

type InvokeScriptResult_Payment struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Address []byte  `protobuf:"bytes,1,opt,name=address,proto3" json:"address,omitempty"`
	Amount  *Amount `protobuf:"bytes,2,opt,name=amount,proto3" json:"amount,omitempty"`
}

func (x *InvokeScriptResult_Payment) Reset() {
	*x = InvokeScriptResult_Payment{}
	if protoimpl.UnsafeEnabled {
		mi := &file_waves_invoke_script_result_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *InvokeScriptResult_Payment) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*InvokeScriptResult_Payment) ProtoMessage() {}

func (x *InvokeScriptResult_Payment) ProtoReflect() protoreflect.Message {
	mi := &file_waves_invoke_script_result_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use InvokeScriptResult_Payment.ProtoReflect.Descriptor instead.
func (*InvokeScriptResult_Payment) Descriptor() ([]byte, []int) {
	return file_waves_invoke_script_result_proto_rawDescGZIP(), []int{0, 0}
}

func (x *InvokeScriptResult_Payment) GetAddress() []byte {
	if x != nil {
		return x.Address
	}
	return nil
}

func (x *InvokeScriptResult_Payment) GetAmount() *Amount {
	if x != nil {
		return x.Amount
	}
	return nil
}

type InvokeScriptResult_Issue struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	AssetId     []byte `protobuf:"bytes,1,opt,name=asset_id,json=assetId,proto3" json:"asset_id,omitempty"`
	Name        string `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
	Description string `protobuf:"bytes,3,opt,name=description,proto3" json:"description,omitempty"`
	Amount      int64  `protobuf:"varint,4,opt,name=amount,proto3" json:"amount,omitempty"`
	Decimals    int32  `protobuf:"varint,5,opt,name=decimals,proto3" json:"decimals,omitempty"`
	Reissuable  bool   `protobuf:"varint,6,opt,name=reissuable,proto3" json:"reissuable,omitempty"`
	Script      []byte `protobuf:"bytes,7,opt,name=script,proto3" json:"script,omitempty"`
	Nonce       int64  `protobuf:"varint,8,opt,name=nonce,proto3" json:"nonce,omitempty"`
}

func (x *InvokeScriptResult_Issue) Reset() {
	*x = InvokeScriptResult_Issue{}
	if protoimpl.UnsafeEnabled {
		mi := &file_waves_invoke_script_result_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *InvokeScriptResult_Issue) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*InvokeScriptResult_Issue) ProtoMessage() {}

func (x *InvokeScriptResult_Issue) ProtoReflect() protoreflect.Message {
	mi := &file_waves_invoke_script_result_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use InvokeScriptResult_Issue.ProtoReflect.Descriptor instead.
func (*InvokeScriptResult_Issue) Descriptor() ([]byte, []int) {
	return file_waves_invoke_script_result_proto_rawDescGZIP(), []int{0, 1}
}

func (x *InvokeScriptResult_Issue) GetAssetId() []byte {
	if x != nil {
		return x.AssetId
	}
	return nil
}

func (x *InvokeScriptResult_Issue) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *InvokeScriptResult_Issue) GetDescription() string {
	if x != nil {
		return x.Description
	}
	return ""
}

func (x *InvokeScriptResult_Issue) GetAmount() int64 {
	if x != nil {
		return x.Amount
	}
	return 0
}

func (x *InvokeScriptResult_Issue) GetDecimals() int32 {
	if x != nil {
		return x.Decimals
	}
	return 0
}

func (x *InvokeScriptResult_Issue) GetReissuable() bool {
	if x != nil {
		return x.Reissuable
	}
	return false
}

func (x *InvokeScriptResult_Issue) GetScript() []byte {
	if x != nil {
		return x.Script
	}
	return nil
}

func (x *InvokeScriptResult_Issue) GetNonce() int64 {
	if x != nil {
		return x.Nonce
	}
	return 0
}

type InvokeScriptResult_Reissue struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	AssetId      []byte `protobuf:"bytes,1,opt,name=asset_id,json=assetId,proto3" json:"asset_id,omitempty"`
	Amount       int64  `protobuf:"varint,2,opt,name=amount,proto3" json:"amount,omitempty"`
	IsReissuable bool   `protobuf:"varint,3,opt,name=is_reissuable,json=isReissuable,proto3" json:"is_reissuable,omitempty"`
}

func (x *InvokeScriptResult_Reissue) Reset() {
	*x = InvokeScriptResult_Reissue{}
	if protoimpl.UnsafeEnabled {
		mi := &file_waves_invoke_script_result_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *InvokeScriptResult_Reissue) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*InvokeScriptResult_Reissue) ProtoMessage() {}

func (x *InvokeScriptResult_Reissue) ProtoReflect() protoreflect.Message {
	mi := &file_waves_invoke_script_result_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use InvokeScriptResult_Reissue.ProtoReflect.Descriptor instead.
func (*InvokeScriptResult_Reissue) Descriptor() ([]byte, []int) {
	return file_waves_invoke_script_result_proto_rawDescGZIP(), []int{0, 2}
}

func (x *InvokeScriptResult_Reissue) GetAssetId() []byte {
	if x != nil {
		return x.AssetId
	}
	return nil
}

func (x *InvokeScriptResult_Reissue) GetAmount() int64 {
	if x != nil {
		return x.Amount
	}
	return 0
}

func (x *InvokeScriptResult_Reissue) GetIsReissuable() bool {
	if x != nil {
		return x.IsReissuable
	}
	return false
}

type InvokeScriptResult_Burn struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	AssetId []byte `protobuf:"bytes,1,opt,name=asset_id,json=assetId,proto3" json:"asset_id,omitempty"`
	Amount  int64  `protobuf:"varint,2,opt,name=amount,proto3" json:"amount,omitempty"`
}

func (x *InvokeScriptResult_Burn) Reset() {
	*x = InvokeScriptResult_Burn{}
	if protoimpl.UnsafeEnabled {
		mi := &file_waves_invoke_script_result_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *InvokeScriptResult_Burn) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*InvokeScriptResult_Burn) ProtoMessage() {}

func (x *InvokeScriptResult_Burn) ProtoReflect() protoreflect.Message {
	mi := &file_waves_invoke_script_result_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use InvokeScriptResult_Burn.ProtoReflect.Descriptor instead.
func (*InvokeScriptResult_Burn) Descriptor() ([]byte, []int) {
	return file_waves_invoke_script_result_proto_rawDescGZIP(), []int{0, 3}
}

func (x *InvokeScriptResult_Burn) GetAssetId() []byte {
	if x != nil {
		return x.AssetId
	}
	return nil
}

func (x *InvokeScriptResult_Burn) GetAmount() int64 {
	if x != nil {
		return x.Amount
	}
	return 0
}

type InvokeScriptResult_SponsorFee struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	MinFee *Amount `protobuf:"bytes,1,opt,name=min_fee,json=minFee,proto3" json:"min_fee,omitempty"`
}

func (x *InvokeScriptResult_SponsorFee) Reset() {
	*x = InvokeScriptResult_SponsorFee{}
	if protoimpl.UnsafeEnabled {
		mi := &file_waves_invoke_script_result_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *InvokeScriptResult_SponsorFee) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*InvokeScriptResult_SponsorFee) ProtoMessage() {}

func (x *InvokeScriptResult_SponsorFee) ProtoReflect() protoreflect.Message {
	mi := &file_waves_invoke_script_result_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use InvokeScriptResult_SponsorFee.ProtoReflect.Descriptor instead.
func (*InvokeScriptResult_SponsorFee) Descriptor() ([]byte, []int) {
	return file_waves_invoke_script_result_proto_rawDescGZIP(), []int{0, 4}
}

func (x *InvokeScriptResult_SponsorFee) GetMinFee() *Amount {
	if x != nil {
		return x.MinFee
	}
	return nil
}

type InvokeScriptResult_ErrorMessage struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Code int32  `protobuf:"varint,1,opt,name=code,proto3" json:"code,omitempty"`
	Text string `protobuf:"bytes,2,opt,name=text,proto3" json:"text,omitempty"`
}

func (x *InvokeScriptResult_ErrorMessage) Reset() {
	*x = InvokeScriptResult_ErrorMessage{}
	if protoimpl.UnsafeEnabled {
		mi := &file_waves_invoke_script_result_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *InvokeScriptResult_ErrorMessage) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*InvokeScriptResult_ErrorMessage) ProtoMessage() {}

func (x *InvokeScriptResult_ErrorMessage) ProtoReflect() protoreflect.Message {
	mi := &file_waves_invoke_script_result_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use InvokeScriptResult_ErrorMessage.ProtoReflect.Descriptor instead.
func (*InvokeScriptResult_ErrorMessage) Descriptor() ([]byte, []int) {
	return file_waves_invoke_script_result_proto_rawDescGZIP(), []int{0, 5}
}

func (x *InvokeScriptResult_ErrorMessage) GetCode() int32 {
	if x != nil {
		return x.Code
	}
	return 0
}

func (x *InvokeScriptResult_ErrorMessage) GetText() string {
	if x != nil {
		return x.Text
	}
	return ""
}

var File_waves_invoke_script_result_proto protoreflect.FileDescriptor

var file_waves_invoke_script_result_proto_rawDesc = []byte{
	0x0a, 0x20, 0x77, 0x61, 0x76, 0x65, 0x73, 0x2f, 0x69, 0x6e, 0x76, 0x6f, 0x6b, 0x65, 0x5f, 0x73,
	0x63, 0x72, 0x69, 0x70, 0x74, 0x5f, 0x72, 0x65, 0x73, 0x75, 0x6c, 0x74, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x12, 0x05, 0x77, 0x61, 0x76, 0x65, 0x73, 0x1a, 0x17, 0x77, 0x61, 0x76, 0x65, 0x73,
	0x2f, 0x74, 0x72, 0x61, 0x6e, 0x73, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x1a, 0x12, 0x77, 0x61, 0x76, 0x65, 0x73, 0x2f, 0x61, 0x6d, 0x6f, 0x75, 0x6e, 0x74,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x88, 0x08, 0x0a, 0x12, 0x49, 0x6e, 0x76, 0x6f, 0x6b,
	0x65, 0x53, 0x63, 0x72, 0x69, 0x70, 0x74, 0x52, 0x65, 0x73, 0x75, 0x6c, 0x74, 0x12, 0x38, 0x0a,
	0x04, 0x64, 0x61, 0x74, 0x61, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x24, 0x2e, 0x77, 0x61,
	0x76, 0x65, 0x73, 0x2e, 0x44, 0x61, 0x74, 0x61, 0x54, 0x72, 0x61, 0x6e, 0x73, 0x61, 0x63, 0x74,
	0x69, 0x6f, 0x6e, 0x44, 0x61, 0x74, 0x61, 0x2e, 0x44, 0x61, 0x74, 0x61, 0x45, 0x6e, 0x74, 0x72,
	0x79, 0x52, 0x04, 0x64, 0x61, 0x74, 0x61, 0x12, 0x3f, 0x0a, 0x09, 0x74, 0x72, 0x61, 0x6e, 0x73,
	0x66, 0x65, 0x72, 0x73, 0x18, 0x02, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x21, 0x2e, 0x77, 0x61, 0x76,
	0x65, 0x73, 0x2e, 0x49, 0x6e, 0x76, 0x6f, 0x6b, 0x65, 0x53, 0x63, 0x72, 0x69, 0x70, 0x74, 0x52,
	0x65, 0x73, 0x75, 0x6c, 0x74, 0x2e, 0x50, 0x61, 0x79, 0x6d, 0x65, 0x6e, 0x74, 0x52, 0x09, 0x74,
	0x72, 0x61, 0x6e, 0x73, 0x66, 0x65, 0x72, 0x73, 0x12, 0x37, 0x0a, 0x06, 0x69, 0x73, 0x73, 0x75,
	0x65, 0x73, 0x18, 0x03, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x1f, 0x2e, 0x77, 0x61, 0x76, 0x65, 0x73,
	0x2e, 0x49, 0x6e, 0x76, 0x6f, 0x6b, 0x65, 0x53, 0x63, 0x72, 0x69, 0x70, 0x74, 0x52, 0x65, 0x73,
	0x75, 0x6c, 0x74, 0x2e, 0x49, 0x73, 0x73, 0x75, 0x65, 0x52, 0x06, 0x69, 0x73, 0x73, 0x75, 0x65,
	0x73, 0x12, 0x3d, 0x0a, 0x08, 0x72, 0x65, 0x69, 0x73, 0x73, 0x75, 0x65, 0x73, 0x18, 0x04, 0x20,
	0x03, 0x28, 0x0b, 0x32, 0x21, 0x2e, 0x77, 0x61, 0x76, 0x65, 0x73, 0x2e, 0x49, 0x6e, 0x76, 0x6f,
	0x6b, 0x65, 0x53, 0x63, 0x72, 0x69, 0x70, 0x74, 0x52, 0x65, 0x73, 0x75, 0x6c, 0x74, 0x2e, 0x52,
	0x65, 0x69, 0x73, 0x73, 0x75, 0x65, 0x52, 0x08, 0x72, 0x65, 0x69, 0x73, 0x73, 0x75, 0x65, 0x73,
	0x12, 0x34, 0x0a, 0x05, 0x62, 0x75, 0x72, 0x6e, 0x73, 0x18, 0x05, 0x20, 0x03, 0x28, 0x0b, 0x32,
	0x1e, 0x2e, 0x77, 0x61, 0x76, 0x65, 0x73, 0x2e, 0x49, 0x6e, 0x76, 0x6f, 0x6b, 0x65, 0x53, 0x63,
	0x72, 0x69, 0x70, 0x74, 0x52, 0x65, 0x73, 0x75, 0x6c, 0x74, 0x2e, 0x42, 0x75, 0x72, 0x6e, 0x52,
	0x05, 0x62, 0x75, 0x72, 0x6e, 0x73, 0x12, 0x4b, 0x0a, 0x0d, 0x65, 0x72, 0x72, 0x6f, 0x72, 0x5f,
	0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x18, 0x06, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x26, 0x2e,
	0x77, 0x61, 0x76, 0x65, 0x73, 0x2e, 0x49, 0x6e, 0x76, 0x6f, 0x6b, 0x65, 0x53, 0x63, 0x72, 0x69,
	0x70, 0x74, 0x52, 0x65, 0x73, 0x75, 0x6c, 0x74, 0x2e, 0x45, 0x72, 0x72, 0x6f, 0x72, 0x4d, 0x65,
	0x73, 0x73, 0x61, 0x67, 0x65, 0x52, 0x0c, 0x65, 0x72, 0x72, 0x6f, 0x72, 0x4d, 0x65, 0x73, 0x73,
	0x61, 0x67, 0x65, 0x12, 0x47, 0x0a, 0x0c, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x6f, 0x72, 0x5f, 0x66,
	0x65, 0x65, 0x73, 0x18, 0x07, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x24, 0x2e, 0x77, 0x61, 0x76, 0x65,
	0x73, 0x2e, 0x49, 0x6e, 0x76, 0x6f, 0x6b, 0x65, 0x53, 0x63, 0x72, 0x69, 0x70, 0x74, 0x52, 0x65,
	0x73, 0x75, 0x6c, 0x74, 0x2e, 0x53, 0x70, 0x6f, 0x6e, 0x73, 0x6f, 0x72, 0x46, 0x65, 0x65, 0x52,
	0x0b, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x6f, 0x72, 0x46, 0x65, 0x65, 0x73, 0x1a, 0x4a, 0x0a, 0x07,
	0x50, 0x61, 0x79, 0x6d, 0x65, 0x6e, 0x74, 0x12, 0x18, 0x0a, 0x07, 0x61, 0x64, 0x64, 0x72, 0x65,
	0x73, 0x73, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x07, 0x61, 0x64, 0x64, 0x72, 0x65, 0x73,
	0x73, 0x12, 0x25, 0x0a, 0x06, 0x61, 0x6d, 0x6f, 0x75, 0x6e, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28,
	0x0b, 0x32, 0x0d, 0x2e, 0x77, 0x61, 0x76, 0x65, 0x73, 0x2e, 0x41, 0x6d, 0x6f, 0x75, 0x6e, 0x74,
	0x52, 0x06, 0x61, 0x6d, 0x6f, 0x75, 0x6e, 0x74, 0x1a, 0xda, 0x01, 0x0a, 0x05, 0x49, 0x73, 0x73,
	0x75, 0x65, 0x12, 0x19, 0x0a, 0x08, 0x61, 0x73, 0x73, 0x65, 0x74, 0x5f, 0x69, 0x64, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x0c, 0x52, 0x07, 0x61, 0x73, 0x73, 0x65, 0x74, 0x49, 0x64, 0x12, 0x12, 0x0a,
	0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d,
	0x65, 0x12, 0x20, 0x0a, 0x0b, 0x64, 0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x69, 0x6f, 0x6e,
	0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0b, 0x64, 0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74,
	0x69, 0x6f, 0x6e, 0x12, 0x16, 0x0a, 0x06, 0x61, 0x6d, 0x6f, 0x75, 0x6e, 0x74, 0x18, 0x04, 0x20,
	0x01, 0x28, 0x03, 0x52, 0x06, 0x61, 0x6d, 0x6f, 0x75, 0x6e, 0x74, 0x12, 0x1a, 0x0a, 0x08, 0x64,
	0x65, 0x63, 0x69, 0x6d, 0x61, 0x6c, 0x73, 0x18, 0x05, 0x20, 0x01, 0x28, 0x05, 0x52, 0x08, 0x64,
	0x65, 0x63, 0x69, 0x6d, 0x61, 0x6c, 0x73, 0x12, 0x1e, 0x0a, 0x0a, 0x72, 0x65, 0x69, 0x73, 0x73,
	0x75, 0x61, 0x62, 0x6c, 0x65, 0x18, 0x06, 0x20, 0x01, 0x28, 0x08, 0x52, 0x0a, 0x72, 0x65, 0x69,
	0x73, 0x73, 0x75, 0x61, 0x62, 0x6c, 0x65, 0x12, 0x16, 0x0a, 0x06, 0x73, 0x63, 0x72, 0x69, 0x70,
	0x74, 0x18, 0x07, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x06, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x12,
	0x14, 0x0a, 0x05, 0x6e, 0x6f, 0x6e, 0x63, 0x65, 0x18, 0x08, 0x20, 0x01, 0x28, 0x03, 0x52, 0x05,
	0x6e, 0x6f, 0x6e, 0x63, 0x65, 0x1a, 0x61, 0x0a, 0x07, 0x52, 0x65, 0x69, 0x73, 0x73, 0x75, 0x65,
	0x12, 0x19, 0x0a, 0x08, 0x61, 0x73, 0x73, 0x65, 0x74, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x0c, 0x52, 0x07, 0x61, 0x73, 0x73, 0x65, 0x74, 0x49, 0x64, 0x12, 0x16, 0x0a, 0x06, 0x61,
	0x6d, 0x6f, 0x75, 0x6e, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x03, 0x52, 0x06, 0x61, 0x6d, 0x6f,
	0x75, 0x6e, 0x74, 0x12, 0x23, 0x0a, 0x0d, 0x69, 0x73, 0x5f, 0x72, 0x65, 0x69, 0x73, 0x73, 0x75,
	0x61, 0x62, 0x6c, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x08, 0x52, 0x0c, 0x69, 0x73, 0x52, 0x65,
	0x69, 0x73, 0x73, 0x75, 0x61, 0x62, 0x6c, 0x65, 0x1a, 0x39, 0x0a, 0x04, 0x42, 0x75, 0x72, 0x6e,
	0x12, 0x19, 0x0a, 0x08, 0x61, 0x73, 0x73, 0x65, 0x74, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x0c, 0x52, 0x07, 0x61, 0x73, 0x73, 0x65, 0x74, 0x49, 0x64, 0x12, 0x16, 0x0a, 0x06, 0x61,
	0x6d, 0x6f, 0x75, 0x6e, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x03, 0x52, 0x06, 0x61, 0x6d, 0x6f,
	0x75, 0x6e, 0x74, 0x1a, 0x34, 0x0a, 0x0a, 0x53, 0x70, 0x6f, 0x6e, 0x73, 0x6f, 0x72, 0x46, 0x65,
	0x65, 0x12, 0x26, 0x0a, 0x07, 0x6d, 0x69, 0x6e, 0x5f, 0x66, 0x65, 0x65, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x0d, 0x2e, 0x77, 0x61, 0x76, 0x65, 0x73, 0x2e, 0x41, 0x6d, 0x6f, 0x75, 0x6e,
	0x74, 0x52, 0x06, 0x6d, 0x69, 0x6e, 0x46, 0x65, 0x65, 0x1a, 0x36, 0x0a, 0x0c, 0x45, 0x72, 0x72,
	0x6f, 0x72, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x12, 0x12, 0x0a, 0x04, 0x63, 0x6f, 0x64,
	0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x05, 0x52, 0x04, 0x63, 0x6f, 0x64, 0x65, 0x12, 0x12, 0x0a,
	0x04, 0x74, 0x65, 0x78, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x74, 0x65, 0x78,
	0x74, 0x42, 0x6b, 0x0a, 0x26, 0x63, 0x6f, 0x6d, 0x2e, 0x77, 0x61, 0x76, 0x65, 0x73, 0x70, 0x6c,
	0x61, 0x74, 0x66, 0x6f, 0x72, 0x6d, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e,
	0x74, 0x72, 0x61, 0x6e, 0x73, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x5a, 0x39, 0x67, 0x69, 0x74,
	0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x77, 0x61, 0x76, 0x65, 0x73, 0x70, 0x6c, 0x61,
	0x74, 0x66, 0x6f, 0x72, 0x6d, 0x2f, 0x67, 0x6f, 0x77, 0x61, 0x76, 0x65, 0x73, 0x2f, 0x70, 0x6b,
	0x67, 0x2f, 0x67, 0x72, 0x70, 0x63, 0x2f, 0x67, 0x65, 0x6e, 0x65, 0x72, 0x61, 0x74, 0x65, 0x64,
	0x2f, 0x77, 0x61, 0x76, 0x65, 0x73, 0xaa, 0x02, 0x05, 0x57, 0x61, 0x76, 0x65, 0x73, 0x62, 0x06,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_waves_invoke_script_result_proto_rawDescOnce sync.Once
	file_waves_invoke_script_result_proto_rawDescData = file_waves_invoke_script_result_proto_rawDesc
)

func file_waves_invoke_script_result_proto_rawDescGZIP() []byte {
	file_waves_invoke_script_result_proto_rawDescOnce.Do(func() {
		file_waves_invoke_script_result_proto_rawDescData = protoimpl.X.CompressGZIP(file_waves_invoke_script_result_proto_rawDescData)
	})
	return file_waves_invoke_script_result_proto_rawDescData
}

var file_waves_invoke_script_result_proto_msgTypes = make([]protoimpl.MessageInfo, 7)
var file_waves_invoke_script_result_proto_goTypes = []interface{}{
	(*InvokeScriptResult)(nil),              // 0: waves.InvokeScriptResult
	(*InvokeScriptResult_Payment)(nil),      // 1: waves.InvokeScriptResult.Payment
	(*InvokeScriptResult_Issue)(nil),        // 2: waves.InvokeScriptResult.Issue
	(*InvokeScriptResult_Reissue)(nil),      // 3: waves.InvokeScriptResult.Reissue
	(*InvokeScriptResult_Burn)(nil),         // 4: waves.InvokeScriptResult.Burn
	(*InvokeScriptResult_SponsorFee)(nil),   // 5: waves.InvokeScriptResult.SponsorFee
	(*InvokeScriptResult_ErrorMessage)(nil), // 6: waves.InvokeScriptResult.ErrorMessage
	(*DataTransactionData_DataEntry)(nil),   // 7: waves.DataTransactionData.DataEntry
	(*Amount)(nil),                          // 8: waves.Amount
}
var file_waves_invoke_script_result_proto_depIdxs = []int32{
	7, // 0: waves.InvokeScriptResult.data:type_name -> waves.DataTransactionData.DataEntry
	1, // 1: waves.InvokeScriptResult.transfers:type_name -> waves.InvokeScriptResult.Payment
	2, // 2: waves.InvokeScriptResult.issues:type_name -> waves.InvokeScriptResult.Issue
	3, // 3: waves.InvokeScriptResult.reissues:type_name -> waves.InvokeScriptResult.Reissue
	4, // 4: waves.InvokeScriptResult.burns:type_name -> waves.InvokeScriptResult.Burn
	6, // 5: waves.InvokeScriptResult.error_message:type_name -> waves.InvokeScriptResult.ErrorMessage
	5, // 6: waves.InvokeScriptResult.sponsor_fees:type_name -> waves.InvokeScriptResult.SponsorFee
	8, // 7: waves.InvokeScriptResult.Payment.amount:type_name -> waves.Amount
	8, // 8: waves.InvokeScriptResult.SponsorFee.min_fee:type_name -> waves.Amount
	9, // [9:9] is the sub-list for method output_type
	9, // [9:9] is the sub-list for method input_type
	9, // [9:9] is the sub-list for extension type_name
	9, // [9:9] is the sub-list for extension extendee
	0, // [0:9] is the sub-list for field type_name
}

func init() { file_waves_invoke_script_result_proto_init() }
func file_waves_invoke_script_result_proto_init() {
	if File_waves_invoke_script_result_proto != nil {
		return
	}
	file_waves_transaction_proto_init()
	file_waves_amount_proto_init()
	if !protoimpl.UnsafeEnabled {
		file_waves_invoke_script_result_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*InvokeScriptResult); i {
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
		file_waves_invoke_script_result_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*InvokeScriptResult_Payment); i {
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
		file_waves_invoke_script_result_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*InvokeScriptResult_Issue); i {
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
		file_waves_invoke_script_result_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*InvokeScriptResult_Reissue); i {
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
		file_waves_invoke_script_result_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*InvokeScriptResult_Burn); i {
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
		file_waves_invoke_script_result_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*InvokeScriptResult_SponsorFee); i {
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
		file_waves_invoke_script_result_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*InvokeScriptResult_ErrorMessage); i {
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
			RawDescriptor: file_waves_invoke_script_result_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   7,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_waves_invoke_script_result_proto_goTypes,
		DependencyIndexes: file_waves_invoke_script_result_proto_depIdxs,
		MessageInfos:      file_waves_invoke_script_result_proto_msgTypes,
	}.Build()
	File_waves_invoke_script_result_proto = out.File
	file_waves_invoke_script_result_proto_rawDesc = nil
	file_waves_invoke_script_result_proto_goTypes = nil
	file_waves_invoke_script_result_proto_depIdxs = nil
}
