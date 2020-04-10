package mock

import "github.com/wavesplatform/gowaves/pkg/proto"

type NoOpPeer struct {
}

func (NoOpPeer) SendMessage(proto.Message) {

}
