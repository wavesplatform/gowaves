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
	// OnReceive fired on new message received.
	OnReceive(EndpointWriter, io.Reader)

	// OnHandshake fired on new successful Handshake received.
	OnHandshake(EndpointWriter, Handshake)

	// OnHandshakeFailed fired on unacceptable Handshake received.
	OnHandshakeFailed(EndpointWriter, Handshake)

	// OnClose fired on Session closed.
	// Don't call Session.Close inside this handler synchronously it will cause deadlock.
	OnClose(EndpointWriter)

	// OnFailure fired on Session failure.
	// Don't call Session.Close inside this handler synchronously it will cause deadlock.
	OnFailure(EndpointWriter, error)
}
