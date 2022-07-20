package net

import (
	"bufio"
	"fmt"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"net"
	"time"
)

type OutgoingPeer struct {
	conn net.Conn
}

func NewConnection(declAddr proto.TCPAddr, address string, ver proto.Version, wavesNetwork string) (OutgoingPeer, error) {
	c, err := net.Dial("tcp", address)
	if err != nil {
		return OutgoingPeer{}, fmt.Errorf("failed to connect to %s: %s", address, err)
	}
	handshake := proto.Handshake{
		AppName:      wavesNetwork,
		Version:      ver,
		NodeName:     "itest",
		NodeNonce:    0x0,
		DeclaredAddr: proto.HandshakeTCPAddr(declAddr),
		Timestamp:    proto.NewTimestampFromTime(time.Now()),
	}

	_, err = handshake.WriteTo(c)
	if err != nil {
		return OutgoingPeer{}, fmt.Errorf("failed to send handshake to %s: %s", address, err)
	}

	_, err = handshake.ReadFrom(bufio.NewReader(c))
	if err != nil {
		return OutgoingPeer{}, fmt.Errorf("failed to read handshake from %s: %s", address, err)
	}

	return OutgoingPeer{conn: c}, nil
}

func (a *OutgoingPeer) SendMessage(m proto.Message) error {
	b, err := m.MarshalBinary()
	if err != nil {
		return err
	}

	_, err = a.conn.Write(b)
	if err != nil {
		return fmt.Errorf("failed to send message: %s", err)
	}
	return nil
}
