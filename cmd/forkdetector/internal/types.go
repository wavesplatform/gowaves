package internal

import (
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"net"
	"strconv"
	"strings"
)

type PeerDesignation struct {
	Address net.IP
	Nonce   uint64
}

func NewPeerDesignation(address net.IP, nonce uint64) PeerDesignation {
	a := make(net.IP, len(address))
	copy(a, address)
	return PeerDesignation{Address: a, Nonce: nonce}
}

func (pd PeerDesignation) String() string {
	var sb strings.Builder
	sb.WriteString(pd.Address.String())
	sb.WriteRune('-')
	sb.WriteString(strconv.FormatUint(pd.Nonce, 10))
	return sb.String()
}

func (pd PeerDesignation) MarshalJSON() ([]byte, error)  {
	sb := strings.Builder{}
	sb.WriteRune('"')
	sb.WriteString(pd.String())
	sb.WriteRune('"')
	return []byte(sb.String()), nil
}

type PeerForkInfo struct {
	Peer PeerDesignation `json:"peer"`
	Lag  int             `json:"lag"`
}

func NewPeerForkInfo(peer PeerDesignation, lag int) PeerForkInfo {
	return PeerForkInfo{Peer: peer, Lag: lag}
}

type Fork struct {
	HeadBlock   crypto.Signature `json:"head_block"`
	Longest     bool             `json:"longest"`
	CommonBlock crypto.Signature `json:"common_block"`
	Height      int              `json:"height"`
	Length      int              `json:"length"`
	Peers       []PeerForkInfo   `json:"peers"`
}

type ForkByHeightLengthAndPeersCount []Fork

func (a ForkByHeightLengthAndPeersCount) Len() int {
	return len(a)
}

func (a ForkByHeightLengthAndPeersCount) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a ForkByHeightLengthAndPeersCount) Less(i, j int) bool {
	if a[i].Height > a[j].Height {
		return true
	}
	if a[i].Length > a[j].Length {
		return true
	}
	return  len(a[i].Peers) > len(a[j].Peers)
}


type PeerDescription struct {
	Address   net.IP           `json:"address"`
	Port      int              `json:"port"`
	Nonce     int              `json:"nonce"`
	Name      string           `json:"name"`
	Version   proto.Version    `json:"version"`
	Connected bool             `json:"connected"`
	LastBlock crypto.Signature `json:"last_block"`
}

type NodeForkInfo struct {
	Address    net.IP        `json:"address"`
	Nonce      int           `json:"nonce"`
	Name       string        `json:"name"`
	Version    proto.Version `json:"version"`
	OnFork     Fork          `json:"on_fork"`
	OtherForks []Fork        `json:"other_forks"`
}
