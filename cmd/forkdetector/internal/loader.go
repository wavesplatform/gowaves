package internal

import (
	"bytes"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	blockReceiveTimeout = time.Minute
)

type signaturesEvent struct {
	conn       *Conn
	signatures []crypto.Signature
}

func newSignaturesEvent(conn *Conn, signatures []crypto.Signature) signaturesEvent {
	s := make([]crypto.Signature, len(signatures))
	copy(s, signatures)
	return signaturesEvent{
		conn:       conn,
		signatures: s,
	}
}

type blockEvent struct {
	conn  *Conn
	block proto.Block
}

type blockRequest struct {
	block       crypto.Signature
	connections []*Conn
}

func newBlockRequest(block crypto.Signature, connections []*Conn) blockRequest {
	c := make([]*Conn, len(connections))
	copy(c, connections)
	return blockRequest{
		block:       block,
		connections: c,
	}
}

type requestQueue struct {
	loaded      bool
	blocks      []crypto.Signature
	connections map[crypto.Signature][]*Conn
}

func newRequestQueue() *requestQueue {
	return &requestQueue{
		blocks:      make([]crypto.Signature, 0),
		connections: make(map[crypto.Signature][]*Conn),
	}
}

func (q *requestQueue) String() string {
	sb := strings.Builder{}
	sb.WriteRune('(')
	sb.WriteString(strconv.Itoa(len(q.connections)))
	sb.WriteRune(')')
	sb.WriteRune('[')
	for i, s := range q.blocks {
		if i != 0 {
			sb.WriteRune(' ')
		}
		ss := s.String()
		sb.WriteString(ss[:6])
		sb.WriteRune('.')
		sb.WriteRune('.')
		sb.WriteString(ss[len(ss)-6:])
		if i == 0 && q.loaded {
			sb.WriteRune(' ')
			sb.WriteRune('|')
		}
	}
	sb.WriteRune(']')
	return sb.String()
}

func (q *requestQueue) enqueue(block crypto.Signature, conn *Conn) {
	list, ok := q.connections[block]
	if ok {
		list = append(list, conn)
		q.connections[block] = list
		return
	}
	q.blocks = append(q.blocks, block)
	list = []*Conn{conn}
	q.connections[block] = list
}

func (q *requestQueue) load() bool {
	if len(q.blocks) == 0 || q.loaded {
		return false
	}
	q.loaded = true
	return true
}

func (q *requestQueue) locked() (blockRequest, bool) {
	if !q.loaded || len(q.blocks) == 0 {
		return blockRequest{}, false
	}
	block := q.blocks[0]
	c, ok := q.connections[block]
	if !ok {
		zap.S().DPanicf("No connections list for block %s", block.String())
	}
	return newBlockRequest(q.blocks[0], c), true
}

func (q *requestQueue) dispose() bool {
	if !q.loaded || len(q.blocks) == 0 {
		return false
	}
	var b crypto.Signature
	b, q.blocks = q.blocks[0], q.blocks[1:]
	delete(q.connections, b)
	q.loaded = false
	return true
}

type blocksLoader struct {
	wg              *sync.WaitGroup
	drawer          *drawer
	requestCh       chan signaturesEvent
	blockCh         chan blockEvent
	notificationsCh chan crypto.Signature
	shutdownCh      chan struct{}
	queue           *requestQueue
	requestTimer    *time.Timer
}

func newBlockLoader(wg *sync.WaitGroup, drawer *drawer) *blocksLoader {
	return &blocksLoader{
		wg:              wg,
		drawer:          drawer,
		requestCh:       make(chan signaturesEvent),
		blockCh:         make(chan blockEvent),
		notificationsCh: make(chan crypto.Signature),
		shutdownCh:      make(chan struct{}),
		queue:           newRequestQueue(),
		requestTimer:    time.NewTimer(blockReceiveTimeout),
	}
}

func (l *blocksLoader) start() {
	defer l.wg.Done()
	for {
		select {
		case <-l.shutdownCh:
			return

		case e := <-l.requestCh: // Signature synchronizer requested to download some blocks
			zap.S().Debugf("[BLD] Blocks requested")
			for _, s := range e.signatures {
				l.queue.enqueue(s, e.conn)
			}
			zap.S().Debugf("[BLD] ( REQUESTING BLOCKS 1")
			l.requestBlock()
			zap.S().Debugf("[BLD] ) BLOCKS REQUESTED 1")

		case <-l.requestTimer.C:
			zap.S().Warnf("[BLD] Failed to receive block in time")
			if !l.queue.dispose() {
				zap.S().Debugf("[BLD] NOTHING TO DISPOSE")
			}
			zap.S().Debugf("[BLD] ( REQUESTING BLOCKS 2")
			l.requestBlock()
			zap.S().Debugf("[BLD] ) BLOCKS REQUESTED 2")

		case e := <-l.blockCh:
			zap.S().Debugf("[%s][BLD] Received block '%s'", e.conn.RawConn.RemoteAddr(), e.block.BlockSignature.String())
			l.requestTimer.Stop()
			pending, ok := l.queue.locked()
			if !ok {
				zap.S().Debugf("[%s][BLD] NO PENDING BLOCKS", e.conn.RawConn.RemoteAddr())
				continue
			}
			if pending.block != e.block.BlockSignature {
				zap.S().Warnf("[%s][BLD] Unexpected block '%s'", e.conn.RawConn.RemoteAddr(), e.block.BlockSignature.String())
				continue
			}
			if pending.connections[0] != e.conn {
				zap.S().Warnf("[%s][BLD] Expected block '%s' but from unexpected connection, was requested on %s", e.conn.RawConn.RemoteAddr(), e.block.BlockSignature.String(), pending.connections[0].RawConn.RemoteAddr())
				continue
			}
			err := l.drawer.appendBlock(e.block)
			if err != nil {
				zap.S().Fatalf("[%s][BLD] Failed to save new block: %v", e.conn.RawConn.RemoteAddr(), err)
				return
			}
			zap.S().Debugf("[%s][BLD] ( NOTIFYING ABOUT %s", e.conn.RawConn.RemoteAddr(), e.block.BlockSignature.String())
			l.notificationsCh <- e.block.BlockSignature
			if !l.queue.dispose() {
				zap.S().Debugf("[BLD] NOTHING TO DISPOSE")
			}
			zap.S().Debugf("[%s][BLD] ) NOTIFICATIONS SENT %s", e.conn.RawConn.RemoteAddr(), e.block.BlockSignature.String())
			zap.S().Debugf("[BLD] ( REQUESTING BLOCKS 3")
			l.requestBlock()
			zap.S().Debugf("[BLD] ) BLOCKS REQUESTED 3")
		}
	}
}

func (l *blocksLoader) shutdownSink() chan<- struct{} {
	return l.shutdownCh
}

func (l *blocksLoader) requestBlock() {
	ok := l.queue.load()
	if !ok {
		zap.S().Warnf("[BLD] No block to request or block already requested: %s", l.queue.String())
		return
	}
	pending, _ := l.queue.locked()
	conn := pending.connections[0]
	zap.S().Infof("[%s][BLD] Requesting block %s: %s", conn.RawConn.RemoteAddr(), pending.block, l.queue.String())
	buf := new(bytes.Buffer)
	m := proto.GetBlockMessage{BlockID: pending.block}
	_, err := m.WriteTo(buf)
	if err != nil {
		zap.S().Errorf("[%s][BLD] Failed to serialize block request message: %v", conn.RawConn.RemoteAddr(), err)
		return
	}
	l.requestTimer.Reset(blockReceiveTimeout)
	_, err = conn.Send(buf.Bytes())
	if err != nil {
		zap.S().Errorf("[%s][BLD] Failed to send block request message: %v", conn.RawConn.RemoteAddr(), err)
		return
	}
}

func (l *blocksLoader) requestsSink() chan<- signaturesEvent {
	return l.requestCh
}

func (l *blocksLoader) blocksSink() chan<- blockEvent {
	return l.blockCh
}

func (l *blocksLoader) notificationsTap() <-chan crypto.Signature {
	return l.notificationsCh
}

type Loader struct {
	interrupt           <-chan struct{}
	drawer              *drawer
	wg                  *sync.WaitGroup
	synchronizers       map[*Conn]*signaturesSynchronizer
	blocksLoader        *blocksLoader
	done                chan struct{}
	newConnectionsCh    chan *Conn
	closedConnectionsCh chan *Conn
	scoreCh             chan *Conn
	signaturesCh        chan signaturesEvent
	notificationsCh     chan crypto.Signature
}

func NewLoader(interrupt <-chan struct{}, drawer *drawer) (*Loader, error) {
	wg := &sync.WaitGroup{}
	bl := newBlockLoader(wg, drawer)
	return &Loader{
		interrupt:           interrupt,
		drawer:              drawer,
		wg:                  wg,
		synchronizers:       make(map[*Conn]*signaturesSynchronizer),
		blocksLoader:        bl,
		done:                make(chan struct{}),
		newConnectionsCh:    make(chan *Conn),
		closedConnectionsCh: make(chan *Conn),
		scoreCh:             make(chan *Conn),
		signaturesCh:        make(chan signaturesEvent),
	}, nil
}

func (l *Loader) Start() <-chan struct{} {
	l.wg.Add(1)
	go l.blocksLoader.start()
	go func() {
		for {
			select {
			case <-l.interrupt:
				zap.S().Debugf("[LDR] Shutting down loader...")
				l.blocksLoader.shutdownSink() <- struct{}{}
				zap.S().Debugf("[LDR] Shutting down %d synchronizers", len(l.synchronizers))
				for _, s := range l.synchronizers {
					s.shutdownSink() <- struct{}{}
				}
				l.wg.Wait()
				close(l.done)
				return

			case c := <-l.newConnectionsCh:
				// Start a new signatures synchronizer
				_, ok := l.synchronizers[c]
				if ok {
					zap.S().Errorf("[%s][LDR] Repetitive attempt to register signatures synchronizer", c.RawConn.RemoteAddr())
					continue
				}
				s := newSignaturesSynchronizer(l.wg, l.drawer, l.blocksRequests(), c)
				l.synchronizers[c] = s
				l.wg.Add(1)
				go s.start()

			case c := <-l.closedConnectionsCh:
				// Shutting down signatures synchronizer
				zap.S().Debugf("[%s][LDR] Connection closed, shutting down signatures synchronizer", c.RawConn.RemoteAddr())
				s, ok := l.synchronizers[c]
				if !ok {
					zap.S().Errorf("[%s][LDR] No signatures synchronizer found", c.RawConn.RemoteAddr())
					continue
				}
				delete(l.synchronizers, c)
				s.shutdownSink() <- struct{}{}

			case c := <-l.scoreCh:
				// New score on connection
				s, ok := l.synchronizers[c]
				if !ok {
					zap.S().Errorf("[%s][LDR] No signatures synchronizer", c.RawConn.RemoteAddr())
					continue
				}
				s.score() <- struct{}{}

			case e := <-l.signaturesCh:
				// Handle new signatures
				zap.S().Debugf("[%s][LDR] Received %d signatures: %s", e.conn.RawConn.RemoteAddr(), len(e.signatures), logSignatures(e.signatures))
				s, ok := l.synchronizers[e.conn]
				if !ok {
					zap.S().Errorf("[%s][LDR] No signatures synchronizer", e.conn.RawConn.RemoteAddr())
					continue
				}
				s.signatures() <- e.signatures

			case e := <-l.blocksLoader.notificationsTap():
				// Notify synchronizers about new block applied by blocks loader
				syncs := make([]*signaturesSynchronizer, len(l.synchronizers))
				i := 0
				for _, s := range l.synchronizers {
					syncs[i] = s
					i++
				}
				go func(synchronizers []*signaturesSynchronizer) {
					zap.S().Debugf("[LDR] Block notification received: %s", e.String())
					zap.S().Debugf("[LDR] Notifying %d synchronizers", len(l.synchronizers))
					for _, s := range synchronizers {
						s.block() <- e
					}
				}(syncs)
			}
		}
	}()
	return l.done
}

func (l *Loader) NewConnections() chan<- *Conn {
	return l.newConnectionsCh
}

func (l *Loader) ClosedConnections() chan<- *Conn {
	return l.closedConnectionsCh
}

func (l *Loader) Score() chan<- *Conn {
	return l.scoreCh
}

func (l *Loader) Signatures() chan<- signaturesEvent {
	return l.signaturesCh
}

func (l *Loader) Blocks() chan<- blockEvent {
	return l.blocksLoader.blocksSink()
}

func (l *Loader) blocksRequests() chan<- signaturesEvent {
	return l.blocksLoader.requestsSink()
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
