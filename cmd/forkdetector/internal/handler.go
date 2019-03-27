package internal

import (
	"bufio"
	"encoding/binary"
	"github.com/go-errors/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/atomic"
	"go.uber.org/zap"
	"io"
	"io/ioutil"
	"math/big"
	"net"
	"sync"
	"time"
)

const (
	peerExchangeInterval = 30 * time.Second
	defaultReadTimeout   = 30 * time.Second
	defaultWriteTimeout  = 30 * time.Second
	headerMagicBytes     = 0x12345678
	maxMessageSize       = 2 * 1024 * 1024 // 2 MB
	signaturesBatchLen   = 101
)

type handler struct {
	interrupt      <-chan struct{}
	conn           net.Conn
	loader         *blockLoader
	closed         *atomic.Bool
	peersRequested *atomic.Bool
	addresses      chan<- []net.TCPAddr
	v              proto.Version
}

func NewHandler(interrupt <-chan struct{}, conn net.Conn, storage *storage, id PeerDesignation, addresses chan<- []net.TCPAddr, v proto.Version) *handler {
	zap.S().Debugf("Creating handler for connection to '%s'", conn.RemoteAddr())
	h := &handler{
		interrupt:      interrupt,
		conn:           conn,
		loader:         newBlockLoader(storage, id),
		closed:         atomic.NewBool(false),
		peersRequested: atomic.NewBool(false),
		addresses:      addresses,
		v:              v,
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
			h.replyWithPeers()
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
		case proto.ContentIDScore:
			mb, err := h.readMessage(r, size)
			if err != nil {
				zap.S().Warnf("Failed to read Score message from '%s': %v", h.conn.RemoteAddr(), err)
				if IsConnectionClosed(err) {
					return
				}
				continue
			}
			var s proto.ScoreMessage
			err = s.UnmarshalBinary(mb)
			if err != nil {
				zap.S().Warnf("Failed to unmarshal Score message from '%s': %v", h.conn.RemoteAddr(), err)
			}
			zap.S().Debugf("Received Score message from '%s'", h.conn.RemoteAddr())
			score := big.NewInt(0).SetBytes(s.Score)
			zap.S().Debugf("New score of '%s' is %s", h.conn.RemoteAddr(), score.String())
			go h.requestBlockSignatures()
		case proto.ContentIDSignatures:
			mb, err := h.readMessage(r, size)
			if err != nil {
				zap.S().Warnf("Failed to read Signatures message from '%s': %v", h.conn.RemoteAddr(), err)
				if IsConnectionClosed(err) {
					return
				}
				continue
			}
			var m proto.SignaturesMessage
			err = m.UnmarshalBinary(mb)
			if err != nil {
				zap.S().Warnf("Failed to unmarshal Signatures message from '%s': %v", h.conn.RemoteAddr(), err)
			}
			zap.S().Debugf("Received Signatures message with %d block signatures from '%s'", len(m.Signatures), h.conn.RemoteAddr())
			if len(m.Signatures) > 2 && m.Signatures[0] == m.Signatures[1] {
				zap.S().Warnf("REPEATED FIRST SIG: %s, '%s', %s", m.Signatures[0].String(), h.conn.RemoteAddr(), h.v.String())
			}
			err = h.loader.appendSignatures(m.Signatures)
			if err != nil {
				zap.S().Warnf("Failed to append signature from '%s': %v", h.conn.RemoteAddr(), err)
				continue
			}
			if h.loader.hasPending() {
				h.requestBlock(h.loader.pending()[0])
			} else {
				zap.S().Infof("No blocks to request from '%s', requesting more signatures...", h.conn.RemoteAddr())
				h.requestBlockSignatures()
			}
		case proto.ContentIDBlock:
			mb, err := h.readMessage(r, size)
			if err != nil {
				zap.S().Warnf("Failed to read Block message from '%s': %v", h.conn.RemoteAddr(), err)
				if IsConnectionClosed(err) {
					return
				}
				continue
			}
			var m proto.BlockMessage
			err = m.UnmarshalBinary(mb)
			if err != nil {
				zap.S().Warnf("Failed to unmarshal Block message from '%s': %v", h.conn.RemoteAddr(), err)
			}
			zap.S().Debugf("Received Block message from '%s'", h.conn.RemoteAddr())
			// Applying block
			var b proto.Block
			err = b.UnmarshalBinary(m.BlockBytes)
			if err != nil {
				zap.S().Warnf("Failed to unmarshal block received from '%s': %v", h.conn.RemoteAddr(), err)
				return
			}
			appended := h.loader.appendBlock(b)
			if !appended {
				zap.S().Debugf("Unrequested block %s from '%s' was dropped", b.BlockSignature.String(), h.conn.RemoteAddr())
				continue
			}
			if h.loader.hasPending() {
				h.requestBlock(h.loader.pending()[0])
				continue
			}
			err = h.loader.dump()
			if err != nil {
				zap.S().Warnf("Failed to dump blocks received from '%s': %v", h.conn.RemoteAddr(), err)
			}
			go h.requestBlockSignatures()
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

func (h *handler) replyWithPeers() {
	err := h.conn.SetWriteDeadline(time.Now().Add(defaultWriteTimeout))
	if err != nil {
		zap.S().Warnf("Failed to set write timout on the connection to '%s': %v", h.conn.RemoteAddr(), err)
		return
	}
	rq := proto.PeersMessage{}
	b, err := rq.MarshalBinary()
	if err != nil {
		zap.S().Warnf("Failed to marshal Peers message to bytes: %v", err)
		return
	}
	err = writeToConn(h.conn, b)
	if err != nil {
		zap.S().Errorf("Failed to send Peers message to '%s': %v", h.conn.RemoteAddr(), err)
		return
	}
	zap.S().Debugf("Replied to '%s' with empty peers message", h.conn.RemoteAddr())
}

func (h *handler) requestBlockSignatures() {
	if h.loader.hasPending() {
		zap.S().Warn("There are pending blocks to receive from '%s'", h.conn.RemoteAddr())
		return
	}
	fs, err := h.loader.front()
	if err != nil {
		zap.S().Warnf("Failed to request block signatures from '%s': %v", h.conn.RemoteAddr(), err)
		return
	}
	zap.S().Debugf("Sending %d known block signatures to '%s'", len(fs), h.conn.RemoteAddr())
	err = h.conn.SetWriteDeadline(time.Now().Add(defaultWriteTimeout))
	if err != nil {
		zap.S().Warnf("Failed to set write timout on the connection to '%s': %v", h.conn.RemoteAddr(), err)
		return
	}
	m := proto.GetSignaturesMessage{Blocks: fs}
	b, err := m.MarshalBinary()
	if err != nil {
		zap.S().Warnf("Failed to marshal GetSignatures message to bytes: %v", err)
		return
	}
	err = writeToConn(h.conn, b)
	if err != nil {
		zap.S().Errorf("Failed to send GetSignatures message to '%s': %v", h.conn.RemoteAddr(), err)
		return
	}
	zap.S().Debugf("Requested new block signatures from '%s'", h.conn.RemoteAddr())
}

func (h *handler) requestBlock(s crypto.Signature) {
	err := h.conn.SetWriteDeadline(time.Now().Add(defaultWriteTimeout))
	if err != nil {
		zap.S().Warnf("Failed to set write timout on the connection to '%s': %v", h.conn.RemoteAddr(), err)
		return
	}
	m := proto.GetBlockMessage{BlockID: s}
	b, err := m.MarshalBinary()
	if err != nil {
		zap.S().Warnf("Failed to marshal GetBlock message to bytes: %v", err)
		return
	}
	err = writeToConn(h.conn, b)
	if err != nil {
		zap.S().Errorf("Failed to send GetBlock message to '%s': %v", h.conn.RemoteAddr(), err)
		return
	}
	zap.S().Debugf("Block %s requested from '%s'", s.String(), h.conn.RemoteAddr())
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

type blockLoader struct {
	storage *storage
	id      PeerDesignation
	mu      sync.Mutex
	present []crypto.Signature
	upfront []crypto.Signature
	blocks  map[crypto.Signature]proto.Block
}

func newBlockLoader(storage *storage, id PeerDesignation) *blockLoader {
	return &blockLoader{
		storage: storage,
		id:      id,
		mu:      sync.Mutex{},
	}
}

func (l *blockLoader) front() ([]crypto.Signature, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	f, err := l.storage.frontBlocks(l.id, signaturesBatchLen)
	if err != nil {
		return nil, err
	}
	l.present = make([]crypto.Signature, len(f))
	copy(l.present, f)
	l.upfront = nil
	l.blocks = nil
	return f, nil
}

func (l *blockLoader) appendSignatures(signatures []crypto.Signature) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	common := intersect(l.present, signatures)
	if len(common) == 0 {
		return errors.New("no common block signatures")
	}
	l.present = nil
	r := skip(signatures, common)
	known := make([]crypto.Signature, 0)
	for _, s := range r {
		ok, err := l.storage.appendBlockSignature(s, l.id)
		if err != nil {
			return err
		}
		if !ok {
			break
		}
		known = append(known, s)
	}
	r = skip(r, known)
	l.upfront = make([]crypto.Signature, len(r))
	copy(l.upfront, r)
	l.blocks = make(map[crypto.Signature]proto.Block)
	return nil
}

func (l *blockLoader) appendBlock(b proto.Block) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	bs := b.BlockSignature
	for _, s := range l.upfront {
		if bs == s {
			l.blocks[bs] = b
			return true
		}
	}
	return false
}

func (l *blockLoader) hasPending() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, s := range l.upfront {
		if _, ok := l.blocks[s]; !ok {
			return true
		}
	}
	return false
}

func (l *blockLoader) pending() []crypto.Signature {
	l.mu.Lock()
	defer l.mu.Unlock()
	r := make([]crypto.Signature, 0)
	for _, s := range l.upfront {
		if _, ok := l.blocks[s]; !ok {
			r = append(r, s)
		}
	}
	return r
}

func (l *blockLoader) dump() error {
	for _, s := range l.upfront {
		b, ok := l.blocks[s]
		if !ok {
			return errors.Errorf("block %s was not loaded", s)
		}
		err := l.storage.handleBlock(b, l.id)
		if err != nil {
			return err
		}
	}
	return nil
}

func intersect(a, b []crypto.Signature) []crypto.Signature {
	r := make([]crypto.Signature, 0)
	for i := 0; i < len(a); i++ {
		e := a[i]
		if contains(b, e) {
			r = append(r, e)
		}
	}
	return r
}

func contains(a []crypto.Signature, e crypto.Signature) bool {
	for i := 0; i < len(a); i++ {
		if a[i] == e {
			return true
		}
	}
	return false
}

func skip(a, c []crypto.Signature) []crypto.Signature {
	var i int
	for i = 0; i < len(a); i++ {
		if !contains(c, a[i]) {
			break
		}
	}
	return a[i:]
}
