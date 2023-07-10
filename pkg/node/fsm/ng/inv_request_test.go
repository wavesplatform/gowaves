package ng

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type messSender struct {
	messages []proto.Message
}

func (a *messSender) SendMessage(m proto.Message) {
	a.messages = append(a.messages, m)
}

func TestInvRequesterImpl_Request(t *testing.T) {
	buf := &messSender{}
	n := NewInvRequester()

	n.Request(buf, proto.NewBlockIDFromSignature(crypto.Signature{}).Bytes())
	require.Equal(t, 1, len(buf.messages))

	n.Request(buf, proto.NewBlockIDFromSignature(crypto.Signature{}).Bytes())
	require.Equal(t, 1, len(buf.messages))

}
