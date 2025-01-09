package networking

import (
	"fmt"
	"net"
)

type addressable interface {
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
}

type sessionAddress struct {
	addr string
}

func (*sessionAddress) Network() string {
	return "session"
}

func (a *sessionAddress) String() string {
	return fmt.Sprintf("session:%s", a.addr)
}
