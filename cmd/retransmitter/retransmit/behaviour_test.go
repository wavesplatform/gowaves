package retransmit_test

import (
	"github.com/wavesplatform/gowaves/pkg/p2p/mock"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/cmd/retransmitter/retransmit"
	"github.com/wavesplatform/gowaves/cmd/retransmitter/retransmit/utils"
	"github.com/wavesplatform/gowaves/pkg/p2p/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/util/byte_helpers"
)

// check if first connected peer sends new transaction, then second receive
// if we send again same transaction, nothing will arrive
func TestClientRecvTransaction(t *testing.T) {
	knownPeers, _ := utils.NewKnownPeers(utils.NoOnStorage{})

	behaviour := retransmit.NewBehaviour(knownPeers, nil)

	peer1 := &mock.Peer{
		Addr: "peer1",
	}
	peer2 := &mock.Peer{
		Addr: "peer2",
	}

	peer1Connected := peer.InfoMessage{
		ID: peer1.Addr,
		Value: &peer.Connected{
			Peer: peer1,
		},
	}

	peer2Connected := peer.InfoMessage{
		ID: peer2.Addr,
		Value: &peer.Connected{
			Peer: peer2,
		},
	}

	behaviour.InfoMessage(peer1Connected)
	behaviour.InfoMessage(peer2Connected)

	assert.Len(t, behaviour.ActiveConnections().Addresses(), 2)

	protomess := peer.ProtoMessage{
		ID: peer1.Addr,
		Message: &proto.TransactionMessage{
			Transaction: byte_helpers.TransferV1.TransactionBytes,
		},
	}

	behaviour.ProtoMessage(protomess)
	assert.Len(t, peer2.SendMessageCalledWith, 1)

	// sending again, and no message should arrive
	behaviour.ProtoMessage(protomess)
	assert.Len(t, peer2.SendMessageCalledWith, 1)

}
