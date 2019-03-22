package network

import (
	"github.com/go-errors/errors"
	"net"
)

var (
	NoEntryFound = errors.New("no entry found")
)

type Register interface {
	ContactByConn(conn net.Conn) (*Contact, error)
}
