package internal

import (
	"bytes"
	"net"
	"sync"
	"time"

	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
)

const (
	reconnectionInterval = 3 * time.Second
	askPeersInterval     = 5 * time.Second
	askPeersDelay        = 30 * time.Second
)

type dispatcher struct {
	interrupt <-chan struct{}
	bind      string
	Opts      *Options
	server    *Server
	registry  *Registry
	schedule  schedule
}

func NewDispatcher(interrupt <-chan struct{}, bind string, opts *Options, registry *Registry) *dispatcher {
	if opts.RecvBufSize <= 0 {
		zap.S().Warnf("Invalid receive buffer size %d, using default value instead", opts.RecvBufSize)
		opts.RecvBufSize = DefaultRecvBufSize
	}
	if opts.SendQueueLen <= 0 {
		zap.S().Warnf("Invalid send queue length %d, using default value instead", opts.SendQueueLen)
		opts.SendQueueLen = DefaultSendQueueLen
	}
	s := NewServer(opts)
	d := &dispatcher{
		interrupt: interrupt,
		bind:      bind,
		Opts:      opts,
		server:    s,
		registry:  registry,
		schedule:  schedule{interval: askPeersDelay},
	}
	return d
}

func (d *dispatcher) Start() <-chan struct{} {
	zap.S().Debug("Starting dispatcher...")
	go func() {
		err := d.server.ListenAndServe(d.bind)
		if err != nil {
			zap.S().Errorf("Failed to start network server: %v", err)
			return
		}
	}()
	stopped := make(chan struct{})
	go func() {
		var (
			reconnectTicker = time.NewTicker(reconnectionInterval)
			askPeersTicker  = time.NewTicker(askPeersInterval)
		)
		defer func() {
			reconnectTicker.Stop()
			askPeersTicker.Stop()
			close(stopped)
		}()
		for {
			select {
			case <-d.interrupt:
				zap.S().Debug("Shutting down dispatcher...")
				zap.S().Debug("Shutting down server...")
				d.server.Stop(StopGracefullyAndWait)
				<-d.server.Stopped()
				zap.S().Debugf("Closing %d outgoing connections", d.schedule.len())
				for _, c := range d.schedule.connections() {
					c.Stop(StopImmediately)
				}
				zap.S().Debug("Server shutdown complete")
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
	return stopped
}

func (d *dispatcher) dial(addr net.Addr) {
	conn := NewConn(d.Opts)
	defer func(conn *Conn) {
		d.schedule.remove(conn)
	}(conn)
	d.schedule.append(conn)
	err := conn.DialAndServe(addr.String())
	if err != nil {
		zap.S().Errorf("Failed to establish connection with %s: %v", addr.String(), err)
		err := d.registry.PeerDiscarded(addr)
		if err != nil {
			zap.S().Fatalf("Failed to update peer state")
		}
		return
	}
}

func (d *dispatcher) askPeers(conn *Conn) {
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

type schedule struct {
	interval time.Duration
	once     sync.Once
	mu       sync.RWMutex
	items    map[*Conn]time.Time
}

func (s *schedule) init() {
	s.items = make(map[*Conn]time.Time)
}

func (s *schedule) append(c *Conn) {
	s.once.Do(s.init)
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.items[c]
	if !ok {
		s.items[c] = time.Now().Add(s.interval)
	}
}

func (s *schedule) remove(c *Conn) {
	s.once.Do(s.init)
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.items, c)
}

func (s *schedule) pull() []*Conn {
	s.once.Do(s.init)
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

func (s *schedule) connections() []*Conn {
	s.once.Do(s.init)
	s.mu.RLock()
	defer s.mu.RUnlock()
	r := make([]*Conn, len(s.items))
	i := 0
	for c := range s.items {
		r[i] = c
		i++
	}
	return r
}

func (s *schedule) len() int {
	s.once.Do(s.init)
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.items)
}
