package networking

import "io"

// Handler is an interface for handling new messages, handshakes and session close events.
type Handler[HS Handshake] interface {
	// OnReceive fired on new message received.
	OnReceive(*Session[HS], io.Reader)

	// OnHandshake fired on new successful Handshake received.
	OnHandshake(*Session[HS], HS)

	// OnHandshakeFailed fired on failed Handshake received.
	OnHandshakeFailed(*Session[HS], HS)

	// OnClose fired on Session closed.
	OnClose(*Session[HS])
}
