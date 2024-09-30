package itests

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/wavesplatform/gowaves/itests/config"
	"github.com/wavesplatform/gowaves/itests/fixtures"
	"github.com/wavesplatform/gowaves/itests/net"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type SimpleChallengingSuite struct {
	fixtures.SingleGoNodeSuite
}

func (s *SimpleChallengingSuite) SetupSuite() {
	s.BaseSetup(config.WithNoGoMining())
}

func (s *SimpleChallengingSuite) TestSimpleChallenging() {
	conn, err := net.NewConnection(
		proto.TCPAddr{},
		config.DefaultIP+":"+s.Docker.GoNode().Ports().BindPort,
		proto.ProtocolVersion(),
		"wavesL",
	)
	require.NoError(s.T(), err, "failed to create connection to go node")
	defer func(conn *net.OutgoingPeer) {
		if clErr := conn.Close(); clErr != nil {
			s.T().Logf("Failed to close connection: %v", clErr)
		}
	}(conn)

	bl := proto.Block{
		BlockHeader:  proto.BlockHeader{},
		Transactions: nil,
	}
	bb, err := bl.MarshalToProtobuf(s.Cfg.BlockchainSettings.AddressSchemeCharacter)
	require.NoError(s.T(), err, "failed to marshal block")
	blMsg := &proto.BlockMessage{BlockBytes: bb}
	err = conn.SendMessage(blMsg)
	require.NoError(s.T(), err, "failed to send block to node")
}

func TestSimpleChallengingSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(SimpleChallengingSuite))
}
