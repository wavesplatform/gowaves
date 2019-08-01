package api

import (
	"context"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
)

type Peer struct {
	Address  string `json:"address"`
	LastSeen uint64 `json:"lastSeen"`
}

type PeersAll struct {
	Peers []Peer `json:"peers"`
}

func (a *App) PeersAll() (*PeersAll, error) {
	peers, err := a.state.Peers()
	if err != nil {
		return nil, &InternalError{err}
	}

	var out []Peer
	for _, row := range peers {
		out = append(out, Peer{Address: row.String()})
	}

	return &PeersAll{Peers: out}, nil
}

type PeersConnectResponse struct {
	Hostname string `json:"hostname"`
	Status   string `json:"status"`
}

func (a *App) PeersConnect(ctx context.Context, apiKey string, addr string) (*PeersConnectResponse, error) {
	err := a.checkAuth(apiKey)
	if err != nil {
		return nil, err
	}

	d := proto.NewTCPAddrFromString(addr)
	if d.Empty() {
		zap.S().Error(apiKey, addr, d)
		return nil, &BadRequestError{errors.New("invalid address")}
	}

	err = a.peers.Connect(ctx, d)
	if err != nil {
		return nil, &BadRequestError{err}
	}

	return &PeersConnectResponse{
		Hostname: d.String(),
		Status:   "Trying to connect",
	}, nil
}

type PeersConnectedResponse struct {
	Peers []PeersConnectedRow `json:"peers"`
}

type PeersConnectedRow struct {
	Address            string `json:"address"`
	DeclaredAddress    string `json:"declaredAddress"`
	PeerName           string `json:"peerName"`
	PeerNonce          uint64 `json:"peerNonce"`
	ApplicationName    string `json:"applicationName"`
	ApplicationVersion string `json:"applicationVersion"`
}

func (a *App) PeersConnected() (*PeersConnectedResponse, error) {
	var out []PeersConnectedRow
	a.peers.EachConnected(func(peer peer.Peer, i *proto.Score) {

		v := PeersConnectedRow{
			Address:            "/" + peer.RemoteAddr().String(),
			DeclaredAddress:    "/" + peer.Handshake().DeclaredAddr.String(),
			PeerName:           peer.Handshake().NodeName,
			PeerNonce:          peer.Handshake().NodeNonce,
			ApplicationName:    peer.Handshake().AppName,
			ApplicationVersion: peer.Handshake().Version.String(),
		}

		out = append(out, v)

	})

	return &PeersConnectedResponse{
		Peers: out,
	}, nil
}
