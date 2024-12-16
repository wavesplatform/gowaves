package networking_test

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"log/slog"
	"runtime"
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
		sc2 := serverHandler.On("OnReceive", ss, encodeMessage("Hello session")).Once().Return()
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
	cl2 := clientHandler.On("OnReceive", cs, encodeMessage("Hi")).Once().Return()
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

	mockProtocol.On("EmptyHandshake").Return(&textHandshake{}, nil)
	serverHandler.On("OnClose", serverSession).Return()
	clientHandler.On("OnClose", clientSession).Return()

	// Lock
	pc, ok := clientConn.(*pipeConn)
	require.True(t, ok)
	pc.writeBlocker.Lock()
	runtime.Gosched()

	// Send handshake to server, but writing will block because the clientConn is locked.
	n, err := clientSession.Write([]byte("hello"))
	require.Error(t, err)
	assert.Equal(t, 0, n)

	runtime.Gosched()

	err = serverSession.Close()
	assert.NoError(t, err)

	// Unlock "timeout" and close client.
	pc.writeBlocker.Unlock()
	err = clientSession.Close()
	assert.Error(t, err)
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

	var serverWG sync.WaitGroup
	var clientWG sync.WaitGroup
	serverWG.Add(1)
	clientWG.Add(1)
	go func() {
		sc1 := serverHandler.On("OnHandshake", serverSession, &textHandshake{v: "hello"}).Once().Return()
		sc1.Run(func(_ mock.Arguments) {
			n, wErr := serverSession.Write([]byte("hello"))
			require.NoError(t, wErr)
			assert.Equal(t, 5, n)
			serverWG.Done()
		})
		serverWG.Wait() // Wait for finishing handshake before closing the pipe.

		// Lock pipe after replying with the handshake from server.
		pc.writeBlocker.Lock()
		clientWG.Done() // Signal that pipe is locked.
	}()

	serverHandler.On("OnClose", serverSession).Return()
	clientHandler.On("OnClose", clientSession).Return()

	// Send handshake to server.
	n, err := clientSession.Write([]byte("hello"))
	require.NoError(t, err)
	assert.Equal(t, 5, n)

	cs1 := clientHandler.On("OnHandshake", clientSession, &textHandshake{v: "hello"}).Once().Return()
	cs1.Run(func(_ mock.Arguments) {
		clientWG.Wait() // Wait for pipe to be locked.
		// On receiving handshake from server, send the message back to server.
		_, msgErr := clientSession.Write(encodeMessage("Hello session"))
		require.Error(t, msgErr)
	})

	time.Sleep(1 * time.Second) // Let timeout occur.

	err = serverSession.Close()
	assert.NoError(t, err) // Expect no error on the server side.

	pc.writeBlocker.Unlock() // Unlock the pipe.

	err = clientSession.Close()
	assert.Error(t, err) // Expect error because connection to the server already closed.
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

	var closeWG sync.WaitGroup
	closeWG.Add(1)

	var wg sync.WaitGroup
	wg.Add(2)

	serverHandler.On("OnClose", serverSession).Return()
	sc1 := serverHandler.On("OnHandshake", serverSession, &textHandshake{v: "hello"}).Once().Return()
	sc1.Run(func(_ mock.Arguments) {
		n, wErr := serverSession.Write([]byte("hello"))
		assert.NoError(t, wErr)
		assert.Equal(t, 5, n)
		go func() {
			// Close server after client received the handshake from server.
			closeWG.Wait() // Wait for client to receive server handshake.
			clErr := serverSession.Close()
			assert.NoError(t, clErr)
			wg.Done()
		}()
	})

	clientHandler.On("OnClose", clientSession).Return()

	// Send handshake to server.
	n, err := clientSession.Write([]byte("hello"))
	require.NoError(t, err)
	assert.Equal(t, 5, n)

	cs1 := clientHandler.On("OnHandshake", clientSession, &textHandshake{v: "hello"}).Once().Return()
	cs1.Run(func(_ mock.Arguments) {
		// On receiving handshake from server, signal to close the server.
		closeWG.Done()
		// Try to send message to server, but it will fail because server is already closed.
		time.Sleep(10 * time.Millisecond) // Wait for server to close.
		_, msgErr := clientSession.Write(encodeMessage("Hello session"))
		require.Error(t, msgErr)
		wg.Done()
	})

	wg.Wait() // Wait for client to finish.
	err = clientSession.Close()
	assert.Error(t, err) // Close reports the same error, because it was registered in the send loop.
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
	defer cancel()

	clientConn, serverConn := testConnPipe()
	net := networking.NewNetwork()

	clientSession, err := net.NewSession(ctx, clientConn, testConfig(t, mockProtocol, clientHandler, "client"))
	require.NoError(t, err)
	serverSession, err := net.NewSession(ctx, serverConn, testConfig(t, mockProtocol, serverHandler, "server"))
	require.NoError(t, err)

	var closeWG sync.WaitGroup
	closeWG.Add(1)

	var wg sync.WaitGroup
	wg.Add(2)

	serverHandler.On("OnClose", serverSession).Return()
	sc1 := serverHandler.On("OnHandshake", serverSession, &textHandshake{v: "hello"}).Once().Return()
	sc1.Run(func(_ mock.Arguments) {
		n, wErr := serverSession.Write([]byte("hello"))
		assert.NoError(t, wErr)
		assert.Equal(t, 5, n)
		go func() {
			closeWG.Wait() // Wait for client to receive server handshake.
			cancel()       // Close parent context.
			wg.Done()
		}()
	})

	clientHandler.On("OnClose", clientSession).Return()

	// Send handshake to server.
	n, err := clientSession.Write([]byte("hello"))
	require.NoError(t, err)
	assert.Equal(t, 5, n)

	cs1 := clientHandler.On("OnHandshake", clientSession, &textHandshake{v: "hello"}).Once().Return()
	cs1.Run(func(_ mock.Arguments) {
		// On receiving handshake from server, signal to close the server.
		closeWG.Done()
		// Try to send message to server, but it will fail because server is already closed.
		time.Sleep(10 * time.Millisecond) // Wait for server to close.
		_, msgErr := clientSession.Write(encodeMessage("Hello session"))
		require.Error(t, msgErr)
		wg.Done()
	})

	wg.Wait() // Wait for client to finish.

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
