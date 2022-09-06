package net

import (
	"bufio"
	"net"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/xenolf/lego/log"

	d "github.com/wavesplatform/gowaves/itests/docker"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type OutgoingPeer struct {
	conn net.Conn
}

func NewConnection(declAddr proto.TCPAddr, address string, ver proto.Version, wavesNetwork string) (*OutgoingPeer, error) {
	c, err := net.Dial("tcp", address)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to connect to %s", address)
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
		return nil, errors.Wrapf(err, "failed to send handshake to %s", address)
	}

	_, err = handshake.ReadFrom(bufio.NewReader(c))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read handshake from %s", address)
	}

	return &OutgoingPeer{conn: c}, nil
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

type NodeConnections struct {
	scalaCon *OutgoingPeer
	goCon    *OutgoingPeer
}

func NewNodeConnections(t *testing.T, p *d.Ports) NodeConnections {
	goCon, err := NewConnection(proto.TCPAddr{}, d.Localhost+":"+p.Go.BindPort, proto.ProtocolVersion, "wavesL")
	assert.NoError(t, err, "failed to create connection to go node")
	scalaCon, err := NewConnection(proto.TCPAddr{}, d.Localhost+":"+p.Scala.BindPort, proto.ProtocolVersion, "wavesL")
	assert.NoError(t, err, "failed to create connection to scala node")

	return NodeConnections{
		scalaCon: scalaCon,
		goCon:    goCon,
	}
}

func (c *NodeConnections) SendToEachNode(t *testing.T, m proto.Message) {
	err := c.goCon.SendMessage(m)
	assert.NoError(t, err, "failed to send TransactionMessage to go node")

	err = c.scalaCon.SendMessage(m)
	assert.NoError(t, err, "failed to send TransactionMessage to scala node")
}

func (c *NodeConnections) Close() {
	if err := c.goCon.Close(); err != nil {
		log.Warnf("Failed to close connection: %s", err)
	}
	if err := c.scalaCon.Close(); err != nil {
		log.Warnf("Failed to close connection: %s", err)
	}
}
