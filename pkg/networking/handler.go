package networking

import "io"

// Handler is an interface for handling new messages, handshakes and session close events.
type Handler[HS Handshake, H Header] interface {
	// OnReceive fired on new message received.
	OnReceive(*Session[HS, H], io.Reader)

	// OnHandshake fired on new successful Handshake received.
	OnHandshake(*Session[HS, H], HS)

	// OnHandshakeFailed fired on failed Handshake received.
	OnHandshakeFailed(*Session[HS, H], HS)

	// OnClose fired on Session closed.
	OnClose(*Session[HS, H])
}
