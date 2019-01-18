package peer

import (
	"context"
	"errors"
	"fmt"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type UniqID string
type Address string

type Direction int

const Incoming Direction = 1
const Outgoing Direction = 2

type IdentifiedMessage struct {
	ID      UniqID
	Message proto.Message
}

type IdentifiedInfo struct {
	ID    UniqID
	Value interface{}
}

type SendToRemoteCallback func(proto.Message, chan []byte)

type ReceiveFromRemoteCallback func(b []byte, id UniqID, resendTo chan IdentifiedMessage)

type OutgoingPeer struct {
	ctx                       context.Context
	resendToParentCh          chan IdentifiedMessage
	address                   string
	cancel                    context.CancelFunc
	reconnectChan             chan struct{}
	connector                 Connector
	receiveFromRemoteCallback ReceiveFromRemoteCallback
	infoCh                    chan interface{}
	writeToRemoteCh           chan []byte
	id                        UniqID
	parentInfoChan            chan IdentifiedInfo
	direction                 Direction
}

func NewPeer(
	ctx context.Context,
	resendToParentCh chan IdentifiedMessage,
	connector Connector,
	id UniqID,
	address string,
	receiveFromRemoteCallback ReceiveFromRemoteCallback,
	parentInfoChan chan IdentifiedInfo) *OutgoingPeer {

	c2, cancel := context.WithCancel(ctx)

	fmt.Println("starting new client")

	return &OutgoingPeer{
		ctx:                       c2,
		cancel:                    cancel,
		resendToParentCh:          resendToParentCh,
		address:                   address,
		connector:                 connector,
		receiveFromRemoteCallback: receiveFromRemoteCallback,
		infoCh:                    make(chan interface{}, 100),
		writeToRemoteCh:           make(chan []byte, 10),
		id:                        id,
		reconnectChan:             make(chan struct{}, 1),
		parentInfoChan:            parentInfoChan,
		direction:                 Outgoing,
	}
}

func (a *OutgoingPeer) Run() {
	readFromRemoteCh := make(chan []byte, 10)

	cancel := a.connector.Connect(readFromRemoteCh, a.writeToRemoteCh, a.infoCh)
	for {
		select {
		case <-a.ctx.Done():
			cancel()
			return

		case bts := <-readFromRemoteCh:
			a.receiveFromRemoteCallback(bts, a.id, a.resendToParentCh)

		case <-a.reconnectChan:
			cancel()
			cancel = a.connector.Connect(readFromRemoteCh, a.writeToRemoteCh, a.infoCh)

		case err := <-a.infoCh:
			a.parentInfoChan <- IdentifiedInfo{
				ID:    a.id,
				Value: err,
			}
		}
	}
}

func (a *OutgoingPeer) Stop() {
	a.cancel()
}

func (a *OutgoingPeer) SendMessage(m proto.Message) {
	b, err := m.MarshalBinary()
	if err != nil {
		a.infoCh <- err
		return
	}
	select {
	case a.writeToRemoteCh <- b:
	default:
	}
}

func (a *OutgoingPeer) Reconnect() error {
	if a.direction == Incoming {
		return errors.New("trying to reconnect to incoming connection")
	}
	select {
	case a.reconnectChan <- struct{}{}:
	default:
	}
	return nil
}

func (a *OutgoingPeer) Direction() Direction {
	return a.direction
}

func (a *OutgoingPeer) Close() {
	a.cancel()
}

func (a *OutgoingPeer) ID() UniqID {
	return a.id
}
