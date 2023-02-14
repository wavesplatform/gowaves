// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v3.21.12
// source: waves/events/grpc/blockchain_updates.proto

package grpc

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// BlockchainUpdatesApiClient is the client API for BlockchainUpdatesApi service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type BlockchainUpdatesApiClient interface {
	GetBlockUpdate(ctx context.Context, in *GetBlockUpdateRequest, opts ...grpc.CallOption) (*GetBlockUpdateResponse, error)
	GetBlockUpdatesRange(ctx context.Context, in *GetBlockUpdatesRangeRequest, opts ...grpc.CallOption) (*GetBlockUpdatesRangeResponse, error)
	Subscribe(ctx context.Context, in *SubscribeRequest, opts ...grpc.CallOption) (BlockchainUpdatesApi_SubscribeClient, error)
}

type blockchainUpdatesApiClient struct {
	cc grpc.ClientConnInterface
}

func NewBlockchainUpdatesApiClient(cc grpc.ClientConnInterface) BlockchainUpdatesApiClient {
	return &blockchainUpdatesApiClient{cc}
}

func (c *blockchainUpdatesApiClient) GetBlockUpdate(ctx context.Context, in *GetBlockUpdateRequest, opts ...grpc.CallOption) (*GetBlockUpdateResponse, error) {
	out := new(GetBlockUpdateResponse)
	err := c.cc.Invoke(ctx, "/waves.events.grpc.BlockchainUpdatesApi/GetBlockUpdate", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *blockchainUpdatesApiClient) GetBlockUpdatesRange(ctx context.Context, in *GetBlockUpdatesRangeRequest, opts ...grpc.CallOption) (*GetBlockUpdatesRangeResponse, error) {
	out := new(GetBlockUpdatesRangeResponse)
	err := c.cc.Invoke(ctx, "/waves.events.grpc.BlockchainUpdatesApi/GetBlockUpdatesRange", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *blockchainUpdatesApiClient) Subscribe(ctx context.Context, in *SubscribeRequest, opts ...grpc.CallOption) (BlockchainUpdatesApi_SubscribeClient, error) {
	stream, err := c.cc.NewStream(ctx, &BlockchainUpdatesApi_ServiceDesc.Streams[0], "/waves.events.grpc.BlockchainUpdatesApi/Subscribe", opts...)
	if err != nil {
		return nil, err
	}
	x := &blockchainUpdatesApiSubscribeClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type BlockchainUpdatesApi_SubscribeClient interface {
	Recv() (*SubscribeEvent, error)
	grpc.ClientStream
}

type blockchainUpdatesApiSubscribeClient struct {
	grpc.ClientStream
}

func (x *blockchainUpdatesApiSubscribeClient) Recv() (*SubscribeEvent, error) {
	m := new(SubscribeEvent)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// BlockchainUpdatesApiServer is the server API for BlockchainUpdatesApi service.
// All implementations should embed UnimplementedBlockchainUpdatesApiServer
// for forward compatibility
type BlockchainUpdatesApiServer interface {
	GetBlockUpdate(context.Context, *GetBlockUpdateRequest) (*GetBlockUpdateResponse, error)
	GetBlockUpdatesRange(context.Context, *GetBlockUpdatesRangeRequest) (*GetBlockUpdatesRangeResponse, error)
	Subscribe(*SubscribeRequest, BlockchainUpdatesApi_SubscribeServer) error
}

// UnimplementedBlockchainUpdatesApiServer should be embedded to have forward compatible implementations.
type UnimplementedBlockchainUpdatesApiServer struct {
}

func (UnimplementedBlockchainUpdatesApiServer) GetBlockUpdate(context.Context, *GetBlockUpdateRequest) (*GetBlockUpdateResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetBlockUpdate not implemented")
}
func (UnimplementedBlockchainUpdatesApiServer) GetBlockUpdatesRange(context.Context, *GetBlockUpdatesRangeRequest) (*GetBlockUpdatesRangeResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetBlockUpdatesRange not implemented")
}
func (UnimplementedBlockchainUpdatesApiServer) Subscribe(*SubscribeRequest, BlockchainUpdatesApi_SubscribeServer) error {
	return status.Errorf(codes.Unimplemented, "method Subscribe not implemented")
}

// UnsafeBlockchainUpdatesApiServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to BlockchainUpdatesApiServer will
// result in compilation errors.
type UnsafeBlockchainUpdatesApiServer interface {
	mustEmbedUnimplementedBlockchainUpdatesApiServer()
}

func RegisterBlockchainUpdatesApiServer(s grpc.ServiceRegistrar, srv BlockchainUpdatesApiServer) {
	s.RegisterService(&BlockchainUpdatesApi_ServiceDesc, srv)
}

func _BlockchainUpdatesApi_GetBlockUpdate_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetBlockUpdateRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BlockchainUpdatesApiServer).GetBlockUpdate(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/waves.events.grpc.BlockchainUpdatesApi/GetBlockUpdate",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BlockchainUpdatesApiServer).GetBlockUpdate(ctx, req.(*GetBlockUpdateRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _BlockchainUpdatesApi_GetBlockUpdatesRange_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetBlockUpdatesRangeRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BlockchainUpdatesApiServer).GetBlockUpdatesRange(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/waves.events.grpc.BlockchainUpdatesApi/GetBlockUpdatesRange",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BlockchainUpdatesApiServer).GetBlockUpdatesRange(ctx, req.(*GetBlockUpdatesRangeRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _BlockchainUpdatesApi_Subscribe_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(SubscribeRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(BlockchainUpdatesApiServer).Subscribe(m, &blockchainUpdatesApiSubscribeServer{stream})
}

type BlockchainUpdatesApi_SubscribeServer interface {
	Send(*SubscribeEvent) error
	grpc.ServerStream
}

type blockchainUpdatesApiSubscribeServer struct {
	grpc.ServerStream
}

func (x *blockchainUpdatesApiSubscribeServer) Send(m *SubscribeEvent) error {
	return x.ServerStream.SendMsg(m)
}

// BlockchainUpdatesApi_ServiceDesc is the grpc.ServiceDesc for BlockchainUpdatesApi service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var BlockchainUpdatesApi_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "waves.events.grpc.BlockchainUpdatesApi",
	HandlerType: (*BlockchainUpdatesApiServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetBlockUpdate",
			Handler:    _BlockchainUpdatesApi_GetBlockUpdate_Handler,
		},
		{
			MethodName: "GetBlockUpdatesRange",
			Handler:    _BlockchainUpdatesApi_GetBlockUpdatesRange_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "Subscribe",
			Handler:       _BlockchainUpdatesApi_Subscribe_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "waves/events/grpc/blockchain_updates.proto",
}
