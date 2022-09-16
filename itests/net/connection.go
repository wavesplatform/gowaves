package net

import (
	"bufio"
	"net"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	d "github.com/wavesplatform/gowaves/itests/docker"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type OutgoingPeer struct {
	conn net.Conn
}

func NewConnection(declAddr proto.TCPAddr, address string, ver proto.Version, wavesNetwork string) (op *OutgoingPeer, err error) {
	c, err := net.Dial("tcp", address)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to connect to %s", address)
	}
	defer func() {
		if err != nil {
			if closeErr := c.Close(); closeErr != nil {
				err = errors.Wrap(err, closeErr.Error())
			}
		}
	}()
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

func NewNodeConnections(p *d.Ports) (NodeConnections, error) {
	goCon, err := NewConnection(proto.TCPAddr{}, d.Localhost+":"+p.Go.BindPort, proto.ProtocolVersion, "wavesL")
	if err != nil {
		return NodeConnections{}, errors.Wrap(err, "failed to create connection to go node")
	}
	scalaCon, err := NewConnection(proto.TCPAddr{}, d.Localhost+":"+p.Scala.BindPort, proto.ProtocolVersion, "wavesL")
	if err != nil {
		if closeErr := goCon.Close(); closeErr != nil {
			err = errors.Wrap(err, closeErr.Error())
		}
		return NodeConnections{}, errors.Wrap(err, "failed to create connection to scala node")
	}
	return NodeConnections{scalaCon: scalaCon, goCon: goCon}, nil
}

func retry(timeout time.Duration, f func() error) error {
	bo := backoff.NewExponentialBackOff()
	bo.InitialInterval = 100 * time.Millisecond
	bo.MaxInterval = 500 * time.Millisecond
	bo.MaxElapsedTime = timeout
	if err := backoff.Retry(f, bo); err != nil {
		if bo.NextBackOff() == backoff.Stop {
			return errors.Wrap(err, "reached retry deadline")
		}
		return err
	}
	return nil
}

func (c *NodeConnections) Reconnect(t *testing.T, p *d.Ports) {
	c.Close(t)
	var newConns NodeConnections
	err := retry(1*time.Second, func() error {
		var err error
		newConns, err = NewNodeConnections(p)
		return err
	})
	require.NoError(t, err, "failed to create new connections")
	*c = newConns
}

func (c *NodeConnections) SendToEachNode(t *testing.T, m proto.Message) {
	err := c.goCon.SendMessage(m)
	assert.NoError(t, err, "failed to send TransactionMessage to go node")

	err = c.scalaCon.SendMessage(m)
	assert.NoError(t, err, "failed to send TransactionMessage to scala node")
}

func (c *NodeConnections) Close(t *testing.T) {
	err := c.goCon.Close()
	assert.NoError(t, err, "failed to close go node connection")

	err = c.scalaCon.Close()
	assert.NoError(t, err, "failed to close scala node connection")
}
