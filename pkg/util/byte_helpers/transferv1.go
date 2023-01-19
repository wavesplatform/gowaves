package byte_helpers

import (
	"time"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type transferWithSigBuilder struct {
	seed      string
	timestamp time.Time
}

func newTransferWithSigBuilder() transferWithSigBuilder {
	return transferWithSigBuilder{
		seed:      "test",
		timestamp: time.Unix(1544715621, 0),
	}

}

func (a transferWithSigBuilder) Seed(s string) transferWithSigBuilder {
	a.seed = s
	return a
}

func (a transferWithSigBuilder) Timestamp(t time.Time) transferWithSigBuilder {
	a.timestamp = t
	return a
}

func (a transferWithSigBuilder) Build() (*proto.TransferWithSig, error) {
	priv, pub, err := crypto.GenerateKeyPair([]byte(a.seed))
	if err != nil {
		return nil, err
	}
	addr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, pub)
	if err != nil {
		return nil, err
	}

	t := proto.NewUnsignedTransferWithSig(
		pub,
		proto.OptionalAsset{},
		proto.OptionalAsset{},
		proto.NewTimestampFromTime(a.timestamp),
		10000,
		10000,
		proto.NewRecipientFromAddress(addr),
		nil,
	)

	err = t.Sign(proto.MainNetScheme, priv)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (a transferWithSigBuilder) MustBuild() *proto.TransferWithSig {
	out, err := a.Build()
	if err != nil {
		panic(err)
	}
	return out
}
