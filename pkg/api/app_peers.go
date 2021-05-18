package api

import (
	"context"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
	"time"
)

type Peer struct {
	Address  string `json:"address"`
	LastSeen uint64 `json:"lastSeen,omitempty"`
}

type PeersKnown struct {
	Peers []Peer `json:"peers"`
}

// PeersAll a list of all known not banned peers with a publicly available declared address
func (a *App) PeersAll() (PeersKnown, error) {
	suspendedIPs := a.peers.Suspended()
	suspendedMap := make(map[string]struct{}, len(suspendedIPs))
	for _, ip := range suspendedIPs {
		suspendedMap[ip] = struct{}{}
	}

	peers, err := a.peers.KnownPeers()
	if err != nil {
		return PeersKnown{}, errors.Wrap(err, "PeersKnown")
	}

	nowMillis := time.Now().UnixNano() / 1_000_000

	out := make([]Peer, 0, len(peers))
	for _, row := range peers {
		ip := row.String()
		if _, in := suspendedMap[ip]; in {
			continue
		}
		// FIXME(nickeksov): add normal last seen (this is crunch...)!!!!
		out = append(out, Peer{
			Address:  "/" + ip,
			LastSeen: uint64(nowMillis),
		})
	}

	return PeersKnown{Peers: out}, nil
}

func (a *App) PeersKnown() (PeersKnown, error) {
	peers, err := a.peers.KnownPeers()
	if err != nil {
		return PeersKnown{}, errors.Wrap(err, "PeersKnown")
	}

	out := make([]Peer, 0, len(peers))
	for _, row := range peers {
		// nickeksov: peers without lastSeen field
		out = append(out, Peer{Address: row.String()})
	}

	return PeersKnown{Peers: out}, nil
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
	Peers []PeerInfo `json:"peers"`
}

type PeerInfo struct {
	Address            string `json:"address"`
	DeclaredAddress    string `json:"declaredAddress"`
	PeerName           string `json:"peerName"`
	PeerNonce          uint64 `json:"peerNonce"`
	ApplicationName    string `json:"applicationName"`
	ApplicationVersion string `json:"applicationVersion"`
}

func peerInfoFromPeer(peer peer.Peer) PeerInfo {
	handshake := peer.Handshake()

	declaredAddrStr := "N/A"
	if !handshake.DeclaredAddr.Empty() {
		declaredAddrStr = handshake.DeclaredAddr.String()
	}

	return PeerInfo{
		Address:            "/" + peer.RemoteAddr().String(),
		DeclaredAddress:    "/" + declaredAddrStr,
		PeerName:           handshake.NodeName,
		PeerNonce:          handshake.NodeNonce,
		ApplicationName:    handshake.AppName,
		ApplicationVersion: handshake.Version.String(),
	}
}

func (a *App) PeersConnected() (PeersConnectedResponse, error) {
	var out []PeerInfo
	a.peers.EachConnected(func(peer peer.Peer, _ *proto.Score) {
		out = append(out, peerInfoFromPeer(peer))
	})

	return PeersConnectedResponse{
		Peers: out,
	}, nil
}

type PeersSuspendedResponse struct {
	Peers []string `json:"peers"`
}

func (a *App) PeersSuspended() (PeersSuspendedResponse, error) {
	peers := a.peers.Suspended()
	return PeersSuspendedResponse{peers}, nil
}

type PeersSpawnedResponse struct {
	Peers []proto.IpPort
}

func (a *App) PeersSpawned() PeersSpawnedResponse {
	rs := a.peers.Spawned()
	return PeersSpawnedResponse{Peers: rs}
}
