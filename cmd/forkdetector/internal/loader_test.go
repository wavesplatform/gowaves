package internal

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"net"
	"testing"
	"time"
)

type mockAddr struct {
	addr string
}

func (a *mockAddr) Network() string {
	return "mock"
}

func (a *mockAddr) String() string {
	return a.addr
}

type mockConn struct {
	local  net.Addr
	remote net.Addr
}

func (c *mockConn) Read(b []byte) (n int, err error) { panic("mock - not implemented") }

func (c *mockConn) Write(b []byte) (n int, err error) { panic("mock - not implemented") }

func (c *mockConn) Close() error { panic("mock - not implemented") }

func (c *mockConn) LocalAddr() net.Addr { return c.local }

func (c *mockConn) RemoteAddr() net.Addr { return c.remote }

func (c *mockConn) SetDeadline(t time.Time) error { panic("mock - not implemented") }

func (c *mockConn) SetReadDeadline(t time.Time) error { panic("mock - not implemented") }

func (c *mockConn) SetWriteDeadline(t time.Time) error { panic("mock - not implemented") }

func createConnectionWithMock(id string) *Conn {
	l := &mockAddr{"local"}
	r := &mockAddr{id}
	c := &mockConn{local: l, remote: r}
	conn := new(Conn)
	conn.RawConn = c
	return conn
}

func TestRequestQueuePickRandomlyFromEmpty(t *testing.T) {
	q := new(requestQueue)
	_, c, ok := q.pickRandomly()
	assert.False(t, ok)
	assert.Nil(t, c)
	assert.Equal(t, 0, len(q.blocks))
}

func TestRequestQueueOneConnection(t *testing.T) {
	b1, err := crypto.NewSignatureFromBase58("3kSNeUztQ6HrTUGTmwWqoCANRy99s65RNyub7ENnWsMQVNNQYFLyfRYQycvshvpus7TazrnevxKvGgDw82D4yhMk")
	require.NoError(t, err)
	b2, err := crypto.NewSignatureFromBase58("443nnYBRjjt8AZApoYtf5zGgukaTdCfQfBhmZQ55nyVkBXmhjzbweBaDVX23D9b5mMMXzLR6YyGHqq14BppHvAQZ")
	require.NoError(t, err)
	b3, err := crypto.NewSignatureFromBase58("YAEPx9iMfjXwbfF7Uxsi18a4y9CNZbJsavmwtRmaXiS6gcsRWdzWeHQU9jDdUNdrwQb76s1mMZNMh7cZvmoyZxz")
	require.NoError(t, err)

	conn := createConnectionWithMock("1")

	q := new(requestQueue)

	q.enqueue(b1, conn)
	assert.Equal(t, 1, len(q.blocks))
	q.enqueue(b2, conn)
	assert.Equal(t, 2, len(q.blocks))

	b, c, ok := q.pickRandomly()
	assert.True(t, ok)
	assert.Equal(t, b1, b)
	assert.Equal(t, conn, c)

	q.enqueue(b3, conn)
	assert.Equal(t, 3, len(q.blocks))

	b, c, ok = q.pickRandomly()
	assert.True(t, ok)
	assert.Equal(t, b2, b)
	assert.Equal(t, conn, c)

	b, c, ok = q.pickRandomly()
	assert.True(t, ok)
	assert.Equal(t, b3, b)
	assert.Equal(t, conn, c)

	b, c, ok = q.pickRandomly()
	assert.False(t, ok)
	assert.Nil(t, c)
}

func TestRequestQueueFewConnections(t *testing.T) {
	b1, err := crypto.NewSignatureFromBase58("3kSNeUztQ6HrTUGTmwWqoCANRy99s65RNyub7ENnWsMQVNNQYFLyfRYQycvshvpus7TazrnevxKvGgDw82D4yhMk")
	require.NoError(t, err)
	b2, err := crypto.NewSignatureFromBase58("443nnYBRjjt8AZApoYtf5zGgukaTdCfQfBhmZQ55nyVkBXmhjzbweBaDVX23D9b5mMMXzLR6YyGHqq14BppHvAQZ")
	require.NoError(t, err)
	b3, err := crypto.NewSignatureFromBase58("YAEPx9iMfjXwbfF7Uxsi18a4y9CNZbJsavmwtRmaXiS6gcsRWdzWeHQU9jDdUNdrwQb76s1mMZNMh7cZvmoyZxz")
	require.NoError(t, err)
	b4, err := crypto.NewSignatureFromBase58("5UwsMWFNvuGEq9s72aYa4wXXDSaTbhtqFaG2Y9y1o6f3nskZ1rjwbZMA47hu8dQopvJ3ZeTFKRG6bo47WHYBvu9T")
	require.NoError(t, err)
	b5, err := crypto.NewSignatureFromBase58("5h3MqnsxhwPgHX281nLiC1oxVjTQn1ZYngk4Ef8EZ9s6zieVyuTpTyMLBkrzwG9jzTpM3CqhWD1szCkDRpMmidTj")
	require.NoError(t, err)

	conn1 := createConnectionWithMock("1")
	conn2 := createConnectionWithMock("2")
	conn3 := createConnectionWithMock("3")
	conn4 := createConnectionWithMock("4")

	q := new(requestQueue)

	q.enqueue(b1, conn1)
	q.enqueue(b1, conn2)
	assert.Equal(t, 1, len(q.blocks))
	q.enqueue(b2, conn3)
	q.enqueue(b2, conn4)
	assert.Equal(t, 2, len(q.blocks))
	q.enqueue(b3, conn1)
	q.enqueue(b3, conn3)
	assert.Equal(t, 3, len(q.blocks))
	q.enqueue(b4, conn2)
	q.enqueue(b4, conn4)
	assert.Equal(t, 4, len(q.blocks))
	q.enqueue(b5, conn4)
	q.enqueue(b5, conn3)
	q.enqueue(b5, conn2)
	q.enqueue(b5, conn1)
	assert.Equal(t, 5, len(q.blocks))

	b, c, ok := q.pickRandomly()
	assert.True(t, ok)
	assert.Equal(t, b1, b)
	assert.Contains(t, []*Conn{conn1, conn2}, c)
	assert.NotContains(t, []*Conn{conn3, conn4}, c)
	assert.Equal(t, 5, len(q.blocks))

	b, c, ok = q.pickRandomly()
	assert.True(t, ok)
	assert.Equal(t, b2, b)
	assert.Contains(t, []*Conn{conn3, conn4}, c)
	assert.NotContains(t, []*Conn{conn1, conn2}, c)
	assert.Equal(t, 5, len(q.blocks))

	b, c, ok = q.pickRandomly()
	assert.True(t, ok)
	assert.Equal(t, b3, b)
	assert.Contains(t, []*Conn{conn1, conn3}, c)
	assert.NotContains(t, []*Conn{conn2, conn4}, c)
	assert.Equal(t, 5, len(q.blocks))

	b, c, ok = q.pickRandomly()
	assert.True(t, ok)
	assert.Equal(t, b4, b)
	assert.Contains(t, []*Conn{conn2, conn4}, c)
	assert.NotContains(t, []*Conn{conn1, conn3}, c)
	assert.Equal(t, 5, len(q.blocks))

	b, c, ok = q.pickRandomly()
	assert.True(t, ok)
	assert.Equal(t, b5, b)
	assert.Contains(t, []*Conn{conn1, conn2, conn3, conn4}, c)
	assert.Equal(t, 5, len(q.blocks))

	b, c, ok = q.pickRandomly()
	assert.False(t, ok)
	assert.Nil(t, c)
	assert.Equal(t, 5, len(q.blocks))
}

func TestRequestQueueEnqueueDequeue(t *testing.T) {
	b0, err := crypto.NewSignatureFromBase58("5STx5DJDUo1PvhTvnk4Mb7tVniJvHMt6RrcHKhtMfsL5xD5mQGrPWesXGAYNpehreBe8sEoaJ8rErqcEyXGwpGBG")
	require.NoError(t, err)
	b1, err := crypto.NewSignatureFromBase58("3kSNeUztQ6HrTUGTmwWqoCANRy99s65RNyub7ENnWsMQVNNQYFLyfRYQycvshvpus7TazrnevxKvGgDw82D4yhMk")
	require.NoError(t, err)
	b2, err := crypto.NewSignatureFromBase58("443nnYBRjjt8AZApoYtf5zGgukaTdCfQfBhmZQ55nyVkBXmhjzbweBaDVX23D9b5mMMXzLR6YyGHqq14BppHvAQZ")
	require.NoError(t, err)
	b3, err := crypto.NewSignatureFromBase58("YAEPx9iMfjXwbfF7Uxsi18a4y9CNZbJsavmwtRmaXiS6gcsRWdzWeHQU9jDdUNdrwQb76s1mMZNMh7cZvmoyZxz")
	require.NoError(t, err)
	b4, err := crypto.NewSignatureFromBase58("5UwsMWFNvuGEq9s72aYa4wXXDSaTbhtqFaG2Y9y1o6f3nskZ1rjwbZMA47hu8dQopvJ3ZeTFKRG6bo47WHYBvu9T")
	require.NoError(t, err)
	b5, err := crypto.NewSignatureFromBase58("5h3MqnsxhwPgHX281nLiC1oxVjTQn1ZYngk4Ef8EZ9s6zieVyuTpTyMLBkrzwG9jzTpM3CqhWD1szCkDRpMmidTj")
	require.NoError(t, err)

	conn := createConnectionWithMock("1")

	q := new(requestQueue)
	assert.Equal(t, 0, len(q.blocks))
	q.dequeue(b0)
	assert.Equal(t, 0, len(q.blocks))

	q.enqueue(b1, conn)
	assert.Equal(t, 1, len(q.blocks))
	q.enqueue(b2, conn)
	assert.Equal(t, 2, len(q.blocks))

	b, c, ok := q.pickRandomly()
	assert.True(t, ok)
	assert.Equal(t, b1, b)
	assert.Equal(t, conn, c)

	q.enqueue(b3, conn)
	assert.Equal(t, 3, len(q.blocks))
	q.dequeue(b0)
	assert.Equal(t, 3, len(q.blocks))
	q.dequeue(b1)
	assert.Equal(t, 2, len(q.blocks))

	b, c, ok = q.pickRandomly()
	assert.True(t, ok)
	assert.Equal(t, b2, b)
	assert.Equal(t, conn, c)

	q.dequeue(b2)
	assert.Equal(t, 1, len(q.blocks))
	q.dequeue(b2)
	assert.Equal(t, 1, len(q.blocks))

	b, c, ok = q.pickRandomly()
	assert.True(t, ok)
	assert.Equal(t, b3, b)
	assert.Equal(t, conn, c)

	b, c, ok = q.pickRandomly()
	assert.False(t, ok)
	assert.Nil(t, c)

	q.dequeue(b3)
	assert.Equal(t, 0, len(q.blocks))

	q.enqueue(b4, conn)
	assert.Equal(t, 1, len(q.blocks))

	b, c, ok = q.pickRandomly()
	assert.True(t, ok)
	assert.Equal(t, b4, b)
	assert.Equal(t, conn, c)

	q.dequeue(b4)
	assert.Equal(t, 0, len(q.blocks))

	q.enqueue(b5, conn)
	assert.Equal(t, 1, len(q.blocks))

	b, c, ok = q.pickRandomly()
	assert.True(t, ok)
	assert.Equal(t, b5, b)
	assert.Equal(t, conn, c)

	q.dequeue(b5)
	assert.Equal(t, 0, len(q.blocks))

	b, c, ok = q.pickRandomly()
	assert.False(t, ok)
	assert.Nil(t, c)

	q.dequeue(b5)
	assert.Equal(t, 0, len(q.blocks))
}

func TestRequestQueueReset(t *testing.T) {
	b1, err := crypto.NewSignatureFromBase58("3kSNeUztQ6HrTUGTmwWqoCANRy99s65RNyub7ENnWsMQVNNQYFLyfRYQycvshvpus7TazrnevxKvGgDw82D4yhMk")
	require.NoError(t, err)
	b2, err := crypto.NewSignatureFromBase58("443nnYBRjjt8AZApoYtf5zGgukaTdCfQfBhmZQ55nyVkBXmhjzbweBaDVX23D9b5mMMXzLR6YyGHqq14BppHvAQZ")
	require.NoError(t, err)
	b3, err := crypto.NewSignatureFromBase58("YAEPx9iMfjXwbfF7Uxsi18a4y9CNZbJsavmwtRmaXiS6gcsRWdzWeHQU9jDdUNdrwQb76s1mMZNMh7cZvmoyZxz")
	require.NoError(t, err)

	conn := createConnectionWithMock("1")

	q := new(requestQueue)

	q.enqueue(b1, conn)
	assert.Equal(t, 1, len(q.blocks))
	q.enqueue(b2, conn)
	assert.Equal(t, 2, len(q.blocks))
	q.enqueue(b3, conn)
	assert.Equal(t, 3, len(q.blocks))

	_, _, ok := q.pickRandomly()
	assert.True(t, ok)
	_, _, ok = q.pickRandomly()
	assert.True(t, ok)
	_, _, ok = q.pickRandomly()
	assert.True(t, ok)
	_, _, ok = q.pickRandomly()
	assert.False(t, ok)

	q.reset()

	_, _, ok = q.pickRandomly()
	assert.True(t, ok)
	_, _, ok = q.pickRandomly()
	assert.True(t, ok)
	_, _, ok = q.pickRandomly()
	assert.True(t, ok)
	_, _, ok = q.pickRandomly()
	assert.False(t, ok)
}
