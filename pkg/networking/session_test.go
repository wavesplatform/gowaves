package networking_test

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/neilotoole/slogt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/wavesplatform/gowaves/pkg/networking"
	netmocks "github.com/wavesplatform/gowaves/pkg/networking/mocks"
)

func TestSuccessfulSession(t *testing.T) {
	defer goleak.VerifyNone(t)

	p := netmocks.NewMockProtocol(t)
	p.On("EmptyHandshake").Return(&textHandshake{})
	p.On("EmptyHandshake").Return(&textHandshake{})

	clientHandler := netmocks.NewMockHandler(t)
	serverHandler := netmocks.NewMockHandler(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	clientConn, serverConn := testConnPipe()
	net := networking.NewNetwork()

	cs, err := net.NewSession(ctx, clientConn, testConfig(t, p, clientHandler, "client"))
	require.NoError(t, err)
	ss, err := net.NewSession(ctx, serverConn, testConfig(t, p, serverHandler, "server"))
	require.NoError(t, err)

	p.On("IsAcceptableHandshake", cs, &textHandshake{v: "hello"}).Once().Return(true)
	p.On("IsAcceptableHandshake", ss, &textHandshake{v: "hello"}).Once().Return(true)
	p.On("EmptyHeader").Return(&textHeader{}, nil)
	p.On("IsAcceptableMessage", cs, &textHeader{l: 2}).Once().Return(true)
	p.On("IsAcceptableMessage", ss, &textHeader{l: 13}).Once().Return(true)

	done := make(chan struct{})
	timeout := time.After(time.Second)

	serverReady := make(chan struct{})
	clientReady := make(chan struct{})

	serverHandler.On("OnHandshake", ss, &textHandshake{v: "hello"}).Once().
		Run(func(_ mock.Arguments) {
			n, wErr := ss.Write([]byte("hello"))
			require.NoError(t, wErr)
			assert.Equal(t, 5, n)
		})
	serverHandler.On("OnReceive", ss, bytes.NewBuffer(encodeMessage("Hello session"))).Once().
		Run(func(_ mock.Arguments) {
			n, wErr := ss.Write(encodeMessage("Hi"))
			require.NoError(t, wErr)
			assert.Equal(t, 6, n)
			close(serverReady)
		})

	clientHandler.On("OnHandshake", cs, &textHandshake{v: "hello"}).Once().
		Run(func(_ mock.Arguments) {
			n, wErr := cs.Write(encodeMessage("Hello session"))
			require.NoError(t, wErr)
			assert.Equal(t, 17, n)
		})
	clientHandler.On("OnReceive", cs, bytes.NewBuffer(encodeMessage("Hi"))).Once().
		Run(func(_ mock.Arguments) {
			close(clientReady)
		})

	go func() {
		n, wErr := cs.Write([]byte("hello")) // Send handshake to server.
		require.NoError(t, wErr)
		assert.Equal(t, 5, n)
	}()

	// Wait for both sides to complete, or timeout
	go func() {
		<-serverReady
		<-clientReady
		done <- struct{}{}
	}()

	select {
	case <-done:
		// success
	case <-timeout:
		assert.Fail(t, "timed out waiting for server to start")
	}

	clientHandler.On("OnClose", cs).Return()
	serverHandler.On("OnClose", ss).Return()
	err = cs.Close()
	assert.NoError(t, err)
	err = ss.Close()
	assert.NoError(t, err)
}

func TestSessionTimeoutOnHandshake(t *testing.T) {
	defer goleak.VerifyNone(t)

	mockProtocol := netmocks.NewMockProtocol(t)
	mockProtocol.On("EmptyHandshake").Return(&textHandshake{})

	clientHandler := netmocks.NewMockHandler(t)
	serverHandler := netmocks.NewMockHandler(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	clientConn, serverConn := testConnPipe()
	net := networking.NewNetwork()

	clientSession, err := net.NewSession(ctx, clientConn, testConfig(t, mockProtocol, clientHandler, "client"))
	require.NoError(t, err)
	clientHandler.On("OnClose", clientSession).Return()

	serverSession, err := net.NewSession(ctx, serverConn, testConfig(t, mockProtocol, serverHandler, "server"))
	require.NoError(t, err)
	serverHandler.On("OnClose", serverSession).Return()

	// Lock
	pc, ok := clientConn.(*pipeConn)
	require.True(t, ok)
	pc.writeBlocker.Lock()

	// Send handshake to server, but writing will block because the clientConn is locked.
	n, err := clientSession.Write([]byte("hello"))
	require.ErrorIs(t, err, networking.ErrConnectionWriteTimeout)
	assert.Equal(t, 0, n)

	err = serverSession.Close()
	assert.NoError(t, err)

	var tWG sync.WaitGroup
	tWG.Add(1)

	// Unlock "timeout" and close client.
	pc.writeBlocker.Unlock()

	go func() {
		err = clientSession.Close()
		assert.ErrorIs(t, err, io.ErrClosedPipe)
		tWG.Done()
	}()

	tWG.Wait()
}

func TestSessionTimeoutOnMessage(t *testing.T) {
	defer goleak.VerifyNone(t)

	mockProtocol := netmocks.NewMockProtocol(t)
	mockProtocol.On("EmptyHandshake").Return(&textHandshake{})
	mockProtocol.On("EmptyHeader").Return(&textHeader{})

	clientHandler := netmocks.NewMockHandler(t)
	serverHandler := netmocks.NewMockHandler(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	clientConn, serverConn := testConnPipe()
	net := networking.NewNetwork()

	clientSession, err := net.NewSession(ctx, clientConn, testConfig(t, mockProtocol, clientHandler, "client"))
	require.NoError(t, err)
	serverSession, err := net.NewSession(ctx, serverConn, testConfig(t, mockProtocol, serverHandler, "server"))
	require.NoError(t, err)

	mockProtocol.On("IsAcceptableHandshake", serverSession, &textHandshake{v: "hello"}).Once().Return(true)
	mockProtocol.On("IsAcceptableHandshake", clientSession, &textHandshake{v: "hello"}).Once().Return(true)

	pc, ok := clientConn.(*pipeConn)
	require.True(t, ok)

	serverHandler.On("OnClose", serverSession).Return()

	handshakeSent := make(chan struct{})
	serverReplied := make(chan struct{})
	pipeLocked := make(chan struct{})
	clientTimedOut := make(chan struct{})

	timeout := time.After(2 * time.Second)

	serverHandler.On("OnClose", serverSession).Return()
	serverHandler.On("OnHandshake", serverSession, &textHandshake{v: "hello"}).Once().
		Run(func(_ mock.Arguments) {
			<-handshakeSent // Wait for client to send handshake, start replying with Handshake only after that.
			n, wErr := serverSession.Write([]byte("hello"))
			require.NoError(t, wErr)
			assert.Equal(t, 5, n)
			close(serverReplied)
		})

	clientHandler.On("OnClose", clientSession).Return()
	clientHandler.On("OnHandshake", clientSession, &textHandshake{v: "hello"}).Once().
		Run(func(_ mock.Arguments) {
			<-pipeLocked // Wait for pipe to be locked.
			// On receiving handshake from server, send the message back to server.
			_, msgErr := clientSession.Write(encodeMessage("Hello session"))
			require.ErrorIs(t, msgErr, networking.ErrConnectionWriteTimeout)
			close(clientTimedOut)
		})

	// Send handshake to server.
	go func() {
		n, csErr := clientSession.Write([]byte("hello"))
		require.NoError(t, csErr)
		assert.Equal(t, 5, n)
		close(handshakeSent)
	}()

	// Lock the pipe after server replies with handshake
	go func() {
		<-serverReplied
		pc.writeBlocker.Lock()
		close(pipeLocked)
	}()

	// Wait for the client to time out, or timeout the test
	select {
	case <-clientTimedOut:
		// OK
	case <-timeout:
		assert.Fail(t, "timed out waiting for handshake")
	}
	err = serverSession.Close()
	assert.NoError(t, err) // Expect no error on the server side.

	pc.writeBlocker.Unlock() // Unlock the pipe.

	err = clientSession.Close()
	assert.ErrorIs(t, err, io.ErrClosedPipe) // Expect this error because connection to the server already closed.
}

func TestDoubleClose(t *testing.T) {
	defer goleak.VerifyNone(t)

	mockProtocol := netmocks.NewMockProtocol(t)
	mockProtocol.On("EmptyHandshake").Return(&textHandshake{})

	clientHandler := netmocks.NewMockHandler(t)
	serverHandler := netmocks.NewMockHandler(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	clientConn, serverConn := testConnPipe()
	net := networking.NewNetwork()

	clientSession, err := net.NewSession(ctx, clientConn, testConfig(t, mockProtocol, clientHandler, "client"))
	require.NoError(t, err)
	serverSession, err := net.NewSession(ctx, serverConn, testConfig(t, mockProtocol, serverHandler, "server"))
	require.NoError(t, err)

	clientHandler.On("OnClose", clientSession).Return()
	serverHandler.On("OnClose", serverSession).Return()

	err = clientSession.Close()
	assert.NoError(t, err)
	err = clientSession.Close()
	assert.NoError(t, err)

	err = serverSession.Close()
	assert.NoError(t, err)
	err = serverSession.Close()
	assert.NoError(t, err)
}

func TestOnClosedByOtherSide(t *testing.T) {
	defer goleak.VerifyNone(t)

	mockProtocol := netmocks.NewMockProtocol(t)
	mockProtocol.On("EmptyHandshake").Return(&textHandshake{})
	mockProtocol.On("EmptyHeader").Return(&textHeader{})

	clientHandler := netmocks.NewMockHandler(t)
	serverHandler := netmocks.NewMockHandler(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	clientConn, serverConn := testConnPipe()
	net := networking.NewNetwork()

	clientSession, err := net.NewSession(ctx, clientConn, testConfig(t, mockProtocol, clientHandler, "client"))
	require.NoError(t, err)
	serverSession, err := net.NewSession(ctx, serverConn, testConfig(t, mockProtocol, serverHandler, "server"))
	require.NoError(t, err)

	mockProtocol.On("IsAcceptableHandshake", clientSession, &textHandshake{v: "hello"}).Once().Return(true)
	mockProtocol.On("IsAcceptableHandshake", serverSession, &textHandshake{v: "hello"}).Once().Return(true)

	clientSentHandshake := make(chan struct{})
	serverSentHandshake := make(chan struct{})
	serverClosed := make(chan struct{})
	clientReceivedClose := make(chan struct{})

	timeout := time.After(2 * time.Second)

	serverHandler.On("OnClose", serverSession).Return()
	serverHandler.On("OnHandshake", serverSession, &textHandshake{v: "hello"}).Once().
		Run(func(_ mock.Arguments) {
			<-clientSentHandshake // Wait for client to send handshake, start replying with Handshake only after that.
			n, wErr := serverSession.Write([]byte("hello"))
			assert.NoError(t, wErr)
			assert.Equal(t, 5, n)
			close(serverSentHandshake)

			go func() {
				// Close server after client received the handshake from server.
				<-clientReceivedClose // Wait for client to receive server handshake.
				clErr := serverSession.Close()
				assert.NoError(t, clErr)
				close(serverClosed)
			}()
		})

	clientHandler.On("OnClose", clientSession).Return()
	clientHandler.On("OnHandshake", clientSession, &textHandshake{v: "hello"}).Once().
		Run(func(_ mock.Arguments) {
			// On receiving handshake from server, signal to close the server.
			close(clientReceivedClose)
			// Try to send message to server, but it will fail because server is already closed.
			<-serverClosed
			_, msgErr := clientSession.Write(encodeMessage("Hello session"))
			require.Error(t, msgErr)
			assert.True(t, errors.Is(msgErr, io.ErrClosedPipe) || errors.Is(msgErr, networking.ErrSessionShutdown))
		})

	// Send handshake to server.
	// Send handshake from client
	go func() {
		n, csErr := clientSession.Write([]byte("hello"))
		require.NoError(t, csErr)
		assert.Equal(t, 5, n)
		close(clientSentHandshake)
	}()

	select {
	case <-serverClosed:
		// OK
	case <-timeout:
		assert.Fail(t, "timed out waiting for server to close")
	}

	err = clientSession.Close()
	if err != nil {
		assert.ErrorIs(t, err, io.ErrClosedPipe) // Only this error can go through.
	}
}

func TestCloseParentContext(t *testing.T) {
	defer goleak.VerifyNone(t)

	mockProtocol := netmocks.NewMockProtocol(t)
	mockProtocol.On("EmptyHandshake").Return(&textHandshake{})
	mockProtocol.On("EmptyHeader").Return(&textHeader{})

	clientHandler := netmocks.NewMockHandler(t)
	serverHandler := netmocks.NewMockHandler(t)

	ctx, cancel := context.WithCancel(context.Background())

	clientConn, serverConn := testConnPipe()
	net := networking.NewNetwork()

	clientSession, err := net.NewSession(ctx, clientConn, testConfig(t, mockProtocol, clientHandler, "client"))
	require.NoError(t, err)
	serverSession, err := net.NewSession(ctx, serverConn, testConfig(t, mockProtocol, serverHandler, "server"))
	require.NoError(t, err)

	mockProtocol.On("IsAcceptableHandshake", clientSession, &textHandshake{v: "hello"}).Once().Return(true)
	mockProtocol.On("IsAcceptableHandshake", serverSession, &textHandshake{v: "hello"}).Once().Return(true)

	clientWG := new(sync.WaitGroup)
	clientWG.Add(1) // Wait for client to send Handshake to server.

	serverWG := new(sync.WaitGroup)
	serverWG.Add(1) // Wait for server to send Handshake to client, after that we will close the parent context.

	testWG := new(sync.WaitGroup)
	testWG.Add(2) // Wait for both client and server to finish.

	serverHandler.On("OnClose", serverSession).Return()
	sc1 := serverHandler.On("OnHandshake", serverSession, &textHandshake{v: "hello"}).Once().Return()
	go func() {
		sc1.Run(func(_ mock.Arguments) {
			clientWG.Wait() // Wait for client to send handshake, start replying with Handshake only after that.
			n, wErr := serverSession.Write([]byte("hello"))
			assert.NoError(t, wErr)
			assert.Equal(t, 5, n)
			go func() {
				serverWG.Wait() // Wait for client to receive server handshake.
				cancel()        // Close parent context.
				testWG.Done()
			}()
		})
	}()
	clientHandler.On("OnClose", clientSession).Return()

	cs1 := clientHandler.On("OnHandshake", clientSession, &textHandshake{v: "hello"}).Once().Return()
	go func() {
		cs1.Run(func(_ mock.Arguments) {
			// On receiving handshake from server, signal to close the server.
			serverWG.Done()
			go func() {
				// Try to send message to server, but it will fail because server is already closed.
				time.Sleep(10 * time.Millisecond) // Wait for server to close.
				_, msgErr := clientSession.Write(encodeMessage("Hello session"))
				require.ErrorIs(t, msgErr, networking.ErrSessionShutdown)
				testWG.Done()
			}()
		})
	}()

	// Send handshake to server.
	n, err := clientSession.Write([]byte("hello"))
	require.NoError(t, err)
	assert.Equal(t, 5, n)
	clientWG.Done() // Signal that handshake was sent to server.

	testWG.Wait() // Wait for all interactions to finish.

	err = clientSession.Close()
	assert.NoError(t, err)
	err = serverSession.Close()
	assert.NoError(t, err)
}

func testConfig(t testing.TB, p networking.Protocol, h networking.Handler, direction string) *networking.Config {
	log := slogt.New(t)
	return networking.NewConfig().
		WithProtocol(p).
		WithHandler(h).
		WithSlogHandler(log.Handler()).
		WithWriteTimeout(1 * time.Second).
		WithKeepAliveDisabled().
		WithSlogAttribute(slog.String("direction", direction))
}

type pipeConn struct {
	reader       *io.PipeReader
	writer       *io.PipeWriter
	writeBlocker sync.Mutex
}

func (p *pipeConn) Read(b []byte) (int, error) {
	return p.reader.Read(b)
}

func (p *pipeConn) Write(b []byte) (int, error) {
	p.writeBlocker.Lock()
	defer p.writeBlocker.Unlock()
	return p.writer.Write(b)
}

func (p *pipeConn) Close() error {
	rErr := p.reader.Close()
	wErr := p.writer.Close()
	return errors.Join(rErr, wErr)
}

func testConnPipe() (io.ReadWriteCloser, io.ReadWriteCloser) {
	read1, write1 := io.Pipe()
	read2, write2 := io.Pipe()
	conn1 := &pipeConn{reader: read1, writer: write2}
	conn2 := &pipeConn{reader: read2, writer: write1}
	return conn1, conn2
}

func encodeMessage(s string) []byte {
	msg := make([]byte, 4+len(s))
	binary.BigEndian.PutUint32(msg[:4], uint32(len(s)))
	copy(msg[4:], s)
	return msg
}

// We have to use the "real" handshake, not a mock, because we are reading or writing to a "real" piped connection.
type textHandshake struct {
	v string
}

func (h *textHandshake) ReadFrom(r io.Reader) (int64, error) {
	buf := make([]byte, 5)
	n, err := io.ReadFull(r, buf)
	if err != nil {
		return int64(n), err
	}
	h.v = string(buf[:n])
	return int64(n), nil
}

func (h *textHandshake) WriteTo(w io.Writer) (int64, error) {
	buf := []byte(h.v)
	n, err := w.Write(buf)
	return int64(n), err
}

// We have to use the "real" header, not a mock, because we are reading or writing to a "real" piped connection.
type textHeader struct {
	l uint32
}

func (h *textHeader) HeaderLength() uint32 {
	return 4
}

func (h *textHeader) PayloadLength() uint32 {
	return h.l
}

func (h *textHeader) ReadFrom(r io.Reader) (int64, error) {
	hdr := make([]byte, 4)
	n, err := io.ReadFull(r, hdr)
	if err != nil {
		return int64(n), err
	}
	h.l = binary.BigEndian.Uint32(hdr)
	return int64(n), nil
}

func (h *textHeader) WriteTo(w io.Writer) (int64, error) {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, h.l)
	n, err := w.Write(buf)
	return int64(n), err
}
