// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v3.19.2
// source: waves/node/grpc/blocks_api.proto

package grpc

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
	wrapperspb "google.golang.org/protobuf/types/known/wrapperspb"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// BlocksApiClient is the client API for BlocksApi service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type BlocksApiClient interface {
	GetBlock(ctx context.Context, in *BlockRequest, opts ...grpc.CallOption) (*BlockWithHeight, error)
	GetBlockRange(ctx context.Context, in *BlockRangeRequest, opts ...grpc.CallOption) (BlocksApi_GetBlockRangeClient, error)
	GetCurrentHeight(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*wrapperspb.UInt32Value, error)
}

type blocksApiClient struct {
	cc grpc.ClientConnInterface
}

func NewBlocksApiClient(cc grpc.ClientConnInterface) BlocksApiClient {
	return &blocksApiClient{cc}
}

func (c *blocksApiClient) GetBlock(ctx context.Context, in *BlockRequest, opts ...grpc.CallOption) (*BlockWithHeight, error) {
	out := new(BlockWithHeight)
	err := c.cc.Invoke(ctx, "/waves.node.grpc.BlocksApi/GetBlock", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *blocksApiClient) GetBlockRange(ctx context.Context, in *BlockRangeRequest, opts ...grpc.CallOption) (BlocksApi_GetBlockRangeClient, error) {
	stream, err := c.cc.NewStream(ctx, &BlocksApi_ServiceDesc.Streams[0], "/waves.node.grpc.BlocksApi/GetBlockRange", opts...)
	if err != nil {
		return nil, err
	}
	x := &blocksApiGetBlockRangeClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type BlocksApi_GetBlockRangeClient interface {
	Recv() (*BlockWithHeight, error)
	grpc.ClientStream
}

type blocksApiGetBlockRangeClient struct {
	grpc.ClientStream
}

func (x *blocksApiGetBlockRangeClient) Recv() (*BlockWithHeight, error) {
	m := new(BlockWithHeight)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *blocksApiClient) GetCurrentHeight(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*wrapperspb.UInt32Value, error) {
	out := new(wrapperspb.UInt32Value)
	err := c.cc.Invoke(ctx, "/waves.node.grpc.BlocksApi/GetCurrentHeight", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// BlocksApiServer is the server API for BlocksApi service.
// All implementations should embed UnimplementedBlocksApiServer
// for forward compatibility
type BlocksApiServer interface {
	GetBlock(context.Context, *BlockRequest) (*BlockWithHeight, error)
	GetBlockRange(*BlockRangeRequest, BlocksApi_GetBlockRangeServer) error
	GetCurrentHeight(context.Context, *emptypb.Empty) (*wrapperspb.UInt32Value, error)
}

// UnimplementedBlocksApiServer should be embedded to have forward compatible implementations.
type UnimplementedBlocksApiServer struct {
}

func (UnimplementedBlocksApiServer) GetBlock(context.Context, *BlockRequest) (*BlockWithHeight, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetBlock not implemented")
}
func (UnimplementedBlocksApiServer) GetBlockRange(*BlockRangeRequest, BlocksApi_GetBlockRangeServer) error {
	return status.Errorf(codes.Unimplemented, "method GetBlockRange not implemented")
}
func (UnimplementedBlocksApiServer) GetCurrentHeight(context.Context, *emptypb.Empty) (*wrapperspb.UInt32Value, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetCurrentHeight not implemented")
}

// UnsafeBlocksApiServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to BlocksApiServer will
// result in compilation errors.
type UnsafeBlocksApiServer interface {
	mustEmbedUnimplementedBlocksApiServer()
}

func RegisterBlocksApiServer(s grpc.ServiceRegistrar, srv BlocksApiServer) {
	s.RegisterService(&BlocksApi_ServiceDesc, srv)
}

func _BlocksApi_GetBlock_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(BlockRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BlocksApiServer).GetBlock(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/waves.node.grpc.BlocksApi/GetBlock",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BlocksApiServer).GetBlock(ctx, req.(*BlockRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _BlocksApi_GetBlockRange_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(BlockRangeRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(BlocksApiServer).GetBlockRange(m, &blocksApiGetBlockRangeServer{stream})
}

type BlocksApi_GetBlockRangeServer interface {
	Send(*BlockWithHeight) error
	grpc.ServerStream
}

type blocksApiGetBlockRangeServer struct {
	grpc.ServerStream
}

func (x *blocksApiGetBlockRangeServer) Send(m *BlockWithHeight) error {
	return x.ServerStream.SendMsg(m)
}

func _BlocksApi_GetCurrentHeight_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(emptypb.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BlocksApiServer).GetCurrentHeight(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/waves.node.grpc.BlocksApi/GetCurrentHeight",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BlocksApiServer).GetCurrentHeight(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

// BlocksApi_ServiceDesc is the grpc.ServiceDesc for BlocksApi service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var BlocksApi_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "waves.node.grpc.BlocksApi",
	HandlerType: (*BlocksApiServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetBlock",
			Handler:    _BlocksApi_GetBlock_Handler,
		},
		{
			MethodName: "GetCurrentHeight",
			Handler:    _BlocksApi_GetCurrentHeight_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "GetBlockRange",
			Handler:       _BlocksApi_GetBlockRange_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "waves/node/grpc/blocks_api.proto",
}
