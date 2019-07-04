package internal

import (
	"bytes"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	blockReceiveTimeout = time.Minute
	batchSize           = 20
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

type requestQueue struct {
	picked      int
	blocks      []crypto.Signature
	connections map[crypto.Signature][]*Conn
	once        sync.Once
	rnd         *rand.Rand
}

func (q *requestQueue) init() {
	q.picked = -1
	q.blocks = make([]crypto.Signature, 0)
	q.connections = make(map[crypto.Signature][]*Conn)
	q.rnd = rand.New(rand.NewSource(time.Now().UnixNano()))
}

func (q *requestQueue) String() string {
	q.once.Do(q.init)

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
		if i == q.picked {
			sb.WriteRune(' ')
			sb.WriteRune('|')
		}
	}
	sb.WriteRune(']')
	return sb.String()
}

func (q *requestQueue) enqueue(block crypto.Signature, conn *Conn) {
	q.once.Do(q.init)

	if conn == nil {
		zap.S().Fatalf("Attempt to insert NIL connection into queue")
	}

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

func (q *requestQueue) pickRandomly() (crypto.Signature, *Conn, bool) {
	q.once.Do(q.init)

	if q.picked == len(q.blocks)-1 {
		return crypto.Signature{}, nil, false
	}
	q.picked++
	sig := q.blocks[q.picked]
	connections, ok := q.connections[sig]
	if !ok {
		zap.S().Fatalf("Failure to locate enqueued connection")
	}
	conn := connections[q.rnd.Intn(len(connections))]
	return sig, conn, true
}

func (q *requestQueue) dequeue(block crypto.Signature) {
	q.once.Do(q.init)

	ok, pos := contains(q.blocks, block)
	if !ok {
		return
	}
	q.blocks = q.blocks[:pos+copy(q.blocks[pos:], q.blocks[pos+1:])]
	delete(q.connections, block)
	q.picked--
}

func (q *requestQueue) reset() {
	q.picked = -1
}

type blocksLoader struct {
	wg              *sync.WaitGroup
	drawer          *drawer
	requestCh       chan signaturesEvent
	blockCh         chan blockEvent
	notificationsCh chan crypto.Signature
	shutdownCh      chan struct{}
	queue           *requestQueue
	pending         map[crypto.Signature]struct{}
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
		queue:           new(requestQueue),
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
			l.requestBlocks()
			zap.S().Debugf("[BLD] ) BLOCKS REQUESTED 1")

		case <-l.requestTimer.C:
			zap.S().Warnf("[BLD] Failed to receive block in time")
			// Not all pending blocks were received during the given period of time
			// If there is a block sequence that can be applied, apply it until first unreceived block
			// For other unreceived blocks re-request them if possible from other peers
			// Restart timer
			l.pending = nil // Dispose all pending blocks
			zap.S().Debugf("[BLD] ( REQUESTING BLOCKS 2")
			l.requestBlocks()
			zap.S().Debugf("[BLD] ) BLOCKS REQUESTED 2")

		case e := <-l.blockCh:
			zap.S().Debugf("[%s][BLD] Received block '%s'", e.conn.RawConn.RemoteAddr(), e.block.BlockSignature.String())
			// Put block in storage and mark it as received in pending

			// In case of all pending blocks were received
			//   - Stop the timer
			//   - Dequeue all pending blocks
			//   - Send notifications
			//   - Request next batch of blocks from queue

			l.requestTimer.Stop()
			//pending, ok := l.queue.locked()
			//if !ok {
			//	zap.S().Debugf("[%s][BLD] NO PENDING BLOCKS", e.conn.RawConn.RemoteAddr())
			//	continue
			//}
			//if pending.block != e.block.BlockSignature {
			//	zap.S().Warnf("[%s][BLD] Unexpected block '%s'", e.conn.RawConn.RemoteAddr(), e.block.BlockSignature.String())
			//	continue
			//}
			//if pending.connections[0] != e.conn {
			//	zap.S().Warnf("[%s][BLD] Expected block '%s' but from unexpected connection, was requested on %s", e.conn.RawConn.RemoteAddr(), e.block.BlockSignature.String(), pending.connections[0].RawConn.RemoteAddr())
			//	continue
			//}
			//err := l.drawer.appendBlock(e.block)
			//if err != nil {
			//	zap.S().Fatalf("[%s][BLD] Failed to save new block: %v", e.conn.RawConn.RemoteAddr(), err)
			//	return
			//}
			//zap.S().Debugf("[%s][BLD] ( NOTIFYING ABOUT %s", e.conn.RawConn.RemoteAddr(), e.block.BlockSignature.String())
			//l.notificationsCh <- e.block.BlockSignature
			//if !l.queue.dispose() {
			//	zap.S().Debugf("[BLD] NOTHING TO DISPOSE")
			//}
			zap.S().Debugf("[%s][BLD] ) NOTIFICATIONS SENT %s", e.conn.RawConn.RemoteAddr(), e.block.BlockSignature.String())
			zap.S().Debugf("[BLD] ( REQUESTING BLOCKS 3")
			l.requestBlocks()
			zap.S().Debugf("[BLD] ) BLOCKS REQUESTED 3")
		}
	}
}

func (l *blocksLoader) shutdownSink() chan<- struct{} {
	return l.shutdownCh
}

func (l *blocksLoader) requestBlocks(exclusion []*Conn) {
	// Request `batchSize` blocks or less
	for i := 0; i < batchSize; i++ {
		sig, conn, ok := l.queue.pickRandomly(exclusion)
		if !ok {
			break // No more blocks left in queue, aborting
		}
		// Request one block from the connection
		err := l.requestBlock(sig, conn)
		if err != nil {
			// If there is an error, unpick the block, and try to pick it from another node (exclude currently selected node).
			exclusion = append(exclusion, conn)
		}
		// If everything is OK, save information about requested block and connection to pending blocks storage
	}
	// Reset the timer

	//ok := l.queue.load()
	//if !ok {
	//	zap.S().Warnf("[BLD] No block to request or block already requested: %s", l.queue.String())
	//	return
	//}
	//pending, _ := l.queue.locked()
	//conn := pending.connections[0]
	//zap.S().Infof("[%s][BLD] Requesting block %s: %s", conn.RawConn.RemoteAddr(), pending.block, l.queue.String())
	//l.requestTimer.Reset(blockReceiveTimeout)
	//}
}

func (l *blocksLoader) requestBlock(sig crypto.Signature, conn *Conn) error {
	if conn == nil {
		zap.S().Fatalf("Empty connection to request block '%s'", sig.String())
		return nil
	}
	buf := new(bytes.Buffer)
	m := proto.GetBlockMessage{BlockID: sig}
	_, err := m.WriteTo(buf)
	if err != nil {
		zap.S().Errorf("[%s][BLD] Failed to serialize block '%s': %v", conn.RawConn.RemoteAddr(), sig.String(), err)
		return err
	}
	_, err = conn.Send(buf.Bytes())
	if err != nil {
		zap.S().Errorf("[%s][BLD] Failed to request block '%s': %v", conn.RawConn.RemoteAddr(), sig.String(), err)
		return err
	}
	return nil
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
	blocksCh            chan blockEvent
	blocksRequestsCh    chan signaturesEvent
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
		blocksCh:            make(chan blockEvent),
		blocksRequestsCh:    make(chan signaturesEvent),
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

			case conn := <-l.newConnectionsCh:
				// Start a new signatures synchronizer
				_, ok := l.synchronizers[conn]
				if ok {
					zap.S().Errorf("[%s][LDR] Repetitive attempt to register signatures synchronizer", conn.RawConn.RemoteAddr())
					continue
				}
				s := newSignaturesSynchronizer(l.wg, l.drawer, l.blocksRequests(), conn)
				l.synchronizers[conn] = s
				l.wg.Add(1)
				go s.start()

			case conn := <-l.closedConnectionsCh:
				// Shutting down signatures synchronizer
				zap.S().Debugf("[%s][LDR] Connection closed, shutting down signatures synchronizer", conn.RawConn.RemoteAddr())
				s, ok := l.synchronizers[conn]
				if !ok {
					zap.S().Errorf("[%s][LDR] No signatures synchronizer found", conn.RawConn.RemoteAddr())
					continue
				}
				delete(l.synchronizers, conn)
				s.shutdownSink() <- struct{}{}

			case conn := <-l.scoreCh:
				// New score on connection
				s, ok := l.synchronizers[conn]
				if !ok {
					zap.S().Errorf("[%s][LDR] No signatures synchronizer", conn.RawConn.RemoteAddr())
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

			case e := <-l.blocksCh:
				go func() {
					l.blocksLoader.blocksSink() <- e
				}()

			case e := <-l.blocksRequestsCh:
				go func() {
					l.blocksLoader.requestsSink() <- e
				}()
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
	return l.blocksCh
}

func (l *Loader) blocksRequests() chan<- signaturesEvent {
	return l.blocksRequestsCh
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
