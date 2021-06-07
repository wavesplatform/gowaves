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

// PeersAll is a list of all known not banned and not suspended peers with a publicly available declared address
func (a *App) PeersAll() (PeersKnown, error) {
	suspended := a.peers.Suspended()
	suspendedIPsMap := make(map[string]struct{}, len(suspended))
	for _, suspendedPeer := range suspended {
		suspendedIPsMap[suspendedPeer.IP.String()] = struct{}{}
	}

	knownPeers := a.peers.KnownPeers()

	nowMillis := unixMillis(time.Now())

	out := make([]Peer, 0, len(knownPeers))
	for _, knownPeer := range knownPeers {
		ip := knownPeer.String()
		if _, in := suspendedIPsMap[ip]; in {
			continue
		}
		// FIXME(nickeksov): add normal lastSeen field
		out = append(out, Peer{
			Address:  "/" + ip,
			LastSeen: uint64(nowMillis),
		})
	}

	return PeersKnown{Peers: out}, nil
}

func (a *App) PeersKnown() (PeersKnown, error) {
	knownPeers := a.peers.KnownPeers()

	out := make([]Peer, 0, len(knownPeers))
	for _, knownPeer := range knownPeers {
		// nickeksov: knownPeers without lastSeen field
		out = append(out, Peer{Address: knownPeer.String()})
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

func (a *App) PeersConnected() PeersConnectedResponse {
	var out []PeerInfo
	a.peers.EachConnected(func(peer peer.Peer, _ *proto.Score) {
		out = append(out, peerInfoFromPeer(peer))
	})

	return PeersConnectedResponse{
		Peers: out,
	}
}

type SuspendedPeerInfo struct {
	Hostname  string `json:"hostname"`
	Timestamp int64  `json:"timestamp"` // nickeskov: timestamp in millis
	Reason    string `json:"reason,omitempty"`
}

func (a *App) PeersSuspended() []SuspendedPeerInfo {
	suspended := a.peers.Suspended()

	out := make([]SuspendedPeerInfo, 0, len(suspended))
	for _, p := range suspended {
		out = append(out, SuspendedPeerInfo{
			Hostname:  "/" + p.IP.String(),
			Timestamp: p.SuspendTimestampMillis,
			Reason:    p.Reason,
		})
	}

	return out
}

type PeersSpawnedResponse struct {
	Peers []proto.IpPort `json:"peers"`
}

func (a *App) PeersSpawned() PeersSpawnedResponse {
	rs := a.peers.Spawned()
	return PeersSpawnedResponse{Peers: rs}
}
