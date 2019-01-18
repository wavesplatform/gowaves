package peer

import "context"

type Connector interface {
	Connect(readFromRemoteCh chan []byte, writeToRemoteCh chan []byte, infoCh chan interface{}) context.CancelFunc
}
