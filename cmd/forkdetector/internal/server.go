package internal

import (
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"net"
	"sync"
	"time"
)

const (
	acceptMinSleep = 10 * time.Millisecond
	acceptMaxSleep = 1 * time.Second
)

type server struct {
	interrupt   <-chan struct{}
	listener    net.Listener
	mu          sync.Mutex
	connections chan net.Conn
}

func NewServer(interrupt <-chan struct{}, bind string) (*server, error) {
	if bind == "" {
		return nil, errors.New("empty address to bind network server")
	}
	l, err := net.Listen("tcp", bind)
	if err != nil {
		return nil, errors.Wrap(err, "failed to bind network server")
	}
	return &server{
		interrupt: interrupt,
		listener:  l,
	}, nil
}

func (s *server) Start() <-chan struct{} {
	zap.S().Debug("Starting network server...")
	done := make(chan struct{})
	tmpDelay := acceptMinSleep
	s.mu.Lock()
	connections := s.getConnectionsChanLocked()
	s.mu.Unlock()
	go func() {
		zap.S().Debug("Waiting for incoming connection")
		for {
			select {
			case <-s.interrupt:
				zap.S().Debug("Shutting down network server...")
				err := s.listener.Close()
				if err != nil {
					zap.S().Warnf("Failed to close network listener: %v", err)
				}
				close(done)
				return
			default:
				err := s.resetListenerDeadline()
				if err != nil {
					zap.S().Warn("Failed to set timeout on network listener: %v", err)
					continue
				}
				conn, err := s.listener.Accept()
				if err != nil {
					if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
						continue
					}
					tmpDelay = s.handleAcceptError(err, tmpDelay)
					continue
				}
				tmpDelay = acceptMinSleep
				connections <- conn
			}
		}
	}()
	return done
}

func (s *server) resetListenerDeadline() error {
	l, ok := s.listener.(*net.TCPListener)
	if ok {
		err := l.SetDeadline(time.Now().Add(3 * time.Second))
		if err != nil {
			return err
		}
		return nil
	}
	return errors.New("not a TCPListener")
}

func (s *server) handleAcceptError(err error, tmpDelay time.Duration) time.Duration {
	if ne, ok := err.(net.Error); ok && ne.Temporary() {
		zap.S().Errorf("Temporary client accept error: %v, sleeping for %dms", ne, tmpDelay/time.Millisecond)
		select {
		case <-time.After(tmpDelay):
		case <-s.interrupt:
			return tmpDelay
		}
		tmpDelay *= 2
		if tmpDelay > acceptMaxSleep {
			tmpDelay = acceptMaxSleep
		}
	} else {
		zap.S().Errorf("Client accept error: %v", err)
	}
	return tmpDelay
}

func (s *server) GetConnections() <-chan net.Conn {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.getConnectionsChanLocked()
}

func (s *server) getConnectionsChanLocked() chan net.Conn {
	if s.connections == nil {
		s.connections = make(chan net.Conn)
	}
	return s.connections
}
