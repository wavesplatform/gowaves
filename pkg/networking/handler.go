package networking

import (
	"io"
	"net"
	"net/netip"
)

// EndpointWriter represents a writable endpoint that exposes the remote address in both net.Addr and
// netip.AddrPort formats.
type EndpointWriter interface {
	io.Writer
	RemoteAddr() net.Addr
	RemoteAddrPort() netip.AddrPort
}

// Handler is an interface for handling new messages, handshakes and session close events.
// All handler functions are called synchronously in goroutines of internal loops.
// Synchronously calling functions of Session itself inside handlers may lead to deadlocks.
type Handler interface {
	// OnReceive is triggered when a new message is received.
	// The message is available for reading from the provided io.Reader.
	OnReceive(EndpointWriter, io.Reader)

	// OnHandshake is triggered when a valid handshake is received.
	// Called after the protocol verifies that the received handshake is acceptable.
	OnHandshake(EndpointWriter, Handshake)

	// OnHandshakeFailed is triggered when an invalid or unacceptable handshake is received.
	// Called after the protocol rejects the handshake.
	OnHandshakeFailed(EndpointWriter, Handshake)

	// OnClose is triggered when the session is closed.
	// Called when the underlying network connection is detected as closed during a read operation.
	OnClose(EndpointWriter)

	// OnFailure is triggered when a session fails due to a network error.
	// Called when an unexpected error occurs while reading from the connection or sending keep-alive messages.
	OnFailure(EndpointWriter, error)
}
