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
	reconnectionInterval = time.Second
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
	}
	return d
}

func (d *Dispatcher) Start() <-chan struct{} {
	zap.S().Debug("Starting dispatcher...")
	reconnectTicker := time.NewTicker(reconnectionInterval)
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
				zap.S().Debugf("Closing %d outgoing connections", len(d.connections))
				for c := range d.connections {
					c.Stop(StopGracefullyAndWait)
				}
				zap.S().Debug("Shutting down server...")
				d.server.Stop(StopGracefullyAndWait)
				<-d.server.Stopped()
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
	d.mu.Unlock()
}

func (d *Dispatcher) removeConnection(conn *Conn) {
	d.mu.Lock()
	delete(d.connections, conn)
	d.mu.Unlock()
}
