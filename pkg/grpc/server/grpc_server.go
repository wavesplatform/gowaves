package server

import (
	"context"
	"net"

	"github.com/pkg/errors"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type Server struct {
	state  state.State
	scheme proto.Scheme
}

func NewServer(state state.State) (*Server, error) {
	settings, err := state.BlockchainSettings()
	if err != nil {
		return nil, err
	}
	return &Server{state: state, scheme: settings.AddressSchemeCharacter}, nil
}

func (s *Server) Run(ctx context.Context, address string) error {
	grpcServer := grpc.NewServer()
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
	defer conn.Close()

	if err := grpcServer.Serve(conn); err != nil {
		return errors.Errorf("grpcServer.Serve: %v", err)
	}
	return nil
}
