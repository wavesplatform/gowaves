package peer

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"net"
)

type IncomingPeer struct {
	conn                      net.Conn
	sendToRemoteMessage       chan proto.Message
	parentInfoChan            chan IdentifiedInfo
	receiveFromRemoteCallback ReceiveFromRemoteCallback
	resendToParentCh          chan IdentifiedMessage
	closeCh                   chan struct{}
}

func NewIncomingPeer(c net.Conn, receiveFromRemoteCallback ReceiveFromRemoteCallback, resendToParentCh chan IdentifiedMessage, parentInfoChan chan IdentifiedInfo) *IncomingPeer {
	peer := &IncomingPeer{
		conn:                      c,
		parentInfoChan:            parentInfoChan,
		receiveFromRemoteCallback: receiveFromRemoteCallback,
		resendToParentCh:          resendToParentCh,
		closeCh:                   make(chan struct{}),
	}
	go peer.run()
	return peer
}

func (a *IncomingPeer) run() {
	readFromRemoteCh := make(chan []byte, 10)
	for {
		select {
		case <-a.closeCh:
			a.conn.Close()
			return
		case mess := <-a.sendToRemoteMessage:
			b, err := mess.MarshalBinary()
			if err != nil {
				fmt.Println(err)
				continue
			}
			_, err = a.conn.Write(b)
			if err != nil {
				ii := IdentifiedInfo{
					ID:    a.ID(),
					Value: err,
				}
				select {
				case a.parentInfoChan <- ii:
				default:
				}
			}
		case bts := <-readFromRemoteCh:
			a.receiveFromRemoteCallback(bts, a.ID(), a.resendToParentCh)
		}
	}
}

func (a *IncomingPeer) Reconnect() error {
	return errors.New("can't reconnect incoming peer")
}

func (a *IncomingPeer) Close() {
	close(a.closeCh)
}

func (a *IncomingPeer) SendMessage(m proto.Message) {

}

func (a *IncomingPeer) ID() UniqID {
	return UniqID(a.conn.RemoteAddr().String())
}

func (a *IncomingPeer) Direction() Direction {
	return Incoming
}
