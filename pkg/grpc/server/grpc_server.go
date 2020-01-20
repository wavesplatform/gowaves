package server

import (
	"context"
	"net"

	"github.com/pkg/errors"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated"
	"github.com/wavesplatform/gowaves/pkg/miner/utxpool"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/types"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type Server struct {
	state  state.StateInfo
	scheme proto.Scheme
	utx    *utxpool.UtxImpl
	sch    types.Scheduler
}

func NewServer(state state.StateInfo, utx *utxpool.UtxImpl, sch types.Scheduler) (*Server, error) {
	s := &Server{}
	if err := s.initServer(state, utx, sch); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Server) initServer(state state.StateInfo, utx *utxpool.UtxImpl, sch types.Scheduler) error {
	settings, err := state.BlockchainSettings()
	if err != nil {
		return err
	}
	s.state = state
	s.scheme = settings.AddressSchemeCharacter
	s.utx = utx
	s.sch = sch
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
