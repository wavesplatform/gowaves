// Code generated by protoc-gen-go. DO NOT EDIT.
// source: order.proto

package grpc

import (
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	math "math"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion3 // please upgrade the proto package

type Order_Side int32

const (
	Order_BUY  Order_Side = 0
	Order_SELL Order_Side = 1
)

var Order_Side_name = map[int32]string{
	0: "BUY",
	1: "SELL",
}

var Order_Side_value = map[string]int32{
	"BUY":  0,
	"SELL": 1,
}

func (x Order_Side) String() string {
	return proto.EnumName(Order_Side_name, int32(x))
}

func (Order_Side) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_cd01338c35d87077, []int{1, 0}
}

type AssetPair struct {
	AmountAssetId        []byte   `protobuf:"bytes,1,opt,name=amount_asset_id,json=amountAssetId,proto3" json:"amount_asset_id,omitempty"`
	PriceAssetId         []byte   `protobuf:"bytes,2,opt,name=price_asset_id,json=priceAssetId,proto3" json:"price_asset_id,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *AssetPair) Reset()         { *m = AssetPair{} }
func (m *AssetPair) String() string { return proto.CompactTextString(m) }
func (*AssetPair) ProtoMessage()    {}
func (*AssetPair) Descriptor() ([]byte, []int) {
	return fileDescriptor_cd01338c35d87077, []int{0}
}

func (m *AssetPair) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_AssetPair.Unmarshal(m, b)
}
func (m *AssetPair) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_AssetPair.Marshal(b, m, deterministic)
}
func (m *AssetPair) XXX_Merge(src proto.Message) {
	xxx_messageInfo_AssetPair.Merge(m, src)
}
func (m *AssetPair) XXX_Size() int {
	return xxx_messageInfo_AssetPair.Size(m)
}
func (m *AssetPair) XXX_DiscardUnknown() {
	xxx_messageInfo_AssetPair.DiscardUnknown(m)
}

var xxx_messageInfo_AssetPair proto.InternalMessageInfo

func (m *AssetPair) GetAmountAssetId() []byte {
	if m != nil {
		return m.AmountAssetId
	}
	return nil
}

func (m *AssetPair) GetPriceAssetId() []byte {
	if m != nil {
		return m.PriceAssetId
	}
	return nil
}

type Order struct {
	ChainId              int32      `protobuf:"varint,1,opt,name=chain_id,json=chainId,proto3" json:"chain_id,omitempty"`
	SenderPublicKey      []byte     `protobuf:"bytes,2,opt,name=sender_public_key,json=senderPublicKey,proto3" json:"sender_public_key,omitempty"`
	MatcherPublicKey     []byte     `protobuf:"bytes,3,opt,name=matcher_public_key,json=matcherPublicKey,proto3" json:"matcher_public_key,omitempty"`
	AssetPair            *AssetPair `protobuf:"bytes,4,opt,name=asset_pair,json=assetPair,proto3" json:"asset_pair,omitempty"`
	OrderSide            Order_Side `protobuf:"varint,5,opt,name=order_side,json=orderSide,proto3,enum=waves.Order_Side" json:"order_side,omitempty"`
	Amount               int64      `protobuf:"varint,6,opt,name=amount,proto3" json:"amount,omitempty"`
	Price                int64      `protobuf:"varint,7,opt,name=price,proto3" json:"price,omitempty"`
	Timestamp            int64      `protobuf:"varint,8,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	Expiration           int64      `protobuf:"varint,9,opt,name=expiration,proto3" json:"expiration,omitempty"`
	MatcherFee           *Amount    `protobuf:"bytes,10,opt,name=matcher_fee,json=matcherFee,proto3" json:"matcher_fee,omitempty"`
	Version              int32      `protobuf:"varint,11,opt,name=version,proto3" json:"version,omitempty"`
	Proofs               [][]byte   `protobuf:"bytes,12,rep,name=proofs,proto3" json:"proofs,omitempty"`
	XXX_NoUnkeyedLiteral struct{}   `json:"-"`
	XXX_unrecognized     []byte     `json:"-"`
	XXX_sizecache        int32      `json:"-"`
}

func (m *Order) Reset()         { *m = Order{} }
func (m *Order) String() string { return proto.CompactTextString(m) }
func (*Order) ProtoMessage()    {}
func (*Order) Descriptor() ([]byte, []int) {
	return fileDescriptor_cd01338c35d87077, []int{1}
}

func (m *Order) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Order.Unmarshal(m, b)
}
func (m *Order) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Order.Marshal(b, m, deterministic)
}
func (m *Order) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Order.Merge(m, src)
}
func (m *Order) XXX_Size() int {
	return xxx_messageInfo_Order.Size(m)
}
func (m *Order) XXX_DiscardUnknown() {
	xxx_messageInfo_Order.DiscardUnknown(m)
}

var xxx_messageInfo_Order proto.InternalMessageInfo

func (m *Order) GetChainId() int32 {
	if m != nil {
		return m.ChainId
	}
	return 0
}

func (m *Order) GetSenderPublicKey() []byte {
	if m != nil {
		return m.SenderPublicKey
	}
	return nil
}

func (m *Order) GetMatcherPublicKey() []byte {
	if m != nil {
		return m.MatcherPublicKey
	}
	return nil
}

func (m *Order) GetAssetPair() *AssetPair {
	if m != nil {
		return m.AssetPair
	}
	return nil
}

func (m *Order) GetOrderSide() Order_Side {
	if m != nil {
		return m.OrderSide
	}
	return Order_BUY
}

func (m *Order) GetAmount() int64 {
	if m != nil {
		return m.Amount
	}
	return 0
}

func (m *Order) GetPrice() int64 {
	if m != nil {
		return m.Price
	}
	return 0
}

func (m *Order) GetTimestamp() int64 {
	if m != nil {
		return m.Timestamp
	}
	return 0
}

func (m *Order) GetExpiration() int64 {
	if m != nil {
		return m.Expiration
	}
	return 0
}

func (m *Order) GetMatcherFee() *Amount {
	if m != nil {
		return m.MatcherFee
	}
	return nil
}

func (m *Order) GetVersion() int32 {
	if m != nil {
		return m.Version
	}
	return 0
}

func (m *Order) GetProofs() [][]byte {
	if m != nil {
		return m.Proofs
	}
	return nil
}

func init() {
	proto.RegisterEnum("waves.Order_Side", Order_Side_name, Order_Side_value)
	proto.RegisterType((*AssetPair)(nil), "waves.AssetPair")
	proto.RegisterType((*Order)(nil), "waves.Order")
}

func init() { proto.RegisterFile("order.proto", fileDescriptor_cd01338c35d87077) }

var fileDescriptor_cd01338c35d87077 = []byte{
	// 413 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x54, 0x92, 0x51, 0x6b, 0xd4, 0x40,
	0x14, 0x85, 0x4d, 0xb3, 0xd9, 0xdd, 0xdc, 0xa4, 0xed, 0x76, 0x10, 0x99, 0x8a, 0x48, 0x58, 0x44,
	0x82, 0x48, 0x2a, 0xf5, 0xc1, 0x67, 0x0b, 0x0a, 0xc5, 0x82, 0x25, 0x45, 0xa4, 0xbe, 0x84, 0xd9,
	0xe4, 0xc6, 0x0e, 0x6e, 0x32, 0xc3, 0x64, 0xb6, 0xda, 0xbf, 0xe4, 0x2f, 0xf1, 0x67, 0x49, 0xee,
	0xcc, 0xee, 0xea, 0xdb, 0xde, 0x73, 0xbe, 0xbd, 0x77, 0xce, 0x21, 0x90, 0x28, 0xd3, 0xa0, 0x29,
	0xb4, 0x51, 0x56, 0xb1, 0xe8, 0xa7, 0xb8, 0xc7, 0xe1, 0x69, 0x2a, 0x3a, 0xb5, 0xe9, 0xad, 0x13,
	0x97, 0xb7, 0x10, 0xbf, 0x1f, 0x06, 0xb4, 0xd7, 0x42, 0x1a, 0xf6, 0x12, 0x8e, 0x9d, 0x59, 0x89,
	0x51, 0xab, 0x64, 0xc3, 0x83, 0x2c, 0xc8, 0xd3, 0xf2, 0xd0, 0xc9, 0x44, 0x5e, 0x36, 0xec, 0x05,
	0x1c, 0x69, 0x23, 0x6b, 0xdc, 0x63, 0x07, 0x84, 0xa5, 0xa4, 0x7a, 0x6a, 0xf9, 0x27, 0x84, 0xe8,
	0xf3, 0x78, 0x9f, 0x9d, 0xc2, 0xbc, 0xbe, 0x13, 0xb2, 0xdf, 0x2e, 0x8c, 0xca, 0x19, 0xcd, 0x97,
	0x0d, 0x7b, 0x05, 0x27, 0x03, 0xf6, 0x0d, 0x9a, 0x4a, 0x6f, 0x56, 0x6b, 0x59, 0x57, 0x3f, 0xf0,
	0xc1, 0x6f, 0x3b, 0x76, 0xc6, 0x35, 0xe9, 0x9f, 0xf0, 0x81, 0xbd, 0x06, 0xd6, 0x09, 0x5b, 0xdf,
	0xfd, 0x0f, 0x87, 0x04, 0x2f, 0xbc, 0xb3, 0xa7, 0xcf, 0x00, 0xdc, 0xf3, 0xb4, 0x90, 0x86, 0x4f,
	0xb2, 0x20, 0x4f, 0xce, 0x17, 0x05, 0x75, 0x50, 0xec, 0x22, 0x97, 0xb1, 0xd8, 0xa5, 0x7f, 0x03,
	0x40, 0x75, 0x55, 0x83, 0x6c, 0x90, 0x47, 0x59, 0x90, 0x1f, 0x9d, 0x9f, 0xf8, 0x3f, 0x50, 0x8e,
	0xe2, 0x46, 0x36, 0x58, 0xc6, 0x04, 0x8d, 0x3f, 0xd9, 0x13, 0x98, 0xba, 0x62, 0xf8, 0x34, 0x0b,
	0xf2, 0xb0, 0xf4, 0x13, 0x7b, 0x0c, 0x11, 0x35, 0xc1, 0x67, 0x24, 0xbb, 0x81, 0x3d, 0x83, 0xd8,
	0xca, 0x0e, 0x07, 0x2b, 0x3a, 0xcd, 0xe7, 0xe4, 0xec, 0x05, 0xf6, 0x1c, 0x00, 0x7f, 0x69, 0x69,
	0x84, 0x95, 0xaa, 0xe7, 0x31, 0xd9, 0xff, 0x28, 0xac, 0x80, 0x64, 0x1b, 0xbe, 0x45, 0xe4, 0x40,
	0x79, 0x0e, 0xb7, 0x79, 0xe8, 0x6e, 0x09, 0x9e, 0xf8, 0x88, 0xc8, 0x38, 0xcc, 0xee, 0xd1, 0x0c,
	0xe3, 0xb2, 0xc4, 0x55, 0xee, 0xc7, 0xf1, 0xd5, 0xda, 0x28, 0xd5, 0x0e, 0x3c, 0xcd, 0xc2, 0x3c,
	0x2d, 0xfd, 0xb4, 0x3c, 0x85, 0x09, 0xa5, 0x9a, 0x41, 0x78, 0xf1, 0xe5, 0x76, 0xf1, 0x88, 0xcd,
	0x61, 0x72, 0xf3, 0xe1, 0xea, 0x6a, 0x11, 0x5c, 0xbc, 0x83, 0xac, 0x56, 0x9d, 0x3b, 0xa6, 0xd7,
	0xc2, 0xb6, 0xca, 0x74, 0xee, 0x03, 0x5a, 0x6d, 0xda, 0x82, 0x0a, 0xf9, 0x96, 0xd4, 0x6b, 0x89,
	0xbd, 0x3d, 0xfb, 0x6e, 0x74, 0xfd, 0xfb, 0x20, 0xfa, 0x3a, 0xb2, 0xab, 0x29, 0x41, 0x6f, 0xff,
	0x06, 0x00, 0x00, 0xff, 0xff, 0x4e, 0x6a, 0x99, 0x47, 0x89, 0x02, 0x00, 0x00,
}
