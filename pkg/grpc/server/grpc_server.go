package server

import (
	"context"
	"net"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"

	"github.com/wavesplatform/gowaves/pkg/api"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves/node/grpc"
	"github.com/wavesplatform/gowaves/pkg/node/messages"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/types"
	"github.com/wavesplatform/gowaves/pkg/util/limit_listener"
)

const (
	DefaultMaxConnections = 128
)

type Server struct {
	state       state.StateInfo
	scheme      proto.Scheme
	utx         types.UtxPool
	wallet      types.EmbeddedWallet
	grpcServer  *grpc.Server
	broadcastCh chan<- *messages.BroadcastTransaction
}

type RunOptions struct {
	MaxConnections int
}

func DefaultRunOptions() *RunOptions {
	return &RunOptions{
		MaxConnections: DefaultMaxConnections,
	}
}

func NewServer(app *api.App) (*Server, error) {
	s := &Server{
		state:       nil,
		scheme:      0,
		utx:         nil,
		wallet:      nil,
		grpcServer:  nil,
		broadcastCh: nil,
	}
	s.grpcServer = createGRPCServerWithHandlers(s)
	s.broadcastCh = app.BroadcastChannel()
	if err := s.initServer(app.State(), app.UtxPool(), app.Wallet(), app.Scheme()); err != nil {
		return nil, err
	}
	return s, nil
}

func createGRPCServerWithHandlers(handlers GrpcHandlers) *grpc.Server {
	grpcServer := grpc.NewServer(
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             10 * time.Second,
			PermitWithoutStream: true,
		}),
	)
	g.RegisterAccountsApiServer(grpcServer, handlers)
	g.RegisterAssetsApiServer(grpcServer, handlers)
	g.RegisterBlockchainApiServer(grpcServer, handlers)
	g.RegisterBlocksApiServer(grpcServer, handlers)
	g.RegisterTransactionsApiServer(grpcServer, handlers)
	reflection.Register(grpcServer) // Register reflection service on gRPC server.
	return grpcServer
}

func (s *Server) initServer(
	state state.StateInfo, utx types.UtxPool, wallet types.EmbeddedWallet, scheme proto.Scheme,
) error {
	s.state = state
	s.scheme = scheme
	s.utx = utx
	s.wallet = wallet
	return nil
}

func (s *Server) Run(ctx context.Context, address string, opts *RunOptions) error {
	if opts == nil {
		opts = DefaultRunOptions()
	}

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

	go func() {
		<-ctx.Done()
		zap.S().Info("Shutting down gRPC server...")
		s.Stop()
	}()
	zap.S().Infof("Starting gRPC server on '%s'", address)
	return s.Serve(conn)
}

// Stop calls underlying gRPC server stop method.
func (s *Server) Stop() {
	s.grpcServer.Stop()
}

// Serve calls underlying gRPC server serve method with provided net.Listener. This call is blocking.
func (s *Server) Serve(l net.Listener) error {
	return s.grpcServer.Serve(l)
}
