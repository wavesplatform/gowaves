package internal

import (
	"bytes"
	"github.com/pkg/errors"
	"github.com/seiflotfy/cuckoofilter"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
	"net"
	"strings"
)

const (
	million         = 1000000
	defaultCapacity = 10 * million
	signaturesCount = 100
)

type readyEvent struct {
	conn *Conn
}

type signaturesEvent struct {
	conn       *Conn
	signatures []crypto.Signature
}

type blockEvent struct {
	conn  *Conn
	block proto.Block
}

type blockRequest struct {
	conn      *Conn
	signature crypto.Signature
}

type Loader struct {
	interrupt         <-chan struct{}
	storage           *storage
	existingBlocks    *cuckoo.Filter
	pendingSignatures map[uint64][]crypto.Signature
	pendingBlocks     []blockRequest
	done              chan struct{}
	readyCh           chan readyEvent
	signaturesCh      chan signaturesEvent
	blockCh           chan blockEvent
	requestCh         chan blockRequest
}

func NewLoader(interrupt <-chan struct{}, storage *storage) (*Loader, error) {
	zap.S().Debugf("Loading existing blocks signatures...")
	signatures, err := storage.AllSignatures()
	if err != nil {
		zap.S().Errorf("Failed to load existing blocks: %v", err)
		return nil, errors.Wrap(err, "failed to load existing blocks signatures")
	}
	f := cuckoo.NewFilter(defaultCapacity)
	for _, s := range signatures {
		f.InsertUnique(s[:])
	}
	zap.S().Debugf("Loaded %d existing blocks signatures", f.Count())
	return &Loader{
		interrupt:         interrupt,
		storage:           storage,
		existingBlocks:    f,
		pendingSignatures: make(map[uint64][]crypto.Signature),
		pendingBlocks:     make([]blockRequest, 0),
		done:              make(chan struct{}),
		readyCh:           make(chan readyEvent),
		signaturesCh:      make(chan signaturesEvent),
		blockCh:           make(chan blockEvent),
		requestCh:         make(chan blockRequest),
	}, nil
}

func (l *Loader) Start() <-chan struct{} {
	go func() {
		for {
			select {
			case <-l.interrupt:
				zap.S().Debugf("Shutting down loader...")
				close(l.done)
				return

			case e := <-l.readyCh:
				ip, _, err := splitAddr(e.conn.RawConn.RemoteAddr())
				if err != nil {
					zap.S().Errorf("[%s] Failed to extract peer IP address: %v", e.conn.RawConn.RemoteAddr(), err)
					continue
				}
				p, ok := l.pendingSignatures[hash(ip)]
				if ok && len(p) > 0 {
					zap.S().Errorf("[%s] Signatures already requested", e.conn.RawConn.RemoteAddr())
					continue
				}
				ok, err = l.requestSignatures(e.conn, ip)
				if err != nil {
					zap.S().Warnf("[%s] Failed to request signatures: %v", e.conn.RawConn.RemoteAddr(), err)
					continue
				}
				if ok {
					zap.S().Debugf("[%s] Signatures requested", e.conn.RawConn.RemoteAddr())
				}

			case e := <-l.signaturesCh:
				zap.S().Debugf("[%s] Received %d signatures: %s", e.conn.RawConn.RemoteAddr(), len(e.signatures), logSignatures(e.signatures))
				ip, _, err := splitAddr(e.conn.RawConn.RemoteAddr())
				if err != nil {
					zap.S().Warnf("[%s] Failed to extract peer IP address: %v", e.conn.RawConn.RemoteAddr(), err)
					continue
				}
				pending, ok := l.pendingSignatures[hash(ip)]
				if !ok {
					zap.S().Warnf("[%s] No pending signatures", e.conn.RawConn.RemoteAddr())
					continue
				}
				unheard := skip(e.signatures, pending)
				nonexistent := make([]crypto.Signature, 0)
				for _, s := range unheard {
					ok := l.existingBlocks.Lookup(s[:])
					if ok {
						continue
					}
					nonexistent = append(nonexistent, s)
				}
				if len(nonexistent) > 0 {
					zap.S().Debugf("[%s] Requesting blocks: %s", e.conn.RawConn.RemoteAddr(), logSignatures(nonexistent))
					appended := false
					for _, s := range nonexistent {
						r := blockRequest{e.conn, s}
						ok := l.pendBlockRequest(r)
						appended = appended || ok
					}
					if appended {
						zap.S().Debugf("[%s] PUT BLOCK '%s' TO REQUEST CH 1", e.conn.RawConn.RemoteAddr(), l.pendingBlocks[0].signature.String())
						l.requestCh <- l.pendingBlocks[0]
					}
					continue
				}
				s := unheard[len(unheard)-1]
				zap.S().Debugf("[%s] Moving peer pointer to block '%s'", e.conn.RawConn.RemoteAddr(), s.String())
				ok, err = l.storage.appendBlockSignature(s, ip)
				if err != nil {
					zap.S().Errorf("[%s] Failed to move peer pointer to block '%s': %v", e.conn.RawConn.RemoteAddr(), s.String(), err)
					continue
				}
				if !ok {
					zap.S().DPanicf("[%s] Attempt to move peer pointer to nonexistent block '%s'", e.conn.RawConn.RemoteAddr(), s.String())
					continue
				}
				ok, err = l.requestSignatures(e.conn, ip)
				if err != nil {
					zap.S().Warnf("[%s] Failed to request signatures: %v", e.conn.RawConn.RemoteAddr(), err)
					continue
				}
				if ok {
					zap.S().Debugf("[%s] Signatures requested", e.conn.RawConn.RemoteAddr())
				}

			case e := <-l.blockCh:
				zap.S().Debugf("[%s] Received block '%s'", e.conn.RawConn.RemoteAddr(), e.block.BlockSignature.String())
				ip, _, err := splitAddr(e.conn.RawConn.RemoteAddr())
				if err != nil {
					zap.S().Errorf("[%s] Failed to parse peer address: %v", e.conn.RawConn.RemoteAddr(), err)
					continue
				}
				if len(l.pendingBlocks) == 0 {
					zap.S().Debugf("[%s] NO PENDING BLOCKS", e.conn.RawConn.RemoteAddr())
					continue
				}
				if l.pendingBlocks[0].signature != e.block.BlockSignature {
					zap.S().Warnf("[%s] Unexpected block '%s'", e.conn.RawConn.RemoteAddr(), e.block.BlockSignature.String())
					continue
				}
				if l.pendingBlocks[0].conn != e.conn {
					zap.S().Warnf("[%s] Expected block '%s' but from unexpected connection", e.conn.RawConn.RemoteAddr(), e.block.BlockSignature.String())
					continue
				}
				if l.existingBlocks.Lookup(e.block.BlockSignature[:]) {
					zap.S().DPanicf("[%s] Somehow block '%s' already in cache", e.conn.RawConn.RemoteAddr(), e.block.BlockSignature.String())
					continue
				}
				err = l.storage.handleBlock(e.block, ip)
				if err != nil {
					zap.S().Errorf("[%s] Failed to save new block: %v", e.conn.RawConn.RemoteAddr(), err)
					continue
				}
				ok := l.existingBlocks.InsertUnique(e.block.BlockSignature[:])
				if !ok {
					zap.S().DPanicf("[%s] Attempt to insert already inserted block ID '%s'", e.conn.RawConn.RemoteAddr(), e.block.BlockSignature.String())
				}

				l.pendingBlocks = l.pendingBlocks[1:] // Pop first block signature from the queue
				if len(l.pendingBlocks) == 0 {
					zap.S().Debugf("[%s] NO BLOCKS TO REQUEST", e.conn.RawConn.RemoteAddr())
					ok, err := l.requestSignatures(e.conn, ip)
					if err != nil {
						zap.S().Errorf("[%s] Failed to request new signatures: %v", e.conn.RawConn.RemoteAddr(), err)
					}
					if ok {
						zap.S().Debugf("[%s] Signatures requested", e.conn.RawConn.RemoteAddr())
					}
					continue
				}
				zap.S().Debugf("[%s] PUT BLOCK '%s' TO REQUEST CH 2", e.conn.RawConn.RemoteAddr(), l.pendingBlocks[0].signature.String())
				l.requestCh <- l.pendingBlocks[0] // Request next block leaving it in the queue
			}
		}
	}()
	go func() {
		for {
			select {
			case br := <-l.requestCh:
				err := l.requestBlock(br.conn, br.signature)
				if err != nil {
					zap.S().Errorf("[%s] Failed to request block '%s': %v", br.conn.RawConn.RemoteAddr(), br.signature.String(), err)
				}
				zap.S().Debugf("[%s] BLOCK REQUESTED '%s'", br.conn.RawConn.RemoteAddr(), br.signature.String())
			}
		}
	}()
	return l.done
}

func (l *Loader) Ready() chan<- readyEvent {
	return l.readyCh
}

func (l *Loader) Signatures() chan<- signaturesEvent {
	return l.signaturesCh
}

func (l *Loader) Blocks() chan<- blockEvent {
	return l.blockCh
}

func (l *Loader) requestSignatures(conn *Conn, ip net.IP) (bool, error) {
	s, err := l.storage.frontBlocks(ip, signaturesCount)
	if err != nil {
		return false, err
	}
	m := proto.GetSignaturesMessage{Blocks: s}
	buf := new(bytes.Buffer)
	_, err = m.WriteTo(buf)
	if err != nil {
		return false, err
	}
	_, err = conn.Send(buf.Bytes())
	if err != nil {
		return false, err
	}
	l.pendingSignatures[hash(ip)] = s
	return true, nil
}

func (l *Loader) requestBlock(conn *Conn, signature crypto.Signature) error {
	buf := new(bytes.Buffer)
	m := proto.GetBlockMessage{BlockID: signature}
	_, err := m.WriteTo(buf)
	if err != nil {
		return err
	}
	_, err = conn.Send(buf.Bytes())
	if err != nil {
		return err
	}
	return nil
}

func (l *Loader) pendBlockRequest(request blockRequest) bool {
	for _, r := range l.pendingBlocks {
		if r.signature == request.signature {
			zap.S().Debugf("[%s] ALREADY IN QUEUE; %s", request.conn.RawConn.RemoteAddr(), request.signature.String())
			return false
		}
	}
	l.pendingBlocks = append(l.pendingBlocks, request)
	zap.S().Debugf("[%s] ADDED TO QUEUE; %s", request.conn.RawConn.RemoteAddr(), request.signature.String())
	return true
}

func logSignatures(signatures []crypto.Signature) string {
	sb := strings.Builder{}
	sb.WriteRune('[')
	for i, s := range signatures {
		if i != 0 {
			sb.WriteRune(' ')
		}
		ss := s.String()
		sb.WriteString(ss[:6])
		sb.WriteRune('.')
		sb.WriteRune('.')
		sb.WriteString(ss[len(ss)-6:])
	}
	sb.WriteRune(']')
	return sb.String()
}

func contains(a []crypto.Signature, e crypto.Signature) bool {
	for i := 0; i < len(a); i++ {
		if a[i] == e {
			return true
		}
	}
	return false
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

func skip(a, c []crypto.Signature) []crypto.Signature {
	var i int
	for i = 0; i < len(a); i++ {
		if !contains(c, a[i]) {
			break
		}
	}
	return a[i:]
}
