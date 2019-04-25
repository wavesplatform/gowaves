package util

import (
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"time"
)

type TransferV1Builder struct {
	seed      string
	timestamp time.Time
}

func NewTransferV1Builder() TransferV1Builder {
	return TransferV1Builder{
		seed:      "test",
		timestamp: time.Unix(1544715621, 0),
	}

}

func (a TransferV1Builder) Seed(s string) TransferV1Builder {
	a.seed = s
	return a
}

func (a TransferV1Builder) Timestamp(t time.Time) TransferV1Builder {
	a.timestamp = t
	return a
}

func (a TransferV1Builder) Build() (*proto.TransferV1, error) {
	priv, pub := crypto.GenerateKeyPair([]byte(a.seed))
	addr, err := proto.NewAddressFromPublicKey(proto.MainNetScheme, pub)
	if err != nil {
		return nil, err
	}

	t := proto.NewUnsignedTransferV1(
		pub,
		proto.OptionalAsset{},
		proto.OptionalAsset{},
		proto.NewTimestampFromTime(a.timestamp),
		10000,
		10000,
		proto.NewRecipientFromAddress(addr),
		"")

	err = t.Sign(priv)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (a TransferV1Builder) MustBuild() *proto.TransferV1 {
	out, err := a.Build()
	if err != nil {
		panic(err)
	}
	return out
}
