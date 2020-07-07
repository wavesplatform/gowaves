package util

import (
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"time"
)

type TransferWithSigBuilder struct {
	seed      string
	timestamp time.Time
}

func NewTransferWithSigBuilder() TransferWithSigBuilder {
	return TransferWithSigBuilder{
		seed:      "test",
		timestamp: time.Unix(1544715621, 0),
	}

}

func (a TransferWithSigBuilder) Seed(s string) TransferWithSigBuilder {
	a.seed = s
	return a
}

func (a TransferWithSigBuilder) Timestamp(t time.Time) TransferWithSigBuilder {
	a.timestamp = t
	return a
}

func (a TransferWithSigBuilder) Build() (*proto.TransferWithSig, error) {
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

func (a TransferWithSigBuilder) MustBuild() *proto.TransferWithSig {
	out, err := a.Build()
	if err != nil {
		panic(err)
	}
	return out
}
