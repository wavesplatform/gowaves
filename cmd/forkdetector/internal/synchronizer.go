package internal

import (
	"bytes"
	"net"
	"sync"

	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
)

type synchronizer struct {
	wg               *sync.WaitGroup
	drawer           *drawer
	requestBlocksCh  chan<- idsEvent
	conn             *Conn
	addr             net.IP
	requested        []proto.BlockID
	pending          map[proto.BlockID]struct{}
	shutdownCh       chan struct{}
	scoreCh          chan struct{}
	idsCh            chan []proto.BlockID
	receivedBlocksCh chan proto.BlockID
}

func newSynchronizer(wg *sync.WaitGroup, drawer *drawer, blocks chan<- idsEvent, conn *Conn) *synchronizer {
	return &synchronizer{
		wg:               wg,
		drawer:           drawer,
		requestBlocksCh:  blocks,
		conn:             conn,
		addr:             extractIPAddress(conn.RawConn.RemoteAddr()),
		pending:          make(map[proto.BlockID]struct{}),
		shutdownCh:       make(chan struct{}),
		scoreCh:          make(chan struct{}),
		idsCh:            make(chan []proto.BlockID),
		receivedBlocksCh: make(chan proto.BlockID),
	}
}

func (s *synchronizer) start() {
	defer s.wg.Done()
	for {
		select {
		case <-s.shutdownCh:
			zap.S().Debugf("[%s][SYN] Shutting down synchronizer for connection '%s'", s.conn.RawConn.RemoteAddr(), s.conn.String())
			zap.S().Debugf("[%s][SYN] Shutdown complete", s.conn.RawConn.RemoteAddr())
			return

		case <-s.scoreCh:
			zap.S().Debugf("[%s][SYN] New score received, requesting id", s.conn.RawConn.RemoteAddr())
			s.requestIds()

		case ids := <-s.idsCh:
			unheard := skip(ids, s.requested)
			if len(unheard) == 0 {
				s.requested = nil
				continue
			}
			nonexistent := make([]proto.BlockID, 0)
			for _, id := range unheard {
				ok, err := s.drawer.hasBlock(id)
				if err != nil {
					zap.S().Fatalf("[%s][SYN] Failed to check block '%s' presence", s.conn.RawConn.RemoteAddr(), id.String())
					return
				}
				if ok {
					continue
				}
				nonexistent = append(nonexistent, id)
			}
			if len(nonexistent) > 0 {
				zap.S().Debugf("[%s][SYN] ( Requesting blocks: %s", s.conn.RawConn.RemoteAddr(), logIds(nonexistent))
				s.requested = nil
				for _, id := range nonexistent {
					s.pending[id] = struct{}{}
				}
				s.requestBlocksCh <- newIdsEvent(s.conn, nonexistent)
				zap.S().Debugf("[%s][SYN] ) Blocks REQUESTED: %s", s.conn.RawConn.RemoteAddr(), logIds(nonexistent))
				continue
			}

			last := unheard[len(unheard)-1]
			err := s.movePeer(last)
			if err != nil {
				zap.S().Fatalf("[%s][SYN] Failed to handle ids: %v", s.conn.RawConn.RemoteAddr(), err)
				return
			}
			s.requested = nil
			s.requestIds()

		case blockId := <-s.receivedBlocksCh:
			if _, ok := s.pending[blockId]; ok {
				delete(s.pending, blockId)
				if len(s.pending) == 0 {
					err := s.movePeer(blockId)
					if err != nil {
						zap.S().Fatalf("[%s][SYN] Failed to update peer link: %v", s.conn.RawConn.RemoteAddr(), err)
						return
					}
					s.requestIds()
				}
			}
		}
	}
}

func (s *synchronizer) shutdownSink() chan<- struct{} {
	return s.shutdownCh
}

func (s *synchronizer) score() chan<- struct{} {
	return s.scoreCh
}

func (s *synchronizer) ids() chan<- []proto.BlockID {
	return s.idsCh
}

func (s *synchronizer) block() chan<- proto.BlockID {
	return s.receivedBlocksCh
}

func (s *synchronizer) requestIds() {
	if len(s.requested) > 0 {
		zap.S().Debugf("[%s][SYN] Ids already requested", s.conn.RawConn.RemoteAddr())
		return
	}
	ids, err := s.drawer.front(s.addr)
	if err != nil {
		zap.S().Fatalf("[%s][SYN] Failed to request ids: %v", s.conn.RawConn.RemoteAddr(), err)
		return
	}
	m := proto.GetBlockIdsMessage{Blocks: ids}
	buf := new(bytes.Buffer)
	_, err = m.WriteTo(buf)
	if err != nil {
		zap.S().Errorf("[%s][SYN] Failed to prepare the ids request: %v", s.conn.RawConn.RemoteAddr(), err)
		return
	}
	_, err = s.conn.Send(buf.Bytes())
	if err != nil {
		zap.S().Errorf("[%s][SYN] Failed to send the ids request: %v", s.conn.RawConn.RemoteAddr(), err)
		return
	}
	s.requested = ids
}

func (s *synchronizer) movePeer(id proto.BlockID) error {
	zap.S().Debugf("[%s][SYN] Moving peer link to block '%s'", s.conn.RawConn.RemoteAddr(), id.String())
	return s.drawer.movePeer(s.addr, id)
}

func contains(a []proto.BlockID, e proto.BlockID) (bool, int) {
	for i := 0; i < len(a); i++ {
		if a[i] == e {
			return true, i
		}
	}
	return false, -1
}

func skip(a, c []proto.BlockID) []proto.BlockID {
	var i int
	for i = 0; i < len(a); i++ {
		if ok, _ := contains(c, a[i]); !ok {
			break
		}
	}
	return a[i:]
}
