package retransmit

import (
	"context"
	"github.com/magiconair/properties/assert"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/network/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"testing"
	"time"
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

func (a *mockPeer) SendMessage(m proto.Message) {
	a.SendMessageCalledWith = append(a.SendMessageCalledWith, m)
}

func (a mockPeer) ID() peer.UniqID {
	return peer.UniqID(a.addr)
}
func errorHandler(peer.UniqID, error, *PeerInfo, *Retransmitter) {

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

func TestClientRecvTransaction(t *testing.T) {
	transaction := createTransaction()
	bts, _ := transaction.MarshalBinary()

	addrToCh := make(map[string]*mockPeer)

	outgoingSpawner := func(addr string, incomeCh chan peer.ProtoMessage, infoCh chan peer.InfoMessage) {
		addrToCh[addr] = &mockPeer{
			addr:     addr,
			incomeCh: incomeCh,
		}
	}

	r := NewRetransmitter(context.Background(), outgoingSpawner, nil, errorHandler)
	go r.Run()

	r.AddAddress("127.0.0.1:100")
	r.AddAddress("127.0.0.1:101")

	<-time.After(10 * time.Millisecond)

	addrToCh["127.0.0.1:100"].incomeCh <- peer.ProtoMessage{
		Message: &proto.TransactionMessage{
			Transaction: bts,
		},
		ID: "127.0.0.1:100",
	}

	<-time.After(10 * time.Millisecond)

	assert.Equal(t, 1, len(addrToCh["127.0.0.1:101"].SendMessageCalledWith))
}
