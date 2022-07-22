package net

import (
	"bufio"
	"net"
	"time"

	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/proto"
)

type OutgoingPeer struct {
	conn net.Conn
}

func NewConnection(declAddr proto.TCPAddr, address string, ver proto.Version, wavesNetwork string) (OutgoingPeer, error) {
	c, err := net.Dial("tcp", address)
	if err != nil {
		return OutgoingPeer{}, errors.Wrapf(err, "failed to connect to %s", address)
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
		return OutgoingPeer{}, errors.Wrapf(err, "failed to send handshake to %s", address)
	}

	_, err = handshake.ReadFrom(bufio.NewReader(c))
	if err != nil {
		return OutgoingPeer{}, errors.Wrapf(err, "failed to read handshake from %s", address)
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
		return errors.Wrapf(err, "failed to send message")
	}
	return nil
}

func (a *OutgoingPeer) Close() error {
	return a.conn.Close()
}
