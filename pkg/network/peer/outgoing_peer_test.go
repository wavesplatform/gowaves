package peer

//
//import (
//	"context"
//	"github.com/wavesplatform/gowaves/pkg/crypto"
//	"github.com/wavesplatform/gowaves/pkg/proto"
//	"net"
//	"testing"
//	"time"
//)
//
//var seed = []byte("test test")
//
//func createTransaction() *proto.TransferV2 {
//	priv, pub := crypto.GenerateKeyPair(seed)
//	addr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, pub)
//	if err != nil {
//		panic(err)
//	}
//
//	t, err := proto.NewUnsignedTransferV2(
//		pub,
//		proto.OptionalAsset{},
//		proto.OptionalAsset{},
//		1544715621,
//		10000,
//		10000,
//		addr,
//		"",
//	)
//
//	err = t.Sign(priv)
//	if err != nil {
//		panic(err)
//	}
//	return t
//}
//
//func TestClientRecvTransaction(t *testing.T) {
//	server, client := net.Pipe()
//	transaction := createTransaction()
//	bts, _ := transaction.MarshalBinary()
//	ch := make(chan proto.Message, 1000)
//
//	creator := func(address Address) (net.Conn, error) {
//		return client, nil
//	}
//
//	c := NewPeer(context.Background(), ch, creator, "", "")
//	c.Run()
//	defer c.Stop()
//	go func() {
//		_, err := server.Write(bts)
//		require.NoError(t, err)
//	}()
//
//	select {
//	case <-ch:
//	case <-time.After(100 * time.Millisecond):
//		t.Fatal("no message received")
//	}
//
//}
