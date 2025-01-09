package networking

import "io"

// Handler is an interface for handling new messages, handshakes and session close events.
type Handler interface {
	// OnReceive fired on new message received.
	OnReceive(*Session, io.Reader)

	// OnHandshake fired on new Handshake received.
	OnHandshake(*Session, Handshake)

	// OnClose fired on Session closed.
	OnClose(*Session)
}
