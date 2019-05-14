package internal

import (
	"bufio"
	"bytes"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
	"math/big"
	"net"
	"strings"
	"time"
)

const (
	defaultApplication = "waves"
)

type ConnHandler struct {
	scheme        byte
	name          string
	nonce         uint64
	publicAddress proto.HandshakeTCPAddr
	registry      *Registry
}

func NewConnHandler(scheme byte, name string, nonce uint64, addr proto.HandshakeTCPAddr, registry *Registry) *ConnHandler {
	return &ConnHandler{
		scheme:        scheme,
		name:          name,
		nonce:         nonce,
		publicAddress: addr,
		registry:      registry,
	}
}

func (h *ConnHandler) OnAccept(conn *Conn) {
	zap.S().Debugf("New incoming connection from %s", conn.RawConn.RemoteAddr())
	var ih proto.Handshake
	err := conn.RawConn.SetReadDeadline(time.Now().Add(handshakeTimeout))
	if err != nil {
		zap.S().Warnf("[%s] Failed to set read timeout: %v", conn.RawConn.RemoteAddr(), err)
		conn.Stop(StopImmediately)
		return
	}
	r := bufio.NewReader(conn.RawConn)
	_, err = ih.ReadFrom(r)
	if err != nil {
		zap.S().Warnf("[%s] Failed to receive handshake: %v", conn.RawConn.RemoteAddr(), err)
		conn.Stop(StopImmediately)
		return
	}
	err = h.registry.Check(conn.RawConn.RemoteAddr(), ih.AppName)
	if err != nil {
		zap.S().Errorf("[%s] Unacceptable peer: %v", conn.RawConn.RemoteAddr(), err)
		conn.Stop(StopImmediately)
		if !ih.DeclaredAddr.Empty() {
			err = h.registry.PeerHostile(ih.DeclaredAddr, ih.NodeNonce, ih.NodeName, ih.Version)
			if err != nil {
				zap.S().Errorf("[%s] Failed to update peer info: %v", conn.RawConn.RemoteAddr(), err)
			}
		}
		return
	}
	out := h.buildHandshake(ih.Version)
	_, err = out.WriteTo(conn.RawConn)
	if err != nil {
		zap.S().Warnf("[%s] Failed to send handshake: %v", conn.RawConn.RemoteAddr(), err)
		conn.Stop(StopImmediately)
		return
	}
	if !ih.DeclaredAddr.Empty() {
		err = h.registry.PeerGreeted(ih.DeclaredAddr, ih.NodeNonce, ih.NodeName, ih.Version)
		if err != nil {
			zap.S().Warnf("[%s] Failed to register accepted peer: %v", conn.RawConn.RemoteAddr(), err)
			return
		}
	}
	err = h.registry.Activate(conn.RawConn.RemoteAddr())
	if err != nil {
		zap.S().Errorf("[%s] Failed to update peer state: %v", conn.RawConn.RemoteAddr(), err)
		conn.Stop(StopImmediately)
		return
	}
	zap.S().Debugf("[%s] Successful handshake with '%s' (nonce=%d, ver=%s, da=%s)", conn.RawConn.RemoteAddr(), ih.NodeName, ih.NodeNonce, ih.Version.String(), ih.DeclaredAddr.String())
}

func (h *ConnHandler) OnConnect(conn *Conn) {
	zap.S().Debugf("New outgoing connection to %s", conn.RawConn.RemoteAddr())
	err := h.registry.PeerConnected(conn.RawConn.RemoteAddr())
	if err != nil {
		zap.S().Errorf("[%s] Failed to register new connection: %v", conn.RawConn.RemoteAddr(), err)
		conn.Stop(StopImmediately)
		err := h.registry.PeerDiscarded(conn.RawConn.RemoteAddr())
		if err != nil {
			zap.S().Errorf("[%s] Failed to update connection state: %v", conn.RawConn.RemoteAddr(), err)
		}
		return
	}
	ver, err := h.registry.SuggestVersion(conn.RawConn.RemoteAddr())
	zap.S().Debugf("[%s] Trying to handshake with version %s", conn.RawConn.RemoteAddr(), ver.String())
	if err != nil {
		zap.S().Errorf("[%s] Failed to suggest the version to handshake with: %v", conn.RawConn.RemoteAddr(), err)
		conn.Stop(StopImmediately)
		err := h.registry.PeerDiscarded(conn.RawConn.RemoteAddr())
		if err != nil {
			zap.S().Errorf("[%s] Failed to update connection state: %v", conn.RawConn.RemoteAddr(), err)
		}
		return
	}
	oh := h.buildHandshake(ver)
	err = conn.RawConn.SetWriteDeadline(time.Now().Add(handshakeTimeout))
	if err != nil {
		zap.S().Warnf("[%s] Failed to set write timeout on connection: %v", conn.RawConn.RemoteAddr(), err)
		conn.Stop(StopImmediately)
		err := h.registry.PeerDiscarded(conn.RawConn.RemoteAddr())
		if err != nil {
			zap.S().Errorf("[%s] Failed to update connection state: %v", conn.RawConn.RemoteAddr(), err)
		}
		return
	}
	_, err = oh.WriteTo(conn.RawConn)
	if err != nil {
		zap.S().Warnf("[%s] Failed to send handshake: %v", conn.RawConn.RemoteAddr(), err)
		conn.Stop(StopImmediately)
		err := h.registry.PeerDiscarded(conn.RawConn.RemoteAddr())
		if err != nil {
			zap.S().Errorf("[%s] Failed to update connection state: %v", conn.RawConn.RemoteAddr(), err)
		}
		return
	}
	var ih proto.Handshake
	err = conn.RawConn.SetReadDeadline(time.Now().Add(handshakeTimeout))
	if err != nil {
		zap.S().Warnf("[%s] Failed to set read timeout on connection %s: %v", conn.RawConn.RemoteAddr(), err)
		conn.Stop(StopImmediately)
		err := h.registry.PeerDiscarded(conn.RawConn.RemoteAddr())
		if err != nil {
			zap.S().Errorf("[%s] Failed to update connection state: %v", conn.RawConn.RemoteAddr(), err)
		}
		return
	}
	r := bufio.NewReader(conn.RawConn)
	_, err = ih.ReadFrom(r)
	if err != nil {
		zap.S().Warnf("[%s] Failed to receive handshake: %v", conn.RawConn.RemoteAddr(), err)
		conn.Stop(StopImmediately)
		err := h.registry.PeerDiscarded(conn.RawConn.RemoteAddr())
		if err != nil {
			zap.S().Errorf("[%s] Failed to update connection state: %v", conn.RawConn.RemoteAddr(), err)
		}
		return
	}
	err = h.registry.Check(conn.RawConn.RemoteAddr(), ih.AppName)
	if err != nil {
		zap.S().Errorf("[%s] Connection declined: %v", conn.RawConn.RemoteAddr(), err)
		conn.Stop(StopImmediately)
		if !ih.DeclaredAddr.Empty() {
			err = h.registry.PeerHostile(ih.DeclaredAddr, ih.NodeNonce, ih.NodeName, ih.Version)
			if err != nil {
				zap.S().Errorf("[%s] Failed to update connection state: %v", conn.RawConn.RemoteAddr(), err)
			}
		}
		return
	}
	err = h.registry.PeerGreeted(conn.RawConn.RemoteAddr(), ih.NodeNonce, ih.NodeName, ih.Version)
	if err != nil {
		zap.S().Errorf("[%s] Failed to update connection state: %v", conn.RawConn.RemoteAddr(), err)
		conn.Stop(StopImmediately)
		return
	}
	err = h.registry.Activate(conn.RawConn.RemoteAddr())
	if err != nil {
		zap.S().Errorf("[%s] Failed to update connection state: %v", conn.RawConn.RemoteAddr(), err)
		conn.Stop(StopImmediately)
		return
	}
	zap.S().Debugf("[%s] Successful handshake with '%s' (nonce=%d, ver=%s, da=%s)", conn.RawConn.RemoteAddr(), ih.NodeName, ih.NodeNonce, ih.Version.String(), ih.DeclaredAddr.String())
}

func (h *ConnHandler) OnReceive(c *Conn, buf []byte) {
	header := proto.Header{}
	err := header.UnmarshalBinary(buf)
	if err != nil {
		zap.S().Errorf("[%s] Failed to unmarshal message header: %v", c.RawConn.RemoteAddr(), err)
		return
	}
	switch header.ContentID {
	case proto.ContentIDGetPeers:
		var m proto.GetPeersMessage
		err = m.UnmarshalBinary(buf)
		if err != nil {
			zap.S().Warnf("[%s] Failed to unmarshal GetPeers message: %v", c.RawConn.RemoteAddr(), err)
			return
		}
		zap.S().Debugf("[%s] Received GetPeers message", c.RawConn.RemoteAddr())
		peers, err := h.registry.Addresses()
		if err != nil {
			zap.S().Warnf("[%s] Failed to get peers to reply with: %v", err)
			return
		}
		h.replyWithPeers(c, peers)
	case proto.ContentIDPeers:
		var m proto.PeersMessage
		err = m.UnmarshalBinary(buf)
		if err != nil {
			zap.S().Warnf("[%s] Failed to unmarshal Peers message: %v", c.RawConn.RemoteAddr(), err)
			return
		}
		addresses := make([]net.TCPAddr, len(m.Peers))
		for i, p := range m.Peers {
			addresses[i] = net.TCPAddr{IP: p.Addr, Port: int(p.Port)}
		}
		n := h.registry.AppendAddresses(addresses)
		if n > 0 {
			zap.S().Debugf("[%s] Appended %d new addresses", c.RawConn.RemoteAddr(), n)
		}
	case proto.ContentIDScore:
		var m proto.ScoreMessage
		err = m.UnmarshalBinary(buf)
		if err != nil {
			zap.S().Warnf("[%s] Failed to unmarshal Score message: %v", c.RawConn.RemoteAddr(), err)
			return
		}
		score := big.NewInt(0).SetBytes(m.Score)
		zap.S().Debugf("[%s] Received Score %s", c.RawConn.RemoteAddr(), score.String())
		//TODO: go h.requestBlockSignatures()
	case proto.ContentIDSignatures:
		var m proto.SignaturesMessage
		err = m.UnmarshalBinary(buf)
		if err != nil {
			zap.S().Warnf("[%s] Failed to unmarshal Signatures message: %v", c.RawConn.RemoteAddr(), err)
			return
		}
		zap.S().Debugf("[%s] Received Signatures message with %d block signatures", c.RawConn.RemoteAddr(), len(m.Signatures))
		//TODO: Process received signatures
		//err = h.loader.appendSignatures(m.Signatures)
		//if err != nil {
		//	zap.S().Warnf("Failed to append signature from '%s': %v", h.conn.RemoteAddr(), err)
		//	continue
		//}
		//if h.loader.hasPending() {
		//	h.requestBlock(h.loader.pending()[0])
		//} else {
		//	zap.S().Infof("No blocks to request from '%s', requesting more signatures...", h.conn.RemoteAddr())
		//	h.requestBlockSignatures()
		//}
	case proto.ContentIDBlock:
		var m proto.BlockMessage
		err = m.UnmarshalBinary(buf)
		if err != nil {
			zap.S().Warnf("[%s] Failed to unmarshal Block message: %v", c.RawConn.RemoteAddr(), err)
			return
		}
		zap.S().Debugf("[%s] Received Block message", c.RawConn.RemoteAddr())
		// Applying block
		var b proto.Block
		err = b.UnmarshalBinary(m.BlockBytes)
		if err != nil {
			zap.S().Warnf("[%s] Failed to unmarshal block: %v", c.RawConn.RemoteAddr(), err)
			return
		}
		//TODO: process received block
		//appended := h.loader.appendBlock(b)
		//if !appended {
		//	zap.S().Debugf("Unrequested block %s from '%s' was dropped", b.BlockSignature.String(), h.conn.RemoteAddr())
		//	continue
		//}
		//if h.loader.hasPending() {
		//	h.requestBlock(h.loader.pending()[0])
		//	continue
		//}
		//err = h.loader.dump()
		//if err != nil {
		//	zap.S().Warnf("Failed to dump blocks received from '%s': %v", h.conn.RemoteAddr(), err)
		//}
		//go h.requestBlockSignatures()
	}
}

func (h *ConnHandler) OnClose(conn *Conn) {
	err := h.registry.Deactivate(conn.RawConn.RemoteAddr())
	if err != nil {
		zap.S().Errorf("[%s] Failed to deactivate peer: %v", conn.RawConn.RemoteAddr(), err)
	}
}

func (h *ConnHandler) buildHandshake(ver proto.Version) *proto.Handshake {
	sb := strings.Builder{}
	sb.WriteString(defaultApplication)
	sb.WriteByte(h.scheme)
	return &proto.Handshake{
		AppName:      sb.String(),
		Version:      ver,
		NodeName:     h.name,
		NodeNonce:    h.nonce,
		DeclaredAddr: h.publicAddress,
		Timestamp:    proto.NewTimestampFromTime(time.Now()),
	}
}

func (h *ConnHandler) replyWithPeers(conn *Conn, peers []net.Addr) {
	infos := make([]proto.PeerInfo, len(peers))
	for i, p := range peers {
		info, err := proto.NewPeerInfoFromString(p.String())
		if err != nil {
			zap.S().Warnf("[%s] Invalid peer '%s': %v", p.String(), err)
			continue
		}
		infos[i] = info
	}
	m := proto.PeersMessage{Peers: infos}
	buf := new(bytes.Buffer)
	_, err := m.WriteTo(buf)
	if err != nil {
		zap.S().Warnf("[%s] Failed to send PeersMessage: %v", conn.RawConn.RemoteAddr(), err)
		return
	}
	_, err = conn.Send(buf.Bytes())
	if err != nil {
		zap.S().Warnf("[%s] Failed to send PeersMessage: %v", conn.RawConn.RemoteAddr(), err)
		return
	}
}
