package byte_helpers

import (
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type GetPeersMessageStruct struct {
	Bytes []byte
}

var GetPeersMessage GetPeersMessageStruct

func init() {

	m := proto.GetPeersMessage{}
	b, _ := m.MarshalBinary()

	GetPeersMessage.Bytes = b
}
