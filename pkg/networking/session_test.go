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
	p.On("EmptyHandshake").Return(&textHandshake{}, nil)
	p.On("IsAcceptableHandshake", &textHandshake{v: "hello"}).Once().Return(true)
	p.On("IsAcceptableHandshake", &textHandshake{v: "hello"}).Once().Return(true)
	p.On("EmptyHeader").Return(&textHeader{}, nil)
	p.On("IsAcceptableMessage", &textHeader{l: 2}).Once().Return(true)
	p.On("IsAcceptableMessage", &textHeader{l: 13}).Once().Return(true)

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

	var sWG sync.WaitGroup
	var cWG sync.WaitGroup
	sWG.Add(1)
	go func() {
		sc1 := serverHandler.On("OnHandshake", ss, &textHandshake{v: "hello"}).Once().Return()
		sc1.Run(func(_ mock.Arguments) {
			n, wErr := ss.Write([]byte("hello"))
			require.NoError(t, wErr)
			assert.Equal(t, 5, n)
		})
		sc2 := serverHandler.On("OnReceive", ss, bytes.NewBuffer(encodeMessage("Hello session"))).
			Once().Return()
		sc2.NotBefore(sc1).
			Run(func(_ mock.Arguments) {
				n, wErr := ss.Write(encodeMessage("Hi"))
				require.NoError(t, wErr)
				assert.Equal(t, 6, n)
				sWG.Done()
			})
		sWG.Wait()
	}()

	cWG.Add(1)
	cl1 := clientHandler.On("OnHandshake", cs, &textHandshake{v: "hello"}).Once().Return()
	cl1.Run(func(_ mock.Arguments) {
		n, wErr := cs.Write(encodeMessage("Hello session"))
		require.NoError(t, wErr)
		assert.Equal(t, 17, n)
	})
	cl2 := clientHandler.On("OnReceive", cs, bytes.NewBuffer(encodeMessage("Hi"))).Once().Return()
	cl2.NotBefore(cl1).
		Run(func(_ mock.Arguments) {
			cWG.Done()
		})

	n, err := cs.Write([]byte("hello")) // Send handshake to server.
	require.NoError(t, err)
	assert.Equal(t, 5, n)

	cWG.Wait() // Wait for server to finish.

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
	mockProtocol.On("EmptyHandshake").Return(&textHandshake{}, nil)

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

	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func() {
		err = clientSession.Close()
		assert.ErrorIs(t, err, io.ErrClosedPipe)
		wg.Done()
	}()

	// Unlock "timeout" and close client.
	pc.writeBlocker.Unlock()
	wg.Wait()
}

func TestSessionTimeoutOnMessage(t *testing.T) {
	defer goleak.VerifyNone(t)

	mockProtocol := netmocks.NewMockProtocol(t)
	mockProtocol.On("EmptyHandshake").Return(&textHandshake{}, nil)
	mockProtocol.On("IsAcceptableHandshake", &textHandshake{v: "hello"}).Once().Return(true)
	mockProtocol.On("IsAcceptableHandshake", &textHandshake{v: "hello"}).Once().Return(true)
	mockProtocol.On("EmptyHeader").Return(&textHeader{}, nil)

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

	pc, ok := clientConn.(*pipeConn)
	require.True(t, ok)

	serverHandler.On("OnClose", serverSession).Return()

	clientWG := new(sync.WaitGroup)
	clientWG.Add(1) // Wait for client to send Handshake to server.

	serverWG := new(sync.WaitGroup)
	serverWG.Add(1) // Wait for server to reply with Handshake to client.

	pipeWG := new(sync.WaitGroup)
	pipeWG.Add(1) // Wait for pipe to be locked.

	testWG := new(sync.WaitGroup)
	testWG.Add(1) // Wait for client fail by timeout.

	serverHandler.On("OnClose", serverSession).Return()
	sc1 := serverHandler.On("OnHandshake", serverSession, &textHandshake{v: "hello"}).Once().Return()
	sc1.Run(func(_ mock.Arguments) {
		clientWG.Wait() // Wait for client to send handshake, start replying with Handshake only after that.
		n, wErr := serverSession.Write([]byte("hello"))
		require.NoError(t, wErr)
		assert.Equal(t, 5, n)
		serverWG.Done()
	})

	clientHandler.On("OnClose", clientSession).Return()
	cs1 := clientHandler.On("OnHandshake", clientSession, &textHandshake{v: "hello"}).Once().Return()
	cs1.Run(func(_ mock.Arguments) {
		pipeWG.Wait() // Wait for pipe to be locked.
		// On receiving handshake from server, send the message back to server.
		_, msgErr := clientSession.Write(encodeMessage("Hello session"))
		require.ErrorIs(t, msgErr, networking.ErrConnectionWriteTimeout)
		testWG.Done()
	})

	go func() {
		serverWG.Wait()        // Wait for finishing handshake before closing the pipe.
		pc.writeBlocker.Lock() // Lock pipe after replying with the handshake from server.
		pipeWG.Done()          // Signal that pipe is locked.
	}()

	// Send handshake to server.
	n, err := clientSession.Write([]byte("hello"))
	require.NoError(t, err)
	assert.Equal(t, 5, n)
	clientWG.Done() // Signal that handshake was sent to server.

	testWG.Wait()

	err = serverSession.Close()
	assert.NoError(t, err) // Expect no error on the server side.

	pc.writeBlocker.Unlock() // Unlock the pipe.

	err = clientSession.Close()
	assert.ErrorIs(t, err, io.ErrClosedPipe) // Expect error because connection to the server already closed.
}

func TestDoubleClose(t *testing.T) {
	defer goleak.VerifyNone(t)

	mockProtocol := netmocks.NewMockProtocol(t)
	mockProtocol.On("EmptyHandshake").Return(&textHandshake{}, nil)

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
	mockProtocol.On("EmptyHandshake").Return(&textHandshake{}, nil)
	mockProtocol.On("IsAcceptableHandshake", &textHandshake{v: "hello"}).Once().Return(true)
	mockProtocol.On("IsAcceptableHandshake", &textHandshake{v: "hello"}).Once().Return(true)
	mockProtocol.On("EmptyHeader").Return(&textHeader{}, nil)

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

	clientWG := new(sync.WaitGroup)
	clientWG.Add(1) // Wait for client to send Handshake to server.

	serverWG := new(sync.WaitGroup)
	serverWG.Add(1) // Wait for server to send Handshake to client, after that close the connection from server.
	closeWG := new(sync.WaitGroup)
	closeWG.Add(1) // Wait for server to close the connection.

	testWG := new(sync.WaitGroup)
	testWG.Add(2) // Wait for both client and server to finish.

	serverHandler.On("OnClose", serverSession).Return()
	sc1 := serverHandler.On("OnHandshake", serverSession, &textHandshake{v: "hello"}).Once().Return()
	sc1.Run(func(_ mock.Arguments) {
		clientWG.Wait() // Wait for client to send handshake, start replying with Handshake only after that.
		n, wErr := serverSession.Write([]byte("hello"))
		assert.NoError(t, wErr)
		assert.Equal(t, 5, n)
		go func() {
			// Close server after client received the handshake from server.
			serverWG.Wait() // Wait for client to receive server handshake.
			clErr := serverSession.Close()
			assert.NoError(t, clErr)
			closeWG.Done()
			testWG.Done()
		}()
	})

	clientHandler.On("OnClose", clientSession).Return()
	cs1 := clientHandler.On("OnHandshake", clientSession, &textHandshake{v: "hello"}).Once().Return()
	cs1.Run(func(_ mock.Arguments) {
		// On receiving handshake from server, signal to close the server.
		serverWG.Done()
		// Try to send message to server, but it will fail because server is already closed.
		closeWG.Wait() // Wait for server to close.
		_, msgErr := clientSession.Write(encodeMessage("Hello session"))
		require.ErrorIs(t, msgErr, io.ErrClosedPipe)
		testWG.Done()
	})

	// Send handshake to server.
	n, err := clientSession.Write([]byte("hello"))
	require.NoError(t, err)
	assert.Equal(t, 5, n)
	clientWG.Done() // Signal that handshake was sent to server.

	testWG.Wait() // Wait for client to finish.
	err = clientSession.Close()
	assert.ErrorIs(t, err, io.ErrClosedPipe) // Close reports the same error, because it was registered in the send loop.
}

func TestCloseParentContext(t *testing.T) {
	defer goleak.VerifyNone(t)

	mockProtocol := netmocks.NewMockProtocol(t)
	mockProtocol.On("EmptyHandshake").Return(&textHandshake{}, nil)
	mockProtocol.On("IsAcceptableHandshake", &textHandshake{v: "hello"}).Once().Return(true)
	mockProtocol.On("IsAcceptableHandshake", &textHandshake{v: "hello"}).Once().Return(true)
	mockProtocol.On("EmptyHeader").Return(&textHeader{}, nil)

	clientHandler := netmocks.NewMockHandler(t)
	serverHandler := netmocks.NewMockHandler(t)

	ctx, cancel := context.WithCancel(context.Background())

	clientConn, serverConn := testConnPipe()
	net := networking.NewNetwork()

	clientSession, err := net.NewSession(ctx, clientConn, testConfig(t, mockProtocol, clientHandler, "client"))
	require.NoError(t, err)
	serverSession, err := net.NewSession(ctx, serverConn, testConfig(t, mockProtocol, serverHandler, "server"))
	require.NoError(t, err)

	clientWG := new(sync.WaitGroup)
	clientWG.Add(1) // Wait for client to send Handshake to server.

	serverWG := new(sync.WaitGroup)
	serverWG.Add(1) // Wait for server to send Handshake to client, after that we will close the parent context.

	testWG := new(sync.WaitGroup)
	testWG.Add(2) // Wait for both client and server to finish.

	serverHandler.On("OnClose", serverSession).Return()
	sc1 := serverHandler.On("OnHandshake", serverSession, &textHandshake{v: "hello"}).Once().Return()
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

	clientHandler.On("OnClose", clientSession).Return()

	cs1 := clientHandler.On("OnHandshake", clientSession, &textHandshake{v: "hello"}).Once().Return()
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
	return networking.NewConfig(p, h).
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
