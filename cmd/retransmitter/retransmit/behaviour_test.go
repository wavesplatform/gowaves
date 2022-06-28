package retransmit_test

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/cmd/retransmitter/retransmit"
	"github.com/wavesplatform/gowaves/cmd/retransmitter/retransmit/utils"
	"github.com/wavesplatform/gowaves/pkg/p2p/mock"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/util/byte_helpers"
)

// check if first connected peer sends new transaction, then second receive
// if we send again same transaction, nothing will arrive
func TestClientRecvTransaction(t *testing.T) {
	knownPeers, _ := utils.NewKnownPeers(utils.NoOnStorage{})

	behaviour := retransmit.NewBehaviour(knownPeers, nil, proto.TestNetScheme)

	peer1 := &mock.Peer{
		Addr:          "peer1",
		RemoteAddress: proto.NewTCPAddr(net.IPv4(8, 8, 8, 8), 80),
	}
	peer2 := &mock.Peer{
		Addr:          "peer2",
		RemoteAddress: proto.NewTCPAddr(net.IPv4(8, 8, 8, 8), 90),
	}

	peer1Connected := peer.InfoMessage{
		Peer: peer1,
		Value: &peer.Connected{
			Peer: peer1,
		},
	}

	peer2Connected := peer.InfoMessage{
		Peer: peer2,
		Value: &peer.Connected{
			Peer: peer2,
		},
	}

	behaviour.InfoMessage(peer1Connected)
	behaviour.InfoMessage(peer2Connected)

	assert.Len(t, behaviour.ActiveConnections().Addresses(), 2)

	protomess := peer.ProtoMessage{
		ID: peer1,
		Message: &proto.TransactionMessage{
			Transaction: byte_helpers.TransferWithSig.TransactionBytes,
		},
	}

	behaviour.ProtoMessage(protomess)
	assert.Len(t, peer2.SendMessageCalledWith, 1)

	// sending again, and no message should arrive
	behaviour.ProtoMessage(protomess)
	assert.Len(t, peer2.SendMessageCalledWith, 1)

}
