package api

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"time"

	"github.com/pkg/errors"

	apiErrs "github.com/wavesplatform/gowaves/pkg/api/errors"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/util/common"
)

type Peer struct {
	Address  string `json:"address"`
	LastSeen uint64 `json:"lastSeen,omitempty"`
}

type PeersKnown struct {
	Peers []Peer `json:"peers"`
}

// PeersAll is a list of all known not banned, not suspended and not blacklisted peers with a publicly
// available declared address.
func (a *App) PeersAll() (PeersKnown, error) {
	blackList := a.peers.BlackList()
	restrictedIPsMap := make(map[string]struct{}, len(blackList))
	for _, blackListedPeer := range blackList {
		restrictedIPsMap[blackListedPeer.IP.String()] = struct{}{}
	}

	knownPeers := a.peers.KnownPeers()

	nowMillis := common.UnixMillisFromTime(time.Now())

	out := make([]Peer, 0, len(knownPeers))
	for _, knownPeer := range knownPeers {
		ip := knownPeer.IP() // extract IP from KnownPeer
		ipStr := ip.String() // convert IP to string for comparison
		if _, in := restrictedIPsMap[ipStr]; in {
			continue
		}
		// FIXME(nickeksov): add normal lastSeen field
		out = append(out, Peer{
			Address:  "/" + knownPeer.String(), // addr with port
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
		slog.Error("Invalid peer's address to connect", "address", addr)
		return nil, apiErrs.NewBadRequestError(errors.New("invalid address"))
	}

	err = a.peers.Connect(ctx, d)
	if err != nil {
		return nil, apiErrs.NewBadRequestError(err)
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

type RestrictedPeerInfo struct {
	Hostname  string `json:"hostname"`
	Timestamp int64  `json:"timestamp"` // nickeskov: timestamp in millis
	Reason    string `json:"reason,omitempty"`
}

func (a *App) PeersBlackListed() []RestrictedPeerInfo {
	blackList := a.peers.BlackList()

	out := make([]RestrictedPeerInfo, 0, len(blackList))
	for _, p := range blackList {
		out = append(out, RestrictedPeerInfo{
			Hostname:  "/" + p.IP.String(),
			Timestamp: p.RestrictTimestampMillis,
			Reason:    p.Reason,
		})
	}

	return out
}

func resolveAddrToIPsV4(addr string) ([]net.IP, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return proto.ResolveHostToIPsv4(addr) // try resolve addr as a host
	}
	if _, pErr := strconv.ParseUint(port, 10, 16); pErr != nil { // validate port num
		return nil, errors.Errorf("invalid port %q", port)
	}
	return proto.ResolveHostToIPsv4(host)
}

func filterUnspecifiedIPs(ips []net.IP) []net.IP {
	filtered := make([]net.IP, 0, len(ips))
	for _, ip := range ips {
		if ip.IsUnspecified() {
			continue
		}
		filtered = append(filtered, ip)
	}
	return filtered
}

func (a *App) PeersBlackList(blacklistedAddr, requestID, clientIP string) error {
	iPsv4, err := resolveAddrToIPsV4(blacklistedAddr)
	if err != nil {
		slog.Info("Invalid peer's address to blacklist",
			slog.String("address", blacklistedAddr),
			slog.String("client-ip", clientIP),
			slog.String("request-id", requestID),
		)
		return apiErrs.NewBadRequestError(errors.Wrapf(err,
			"failed to resolve blacklisted host '%s'", blacklistedAddr,
		))
	}
	iPsv4Filtered := filterUnspecifiedIPs(iPsv4)
	if len(iPsv4Filtered) == 0 {
		slog.Warn("No peer's blacklisted host found",
			slog.String("address", blacklistedAddr),
			slog.String("client-ip", clientIP),
			slog.String("request-id", requestID),
			slog.Any("resolved-ips", iPsv4),
		)
		return apiErrs.NewBadRequestError(errors.Errorf(
			"no valid IPs found for blacklisted host '%s'", blacklistedAddr,
		))
	}
	now := time.Now().UTC()
	reason := fmt.Sprintf(
		"blacklisted by API at now='%s' by client='%s' with request-id='%s' addresses='%v'",
		now.Format(time.RFC3339), clientIP, requestID, iPsv4Filtered,
	)
	for _, ip := range iPsv4Filtered {
		ipAddr := proto.NewTCPAddr(ip, 0)
		a.peers.AddToBlackListByIP(ipAddr, now, reason)
	}
	return nil
}

type PeersClearBlackListResponse struct {
	Result string `json:"result"`
}

func (a *App) PeersClearBlackList() PeersClearBlackListResponse {
	resp := PeersClearBlackListResponse{}
	if err := a.peers.ClearBlackList(); err != nil {
		resp.Result = fmt.Sprintf("failed to clear blacklist: %s", err.Error())
	} else {
		resp.Result = "blacklist cleared"
	}
	return resp
}

type PeersSpawnedResponse struct {
	Peers []proto.IpPort `json:"peers"`
}

func (a *App) PeersSpawned() PeersSpawnedResponse {
	rs := a.peers.Spawned()
	return PeersSpawnedResponse{Peers: rs}
}
