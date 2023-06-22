// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v4.23.3
// source: waves/node/grpc/blockchain_api.proto

package grpc

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// BlockchainApiClient is the client API for BlockchainApi service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type BlockchainApiClient interface {
	GetActivationStatus(ctx context.Context, in *ActivationStatusRequest, opts ...grpc.CallOption) (*ActivationStatusResponse, error)
	GetBaseTarget(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*BaseTargetResponse, error)
	GetCumulativeScore(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*ScoreResponse, error)
}

type blockchainApiClient struct {
	cc grpc.ClientConnInterface
}

func NewBlockchainApiClient(cc grpc.ClientConnInterface) BlockchainApiClient {
	return &blockchainApiClient{cc}
}

func (c *blockchainApiClient) GetActivationStatus(ctx context.Context, in *ActivationStatusRequest, opts ...grpc.CallOption) (*ActivationStatusResponse, error) {
	out := new(ActivationStatusResponse)
	err := c.cc.Invoke(ctx, "/waves.node.grpc.BlockchainApi/GetActivationStatus", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *blockchainApiClient) GetBaseTarget(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*BaseTargetResponse, error) {
	out := new(BaseTargetResponse)
	err := c.cc.Invoke(ctx, "/waves.node.grpc.BlockchainApi/GetBaseTarget", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *blockchainApiClient) GetCumulativeScore(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*ScoreResponse, error) {
	out := new(ScoreResponse)
	err := c.cc.Invoke(ctx, "/waves.node.grpc.BlockchainApi/GetCumulativeScore", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// BlockchainApiServer is the server API for BlockchainApi service.
// All implementations should embed UnimplementedBlockchainApiServer
// for forward compatibility
type BlockchainApiServer interface {
	GetActivationStatus(context.Context, *ActivationStatusRequest) (*ActivationStatusResponse, error)
	GetBaseTarget(context.Context, *emptypb.Empty) (*BaseTargetResponse, error)
	GetCumulativeScore(context.Context, *emptypb.Empty) (*ScoreResponse, error)
}

// UnimplementedBlockchainApiServer should be embedded to have forward compatible implementations.
type UnimplementedBlockchainApiServer struct {
}

func (UnimplementedBlockchainApiServer) GetActivationStatus(context.Context, *ActivationStatusRequest) (*ActivationStatusResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetActivationStatus not implemented")
}
func (UnimplementedBlockchainApiServer) GetBaseTarget(context.Context, *emptypb.Empty) (*BaseTargetResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetBaseTarget not implemented")
}
func (UnimplementedBlockchainApiServer) GetCumulativeScore(context.Context, *emptypb.Empty) (*ScoreResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetCumulativeScore not implemented")
}

// UnsafeBlockchainApiServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to BlockchainApiServer will
// result in compilation errors.
type UnsafeBlockchainApiServer interface {
	mustEmbedUnimplementedBlockchainApiServer()
}

func RegisterBlockchainApiServer(s grpc.ServiceRegistrar, srv BlockchainApiServer) {
	s.RegisterService(&BlockchainApi_ServiceDesc, srv)
}

func _BlockchainApi_GetActivationStatus_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ActivationStatusRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BlockchainApiServer).GetActivationStatus(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/waves.node.grpc.BlockchainApi/GetActivationStatus",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BlockchainApiServer).GetActivationStatus(ctx, req.(*ActivationStatusRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _BlockchainApi_GetBaseTarget_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(emptypb.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BlockchainApiServer).GetBaseTarget(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/waves.node.grpc.BlockchainApi/GetBaseTarget",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BlockchainApiServer).GetBaseTarget(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _BlockchainApi_GetCumulativeScore_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(emptypb.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BlockchainApiServer).GetCumulativeScore(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/waves.node.grpc.BlockchainApi/GetCumulativeScore",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BlockchainApiServer).GetCumulativeScore(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

// BlockchainApi_ServiceDesc is the grpc.ServiceDesc for BlockchainApi service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var BlockchainApi_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "waves.node.grpc.BlockchainApi",
	HandlerType: (*BlockchainApiServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetActivationStatus",
			Handler:    _BlockchainApi_GetActivationStatus_Handler,
		},
		{
			MethodName: "GetBaseTarget",
			Handler:    _BlockchainApi_GetBaseTarget_Handler,
		},
		{
			MethodName: "GetCumulativeScore",
			Handler:    _BlockchainApi_GetCumulativeScore_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "waves/node/grpc/blockchain_api.proto",
}
