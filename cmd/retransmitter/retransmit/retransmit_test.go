package retransmit

import (
	"context"
	"github.com/stretchr/testify/assert"

	"github.com/wavesplatform/gowaves/cmd/retransmitter/retransmit/utils"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/libs/bytespool"
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

func TestClientRecvTransaction(t *testing.T) {
	ctx := context.Background()
	transaction := createTransaction()
	bts, _ := transaction.MarshalBinary()

	addrToCh := make(map[string]*mockPeer)

	outgoingSpawner := func(ctx context.Context, params peer.OutgoingPeerParams) {
		addrToCh[params.Address] = &mockPeer{
			addr:     params.Address,
			incomeCh: params.Parent.MessageCh,
		}

		params.Parent.InfoCh <- peer.InfoMessage{
			ID: params.Address,
			Value: &peer.Connected{
				Peer: addrToCh[params.Address],
			},
		}
	}

	knownPeers, _ := utils.NewKnownPeers(utils.NoOnStorage{})
	counter := utils.NewCounter(ctx)
	pool := bytespool.NewBytesPool(1, 2*1024*1024)

	r := NewRetransmitter("wavesD", proto.PeerInfo{}, knownPeers, counter, outgoingSpawner, nil, nil, pool)
	go r.Run(ctx)

	r.AddAddress(ctx, "127.0.0.1:100")
	r.AddAddress(ctx, "127.0.0.1:101")

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
