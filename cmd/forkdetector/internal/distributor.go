package internal

import (
	"sync"

	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
)

type distributor struct {
	drawer              *drawer
	wg                  *sync.WaitGroup
	synchronizers       map[*Conn]*synchronizer
	blocksLoader        *loader
	shutdownCh          <-chan struct{}
	doneCh              chan struct{}
	newConnectionsCh    chan *Conn
	closedConnectionsCh chan *Conn
	scoreCh             chan *Conn
	idsCh               chan idsEvent
}

func NewDistributor(shutdown <-chan struct{}, drawer *drawer) (*distributor, error) {
	wg := &sync.WaitGroup{}
	bl := newLoader(wg, drawer)
	return &distributor{
		drawer:              drawer,
		wg:                  wg,
		synchronizers:       make(map[*Conn]*synchronizer),
		blocksLoader:        bl,
		shutdownCh:          shutdown,
		doneCh:              make(chan struct{}),
		newConnectionsCh:    make(chan *Conn),
		closedConnectionsCh: make(chan *Conn),
		scoreCh:             make(chan *Conn),
		idsCh:               make(chan idsEvent),
	}, nil
}

func (l *distributor) Start() <-chan struct{} {
	l.wg.Add(1)
	go l.blocksLoader.start()
	go func() {
		for {
			select {
			case <-l.shutdownCh:
				zap.S().Debugf("[DTR] Shutting down distributor...")
				zap.S().Debug("[DTR] Shutting down block loader")
				l.blocksLoader.shutdownSink() <- struct{}{}
				zap.S().Debugf("[DTR] Shutting down %d synchronizers", len(l.synchronizers))
				for _, s := range l.synchronizers {
					s.shutdownSink() <- struct{}{}
				}
				l.wg.Wait()
				close(l.doneCh)
				zap.S().Debug("[DTR] Shutdown complete")
				return

			case conn := <-l.newConnectionsCh:
				// Start a new ids synchronizer
				_, ok := l.synchronizers[conn]
				if ok {
					zap.S().Errorf("[%s][DTR] Repetitive attempt to register ids synchronizer", conn.RawConn.RemoteAddr())
					continue
				}
				s := newSynchronizer(l.wg, l.drawer, l.blocksRequestsSink(), conn)
				l.synchronizers[conn] = s
				l.wg.Add(1)
				go s.start()

			case conn := <-l.closedConnectionsCh:
				// Shutting down ids synchronizer
				zap.S().Debugf("[%s][DTR] Connection closed, shutting down ids synchronizer", conn.RawConn.RemoteAddr())
				s, ok := l.synchronizers[conn]
				if !ok {
					zap.S().Errorf("[%s][DTR] No ids synchronizer found", conn.RawConn.RemoteAddr())
					continue
				}
				delete(l.synchronizers, conn)
				s.shutdownSink() <- struct{}{}

			case conn := <-l.scoreCh:
				// New score on connection
				s, ok := l.synchronizers[conn]
				if !ok {
					zap.S().Errorf("[%s][DTR] No ids synchronizer", conn.RawConn.RemoteAddr())
					continue
				}
				go func(s *synchronizer) {
					s.score() <- struct{}{}
				}(s)

			case e := <-l.idsCh:
				// Handle new ids
				zap.S().Debugf("[%s][DTR] Received %d ids: %s", e.conn.RawConn.RemoteAddr(), len(e.ids), logIds(e.ids))
				s, ok := l.synchronizers[e.conn]
				if !ok {
					zap.S().Errorf("[%s][DTR] No ids synchronizer", e.conn.RawConn.RemoteAddr())
					continue
				}
				go func(sync *synchronizer, ids []proto.BlockID) {
					sync.ids() <- ids
				}(s, e.ids)

			case e := <-l.blocksLoader.notificationsTap():
				// Notify synchronizers about new block applied by blocks loader
				zap.S().Debugf("[DTR] Notifying %d synchronizers about block %s", len(l.synchronizers), e.String())
				for _, s := range l.synchronizers {
					go func(sync *synchronizer, id proto.BlockID) {
						sync.block() <- id
					}(s, e)
				}
			}
		}
	}()
	return l.doneCh
}

func (l *distributor) NewConnectionsSink() chan<- *Conn {
	return l.newConnectionsCh
}

func (l *distributor) ClosedConnectionsSink() chan<- *Conn {
	return l.closedConnectionsCh
}

func (l *distributor) ScoreSink() chan<- *Conn {
	return l.scoreCh
}

func (l *distributor) IdsSink() chan<- idsEvent {
	return l.idsCh
}

func (l *distributor) BlocksSink() chan<- blockEvent {
	return l.blocksLoader.blocksSink()
}

func (l *distributor) blocksRequestsSink() chan<- idsEvent {
	return l.blocksLoader.requestsSink()
}
