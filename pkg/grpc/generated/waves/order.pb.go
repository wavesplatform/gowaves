// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.27.1
// 	protoc        v4.23.3
// source: waves/order.proto

package waves

import (
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

type Order_Side int32

const (
	Order_BUY  Order_Side = 0
	Order_SELL Order_Side = 1
)

// Enum value maps for Order_Side.
var (
	Order_Side_name = map[int32]string{
		0: "BUY",
		1: "SELL",
	}
	Order_Side_value = map[string]int32{
		"BUY":  0,
		"SELL": 1,
	}
)

func (x Order_Side) Enum() *Order_Side {
	p := new(Order_Side)
	*p = x
	return p
}

func (x Order_Side) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Order_Side) Descriptor() protoreflect.EnumDescriptor {
	return file_waves_order_proto_enumTypes[0].Descriptor()
}

func (Order_Side) Type() protoreflect.EnumType {
	return &file_waves_order_proto_enumTypes[0]
}

func (x Order_Side) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use Order_Side.Descriptor instead.
func (Order_Side) EnumDescriptor() ([]byte, []int) {
	return file_waves_order_proto_rawDescGZIP(), []int{1, 0}
}

type Order_PriceMode int32

const (
	Order_DEFAULT        Order_PriceMode = 0
	Order_FIXED_DECIMALS Order_PriceMode = 1
	Order_ASSET_DECIMALS Order_PriceMode = 2
)

// Enum value maps for Order_PriceMode.
var (
	Order_PriceMode_name = map[int32]string{
		0: "DEFAULT",
		1: "FIXED_DECIMALS",
		2: "ASSET_DECIMALS",
	}
	Order_PriceMode_value = map[string]int32{
		"DEFAULT":        0,
		"FIXED_DECIMALS": 1,
		"ASSET_DECIMALS": 2,
	}
)

func (x Order_PriceMode) Enum() *Order_PriceMode {
	p := new(Order_PriceMode)
	*p = x
	return p
}

func (x Order_PriceMode) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Order_PriceMode) Descriptor() protoreflect.EnumDescriptor {
	return file_waves_order_proto_enumTypes[1].Descriptor()
}

func (Order_PriceMode) Type() protoreflect.EnumType {
	return &file_waves_order_proto_enumTypes[1]
}

func (x Order_PriceMode) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use Order_PriceMode.Descriptor instead.
func (Order_PriceMode) EnumDescriptor() ([]byte, []int) {
	return file_waves_order_proto_rawDescGZIP(), []int{1, 1}
}

type AssetPair struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	AmountAssetId []byte `protobuf:"bytes,1,opt,name=amount_asset_id,json=amountAssetId,proto3" json:"amount_asset_id,omitempty"`
	PriceAssetId  []byte `protobuf:"bytes,2,opt,name=price_asset_id,json=priceAssetId,proto3" json:"price_asset_id,omitempty"`
}

func (x *AssetPair) Reset() {
	*x = AssetPair{}
	if protoimpl.UnsafeEnabled {
		mi := &file_waves_order_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *AssetPair) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AssetPair) ProtoMessage() {}

func (x *AssetPair) ProtoReflect() protoreflect.Message {
	mi := &file_waves_order_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AssetPair.ProtoReflect.Descriptor instead.
func (*AssetPair) Descriptor() ([]byte, []int) {
	return file_waves_order_proto_rawDescGZIP(), []int{0}
}

func (x *AssetPair) GetAmountAssetId() []byte {
	if x != nil {
		return x.AmountAssetId
	}
	return nil
}

func (x *AssetPair) GetPriceAssetId() []byte {
	if x != nil {
		return x.PriceAssetId
	}
	return nil
}

type Order struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	ChainId          int32           `protobuf:"varint,1,opt,name=chain_id,json=chainId,proto3" json:"chain_id,omitempty"`
	MatcherPublicKey []byte          `protobuf:"bytes,3,opt,name=matcher_public_key,json=matcherPublicKey,proto3" json:"matcher_public_key,omitempty"`
	AssetPair        *AssetPair      `protobuf:"bytes,4,opt,name=asset_pair,json=assetPair,proto3" json:"asset_pair,omitempty"`
	OrderSide        Order_Side      `protobuf:"varint,5,opt,name=order_side,json=orderSide,proto3,enum=waves.Order_Side" json:"order_side,omitempty"`
	Amount           int64           `protobuf:"varint,6,opt,name=amount,proto3" json:"amount,omitempty"`
	Price            int64           `protobuf:"varint,7,opt,name=price,proto3" json:"price,omitempty"`
	Timestamp        int64           `protobuf:"varint,8,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	Expiration       int64           `protobuf:"varint,9,opt,name=expiration,proto3" json:"expiration,omitempty"`
	MatcherFee       *Amount         `protobuf:"bytes,10,opt,name=matcher_fee,json=matcherFee,proto3" json:"matcher_fee,omitempty"`
	Version          int32           `protobuf:"varint,11,opt,name=version,proto3" json:"version,omitempty"`
	Proofs           [][]byte        `protobuf:"bytes,12,rep,name=proofs,proto3" json:"proofs,omitempty"`
	PriceMode        Order_PriceMode `protobuf:"varint,14,opt,name=price_mode,json=priceMode,proto3,enum=waves.Order_PriceMode" json:"price_mode,omitempty"`
	Attachment       []byte          `protobuf:"bytes,15,opt,name=attachment,proto3" json:"attachment,omitempty"`
	// Types that are assignable to Sender:
	//
	//	*Order_SenderPublicKey
	//	*Order_Eip712Signature
	Sender isOrder_Sender `protobuf_oneof:"sender"`
}

func (x *Order) Reset() {
	*x = Order{}
	if protoimpl.UnsafeEnabled {
		mi := &file_waves_order_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Order) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Order) ProtoMessage() {}

func (x *Order) ProtoReflect() protoreflect.Message {
	mi := &file_waves_order_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Order.ProtoReflect.Descriptor instead.
func (*Order) Descriptor() ([]byte, []int) {
	return file_waves_order_proto_rawDescGZIP(), []int{1}
}

func (x *Order) GetChainId() int32 {
	if x != nil {
		return x.ChainId
	}
	return 0
}

func (x *Order) GetMatcherPublicKey() []byte {
	if x != nil {
		return x.MatcherPublicKey
	}
	return nil
}

func (x *Order) GetAssetPair() *AssetPair {
	if x != nil {
		return x.AssetPair
	}
	return nil
}

func (x *Order) GetOrderSide() Order_Side {
	if x != nil {
		return x.OrderSide
	}
	return Order_BUY
}

func (x *Order) GetAmount() int64 {
	if x != nil {
		return x.Amount
	}
	return 0
}

func (x *Order) GetPrice() int64 {
	if x != nil {
		return x.Price
	}
	return 0
}

func (x *Order) GetTimestamp() int64 {
	if x != nil {
		return x.Timestamp
	}
	return 0
}

func (x *Order) GetExpiration() int64 {
	if x != nil {
		return x.Expiration
	}
	return 0
}

func (x *Order) GetMatcherFee() *Amount {
	if x != nil {
		return x.MatcherFee
	}
	return nil
}

func (x *Order) GetVersion() int32 {
	if x != nil {
		return x.Version
	}
	return 0
}

func (x *Order) GetProofs() [][]byte {
	if x != nil {
		return x.Proofs
	}
	return nil
}

func (x *Order) GetPriceMode() Order_PriceMode {
	if x != nil {
		return x.PriceMode
	}
	return Order_DEFAULT
}

func (x *Order) GetAttachment() []byte {
	if x != nil {
		return x.Attachment
	}
	return nil
}

func (m *Order) GetSender() isOrder_Sender {
	if m != nil {
		return m.Sender
	}
	return nil
}

func (x *Order) GetSenderPublicKey() []byte {
	if x, ok := x.GetSender().(*Order_SenderPublicKey); ok {
		return x.SenderPublicKey
	}
	return nil
}

func (x *Order) GetEip712Signature() []byte {
	if x, ok := x.GetSender().(*Order_Eip712Signature); ok {
		return x.Eip712Signature
	}
	return nil
}

type isOrder_Sender interface {
	isOrder_Sender()
}

type Order_SenderPublicKey struct {
	SenderPublicKey []byte `protobuf:"bytes,2,opt,name=sender_public_key,json=senderPublicKey,proto3,oneof"`
}

type Order_Eip712Signature struct {
	Eip712Signature []byte `protobuf:"bytes,13,opt,name=eip712_signature,json=eip712Signature,proto3,oneof"`
}

func (*Order_SenderPublicKey) isOrder_Sender() {}

func (*Order_Eip712Signature) isOrder_Sender() {}

var File_waves_order_proto protoreflect.FileDescriptor

var file_waves_order_proto_rawDesc = []byte{
	0x0a, 0x11, 0x77, 0x61, 0x76, 0x65, 0x73, 0x2f, 0x6f, 0x72, 0x64, 0x65, 0x72, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x12, 0x05, 0x77, 0x61, 0x76, 0x65, 0x73, 0x1a, 0x12, 0x77, 0x61, 0x76, 0x65,
	0x73, 0x2f, 0x61, 0x6d, 0x6f, 0x75, 0x6e, 0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x59,
	0x0a, 0x09, 0x41, 0x73, 0x73, 0x65, 0x74, 0x50, 0x61, 0x69, 0x72, 0x12, 0x26, 0x0a, 0x0f, 0x61,
	0x6d, 0x6f, 0x75, 0x6e, 0x74, 0x5f, 0x61, 0x73, 0x73, 0x65, 0x74, 0x5f, 0x69, 0x64, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x0c, 0x52, 0x0d, 0x61, 0x6d, 0x6f, 0x75, 0x6e, 0x74, 0x41, 0x73, 0x73, 0x65,
	0x74, 0x49, 0x64, 0x12, 0x24, 0x0a, 0x0e, 0x70, 0x72, 0x69, 0x63, 0x65, 0x5f, 0x61, 0x73, 0x73,
	0x65, 0x74, 0x5f, 0x69, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x0c, 0x70, 0x72, 0x69,
	0x63, 0x65, 0x41, 0x73, 0x73, 0x65, 0x74, 0x49, 0x64, 0x22, 0x9a, 0x05, 0x0a, 0x05, 0x4f, 0x72,
	0x64, 0x65, 0x72, 0x12, 0x19, 0x0a, 0x08, 0x63, 0x68, 0x61, 0x69, 0x6e, 0x5f, 0x69, 0x64, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x05, 0x52, 0x07, 0x63, 0x68, 0x61, 0x69, 0x6e, 0x49, 0x64, 0x12, 0x2c,
	0x0a, 0x12, 0x6d, 0x61, 0x74, 0x63, 0x68, 0x65, 0x72, 0x5f, 0x70, 0x75, 0x62, 0x6c, 0x69, 0x63,
	0x5f, 0x6b, 0x65, 0x79, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x10, 0x6d, 0x61, 0x74, 0x63,
	0x68, 0x65, 0x72, 0x50, 0x75, 0x62, 0x6c, 0x69, 0x63, 0x4b, 0x65, 0x79, 0x12, 0x2f, 0x0a, 0x0a,
	0x61, 0x73, 0x73, 0x65, 0x74, 0x5f, 0x70, 0x61, 0x69, 0x72, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x10, 0x2e, 0x77, 0x61, 0x76, 0x65, 0x73, 0x2e, 0x41, 0x73, 0x73, 0x65, 0x74, 0x50, 0x61,
	0x69, 0x72, 0x52, 0x09, 0x61, 0x73, 0x73, 0x65, 0x74, 0x50, 0x61, 0x69, 0x72, 0x12, 0x30, 0x0a,
	0x0a, 0x6f, 0x72, 0x64, 0x65, 0x72, 0x5f, 0x73, 0x69, 0x64, 0x65, 0x18, 0x05, 0x20, 0x01, 0x28,
	0x0e, 0x32, 0x11, 0x2e, 0x77, 0x61, 0x76, 0x65, 0x73, 0x2e, 0x4f, 0x72, 0x64, 0x65, 0x72, 0x2e,
	0x53, 0x69, 0x64, 0x65, 0x52, 0x09, 0x6f, 0x72, 0x64, 0x65, 0x72, 0x53, 0x69, 0x64, 0x65, 0x12,
	0x16, 0x0a, 0x06, 0x61, 0x6d, 0x6f, 0x75, 0x6e, 0x74, 0x18, 0x06, 0x20, 0x01, 0x28, 0x03, 0x52,
	0x06, 0x61, 0x6d, 0x6f, 0x75, 0x6e, 0x74, 0x12, 0x14, 0x0a, 0x05, 0x70, 0x72, 0x69, 0x63, 0x65,
	0x18, 0x07, 0x20, 0x01, 0x28, 0x03, 0x52, 0x05, 0x70, 0x72, 0x69, 0x63, 0x65, 0x12, 0x1c, 0x0a,
	0x09, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x18, 0x08, 0x20, 0x01, 0x28, 0x03,
	0x52, 0x09, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x12, 0x1e, 0x0a, 0x0a, 0x65,
	0x78, 0x70, 0x69, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x09, 0x20, 0x01, 0x28, 0x03, 0x52,
	0x0a, 0x65, 0x78, 0x70, 0x69, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x2e, 0x0a, 0x0b, 0x6d,
	0x61, 0x74, 0x63, 0x68, 0x65, 0x72, 0x5f, 0x66, 0x65, 0x65, 0x18, 0x0a, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x0d, 0x2e, 0x77, 0x61, 0x76, 0x65, 0x73, 0x2e, 0x41, 0x6d, 0x6f, 0x75, 0x6e, 0x74, 0x52,
	0x0a, 0x6d, 0x61, 0x74, 0x63, 0x68, 0x65, 0x72, 0x46, 0x65, 0x65, 0x12, 0x18, 0x0a, 0x07, 0x76,
	0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x18, 0x0b, 0x20, 0x01, 0x28, 0x05, 0x52, 0x07, 0x76, 0x65,
	0x72, 0x73, 0x69, 0x6f, 0x6e, 0x12, 0x16, 0x0a, 0x06, 0x70, 0x72, 0x6f, 0x6f, 0x66, 0x73, 0x18,
	0x0c, 0x20, 0x03, 0x28, 0x0c, 0x52, 0x06, 0x70, 0x72, 0x6f, 0x6f, 0x66, 0x73, 0x12, 0x35, 0x0a,
	0x0a, 0x70, 0x72, 0x69, 0x63, 0x65, 0x5f, 0x6d, 0x6f, 0x64, 0x65, 0x18, 0x0e, 0x20, 0x01, 0x28,
	0x0e, 0x32, 0x16, 0x2e, 0x77, 0x61, 0x76, 0x65, 0x73, 0x2e, 0x4f, 0x72, 0x64, 0x65, 0x72, 0x2e,
	0x50, 0x72, 0x69, 0x63, 0x65, 0x4d, 0x6f, 0x64, 0x65, 0x52, 0x09, 0x70, 0x72, 0x69, 0x63, 0x65,
	0x4d, 0x6f, 0x64, 0x65, 0x12, 0x1e, 0x0a, 0x0a, 0x61, 0x74, 0x74, 0x61, 0x63, 0x68, 0x6d, 0x65,
	0x6e, 0x74, 0x18, 0x0f, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x0a, 0x61, 0x74, 0x74, 0x61, 0x63, 0x68,
	0x6d, 0x65, 0x6e, 0x74, 0x12, 0x2c, 0x0a, 0x11, 0x73, 0x65, 0x6e, 0x64, 0x65, 0x72, 0x5f, 0x70,
	0x75, 0x62, 0x6c, 0x69, 0x63, 0x5f, 0x6b, 0x65, 0x79, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0c, 0x48,
	0x00, 0x52, 0x0f, 0x73, 0x65, 0x6e, 0x64, 0x65, 0x72, 0x50, 0x75, 0x62, 0x6c, 0x69, 0x63, 0x4b,
	0x65, 0x79, 0x12, 0x2b, 0x0a, 0x10, 0x65, 0x69, 0x70, 0x37, 0x31, 0x32, 0x5f, 0x73, 0x69, 0x67,
	0x6e, 0x61, 0x74, 0x75, 0x72, 0x65, 0x18, 0x0d, 0x20, 0x01, 0x28, 0x0c, 0x48, 0x00, 0x52, 0x0f,
	0x65, 0x69, 0x70, 0x37, 0x31, 0x32, 0x53, 0x69, 0x67, 0x6e, 0x61, 0x74, 0x75, 0x72, 0x65, 0x22,
	0x19, 0x0a, 0x04, 0x53, 0x69, 0x64, 0x65, 0x12, 0x07, 0x0a, 0x03, 0x42, 0x55, 0x59, 0x10, 0x00,
	0x12, 0x08, 0x0a, 0x04, 0x53, 0x45, 0x4c, 0x4c, 0x10, 0x01, 0x22, 0x40, 0x0a, 0x09, 0x50, 0x72,
	0x69, 0x63, 0x65, 0x4d, 0x6f, 0x64, 0x65, 0x12, 0x0b, 0x0a, 0x07, 0x44, 0x45, 0x46, 0x41, 0x55,
	0x4c, 0x54, 0x10, 0x00, 0x12, 0x12, 0x0a, 0x0e, 0x46, 0x49, 0x58, 0x45, 0x44, 0x5f, 0x44, 0x45,
	0x43, 0x49, 0x4d, 0x41, 0x4c, 0x53, 0x10, 0x01, 0x12, 0x12, 0x0a, 0x0e, 0x41, 0x53, 0x53, 0x45,
	0x54, 0x5f, 0x44, 0x45, 0x43, 0x49, 0x4d, 0x41, 0x4c, 0x53, 0x10, 0x02, 0x42, 0x08, 0x0a, 0x06,
	0x73, 0x65, 0x6e, 0x64, 0x65, 0x72, 0x42, 0x65, 0x0a, 0x20, 0x63, 0x6f, 0x6d, 0x2e, 0x77, 0x61,
	0x76, 0x65, 0x73, 0x70, 0x6c, 0x61, 0x74, 0x66, 0x6f, 0x72, 0x6d, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x62, 0x75, 0x66, 0x2e, 0x6f, 0x72, 0x64, 0x65, 0x72, 0x5a, 0x39, 0x67, 0x69, 0x74, 0x68,
	0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x77, 0x61, 0x76, 0x65, 0x73, 0x70, 0x6c, 0x61, 0x74,
	0x66, 0x6f, 0x72, 0x6d, 0x2f, 0x67, 0x6f, 0x77, 0x61, 0x76, 0x65, 0x73, 0x2f, 0x70, 0x6b, 0x67,
	0x2f, 0x67, 0x72, 0x70, 0x63, 0x2f, 0x67, 0x65, 0x6e, 0x65, 0x72, 0x61, 0x74, 0x65, 0x64, 0x2f,
	0x77, 0x61, 0x76, 0x65, 0x73, 0xaa, 0x02, 0x05, 0x57, 0x61, 0x76, 0x65, 0x73, 0x62, 0x06, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_waves_order_proto_rawDescOnce sync.Once
	file_waves_order_proto_rawDescData = file_waves_order_proto_rawDesc
)

func file_waves_order_proto_rawDescGZIP() []byte {
	file_waves_order_proto_rawDescOnce.Do(func() {
		file_waves_order_proto_rawDescData = protoimpl.X.CompressGZIP(file_waves_order_proto_rawDescData)
	})
	return file_waves_order_proto_rawDescData
}

var file_waves_order_proto_enumTypes = make([]protoimpl.EnumInfo, 2)
var file_waves_order_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_waves_order_proto_goTypes = []interface{}{
	(Order_Side)(0),      // 0: waves.Order.Side
	(Order_PriceMode)(0), // 1: waves.Order.PriceMode
	(*AssetPair)(nil),    // 2: waves.AssetPair
	(*Order)(nil),        // 3: waves.Order
	(*Amount)(nil),       // 4: waves.Amount
}
var file_waves_order_proto_depIdxs = []int32{
	2, // 0: waves.Order.asset_pair:type_name -> waves.AssetPair
	0, // 1: waves.Order.order_side:type_name -> waves.Order.Side
	4, // 2: waves.Order.matcher_fee:type_name -> waves.Amount
	1, // 3: waves.Order.price_mode:type_name -> waves.Order.PriceMode
	4, // [4:4] is the sub-list for method output_type
	4, // [4:4] is the sub-list for method input_type
	4, // [4:4] is the sub-list for extension type_name
	4, // [4:4] is the sub-list for extension extendee
	0, // [0:4] is the sub-list for field type_name
}

func init() { file_waves_order_proto_init() }
func file_waves_order_proto_init() {
	if File_waves_order_proto != nil {
		return
	}
	file_waves_amount_proto_init()
	if !protoimpl.UnsafeEnabled {
		file_waves_order_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*AssetPair); i {
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
		file_waves_order_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Order); i {
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
	file_waves_order_proto_msgTypes[1].OneofWrappers = []interface{}{
		(*Order_SenderPublicKey)(nil),
		(*Order_Eip712Signature)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_waves_order_proto_rawDesc,
			NumEnums:      2,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_waves_order_proto_goTypes,
		DependencyIndexes: file_waves_order_proto_depIdxs,
		EnumInfos:         file_waves_order_proto_enumTypes,
		MessageInfos:      file_waves_order_proto_msgTypes,
	}.Build()
	File_waves_order_proto = out.File
	file_waves_order_proto_rawDesc = nil
	file_waves_order_proto_goTypes = nil
	file_waves_order_proto_depIdxs = nil
}
