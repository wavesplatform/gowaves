package internal

import (
	"bytes"
	"strings"
	"sync"
	"time"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
)

const (
	blockReceiveTimeout = time.Minute
	batchSize           = 50
)

type loader struct {
	wg              *sync.WaitGroup
	drawer          *drawer
	requestCh       chan signaturesEvent
	blockCh         chan blockEvent
	notificationsCh chan crypto.Signature
	shutdownCh      chan struct{}
	queue           *requestQueue
	pending         *pendingQueue
	requestTimer    *time.Timer
}

func newLoader(wg *sync.WaitGroup, drawer *drawer) *loader {
	return &loader{
		wg:              wg,
		drawer:          drawer,
		requestCh:       make(chan signaturesEvent),
		blockCh:         make(chan blockEvent),
		notificationsCh: make(chan crypto.Signature),
		shutdownCh:      make(chan struct{}),
		queue:           new(requestQueue),
		pending:         new(pendingQueue),
		requestTimer:    time.NewTimer(blockReceiveTimeout),
	}
}

func (l *loader) start() {
	defer l.wg.Done()
	for {
		select {
		case <-l.shutdownCh:
			zap.S().Debugf("[LDR] Shutting down loader")
			//close(l.requestCh)
			//close(l.blockCh)
			close(l.notificationsCh)
			//close(l.shutdownCh)
			zap.S().Debugf("[LDR] Shutdown complete")
			return

		case e := <-l.requestCh: // Signature synchronizer requested to download some blocks
			zap.S().Debugf("[LDR] Blocks requested: %s", logSignatures(e.signatures))

			for _, s := range e.signatures {
				l.queue.enqueue(s, e.conn)
			}
			zap.S().Debugf("[LDR] ( REQUESTING BLOCKS 1")
			l.requestBlocks(nil)
			zap.S().Debugf("[LDR] ) BLOCKS REQUESTED 1")

		case <-l.requestTimer.C:
			zap.S().Warnf("[LDR] Failed to receive blocks in time")
			// Not all pending blocks were received during the given period of time
			// If there is a block sequence that can be applied, apply it until first unreceived block
			l.applyBlocks()
			if l.pending.len() > 0 { // For other unreceived blocks re-request them if possible from other peers
				zap.S().Debugf("[LDR] ( REQUESTING BLOCKS 2")
				l.requestBlocks(l.pending.connections())
				zap.S().Debugf("[LDR] ) BLOCKS REQUESTED 2")
			}

		case e := <-l.blockCh:
			zap.S().Debugf("[%s][LDR] Received block '%s'", e.conn.RawConn.RemoteAddr(), e.block.BlockSignature.String())
			zap.S().Debugf("[LDR] Pending blocks: %s", l.pending.String())
			l.pending.update(e.block)
			// Apply all sequentially received blocks
			l.applyBlocks()
			// In case of all pending blocks were received
			//   - Stop the timer
			//   - Request next batch of blocks from queue
			if l.pending.len() == 0 {
				l.requestTimer.Stop()
				zap.S().Debugf("[LDR] ( REQUESTING BLOCKS 3")
				l.requestBlocks(nil)
				zap.S().Debugf("[LDR] ) BLOCKS REQUESTED 3")
			}
		}
	}
}

func (l *loader) requestBlocks(exclusion []*Conn) {
	// Request `batchSize` blocks or less
	for i := l.pending.len(); i < batchSize; i++ {
		sig, conn, ok := l.queue.pickRandomly(exclusion)
		if !ok {
			break // No more blocks left in queue, aborting
		}
		// Request one block from the connection
		err := l.requestBlock(sig, conn)
		if err != nil {
			// If there is an error, unpick the block, and try to pick it from another node (exclude currently selected node) on the next iteration.
			exclusion = append(exclusion, conn)
			l.queue.unpick()
			i--
			zap.S().Warnf("[LDR] Will request block '%s' again excluding peers %v", sig.String(), exclusion)
			continue
		}
		// If everything is OK, save information about requested block and connection to pending blocks storage
		l.pending.enqueue(sig, conn)
	}
	// Reset the timer
	l.requestTimer.Reset(blockReceiveTimeout)
}

func (l *loader) requestBlock(sig crypto.Signature, conn *Conn) error {
	zap.S().Infof("[%s][LDR] Requesting block '%s'", conn.RawConn.RemoteAddr(), sig.String())
	if conn == nil {
		zap.S().Fatalf("Empty connection to request block '%s'", sig.String())
		return nil
	}
	buf := new(bytes.Buffer)
	m := proto.GetBlockMessage{BlockID: sig}
	_, err := m.WriteTo(buf)
	if err != nil {
		zap.S().Errorf("[%s][LDR] Failed to serialize block '%s': %v", conn.RawConn.RemoteAddr(), sig.String(), err)
		return err
	}
	_, err = conn.Send(buf.Bytes())
	if err != nil {
		zap.S().Errorf("[%s][LDR] Failed to request block '%s': %v", conn.RawConn.RemoteAddr(), sig.String(), err)
		return err
	}
	return nil
}

func (l *loader) applyBlocks() {
	for block, ok := l.pending.dequeue(); ok; block, ok = l.pending.dequeue() {
		zap.S().Infof("Applying block '%s", block.BlockSignature.String())
		err := l.drawer.appendBlock(block)
		if err != nil {
			zap.S().Fatalf("[LDR] Failed to apply block '%s': %v", block.BlockSignature.String(), err)
			return
		}
		zap.S().Debugf("[LDR] ( NOTIFYING ABOUT BLOCK '%s'", block.BlockSignature.String())
		l.notificationsCh <- block.BlockSignature
		zap.S().Debugf("[LDR] ) NOTIFICATIONS SENT '%s'", block.BlockSignature.String())
	}
}

func (l *loader) shutdownSink() chan<- struct{} {
	return l.shutdownCh
}

func (l *loader) requestsSink() chan<- signaturesEvent {
	return l.requestCh
}

func (l *loader) blocksSink() chan<- blockEvent {
	return l.blockCh
}

func (l *loader) notificationsTap() <-chan crypto.Signature {
	return l.notificationsCh
}

func logSignatures(signatures []crypto.Signature) string {
	sb := strings.Builder{}
	sb.WriteRune('[')
	for i, s := range signatures {
		if i != 0 {
			sb.WriteRune(' ')
		}
		sb.WriteString(s.ShortString())
	}
	sb.WriteRune(']')
	return sb.String()
}
