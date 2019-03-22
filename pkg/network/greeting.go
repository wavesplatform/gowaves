package network

import "net"

// Greeter is the interface that wraps the Greet method.
//
// Greet sends the handshake message to the party and validates the reply.
type Greeter interface {
	Greet(conn net.Conn) (*Contact, error)
}

// Responder is the interface that wraps the Respond method.
//
// Respond reads the handshake from given connection and sends the handshake in reply.
type Responder interface {
	Respond(conn net.Conn) (*Contact, error)
}

// Handshaker is the interface that combines Greeter and Responder interfaces.
type Handshaker interface {
	Greet(conn net.Conn) (*Contact, error)
	Respond(conn net.Conn) (*Contact, error)
}