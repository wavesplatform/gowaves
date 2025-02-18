package networking

import (
	"context"
	"errors"
	"io"
)

const Namespace = "NET"

// TODO: Consider special Error type for all [networking] errors.
var (
	// ErrInvalidConfigurationNoProtocol is used when the configuration has no protocol.
	ErrInvalidConfigurationNoProtocol = errors.New("invalid configuration: empty protocol")

	// ErrInvalidConfigurationNoHandler is used when the configuration has no handler.
	ErrInvalidConfigurationNoHandler = errors.New("invalid configuration: empty handler")

	// ErrInvalidConfigurationNoKeepAliveInterval is used when the configuration has an invalid keep-alive interval.
	ErrInvalidConfigurationNoKeepAliveInterval = errors.New("invalid configuration: invalid keep-alive interval value")

	// ErrInvalidConfigurationNoWriteTimeout is used when the configuration has an invalid write timeout.
	ErrInvalidConfigurationNoWriteTimeout = errors.New("invalid configuration: invalid write timeout value")

	// ErrUnacceptableHandshake is used when the handshake is not accepted.
	ErrUnacceptableHandshake = errors.New("handshake is not accepted")

	// ErrSessionShutdown is used if there is a shutdown during an operation.
	ErrSessionShutdown = errors.New("session shutdown")

	// ErrConnectionWriteTimeout indicates that we hit the timeout writing to the underlying stream connection.
	ErrConnectionWriteTimeout = errors.New("connection write timeout")

	// ErrKeepAliveProtocolFailure is used when the protocol failed to provide a keep-alive message.
	ErrKeepAliveProtocolFailure = errors.New("protocol failed to provide a keep-alive message")

	// ErrConnectionClosedOnRead indicates that the connection was closed while reading.
	ErrConnectionClosedOnRead = errors.New("connection closed on read")

	// ErrKeepAliveTimeout indicates that we failed to send keep-alive message and abandon a keep-alive loop.
	ErrKeepAliveTimeout = errors.New("keep-alive loop timeout")

	// ErrEmptyTimerPool is raised on creation of Session with a nil pool.
	ErrEmptyTimerPool = errors.New("empty timer pool")
)

type Network[HS Handshake] struct {
	tp *timerPool
}

func NewNetwork[HS Handshake]() *Network[HS] {
	return &Network[HS]{
		tp: newTimerPool(),
	}
}

func (n *Network[HS]) NewSession(ctx context.Context, conn io.ReadWriteCloser, conf *Config[HS]) (*Session[HS], error) {
	return newSession(ctx, conf, conn, n.tp)
}
