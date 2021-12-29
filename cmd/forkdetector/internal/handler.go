package internal

import (
	"bufio"
	"bytes"
	"math/big"
	"net"
	"strings"
	"time"

	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
)

const (
	defaultApplication = "waves"
)

type ConnHandler struct {
	scheme            byte
	name              string
	nonce             uint64
	publicAddress     proto.HandshakeTCPAddr
	registry          *Registry
	newConnectionCh   chan<- *Conn
	closeConnectionCh chan<- *Conn
	scoreCh           chan<- *Conn
	idsCh             chan<- idsEvent
	blockCh           chan<- blockEvent
}

func NewConnHandler(scheme byte, name string, nonce uint64, addr net.TCPAddr, registry *Registry, newConn, closeConn, score chan<- *Conn, ids chan<- idsEvent, blocks chan<- blockEvent) *ConnHandler {
	return &ConnHandler{
		scheme:            scheme,
		name:              name,
		nonce:             nonce,
		publicAddress:     proto.HandshakeTCPAddr(addr),
		registry:          registry,
		newConnectionCh:   newConn,
		closeConnectionCh: closeConn,
		scoreCh:           score,
		idsCh:             ids,
		blockCh:           blocks,
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
		if strings.Contains(err.Error(), "tcp addr is too large") {
			err = h.registry.PeerHostile(conn.RawConn.RemoteAddr(), 0, "N/A", proto.Version{})
			if err != nil {
				zap.S().Errorf("[%s] Failed to update peer info: %v", conn.RawConn.RemoteAddr(), err)
			}
		}
		conn.Stop(StopImmediately)
		return
	}
	err = h.registry.Check(conn.RawConn.RemoteAddr(), ih.AppName)
	if err != nil {
		zap.S().Errorf("[%s] Unacceptable peer: %v", conn.RawConn.RemoteAddr(), err)
		conn.Stop(StopImmediately)
		if !ih.DeclaredAddr.Empty() {
			a := net.TCPAddr(ih.DeclaredAddr)
			err = h.registry.PeerHostile(&a, ih.NodeNonce, ih.NodeName, ih.Version)
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
		a := net.TCPAddr(ih.DeclaredAddr)
		err = h.registry.PeerGreeted(&a, ih.NodeNonce, ih.NodeName, ih.Version)
		if err != nil {
			zap.S().Warnf("[%s] Failed to register accepted peer: %v", conn.RawConn.RemoteAddr(), err)
			return
		}
	}
	err = h.registry.Activate(conn.RawConn.RemoteAddr(), ih)
	if err != nil {
		zap.S().Errorf("[%s] Failed to update peer state: %v", conn.RawConn.RemoteAddr(), err)
		conn.Stop(StopImmediately)
		return
	}
	zap.S().Debugf("[%s] Successful handshake with '%s' (nonce=%d, ver=%s, da=%s)", conn.RawConn.RemoteAddr(), ih.NodeName, ih.NodeNonce, ih.Version.String(), ih.DeclaredAddr.String())
	h.newConnectionCh <- conn
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
	if err != nil {
		zap.S().Errorf("[%s] Failed to suggest the version to handshake with: %v", conn.RawConn.RemoteAddr(), err)
		conn.Stop(StopImmediately)
		err := h.registry.PeerDiscarded(conn.RawConn.RemoteAddr())
		if err != nil {
			zap.S().Errorf("[%s] Failed to update connection state: %v", conn.RawConn.RemoteAddr(), err)
		}
		return
	}
	zap.S().Debugf("[%s] Trying to handshake with version %s", conn.RawConn.RemoteAddr(), ver.String())
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
		if strings.Contains(err.Error(), "tcp addr is too large") {
			err = h.registry.PeerHostile(conn.RawConn.RemoteAddr(), 0, "N/A", proto.Version{})
			if err != nil {
				zap.S().Errorf("[%s] Failed to update peer info: %v", conn.RawConn.RemoteAddr(), err)
			}
			return
		}
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
			a := net.TCPAddr(ih.DeclaredAddr)
			err = h.registry.PeerHostile(&a, ih.NodeNonce, ih.NodeName, ih.Version)
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
	err = h.registry.Activate(conn.RawConn.RemoteAddr(), ih)
	if err != nil {
		zap.S().Errorf("[%s] Failed to update connection state: %v", conn.RawConn.RemoteAddr(), err)
		conn.Stop(StopImmediately)
		return
	}
	zap.S().Debugf("[%s] Successful handshake with '%s' (nonce=%d, ver=%s, da=%s)", conn.RawConn.RemoteAddr(), ih.NodeName, ih.NodeNonce, ih.Version.String(), ih.DeclaredAddr.String())
	h.newConnectionCh <- conn
}

func (h *ConnHandler) OnReceive(conn *Conn, buf []byte) {
	header := proto.Header{}
	err := header.UnmarshalBinary(buf)
	if err != nil {
		zap.S().Errorf("[%s] Failed to unmarshal message header: %v", conn.RawConn.RemoteAddr(), err)
		return
	}
	switch header.ContentID {
	case proto.ContentIDGetPeers:
		var m proto.GetPeersMessage
		err = m.UnmarshalBinary(buf)
		if err != nil {
			zap.S().Warnf("[%s] Failed to unmarshal GetPeers message: %v", conn.RawConn.RemoteAddr(), err)
			return
		}
		h.replyWithEmptyPeers(conn)
	case proto.ContentIDPeers:
		var m proto.PeersMessage
		err = m.UnmarshalBinary(buf)
		if err != nil {
			zap.S().Warnf("[%s] Failed to unmarshal Peers message: %v", conn.RawConn.RemoteAddr(), err)
			return
		}
		addresses := make([]net.TCPAddr, len(m.Peers))
		for i, p := range m.Peers {
			addresses[i] = net.TCPAddr{IP: p.Addr, Port: int(p.Port)}
		}
		n := h.registry.AppendAddresses(addresses)
		if n > 0 {
			zap.S().Debugf("[%s] Appended %d new addresses", conn.RawConn.RemoteAddr(), n)
		}
	case proto.ContentIDScore:
		var m proto.ScoreMessage
		err = m.UnmarshalBinary(buf)
		if err != nil {
			zap.S().Warnf("[%s] Failed to unmarshal Score message: %v", conn.RawConn.RemoteAddr(), err)
			return
		}
		score := new(big.Int).SetBytes(m.Score)
		zap.S().Debugf("[%s] Received Score %s", conn.RawConn.RemoteAddr(), score.String())
		h.scoreCh <- conn
	case proto.ContentIDSignatures:
		var m proto.SignaturesMessage
		err = m.UnmarshalBinary(buf)
		if err != nil {
			zap.S().Warnf("[%s] Failed to unmarshal Signatures message: %v", conn.RawConn.RemoteAddr(), err)
			return
		}
		ids := make([]proto.BlockID, len(m.Signatures))
		for i, sig := range m.Signatures {
			ids[i] = proto.NewBlockIDFromSignature(sig)
		}
		h.idsCh <- newIdsEvent(conn, ids)
	case proto.ContentIDBlock:
		var m proto.BlockMessage
		err = m.UnmarshalBinary(buf)
		if err != nil {
			zap.S().Warnf("[%s] Failed to unmarshal Block message: %v", conn.RawConn.RemoteAddr(), err)
			return
		}
		// Applying block
		var b proto.Block
		err = b.UnmarshalBinary(m.BlockBytes, h.scheme)
		if err != nil {
			zap.S().Warnf("[%s] Failed to unmarshal block: %v", conn.RawConn.RemoteAddr(), err)
			return
		}
		h.blockCh <- blockEvent{conn: conn, block: &b}
	}
}

func (h *ConnHandler) OnClose(conn *Conn) {
	h.closeConnectionCh <- conn
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

func (h *ConnHandler) replyWithEmptyPeers(conn *Conn) {
	m := proto.PeersMessage{}
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
