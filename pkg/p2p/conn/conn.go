package conn

import (
	"bytes"
	"context"
	"io"
	"net"
	"time"

	"github.com/pkg/errors"
	"github.com/valyala/bytebufferpool"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/atomic"
	"go.uber.org/zap"
)

const (
	maxMessageSize              = 2 << (10 * 2)
	maxConnIODurationPerMessage = 15 * time.Second
	MaxConnIdleIODuration       = 5 * time.Minute
)

type Connection interface {
	io.Closer
	Conn() net.Conn
	SendClosed() bool
	ReceiveClosed() bool
}

type readDeadlineSetter interface {
	SetReadDeadline(t time.Time) error
}

//go:generate moq -out deadline_reader_moq.go ./ deadlineReader:mockDeadlineReader
type deadlineReader interface {
	io.Reader
	readDeadlineSetter
}

type deadlineWriter interface {
	io.Writer
	SetWriteDeadline(t time.Time) error
}

func sendToRemote(ctx context.Context, conn deadlineWriter, toRemoteCh chan []byte, now func() time.Time) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case bts := <-toRemoteCh:
			deadline := now().Add(maxConnIODurationPerMessage)
			if err := conn.SetWriteDeadline(deadline); err != nil {
				return errors.Wrapf(err, "failed to set write deadline to %q", deadline.String())
			}
			_, err := conn.Write(bts)
			if err != nil {
				return errors.Wrap(err, "failed to write data to remote")
			}
		}
	}
}

type multiReader struct {
	readers []io.Reader
}

func newMultiReader(readers ...io.Reader) *multiReader {
	return &multiReader{readers: readers}
}

func (mr *multiReader) Read(p []byte) (n int, err error) {
	for len(mr.readers) > 0 {
		n, err = mr.readers[0].Read(p)
		if errors.Is(err, io.EOF) {
			mr.readers[0] = nil // permit earlier GC
			mr.readers = mr.readers[1:]
		}
		if n > 0 || !errors.Is(err, io.EOF) {
			if errors.Is(err, io.EOF) && len(mr.readers) > 0 {
				// Don't return EOF yet. More readers remain.
				err = nil
			}
			return
		}
	}
	return 0, io.EOF
}

func (mr *multiReader) Reset(readers ...io.Reader) {
	mr.readers = readers
}

// SkipFilter indicates that the network message should be skipped.
type SkipFilter func(proto.Header) bool

func receiveFromRemote(conn deadlineReader, fromRemoteCh chan *bytebufferpool.ByteBuffer, skip SkipFilter, addr string, now func() time.Time) error {
	var (
		firstByteBuff   = make([]byte, 1)
		firstByteReader = bytes.NewReader(firstByteBuff)
		reader          = newMultiReader(firstByteReader, conn)
	)
	for {
		idleDeadline := now().Add(MaxConnIdleIODuration)
		if err := conn.SetReadDeadline(idleDeadline); err != nil {
			return errors.Wrapf(err, "failed to set read deadline to %q", idleDeadline.String())
		}
		if _, err := io.ReadFull(conn, firstByteBuff); err != nil {
			return errors.Wrap(err, "failed to message first byte")
		}

		messageDeadline := now().Add(maxConnIODurationPerMessage)
		if err := conn.SetReadDeadline(messageDeadline); err != nil {
			return errors.Wrapf(err, "failed to set read deadline to %q", messageDeadline.String())
		}
		firstByteReader.Reset(firstByteBuff)
		reader.Reset(firstByteReader, conn)

		header := proto.Header{}
		if _, err := header.ReadFrom(reader); err != nil {
			return errors.Wrap(err, "failed to read header")
		}
		// received too big message, probably it's an error
		if l := int(header.HeaderLength() + header.PayloadLength); l > maxMessageSize {
			return errors.Errorf("received too long message, size=%d > max=%d", l, maxMessageSize)
		}

		if skip(header) {
			if _, err := io.CopyN(io.Discard, reader, int64(header.PayloadLength)); err != nil {
				return errors.Wrap(err, "failed to skip payload")
			}
			continue
		}

		b := bytebufferpool.Get()
		// put header before payload
		if _, err := header.WriteTo(b); err != nil {
			bytebufferpool.Put(b)
			return errors.Wrap(err, "failed to write header into buff")
		}
		// then read all message to remaining buffer
		if _, err := io.CopyN(b, reader, int64(header.PayloadLength)); err != nil {
			bytebufferpool.Put(b)
			return errors.Wrap(err, "failed to read payload into buffer")
		}
		select {
		case fromRemoteCh <- b:
		default:
			bytebufferpool.Put(b)
			zap.S().Debugf("[%s] Failed to send bytes from network to upstream channel because it's full", addr)
		}
	}
}

type ConnectionImpl struct {
	sendClosed    *atomic.Bool
	receiveClosed *atomic.Bool
	conn          net.Conn
	cancel        context.CancelFunc
}

func (a *ConnectionImpl) Close() error {
	a.cancel()
	return a.conn.Close()
}

func (a *ConnectionImpl) Conn() net.Conn {
	return a.conn
}

func (a *ConnectionImpl) SendClosed() bool {
	return a.sendClosed.Load()
}

func (a *ConnectionImpl) ReceiveClosed() bool {
	return a.receiveClosed.Load()
}
