package internal

import (
	"bufio"
	"encoding/binary"
	"io"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"
)

const (
	DefaultRecvBufSize     = 4 << 10          // Default size of the receiving buffer, 4096 bytes.
	DefaultSendQueueLen    = 1 << 7           // Default length of the sending queue (size of the channel) - 64 packets.
	maxTemporaryErrorDelay = 1 * time.Second  // Maximum value of the temporary error delay.
	handshakeTimeout       = 30 * time.Second // Duration of timeout on handshake operations.
	maxPayloadLength       = 2 << 20          // Maximum expected payload length 2MB.
	magic                  = 0x12345678       // Protocol magic bytes
	sizeLength             = 4                // Length of the encoded size field in the network message
	magicLength            = 4                // Length of the field with magic bytes
)

const (
	StopImmediately       = iota // StopImmediately mean stop directly, the cached data maybe will not send.
	StopGracefullyAndWait        // StopGracefullyAndWait stop and block until cached data sent.
)

const (
	connStateInitial int32 = iota
	connStateNormal
	connStateStopping
	connStateStopped
)

type Handler interface {
	OnAccept(*Conn)          // OnAccept occurs then server accepts new connection.
	OnConnect(*Conn)         // OnConnect happens on new client connection.
	OnReceive(*Conn, []byte) // OnReceive executes on new bytes arrival.
	OnClose(*Conn)           // OnClose fires then connection is closed.
}

type Options struct {
	Handler         Handler
	RecvBufSize     int           // default is DefaultRecvBufSize if you don't set.
	SendQueueLen    int           // default is DefaultSendQueueLen if you don't set.
	AsyncWrite      bool          // default is DefaultAsyncWrite  if you don't set.
	NoDelay         bool          // default is true
	KeepAlive       bool          // default is false
	KeepAlivePeriod time.Duration // default is 0, mean use system setting.
	ReadDeadline    time.Duration // default is 0, means Read will not time out.
	WriteDeadline   time.Duration // default is 0, means Write will not time out.
}

func NewOptions(h Handler) *Options {
	if h == nil {
		panic("nil handler is not allowed")
	}
	return &Options{
		Handler:      h,
		RecvBufSize:  DefaultRecvBufSize,
		SendQueueLen: DefaultSendQueueLen,
		AsyncWrite:   true,
		NoDelay:      true,
		KeepAlive:    false,
	}
}

type Conn struct {
	sync.Mutex
	Opts        *Options
	RawConn     net.Conn
	UserData    interface{}
	sendBufList chan []byte
	closed      chan struct{}
	state       int32
	wg          sync.WaitGroup
	once        sync.Once
	SendDropped uint32
	sendBytes   uint64
	recvBytes   uint64
	dropped     uint32
}

// NewConn return new connection.
func NewConn(opts *Options) *Conn {
	if opts.RecvBufSize <= 0 {
		zap.S().Warnf("Invalid receiving buffer size %d, using default %d", opts.RecvBufSize, DefaultRecvBufSize)
		opts.RecvBufSize = DefaultRecvBufSize
	}
	if opts.SendQueueLen <= 0 {
		zap.S().Warnf("Invalid sending queue length %d, using default %d", opts.SendQueueLen, DefaultSendQueueLen)
		opts.SendQueueLen = DefaultSendQueueLen
	}
	c := &Conn{
		Opts:        opts,
		sendBufList: make(chan []byte, opts.SendQueueLen),
		closed:      make(chan struct{}),
		state:       connStateInitial,
	}
	return c
}

func (c *Conn) String() string {
	s := atomic.LoadInt32(&c.state)
	var sc rune
	switch s {
	case connStateInitial:
		sc = '>'
	case connStateNormal:
		sc = '-'
	case connStateStopping:
		sc = '|'
	case connStateStopped:
		sc = 'X'
	}
	la := "N/A"
	ra := "N/A"
	if c.RawConn != nil {
		la = c.RawConn.LocalAddr().String()
		ra = c.RawConn.RemoteAddr().String()
	}
	sb := strings.Builder{}
	sb.WriteString(la)
	sb.WriteRune('-')
	sb.WriteRune(sc)
	sb.WriteRune('>')
	sb.WriteString(ra)
	return sb.String()
}

// SendBytes returns the total number of bytes sent over the connection.
func (c *Conn) SendBytes() uint64 {
	return atomic.LoadUint64(&c.sendBytes)
}

// RecvBytes returns the total number of bytes received over the connection.
func (c *Conn) RecvBytes() uint64 {
	return atomic.LoadUint64(&c.recvBytes)
}

// DroppedPacket return the total dropped packet.
func (c *Conn) DroppedPacket() uint32 {
	return atomic.LoadUint32(&c.dropped)
}

// Stop stops the conn.
func (c *Conn) Stop(mode int) {
	c.once.Do(func() {
		if mode == StopImmediately {
			atomic.StoreInt32(&c.state, connStateStopped)
			close(c.closed)
			if c.RawConn != nil {
				err := c.RawConn.Close()
				if err != nil {
					zap.S().Warnf("[%s] Failed to close the connection properly: %v", c.RawConn.RemoteAddr(), err)
				}
			}
		} else {
			atomic.StoreInt32(&c.state, connStateStopping)
			close(c.closed)
			if mode == StopGracefullyAndWait {
				c.wg.Wait()
			}
		}
	})
}

// IsStopped return true if Conn is stopped or stopping, otherwise return false.
func (c *Conn) IsStopped() bool {
	v := atomic.LoadInt32(&c.state)
	return v == connStateStopping || v == connStateStopped
}

func (c *Conn) serve() {
	if c.IsStopped() {
		return
	}
	s := atomic.LoadInt32(&c.state)
	if s == connStateInitial {
		atomic.StoreInt32(&c.state, connStateNormal)
	}
	tcpConn := c.RawConn.(*net.TCPConn)
	err := tcpConn.SetNoDelay(c.Opts.NoDelay)
	if err != nil {
		zap.S().Warnf("[%s] Failed to configure TCP connection: %v", c.RawConn.RemoteAddr(), err)
	}
	err = tcpConn.SetKeepAlive(c.Opts.KeepAlive)
	if err != nil {
		zap.S().Warnf("[%s] Failed to configure TCP connection: %v", c.RawConn.RemoteAddr(), err)
	}
	if c.Opts.KeepAlivePeriod != 0 {
		err = tcpConn.SetKeepAlivePeriod(c.Opts.KeepAlivePeriod)
		if err != nil {
			zap.S().Warnf("[%s] Failed to configure TCP connection: %v", c.RawConn.RemoteAddr(), err)
		}
	}
	if c.Opts.AsyncWrite {
		c.wg.Add(2)
		go c.sendLoop()
	} else {
		c.wg.Add(1)
	}
	c.recvLoop()
	c.Opts.Handler.OnClose(c)
}

func (c *Conn) sleepForDelay(d time.Duration, err error) time.Duration {
	if d == 0 {
		d = 5 * time.Millisecond
	} else {
		d *= 2
	}
	if d > maxTemporaryErrorDelay {
		d = maxTemporaryErrorDelay
	}
	zap.S().Warnf("[%s] Temporary error (retrying in %s): %v", c.RawConn.RemoteAddr(), d, err)
	time.Sleep(d)
	return d
}

func (c *Conn) recvLoop() {
	reader := newSafeReader(c)
	defer func() {
		c.wg.Done()
	}()
	sizeBuf := make([]byte, sizeLength)
	magicBuf := make([]byte, magicLength)
	for !reader.abort {
		reader.reset()

		n, i := reader.readUint32(sizeBuf)
		if n == 0 {
			continue
		}
		pll := int(i)
		atomic.AddUint64(&c.recvBytes, uint64(n))
		if pll > maxPayloadLength {
			zap.S().Warnf("[%s] Payload is bigger than allowed, discarding", c.RawConn.RemoteAddr())
			reader.discard(pll)
			continue
		}

		n, m := reader.readUint32(magicBuf)
		if n == 0 {
			continue
		}
		atomic.AddUint64(&c.recvBytes, uint64(n))
		if m != magic {
			zap.S().Warnf("[%s] Invalid magic bytes, discarding", c.RawConn.RemoteAddr())
			reader.discard(pll)
			continue
		}

		buf := getBufferFromPool(pll - magicLength)

		payload := buf[:pll-magicLength]
		n = reader.read(buf)
		if n == 0 {
			continue
		}
		atomic.AddUint64(&c.recvBytes, uint64(n))

		if !reader.skip && !reader.abort {
			result := make([]byte, sizeLength+magicLength+pll)
			copy(result, sizeBuf)
			copy(result[sizeLength:], magicBuf)
			copy(result[sizeLength+magicLength:], payload)
			c.Opts.Handler.OnReceive(c, result)
		}
		putBufferToPool(buf)
	}
}

func (c *Conn) sendBuf(buf []byte) (int, error) {
	sent := 0
	var delay time.Duration
	for sent < len(buf) {
		if c.Opts.WriteDeadline != 0 {
			err := c.RawConn.SetWriteDeadline(time.Now().Add(c.Opts.WriteDeadline))
			if err != nil {
				zap.S().Warnf("[%s] Failed to set write deadline: %v", c.RawConn.RemoteAddr(), err)
			}
		}
		wn, err := c.RawConn.Write(buf[sent:])
		if wn > 0 {
			sent += wn
			atomic.AddUint64(&c.sendBytes, uint64(wn))
		}
		if err != nil {
			if netErr, ok := err.(net.Error); ok {
				if netErr.Timeout() {
					zap.S().Debugf("[%s] Send time out", c.RawConn.RemoteAddr())
				} else {
					delay = c.sleepForDelay(delay, err)
					continue
				}
			}
			if !c.IsStopped() {
				zap.S().Errorf("[%s] Send error: %v", c.RawConn.RemoteAddr(), err)
				c.Stop(StopImmediately)
			}
			return sent, err
		}
		delay = 0
	}
	return sent, nil
}

func (c *Conn) sendLoop() {
	defer func() {
		c.wg.Done()
	}()
	for {
		if atomic.LoadInt32(&c.state) == connStateStopped {
			return
		}
		select {
		case buf, ok := <-c.sendBufList:
			if !ok {
				return
			}
			_, err := c.sendBuf(buf)
			if err != nil {
				return
			}
			putBufferToPool(buf)
		case <-c.closed:
			if atomic.LoadInt32(&c.state) == connStateStopping {
				if len(c.sendBufList) == 0 {
					atomic.SwapInt32(&c.state, connStateStopped)
					err := c.RawConn.Close()
					if err != nil {
						zap.S().Warnf("[%s] Failed to close network connection: %v", c.RawConn.RemoteAddr(), err)
					}
					return
				}
			}
		}
	}
}

func (c *Conn) Send(buf []byte) (int, error) {
	s := atomic.LoadInt32(&c.state)
	if s == connStateInitial {
		return 0, nil
	}
	if s == connStateStopping || s == connStateStopped {
		return 0, errors.New("unable to send to closed connection")
	}
	bufLen := len(buf)
	if bufLen <= 0 {
		return 0, nil
	}
	if c.Opts.AsyncWrite {
		buffer := getBufferFromPool(len(buf))
		copy(buffer, buf)
		select {
		case c.sendBufList <- buffer:
			return bufLen, nil
		default:
			atomic.AddUint32(&c.dropped, 1)
			return 0, errors.New("send queue is full")
		}
	} else {
		c.Lock()
		n, err := c.sendBuf(buf)
		c.Unlock()
		return n, err
	}
}

func (c *Conn) DialAndServe(addr string) error {
	//TODO: handle close of a connection that is dialing
	rawConn, err := net.DialTimeout("tcp", addr, c.Opts.WriteDeadline)
	if err != nil {
		return err
	}
	select {
	case <-c.closed:
		return errors.New("Connection closed while instantiation")
	default:
	}
	c.RawConn = rawConn
	c.Opts.Handler.OnConnect(c)
	c.serve()
	return nil
}

type Server struct {
	Opts        *Options
	stopped     chan struct{}
	wg          sync.WaitGroup
	mu          sync.Mutex
	stopping    bool
	once        sync.Once
	ln          net.Listener
	connections map[*Conn]struct{}
}

func NewServer(opts *Options) *Server {
	if opts.RecvBufSize <= 0 {
		zap.S().Warnf("Invalid receive buffer size %d, using default value instead", opts.RecvBufSize)
		opts.RecvBufSize = DefaultRecvBufSize
	}
	if opts.SendQueueLen <= 0 {
		zap.S().Warnf("Invalid send queue length %d, using default value instead", opts.SendQueueLen)
		opts.SendQueueLen = DefaultSendQueueLen
	}
	s := &Server{
		Opts:        opts,
		stopped:     make(chan struct{}),
		connections: make(map[*Conn]struct{}),
	}
	return s
}

func (s *Server) ListenAndServe(addr string) error {
	if addr == "" {
		return errors.New("empty address to bind network server")
	}
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	s.Serve(l)
	return nil
}

func (s *Server) Serve(l net.Listener) {
	defer s.wg.Done()
	s.wg.Add(1)

	s.mu.Lock()
	s.ln = l
	s.mu.Unlock()
	zap.S().Infof("Server listen on: %s", l.Addr().String())

	var delay time.Duration
	for {
		conn, err := l.Accept()
		if err != nil {
			if _, ok := err.(net.Error); ok {
				if delay == 0 {
					delay = 5 * time.Millisecond
				} else {
					delay *= 2
				}
				if delay > maxTemporaryErrorDelay {
					delay = maxTemporaryErrorDelay
				}
				zap.S().Warnf("Failed to accept new connection on %s: %v", s.ln.Addr().String(), err)
				timer := time.NewTimer(delay)
				select {
				case <-timer.C:
					continue
				case <-s.stopped:
					if !timer.Stop() {
						<-timer.C
					}
					return
				}
			}
			if !s.IsStopped() && !s.stopping {
				zap.S().Errorf("Server %s stopped due to the error: %v", s.ln.Addr().String(), err)
				s.Stop(StopImmediately)
			}
			return
		}
		delay = 0
		go s.handleRawConn(conn)
	}
}

func (s *Server) Stopped() <-chan struct{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.stopped
}

func (s *Server) IsStopped() bool {
	select {
	case <-s.stopped:
		return true
	default:
		return false
	}
}

func (s *Server) Stop(mode int) {
	s.once.Do(func() {
		s.mu.Lock()
		ln := s.ln
		s.ln = nil
		connections := s.connections
		s.connections = nil
		s.stopping = true
		s.mu.Unlock()

		if ln != nil {
			err := ln.Close()
			if err != nil {
				zap.S().Warnf("Failed to close listener: %v", err)
			}
		}
		zap.S().Debugf("Closing %d incoming connections", len(connections))
		for c := range connections {
			c.Stop(mode)
		}

		if mode == StopGracefullyAndWait {
			s.wg.Wait()
		}
		close(s.stopped)
	})
}

func (s *Server) handleRawConn(conn net.Conn) {
	s.mu.Lock()
	if s.connections == nil { // server stopped
		s.mu.Unlock()
		err := conn.Close()
		if err != nil {
			zap.S().Warnf("Failed to close connection: %v", err)
		}
		return
	}
	s.mu.Unlock()

	tcpConn := NewConn(s.Opts)
	tcpConn.RawConn = conn

	if !s.addConn(tcpConn) {
		tcpConn.Stop(StopImmediately)
		return
	}

	s.wg.Add(1)
	defer func() {
		s.removeConn(tcpConn)
		s.wg.Done()
	}()

	s.Opts.Handler.OnAccept(tcpConn)
	tcpConn.serve()
}

func (s *Server) addConn(conn *Conn) bool {
	s.mu.Lock()
	if s.connections == nil {
		s.mu.Unlock()
		return false
	}
	s.connections[conn] = struct{}{}
	s.mu.Unlock()
	return true
}

func (s *Server) removeConn(conn *Conn) {
	s.mu.Lock()
	if s.connections != nil {
		delete(s.connections, conn)
	}
	s.mu.Unlock()
}

var (
	bufferPool1K = &sync.Pool{
		New: func() interface{} {
			buf := make([]byte, 1<<10)
			return &buf
		},
	}
	bufferPool2K = &sync.Pool{
		New: func() interface{} {
			buf := make([]byte, 2<<10)
			return &buf
		},
	}
	bufferPool4K = &sync.Pool{
		New: func() interface{} {
			buf := make([]byte, 4<<10)
			return &buf
		},
	}
	bufferPoolBig = &sync.Pool{}
)

type safeReader struct {
	conn   *Conn
	reader *bufio.Reader
	abort  bool
	skip   bool
	delay  time.Duration
}

func newSafeReader(conn *Conn) *safeReader {
	return &safeReader{conn: conn, reader: bufio.NewReader(conn.RawConn)}
}

func (r *safeReader) reset() {
	if !r.abort {
		r.skip = false
	}
}

func (r *safeReader) read(buf []byte) uint64 {
	if r.skip || r.abort {
		return 0
	}
	if r.conn.Opts.ReadDeadline != 0 {
		err := r.conn.RawConn.SetReadDeadline(time.Now().Add(r.conn.Opts.ReadDeadline))
		if err != nil {
			zap.S().Warnf("[%s] Failed to set read deadline: %v", r.conn.RawConn.RemoteAddr(), err)
		}
	}
	n, err := io.ReadFull(r.reader, buf)
	if err != nil {
		if netErr, ok := err.(net.Error); ok {
			if netErr.Timeout() {
				zap.S().Debugf("[%s] Receive time out", r.conn.RawConn.RemoteAddr())
			} else {
				if r.delay == 0 {
					r.delay = 5 * time.Millisecond
				} else {
					r.delay *= 2
				}
				if r.delay > maxTemporaryErrorDelay {
					r.delay = maxTemporaryErrorDelay
				}
				zap.S().Warnf("[%s] Network error (retrying in %s): %v", r.conn.RawConn.RemoteAddr(), r.delay, netErr)
				time.Sleep(r.delay)
				r.skip = true
				return 0
			}
		}
		if !r.conn.IsStopped() {
			if err != io.EOF {
				zap.S().Errorf("[%s] Receive error: %v", r.conn.RawConn.RemoteAddr(), err)
			}
			r.conn.Stop(StopImmediately)
		}
		r.abort = true
		return 0
	}
	r.delay = 0
	return uint64(n)
}

func (r *safeReader) readUint32(buf []byte) (uint64, uint32) {
	if r.skip || r.abort {
		return 0, 0
	}
	n := r.read(buf)
	i := binary.BigEndian.Uint32(buf)
	return n, i
}

func (r *safeReader) discard(n int) {
	if r.abort {
		return
	}
	r.skip = true
	d, err := r.reader.Discard(n)
	if err != nil {
		zap.S().Errorf("[%s] Failed to discard connection buffer: %v", r.conn.RawConn.RemoteAddr(), err)
	}
	zap.S().Debugf("[%s] %d bytes have been discarded", r.conn.RawConn.RemoteAddr(), d)
}

func getBufferFromPool(targetSize int) []byte {
	var buf []byte
	switch {
	case targetSize <= 1<<10:
		buf = *bufferPool1K.Get().(*[]byte)
	case targetSize <= 2<<10:
		buf = *bufferPool2K.Get().(*[]byte)
	case targetSize <= 4<<10:
		buf = *bufferPool4K.Get().(*[]byte)
	default:
		itr := bufferPoolBig.Get()
		if itr != nil && cap(*itr.(*[]byte)) >= targetSize {
			buf = *itr.(*[]byte)
		} else {
			buf = make([]byte, targetSize)
		}
	}
	buf = buf[:targetSize]
	return buf
}

func putBufferToPool(buf []byte) {
	c := cap(buf)
	switch {
	case c <= 1<<10:
		// To understand why we store pointer to slice here, refer: https://staticcheck.io/docs/checks#SA6002.
		bufferPool1K.Put(&buf)
	case c <= 2<<10:
		// To understand why we store pointer to slice here, refer: https://staticcheck.io/docs/checks#SA6002.
		bufferPool2K.Put(&buf)
	case c <= 4<<10:
		// To understand why we store pointer to slice here, refer: https://staticcheck.io/docs/checks#SA6002.
		bufferPool4K.Put(&buf)
	default:
		// To understand why we store pointer to slice here, refer: https://staticcheck.io/docs/checks#SA6002.
		bufferPoolBig.Put(&buf)
	}
}
