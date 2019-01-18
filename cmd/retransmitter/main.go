package main

import (
	"context"
	"fmt"
	"github.com/wavesplatform/gowaves/pkg/network/peer"
	"github.com/wavesplatform/gowaves/pkg/network/peer/connection"
	"github.com/wavesplatform/gowaves/pkg/network/retransmit"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"net"
	"time"
)

type Connector struct {
	addr string
	v    proto.Version
}

func (a Connector) Connect(readFromRemoteCh chan []byte, writeToRemoteCh chan []byte, infoCh chan interface{}) context.CancelFunc {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	_ = connection.NewConnection(ctx, a.addr, a.v, readFromRemoteCh, writeToRemoteCh, infoCh)
	return cancel
}

// filter only transactions and
func receiveFromRemoteCallbackfunc(b []byte, id peer.UniqID, resendTo chan peer.IdentifiedMessage) {

	if len(b) < 9 {
		return
	}

	switch b[8] {
	case proto.ContentIDTransaction:

		m := &proto.TransactionMessage{}
		err := m.UnmarshalBinary(b)
		if err != nil {
			fmt.Println(err)
			return
		}

		mess := peer.IdentifiedMessage{
			ID:      id,
			Message: m,
		}

		select {
		case resendTo <- mess:
		default:
		}

	case proto.ContentIDPeers:
		fmt.Println("got proto.ContentIDPeers message", id)
	case proto.ContentIDGetPeers:
	default:
		fmt.Println("bytes id ", b[8])
		return
	}
}

func errorhandlerFunc(*retransmit.PeerInfo, *retransmit.Retransmitter) {
	fmt.Println("called errorhandlerFunc")
}

func main() {

	ctx := context.Background()

	outgoingSpawner := func(addr string, incomeCh chan peer.IdentifiedMessage, infoCh chan peer.IdentifiedInfo) peer.Peer {

		v := proto.Version{
			Major: 0,
			Minor: 15,
			Patch: 0,
		}

		connector := Connector{
			addr: addr,
			v:    v,
		}

		c := peer.NewPeer(ctx, incomeCh, connector, peer.UniqID(addr), addr, receiveFromRemoteCallbackfunc, infoCh)
		go c.Run()
		return c
	}

	incomingSpawner := func(conn net.Conn, income chan peer.IdentifiedMessage, infoCh chan peer.IdentifiedInfo) peer.Peer {
		return peer.NewIncomingPeer(conn, receiveFromRemoteCallbackfunc, income, infoCh)
	}

	r := retransmit.NewRetransmitter(ctx, outgoingSpawner, incomingSpawner, errorhandlerFunc)

	go r.Run()

	//r.AddAddress("195.201.172.78:6868")
	r.AddAddress("34.253.153.4:6868")

	<-time.After(100 * time.Minute)
}
