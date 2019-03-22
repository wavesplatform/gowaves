package internal

import (
	"bufio"
	"encoding/binary"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/atomic"
	"go.uber.org/zap"
	"io"
	"io/ioutil"
	"net"
	"time"
)

const (
	peerExchangeInterval = 30 * time.Second
	defaultReadTimeout   = 30 * time.Second
	defaultWriteTimeout  = 30 * time.Second
	headerMagicBytes     = 0x12345678
	maxMessageSize       = 2 * 1024 * 1024 // 2 MB
)

type handler struct {
	interrupt      <-chan struct{}
	conn           net.Conn
	closed         *atomic.Bool
	peersRequested *atomic.Bool
	addresses      chan<- []net.TCPAddr
}

func NewHandler(interrupt <-chan struct{}, conn net.Conn, addresses chan<- []net.TCPAddr) *handler {
	zap.S().Debugf("Creating handler for connection to '%s'", conn.RemoteAddr())
	h := &handler{
		interrupt:      interrupt,
		conn:           conn,
		closed:         atomic.NewBool(false),
		peersRequested: atomic.NewBool(false),
		addresses:      addresses,
	}
	go h.handle()
	go h.read()
	return h
}

func (h *handler) handle() {
	peerExchangeTicker := time.NewTicker(peerExchangeInterval)
	for {
		select {
		case <-h.interrupt:
			if h.closed.CAS(false, true) {
				h.close()
			}
			return
		case <-peerExchangeTicker.C:
			go h.sendGetPeers()
		}
	}
}

func (h *handler) read() {
	r := bufio.NewReader(h.conn)
	for {
		if h.closed.Load() {
			return
		}
		err := h.conn.SetReadDeadline(time.Now().Add(defaultReadTimeout))
		if err != nil {
			zap.S().Warnf("Failed to set read deadline: %v", err)
			if IsConnectionClosed(err) {
				return
			}
			continue
		}
		header, err := r.Peek(9)
		if err != nil {
			if IsConnectionClosed(err) {
				return
			}
			zap.S().Warnf("Failed to read message header from '%s': %v", h.conn.RemoteAddr(), err)
			continue
		}
		size := binary.BigEndian.Uint32(header[0:4])
		magic := binary.BigEndian.Uint32(header[4:8])
		content := header[8]
		if size > maxMessageSize {
			zap.S().Warnf("Received a message of size %d and content type %d from '%s', message will be dropped because of too big size", size, content, h.conn.RemoteAddr())
			err = h.dropMessage(r, size)
			if IsConnectionClosed(err) {
				return
			}
			continue
		}
		if magic != headerMagicBytes {
			zap.S().Warn("Incorrect magic bytes from '%s'", h.conn.RemoteAddr())
			err = h.dropMessage(r, size)
			if IsConnectionClosed(err) {
				return
			}
			continue
		}
		switch content {
		case proto.ContentIDGetPeers:
			mb, err := h.readMessage(r, size)
			if err != nil {
				zap.S().Warnf("Failed to read GetPeers message from '%s': %v", h.conn.RemoteAddr(), err)
				if IsConnectionClosed(err) {
					return
				}
				continue
			}
			var gp proto.GetPeersMessage
			err = gp.UnmarshalBinary(mb)
			if err != nil {
				zap.S().Warnf("Failed to unmarshal GetPeers message from '%s': %v", h.conn.RemoteAddr(), err)
			}
			zap.S().Debugf("Received GetPeers message from '%s'", h.conn.RemoteAddr())
		case proto.ContentIDPeers:
			h.peersRequested.Store(false)
			mb, err := h.readMessage(r, size)
			if err != nil {
				zap.S().Warnf("Failed to read Peers message from '%s': %v", h.conn.RemoteAddr(), err)
				if IsConnectionClosed(err) {
					return
				}
				continue
			}
			var ps proto.PeersMessage
			err = ps.UnmarshalBinary(mb)
			if err != nil {
				zap.S().Warnf("Failed to unmarshal Peers message from '%s': %v", h.conn.RemoteAddr(), err)
			}
			addresses := make([]net.TCPAddr, len(ps.Peers))
			for i, p := range ps.Peers {
				addresses[i] = net.TCPAddr{IP: p.Addr, Port: int(p.Port)}
			}
			h.addresses <- addresses
		default:
			err := h.dropMessage(r, size)
			if err != nil {
				zap.S().Warnf("Failed to drop unexpected message from '%s': %v", h.conn.RemoteAddr(), err)
				if IsConnectionClosed(err) {
					return
				}
				continue
			}
		}
	}
}

func (h *handler) readMessage(r io.Reader, s uint32) ([]byte, error) {
	if h.closed.Load() {
		return nil, nil
	}
	b := make([]byte, 4+int(s))
	_, err := io.ReadFull(r, b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (h *handler) dropMessage(r io.Reader, s uint32) error {
	if h.closed.Load() {
		return nil
	}
	_, err := io.CopyN(ioutil.Discard, r, int64(4+s))
	return err
}

func (h *handler) sendGetPeers() {
	if h.peersRequested.CAS(false, true) {
		zap.S().Debugf("Sending GetPeers message to '%s'", h.conn.RemoteAddr())
		err := h.conn.SetWriteDeadline(time.Now().Add(defaultWriteTimeout))
		rq := proto.GetPeersMessage{}
		b, err := rq.MarshalBinary()
		if err != nil {
			zap.S().Warnf("Failed to marshal GetPeers message to bytes: %v", err)
			h.peersRequested.Store(false)
			return
		}
		err = writeToConn(h.conn, b)
		if err != nil {
			zap.S().Errorf("Failed to send GetPeers message to '%s': %v", h.conn.RemoteAddr(), err)
			h.peersRequested.Store(false)
			return
		}
	}
}

func writeToConn(conn net.Conn, data []byte) error {
	var start, c int
	var err error
	for {
		if c, err = conn.Write(data[start:]); err != nil {
			return err
		}
		start += c
		if c == 0 || start == len(data) {
			break
		}
	}
	return nil
}

func (h *handler) close() {
	err := h.conn.Close()
	if err != nil {
		zap.S().Warnf("Failed to close connection to '%s': %v", h.conn.RemoteAddr(), err)
	}
}
