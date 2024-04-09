// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.27.1
// 	protoc        v5.26.1
// source: waves/reward_share.proto

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

type RewardShare struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Address []byte `protobuf:"bytes,1,opt,name=address,proto3" json:"address,omitempty"`
	Reward  int64  `protobuf:"varint,2,opt,name=reward,proto3" json:"reward,omitempty"`
}

func (x *RewardShare) Reset() {
	*x = RewardShare{}
	if protoimpl.UnsafeEnabled {
		mi := &file_waves_reward_share_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *RewardShare) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RewardShare) ProtoMessage() {}

func (x *RewardShare) ProtoReflect() protoreflect.Message {
	mi := &file_waves_reward_share_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RewardShare.ProtoReflect.Descriptor instead.
func (*RewardShare) Descriptor() ([]byte, []int) {
	return file_waves_reward_share_proto_rawDescGZIP(), []int{0}
}

func (x *RewardShare) GetAddress() []byte {
	if x != nil {
		return x.Address
	}
	return nil
}

func (x *RewardShare) GetReward() int64 {
	if x != nil {
		return x.Reward
	}
	return 0
}

var File_waves_reward_share_proto protoreflect.FileDescriptor

var file_waves_reward_share_proto_rawDesc = []byte{
	0x0a, 0x18, 0x77, 0x61, 0x76, 0x65, 0x73, 0x2f, 0x72, 0x65, 0x77, 0x61, 0x72, 0x64, 0x5f, 0x73,
	0x68, 0x61, 0x72, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x05, 0x77, 0x61, 0x76, 0x65,
	0x73, 0x22, 0x3f, 0x0a, 0x0b, 0x52, 0x65, 0x77, 0x61, 0x72, 0x64, 0x53, 0x68, 0x61, 0x72, 0x65,
	0x12, 0x18, 0x0a, 0x07, 0x61, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x0c, 0x52, 0x07, 0x61, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x12, 0x16, 0x0a, 0x06, 0x72, 0x65,
	0x77, 0x61, 0x72, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x03, 0x52, 0x06, 0x72, 0x65, 0x77, 0x61,
	0x72, 0x64, 0x42, 0x5f, 0x0a, 0x1a, 0x63, 0x6f, 0x6d, 0x2e, 0x77, 0x61, 0x76, 0x65, 0x73, 0x70,
	0x6c, 0x61, 0x74, 0x66, 0x6f, 0x72, 0x6d, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66,
	0x5a, 0x39, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x77, 0x61, 0x76,
	0x65, 0x73, 0x70, 0x6c, 0x61, 0x74, 0x66, 0x6f, 0x72, 0x6d, 0x2f, 0x67, 0x6f, 0x77, 0x61, 0x76,
	0x65, 0x73, 0x2f, 0x70, 0x6b, 0x67, 0x2f, 0x67, 0x72, 0x70, 0x63, 0x2f, 0x67, 0x65, 0x6e, 0x65,
	0x72, 0x61, 0x74, 0x65, 0x64, 0x2f, 0x77, 0x61, 0x76, 0x65, 0x73, 0xaa, 0x02, 0x05, 0x57, 0x61,
	0x76, 0x65, 0x73, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_waves_reward_share_proto_rawDescOnce sync.Once
	file_waves_reward_share_proto_rawDescData = file_waves_reward_share_proto_rawDesc
)

func file_waves_reward_share_proto_rawDescGZIP() []byte {
	file_waves_reward_share_proto_rawDescOnce.Do(func() {
		file_waves_reward_share_proto_rawDescData = protoimpl.X.CompressGZIP(file_waves_reward_share_proto_rawDescData)
	})
	return file_waves_reward_share_proto_rawDescData
}

var file_waves_reward_share_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_waves_reward_share_proto_goTypes = []interface{}{
	(*RewardShare)(nil), // 0: waves.RewardShare
}
var file_waves_reward_share_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_waves_reward_share_proto_init() }
func file_waves_reward_share_proto_init() {
	if File_waves_reward_share_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_waves_reward_share_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*RewardShare); i {
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
			RawDescriptor: file_waves_reward_share_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_waves_reward_share_proto_goTypes,
		DependencyIndexes: file_waves_reward_share_proto_depIdxs,
		MessageInfos:      file_waves_reward_share_proto_msgTypes,
	}.Build()
	File_waves_reward_share_proto = out.File
	file_waves_reward_share_proto_rawDesc = nil
	file_waves_reward_share_proto_goTypes = nil
	file_waves_reward_share_proto_depIdxs = nil
}
