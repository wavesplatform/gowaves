package server

import (
	"context"
	"net"

	"github.com/pkg/errors"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves/node/grpc"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/services"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/types"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type Server struct {
	state  state.StateInfo
	scheme proto.Scheme
	utx    types.UtxPool
	wallet types.EmbeddedWallet
}

func NewServer(services services.Services) (*Server, error) {
	s := &Server{}
	if err := s.initServer(services.State, services.UtxPool, services.Wallet); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Server) initServer(state state.StateInfo, utx types.UtxPool, sch types.EmbeddedWallet) error {
	settings, err := state.BlockchainSettings()
	if err != nil {
		return err
	}
	s.state = state
	s.scheme = settings.AddressSchemeCharacter
	s.utx = utx
	s.wallet = sch
	return nil
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
