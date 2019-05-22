package internal

import (
	"bytes"
	"github.com/pkg/errors"
	"github.com/seiflotfy/cuckoofilter"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/atomic"
	"go.uber.org/zap"
	"net"
	"strings"
	"sync"
)

const (
	million         = 1000000
	defaultCapacity = 10 * million
	signaturesCount = 100
)

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

type blocksCache struct {
	mu    sync.Mutex
	cache *cuckoo.Filter
}

func newBlocksCache(signatures []crypto.Signature) *blocksCache {
	f := cuckoo.NewFilter(defaultCapacity)
	for _, s := range signatures {
		f.InsertUnique(s[:])
	}
	zap.S().Debugf("Loaded %d existing blocks signatures", f.Count())
	return &blocksCache{cache: f}
}

func (c *blocksCache) put(signature crypto.Signature) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.cache.InsertUnique(signature[:])
}

func (c *blocksCache) contain(signature crypto.Signature) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.cache.Lookup(signature[:])
}

type signaturesSynchronizer struct {
	wg               *sync.WaitGroup
	storage          *storage
	cache            *blocksCache
	requestBlocksCh  chan<- signaturesEvent
	conn             *Conn
	addr             net.IP
	pending          []crypto.Signature
	working          *atomic.Bool
	scoreCh          chan struct{}
	signaturesCh     chan []crypto.Signature
	receivedBlocksCh chan crypto.Signature
}

func newSignaturesSynchronizer(wg *sync.WaitGroup, storage *storage, cache *blocksCache, blocks chan<- signaturesEvent, conn *Conn) *signaturesSynchronizer {
	return &signaturesSynchronizer{
		wg:               wg,
		storage:          storage,
		cache:            cache,
		requestBlocksCh:  blocks,
		conn:             conn,
		addr:             extractIPAddress(conn.RawConn.RemoteAddr()),
		working:          atomic.NewBool(false),
		scoreCh:          make(chan struct{}),
		signaturesCh:     make(chan []crypto.Signature),
		receivedBlocksCh: make(chan crypto.Signature),
	}
}

func (s *signaturesSynchronizer) start() {
	defer s.wg.Done()
	s.working.Store(true)
	go func() {
		for s.working.Load() {
			select {
			case <-s.scoreCh:
				s.requestSignatures()

			case signatures := <-s.signaturesCh:
				unheard := skip(signatures, s.pending)
				nonexistent := make([]crypto.Signature, 0)
				for _, sig := range unheard {
					if s.cache.contain(sig) {
						continue
					}
					nonexistent = append(nonexistent, sig)
				}
				if len(nonexistent) > 0 {
					zap.S().Debugf("[%s] Requesting blocks: %s", s.conn.RawConn.RemoteAddr(), logSignatures(nonexistent))
					s.requestBlocksCh <- signaturesEvent{conn: s.conn, signatures: nonexistent}
					continue
				}

				last := unheard[len(unheard)-1]
				err := s.movePeerLink(last)
				if err != nil {
					zap.S().Errorf("[%s] Failed to handle signatures: %v", s.conn.RawConn.RemoteAddr(), err)
					continue
				}
				s.pending = nil
				s.requestSignatures()

			case bs := <-s.receivedBlocksCh:
				// Requested block received move peer pointer
				zap.S().Debugf("[%s] PENDED SIGS %s", s.conn.RawConn.RemoteAddr(), logSignatures(s.pending))
				var i int
				for i = 0; i < len(s.pending); i++ {
					if bs == s.pending[i] {
						zap.S().Debugf("[%s] HAS BLOCK IN PENDING %s", s.conn.RawConn.RemoteAddr(), bs.String())
						break
					}
				}
				var sig crypto.Signature
				zap.S().Debugf("[%s] i = %d, l = %d", s.conn.RawConn.RemoteAddr(), i, len(s.pending))
				if i < len(s.pending) {
					sig = s.pending[i]
					s.pending = s.pending[i+1:]
				} else {
					zap.S().DPanicf("[%s] Unexpected block notification: %s", s.conn.RawConn.RemoteAddr(), bs.String())
					continue
				}
				err := s.movePeerLink(sig)
				if err != nil {
					zap.S().Errorf("[%s] Failed to handle block: %v", s.conn.RawConn.RemoteAddr(), err)
					continue
				}
				if len(s.pending) == 0 {
					s.requestSignatures()
				}
			}
		}
	}()
}

func (s *signaturesSynchronizer) stop() {
	s.working.CAS(true, false)
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
	if len(s.pending) > 0 {
		zap.S().Debugf("[%s] Signatures already requested", s.conn.RawConn.RemoteAddr())
		return
	}
	signatures, err := s.storage.frontBlocks(s.addr, signaturesCount)
	if err != nil {
		zap.S().Errorf("[%s] Failed to request signatures: %v", s.conn.RawConn.RemoteAddr(), err)
		return
	}
	m := proto.GetSignaturesMessage{Blocks: signatures}
	buf := new(bytes.Buffer)
	_, err = m.WriteTo(buf)
	if err != nil {
		zap.S().Errorf("[%s] Failed to prepare the signatures request: %v", s.conn.RawConn.RemoteAddr(), err)
		return
	}
	_, err = s.conn.Send(buf.Bytes())
	if err != nil {
		zap.S().Errorf("[%s] Failed to send the signatures request: %v", s.conn.RawConn.RemoteAddr(), err)
		return
	}
	s.pending = signatures
	zap.S().Debugf("[%s] PENDING SIGNATURES: %s", s.conn.RawConn.RemoteAddr(), logSignatures(s.pending))
}

func (s *signaturesSynchronizer) movePeerLink(signature crypto.Signature) error {
	zap.S().Debugf("[%s] Moving peer link to block '%s'", s.conn.RawConn.RemoteAddr(), signature.String())
	err := s.storage.updatePeerLink(s.addr, signature)
	if err != nil {
		return err
	}
	return nil
}

type waitingList struct {
	signature   crypto.Signature
	connections []*Conn
}

func (w *waitingList) hasConnection(conn *Conn) bool {
	for _, c := range w.connections {
		if c == conn {
			return true
		}
	}
	return false
}

func (w *waitingList) request() blockRequest {
	return blockRequest{conn: w.connections[0], signature: w.signature}
}

func (w *waitingList) appendConnection(conn *Conn) bool {
	if w.hasConnection(conn) {
		return false
	}
	w.connections = append(w.connections, conn)
	return true
}

type Loader struct {
	interrupt           <-chan struct{}
	storage             *storage
	existingBlocks      *blocksCache
	wg                  sync.WaitGroup
	synchronizers       map[*Conn]*signaturesSynchronizer
	pendingBlocks       []waitingList
	done                chan struct{}
	newConnectionsCh    chan *Conn
	closedConnectionsCh chan *Conn
	scoreCh             chan *Conn
	signaturesCh        chan signaturesEvent
	blockCh             chan blockEvent
	blocksRequestsCh    chan signaturesEvent // Channel to request blocks downloading, used by signature synchronizers
	requestCh           chan blockRequest
}

func NewLoader(interrupt <-chan struct{}, storage *storage) (*Loader, error) {
	zap.S().Debugf("Loading existing blocks signatures...")
	signatures, err := storage.AllSignatures()
	if err != nil {
		zap.S().Errorf("Failed to load existing blocks: %v", err)
		return nil, errors.Wrap(err, "failed to load existing blocks signatures")
	}
	return &Loader{
		interrupt:           interrupt,
		storage:             storage,
		existingBlocks:      newBlocksCache(signatures),
		wg:                  sync.WaitGroup{},
		synchronizers:       make(map[*Conn]*signaturesSynchronizer),
		pendingBlocks:       make([]waitingList, 0),
		done:                make(chan struct{}),
		newConnectionsCh:    make(chan *Conn),
		closedConnectionsCh: make(chan *Conn),
		scoreCh:             make(chan *Conn),
		signaturesCh:        make(chan signaturesEvent),
		blockCh:             make(chan blockEvent),
		blocksRequestsCh:    make(chan signaturesEvent),
		requestCh:           make(chan blockRequest),
	}, nil
}

func (l *Loader) Start() <-chan struct{} {
	go func() {
		for {
			select {
			case <-l.interrupt:
				zap.S().Debugf("Shutting down loader...")
				for _, s := range l.synchronizers {
					s.stop()
				}
				l.wg.Wait()
				close(l.done)
				return

			case c := <-l.newConnectionsCh:
				// Start a new signatures synchronizer
				_, ok := l.synchronizers[c]
				if ok {
					zap.S().Errorf("[%s] Repetitive attempt to register signatures synchronizer", c.RawConn.RemoteAddr())
					continue
				}
				s := newSignaturesSynchronizer(&l.wg, l.storage, l.existingBlocks, l.blocksRequests(), c)
				l.synchronizers[c] = s
				l.wg.Add(1)
				s.start()

			case c := <-l.closedConnectionsCh:
				// Shutting down signatures synchronizer
				s, ok := l.synchronizers[c]
				if !ok {
					zap.S().Errorf("[%s] No signatures synchronizer", c.RawConn.RemoteAddr())
					continue
				}
				s.stop()

			case c := <-l.scoreCh:
				// New score on connection
				s, ok := l.synchronizers[c]
				if !ok {
					zap.S().Errorf("[%s] No signatures synchronizer", c.RawConn.RemoteAddr())
					continue
				}
				s.score() <- struct{}{}

			case e := <-l.signaturesCh:
				// Handle new signatures
				zap.S().Debugf("[%s] Received %d signatures: %s", e.conn.RawConn.RemoteAddr(), len(e.signatures), logSignatures(e.signatures))
				s, ok := l.synchronizers[e.conn]
				if !ok {
					zap.S().Errorf("[%s] No signatures synchronizer", e.conn.RawConn.RemoteAddr())
					continue
				}
				s.signatures() <- e.signatures

			case e := <-l.blocksRequestsCh:
				// Signature synchronizer requested to download some blocks
				// Decide which blocks should be actually downloaded and put them to queue
				// If block already in queue add the requester (signature synchronizer) to the notification list
				zap.S().Debugf("[%s] REQUESTING BLOCKS %s", e.conn.RawConn.RemoteAddr(), logSignatures(e.signatures))
				appended := false
				for _, requestedBlock := range e.signatures {
					exist := false
					for _, pendingBlock := range l.pendingBlocks {
						if pendingBlock.signature == requestedBlock {
							//zap.S().Debugf("[%s] BLOCK ALREADY IN QUEUE %s", e.conn.RawConn.RemoteAddr(), requestedBlock.String())
							if pendingBlock.appendConnection(e.conn) {
								//zap.S().Debugf("[%s] ADDED TO WAITING LIST OF BLOCK %s", e.conn.RawConn.RemoteAddr(), requestedBlock.String())
							}
							exist = true
							break
						}
					}
					if !exist {
						zap.S().Debugf("[%s] APPENDING BLOCK TO QUEUE %s", e.conn.RawConn.RemoteAddr(), requestedBlock.String())
						wl := waitingList{signature: requestedBlock}
						wl.appendConnection(e.conn)
						l.pendingBlocks = append(l.pendingBlocks, wl)
						appended = true
					}

				}
				if appended {
					zap.S().Debugf("[%s] PUT BLOCK '%s' TO REQUEST CH 1", e.conn.RawConn.RemoteAddr(), l.pendingBlocks[0].signature.String())
					l.requestCh <- l.pendingBlocks[0].request()
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
				if !l.pendingBlocks[0].hasConnection(e.conn) {
					zap.S().Warnf("[%s] Expected block '%s' but from unexpected connection", e.conn.RawConn.RemoteAddr(), e.block.BlockSignature.String())
					continue
				}
				if l.existingBlocks.contain(e.block.BlockSignature) {
					zap.S().DPanicf("[%s] Somehow block '%s' already in cache", e.conn.RawConn.RemoteAddr(), e.block.BlockSignature.String())
					continue
				}
				err = l.storage.handleBlock(e.block, ip) //TODO: just add the block to storage, don't move peer link here
				if err != nil {
					zap.S().Errorf("[%s] Failed to save new block: %v", e.conn.RawConn.RemoteAddr(), err)
					continue
				}
				ok := l.existingBlocks.put(e.block.BlockSignature)
				if !ok {
					zap.S().DPanicf("[%s] Attempt to insert already inserted block ID '%s'", e.conn.RawConn.RemoteAddr(), e.block.BlockSignature.String())
				}

				wl := l.pendingBlocks[0]
				for _, c := range wl.connections {
					s, ok := l.synchronizers[c]
					if !ok {
						zap.S().Debugf("[%s] Synchronizer not found", c.RawConn.RemoteAddr())
						continue
					}
					s.block() <- wl.signature
				}
				l.pendingBlocks = l.pendingBlocks[1:] // Pop first block signature from the queue
				if len(l.pendingBlocks) == 0 {
					zap.S().Debugf("[%s] NO BLOCKS TO REQUEST", e.conn.RawConn.RemoteAddr())
					continue
				}
				zap.S().Debugf("[%s] PUT BLOCK '%s' TO REQUEST CH 2", e.conn.RawConn.RemoteAddr(), l.pendingBlocks[0].signature.String())
				l.requestCh <- l.pendingBlocks[0].request() // Request next block leaving it in the queue
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
	return l.blockCh
}

func (l *Loader) blocksRequests() chan<- signaturesEvent {
	return l.blocksRequestsCh
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

//TODO: remove?
//func intersect(a, b []crypto.Signature) []crypto.Signature {
//	r := make([]crypto.Signature, 0)
//	for i := 0; i < len(a); i++ {
//		e := a[i]
//		if contains(b, e) {
//			r = append(r, e)
//		}
//	}
//	return r
//}

func skip(a, c []crypto.Signature) []crypto.Signature {
	var i int
	for i = 0; i < len(a); i++ {
		if !contains(c, a[i]) {
			break
		}
	}
	return a[i:]
}
