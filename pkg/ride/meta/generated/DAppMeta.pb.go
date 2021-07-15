// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.25.0
// 	protoc        v3.17.3
// source: DAppMeta.proto

package generated

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

type DAppMeta struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Version                            int32                                      `protobuf:"varint,1,opt,name=version,proto3" json:"version,omitempty"`
	Funcs                              []*DAppMeta_CallableFuncSignature          `protobuf:"bytes,2,rep,name=funcs,proto3" json:"funcs,omitempty"`
	CompactNameAndOriginalNamePairList []*DAppMeta_CompactNameAndOriginalNamePair `protobuf:"bytes,3,rep,name=compactNameAndOriginalNamePairList,proto3" json:"compactNameAndOriginalNamePairList,omitempty"`
}

func (x *DAppMeta) Reset() {
	*x = DAppMeta{}
	if protoimpl.UnsafeEnabled {
		mi := &file_DAppMeta_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *DAppMeta) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DAppMeta) ProtoMessage() {}

func (x *DAppMeta) ProtoReflect() protoreflect.Message {
	mi := &file_DAppMeta_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DAppMeta.ProtoReflect.Descriptor instead.
func (*DAppMeta) Descriptor() ([]byte, []int) {
	return file_DAppMeta_proto_rawDescGZIP(), []int{0}
}

func (x *DAppMeta) GetVersion() int32 {
	if x != nil {
		return x.Version
	}
	return 0
}

func (x *DAppMeta) GetFuncs() []*DAppMeta_CallableFuncSignature {
	if x != nil {
		return x.Funcs
	}
	return nil
}

func (x *DAppMeta) GetCompactNameAndOriginalNamePairList() []*DAppMeta_CompactNameAndOriginalNamePair {
	if x != nil {
		return x.CompactNameAndOriginalNamePairList
	}
	return nil
}

type DAppMeta_CallableFuncSignature struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Types []byte `protobuf:"bytes,1,opt,name=types,proto3" json:"types,omitempty"`
}

func (x *DAppMeta_CallableFuncSignature) Reset() {
	*x = DAppMeta_CallableFuncSignature{}
	if protoimpl.UnsafeEnabled {
		mi := &file_DAppMeta_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *DAppMeta_CallableFuncSignature) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DAppMeta_CallableFuncSignature) ProtoMessage() {}

func (x *DAppMeta_CallableFuncSignature) ProtoReflect() protoreflect.Message {
	mi := &file_DAppMeta_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DAppMeta_CallableFuncSignature.ProtoReflect.Descriptor instead.
func (*DAppMeta_CallableFuncSignature) Descriptor() ([]byte, []int) {
	return file_DAppMeta_proto_rawDescGZIP(), []int{0, 0}
}

func (x *DAppMeta_CallableFuncSignature) GetTypes() []byte {
	if x != nil {
		return x.Types
	}
	return nil
}

type DAppMeta_CompactNameAndOriginalNamePair struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	CompactName  string `protobuf:"bytes,1,opt,name=compactName,proto3" json:"compactName,omitempty"`
	OriginalName string `protobuf:"bytes,2,opt,name=originalName,proto3" json:"originalName,omitempty"`
}

func (x *DAppMeta_CompactNameAndOriginalNamePair) Reset() {
	*x = DAppMeta_CompactNameAndOriginalNamePair{}
	if protoimpl.UnsafeEnabled {
		mi := &file_DAppMeta_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *DAppMeta_CompactNameAndOriginalNamePair) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DAppMeta_CompactNameAndOriginalNamePair) ProtoMessage() {}

func (x *DAppMeta_CompactNameAndOriginalNamePair) ProtoReflect() protoreflect.Message {
	mi := &file_DAppMeta_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DAppMeta_CompactNameAndOriginalNamePair.ProtoReflect.Descriptor instead.
func (*DAppMeta_CompactNameAndOriginalNamePair) Descriptor() ([]byte, []int) {
	return file_DAppMeta_proto_rawDescGZIP(), []int{0, 1}
}

func (x *DAppMeta_CompactNameAndOriginalNamePair) GetCompactName() string {
	if x != nil {
		return x.CompactName
	}
	return ""
}

func (x *DAppMeta_CompactNameAndOriginalNamePair) GetOriginalName() string {
	if x != nil {
		return x.OriginalName
	}
	return ""
}

var File_DAppMeta_proto protoreflect.FileDescriptor

var file_DAppMeta_proto_rawDesc = []byte{
	0x0a, 0x0e, 0x44, 0x41, 0x70, 0x70, 0x4d, 0x65, 0x74, 0x61, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x12, 0x05, 0x77, 0x61, 0x76, 0x65, 0x73, 0x22, 0xf8, 0x02, 0x0a, 0x08, 0x44, 0x41, 0x70, 0x70,
	0x4d, 0x65, 0x74, 0x61, 0x12, 0x18, 0x0a, 0x07, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x05, 0x52, 0x07, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x12, 0x3b,
	0x0a, 0x05, 0x66, 0x75, 0x6e, 0x63, 0x73, 0x18, 0x02, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x25, 0x2e,
	0x77, 0x61, 0x76, 0x65, 0x73, 0x2e, 0x44, 0x41, 0x70, 0x70, 0x4d, 0x65, 0x74, 0x61, 0x2e, 0x43,
	0x61, 0x6c, 0x6c, 0x61, 0x62, 0x6c, 0x65, 0x46, 0x75, 0x6e, 0x63, 0x53, 0x69, 0x67, 0x6e, 0x61,
	0x74, 0x75, 0x72, 0x65, 0x52, 0x05, 0x66, 0x75, 0x6e, 0x63, 0x73, 0x12, 0x7e, 0x0a, 0x22, 0x63,
	0x6f, 0x6d, 0x70, 0x61, 0x63, 0x74, 0x4e, 0x61, 0x6d, 0x65, 0x41, 0x6e, 0x64, 0x4f, 0x72, 0x69,
	0x67, 0x69, 0x6e, 0x61, 0x6c, 0x4e, 0x61, 0x6d, 0x65, 0x50, 0x61, 0x69, 0x72, 0x4c, 0x69, 0x73,
	0x74, 0x18, 0x03, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x2e, 0x2e, 0x77, 0x61, 0x76, 0x65, 0x73, 0x2e,
	0x44, 0x41, 0x70, 0x70, 0x4d, 0x65, 0x74, 0x61, 0x2e, 0x43, 0x6f, 0x6d, 0x70, 0x61, 0x63, 0x74,
	0x4e, 0x61, 0x6d, 0x65, 0x41, 0x6e, 0x64, 0x4f, 0x72, 0x69, 0x67, 0x69, 0x6e, 0x61, 0x6c, 0x4e,
	0x61, 0x6d, 0x65, 0x50, 0x61, 0x69, 0x72, 0x52, 0x22, 0x63, 0x6f, 0x6d, 0x70, 0x61, 0x63, 0x74,
	0x4e, 0x61, 0x6d, 0x65, 0x41, 0x6e, 0x64, 0x4f, 0x72, 0x69, 0x67, 0x69, 0x6e, 0x61, 0x6c, 0x4e,
	0x61, 0x6d, 0x65, 0x50, 0x61, 0x69, 0x72, 0x4c, 0x69, 0x73, 0x74, 0x1a, 0x2d, 0x0a, 0x15, 0x43,
	0x61, 0x6c, 0x6c, 0x61, 0x62, 0x6c, 0x65, 0x46, 0x75, 0x6e, 0x63, 0x53, 0x69, 0x67, 0x6e, 0x61,
	0x74, 0x75, 0x72, 0x65, 0x12, 0x14, 0x0a, 0x05, 0x74, 0x79, 0x70, 0x65, 0x73, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x0c, 0x52, 0x05, 0x74, 0x79, 0x70, 0x65, 0x73, 0x1a, 0x66, 0x0a, 0x1e, 0x43, 0x6f,
	0x6d, 0x70, 0x61, 0x63, 0x74, 0x4e, 0x61, 0x6d, 0x65, 0x41, 0x6e, 0x64, 0x4f, 0x72, 0x69, 0x67,
	0x69, 0x6e, 0x61, 0x6c, 0x4e, 0x61, 0x6d, 0x65, 0x50, 0x61, 0x69, 0x72, 0x12, 0x20, 0x0a, 0x0b,
	0x63, 0x6f, 0x6d, 0x70, 0x61, 0x63, 0x74, 0x4e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x0b, 0x63, 0x6f, 0x6d, 0x70, 0x61, 0x63, 0x74, 0x4e, 0x61, 0x6d, 0x65, 0x12, 0x22,
	0x0a, 0x0c, 0x6f, 0x72, 0x69, 0x67, 0x69, 0x6e, 0x61, 0x6c, 0x4e, 0x61, 0x6d, 0x65, 0x18, 0x02,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x0c, 0x6f, 0x72, 0x69, 0x67, 0x69, 0x6e, 0x61, 0x6c, 0x4e, 0x61,
	0x6d, 0x65, 0x42, 0x63, 0x0a, 0x1f, 0x63, 0x6f, 0x6d, 0x2e, 0x77, 0x61, 0x76, 0x65, 0x73, 0x70,
	0x6c, 0x61, 0x74, 0x66, 0x6f, 0x72, 0x6d, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66,
	0x2e, 0x64, 0x61, 0x70, 0x70, 0x5a, 0x38, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f,
	0x6d, 0x2f, 0x77, 0x61, 0x76, 0x65, 0x73, 0x70, 0x6c, 0x61, 0x74, 0x66, 0x6f, 0x72, 0x6d, 0x2f,
	0x67, 0x6f, 0x77, 0x61, 0x76, 0x65, 0x73, 0x2f, 0x70, 0x6b, 0x67, 0x2f, 0x72, 0x69, 0x64, 0x65,
	0x2f, 0x6d, 0x65, 0x74, 0x61, 0x2f, 0x67, 0x65, 0x6e, 0x65, 0x72, 0x61, 0x74, 0x65, 0x64, 0xaa,
	0x02, 0x05, 0x57, 0x61, 0x76, 0x65, 0x73, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_DAppMeta_proto_rawDescOnce sync.Once
	file_DAppMeta_proto_rawDescData = file_DAppMeta_proto_rawDesc
)

func file_DAppMeta_proto_rawDescGZIP() []byte {
	file_DAppMeta_proto_rawDescOnce.Do(func() {
		file_DAppMeta_proto_rawDescData = protoimpl.X.CompressGZIP(file_DAppMeta_proto_rawDescData)
	})
	return file_DAppMeta_proto_rawDescData
}

var file_DAppMeta_proto_msgTypes = make([]protoimpl.MessageInfo, 3)
var file_DAppMeta_proto_goTypes = []interface{}{
	(*DAppMeta)(nil),                                // 0: waves.DAppMeta
	(*DAppMeta_CallableFuncSignature)(nil),          // 1: waves.DAppMeta.CallableFuncSignature
	(*DAppMeta_CompactNameAndOriginalNamePair)(nil), // 2: waves.DAppMeta.CompactNameAndOriginalNamePair
}
var file_DAppMeta_proto_depIdxs = []int32{
	1, // 0: waves.DAppMeta.funcs:type_name -> waves.DAppMeta.CallableFuncSignature
	2, // 1: waves.DAppMeta.compactNameAndOriginalNamePairList:type_name -> waves.DAppMeta.CompactNameAndOriginalNamePair
	2, // [2:2] is the sub-list for method output_type
	2, // [2:2] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_DAppMeta_proto_init() }
func file_DAppMeta_proto_init() {
	if File_DAppMeta_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_DAppMeta_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*DAppMeta); i {
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
		file_DAppMeta_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*DAppMeta_CallableFuncSignature); i {
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
		file_DAppMeta_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*DAppMeta_CompactNameAndOriginalNamePair); i {
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
			RawDescriptor: file_DAppMeta_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   3,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_DAppMeta_proto_goTypes,
		DependencyIndexes: file_DAppMeta_proto_depIdxs,
		MessageInfos:      file_DAppMeta_proto_msgTypes,
	}.Build()
	File_DAppMeta_proto = out.File
	file_DAppMeta_proto_rawDesc = nil
	file_DAppMeta_proto_goTypes = nil
	file_DAppMeta_proto_depIdxs = nil
}
