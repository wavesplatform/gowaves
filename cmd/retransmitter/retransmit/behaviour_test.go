package retransmit_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/cmd/retransmitter/retransmit"
	"github.com/wavesplatform/gowaves/cmd/retransmitter/retransmit/utils"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/network/conn"
	"github.com/wavesplatform/gowaves/pkg/network/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/util/byte_helpers"
)

var seed = []byte("test test")

type mockPeer struct {
	addr                  string
	SendMessageCalledWith []proto.Message
	incomeCh              chan peer.ProtoMessage
}

func (mockPeer) Direction() peer.Direction {
	panic("implement me")
}

func (mockPeer) Reconnect() error {
	panic("implement me")
}

func (mockPeer) Close() {
	panic("implement me")
}

func (mockPeer) Connection() conn.Connection {
	panic("implement me")
}

func (a *mockPeer) SendMessage(m proto.Message) {
	a.SendMessageCalledWith = append(a.SendMessageCalledWith, m)
}

func (a mockPeer) ID() string {
	return a.addr
}

func createTransaction() *proto.TransferV2 {
	priv, pub := crypto.GenerateKeyPair(seed)
	addr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, pub)
	if err != nil {
		panic(err)
	}

	t, err := proto.NewUnsignedTransferV2(
		pub,
		proto.OptionalAsset{},
		proto.OptionalAsset{},
		1544715621,
		10000,
		10000,
		addr,
		"",
	)

	err = t.Sign(priv)
	if err != nil {
		panic(err)
	}
	return t
}

// check if first connected peer sends new transaction, then second receive
// if we send again same transaction, nothing will arrive
func TestClientRecvTransaction(t *testing.T) {
	knownPeers, _ := utils.NewKnownPeers(utils.NoOnStorage{})

	behaviour := retransmit.NewBehaviour(knownPeers, nil)

	peer1 := &mockPeer{
		addr: "peer1",
	}
	peer2 := &mockPeer{
		addr: "peer2",
	}

	peer1Connected := peer.InfoMessage{
		ID: peer1.addr,
		Value: &peer.Connected{
			Peer: peer1,
		},
	}

	peer2Connected := peer.InfoMessage{
		ID: peer2.addr,
		Value: &peer.Connected{
			Peer: peer2,
		},
	}

	behaviour.InfoMessage(peer1Connected)
	behaviour.InfoMessage(peer2Connected)

	assert.Len(t, behaviour.ActiveConnections().Addresses(), 2)

	protomess := peer.ProtoMessage{
		ID: peer1.addr,
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
