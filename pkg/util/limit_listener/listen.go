package limit_listener

import (
	"net"
	"sync"
	"time"

	"github.com/elliotchance/orderedmap/v2"
)

// LimitListener returns a Listener that accepts at most n simultaneous
// connections from the provided Listener.
func LimitListener(l net.Listener, n int) net.Listener {
	return &limitListener{
		Listener: l,
		sem:      make(chan struct{}, n),
		done:     make(chan struct{}),

		nextConnID:           0,
		connMap:              newConnectionsMap(),
		waitConnQuotaTimeout: 1 * time.Second,
	}
}

type limitListener struct {
	net.Listener
	sem       chan struct{}
	closeOnce sync.Once     // ensures the done chan is only closed once
	done      chan struct{} // no values sent; closed when Close is called

	nextConnID           connectionID
	waitConnQuotaTimeout time.Duration
	connMap              *connectionsMap
}

func newConnectionsMap() *connectionsMap {
	return &connectionsMap{
		OrderedMap: orderedmap.NewOrderedMap[connectionID, *limitListenerConn](),
		mu:         sync.Mutex{},
	}
}

type connectionsMap struct {
	*orderedmap.OrderedMap[connectionID, *limitListenerConn]
	mu sync.Mutex
}

func (m *connectionsMap) setReadOperation(conn *limitListenerConn) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// In order to change insertion order we need to Delete before Set
	m.Delete(conn.id)
	m.Set(conn.id, conn)
}

func (m *connectionsMap) removeOldestConnection() {

	m.mu.Lock()
	el := m.Front()
	if el == nil {
		m.mu.Unlock()
		return
	}
	m.Delete(el.Value.id)
	m.mu.Unlock()

	_ = el.Value.Close()
}

func (m *connectionsMap) removeConnection(conn *limitListenerConn) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Delete(conn.id)
}

type connectionID uint64

func (l *limitListener) getNewConnID() connectionID {
	res := l.nextConnID
	l.nextConnID++
	return res
}

// acquire acquires the limiting semaphore. Returns true if successfully
// accquired, false if the listener is closed and the semaphore is not
// acquired.
func (l *limitListener) acquire() bool {
	for {
		timer := time.NewTimer(l.waitConnQuotaTimeout)
		stopTimer := func() {
			if !timer.Stop() {
				<-timer.C
			}
		}
		select {
		case <-l.done:
			stopTimer()
			return false
		case l.sem <- struct{}{}:
			stopTimer()
			return true
		case <-timer.C:
			l.closeForce()
		}
	}
}

func (l *limitListener) closeForce() {
	l.connMap.removeOldestConnection()
}

func (l *limitListener) markReadOperConn(conn *limitListenerConn) {
	l.connMap.setReadOperation(conn)
}

func (l *limitListener) release(conn *limitListenerConn) {
	<-l.sem
	if conn != nil {
		l.connMap.removeConnection(conn)
	}
}

func (l *limitListener) Accept() (net.Conn, error) {

	if !l.acquire() {
		// If the semaphore isn't acquired because the listener was closed, expect
		// that this call to accept won't block, but immediately return an error.
		// If it instead returns a spurious connection (due to a bug in the
		// Listener, such as https://golang.org/issue/50216), we immediately close
		// it and try again. Some buggy Listener implementations (like the one in
		// the aforementioned issue) seem to assume that Accept will be called to
		// completion, and may otherwise fail to clean up the client end of pending
		// connections.
		for {
			c, err := l.Listener.Accept()
			if err != nil {
				return nil, err
			}
			_ = c.Close()
		}
	}

	c, err := l.Listener.Accept()
	if err != nil {
		l.release(nil)
		return nil, err
	}
	return &limitListenerConn{Conn: c, release: l.release, markReadOper: l.markReadOperConn, id: l.getNewConnID()}, nil
}

func (l *limitListener) Close() error {
	err := l.Listener.Close()
	l.closeOnce.Do(func() { close(l.done) })
	return err
}

type limitListenerConn struct {
	net.Conn
	releaseOnce sync.Once
	release     func(*limitListenerConn)

	id           connectionID
	markReadOper func(*limitListenerConn)
}

func (l *limitListenerConn) Close() error {
	err := l.Conn.Close()
	l.releaseOnce.Do(func() { l.release(l) })
	return err
}

func (l *limitListenerConn) Read(b []byte) (n int, err error) {
	l.markReadOper(l)
	return l.Conn.Read(b)
}
