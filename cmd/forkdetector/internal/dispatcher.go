package internal

import (
	"bytes"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
	"net"
	"sync"
	"time"
)

const (
	reconnectionInterval = 3 * time.Second
	askPeersInterval     = 5 * time.Second
	askPeersDelay        = 30 * time.Second
)

type Dispatcher struct {
	interrupt   <-chan struct{}
	bind        string
	Opts        *Options
	server      *Server
	stopped     chan struct{}
	registry    *Registry
	mu          sync.Mutex
	connections map[*Conn]struct{}
	schedule    schedule
}

func NewDispatcher(interrupt <-chan struct{}, bind string, opts *Options, registry *Registry) *Dispatcher {
	if opts.RecvBufSize <= 0 {
		zap.S().Warnf("Invalid receive buffer size %d, using default value instead", opts.RecvBufSize)
		opts.RecvBufSize = DefaultRecvBufSize
	}
	if opts.SendQueueLen <= 0 {
		zap.S().Warnf("Invalid send queue length %d, using default value instead", opts.SendQueueLen)
		opts.SendQueueLen = DefaultSendQueueLen
	}
	s := NewServer(opts)
	d := &Dispatcher{
		interrupt:   interrupt,
		bind:        bind,
		Opts:        opts,
		server:      s,
		stopped:     make(chan struct{}),
		registry:    registry,
		connections: make(map[*Conn]struct{}),
		mu:          sync.Mutex{},
		schedule:    schedule{interval: askPeersDelay},
	}
	return d
}

func (d *Dispatcher) Start() <-chan struct{} {
	zap.S().Debug("Starting dispatcher...")
	reconnectTicker := time.NewTicker(reconnectionInterval)
	askPeersTicker := time.NewTicker(askPeersInterval)
	go func() {
		err := d.server.ListenAndServe(d.bind)
		if err != nil {
			zap.S().Errorf("Failed to start network server: %v", err)
			return
		}
	}()
	go func() {
		for {
			select {
			case <-d.interrupt:
				zap.S().Debug("Shutting down dispatcher...")
				zap.S().Debug("Shutting down server...")
				d.server.Stop(StopGracefullyAndWait)
				<-d.server.Stopped()
				zap.S().Debugf("Closing %d outgoing connections", d.connectionsCount())
				for c := range d.connections {
					c.Stop(StopGracefullyAndWait)
				}
				zap.S().Debug("Server shutdown complete")
				close(d.stopped)
				return
			case <-reconnectTicker.C:
				addresses, err := d.registry.TakeAvailableAddresses()
				if err != nil {
					zap.S().Warnf("Failed to get available addresses to connect: %v", err)
					continue
				}
				for _, a := range addresses {
					go d.dial(a)
				}
			case <-askPeersTicker.C:
				for _, c := range d.schedule.pull() {
					d.askPeers(c)
				}
			}
		}
	}()
	return d.stopped
}

func (d *Dispatcher) dial(addr net.Addr) {
	c := NewConn(d.Opts)
	defer func(conn *Conn) {
		d.removeConnection(conn)
	}(c)
	d.addConnection(c)
	err := c.DialAndServe(addr.String())
	if err != nil {
		zap.S().Errorf("Failed to establish connection with %s: %v", addr.String(), err)
		err := d.registry.PeerDiscarded(addr)
		if err != nil {

		}
		return
	}
}

func (d *Dispatcher) askPeers(conn *Conn) {
	buf := new(bytes.Buffer)
	m := proto.GetPeersMessage{}
	_, err := m.WriteTo(buf)
	if err != nil {
		zap.S().Warnf("Failed to ask for new peers: %v", err)
		return
	}
	_, err = conn.Send(buf.Bytes())
	if err != nil {
		zap.S().Warnf("Failed to ask for new peers: %v", err)
		return
	}
}

func (d *Dispatcher) addConnection(conn *Conn) {
	d.mu.Lock()
	d.connections[conn] = struct{}{}
	d.schedule.append(conn)
	d.mu.Unlock()
}

func (d *Dispatcher) removeConnection(conn *Conn) {
	d.mu.Lock()
	delete(d.connections, conn)
	d.schedule.remove(conn)
	d.mu.Unlock()
}

func (d *Dispatcher) connectionsCount() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return len(d.connections)
}

type schedule struct {
	interval time.Duration
	once     sync.Once
	mu       sync.Mutex
	items    map[*Conn]time.Time
}

func (s *schedule) append(c *Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.once.Do(func() {
		s.items = make(map[*Conn]time.Time)
	})

	_, ok := s.items[c]
	if !ok {
		s.items[c] = time.Now().Add(s.interval)
	}
}

func (s *schedule) remove(c *Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.items, c)
}

func (s *schedule) pull() []*Conn {
	s.mu.Lock()
	defer s.mu.Unlock()

	r := make([]*Conn, 0)
	t := time.Now()
	for k, v := range s.items {
		if t.After(v) {
			r = append(r, k)
			s.items[k] = t.Add(s.interval)
		}
	}
	return r
}
