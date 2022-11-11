package server

import (
	"context"
	"net"
	"time"

	"github.com/pkg/errors"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves/node/grpc"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/services"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/types"
	"github.com/wavesplatform/gowaves/pkg/util/limit_listener"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
)

const (
	DefaultMaxConnections = 128
)

type Server struct {
	state      state.StateInfo
	scheme     proto.Scheme
	utx        types.UtxPool
	wallet     types.EmbeddedWallet
	services   services.Services
	handlers   GrpcHandlers
	grpcServer *grpc.Server
}

type RunOptions struct {
	MaxConnections int
}

func DefaultRunOptions() *RunOptions {
	return &RunOptions{
		MaxConnections: DefaultMaxConnections,
	}
}

func NewServer(services services.Services) (*Server, error) {
	s := &Server{}
	s.services = services
	if err := s.initServer(services.State, services.UtxPool, services.Wallet); err != nil {
		return nil, err
	}
	s.handlers = s
	return s, nil
}

func NewServerWithHandlers(services services.Services, h GrpcHandlers) (*Server, error) {
	s, err := NewServer(services)
	if err != nil {
		return nil, err
	}
	s.handlers = h
	return s, nil
}

func (s *Server) initServer(state state.StateInfo, utx types.UtxPool, sch types.EmbeddedWallet) error {
	s.state = state
	s.scheme = s.services.Scheme
	s.utx = utx
	s.wallet = sch
	return nil
}

func (s *Server) Run(ctx context.Context, address string, opts *RunOptions) error {
	if opts == nil {
		opts = DefaultRunOptions()
	}

	grpcServer := grpc.NewServer(
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             10 * time.Second,
			PermitWithoutStream: true,
		}),
	)
	s.grpcServer = grpcServer
	g.RegisterAccountsApiServer(grpcServer, s)
	g.RegisterAssetsApiServer(grpcServer, s)
	g.RegisterBlockchainApiServer(grpcServer, s)
	g.RegisterBlocksApiServer(grpcServer, s)
	g.RegisterTransactionsApiServer(grpcServer, s)

	go func() {
		<-ctx.Done()
		zap.S().Info("Shutting down gRPC server...")
		grpcServer.Stop()
	}()

	conn, err := net.Listen("tcp", address)
	if err != nil {
		return errors.Errorf("net.Listen: %v", err)
	}

	if opts.MaxConnections > 0 {
		conn = limit_listener.LimitListener(conn, opts.MaxConnections)
		zap.S().Debugf("Set limit for number of simultaneous connections for gRPC API to %d", opts.MaxConnections)
	}

	defer func(conn net.Listener) {
		err := conn.Close()
		if err != nil {
			zap.S().Errorf("Failed to close gRPC server connection: %v", err)
		}
	}(conn)

	if err := grpcServer.Serve(conn); err != nil {
		return errors.Errorf("grpcServer.Serve: %v", err)
	}
	return nil
}

func (s *Server) Stop() {
	s.grpcServer.Stop()
}

func (s *Server) Serve(l net.Listener) error {
	grpcServer := grpc.NewServer(
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             10 * time.Second,
			PermitWithoutStream: true,
		}),
	)
	g.RegisterAccountsApiServer(grpcServer, s.handlers)
	g.RegisterAssetsApiServer(grpcServer, s.handlers)
	g.RegisterBlockchainApiServer(grpcServer, s.handlers)
	g.RegisterBlocksApiServer(grpcServer, s.handlers)
	g.RegisterTransactionsApiServer(grpcServer, s.handlers)
	s.grpcServer = grpcServer

	// Register reflection service on gRPC server.
	reflection.Register(grpcServer)
	if err := grpcServer.Serve(l); err != nil {
		return errors.Errorf("grpcServer.Serve: %v", err)
	}

	return nil
}
