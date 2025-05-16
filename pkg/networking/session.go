package networking

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wavesplatform/gowaves/pkg/execution"
)

// Session is used to wrap a reliable ordered connection.
type Session struct {
	g      *execution.TaskGroup
	ctx    context.Context
	cancel context.CancelFunc

	config *Config
	logger *slog.Logger
	tp     *timerPool

	connWriteLock sync.Mutex         // connLock is used to lock the connection for Write and Close.
	conn          io.ReadWriteCloser // conn is the underlying connection
	bufRead       *bufio.Reader      // buffered reader wrapped around the connection

	receiveLock   sync.Mutex    // Guards the receiveBuffer.
	receiveBuffer *bytes.Buffer // receiveBuffer is used to store the incoming data.

	sendLock sync.Mutex       // Guards the sendCh.
	sendCh   chan *sendPacket // sendCh is used to send data to the connection.

	receiving   atomic.Bool // Indicates that receiveLoop already running.
	established atomic.Bool // Indicates that incoming Handshake was successfully accepted.
	closing     atomic.Bool // Indicates that the session is closing.
}

// NewSession is used to construct a new session.
func newSession(ctx context.Context, config *Config, conn io.ReadWriteCloser, tp *timerPool) (*Session, error) {
	if config.protocol == nil {
		return nil, ErrInvalidConfigurationNoProtocol
	}
	if config.handler == nil {
		return nil, ErrInvalidConfigurationNoHandler
	}
	if config.keepAlive && config.keepAliveInterval <= 0 {
		return nil, ErrInvalidConfigurationNoKeepAliveInterval
	}
	if config.connectionWriteTimeout <= 0 {
		return nil, ErrInvalidConfigurationNoWriteTimeout
	}
	if tp == nil {
		return nil, ErrEmptyTimerPool
	}

	sCtx, cancel := context.WithCancel(ctx)
	s := &Session{
		g:       execution.NewTaskGroup(suppressContextCancellationError),
		ctx:     sCtx,
		cancel:  cancel,
		config:  config,
		tp:      tp,
		conn:    conn,
		bufRead: bufio.NewReader(conn),
		sendCh:  make(chan *sendPacket, 1), // TODO: Make the size of send channel configurable.
	}

	slogHandler := config.slogHandler
	if slogHandler == nil {
		slogHandler = discardingHandler{}
	}

	sa := [...]any{
		slog.String("namespace", Namespace),
		slog.String("remote", s.RemoteAddr().String()),
	}
	attrs := append(sa[:], config.attributes...)
	s.logger = slog.New(slogHandler).With(attrs...)

	s.g.Run(s.receiveLoop)
	s.g.Run(s.sendLoop)
	if s.config.keepAlive {
		s.g.Run(s.keepaliveLoop)
	}

	return s, nil
}

func (s *Session) String() string {
	return fmt.Sprintf("Session{local=%s,remote=%s}", s.LocalAddr(), s.RemoteAddr())
}

// LocalAddr returns the local network address.
func (s *Session) LocalAddr() net.Addr {
	if a, ok := s.conn.(addressable); ok {
		return a.LocalAddr()
	}
	return &sessionAddress{addr: "local"}
}

// RemoteAddr returns the remote network address.
func (s *Session) RemoteAddr() net.Addr {
	if a, ok := s.conn.(addressable); ok {
		return a.RemoteAddr()
	}
	return &sessionAddress{addr: "remote"}
}

// Close is used to close the session. It is safe to call Close multiple times from different goroutines,
// subsequent calls do nothing.
func (s *Session) Close() error {
	var err error
	if s.closing.CompareAndSwap(false, true) {
		s.logger.Debug("Closing session")
		s.connWriteLock.Lock()
		clErr := s.conn.Close() // Close the underlying connection.
		s.connWriteLock.Unlock()
		if clErr != nil {
			s.logger.Warn("Failed to close underlying connection", "error", clErr)
		}
		s.logger.Debug("Underlying connection closed")

		s.cancel() // Cancel the underlying context to interrupt the loops.

		s.logger.Debug("Waiting for loops to finish")
		err = s.g.Wait() // Wait for loops to finish.

		err = errors.Join(err, clErr) // Combine loops finalization errors with connection close error.

		s.logger.Debug("Session closed", "error", err)
	}
	return err
}

// Write is used to write to the session. It is safe to call Write and/or Close concurrently.
func (s *Session) Write(msg []byte) (int, error) {
	s.sendLock.Lock()
	defer s.sendLock.Unlock()

	if err := s.waitForSend(msg); err != nil {
		return 0, err
	}

	return len(msg), nil
}

// waitForSend waits to send a data, checking for a potential context cancellation.
func (s *Session) waitForSend(data []byte) error {
	// Channel to receive an error from sendLoop goroutine.
	// This channel is created per send. It will be closed later by sendLoop in case of successful send.
	// If we fail to send, the errCh is not closed and will be GCed when the Session is closed.
	errCh := make(chan error, 1)

	timer := s.tp.Get()
	timer.Reset(s.config.connectionWriteTimeout)
	defer s.tp.Put(timer)

	if s.logger.Enabled(s.ctx, slog.LevelDebug) {
		s.logger.Debug("Sending data", "data", base64.StdEncoding.EncodeToString(data))
	}
	select {
	case s.sendCh <- newSendPacket(data, errCh):
		s.logger.Debug("Data written into send channel")
	case <-s.ctx.Done():
		s.logger.Debug("Session shutdown while sending data")
		return ErrSessionShutdown
	case <-timer.C:
		s.logger.Debug("Connection write timeout while sending data")
		return ErrConnectionWriteTimeout
	}

	select {
	case err, ok := <-errCh:
		if !ok { // Channel was closed by sendLoop (successful send).
			s.logger.Debug("Data sent successfully")
			return nil // No error, data was sent successfully.
		}
		s.logger.Debug("Error sending data", "error", err)
		return err
	case <-s.ctx.Done():
		s.logger.Debug("Session shutdown while waiting send error")
		return ErrSessionShutdown
	case <-timer.C:
		s.logger.Debug("Connection write timeout while waiting send error")
		return ErrConnectionWriteTimeout
	}
}

// sendLoop is a long-running goroutine that sends data to the connection.
func (s *Session) sendLoop() error {
	var dataBuf bytes.Buffer
	for {
		dataBuf.Reset()

		select {
		case <-s.ctx.Done():
			s.logger.Debug("Exiting connection send loop")
			return s.ctx.Err()

		case packet := <-s.sendCh:
			packet.mu.Lock()
			_, rErr := dataBuf.ReadFrom(packet.r)
			if rErr != nil {
				packet.mu.Unlock()
				s.logger.Error("Failed to copy data into buffer", "error", rErr)
				s.asyncSendErr(packet.err, rErr)
				return rErr
			}
			packet.mu.Unlock()

			if dataBuf.Len() > 0 {
				data := dataBuf.Bytes()
				if s.logger.Enabled(s.ctx, slog.LevelDebug) {
					s.logger.Debug("Sending data to connection",
						"len", len(data),
						"data", base64.StdEncoding.EncodeToString(data))
				}
				written, err := s.writeConnIfNotClosed(data)
				if err != nil {
					s.logger.Error("Failed to write data into connection", "error", err)
					s.asyncSendErr(packet.err, err)
					return err
				}
				if written {
					s.logger.Debug("Data written into connection")
				}
			}

			// No error, close the channel.
			close(packet.err)
		}
	}
}

func (s *Session) writeConnIfNotClosed(b []byte) (bool, error) {
	s.connWriteLock.Lock()
	defer s.connWriteLock.Unlock()

	// Check if the session is closing or the context is done before writing to the connection
	// (in case of parent context cancellation).
	if s.closing.Load() || s.ctx.Err() != nil {
		return false, nil
	}

	_, err := s.conn.Write(b) // TODO: We are locking here, because no timeout set on connection itself.
	return true, err
}

// receiveLoop continues to receive data until a fatal error is encountered or underlying connection is closed.
// Receive loop works after handshake and accepts only length-prepended messages.
func (s *Session) receiveLoop() error {
	if !s.receiving.CompareAndSwap(false, true) {
		return nil // Prevent running multiple receive loops.
	}
	for {
		if err := s.receive(); err != nil {
			if errors.Is(err, ErrConnectionClosedOnRead) {
				s.config.handler.OnClose(s)
				return nil // Exit normally on connection close.
			}
			s.config.handler.OnFailure(s, err)
			return err
		}
	}
}

func (s *Session) receive() error {
	if s.established.Load() {
		hdr := s.config.protocol.EmptyHeader()
		return s.readMessage(hdr)
	}
	return s.readHandshake()
}

func (s *Session) readHandshake() error {
	s.logger.Debug("Reading handshake")

	hs := s.config.protocol.EmptyHandshake()
	_, err := hs.ReadFrom(s.bufRead)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return ErrConnectionClosedOnRead
		}
		if errMsg := err.Error(); strings.Contains(errMsg, "closed") ||
			strings.Contains(errMsg, "reset by peer") {
			return errors.Join(ErrConnectionClosedOnRead, err) // Wrap the error with ErrConnectionClosedOnRead.
		}
		s.logger.Error("Failed to read handshake from connection", "error", err)
		return err
	}
	s.logger.Debug("Handshake successfully read")

	if !s.config.protocol.IsAcceptableHandshake(s, hs) {
		s.logger.Error("Handshake is not acceptable")
		s.config.handler.OnHandshakeFailed(s, hs)
		return ErrUnacceptableHandshake
	}
	// Handshake is acceptable, we can switch the session into established state.
	s.established.Store(true)
	s.config.handler.OnHandshake(s, hs)
	return nil
}

func (s *Session) readMessage(hdr Header) error {
	// Read the header
	if _, err := hdr.ReadFrom(s.bufRead); err != nil {
		if errors.Is(err, io.EOF) {
			return ErrConnectionClosedOnRead
		}
		if errMsg := err.Error(); strings.Contains(errMsg, "closed") ||
			strings.Contains(errMsg, "reset by peer") ||
			strings.Contains(errMsg, "broken pipe") { // In Docker network built on top of pipe, we get this error on close.
			return errors.Join(ErrConnectionClosedOnRead, err) // Wrap the error with ErrConnectionClosedOnRead.
		}
		s.logger.Error("Failed to read header", "error", err)
		return err
	}
	if !s.config.protocol.IsAcceptableMessage(s, hdr) {
		// We have to discard the remaining part of the message.
		if _, err := io.CopyN(io.Discard, s.bufRead, int64(hdr.PayloadLength())); err != nil {
			s.logger.Error("Failed to discard message", "error", err)
			return err
		}
	}
	// Read the new data
	if err := s.readMessagePayload(hdr, s.bufRead); err != nil {
		s.logger.Error("Failed to read message", "error", err)
		return err
	}
	return nil
}

func (s *Session) readMessagePayload(hdr Header, conn io.Reader) error {
	// Wrap in a limited reader
	s.logger.Debug("Reading message payload", "len", hdr.PayloadLength())
	conn = io.LimitReader(conn, int64(hdr.PayloadLength()))

	// Copy into buffer
	s.receiveLock.Lock()
	defer s.receiveLock.Unlock()

	if s.receiveBuffer == nil {
		// Allocate the receiving buffer just-in-time to fit the full message.
		s.receiveBuffer = bytes.NewBuffer(make([]byte, 0, hdr.HeaderLength()+hdr.PayloadLength()))
	}
	defer s.receiveBuffer.Reset()
	_, err := hdr.WriteTo(s.receiveBuffer)
	if err != nil {
		s.logger.Error("Failed to write header to receiving buffer", "error", err)
		return err
	}
	n, err := io.Copy(s.receiveBuffer, conn)
	if err != nil {
		s.logger.Error("Failed to copy payload to receiving buffer", "error", err)
		return err
	}
	s.logger.Debug("Message payload successfully read", "len", n)

	// We lock the buffer from modification on the time of invocation of OnReceive handler.
	// The slice of bytes passed into the handler is only valid for the duration of the handler invocation.
	// So inside the handler better deserialize message or make a copy of the bytes.
	if s.logger.Enabled(s.ctx, slog.LevelDebug) {
		s.logger.Debug("Invoking OnReceive handler", "message",
			base64.StdEncoding.EncodeToString(s.receiveBuffer.Bytes()))
	}
	s.config.handler.OnReceive(s, s.receiveBuffer) // Invoke OnReceive handler.
	return nil
}

// keepaliveLoop is a long-running goroutine that periodically sends a Ping message to keep the connection alive.
func (s *Session) keepaliveLoop() error {
	for {
		select {
		case <-s.ctx.Done():
			return s.ctx.Err()
		case <-time.After(s.config.keepAliveInterval):
			if s.established.Load() {
				// Get actual Ping message from Protocol.
				p, err := s.config.protocol.Ping()
				if err != nil {
					s.logger.Error("Failed to get ping message", "error", err)
					return ErrKeepAliveProtocolFailure
				}
				if sndErr := s.waitForSend(p); sndErr != nil {
					if errors.Is(sndErr, ErrSessionShutdown) {
						return nil // Exit normally on session termination.
					}
					s.logger.Error("Failed to send ping message", "error", err)
					return ErrKeepAliveTimeout
				}
			}
		}
	}
}

// sendPacket is used to send data.
type sendPacket struct {
	mu  sync.Mutex // Protects reading from r concurrently with sendLoop consumption.
	r   io.Reader
	err chan<- error
}

func newSendPacket(data []byte, ch chan<- error) *sendPacket {
	return &sendPacket{r: bytes.NewReader(data), err: ch}
}

// asyncSendErr is used to try an async send of an error.
func (s *Session) asyncSendErr(ch chan<- error, err error) {
	if ch == nil {
		return
	}
	select {
	case ch <- err:
		s.logger.Debug("Error sent to channel", "error", err)
	default:
	}
}

func suppressContextCancellationError(err error) error {
	if errors.Is(err, context.Canceled) {
		return nil
	}
	return err
}
