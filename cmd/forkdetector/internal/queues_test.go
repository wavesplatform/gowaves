package internal

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
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
	_, c, ok := q.pickRandomly(nil)
	assert.False(t, ok)
	assert.Nil(t, c)
	assert.Equal(t, 0, len(q.blocks))
}

func TestRequestQueueOneConnection(t *testing.T) {
	sig1, err := crypto.NewSignatureFromBase58("3kSNeUztQ6HrTUGTmwWqoCANRy99s65RNyub7ENnWsMQVNNQYFLyfRYQycvshvpus7TazrnevxKvGgDw82D4yhMk")
	require.NoError(t, err)
	b1 := proto.NewBlockIDFromSignature(sig1)
	sig2, err := crypto.NewSignatureFromBase58("443nnYBRjjt8AZApoYtf5zGgukaTdCfQfBhmZQ55nyVkBXmhjzbweBaDVX23D9b5mMMXzLR6YyGHqq14BppHvAQZ")
	require.NoError(t, err)
	b2 := proto.NewBlockIDFromSignature(sig2)
	sig3, err := crypto.NewSignatureFromBase58("YAEPx9iMfjXwbfF7Uxsi18a4y9CNZbJsavmwtRmaXiS6gcsRWdzWeHQU9jDdUNdrwQb76s1mMZNMh7cZvmoyZxz")
	require.NoError(t, err)
	b3 := proto.NewBlockIDFromSignature(sig3)

	conn := createConnectionWithMock("1")

	q := new(requestQueue)

	q.enqueue(b1, conn)
	assert.Equal(t, 1, len(q.blocks))
	q.enqueue(b2, conn)
	assert.Equal(t, 2, len(q.blocks))

	b, c, ok := q.pickRandomly(nil)
	assert.True(t, ok)
	assert.Equal(t, b1, b)
	assert.Equal(t, conn, c)

	q.enqueue(b3, conn)
	assert.Equal(t, 3, len(q.blocks))

	b, c, ok = q.pickRandomly(nil)
	assert.True(t, ok)
	assert.Equal(t, b2, b)
	assert.Equal(t, conn, c)

	b, c, ok = q.pickRandomly(nil)
	assert.True(t, ok)
	assert.Equal(t, b3, b)
	assert.Equal(t, conn, c)

	_, c, ok = q.pickRandomly(nil)
	assert.False(t, ok)
	assert.Nil(t, c)
}

func TestRequestQueueFewConnections(t *testing.T) {
	sig1, err := crypto.NewSignatureFromBase58("3kSNeUztQ6HrTUGTmwWqoCANRy99s65RNyub7ENnWsMQVNNQYFLyfRYQycvshvpus7TazrnevxKvGgDw82D4yhMk")
	require.NoError(t, err)
	b1 := proto.NewBlockIDFromSignature(sig1)
	sig2, err := crypto.NewSignatureFromBase58("443nnYBRjjt8AZApoYtf5zGgukaTdCfQfBhmZQ55nyVkBXmhjzbweBaDVX23D9b5mMMXzLR6YyGHqq14BppHvAQZ")
	require.NoError(t, err)
	b2 := proto.NewBlockIDFromSignature(sig2)
	sig3, err := crypto.NewSignatureFromBase58("YAEPx9iMfjXwbfF7Uxsi18a4y9CNZbJsavmwtRmaXiS6gcsRWdzWeHQU9jDdUNdrwQb76s1mMZNMh7cZvmoyZxz")
	require.NoError(t, err)
	b3 := proto.NewBlockIDFromSignature(sig3)
	sig4, err := crypto.NewSignatureFromBase58("5UwsMWFNvuGEq9s72aYa4wXXDSaTbhtqFaG2Y9y1o6f3nskZ1rjwbZMA47hu8dQopvJ3ZeTFKRG6bo47WHYBvu9T")
	require.NoError(t, err)
	b4 := proto.NewBlockIDFromSignature(sig4)
	sig5, err := crypto.NewSignatureFromBase58("5h3MqnsxhwPgHX281nLiC1oxVjTQn1ZYngk4Ef8EZ9s6zieVyuTpTyMLBkrzwG9jzTpM3CqhWD1szCkDRpMmidTj")
	require.NoError(t, err)
	b5 := proto.NewBlockIDFromSignature(sig5)

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

	b, c, ok := q.pickRandomly(nil)
	assert.True(t, ok)
	assert.Equal(t, b1, b)
	assert.Contains(t, []*Conn{conn1, conn2}, c)
	assert.NotContains(t, []*Conn{conn3, conn4}, c)
	assert.Equal(t, 5, len(q.blocks))

	b, c, ok = q.pickRandomly(nil)
	assert.True(t, ok)
	assert.Equal(t, b2, b)
	assert.Contains(t, []*Conn{conn3, conn4}, c)
	assert.NotContains(t, []*Conn{conn1, conn2}, c)
	assert.Equal(t, 5, len(q.blocks))

	b, c, ok = q.pickRandomly(nil)
	assert.True(t, ok)
	assert.Equal(t, b3, b)
	assert.Contains(t, []*Conn{conn1, conn3}, c)
	assert.NotContains(t, []*Conn{conn2, conn4}, c)
	assert.Equal(t, 5, len(q.blocks))

	b, c, ok = q.pickRandomly(nil)
	assert.True(t, ok)
	assert.Equal(t, b4, b)
	assert.Contains(t, []*Conn{conn2, conn4}, c)
	assert.NotContains(t, []*Conn{conn1, conn3}, c)
	assert.Equal(t, 5, len(q.blocks))

	b, c, ok = q.pickRandomly(nil)
	assert.True(t, ok)
	assert.Equal(t, b5, b)
	assert.Contains(t, []*Conn{conn1, conn2, conn3, conn4}, c)
	assert.Equal(t, 5, len(q.blocks))

	_, c, ok = q.pickRandomly(nil)
	assert.False(t, ok)
	assert.Nil(t, c)
	assert.Equal(t, 5, len(q.blocks))
}

func TestRequestQueueEnqueueDequeue(t *testing.T) {
	sig0, err := crypto.NewSignatureFromBase58("5STx5DJDUo1PvhTvnk4Mb7tVniJvHMt6RrcHKhtMfsL5xD5mQGrPWesXGAYNpehreBe8sEoaJ8rErqcEyXGwpGBG")
	require.NoError(t, err)
	b0 := proto.NewBlockIDFromSignature(sig0)
	sig1, err := crypto.NewSignatureFromBase58("3kSNeUztQ6HrTUGTmwWqoCANRy99s65RNyub7ENnWsMQVNNQYFLyfRYQycvshvpus7TazrnevxKvGgDw82D4yhMk")
	require.NoError(t, err)
	b1 := proto.NewBlockIDFromSignature(sig1)
	sig2, err := crypto.NewSignatureFromBase58("443nnYBRjjt8AZApoYtf5zGgukaTdCfQfBhmZQ55nyVkBXmhjzbweBaDVX23D9b5mMMXzLR6YyGHqq14BppHvAQZ")
	require.NoError(t, err)
	b2 := proto.NewBlockIDFromSignature(sig2)
	sig3, err := crypto.NewSignatureFromBase58("YAEPx9iMfjXwbfF7Uxsi18a4y9CNZbJsavmwtRmaXiS6gcsRWdzWeHQU9jDdUNdrwQb76s1mMZNMh7cZvmoyZxz")
	require.NoError(t, err)
	b3 := proto.NewBlockIDFromSignature(sig3)
	sig4, err := crypto.NewSignatureFromBase58("5UwsMWFNvuGEq9s72aYa4wXXDSaTbhtqFaG2Y9y1o6f3nskZ1rjwbZMA47hu8dQopvJ3ZeTFKRG6bo47WHYBvu9T")
	require.NoError(t, err)
	b4 := proto.NewBlockIDFromSignature(sig4)
	sig5, err := crypto.NewSignatureFromBase58("5h3MqnsxhwPgHX281nLiC1oxVjTQn1ZYngk4Ef8EZ9s6zieVyuTpTyMLBkrzwG9jzTpM3CqhWD1szCkDRpMmidTj")
	require.NoError(t, err)
	b5 := proto.NewBlockIDFromSignature(sig5)

	conn := createConnectionWithMock("1")

	q := new(requestQueue)
	assert.Equal(t, 0, len(q.blocks))
	q.dequeue(b0)
	assert.Equal(t, 0, len(q.blocks))

	q.enqueue(b1, conn)
	assert.Equal(t, 1, len(q.blocks))
	q.enqueue(b2, conn)
	assert.Equal(t, 2, len(q.blocks))

	b, c, ok := q.pickRandomly(nil)
	assert.True(t, ok)
	assert.Equal(t, b1, b)
	assert.Equal(t, conn, c)

	q.enqueue(b3, conn)
	assert.Equal(t, 3, len(q.blocks))
	q.dequeue(b0)
	assert.Equal(t, 3, len(q.blocks))
	q.dequeue(b1)
	assert.Equal(t, 2, len(q.blocks))

	b, c, ok = q.pickRandomly(nil)
	assert.True(t, ok)
	assert.Equal(t, b2, b)
	assert.Equal(t, conn, c)

	q.dequeue(b2)
	assert.Equal(t, 1, len(q.blocks))
	q.dequeue(b2)
	assert.Equal(t, 1, len(q.blocks))

	b, c, ok = q.pickRandomly(nil)
	assert.True(t, ok)
	assert.Equal(t, b3, b)
	assert.Equal(t, conn, c)

	_, c, ok = q.pickRandomly(nil)
	assert.False(t, ok)
	assert.Nil(t, c)

	q.dequeue(b3)
	assert.Equal(t, 0, len(q.blocks))

	q.enqueue(b4, conn)
	assert.Equal(t, 1, len(q.blocks))

	b, c, ok = q.pickRandomly(nil)
	assert.True(t, ok)
	assert.Equal(t, b4, b)
	assert.Equal(t, conn, c)

	q.dequeue(b4)
	assert.Equal(t, 0, len(q.blocks))

	q.enqueue(b5, conn)
	assert.Equal(t, 1, len(q.blocks))

	b, c, ok = q.pickRandomly(nil)
	assert.True(t, ok)
	assert.Equal(t, b5, b)
	assert.Equal(t, conn, c)

	q.dequeue(b5)
	assert.Equal(t, 0, len(q.blocks))

	_, c, ok = q.pickRandomly(nil)
	assert.False(t, ok)
	assert.Nil(t, c)

	q.dequeue(b5)
	assert.Equal(t, 0, len(q.blocks))
}

func TestRequestQueueReset(t *testing.T) {
	sig1, err := crypto.NewSignatureFromBase58("3kSNeUztQ6HrTUGTmwWqoCANRy99s65RNyub7ENnWsMQVNNQYFLyfRYQycvshvpus7TazrnevxKvGgDw82D4yhMk")
	require.NoError(t, err)
	b1 := proto.NewBlockIDFromSignature(sig1)
	sig2, err := crypto.NewSignatureFromBase58("443nnYBRjjt8AZApoYtf5zGgukaTdCfQfBhmZQ55nyVkBXmhjzbweBaDVX23D9b5mMMXzLR6YyGHqq14BppHvAQZ")
	require.NoError(t, err)
	b2 := proto.NewBlockIDFromSignature(sig2)
	sig3, err := crypto.NewSignatureFromBase58("YAEPx9iMfjXwbfF7Uxsi18a4y9CNZbJsavmwtRmaXiS6gcsRWdzWeHQU9jDdUNdrwQb76s1mMZNMh7cZvmoyZxz")
	require.NoError(t, err)
	b3 := proto.NewBlockIDFromSignature(sig3)

	conn := createConnectionWithMock("1")

	q := new(requestQueue)

	q.enqueue(b1, conn)
	assert.Equal(t, 1, len(q.blocks))
	q.enqueue(b2, conn)
	assert.Equal(t, 2, len(q.blocks))
	q.enqueue(b3, conn)
	assert.Equal(t, 3, len(q.blocks))

	_, _, ok := q.pickRandomly(nil)
	assert.True(t, ok)
	_, _, ok = q.pickRandomly(nil)
	assert.True(t, ok)
	_, _, ok = q.pickRandomly(nil)
	assert.True(t, ok)
	_, _, ok = q.pickRandomly(nil)
	assert.False(t, ok)

	q.reset()

	_, _, ok = q.pickRandomly(nil)
	assert.True(t, ok)
	_, _, ok = q.pickRandomly(nil)
	assert.True(t, ok)
	_, _, ok = q.pickRandomly(nil)
	assert.True(t, ok)
	_, _, ok = q.pickRandomly(nil)
	assert.False(t, ok)
}

func TestRequestQueueUnpick(t *testing.T) {
	sig1, err := crypto.NewSignatureFromBase58("3kSNeUztQ6HrTUGTmwWqoCANRy99s65RNyub7ENnWsMQVNNQYFLyfRYQycvshvpus7TazrnevxKvGgDw82D4yhMk")
	require.NoError(t, err)
	b1 := proto.NewBlockIDFromSignature(sig1)
	sig2, err := crypto.NewSignatureFromBase58("443nnYBRjjt8AZApoYtf5zGgukaTdCfQfBhmZQ55nyVkBXmhjzbweBaDVX23D9b5mMMXzLR6YyGHqq14BppHvAQZ")
	require.NoError(t, err)
	b2 := proto.NewBlockIDFromSignature(sig2)
	sig3, err := crypto.NewSignatureFromBase58("YAEPx9iMfjXwbfF7Uxsi18a4y9CNZbJsavmwtRmaXiS6gcsRWdzWeHQU9jDdUNdrwQb76s1mMZNMh7cZvmoyZxz")
	require.NoError(t, err)
	b3 := proto.NewBlockIDFromSignature(sig3)

	conn := createConnectionWithMock("1")

	q := new(requestQueue)

	q.enqueue(b1, conn)
	assert.Equal(t, 1, len(q.blocks))
	q.enqueue(b2, conn)
	assert.Equal(t, 2, len(q.blocks))
	q.enqueue(b3, conn)
	assert.Equal(t, 3, len(q.blocks))

	for i := 0; i < 6; i++ {
		_, _, ok := q.pickRandomly(nil)
		assert.True(t, ok)
		if i%2 == 0 {
			q.unpick()
		}
	}
	_, _, ok := q.pickRandomly(nil)
	assert.False(t, ok)
}

func TestRequestQueueExclusion(t *testing.T) {
	sig1, err := crypto.NewSignatureFromBase58("3kSNeUztQ6HrTUGTmwWqoCANRy99s65RNyub7ENnWsMQVNNQYFLyfRYQycvshvpus7TazrnevxKvGgDw82D4yhMk")
	require.NoError(t, err)
	b1 := proto.NewBlockIDFromSignature(sig1)
	sig2, err := crypto.NewSignatureFromBase58("443nnYBRjjt8AZApoYtf5zGgukaTdCfQfBhmZQ55nyVkBXmhjzbweBaDVX23D9b5mMMXzLR6YyGHqq14BppHvAQZ")
	require.NoError(t, err)
	b2 := proto.NewBlockIDFromSignature(sig2)
	sig3, err := crypto.NewSignatureFromBase58("YAEPx9iMfjXwbfF7Uxsi18a4y9CNZbJsavmwtRmaXiS6gcsRWdzWeHQU9jDdUNdrwQb76s1mMZNMh7cZvmoyZxz")
	require.NoError(t, err)
	b3 := proto.NewBlockIDFromSignature(sig3)

	conn1 := createConnectionWithMock("1")
	conn2 := createConnectionWithMock("2")
	conn3 := createConnectionWithMock("3")
	conn4 := createConnectionWithMock("4")

	q := new(requestQueue)

	q.enqueue(b1, conn1)
	q.enqueue(b1, conn2)
	q.enqueue(b1, conn3)
	q.enqueue(b1, conn4)
	q.enqueue(b2, conn1)
	q.enqueue(b2, conn3)
	q.enqueue(b3, conn2)
	q.enqueue(b3, conn4)

	_, c, ok := q.pickRandomly([]*Conn{conn1, conn2})
	assert.True(t, ok)
	assert.Contains(t, []*Conn{conn3, conn4}, c)
	assert.NotContains(t, []*Conn{conn1, conn2}, c)

	_, c, ok = q.pickRandomly([]*Conn{conn1})
	assert.True(t, ok)
	assert.Contains(t, []*Conn{conn3}, c)
	assert.NotContains(t, []*Conn{conn1}, c)

	_, c, ok = q.pickRandomly([]*Conn{conn2, conn4})
	assert.True(t, ok)
	assert.Contains(t, []*Conn{conn2, conn4}, c)
}

func TestPendingQueueEmpty(t *testing.T) {
	q := new(pendingQueue)
	assert.Equal(t, 0, q.len())
	assert.Equal(t, 0, len(q.connections()))
	_, ok := q.dequeue()
	assert.False(t, ok)
}
func TestPendingQueueEnqueueDequeue(t *testing.T) {
	sig1, err := crypto.NewSignatureFromBase58("3kSNeUztQ6HrTUGTmwWqoCANRy99s65RNyub7ENnWsMQVNNQYFLyfRYQycvshvpus7TazrnevxKvGgDw82D4yhMk")
	require.NoError(t, err)
	b1 := &proto.Block{BlockHeader: proto.BlockHeader{BlockSignature: sig1}}
	s1 := proto.NewBlockIDFromSignature(sig1)
	sig2, err := crypto.NewSignatureFromBase58("443nnYBRjjt8AZApoYtf5zGgukaTdCfQfBhmZQ55nyVkBXmhjzbweBaDVX23D9b5mMMXzLR6YyGHqq14BppHvAQZ")
	require.NoError(t, err)
	s2 := proto.NewBlockIDFromSignature(sig2)
	b2 := &proto.Block{BlockHeader: proto.BlockHeader{BlockSignature: sig2}}
	sig3, err := crypto.NewSignatureFromBase58("YAEPx9iMfjXwbfF7Uxsi18a4y9CNZbJsavmwtRmaXiS6gcsRWdzWeHQU9jDdUNdrwQb76s1mMZNMh7cZvmoyZxz")
	require.NoError(t, err)
	b3 := &proto.Block{BlockHeader: proto.BlockHeader{BlockSignature: sig3}}
	s3 := proto.NewBlockIDFromSignature(sig3)

	conn1 := createConnectionWithMock("1")
	conn2 := createConnectionWithMock("2")
	conn3 := createConnectionWithMock("3")

	q := new(pendingQueue)
	q.enqueue(s1, conn1)
	q.enqueue(s2, conn2)
	q.enqueue(s3, conn3)
	assert.Equal(t, 3, q.len())
	q.update(b1)
	q.update(b2)
	q.update(b3)

	b, ok := q.dequeue()
	assert.True(t, ok)
	assert.Equal(t, b1, b)
	assert.Equal(t, 2, q.len())

	b, ok = q.dequeue()
	assert.True(t, ok)
	assert.Equal(t, b2, b)
	assert.Equal(t, 1, q.len())

	b, ok = q.dequeue()
	assert.True(t, ok)
	assert.Equal(t, b3, b)
	assert.Equal(t, 0, q.len())

	_, ok = q.dequeue()
	assert.False(t, ok)
}

func TestPendingQueueUpdate1(t *testing.T) {
	sig1, err := crypto.NewSignatureFromBase58("3kSNeUztQ6HrTUGTmwWqoCANRy99s65RNyub7ENnWsMQVNNQYFLyfRYQycvshvpus7TazrnevxKvGgDw82D4yhMk")
	require.NoError(t, err)
	s1 := proto.NewBlockIDFromSignature(sig1)
	b1 := &proto.Block{BlockHeader: proto.BlockHeader{BlockSignature: sig1}}
	sig2, err := crypto.NewSignatureFromBase58("443nnYBRjjt8AZApoYtf5zGgukaTdCfQfBhmZQ55nyVkBXmhjzbweBaDVX23D9b5mMMXzLR6YyGHqq14BppHvAQZ")
	require.NoError(t, err)
	b2 := &proto.Block{BlockHeader: proto.BlockHeader{BlockSignature: sig2}}
	s2 := proto.NewBlockIDFromSignature(sig2)
	sig3, err := crypto.NewSignatureFromBase58("YAEPx9iMfjXwbfF7Uxsi18a4y9CNZbJsavmwtRmaXiS6gcsRWdzWeHQU9jDdUNdrwQb76s1mMZNMh7cZvmoyZxz")
	require.NoError(t, err)
	b3 := &proto.Block{BlockHeader: proto.BlockHeader{BlockSignature: sig3}}
	s3 := proto.NewBlockIDFromSignature(sig3)

	conn1 := createConnectionWithMock("1")
	conn2 := createConnectionWithMock("2")
	conn3 := createConnectionWithMock("3")

	q := new(pendingQueue)
	q.enqueue(s1, conn1)
	q.enqueue(s2, conn2)
	q.enqueue(s3, conn3)
	assert.Equal(t, 3, q.len())

	_, ok := q.dequeue()
	assert.False(t, ok)

	q.update(b3)
	_, ok = q.dequeue()
	assert.False(t, ok)

	q.update(b2)
	_, ok = q.dequeue()
	assert.False(t, ok)

	q.update(b1)

	b, ok := q.dequeue()
	assert.True(t, ok)
	assert.Equal(t, b1, b)
	b, ok = q.dequeue()
	assert.True(t, ok)
	assert.Equal(t, b2, b)
	b, ok = q.dequeue()
	assert.True(t, ok)
	assert.Equal(t, b3, b)
	_, ok = q.dequeue()
	assert.False(t, ok)
}

func TestPendingQueueUpdate2(t *testing.T) {
	sig1, err := crypto.NewSignatureFromBase58("3kSNeUztQ6HrTUGTmwWqoCANRy99s65RNyub7ENnWsMQVNNQYFLyfRYQycvshvpus7TazrnevxKvGgDw82D4yhMk")
	require.NoError(t, err)
	s1 := proto.NewBlockIDFromSignature(sig1)
	b1 := &proto.Block{BlockHeader: proto.BlockHeader{BlockSignature: sig1}}
	sig2, err := crypto.NewSignatureFromBase58("443nnYBRjjt8AZApoYtf5zGgukaTdCfQfBhmZQ55nyVkBXmhjzbweBaDVX23D9b5mMMXzLR6YyGHqq14BppHvAQZ")
	require.NoError(t, err)
	s2 := proto.NewBlockIDFromSignature(sig2)
	b2 := &proto.Block{BlockHeader: proto.BlockHeader{BlockSignature: sig2}}
	sig3, err := crypto.NewSignatureFromBase58("YAEPx9iMfjXwbfF7Uxsi18a4y9CNZbJsavmwtRmaXiS6gcsRWdzWeHQU9jDdUNdrwQb76s1mMZNMh7cZvmoyZxz")
	require.NoError(t, err)
	s3 := proto.NewBlockIDFromSignature(sig3)
	b3 := &proto.Block{BlockHeader: proto.BlockHeader{BlockSignature: sig3}}

	conn1 := createConnectionWithMock("1")
	conn2 := createConnectionWithMock("2")
	conn3 := createConnectionWithMock("3")

	q := new(pendingQueue)
	q.enqueue(s1, conn1)
	q.enqueue(s2, conn2)
	q.enqueue(s3, conn3)
	assert.Equal(t, 3, q.len())

	_, ok := q.dequeue()
	assert.False(t, ok)

	q.update(b3)
	_, ok = q.dequeue()
	assert.False(t, ok)

	q.update(b1)
	b, ok := q.dequeue()
	assert.True(t, ok)
	assert.Equal(t, b1, b)

	q.update(b2)
	b, ok = q.dequeue()
	assert.True(t, ok)
	assert.Equal(t, b2, b)
	b, ok = q.dequeue()
	assert.True(t, ok)
	assert.Equal(t, b3, b)
	_, ok = q.dequeue()
	assert.False(t, ok)
}

func TestPendingQueueConnections(t *testing.T) {
	sig1, err := crypto.NewSignatureFromBase58("3kSNeUztQ6HrTUGTmwWqoCANRy99s65RNyub7ENnWsMQVNNQYFLyfRYQycvshvpus7TazrnevxKvGgDw82D4yhMk")
	require.NoError(t, err)
	b1 := &proto.Block{BlockHeader: proto.BlockHeader{BlockSignature: sig1}}
	s1 := proto.NewBlockIDFromSignature(sig1)
	sig2, err := crypto.NewSignatureFromBase58("443nnYBRjjt8AZApoYtf5zGgukaTdCfQfBhmZQ55nyVkBXmhjzbweBaDVX23D9b5mMMXzLR6YyGHqq14BppHvAQZ")
	require.NoError(t, err)
	b2 := &proto.Block{BlockHeader: proto.BlockHeader{BlockSignature: sig2}}
	s2 := proto.NewBlockIDFromSignature(sig2)
	sig3, err := crypto.NewSignatureFromBase58("YAEPx9iMfjXwbfF7Uxsi18a4y9CNZbJsavmwtRmaXiS6gcsRWdzWeHQU9jDdUNdrwQb76s1mMZNMh7cZvmoyZxz")
	require.NoError(t, err)
	b3 := &proto.Block{BlockHeader: proto.BlockHeader{BlockSignature: sig3}}
	s3 := proto.NewBlockIDFromSignature(sig3)

	conn1 := createConnectionWithMock("1")
	conn2 := createConnectionWithMock("2")
	conn3 := createConnectionWithMock("3")

	q := new(pendingQueue)
	q.enqueue(s1, conn1)
	q.enqueue(s2, conn2)
	q.enqueue(s3, conn3)
	assert.ElementsMatch(t, []*Conn{conn1, conn2, conn3}, q.connections())

	q.update(b1)
	q.update(b2)
	q.update(b3)
	assert.ElementsMatch(t, []*Conn{conn1, conn2, conn3}, q.connections())

	_, ok := q.dequeue()
	assert.True(t, ok)
	assert.ElementsMatch(t, []*Conn{conn2, conn3}, q.connections())

	_, ok = q.dequeue()
	assert.True(t, ok)
	assert.ElementsMatch(t, []*Conn{conn3}, q.connections())

	_, ok = q.dequeue()
	assert.True(t, ok)
	assert.Equal(t, 0, len(q.connections()))
}
