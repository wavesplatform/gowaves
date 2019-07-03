package internal

import (
	"bytes"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
	"net"
	"sync"
)

type signaturesSynchronizer struct {
	wg               *sync.WaitGroup
	drawer           *drawer
	requestBlocksCh  chan<- signaturesEvent
	conn             *Conn
	addr             net.IP
	requested        []crypto.Signature
	pending          map[crypto.Signature]struct{}
	shutdownCh       chan struct{}
	scoreCh          chan struct{}
	signaturesCh     chan []crypto.Signature
	receivedBlocksCh chan crypto.Signature
}

func newSignaturesSynchronizer(wg *sync.WaitGroup, drawer *drawer, blocks chan<- signaturesEvent, conn *Conn) *signaturesSynchronizer {
	return &signaturesSynchronizer{
		wg:               wg,
		drawer:           drawer,
		requestBlocksCh:  blocks,
		conn:             conn,
		addr:             extractIPAddress(conn.RawConn.RemoteAddr()),
		pending:          make(map[crypto.Signature]struct{}),
		shutdownCh:       make(chan struct{}),
		scoreCh:          make(chan struct{}),
		signaturesCh:     make(chan []crypto.Signature),
		receivedBlocksCh: make(chan crypto.Signature),
	}
}

func (s *signaturesSynchronizer) start() {
	defer s.wg.Done()
	for {
		select {
		case <-s.shutdownCh:
			zap.S().Debugf("[%s][SYN] Shutdown complete", s.conn.RawConn.RemoteAddr())
			return

		case <-s.scoreCh:
			zap.S().Debugf("[%s][SYN] New score received, requesting signature", s.conn.RawConn.RemoteAddr())
			s.requestSignatures()

		case signatures := <-s.signaturesCh:
			unheard := skip(signatures, s.requested)
			nonexistent := make([]crypto.Signature, 0)
			for _, sig := range unheard {
				ok, err := s.drawer.hasBlock(sig)
				if err != nil {
					zap.S().Fatalf("[%s][SYN] Failed to check block '%s' presence", s.conn.RawConn.RemoteAddr(), sig.String())
					return
				}
				if ok {
					continue
				}
				nonexistent = append(nonexistent, sig)
			}
			if len(nonexistent) > 0 {
				zap.S().Debugf("[%s][SYN] ( Requesting blocks: %s", s.conn.RawConn.RemoteAddr(), logSignatures(nonexistent))
				s.requested = nil
				for _, sig := range nonexistent {
					s.pending[sig] = struct{}{}
				}
				s.requestBlocksCh <- newSignaturesEvent(s.conn, nonexistent)
				zap.S().Debugf("[%s][SYN] ) Blocks REQUESTED: %s", s.conn.RawConn.RemoteAddr(), logSignatures(nonexistent))
				continue
			}

			last := unheard[len(unheard)-1]
			err := s.movePeer(last)
			if err != nil {
				zap.S().Fatalf("[%s][SYN] Failed to handle signatures: %v", s.conn.RawConn.RemoteAddr(), err)
				return
			}
			s.requested = nil
			s.requestSignatures()

		case blockSignature := <-s.receivedBlocksCh:
			if _, ok := s.pending[blockSignature]; ok {
				delete(s.pending, blockSignature)
				if len(s.pending) == 0 {
					err := s.movePeer(blockSignature)
					if err != nil {
						zap.S().Fatalf("[%s][SYN] Failed to update peer link: %v", s.conn.RawConn.RemoteAddr(), err)
						return
					}
					s.requestSignatures()
				}
			}
		}
	}
}

func (s *signaturesSynchronizer) shutdownSink() chan<- struct{} {
	return s.shutdownCh
}

func (s *signaturesSynchronizer) score() chan<- struct{} {
	return s.scoreCh
}

func (s *signaturesSynchronizer) signatures() chan<- []crypto.Signature {
	return s.signaturesCh
}

func (s *signaturesSynchronizer) block() chan<- crypto.Signature {
	return s.receivedBlocksCh
}

func (s *signaturesSynchronizer) requestSignatures() {
	if len(s.requested) > 0 {
		zap.S().Debugf("[%s][SYN] Signatures already requested", s.conn.RawConn.RemoteAddr())
		return
	}
	signatures, err := s.drawer.front(s.addr)
	if err != nil {
		zap.S().Fatalf("[%s][SYN] Failed to request signatures: %v", s.conn.RawConn.RemoteAddr(), err)
		return
	}
	m := proto.GetSignaturesMessage{Blocks: signatures}
	buf := new(bytes.Buffer)
	_, err = m.WriteTo(buf)
	if err != nil {
		zap.S().Errorf("[%s][SYN] Failed to prepare the signatures request: %v", s.conn.RawConn.RemoteAddr(), err)
		return
	}
	_, err = s.conn.Send(buf.Bytes())
	if err != nil {
		zap.S().Errorf("[%s][SYN] Failed to send the signatures request: %v", s.conn.RawConn.RemoteAddr(), err)
		return
	}
	s.requested = signatures
}

func (s *signaturesSynchronizer) movePeer(signature crypto.Signature) error {
	zap.S().Debugf("[%s][SYN] Moving peer link to block '%s'", s.conn.RawConn.RemoteAddr(), signature.String())
	err := s.drawer.movePeer(s.addr, signature)
	if err != nil {
		return err
	}
	return nil
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
