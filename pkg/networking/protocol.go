package networking

import "io"

// Header is the interface that should be implemented by the real message header packet.
type Header interface {
	io.ReaderFrom
	io.WriterTo
	HeaderLength() uint32
	PayloadLength() uint32
}

// Handshake is the common interface for a handshake packet.
type Handshake interface {
	io.ReaderFrom
	io.WriterTo
}

// Protocol is the interface for the network protocol implementation.
// It provides the methods to create the handshake packet, message header, and ping packet.
// It also provides the methods to validate the handshake and message header packets.
type Protocol interface {
	// EmptyHandshake returns the empty instance of the handshake packet.
	EmptyHandshake() Handshake

	// EmptyHeader returns the empty instance of the message header.
	EmptyHeader() Header

	// Ping return the actual ping packet.
	Ping() ([]byte, error)

	// IsAcceptableHandshake checks the handshake is acceptable.
	IsAcceptableHandshake(*Session, Handshake) bool

	// IsAcceptableMessage checks the message is acceptable by examining its header.
	// If return false, the message will be discarded.
	IsAcceptableMessage(*Session, Header) bool
}
